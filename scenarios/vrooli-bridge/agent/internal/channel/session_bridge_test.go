package channel

import (
	"log"
	"runtime"
	"strings"
	"testing"
	"time"

	"vrooli-bridge/agent/internal/config"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/repo-contract-go/repocontracttest"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
)

func TestOpenNodeSessionUsesNativePTYAndResizes(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "creack/pty reports unsupported on Windows; the agent uses the documented pipe fallback")
	}

	c := NewClient(config.Config{}, WithLogger(log.Default()))
	c.openNodeSession("session-pty", &sessionv1.Open{Shell: "/bin/sh"})
	t.Cleanup(func() { c.closeNodeSession("session-pty", "test_done") })

	var current *nodeSession
	require.Eventually(t, func() bool {
		c.mu.Lock()
		current = c.sessions["session-pty"]
		c.mu.Unlock()
		return current != nil
	}, time.Second, time.Millisecond)
	require.NotNil(t, current.terminal, "supported Unix targets must allocate a native PTY")

	c.resizeNodeSession("session-pty", &sessionv1.Resize{Columns: 120, Rows: 40})
}

func TestInteractiveShellDefaultsBeforeCleaningEmptyInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "default shell selection is platform-specific on Windows")
	}
	t.Setenv("SHELL", "")
	require.Equal(t, "/bin/sh", interactiveShell(""))
	require.Equal(t, "/bin/sh", interactiveShell("   "))
}

func TestInteractiveShellFallsBackWhenDefaultIsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "POSIX fallback shell selection is platform-specific on Windows")
	}
	t.Setenv("SHELL", "/does/not/exist/zsh")
	require.Equal(t, "/bin/sh", interactiveShell(""))
}

func TestInteractiveCommandEnvMakesConfiguredVrooliBinDiscoverable(t *testing.T) {
	env := interactiveCommandEnv("/Users/test/.vrooli/bin/vrooli", []string{"PATH=/usr/bin:/bin", "HOME=/Users/test"})
	require.Contains(t, env, "HOME=/Users/test")
	var pathValue string
	for _, value := range env {
		if strings.HasPrefix(value, "PATH=") {
			pathValue = strings.TrimPrefix(value, "PATH=")
		}
	}
	require.Equal(t, "/Users/test/.vrooli/bin:/usr/bin:/bin", pathValue)
}
