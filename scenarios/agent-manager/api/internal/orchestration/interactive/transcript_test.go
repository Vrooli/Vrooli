package interactive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

func TestSlugifyCwd(t *testing.T) {
	cases := map[string]string{
		"/home/matthalloran8/Vrooli": "-home-matthalloran8-Vrooli",
		"/work/dir":                  "-work-dir",
		"relative/path":              "relative-path",
		"/a.b_c":                     "-a-b-c",
	}
	for in, want := range cases {
		if got := SlugifyCwd(in); got != want {
			t.Errorf("SlugifyCwd(%q) = %q, want %q", in, got, want)
		}
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestFindTranscript_Codex(t *testing.T) {
	runDir := t.TempDir()
	want := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13", "rollout-2026-07-13T10-00-00-abc.jsonl")
	writeFile(t, want)

	got, err := findTranscript(DiscoverParams{RunnerType: domain.RunnerTypeCodex, RunDir: runDir})
	if err != nil {
		t.Fatalf("findTranscript: %v", err)
	}
	if got != want {
		t.Fatalf("codex transcript: got %q, want %q", got, want)
	}
}

func TestFindTranscript_Grok(t *testing.T) {
	runDir := t.TempDir()
	want := filepath.Join(runDir, "grok", "sessions", "%2Fwork%2Fdir", "sess-123", "updates.jsonl")
	writeFile(t, want)

	got, err := findTranscript(DiscoverParams{RunnerType: domain.RunnerTypeGrok, RunDir: runDir})
	if err != nil {
		t.Fatalf("findTranscript: %v", err)
	}
	if got != want {
		t.Fatalf("grok transcript: got %q, want %q", got, want)
	}
}

func TestFindTranscript_ClaudeNewestAfterLaunch(t *testing.T) {
	home := t.TempDir()
	wd := "/work/dir"
	slugDir := filepath.Join(home, ".claude", "projects", SlugifyCwd(wd))

	stale := filepath.Join(slugDir, "old-session.jsonl")
	fresh := filepath.Join(slugDir, "new-session.jsonl")
	writeFile(t, stale)
	writeFile(t, fresh)

	launch := time.Now()
	// stale predates launch; fresh is written after.
	old := launch.Add(-1 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatalf("chtimes stale: %v", err)
	}
	newer := launch.Add(1 * time.Second)
	if err := os.Chtimes(fresh, newer, newer); err != nil {
		t.Fatalf("chtimes fresh: %v", err)
	}

	got, err := findTranscript(DiscoverParams{
		RunnerType: domain.RunnerTypeClaudeCode,
		WorkingDir: wd,
		LaunchedAt: launch,
		HomeDir:    home,
	})
	if err != nil {
		t.Fatalf("findTranscript: %v", err)
	}
	if got != fresh {
		t.Fatalf("claude transcript: got %q, want the post-launch file %q", got, fresh)
	}
}

func TestFindTranscript_NoMatchYet(t *testing.T) {
	// Empty run dir: codex transcript not written yet → "" (transient), no error.
	got, err := findTranscript(DiscoverParams{RunnerType: domain.RunnerTypeCodex, RunDir: t.TempDir()})
	if err != nil {
		t.Fatalf("findTranscript: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty path when no transcript yet, got %q", got)
	}
}

func TestFindTranscript_OpenCodeUnsupported(t *testing.T) {
	if _, err := findTranscript(DiscoverParams{RunnerType: domain.RunnerTypeOpenCode}); err == nil {
		t.Fatal("expected error for opencode transcript discovery (descoped)")
	}
}
