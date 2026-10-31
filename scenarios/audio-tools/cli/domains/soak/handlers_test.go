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

func TestDefaultFeedMSSeparatesVirtualAndRealtimeLanes(t *testing.T) {
	if got := defaultFeedMS("accelerated"); got != 50 {
		t.Fatalf("accelerated default feed = %d, want 50ms", got)
	}
	if got := defaultFeedMS("realtime"); got != 4000 {
		t.Fatalf("realtime default feed = %d, want 4000ms", got)
	}
}

func TestValidateSoakDriverHealthAllowsFreshSessionAlongsideForeignSession(t *testing.T) {
	err := validateSoakDriverHealth(soakDriverHealthDocument{
		Ready:    true,
		Sessions: 1,
		Capacity: 10,
	}, "http://bas")
	if err != nil {
		t.Fatalf("fresh isolated session should be admitted with spare capacity: %v", err)
	}
}

func TestValidateSoakDriverHealthReadsNestedCapacity(t *testing.T) {
	health := soakDriverHealthDocument{Ready: true, Sessions: 1}
	health.Summary.Capacity = 10
	if health.Capacity == 0 {
		health.Capacity = health.Summary.Capacity
	}
	if err := validateSoakDriverHealth(health, "http://bas"); err != nil {
		t.Fatalf("nested driver capacity should admit a fresh session: %v", err)
	}
}

func TestValidateSoakDriverHealthRejectsExhaustedCapacity(t *testing.T) {
	err := validateSoakDriverHealth(soakDriverHealthDocument{
		Ready:    true,
		Sessions: 10,
		Capacity: 10,
	}, "http://bas")
	if err == nil || !strings.Contains(err.Error(), "reached session capacity") {
		t.Fatalf("error = %v, want exhausted-capacity refusal", err)
	}
}

func TestValidateSoakDriverHealthAllowsLegacyHealthWithoutCapacity(t *testing.T) {
	if err := validateSoakDriverHealth(soakDriverHealthDocument{Ready: true, Sessions: 1}, "http://bas"); err != nil {
		t.Fatalf("legacy BAS health without capacity should defer admission to session start: %v", err)
	}
}

func TestValidateSoakDriverHealthRejectsNegativeCapacity(t *testing.T) {
	err := validateSoakDriverHealth(soakDriverHealthDocument{Ready: true, Capacity: -1}, "http://bas")
	if err == nil || !strings.Contains(err.Error(), "invalid session capacity") {
		t.Fatalf("error = %v, want invalid-capacity refusal", err)
	}
}
