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

func (s *Server) performSmokeTest(ctx context.Context, smokeTestID, scenarioName, desktopPath, artifactPath, platform string) {
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

	steps := make([]struct {
		name    string
		command string
		env     []string
		timeout time.Duration
	}, 0, 3)

	if _, err := os.Stat(filepath.Join(desktopPath, "node_modules")); err != nil {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
			status.Status = "failed"
			status.Error = "node_modules missing; run npm install in platforms/electron before smoke testing"
			status.Logs = append(status.Logs, "Smoke test failed: node_modules missing")
			now := time.Now()
			status.CompletedAt = &now
		})
		return
	}

	if _, err := os.Stat(filepath.Join(desktopPath, "dist", "main.js")); err != nil {
		s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
			status.Status = "failed"
			status.Error = "dist/main.js missing; run npm run build in platforms/electron before smoke testing"
			status.Logs = append(status.Logs, "Smoke test failed: dist/main.js missing")
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
	steps = append(steps, struct {
		name    string
		command string
		env     []string
		timeout time.Duration
	}{"smoke-test", "npm run smoke-test", smokeEnv, time.Duration(smokeTestTimeoutSeconds) * time.Second})

	for _, step := range steps {
		if ctx.Err() != nil {
			s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
				status.Status = "failed"
				status.Error = "smoke test cancelled"
				now := time.Now()
				status.CompletedAt = &now
			})
			return
		}

		command := step.command
		if step.name == "smoke-test" && runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" {
			if _, err := exec.LookPath("xvfb-run"); err == nil {
				command = fmt.Sprintf("xvfb-run -a %s", command)
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

		output, err := s.runSmokeTestCommand(ctx, desktopPath, command, step.env, step.timeout)
		logEntry := fmt.Sprintf("[%s] %s", step.name, command)
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
				status.Error = fmt.Sprintf("%s failed: %v", step.name, err)
				status.Logs = append(status.Logs, "Smoke test failed")
				now := time.Now()
				status.CompletedAt = &now
			})
			return
		}
	}

	s.smokeTests.Update(smokeTestID, func(status *SmokeTestStatus) {
		status.Status = "passed"
		status.Logs = append(status.Logs, "Smoke test passed")
		now := time.Now()
		status.CompletedAt = &now
	})
}

func (s *Server) runSmokeTestCommand(parent context.Context, desktopPath, command string, extraEnv []string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = time.Duration(smokeTestTimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = desktopPath
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
