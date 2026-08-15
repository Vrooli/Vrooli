package main

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2SessionReturnsComputedFirstUnsatisfiedStep(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","session":{"step":1},"scenarios":{"demo":{"enabled":true}},"resources":{},"host_tools":{}}`)
	w := doGet(t, NewServer(), "/api/v2/session")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"first_unsatisfied_step":7`) {
		t.Fatalf("session = %d: %s", w.Code, w.Body.String())
	}
}
