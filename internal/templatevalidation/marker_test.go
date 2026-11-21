package templatevalidation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunMarkerRoundTrip(t *testing.T) {
	repoRoot := t.TempDir()
	tempRoot := filepath.Join(t.TempDir(), "vrooli-template-deep-alpha")
	scenarioPath := filepath.Join(tempRoot, "scenarios", "template-validation-react-vite-deep")
	marker, err := NewRunMarker(NewRunMarkerInput{
		RepoRoot:     repoRoot,
		Template:     "react-vite",
		ScenarioID:   "template-validation-react-vite-deep",
		ScenarioPath: scenarioPath,
		TempRoot:     tempRoot,
		Retained:     true,
		Now:          time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewRunMarker: %v", err)
	}
	marker.RelocationArtifacts = []string{filepath.Join(repoRoot, "packages", "proto", "schemas", marker.ScenarioID)}
	if err := WriteMarker(marker); err != nil {
		t.Fatalf("WriteMarker: %v", err)
	}
	got, err := ReadMarker(MarkerPath(tempRoot))
	if err != nil {
		t.Fatalf("ReadMarker: %v", err)
	}
	if got.RunID == "" || !strings.HasPrefix(got.RunID, marker.ScenarioID+"-20260506-120000-") {
		t.Fatalf("run id = %q", got.RunID)
	}
	if err := ValidateMarker(repoRoot, MarkerPath(tempRoot), got); err != nil {
		t.Fatalf("ValidateMarker: %v", err)
	}
}

func TestValidateMarkerRejectsUnsafePaths(t *testing.T) {
	repoRoot := t.TempDir()
	tempRoot := filepath.Join(t.TempDir(), "vrooli-template-deep-alpha")
	base := RunMarker{
		Version:      MarkerVersion,
		RunID:        "run-1",
		RepoRoot:     repoRoot,
		Template:     "react-vite",
		ScenarioID:   "template-validation-react-vite-deep",
		ScenarioPath: filepath.Join(tempRoot, "scenarios", "template-validation-react-vite-deep"),
		TempRoot:     tempRoot,
		CreatedAt:    time.Now().UTC(),
	}
	if err := os.MkdirAll(filepath.Dir(MarkerPath(tempRoot)), 0o755); err != nil {
		t.Fatalf("mkdir marker dir: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(RunMarker) RunMarker
		want   string
	}{
		{
			name: "scenario outside temp root",
			mutate: func(marker RunMarker) RunMarker {
				marker.ScenarioPath = filepath.Join(repoRoot, "scenarios", marker.ScenarioID)
				return marker
			},
			want: "outside temp root",
		},
		{
			name: "artifact outside repo root",
			mutate: func(marker RunMarker) RunMarker {
				marker.RelocationArtifacts = []string{filepath.Join(t.TempDir(), "packages", "proto", "schemas", marker.ScenarioID)}
				return marker
			},
			want: "outside repo root",
		},
		{
			name: "temp root mismatch",
			mutate: func(marker RunMarker) RunMarker {
				marker.TempRoot = filepath.Join(t.TempDir(), "vrooli-template-deep-other")
				return marker
			},
			want: "does not match marker directory",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMarker(repoRoot, MarkerPath(tempRoot), tc.mutate(base))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateMarker error = %v, want %q", err, tc.want)
			}
		})
	}
}
