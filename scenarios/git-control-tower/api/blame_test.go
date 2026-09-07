package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git-control-tower/internal/provenance"
)

func TestJoinBlameEvidenceDoesNotPromoteMixedLineCommits(t *testing.T) {
	joined := JoinBlameEvidence(BlameFile{
		Path:          "mixed.go",
		ContentDigest: "sha256:content",
		Lines:         []BlameLine{{Commit: "commit-a"}, {Commit: "commit-b"}},
	}, true, []provenance.Evidence{{ContentDigest: "sha256:content", CommitID: "commit-a", Visibility: "public"}})
	if joined.Standing == provenance.ExactContent || joined.Standing == provenance.CommitFile {
		t.Fatalf("mixed line ownership must not produce file-level authorship: %+v", joined)
	}
}

type fakeBlameRunner struct {
	output map[string][]byte
	err    map[string]error
}

func (f fakeBlameRunner) Blame(_ context.Context, _, _, path string, _ int) ([]byte, error) {
	if err := f.err[path]; err != nil {
		return nil, err
	}
	return f.output[path], nil
}

func TestParseBlamePorcelainRangeAndMetadata(t *testing.T) {
	data := []byte("0123456789abcdef 1 1 1\nauthor Ada\nauthor-time 1700000000\nsummary first\nfilename old.txt\n\tfirst\n0123456789abcdef 2 2 1\nauthor Ada\nauthor-time 1700000001\nsummary second\n\tsecond\n")
	lines, truncated, err := parseBlamePorcelain(data, "file.txt", 10, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(lines) != 1 || lines[0].Content != "second" || lines[0].Author != "Ada" || lines[0].OriginalPath != "" {
		t.Fatalf("unexpected blame result: truncated=%v lines=%+v", truncated, lines)
	}
}

func TestReadBlameBoundsSortsAndClassifies(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := fakeBlameRunner{
		output: map[string][]byte{
			"tracked.txt": []byte("abcdef12 1 1 1\nauthor A\nauthor-time 1\nsummary s\n\tx\n"),
		},
		err: map[string]error{"missing.txt": errors.New("fatal: path missing does not exist")},
	}
	got, err := ReadBlame(context.Background(), runner, BlameRequest{RepoDir: root, Paths: []string{"missing.txt", "tracked.txt"}, MaxLines: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got.Files[0].Path != "missing.txt" || got.Files[0].Status != "deleted" || got.Files[1].Status != "native_commit" || got.Files[1].ContentDigest == "" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if !reflect.DeepEqual(got.Files[0].Lines, []BlameLine(nil)) {
		t.Fatalf("unavailable file returned lines: %+v", got.Files[0].Lines)
	}
}

func TestReadBlameRejectsTraversalAndCapsPaths(t *testing.T) {
	root := t.TempDir()
	runner := fakeBlameRunner{}
	if _, err := ReadBlame(context.Background(), runner, BlameRequest{RepoDir: root, Paths: []string{"../outside"}}); err == nil {
		t.Fatal("expected traversal rejection")
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ReadBlame(context.Background(), runner, BlameRequest{RepoDir: root, Paths: []string{"*.txt"}, MaxPaths: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Files) != 1 || !got.Truncated {
		t.Fatalf("expected deterministic path truncation: %+v", got)
	}
}

func TestParseBlamePorcelainRejectsMalformedOutput(t *testing.T) {
	if _, _, err := parseBlamePorcelain([]byte("\twithout header\n"), "x", 10, 0, 0); err == nil {
		t.Fatal("expected malformed output error")
	}
}
