package audits

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// writeFile creates a file with content, making parent dirs as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestWalkTree_CountsFilesDirsSymlinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "alpha")
	writeFile(t, filepath.Join(root, "sub/b.txt"), "bravo")
	writeFile(t, filepath.Join(root, "sub/c.txt"), "charlie")
	if runtime.GOOS != "windows" {
		if err := os.Symlink("a.txt", filepath.Join(root, "link")); err != nil {
			t.Fatalf("symlink: %v", err)
		}
	}

	res, err := walkTree(root, walkOptions{includeContentHash: true}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("walkTree: %v", err)
	}
	s := res.summary
	if s.Files != 3 {
		t.Errorf("files = %d, want 3", s.Files)
	}
	if s.Directories != 1 {
		t.Errorf("directories = %d, want 1 (sub)", s.Directories)
	}
	wantBytes := int64(len("alpha") + len("bravo") + len("charlie"))
	if s.RegularBytes != wantBytes {
		t.Errorf("regularBytes = %d, want %d", s.RegularBytes, wantBytes)
	}
	if runtime.GOOS != "windows" && s.Symlinks != 1 {
		t.Errorf("symlinks = %d, want 1", s.Symlinks)
	}
	if s.PathListSHA256 == "" || s.TreeContentSHA == "" {
		t.Errorf("expected non-empty hashes, got path=%q content=%q", s.PathListSHA256, s.TreeContentSHA)
	}
}

func TestWalkTree_PathHashIndependentOfCreationOrder(t *testing.T) {
	build := func() string {
		root := t.TempDir()
		// Create in different orders across the two trees.
		return root
	}
	rootA := build()
	writeFile(t, filepath.Join(rootA, "z.txt"), "z")
	writeFile(t, filepath.Join(rootA, "a.txt"), "a")
	writeFile(t, filepath.Join(rootA, "m/n.txt"), "n")

	rootB := build()
	writeFile(t, filepath.Join(rootB, "a.txt"), "a")
	writeFile(t, filepath.Join(rootB, "m/n.txt"), "n")
	writeFile(t, filepath.Join(rootB, "z.txt"), "z")

	a, err := walkTree(rootA, walkOptions{includeContentHash: true}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("walk A: %v", err)
	}
	b, err := walkTree(rootB, walkOptions{includeContentHash: true}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("walk B: %v", err)
	}
	if a.summary.PathListSHA256 != b.summary.PathListSHA256 {
		t.Errorf("path-list hash differs across creation order: %s vs %s", a.summary.PathListSHA256, b.summary.PathListSHA256)
	}
	if a.summary.TreeContentSHA != b.summary.TreeContentSHA {
		t.Errorf("content hash differs across creation order")
	}
}

func TestWalkTree_ContentHashChangesOnContentChange(t *testing.T) {
	root := t.TempDir()
	const orig, changed = "original", "changed-and-longer"
	if len(orig) == len(changed) {
		t.Fatalf("test precondition: lengths must differ")
	}
	writeFile(t, filepath.Join(root, "a.txt"), orig)
	before, _ := walkTree(root, walkOptions{includeContentHash: true}, time.Unix(0, 0))

	writeFile(t, filepath.Join(root, "a.txt"), changed)
	after, _ := walkTree(root, walkOptions{includeContentHash: true}, time.Unix(0, 0))

	if before.summary.TreeContentSHA == after.summary.TreeContentSHA {
		t.Errorf("content hash unchanged after editing a file")
	}
	// Path-list hash also changes because file size differs.
	if before.summary.PathListSHA256 == after.summary.PathListSHA256 {
		t.Errorf("path-list hash unchanged after size change")
	}
}

func TestWalkTree_ContentHashOmittedWhenDisabled(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "alpha")
	res, _ := walkTree(root, walkOptions{includeContentHash: false}, time.Unix(0, 0))
	if res.summary.TreeContentSHA != "" {
		t.Errorf("expected empty content hash when disabled, got %q", res.summary.TreeContentSHA)
	}
	if res.summary.PathListSHA256 == "" {
		t.Errorf("path-list hash should still be computed")
	}
}

func TestWalkTree_DetectsSQLiteByMagic(t *testing.T) {
	root := t.TempDir()
	// A real SQLite db (magic header present).
	dbPath := filepath.Join(root, "events.db")
	makeSQLiteDB(t, dbPath, "CREATE TABLE t (id INTEGER)")
	// A non-SQLite file named .db (no magic) must NOT be detected.
	writeFile(t, filepath.Join(root, "decoy.db"), "not a database")

	res, err := walkTree(root, walkOptions{detectSQLite: true}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(res.sqlite) != 1 {
		t.Fatalf("expected 1 sqlite candidate, got %d: %+v", len(res.sqlite), res.sqlite)
	}
	if res.sqlite[0].Rel != "events.db" {
		t.Errorf("candidate rel = %q, want events.db", res.sqlite[0].Rel)
	}
}

func TestWalkTree_MaxModTimeTracked(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old.txt")
	writeFile(t, old, "old")
	recent := filepath.Join(root, "recent.txt")
	writeFile(t, recent, "recent")

	past := time.Now().Add(-48 * time.Hour)
	future := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}
	if err := os.Chtimes(recent, future, future); err != nil {
		t.Fatalf("chtimes recent: %v", err)
	}

	res, _ := walkTree(root, walkOptions{}, time.Unix(0, 0))
	if res.summary.MaxModTime.Before(future.Add(-time.Second)) {
		t.Errorf("MaxModTime = %v, want >= %v", res.summary.MaxModTime, future)
	}
}

// makeSQLiteDB creates a real SQLite database file at path with the given DDL.
func makeSQLiteDB(t *testing.T, path, ddl string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), ddl); err != nil {
		t.Fatalf("exec ddl: %v", err)
	}
}
