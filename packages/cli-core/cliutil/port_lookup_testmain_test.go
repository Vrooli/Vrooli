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
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
