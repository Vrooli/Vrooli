package main

import (
	"os"
	"path/filepath"
	"testing"
)

func copySetpointFixture(t *testing.T, configDir string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "config", "setpoint.json"))
	if err != nil {
		t.Fatalf("read checked-in setpoint fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "setpoint.json"), raw, 0o600); err != nil {
		t.Fatalf("copy checked-in setpoint fixture: %v", err)
	}
}
