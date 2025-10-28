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
	"time"

	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/utils"
)

type commandOptions struct {
	Dir   string
	Env   []string
	Stdin io.Reader
}

type agentCommand interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	Kill() error
	Process() *os.Process
}

type execAgentCommand struct {
	cmd *exec.Cmd
}

func (c *execAgentCommand) StdoutPipe() (io.ReadCloser, error) {
	return c.cmd.StdoutPipe()
}

func (c *execAgentCommand) StderrPipe() (io.ReadCloser, error) {
	return c.cmd.StderrPipe()
}

func (c *execAgentCommand) Start() error {
	return c.cmd.Start()
}

func (c *execAgentCommand) Wait() error {
	return c.cmd.Wait()
}

func (c *execAgentCommand) Kill() error {
	if c.cmd.Process == nil {
		return nil
	}
	return c.cmd.Process.Kill()
}

func (c *execAgentCommand) Process() *os.Process {
	return c.cmd.Process
}

type commandFactory func(ctx context.Context, name string, args []string, opts commandOptions) (agentCommand, error)

func defaultCommandFactory(ctx context.Context, name string, args []string, opts commandOptions) (agentCommand, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env
	cmd.Stdin = opts.Stdin
	return &execAgentCommand{cmd: cmd}, nil
}

func buildAgentCommandArgs(rawParts []string, issueID string) (string, []string) {
	agentTag := fmt.Sprintf("app-issue-tracker-%s", issueID)
	parts := rawParts
	if len(parts) == 0 {
		parts = []string{"run", "--tag", "{{TAG}}", "-"}
	}

	args := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "{{TAG}}" {
			args = append(args, agentTag)
			continue
		}
		args = append(args, part)
	}

	return agentTag, args
}

func buildClaudeEnvironment(base []string, settings AgentSettings, agentTag, transcriptFile, lastMessageFile string, timeoutSeconds int) []string {
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

	env := append([]string{}, base...)
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
	)
	return env
}

func createTranscriptPaths(root, agentTag string, now func() time.Time) (string, string, error) {
	transcriptDir := filepath.Join(root, "tmp", "codex")
	if err := os.MkdirAll(transcriptDir, 0o755); err != nil {
		return "", "", err
	}

	fileSafeTag := utils.ForFilename(agentTag)
	timestamp := now().UnixNano()
	baseName := fmt.Sprintf("%s-%d", fileSafeTag, timestamp)
	transcriptFile := filepath.Join(transcriptDir, fmt.Sprintf("%s-conversation.jsonl", baseName))
	lastMessageFile := filepath.Join(transcriptDir, fmt.Sprintf("%s-last.txt", baseName))
	return transcriptFile, lastMessageFile, nil
}

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

