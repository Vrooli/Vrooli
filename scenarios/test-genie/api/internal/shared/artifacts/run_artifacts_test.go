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

func TestListRunVisuals(t *testing.T) {
	scenarioDir := t.TempDir()
	runID := "run-v"
	pagesDir := RunUISmokePagesDir(scenarioDir, runID)

	// page "/" with screenshot + video.
	rootDir := filepath.Join(pagesDir, "_root_")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "page.json"), []byte(`{"page":"/","label":"Home"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "screenshot.png"), []byte("PNGDATA"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "video.webm"), []byte("VID"), 0o644); err != nil {
		t.Fatal(err)
	}

	// page "/backlog" with screenshot only.
	backlogDir := filepath.Join(pagesDir, "backlog")
	if err := os.MkdirAll(backlogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backlogDir, "page.json"), []byte(`{"page":"/backlog","label":"Backlog"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backlogDir, "screenshot.png"), []byte("PNG2"), 0o644); err != nil {
		t.Fatal(err)
	}

	visuals, err := ListRunVisuals(scenarioDir, runID)
	if err != nil {
		t.Fatalf("ListRunVisuals error: %v", err)
	}
	if len(visuals) != 2 {
		t.Fatalf("expected 2 visuals, got %d", len(visuals))
	}
	// Sorted by page route: "/" before "/backlog".
	if visuals[0].Page != "/" || visuals[0].Label != "Home" {
		t.Fatalf("first visual = %+v", visuals[0])
	}
	if visuals[0].ScreenshotRelPath != "ui-smoke/pages/_root_/screenshot.png" {
		t.Fatalf("screenshot rel path = %q", visuals[0].ScreenshotRelPath)
	}
	if visuals[0].VideoRelPath != "ui-smoke/pages/_root_/video.webm" {
		t.Fatalf("video rel path = %q", visuals[0].VideoRelPath)
	}
	if visuals[0].ScreenshotSizeBytes != int64(len("PNGDATA")) {
		t.Fatalf("screenshot size = %d", visuals[0].ScreenshotSizeBytes)
	}
	if visuals[1].Page != "/backlog" || visuals[1].VideoRelPath != "" {
		t.Fatalf("second visual = %+v", visuals[1])
	}

	// Resolution of a returned rel path must work and stay inside the run dir.
	if _, err := ResolveRunArtifact(scenarioDir, runID, visuals[0].ScreenshotRelPath); err != nil {
		t.Fatalf("ResolveRunArtifact(%q) error: %v", visuals[0].ScreenshotRelPath, err)
	}
}

func TestListRunVisualsEmptyWhenNoCaptures(t *testing.T) {
	visuals, err := ListRunVisuals(t.TempDir(), "missing")
	if err != nil {
		t.Fatalf("expected nil error for missing run, got %v", err)
	}
	if len(visuals) != 0 {
		t.Fatalf("expected empty slice, got %d", len(visuals))
	}
}
