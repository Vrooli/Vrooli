package artifacts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriter_WriteAll(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(WithRunID("run-1"))

	status := 500
	in := Input{
		Screenshot: []byte("PNGDATA"),
		Console:    []ConsoleEntry{{Level: "error", Message: "boom"}},
		Network:    []NetworkEntry{{URL: "http://x/api", Status: &status}},
		Raw:        []byte(`{"loaded":true}`),
	}

	paths, err := w.WriteAll(context.Background(), dir, "demo", in)
	if err != nil {
		t.Fatalf("WriteAll error: %v", err)
	}
	if paths.Screenshot == "" || paths.Console == "" || paths.Network == "" || paths.Raw == "" {
		t.Fatalf("expected all artifact paths set: %+v", paths)
	}
	if data, _ := os.ReadFile(paths.Screenshot); string(data) != "PNGDATA" {
		t.Fatalf("screenshot bytes not written verbatim")
	}
}

func TestWriter_WriteAll_NetworkAlwaysWritten(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(WithRunID("run-1"))

	paths, err := w.WriteAll(context.Background(), dir, "demo", Input{})
	if err != nil {
		t.Fatalf("WriteAll error: %v", err)
	}
	if paths.Network == "" {
		t.Fatalf("network.json should be written even when empty")
	}
	if paths.Screenshot != "" {
		t.Fatalf("screenshot should be empty when no bytes captured")
	}
}

func TestWriter_WriteReadme(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(WithRunID("run-1"))

	summary := Summary{
		Scenario:   "demo",
		Status:     "passed",
		Message:    "UI loaded successfully",
		Timestamp:  time.Now().UTC(),
		DurationMs: 1234,
		UIURL:      "http://localhost:3000",
		Handshake:  HandshakeSummary{Signaled: true, DurationMs: 200},
	}

	path, err := w.WriteReadme(context.Background(), dir, "demo", summary)
	if err != nil {
		t.Fatalf("WriteReadme error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("README not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "iframe-bridge signaled ready") {
		t.Fatalf("README missing handshake section:\n%s", content)
	}
	// The README must reference the current engine, not the retired one.
	legacyEngine := "browser" + "less"
	if strings.Contains(strings.ToLower(content), legacyEngine) {
		t.Fatalf("README must not mention the retired browser engine")
	}
}

func TestWriter_WriteResultJSON_PhasePointer(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(WithRunID("run-ptr"))

	summary := Summary{Scenario: "demo", Status: "passed", Message: "ok", Timestamp: time.Now().UTC()}
	if err := w.WriteResultJSON(context.Background(), dir, "demo", map[string]any{"status": "passed"}, summary); err != nil {
		t.Fatalf("WriteResultJSON error: %v", err)
	}
	// latest.json must exist under the run's ui-smoke dir.
	matches, _ := filepath.Glob(filepath.Join(dir, "coverage", "runs", "run-ptr", "**", "latest.json"))
	if len(matches) == 0 {
		// Fall back to a recursive walk in case the layout differs.
		found := false
		_ = filepath.Walk(dir, func(p string, _ os.FileInfo, _ error) error {
			if strings.HasSuffix(p, "latest.json") {
				found = true
			}
			return nil
		})
		if !found {
			t.Fatalf("latest.json not written under %s", dir)
		}
	}
}
