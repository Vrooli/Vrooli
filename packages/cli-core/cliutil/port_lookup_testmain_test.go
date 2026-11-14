package cliutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain runs the package detached from any agent session that launches it,
// as cliapptest.RunWithoutAgentSession does for other packages (which this
// package cannot import): the ambient agent variables are cleared and TMPDIR
// moves AgentCredentialSocket away from the live broker. Tests that exercise
// agent behaviour set what they need with t.Setenv.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "vrooli-cliutil-tests-")
	if err != nil {
		os.Exit(1)
	}
	for _, key := range AmbientAgentEnvironment() {
		_ = os.Unsetenv(key)
	}
	_ = os.Setenv("TMPDIR", dir)
	_ = os.Setenv(portLookupStatsFileEnv, filepath.Join(dir, "stats.log"))
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