// executeClaudeCode executes AI agent directly (aligned with ecosystem-manager pattern)
// This replaces the bash wrapper approach with direct Go execution
func (s *Server) executeClaudeCode(ctx context.Context, prompt string, issueID string, startTime time.Time, timeoutDuration time.Duration) (*ClaudeExecutionResult, error) {
	settings := GetAgentSettings()

	logging.LogInfo(
		"Executing AI agent",
		"provider", settings.Provider,
		"issue_id", issueID,
		"prompt_length", len(prompt),
		"timeout", timeoutDuration,
	)

	agentTag, args := buildAgentCommandArgs(settings.Command, issueID)

	logging.LogDebug(
		"Prepared agent CLI command",
		"command", settings.CLICommand,
		"arguments", strings.Join(args, " "),
	)

	transcriptFile, lastMessageFile, pathErr := createTranscriptPaths(s.config.ScenarioRoot, agentTag, time.Now)
	if pathErr != nil {
		err := fmt.Errorf("failed to create agent transcript directory: %w", pathErr)
		return &ClaudeExecutionResult{Success: false, Error: err.Error()}, err
	}

	timeoutSeconds := int(timeoutDuration.Seconds())
	env := buildClaudeEnvironment(os.Environ(), settings, agentTag, transcriptFile, lastMessageFile, timeoutSeconds)

	cmd, cmdErr := s.commandFactory(ctx, settings.CLICommand, args, commandOptions{
		Dir:   s.config.ScenarioRoot,
		Env:   env,
		Stdin: strings.NewReader(prompt),
	})
	if cmdErr != nil {
		err := fmt.Errorf("failed to construct agent command: %w", cmdErr)
		return &ClaudeExecutionResult{
			Success:         false,
			Error:           err.Error(),
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}, err
	}

	logging.LogInfo(
		"Agent execution settings applied",
		"provider", settings.Provider,
		"max_turns", settings.MaxTurns,
		"allowed_tools", settings.AllowedTools,
		"skip_permissions", settings.SkipPermissions,
		"timeout_seconds", timeoutSeconds,
	)

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
			Error:           fmt.Sprintf("Failed to start agent: %v", err),
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}, err
	}

	if proc := cmd.Process(); proc != nil {
		logging.LogInfo("Agent started", "provider", settings.Provider, "agent_tag", agentTag, "pid", proc.Pid)
	} else {
		logging.LogInfo("Agent started", "provider", settings.Provider, "agent_tag", agentTag, "pid", 0)
	}

	// Stream output (like ecosystem-manager)
	var stdoutBuilder, stderrBuilder, combinedBuilder strings.Builder
	var combinedMu sync.Mutex

	readsDone := make(chan struct{})

	streamPipe := func(stream string, reader io.ReadCloser) {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()

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
			logging.LogWarn("Failed to read agent stream", "stream", stream, "issue_id", issueID, "error", scanErr)
		}
	}

	// Start streaming goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); streamPipe("stdout", stdoutPipe) }()
	go func() { defer wg.Done(); streamPipe("stderr", stderrPipe) }()

	// Wait for command to complete
	go func() {
		wg.Wait()
		close(readsDone)
	}()

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
		logging.LogWarn(
			"Agent execution timed out",
			"provider", settings.Provider,
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
		logging.LogWarn(
			"Agent max turns limit reached",
			"provider", settings.Provider,
			"issue_id", issueID,
			"max_turns", settings.MaxTurns,
		)
		result := &ClaudeExecutionResult{
			Success:          false,
			Output:           combinedOutput,
			LastMessage:      finalMessage,
			Error:            fmt.Sprintf("Agent reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", settings.MaxTurns),
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
		logging.LogWarn("Rate limit detected during agent execution", "provider", settings.Provider, "issue_id", issueID)
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
			logging.LogInfo(
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
		logging.LogErrorErr(
			"Agent execution failed",
			waitErr,
			"provider", settings.Provider,
			"issue_id", issueID,
			"exit_code", exitCode,
		)
		result := &ClaudeExecutionResult{
			Success:         false,
			Output:          combinedOutput,
			LastMessage:     finalMessage,
			Error:           fmt.Sprintf("Agent execution failed (exit code %d): %s", exitCode, waitErr),
			ExitCode:        exitCode,
			ExecutionTime:   executionTime,
			TranscriptPath:  transcriptFile,
			LastMessagePath: lastMessageFile,
		}
		return s.completeClaudeResult(result, transcriptFile, lastMessageFile, prompt, combinedOutput, finalMessage, settings), nil
	}

	// Success case
	logging.LogInfo(
		"Agent execution completed",
		"provider", settings.Provider,
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
		logging.LogWarn("Failed to persist last message fallback", "path", lastMessageFile, "error", err)
	}

	if err := ensureTranscriptFile(transcriptFile, prompt, result.LastMessage, combinedOutput, settings); err != nil {
		logging.LogWarn("Failed to persist transcript fallback", "path", transcriptFile, "error", err)
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
	cleaned := strings.TrimSpace(utils.StripANSI(output))
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
