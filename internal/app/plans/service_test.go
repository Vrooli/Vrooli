package plans

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServiceAddListShowArchive(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "demo-repo")
	svc := Service{
		Root: repo,
		Home: home,
		Now:  func() time.Time { return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC) },
	}

	added, err := svc.Add(AddRequest{
		Title:   "Plan Lifecycle Hygiene",
		Content: "# Plan\n\nBody\n",
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !added.Success {
		t.Fatalf("Add success = false")
	}
	if !strings.Contains(filepath.ToSlash(added.Plan.Path), "/.vrooli/plans/") {
		t.Fatalf("plan path = %q, want user-scoped .vrooli plan path", added.Plan.Path)
	}
	if strings.HasPrefix(filepath.Clean(added.Plan.Path), filepath.Clean(repo)) {
		t.Fatalf("plan path %q should not be under repo %q", added.Plan.Path, repo)
	}

	listed, err := svc.List(ListRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed.Plans) != 1 || listed.Plans[0].ID != added.Plan.ID {
		t.Fatalf("listed plans = %#v, want added plan", listed.Plans)
	}

	shown, err := svc.Show(ShowRequest{Ref: added.Plan.Slug})
	if err != nil {
		t.Fatalf("Show by slug: %v", err)
	}
	if !strings.Contains(shown.Content, "Body") {
		t.Fatalf("shown content = %q, want body", shown.Content)
	}

	archived, err := svc.Archive(ArchiveRequest{Ref: added.Plan.ID})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !archived.Plan.Archived {
		t.Fatalf("archived flag = false")
	}

	active, err := svc.List(ListRequest{})
	if err != nil {
		t.Fatalf("List active: %v", err)
	}
	if len(active.Plans) != 0 {
		t.Fatalf("active plans = %#v, want none after archive", active.Plans)
	}
}

func TestServiceImportAndExport(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "demo-repo")
	source := filepath.Join(t.TempDir(), "docs", "plans", "legacy-plan.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("# Legacy Plan\n\nKeep this.\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	svc := Service{Root: repo, Home: home}

	imported, err := svc.Import(ImportRequest{Path: source})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if imported.Plan.SourcePath != source {
		t.Fatalf("SourcePath = %q, want %q", imported.Plan.SourcePath, source)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source should be preserved by default: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "exported.md")
	exported, err := svc.Export(ExportRequest{Ref: imported.Plan.ID, To: dest})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if exported.Path != dest {
		t.Fatalf("exported path = %q, want %q", exported.Path, dest)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read exported: %v", err)
	}
	if !strings.Contains(string(data), "Keep this.") {
		t.Fatalf("exported content = %q, want imported content", string(data))
	}
}
