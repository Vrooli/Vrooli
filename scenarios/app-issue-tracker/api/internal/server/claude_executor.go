package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ClaudeExecutionResult contains the result of Claude Code execution
type ClaudeExecutionResult struct {
	Success          bool
	Output           string
	LastMessage      string
	Error            string
	MaxTurnsExceeded bool
	ExitCode         int
	ExecutionTime    time.Duration
	TranscriptPath   string
	LastMessagePath  string
}

// executeClaudeCode executes Claude Code directly (aligned with ecosystem-manager pattern)
// This replaces the bash wrapper approach with direct Go execution
func (s *Server) executeClaudeCode(ctx context.Context, prompt string, issueID string, startTime time.Time, timeoutDuration time.Duration) (*ClaudeExecutionResult, error) {
	settings := GetAgentSettings()
	const idleTimeout = 10 * time.Minute

	LogInfo(
		"Executing Claude Code",
		"issue_id", issueID,
		"prompt_length", len(prompt),
		"timeout", timeoutDuration,
	)

	// Create agent tag for tracking
	agentTag := fmt.Sprintf("app-issue-tracker-%s", issueID)

	commandParts := settings.Command
	if len(commandParts) == 0 {
		commandParts = []string{"run", "--tag", "{{TAG}}", "-"}
	}

	args := make([]string, 0, len(commandParts))
	for _, part := range commandParts {
		if part == "{{TAG}}" {
			args = append(args, agentTag)
			continue
		}
		args = append(args, part)
	}

	LogDebug(
		"Prepared agent CLI command",
		"command", settings.CLICommand,
		"arguments", strings.Join(args, " "),
	)

	// Build command with context timeout
	cmd := exec.CommandContext(ctx, settings.CLICommand, args...)
	cmd.Dir = s.config.ScenarioRoot

	transcriptDir := filepath.Join(s.config.ScenarioRoot, "tmp", "codex")
	if err := os.MkdirAll(transcriptDir, 0o755); err != nil {
		return &ClaudeExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create Codex transcript directory: %v", err),
		}, err
	}

	fileSafeTag := sanitizeForFilename(agentTag)
	timestamp := time.Now().UnixNano()
	transcriptFile := filepath.Join(transcriptDir, fmt.Sprintf("%s-%d-conversation.jsonl", fileSafeTag, timestamp))
	lastMessageFile := filepath.Join(transcriptDir, fmt.Sprintf("%s-%d-last.txt", fileSafeTag, timestamp))

	workspaceDir, err := os.MkdirTemp("", fmt.Sprintf("codex-%s-", fileSafeTag))
	if err != nil {
		return &ClaudeExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create Codex workspace directory: %v", err),
		}, err
	}
	defer func() {
		if removeErr := os.RemoveAll(workspaceDir); removeErr != nil {
			LogWarn("Failed to clean Codex workspace", "workspace", workspaceDir, "error", removeErr)
		}
	}()

	LogDebug("Codex workspace directory prepared", "workspace", workspaceDir)

	// Set environment variables (ecosystem-manager pattern)
	skipPermissionsValue := "no"
	if settings.SkipPermissions {
		skipPermissionsValue = "yes"
	}
	codexSkipValue := "false"
	codexSkipConfirm := "false"
	if settings.SkipPermissions {
		codexSkipValue = "true"
		codexSkipConfirm = "true"
	}

	timeoutSeconds := int(timeoutDuration.Seconds())
	env := append([]string{}, os.Environ()...)
	env = append(env,
		"MAX_TURNS="+strconv.Itoa(settings.MaxTurns),
		"ALLOWED_TOOLS="+settings.AllowedTools,
		"TIMEOUT="+strconv.Itoa(timeoutSeconds),
		"SKIP_PERMISSIONS="+skipPermissionsValue,
		"AGENT_TAG="+agentTag,
		"CODEX_MAX_TURNS="+strconv.Itoa(settings.MaxTurns),
		"CODEX_ALLOWED_TOOLS="+settings.AllowedTools,
		"CODEX_TIMEOUT="+strconv.Itoa(timeoutSeconds),
		"CODEX_SKIP_PERMISSIONS="+codexSkipValue,
		"CODEX_SKIP_CONFIRMATIONS="+codexSkipConfirm,
		"CODEX_AGENT_TAG="+agentTag,
		"CODEX_TRANSCRIPT_FILE="+transcriptFile,
		"CODEX_LAST_MESSAGE_FILE="+lastMessageFile,
		"CODEX_WORKSPACE="+workspaceDir,
	)
	cmd.Env = env

	LogInfo(
		"Claude execution settings applied",
		"max_turns", settings.MaxTurns,
		"allowed_tools", settings.AllowedTools,
		"skip_permissions", settings.SkipPermissions,
		"timeout_seconds", timeoutSeconds,
	)

	// Pipe prompt via stdin
	cmd.Stdin = strings.NewReader(prompt)

	// Set up stdout/stderr pipes for streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return &ClaudeExecutionResult{
			Success:         false,
			Error:           fmt.Sprintf("Failed to create stdout pipe: %v", err),
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return &ClaudeExecutionResult{
			Success:         false,
			Error:           fmt.Sprintf("Failed to create stderr pipe: %v", err),
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return &ClaudeExecutionResult{
			Success:         false,
			Error:           fmt.Sprintf("Failed to start Claude Code: %v", err),
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}, err
	}

	LogInfo("Claude Code agent started", "agent_tag", agentTag, "pid", cmd.Process.Pid)

	// Stream output (like ecosystem-manager)
	var stdoutBuilder, stderrBuilder, combinedBuilder strings.Builder
	var combinedMu sync.Mutex
	var lastActivity int64
	atomic.StoreInt64(&lastActivity, time.Now().UnixNano())
	idleTriggered := atomic.Bool{}

	readsDone := make(chan struct{})
	idleMonitorStop := make(chan struct{})

	startIdleMonitor := func() {
		if idleTimeout <= 0 {
			return
		}

		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					last := time.Unix(0, atomic.LoadInt64(&lastActivity))
					if time.Since(last) >= idleTimeout {
						idleTriggered.Store(true)
						LogWarn(
							"Idle timeout detected for Claude agent",
							"issue_id", issueID,
							"idle_timeout", idleTimeout,
						)
						if cmd.Process != nil {
							if err := cmd.Process.Kill(); err != nil {
								LogWarn(
									"Failed to terminate idle Claude agent",
									"issue_id", issueID,
									"error", err,
								)
							}
						}
						return
					}
				case <-idleMonitorStop:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	streamPipe := func(stream string, reader io.ReadCloser) {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			atomic.StoreInt64(&lastActivity, time.Now().UnixNano())

			combinedMu.Lock()
			if stream == "stderr" {
				stderrBuilder.WriteString(line)
				stderrBuilder.WriteByte('\n')
			} else {
				stdoutBuilder.WriteString(line)
				stdoutBuilder.WriteByte('\n')
			}
			combinedBuilder.WriteString(line)
			combinedBuilder.WriteByte('\n')
			combinedMu.Unlock()
		}
		if scanErr := scanner.Err(); scanErr != nil && scanErr != io.EOF {
			LogWarn("Failed to read agent stream", "stream", stream, "issue_id", issueID, "error", scanErr)
		}
	}

	// Start streaming goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); streamPipe("stdout", stdoutPipe) }()
	go func() { defer wg.Done(); streamPipe("stderr", stderrPipe) }()

	startIdleMonitor()

	// Wait for command to complete
	go func() {
		wg.Wait()
		close(readsDone)
	}()

	defer close(idleMonitorStop)

	// Wait for either command completion or context timeout
	waitErrChan := make(chan error, 1)
	go func() {
		waitErrChan <- cmd.Wait()
	}()

	var waitErr error
	select {
	case waitErr = <-waitErrChan:
		// Command completed
	case <-ctx.Done():
		// Context timeout - command will be killed by CommandContext
		waitErr = ctx.Err()
	}

	// Wait for all output to be read
	<-readsDone

	// Collect output
	combinedOutput := combinedBuilder.String()
	if strings.TrimSpace(combinedOutput) == "" {
		combinedOutput = "(no output captured from Claude Code)"
	}

	finalMessage := extractFinalAgentMessage(combinedOutput)

	executionTime := time.Since(startTime)
	ctxErr := ctx.Err()

	// Determine exit code
	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Check for timeout (CRITICAL: ecosystem-manager pattern)
	// Timeout takes precedence regardless of wait error
	if ctxErr == context.DeadlineExceeded {
		LogWarn(
			"Claude Code execution timed out",
			"issue_id", issueID,
			"timeout", timeoutDuration,
			"execution_time", executionTime.Round(time.Second),
		)
		result := &ClaudeExecutionResult{
			Success:         false,
			Output:          combinedOutput,
			LastMessage:     finalMessage,
			Error:           fmt.Sprintf("⏰ TIMEOUT: Execution exceeded %v limit (ran for %v). Consider increasing timeout in Settings or simplifying the task.", timeoutDuration, executionTime.Round(time.Second)),
			ExitCode:        exitCode,
			ExecutionTime:   executionTime,
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}
		return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
	}

	// Check for max turns exceeded
	if detectMaxTurnsExceeded(combinedOutput) {
		LogWarn(
			"Claude max turns limit reached",
			"issue_id", issueID,
			"max_turns", settings.MaxTurns,
		)
		result := &ClaudeExecutionResult{
			Success:          false,
			Output:           combinedOutput,
			LastMessage:      finalMessage,
			Error:            fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", settings.MaxTurns),
			MaxTurnsExceeded: true,
			ExitCode:         exitCode,
			ExecutionTime:    executionTime,
			TranscriptPath:   transcriptFile,
			LastMessagePath:  lastMessageFile,
		}
		return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
	}

	// Check for rate limits
	lowerOutput := strings.ToLower(combinedOutput)
	if strings.Contains(lowerOutput, "usage limit") ||
		strings.Contains(lowerOutput, "rate limit") ||
		strings.Contains(lowerOutput, "ai usage limit reached") ||
		strings.Contains(lowerOutput, "rate/usage limit reached") ||
		strings.Contains(lowerOutput, "429") ||
		strings.Contains(lowerOutput, "too many requests") ||
		strings.Contains(lowerOutput, "quota exceeded") {
		LogWarn("Rate limit detected during Claude execution", "issue_id", issueID)
		// Note: Rate limit handling will be done by caller
		result := &ClaudeExecutionResult{
			Success:         false,
			Output:          combinedOutput,
			LastMessage:     finalMessage,
			Error:           "RATE_LIMIT: API rate limit reached",
			ExitCode:        exitCode,
			ExecutionTime:   executionTime,
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}
		return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
	}

	// Check for non-zero exit (ecosystem-manager pattern)
	// CRITICAL: Check output quality, not just exit codes
	if waitErr != nil && exitCode != 0 {
		if idleTriggered.Load() {
			result := &ClaudeExecutionResult{
				Success:         false,
				Output:          combinedOutput,
				LastMessage:     finalMessage,
				Error:           fmt.Sprintf("Claude Code execution ended due to inactivity (no output for %v)", idleTimeout),
				ExitCode:        exitCode,
				ExecutionTime:   executionTime,
				TranscriptPath:  transcriptFile,
				LastMessagePath: lastMessageFile,
			}
			return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
		}
		// Check if we have substantial, structured output despite error
		hasValidReport := false
		if len(combinedOutput) > 500 {
			hasValidReport = strings.Contains(lowerOutput, "investigation summary") ||
				strings.Contains(lowerOutput, "root cause") ||
				strings.Contains(lowerOutput, "remediation") ||
				strings.Contains(lowerOutput, "validation plan") ||
				strings.Contains(lowerOutput, "confidence assessment")
		}

		if hasValidReport {
			LogInfo(
				"Agent produced structured output despite non-zero exit",
				"issue_id", issueID,
				"exit_code", exitCode,
			)
			// Override failure - agent produced valid work despite non-zero exit
			result := &ClaudeExecutionResult{
				Success:         true,
				Output:          combinedOutput,
				LastMessage:     finalMessage,
				ExitCode:        exitCode,
				ExecutionTime:   executionTime,
				TranscriptPath:  transcriptFile,
				LastMessagePath: lastMessageFile,
			}
			return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
		}

		// Real failure - no valid output
		LogError(
			"Claude Code execution failed",
			"issue_id", issueID,
			"exit_code", exitCode,
			"error", waitErr,
		)
		result := &ClaudeExecutionResult{
			Success:         false,
			Output:          combinedOutput,
			LastMessage:     finalMessage,
			Error:           fmt.Sprintf("Claude Code execution failed (exit code %d): %s", exitCode, waitErr),
			ExitCode:        exitCode,
			ExecutionTime:   executionTime,
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}
		return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
	}

	// Success case
	LogInfo(
		"Claude Code execution completed",
		"issue_id", issueID,
		"output_length", len(combinedOutput),
		"execution_time", executionTime.Round(time.Second),
	)

	result := &ClaudeExecutionResult{
		Success:         true,
		Output:          combinedOutput,
		LastMessage:     finalMessage,
		ExitCode:        exitCode,
		ExecutionTime:   executionTime,
		TranscriptPath:  transcriptFile,
		LastMessagePath: lastMessageFile,
	}
	return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
}

