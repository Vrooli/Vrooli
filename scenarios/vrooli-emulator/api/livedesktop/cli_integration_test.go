//go:build integration

package livedesktop

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cliBinaryOnce sync.Once
	cliBinaryPath string
	cliBinaryErr  error
)

// buildCLIBinary compiles vrooli-emulator/cli once into a per-process temp dir.
// Subsequent calls reuse the cached path.
func buildCLIBinary(t *testing.T) string {
	t.Helper()
	cliBinaryOnce.Do(func() {
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			cliBinaryErr = nil
			t.Fatalf("runtime.Caller failed")
			return
		}
		// thisFile = .../scenarios/vrooli-emulator/api/livedesktop/cli_integration_test.go
		cliDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "cli")
		cliDir, err := filepath.Abs(cliDir)
		if err != nil {
			cliBinaryErr = err
			return
		}
		if _, err := os.Stat(cliDir); err != nil {
			cliBinaryErr = err
			return
		}

		buildDir, err := os.MkdirTemp("", "vrooli-emulator-cli-build-")
		if err != nil {
			cliBinaryErr = err
			return
		}
		out := filepath.Join(buildDir, "vrooli-emulator")
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "build", "-o", out, ".")
		cmd.Dir = cliDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			cliBinaryErr = &cliBuildError{err: err, stderr: stderr.String()}
			return
		}
		cliBinaryPath = out
	})

	require.NoError(t, cliBinaryErr, "failed to build CLI binary")
	require.NotEmpty(t, cliBinaryPath)
	return cliBinaryPath
}

type cliBuildError struct {
	err    error
	stderr string
}

func (e *cliBuildError) Error() string {
	return e.err.Error() + ": " + e.stderr
}

// runCLI execs the built CLI with API_BASE_URL set; returns combined stdout/stderr.
func runCLI(t *testing.T, binary, apiBaseURL string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = append(os.Environ(), "API_BASE_URL="+apiBaseURL)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCLISmoke_AllSessionSubcommands(t *testing.T) {
	if _, err := os.Stat(xclockPath); err != nil {
		t.Skipf("%s not found — install with `apt-get install -y x11-apps`", xclockPath)
	}

	f := setupIntegrationServer(t)
	bin := buildCLIBinary(t)

	// 1. session list (empty)
	out, err := runCLI(t, bin, f.URL, "session", "list")
	require.NoErrorf(t, err, "session list (empty) failed: %s", out)
	assert.Contains(t, out, "0 session(s)")

	// 2. session create --headless
	scenario := uniqueScenarioName()
	out, err = runCLI(t, bin, f.URL,
		"session", "create",
		"--scenario", scenario,
		"--headless",
		"--width", "1024",
		"--height", "768",
	)
	require.NoErrorf(t, err, "session create failed: %s", out)

	idMatch := regexp.MustCompile(`id=([0-9a-fA-F-]{36})`).FindStringSubmatch(out)
	require.NotNil(t, idMatch, "could not parse session id from create output:\n%s", out)
	sessionID := idMatch[1]
	assert.Regexp(t, regexp.MustCompile(`display=:\d+`), out, "headless create should surface display")

	// Always teardown via DELETE on the in-process service so leftover Xvfb dies.
	t.Cleanup(func() {
		_, _ = f.delete(t, "/api/v1/sessions/"+sessionID)
		assertNoStrayProcesses(t, scenario)
	})

	// 3. session list (after create)
	out, err = runCLI(t, bin, f.URL, "session", "list")
	require.NoErrorf(t, err, "session list (after create) failed: %s", out)
	assert.Contains(t, out, sessionID)

	// 4. session exec <id> launch_app --app-path /usr/bin/xclock
	out, err = runCLI(t, bin, f.URL,
		"session", "exec", sessionID, "launch_app",
		"--app-path", xclockPath,
	)
	require.NoErrorf(t, err, "session exec launch_app failed: %s", out)
	assert.Contains(t, out, `Action "launch_app" completed`)

	// 5. session logs <id> (single-shot)
	out, err = runCLI(t, bin, f.URL, "session", "logs", sessionID)
	require.NoErrorf(t, err, "session logs failed: %s", out)
	assert.Contains(t, out, sessionID)
	assert.Contains(t, out, "state=running")

	// 6. session destroy <id>
	out, err = runCLI(t, bin, f.URL, "session", "destroy", sessionID)
	require.NoErrorf(t, err, "session destroy failed: %s", out)
	assert.Contains(t, out, "Session "+sessionID+" destroyed")
}
