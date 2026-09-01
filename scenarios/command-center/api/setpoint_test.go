package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSetpointRejectsMalformedInput(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setpoint.json")
	if err := os.WriteFile(p, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(p, nil); err == nil {
		t.Fatal("expected loud parse failure")
	}
}
func TestLoadSetpointRejectsBarEqualToAuthoredValue(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setpoint.json")
	body := `{"schemaVersion":"1.0.0","bars":[{"metricId":"m","bar":10,"authoredAgainst":10,"decisionRef":"d"}]}`
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSetpoint(p, nil); err == nil {
		t.Fatal("expected equal-bar failure")
	}
}
