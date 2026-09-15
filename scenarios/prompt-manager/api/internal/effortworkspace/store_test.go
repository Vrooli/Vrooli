package effortworkspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, path string, value manifest) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAndReadUseCanonicalEffortReference(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "efforts", "ux-campaign")
	if err := os.MkdirAll(filepath.Join(workspace, "sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, filepath.Join(workspace, "effort.json"), manifest{EffortRef: "effort:ux-1", Slug: "ux-campaign", Stage: "intake"})
	if err := os.WriteFile(filepath.Join(workspace, "sources", "intent.md"), []byte("accepted intent"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := New(root, nil)
	resolved, err := s.resolve("effort:ux-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Slug != "ux-campaign" || resolved.Stage != "intake" {
		t.Fatalf("resolved metadata = %+v", resolved.Workspace)
	}
	content, err := s.Read(t.Context(), "effort:ux-1", "sources/intent.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if content.Content != "accepted intent" {
		t.Fatalf("content = %q", content.Content)
	}
}

func TestResolveSupportsLegacyWorkspaceReferenceWhenManifestHasNoOpaqueRef(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "efforts", "aquila-launch")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, filepath.Join(workspace, "effort.json"), manifest{Slug: "aquila-launch", Stage: "execution"})

	s := New(root, nil)
	resolved, err := s.resolve("workspace:/repo#aquila-launch")
	if err != nil {
		t.Fatalf("resolve legacy workspace reference: %v", err)
	}
	if resolved.EffortRef != "workspace:/repo#aquila-launch" || resolved.Slug != "aquila-launch" {
		t.Fatalf("resolved workspace = %+v", resolved.Workspace)
	}
}

func TestReadRejectsTraversalAndSymlink(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "efforts", "safe")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, filepath.Join(workspace, "effort.json"), manifest{EffortRef: "effort:safe"})
	outside := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "link.txt")); err != nil {
		t.Fatal(err)
	}

	s := New(root, nil)
	if _, err := s.Read(t.Context(), "effort:safe", "../outside.txt"); err == nil {
		t.Fatal("traversal read unexpectedly succeeded")
	}
	if _, err := s.Read(t.Context(), "effort:safe", "link.txt"); err == nil {
		t.Fatal("symlink read unexpectedly succeeded")
	}
}

func TestListFilesSkipsHiddenAndSymlinkEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "visible.md"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".secret"), []byte("hidden"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "visible.md"), filepath.Join(root, "link.md")); err != nil {
		t.Fatal(err)
	}
	files, err := listFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "visible.md" {
		t.Fatalf("files = %+v", files)
	}
}
