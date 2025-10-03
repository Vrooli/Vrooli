package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// CodexAgentOptions describes how to invoke the Codex CLI.
type CodexAgentOptions struct {
	Prompt          string
	AllowedTools    string
	MaxTurns        int
	TimeoutSeconds  int
	SkipPermissions bool
	WorkingDir      string
	Command         string
	DefaultModel    string
	AgentTag        string
	ExtraEnv        map[string]string
}

// CodexAgentResult captures execution outcome and metadata.
type CodexAgentResult struct {
	Success           bool
	Output            string
	Stdout            string
	Stderr            string
	Combined          string
	Duration          time.Duration
	StartedAt         time.Time
	RateLimited       bool
	RetryAfterSeconds int
	Timeout           bool
	IdleTimeout       bool
	Error             error
}

func ExecuteCodexAgent(ctx context.Context, opts CodexAgentOptions) CodexAgentResult {
	result := CodexAgentResult{StartedAt: time.Now()}

	if strings.TrimSpace(opts.Prompt) == "" {
		result.Error = errors.New("prompt is required")
		return result
	}

	binary := opts.Command
	if binary == "" {
		binary = "resource-codex"
	}

	args := []string{"content", "execute", opts.Prompt}
	if opts.AllowedTools != "" {
		args = append(args, "--allowed-tools", opts.AllowedTools)
	}
	if opts.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(opts.MaxTurns))
	}
	if opts.TimeoutSeconds > 0 {
		args = append(args, "--timeout", strconv.Itoa(opts.TimeoutSeconds))
	}
	if opts.SkipPermissions {
		args = append(args, "--skip-permissions")
	}

	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(execCtx, binary, args...)
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	env := os.Environ()
	if opts.AllowedTools != "" {
		env = append(env, "CODEX_ALLOWED_TOOLS="+opts.AllowedTools)
	}
	if opts.MaxTurns > 0 {
		env = append(env, "CODEX_MAX_TURNS="+strconv.Itoa(opts.MaxTurns))
	}
	if opts.TimeoutSeconds > 0 {
		env = append(env, "CODEX_TIMEOUT="+strconv.Itoa(opts.TimeoutSeconds))
	}
	if opts.DefaultModel != "" {
		env = append(env, "CODEX_DEFAULT_MODEL="+opts.DefaultModel)
	}
	if opts.AgentTag != "" {
		env = append(env, "CODEX_AGENT_TAG="+opts.AgentTag)
	}
	for k, v := range opts.ExtraEnv {
		if k == "" {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	setProcessGroup(cmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to obtain stdout: %w", err)
		return result
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to obtain stderr: %w", err)
		return result
	}

	var stdoutBuilder, stderrBuilder, combinedBuilder strings.Builder
	var combinedMu sync.Mutex
	var readWG sync.WaitGroup
	lastActivity := time.Now()
	var lastActivityNano int64
	atomic.StoreInt64(&lastActivityNano, lastActivity.UnixNano())

	streamReader := func(stream string, reader io.ReadCloser, builder *strings.Builder) {
		defer readWG.Done()
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if builder != nil {
				builder.WriteString(line)
				builder.WriteByte('\n')
			}
			combinedMu.Lock()
			combinedBuilder.WriteString(line)
			combinedBuilder.WriteByte('\n')
			combinedMu.Unlock()
			atomic.StoreInt64(&lastActivityNano, time.Now().UnixNano())
		}
	}

	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start %s: %w", binary, err)
		return result
	}

	readWG.Add(2)
	go streamReader("stdout", stdoutPipe, &stdoutBuilder)
	go streamReader("stderr", stderrPipe, &stderrBuilder)

	readsDone := make(chan struct{})
	go func() {
		readWG.Wait()
		close(readsDone)
	}()

	idleLimit := computeIdleLimit(opts.TimeoutSeconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-readsDone:
				return
			case <-ticker.C:
				last := time.Unix(0, atomic.LoadInt64(&lastActivityNano))
				if time.Since(last) > idleLimit {
					result.IdleTimeout = true
					cancel()
					return
				}
			}
		}
	}()

	waitErr := cmd.Wait()
	<-readsDone
	result.Duration = time.Since(result.StartedAt)

	stdoutOutput := stdoutBuilder.String()
	stderrOutput := stderrBuilder.String()

	combined := combinedBuilder.String()
	if strings.TrimSpace(combined) == "" {
		combined = strings.TrimSpace(stdoutOutput + "\n" + stderrOutput)
	}

	result.Stdout = stdoutOutput
	result.Stderr = stderrOutput
	result.Combined = combined

	if ctxErr := execCtx.Err(); errors.Is(ctxErr, context.DeadlineExceeded) {
		result.Timeout = true
		if waitErr != nil {
			result.Error = fmt.Errorf("codex execution timed out after %d seconds", opts.TimeoutSeconds)
		} else {
			result.Error = fmt.Errorf("codex execution timed out")
		}
	} else if result.IdleTimeout {
		result.Error = fmt.Errorf("codex execution became idle for %s", idleLimit)
	} else if waitErr != nil {
		result.Error = fmt.Errorf("codex execution failed: %w", waitErr)
	}

	detectRateLimit(&result)

	if result.Error == nil {
		result.Success = true
		result.Output = combined
	}

	if cmd.Process != nil {
		_ = killProcessGroup(cmd.Process.Pid)
	}

	return result
}

func computeIdleLimit(timeoutSeconds int) time.Duration {
	if timeoutSeconds <= 0 {
		return 2 * time.Minute
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	half := timeout / 2
	if half > 5*time.Minute {
		half = 5 * time.Minute
	}
	if half < 2*time.Minute {
		half = 2 * time.Minute
	}
	return half
}

func detectRateLimit(result *CodexAgentResult) {
	combined := strings.ToLower(result.Combined)
	if combined == "" {
		combined = strings.ToLower(result.Stderr + "\n" + result.Stdout)
	}

	if strings.Contains(combined, "rate limit") || strings.Contains(combined, "429") || strings.Contains(combined, "quota exceeded") {
		result.RateLimited = true
		if result.Error == nil {
			result.Error = errors.New("codex rate limit reached")
		}
		if seconds := extractRetryAfterSeconds(combined); seconds > 0 {
			result.RetryAfterSeconds = seconds
		} else if result.RetryAfterSeconds == 0 {
			result.RetryAfterSeconds = 120
		}
	}
}

var retryAfterRegex = regexp.MustCompile(`retry[^0-9]{0,10}(?:after|in)\s*(\d+)\s*(?:seconds|secs|s)?`)

func extractRetryAfterSeconds(text string) int {
	matches := retryAfterRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0
	}
	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return value
}

func setProcessGroup(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func killProcessGroup(pid int) error {
	if pid <= 0 {
		return nil
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return err
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
