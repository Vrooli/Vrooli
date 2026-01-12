package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const smokeTestTimeoutSeconds = 30

func currentSmokeTestPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "win"
	case "darwin":
		return "mac"
	default:
		return "linux"
	}
}

func (s *Server) performSmokeTest(ctx context.Context, smokeTestID, scenarioName, artifactPath, platform string) {
	if _, ok := s.smokeTests.Get(smokeTestID); !ok {
		return
	}
	defer s.clearSmokeTestCancel(smokeTestID)

	defer func() {
		if r := recover(); r != nil {
			s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
				status.Status = "failed"
				status.Error = fmt.Sprintf("panic: %v", r)
				now := time.Now()
				status.CompletedAt = &now
			})
		}
	}()

	s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
		status.Logs = append(status.Logs, fmt.Sprintf("Starting smoke test for %s on %s (artifact: %s)", scenarioName, platform, filepath.Base(artifactPath)))
	})

	if ctx.Err() != nil {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
			status.Status = "failed"
			status.Error = "smoke test cancelled"
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	if _, err := os.Stat(artifactPath); err != nil {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
			status.Status = "failed"
			status.Error = fmt.Sprintf("artifact not found: %v", err)
			status.Logs = append(status.Logs, "Smoke test failed: artifact missing")
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	commandName, commandArgs, displayCommand, err := resolveSmokeTestCommand(platform, artifactPath)
	if err != nil {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
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
		fmt.Sprintf("SMOKE_TEST_TIMEOUT_MS=%d", smokeTestTimeoutSeconds*1000),
		fmt.Sprintf("SMOKE_TEST_UPLOAD_URL=%s", uploadURL),
	}
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" {
		if _, err := exec.LookPath("xvfb-run"); err == nil {
			commandArgs = append([]string{"-a", commandName}, commandArgs...)
			commandName = "xvfb-run"
			displayCommand = "xvfb-run -a " + displayCommand
		} else {
			s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
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
	output, err := s.runSmokeTestCommand(ctx, workDir, commandName, commandArgs, smokeEnv, time.Duration(smokeTestTimeoutSeconds)*time.Second)
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

	s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
		status.Logs = append(status.Logs, logEntry)
	})

	if strings.Contains(output, "SMOKE_TEST_UPLOAD=ok") {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
			status.TelemetryUploaded = true
			status.TelemetryUploadError = ""
		})
	} else if strings.Contains(output, "SMOKE_TEST_UPLOAD=error") {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
			status.TelemetryUploadError = "telemetry upload failed (see logs)"
		})
	}

	if err != nil {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
			status.Status = "failed"
			status.Error = fmt.Sprintf("smoke-test failed: %v", err)
			status.Logs = append(status.Logs, "Smoke test failed")
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
		status.Status = "passed"
		status.Logs = append(status.Logs, "Smoke test passed")
		now := time.Now()
		status.CompletedAt = &now
	})
}

func resolveSmokeTestCommand(platform, artifactPath string) (string, []string, string, error) {
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

func (s *Server) runSmokeTestCommand(parent context.Context, workDir, command string, args []string, extraEnv []string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = time.Duration(smokeTestTimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("command timed out after %s", timeout)
	}
	if ctx.Err() == context.Canceled {
		return string(output), fmt.Errorf("command cancelled")
	}
	return string(output), err
}
