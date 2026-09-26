package cliutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "vrooli-port-lookup-tests-")
	if err != nil {
		os.Exit(1)
	}
	os.Setenv(portLookupStatsFileEnv, filepath.Join(dir, "stats.log"))
	marker, hadMarker := os.LookupEnv(AgentSessionEnv)
	_ = os.Unsetenv(AgentSessionEnv)
	code := m.Run()
	if hadMarker {
		_ = os.Setenv(AgentSessionEnv, marker)
	}
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
