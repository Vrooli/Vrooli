package smoketest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// DefaultService is the default implementation of Service.
type DefaultService struct {
	store             Store
	cancelManager     CancelManager
	telemetryIngestor TelemetryIngestor
	port              int
	logger            Logger
}

// ServiceOption configures a DefaultService.
type ServiceOption func(*DefaultService)

// WithStore sets the smoke test store.
func WithStore(store Store) ServiceOption {
	return func(s *DefaultService) {
		s.store = store
	}
}

// WithCancelManager sets the cancel manager.
func WithCancelManager(cm CancelManager) ServiceOption {
	return func(s *DefaultService) {
		s.cancelManager = cm
	}
}

// WithTelemetryIngestor sets the telemetry ingestor.
func WithTelemetryIngestor(ti TelemetryIngestor) ServiceOption {
	return func(s *DefaultService) {
		s.telemetryIngestor = ti
	}
}

// WithPort sets the server port for telemetry uploads.
func WithPort(port int) ServiceOption {
	return func(s *DefaultService) {
		s.port = port
	}
}

// WithLogger sets the logger.
func WithLogger(logger Logger) ServiceOption {
	return func(s *DefaultService) {
		s.logger = logger
	}
}

// NewService creates a new smoke test service with the given options.
func NewService(opts ...ServiceOption) *DefaultService {
	s := &DefaultService{
		store:         NewInMemoryStore(),
		cancelManager: NewCancelManager(),
		port:          8080,
		logger:        &noopLogger{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CurrentPlatform returns the current platform identifier.
func (s *DefaultService) CurrentPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "win"
	case "darwin":
		return "mac"
	default:
		return "linux"
	}
}

// PerformSmokeTest runs a smoke test on a built application.
func (s *DefaultService) PerformSmokeTest(ctx context.Context, smokeTestID, scenarioName, artifactPath, platform string) {
	if _, ok := s.store.Get(smokeTestID); !ok {
		return
	}
	defer s.cancelManager.Clear(smokeTestID)

	defer func() {
		if r := recover(); r != nil {
			s.store.Update(smokeTestID, func(status *Status) {
				status.Status = "failed"
				status.Error = fmt.Sprintf("panic: %v", r)
				now := time.Now()
				status.CompletedAt = &now
			})
		}
	}()

	s.store.Update(smokeTestID, func(status *Status) {
		status.Logs = append(status.Logs, fmt.Sprintf("Starting smoke test for %s on %s (artifact: %s)", scenarioName, platform, filepath.Base(artifactPath)))
	})

	if ctx.Err() != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Status = "failed"
			status.Error = "smoke test cancelled"
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	if _, err := os.Stat(artifactPath); err != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Status = "failed"
			status.Error = fmt.Sprintf("artifact not found: %v", err)
			status.Logs = append(status.Logs, "Smoke test failed: artifact missing")
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	commandName, commandArgs, displayCommand, err := s.resolveSmokeTestCommand(platform, artifactPath)
	if err != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Status = "failed"
			status.Error = err.Error()
			status.Logs = append(status.Logs, "Smoke test failed: artifact not runnable")
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	uploadURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/deployment/telemetry", s.port)
	smokeEnv := []string{
		"SMOKE_TEST=1",
		fmt.Sprintf("SMOKE_TEST_TIMEOUT_MS=%d", TimeoutSeconds*1000),
		fmt.Sprintf("SMOKE_TEST_UPLOAD_URL=%s", uploadURL),
	}
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" {
		if _, err := exec.LookPath("xvfb-run"); err == nil {
			commandArgs = append([]string{"-a", commandName}, commandArgs...)
			commandName = "xvfb-run"
			displayCommand = "xvfb-run -a " + displayCommand
		} else {
			s.store.Update(smokeTestID, func(status *Status) {
				status.Status = "failed"
				status.Error = "DISPLAY is not set and xvfb-run is unavailable; cannot run Electron smoke test headlessly"
				status.Logs = append(status.Logs, "Smoke test failed: DISPLAY is not set and xvfb-run is unavailable")
				now := time.Now()
				status.CompletedAt = &now
			})
			return
		}
	}

	workDir := filepath.Dir(artifactPath)
	output, err := s.runSmokeTestCommand(ctx, workDir, commandName, commandArgs, smokeEnv, time.Duration(TimeoutSeconds)*time.Second)
	logEntry := fmt.Sprintf("[smoke-test] %s", displayCommand)
	if err != nil {
		logEntry += fmt.Sprintf("\nFAILED: %v", err)
	} else {
		logEntry += "\nSUCCESS"
	}
	if len(output) < 500 {
		logEntry += fmt.Sprintf("\nOutput: %s", output)
	} else {
		logEntry += fmt.Sprintf("\nOutput: %s... (%d bytes)", output[:500], len(output))
	}

	s.store.Update(smokeTestID, func(status *Status) {
		status.Logs = append(status.Logs, logEntry)
	})

	if strings.Contains(output, "SMOKE_TEST_UPLOAD=ok") {
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploaded = true
			status.TelemetryUploadError = ""
		})
	} else if strings.Contains(output, "SMOKE_TEST_UPLOAD=error") {
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploadError = "telemetry upload failed (see logs)"
		})
	}

	if !strings.Contains(output, "SMOKE_TEST_UPLOAD=ok") {
		s.attemptTelemetryFallback(smokeTestID, scenarioName, platform, artifactPath, output)
	}

	if err != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Status = "failed"
			status.Error = fmt.Sprintf("smoke-test failed: %v", err)
			status.Logs = append(status.Logs, "Smoke test failed")
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	if !strings.Contains(output, "SMOKE_TEST_RESULT=passed") {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Status = "failed"
			status.Error = "smoke test did not report success"
			status.Logs = append(status.Logs, "Smoke test failed: missing SMOKE_TEST_RESULT=passed")
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	s.store.Update(smokeTestID, func(status *Status) {
		status.Status = "passed"
		status.Logs = append(status.Logs, "Smoke test passed")
		now := time.Now()
		status.CompletedAt = &now
	})
}

