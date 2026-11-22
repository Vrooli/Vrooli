package annotations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoldRootResolvesRelativePathsAndKeepsAbsoluteOnes(t *testing.T) {
	root := goldRoot(t.TempDir())
	absolute := filepath.Join(t.TempDir(), "elsewhere.jsonl")
	if got, err := root.resolve("work.blocked_by_tool.holdout.jsonl"); err != nil || got != filepath.Join(string(root), "work.blocked_by_tool.holdout.jsonl") {
		t.Fatalf("relative resolved to %q, %v; want it under the root", got, err)
	}
	if got, err := root.resolve("mood/dissatisfied.tuning.jsonl"); err != nil || got != filepath.Join(string(root), "mood", "dissatisfied.tuning.jsonl") {
		t.Fatalf("nested relative resolved to %q, %v", got, err)
	}
	if got, err := root.resolve(" " + absolute + " "); err != nil || got != absolute {
		t.Fatalf("absolute resolved to %q, %v; want it as written", got, err)
	}
	if _, err := goldRoot("").resolve("gold.jsonl"); err == nil || !strings.Contains(err.Error(), "no gold root") {
		t.Fatalf("a relative path with no root: err = %v, want the missing root named", err)
	}
	if got, err := goldRoot("").resolve(absolute); err != nil || got != absolute {
		t.Fatalf("absolute with no root = %q, %v", got, err)
	}
}

func TestGoldRootNamesFilesRelativeWhenInsideTheRoot(t *testing.T) {
	root := goldRoot(t.TempDir())
	inside := filepath.Join(string(root), "mood", "pleased.holdout.jsonl")
	outside := filepath.Join(t.TempDir(), "other.jsonl")
	if got := root.name(inside); got != "mood/pleased.holdout.jsonl" {
		t.Fatalf("name(inside) = %q, want a root-relative name so the record survives moving the root", got)
	}
	if got := root.name(outside); got != outside {
		t.Fatalf("name(outside) = %q, want the absolute path", got)
	}
	if got := goldRoot("").name(outside); got != outside {
		t.Fatalf("name with no root = %q", got)
	}
}

func TestGoldRootCreateWritesPrivateFilesInAPrivateDirectory(t *testing.T) {
	root := goldRoot(filepath.Join(t.TempDir(), "gold"))
	file, resolved, err := root.create("mood/draft.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	if resolved != filepath.Join(string(root), "mood", "draft.jsonl") {
		t.Fatalf("created %q, want a path under the root", resolved)
	}
	assertMode(t, resolved, 0o600)
	assertMode(t, filepath.Dir(resolved), 0o700)

	// A file that already exists with looser permissions is tightened.
	loose := filepath.Join(string(root), "loose.jsonl")
	if err := os.WriteFile(loose, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	again, _, err := root.create("loose.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	again.Close()
	assertMode(t, loose, 0o600)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}

func TestBenchmarkReadsARelativeGoldFromTheRootAndRecordsItRootRelative(t *testing.T) {
	dir := t.TempDir()
	root := goldRoot(dir)
	row := `{"run_id":"r1","label":"work.blocked_by_tool","positive":true}` + "\n"
	for _, name := range []string{"spin.holdout.jsonl", "spin.tuning.jsonl"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(row), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The working directory holds a decoy: a relative --gold must not read it.
	t.Chdir(t.TempDir())
	if err := os.WriteFile("spin.holdout.jsonl", []byte(`{"run_id":"decoy","label":"work.blocked_by_tool","positive":false}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request, err := benchmarkRequestFrom(root, "spin.holdout.jsonl", "", "holdout", "spin.tuning.jsonl", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Rows) != 1 || request.Rows[0].RunID != "r1" || len(request.FitRows) != 1 {
		t.Fatalf("rows = %+v fit = %+v, want the root's files, not the working directory's", request.Rows, request.FitRows)
	}
	if request.GoldFile != "spin.holdout.jsonl" {
		t.Fatalf("recorded gold_file = %q, want the root-relative name", request.GoldFile)
	}
	if _, err := benchmarkRequestFrom(root, "missing.jsonl", "", "tuning", "", "", "", "", ""); err == nil {
		t.Fatal("a gold file missing from the root was accepted")
	}
}

func TestCalibrationAndRunsFromResolveAgainstTheRoot(t *testing.T) {
	dir := t.TempDir()
	root := goldRoot(dir)
	if err := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte(`{"run_id":"r1","scope_id":"r1-m1","label":"operator.pleased","positive":true}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request, err := calibrationRequestFrom(root, []string{"a.jsonl"}, "operator.pleased", "", "", "")
	if err != nil || len(request.Rows) != 1 || request.GoldFile != "a.jsonl" {
		t.Fatalf("calibration request = %+v, %v", request, err)
	}
	subjects, err := explicitSubjects(root, nil, []string{"a.jsonl"})
	if err != nil || strings.Join(subjects.MessageIDs, ",") != "r1-m1" {
		t.Fatalf("subjects = %+v, %v", subjects, err)
	}
}
