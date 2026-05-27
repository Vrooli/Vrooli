package artifacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRunArtifactRejectsTraversal(t *testing.T) {
	scenarioDir := t.TempDir()
	cases := []struct {
		name    string
		rel     string
		wantErr bool
	}{
		{"empty", "", true},
		{"escape-dotdot", "../../../../etc/passwd", true},
		{"escape-abs", "/etc/passwd", true},
		{"valid-video", "automation/login/video/a.webm", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			abs, err := ResolveRunArtifact(scenarioDir, "run-1", tc.rel)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got path %q", tc.rel, abs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			runRoot := RunDir(scenarioDir, "run-1")
			if rel, _ := filepath.Rel(runRoot, abs); rel != filepath.FromSlash(tc.rel) {
				t.Fatalf("resolved %q outside run root (rel=%q)", abs, rel)
			}
		})
	}
}

func TestListRunVideos(t *testing.T) {
	scenarioDir := t.TempDir()
	runID := "run-1"

	// No automation dir yet → empty, no error.
	got, err := ListRunVideos(scenarioDir, runID)
	if err != nil || len(got) != 0 {
		t.Fatalf("expected empty, got %v err=%v", got, err)
	}

	videoDir := filepath.Join(RunAutomationDir(scenarioDir, runID), "login-smoke", "video")
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(videoDir, "run.webm"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err = ListRunVideos(scenarioDir, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 video, got %d", len(got))
	}
	if got[0].Workflow != "login-smoke" || got[0].RelPath != "automation/login-smoke/video/run.webm" {
		t.Fatalf("unexpected video: %+v", got[0])
	}
	if got[0].SizeBytes != 4 {
		t.Fatalf("expected size 4, got %d", got[0].SizeBytes)
	}
}
