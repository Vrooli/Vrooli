package privsep

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/envkit-go"
	"github.com/stretchr/testify/require"
)

func TestOSStepRunnerDoesNotWaitForInheritedPipeHandles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the regression command uses the POSIX shell available on Linux and macOS")
	}

	dir := t.TempDir()
	childPIDPath := filepath.Join(dir, "child.pid")
	cleanupChild := func() {
		pidBytes, err := os.ReadFile(childPIDPath)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
		if err != nil {
			return
		}
		if child, err := os.FindProcess(pid); err == nil {
			_ = child.Kill()
			_, _ = child.Wait()
		}
	}
	defer cleanupChild()

	done := make(chan struct {
		code int
		err  error
	}, 1)
	go func() {
		code, err := (osStepRunner{}).Run(context.Background(), []string{"sh", "-c", "sleep 30 & echo $! > \"$1\"; exit 0", "sh", childPIDPath}, dir, func(string) {})
		done <- struct {
			code int
			err  error
		}{code: code, err: err}
	}()

	select {
	case result := <-done:
		require.NoError(t, result.err)
		require.Equal(t, 0, result.code)
	case <-time.After(2 * time.Second):
		t.Fatal("step runner waited for a descendant's inherited stdout/stderr handles")
	}
}

func TestBoundaryEnvironmentReplacesInheritedValues(t *testing.T) {
	got := envkit.WithOverlay(
		envkit.Env{"HOME=/old", "PATH=/bin", "KEEP=yes"},
		envkit.ForeignScenario,
		envkit.Env{"HOME=/runner", "VROOLI_ROOT=/tmp/work", "PATH=/custom"},
	)
	require.Equal(t, envkit.Env{"HOME=/runner", "KEEP=yes", "PATH=/custom", "VROOLI_ROOT=/tmp/work"}, got)
}