func (s *DefaultService) resolveSmokeTestCommand(platform, artifactPath string) (string, []string, string, error) {
	switch platform {
	case "linux":
		if strings.HasSuffix(artifactPath, ".AppImage") {
			if err := ensureExecutable(artifactPath); err != nil {
				return "", nil, "", fmt.Errorf("failed to set AppImage executable bit: %w", err)
			}
			return artifactPath, []string{"--smoke-test"}, fmt.Sprintf("%s --smoke-test", artifactPath), nil
		}
		return "", nil, "", fmt.Errorf("unsupported linux artifact for smoke test: %s", filepath.Base(artifactPath))
	case "win":
		if strings.HasSuffix(strings.ToLower(artifactPath), ".exe") {
			return artifactPath, []string{"--smoke-test"}, fmt.Sprintf("%s --smoke-test", artifactPath), nil
		}
		return "", nil, "", fmt.Errorf("unsupported windows artifact for smoke test: %s", filepath.Base(artifactPath))
	case "mac":
		if strings.HasSuffix(artifactPath, ".app") {
			executable, err := resolveMacAppExecutable(artifactPath)
			if err != nil {
				return "", nil, "", err
			}
			return executable, []string{"--smoke-test"}, fmt.Sprintf("%s --smoke-test", executable), nil
		}
		return "", nil, "", fmt.Errorf("unsupported macOS artifact for smoke test: %s", filepath.Base(artifactPath))
	default:
		return "", nil, "", fmt.Errorf("unsupported platform for smoke test: %s", platform)
	}
}

func resolveMacAppExecutable(appPath string) (string, error) {
	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	entries, err := os.ReadDir(macosDir)
	if err != nil {
		return "", fmt.Errorf("failed to read app bundle: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		executable := filepath.Join(macosDir, entry.Name())
		if err := ensureExecutable(executable); err != nil {
			return "", fmt.Errorf("failed to make app executable: %w", err)
		}
		return executable, nil
	}
	return "", fmt.Errorf("no executable found in %s", macosDir)
}

func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode()
	if mode&0o111 != 0 {
		return nil
	}
	return os.Chmod(path, mode|0o111)
}

