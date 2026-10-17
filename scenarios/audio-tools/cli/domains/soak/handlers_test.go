package soak

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEvidencePathAnchorsRelativeExportToScenario(t *testing.T) {
	root := t.TempDir()
	h := &handlers{
		getenv: func(key string) string {
			if key == "AUDIO_TOOLS_SCENARIO_ROOT" {
				return root
			}
			return ""
		},
		getwd: func() (string, error) { return "/unused", nil },
	}

	got, err := h.resolveEvidencePath("coverage/soak.json")
	if err != nil {
		t.Fatalf("resolveEvidencePath: %v", err)
	}
	want := filepath.Join(root, "coverage", "soak.json")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestResolveEvidencePathRejectsRelativeEscape(t *testing.T) {
	root := t.TempDir()
	h := &handlers{
		getenv: func(key string) string {
			if key == "AUDIO_TOOLS_SCENARIO_ROOT" {
				return root
			}
			return ""
		},
		getwd: func() (string, error) { return "/unused", nil },
	}

	_, err := h.resolveEvidencePath("../coverage/soak.json")
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("error = %v, want relative escape rejection", err)
	}
}

func TestResolveEvidencePathPreservesExplicitAbsoluteExport(t *testing.T) {
	root := t.TempDir()
	h := &handlers{getenv: func(string) string { return root }}
	expected := filepath.Join(t.TempDir(), "evidence.json")

	got, err := h.resolveEvidencePath(expected)
	if err != nil {
		t.Fatalf("resolveEvidencePath: %v", err)
	}
	if got != expected {
		t.Fatalf("path = %q, want %q", got, expected)
	}
}