// detectMaxTurnsExceeded checks if the output indicates max turns was reached
func detectMaxTurnsExceeded(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "max turns") && strings.Contains(lower, "reached")
}

func (s *Server) completeClaudeResult(result *ClaudeExecutionResult, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage string, settings AgentSettings) *ClaudeExecutionResult {
	if result == nil {
		return nil
	}

	if trimmed := strings.TrimSpace(result.LastMessage); trimmed == "" {
		result.LastMessage = strings.TrimSpace(finalMessage)
	}

	if err := ensureLastMessageFile(lastMessageFile, result.LastMessage); err != nil {
		LogWarn("Failed to persist last message fallback", "path", lastMessageFile, "error", err)
	}

	if err := ensureTranscriptFile(transcriptFile, prompt, result.LastMessage, combinedOutput, settings); err != nil {
		LogWarn("Failed to persist transcript fallback", "path", transcriptFile, "error", err)
	}

	return result
}

func ensureLastMessageFile(path, lastMessage string) error {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(lastMessage) == "" {
		return nil
	}
	if fileExistsAndNotEmpty(path) {
		return nil
	}
	return os.WriteFile(path, []byte(lastMessage), 0o644)
}

func ensureTranscriptFile(path, prompt, lastMessage, combinedOutput string, settings AgentSettings) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if fileExistsAndNotEmpty(path) {
		return nil
	}
	if strings.TrimSpace(prompt) == "" && strings.TrimSpace(lastMessage) == "" && strings.TrimSpace(combinedOutput) == "" {
		return nil
	}
	return writeFallbackTranscript(path, prompt, lastMessage, combinedOutput, settings)
}