func (s *DefaultService) runSmokeTestCommand(parent context.Context, workDir, command string, args []string, extraEnv []string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = time.Duration(TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	type result struct {
		output []byte
		err    error
	}
	resultCh := make(chan result, 1)

	go func() {
		output, err := cmd.CombinedOutput()
		resultCh <- result{output: output, err: err}
	}()

	select {
	case res := <-resultCh:
		return string(res.output), res.err
	case <-ctx.Done():
		terminateProcess(cmd)
		select {
		case res := <-resultCh:
			if ctx.Err() == context.DeadlineExceeded {
				return string(res.output), fmt.Errorf("command timed out after %s", timeout)
			}
			return string(res.output), fmt.Errorf("command cancelled")
		case <-time.After(2 * time.Second):
			if ctx.Err() == context.DeadlineExceeded {
				return "", fmt.Errorf("command timed out after %s", timeout)
			}
			return "", fmt.Errorf("command cancelled")
		}
	}
}

func terminateProcess(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return
	}
	_ = cmd.Process.Kill()
}

func (s *DefaultService) attemptTelemetryFallback(smokeTestID, scenarioName, platform, artifactPath, output string) {
	if s.telemetryIngestor == nil {
		return
	}

	telemetryPath := extractTelemetryPath(output)
	if telemetryPath == "" {
		appName := resolveAppNameFromArtifact(artifactPath, scenarioName)
		if appName != "" {
			telemetryPath = resolveTelemetryPath(platform, appName)
		}
	}
	if telemetryPath == "" {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Logs = append(status.Logs, "Telemetry fallback skipped: telemetry path not found")
		})
		return
	}

	events, err := readTelemetryEvents(telemetryPath, 500)
	if err != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploadError = fmt.Sprintf("telemetry fallback read failed: %v", err)
			status.Logs = append(status.Logs, fmt.Sprintf("Telemetry fallback failed: %v", err))
		})
		return
	}
	if len(events) == 0 {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Logs = append(status.Logs, "Telemetry fallback skipped: no events found")
		})
		return
	}

	_, ingested, err := s.telemetryIngestor.IngestEvents(scenarioName, "", "smoke-test-timeout", events)
	if err != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploadError = fmt.Sprintf("telemetry fallback upload failed: %v", err)
			status.Logs = append(status.Logs, fmt.Sprintf("Telemetry fallback upload failed: %v", err))
		})
		return
	}

	s.store.Update(smokeTestID, func(status *Status) {
		status.TelemetryUploaded = true
		status.TelemetryUploadError = ""
		status.Logs = append(status.Logs, fmt.Sprintf("Telemetry fallback uploaded %d events from %s", ingested, telemetryPath))
	})
}

func extractTelemetryPath(output string) string {
	const marker = "[Desktop App] Telemetry initialized at "
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, marker) {
			idx := strings.Index(line, marker)
			if idx >= 0 {
				return strings.TrimSpace(line[idx+len(marker):])
			}
		}
	}
	return ""
}

func resolveAppNameFromArtifact(artifactPath, fallback string) string {
	dir := filepath.Dir(artifactPath)
	pkgPath := filepath.Clean(filepath.Join(dir, "..", "package.json"))
	file, err := os.Open(pkgPath)
	if err != nil {
		return fallback
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		return fallback
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fallback
	}
	if name, ok := payload["name"].(string); ok && name != "" {
		return name
	}
	return fallback
}

func resolveTelemetryPath(platform, appName string) string {
	if appName == "" {
		return ""
	}
	switch platform {
	case "win":
		base := os.Getenv("APPDATA")
		if base == "" {
			return ""
		}
		return filepath.Join(base, appName, "deployment-telemetry.jsonl")
	case "mac":
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, "Library", "Application Support", appName, "deployment-telemetry.jsonl")
	default:
		config := os.Getenv("XDG_CONFIG_HOME")
		if config == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			config = filepath.Join(home, ".config")
		}
		return filepath.Join(config, appName, "deployment-telemetry.jsonl")
	}
}

func readTelemetryEvents(path string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 500
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	events := make([]map[string]interface{}, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if len(events) >= limit {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return events, err
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return events, err
	}
	return events, nil
}

// noopLogger is a no-op logger for when logging is not needed.
type noopLogger struct{}

func (l *noopLogger) Info(msg string, args ...interface{})  {}
func (l *noopLogger) Warn(msg string, args ...interface{})  {}
func (l *noopLogger) Error(msg string, args ...interface{}) {}
