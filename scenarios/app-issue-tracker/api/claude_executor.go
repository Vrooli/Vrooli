package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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
	Error            string
	MaxTurnsExceeded bool
	ExitCode         int
	ExecutionTime    time.Duration
}

// executeClaudeCode executes Claude Code directly (aligned with ecosystem-manager pattern)
// This replaces the bash wrapper approach with direct Go execution
func (s *Server) executeClaudeCode(ctx context.Context, prompt string, issueID string, startTime time.Time, timeoutDuration time.Duration) (*ClaudeExecutionResult, error) {
	settings := GetAgentSettings()

	log.Printf("Executing Claude Code for issue %s (prompt length: %d characters, timeout: %v)",
		issueID, len(prompt), timeoutDuration)

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

	log.Printf("Agent CLI command: %s %s", settings.CLICommand, strings.Join(args, " "))

	// Build command with context timeout
	cmd := exec.CommandContext(ctx, settings.CLICommand, args...)
	cmd.Dir = s.config.ScenarioRoot

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
	cmd.Env = append(os.Environ(),
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
	)

	log.Printf("Claude execution settings: MAX_TURNS=%d, ALLOWED_TOOLS=%s, SKIP_PERMISSIONS=%v, TIMEOUT=%ds",
		settings.MaxTurns, settings.AllowedTools, settings.SkipPermissions, timeoutSeconds)

	// Pipe prompt via stdin
	cmd.Stdin = strings.NewReader(prompt)

	// Set up stdout/stderr pipes for streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return &ClaudeExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create stdout pipe: %v", err),
		}, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return &ClaudeExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create stderr pipe: %v", err),
		}, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return &ClaudeExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to start Claude Code: %v", err),
		}, err
	}

	log.Printf("Claude Code agent %s started (pid %d)", agentTag, cmd.Process.Pid)

	// Stream output (like ecosystem-manager)
	var stdoutBuilder, stderrBuilder, combinedBuilder strings.Builder
	var combinedMu sync.Mutex
	var lastActivity int64
	atomic.StoreInt64(&lastActivity, time.Now().UnixNano())

	readsDone := make(chan struct{})

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
			log.Printf("Error reading %s for issue %s: %v", stream, issueID, scanErr)
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
		log.Printf("⏰ TIMEOUT: Issue %s execution exceeded %v limit (ran for %v)",
			issueID, timeoutDuration, executionTime.Round(time.Second))
		return &ClaudeExecutionResult{
			Success:       false,
			Output:        combinedOutput,
			Error:         fmt.Sprintf("⏰ TIMEOUT: Execution exceeded %v limit (ran for %v). Consider increasing timeout in Settings or simplifying the task.", timeoutDuration, executionTime.Round(time.Second)),
			ExitCode:      exitCode,
			ExecutionTime: executionTime,
		}, nil
	}

	// Check for max turns exceeded
	if detectMaxTurnsExceeded(combinedOutput) {
		log.Printf("Issue %s: MAX_TURNS limit reached (%d)", issueID, settings.MaxTurns)
		return &ClaudeExecutionResult{
			Success:          false,
			Output:           combinedOutput,
			Error:            fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", settings.MaxTurns),
			MaxTurnsExceeded: true,
			ExitCode:         exitCode,
			ExecutionTime:    executionTime,
		}, nil
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
		log.Printf("Rate limit detected for issue %s", issueID)
		// Note: Rate limit handling will be done by caller
		return &ClaudeExecutionResult{
			Success:       false,
			Output:        combinedOutput,
			Error:         "RATE_LIMIT: API rate limit reached",
			ExitCode:      exitCode,
			ExecutionTime: executionTime,
		}, nil
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
			log.Printf("Issue %s: Agent exited with code %d but produced valid structured report - treating as success",
				issueID, exitCode)
			// Override failure - agent produced valid work despite non-zero exit
			return &ClaudeExecutionResult{
				Success:       true,
				Output:        combinedOutput,
				ExitCode:      exitCode,
				ExecutionTime: executionTime,
			}, nil
		}

		// Real failure - no valid output
		log.Printf("Claude Code failed for issue %s with exit code %d", issueID, exitCode)
		return &ClaudeExecutionResult{
			Success:       false,
			Output:        combinedOutput,
			Error:         fmt.Sprintf("Claude Code execution failed (exit code %d): %s", exitCode, waitErr),
			ExitCode:      exitCode,
			ExecutionTime: executionTime,
		}, nil
	}

	// Success case
	log.Printf("Claude Code completed successfully for issue %s (output length: %d characters, time: %v)",
		issueID, len(combinedOutput), executionTime.Round(time.Second))

	return &ClaudeExecutionResult{
		Success:       true,
		Output:        combinedOutput,
		ExitCode:      exitCode,
		ExecutionTime: executionTime,
	}, nil
}

// detectMaxTurnsExceeded checks if the output indicates max turns was reached
func detectMaxTurnsExceeded(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "max turns") && strings.Contains(lower, "reached")
}