func writeFallbackTranscript(path, prompt, lastMessage, combinedOutput string, settings AgentSettings) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	now := time.Now().UTC().Format(time.RFC3339)

	metadata := map[string]interface{}{
		"sandbox":      "fallback-transcript",
		"provider":     settings.Provider,
		"generated":    true,
		"generated_at": now,
	}
	if err := writeJSONLine(writer, metadata); err != nil {
		return err
	}

	if strings.TrimSpace(prompt) != "" {
		if err := writeJSONLine(writer, map[string]interface{}{"prompt": prompt}); err != nil {
			return err
		}
	}

	if strings.TrimSpace(lastMessage) != "" {
		entry := map[string]interface{}{
			"id":        fmt.Sprintf("fallback-%d", time.Now().UnixNano()),
			"timestamp": now,
			"msg": map[string]interface{}{
				"type":    "final_response",
				"role":    "assistant",
				"message": lastMessage,
			},
		}
		if err := writeJSONLine(writer, entry); err != nil {
			return err
		}
	}

	trimmedOutput := strings.TrimSpace(combinedOutput)
	if trimmedOutput != "" && strings.TrimSpace(lastMessage) != trimmedOutput {
		rawEntry := map[string]interface{}{
			"id":        fmt.Sprintf("raw-%d", time.Now().UnixNano()),
			"timestamp": now,
			"raw": map[string]interface{}{
				"combined_output": trimmedOutput,
			},
		}
		if err := writeJSONLine(writer, rawEntry); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func writeJSONLine(writer *bufio.Writer, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	return nil
}

func fileExistsAndNotEmpty(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

func extractFinalAgentMessage(output string) string {
	cleaned := strings.TrimSpace(stripANSI(output))
	if cleaned == "" {
		return ""
	}

	lines := strings.Split(cleaned, "\n")
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" || strings.HasPrefix(last, "[20") || strings.HasPrefix(last, "[SUCCESS") {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}
	cleaned = strings.TrimSpace(strings.Join(lines, "\n"))
	if cleaned == "" {
		return ""
	}

	lower := strings.ToLower(cleaned)
	if idx := strings.LastIndex(lower, "**summary**"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[idx:])
	} else if idx := strings.LastIndex(lower, "summary:"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[idx:])
	}

	return cleaned
}

func sanitizeForFilename(input string) string {
	var builder strings.Builder
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	if builder.Len() == 0 {
		return "codex"
	}

	return builder.String()
}
