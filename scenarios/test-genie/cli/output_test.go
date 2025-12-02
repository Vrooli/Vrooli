package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecutionPrinterIncludesGuidesAndInsights(t *testing.T) {
	tmp := t.TempDir()

	unitLog := filepath.Join(tmp, "unit.log")
	if err := os.WriteFile(unitLog, []byte("❌ test failed: sample\n"), 0o644); err != nil {
		t.Fatalf("write unit log: %v", err)
	}
	integrationLog := filepath.Join(tmp, "integration.log")
	if err := os.WriteFile(integrationLog, []byte("UI bundle is stale and must be rebuilt\n"), 0o644); err != nil {
		t.Fatalf("write integration log: %v", err)
	}

	resp := ExecuteResponse{
		Success:     false,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		CompletedAt: time.Now().Add(30 * time.Second).UTC().Format(time.RFC3339),
		PhaseSummary: PhaseSummary{
			Total:           2,
			Failed:          2,
			DurationSeconds: 30,
		},
		Phases: []ExecutePhase{
			{Name: "unit", Status: "failed", DurationSeconds: 8, LogPath: unitLog},
			{Name: "integration", Status: "failed", DurationSeconds: 12, LogPath: integrationLog},
		},
	}

	var buf bytes.Buffer
	printer := newExecutionPrinter(&buf, "demo-scenario", "", nil, nil, false, nil)
	printer.Print(resp)

	out := buf.String()
	for _, token := range []string{
		"Quick fix guide:",
		"Phase-specific debug guides:",
		"UI bundle",
		"artifact roots:",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("expected output to contain %q\n----\n%s\n----", token, out)
		}
	}
}
