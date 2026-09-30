package assistantmigration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryExportAndReconcilePreserveStableRecords(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data", "tasks"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "data", "contexts"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data/tasks", "issue-a.md"), []byte("keep this history\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data/contexts", "capture-a.txt"), []byte("private context\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := Inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Records) != 2 || manifest.Records[0].RelativePath != "data/contexts/capture-a.txt" {
		t.Fatalf("unexpected inventory: %+v", manifest.Records)
	}
	destination := filepath.Join(t.TempDir(), "export")
	exported, err := Export(root, destination)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Records[0].SHA256 != manifest.Records[0].SHA256 {
		t.Fatal("export changed record identity")
	}
	if err := Reconcile(root, manifest); err != nil {
		t.Fatalf("unchanged source did not reconcile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "manifest.json")); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileRejectsChangedSourceAndExportNeverTargetsSource(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data", "tasks"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "data/tasks", "issue-a.md")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := Inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Reconcile(root, manifest); err == nil {
		t.Fatal("changed source reconciled")
	}
	if _, err := Export(root, root); err == nil {
		t.Fatal("export accepted source destination")
	}
}
