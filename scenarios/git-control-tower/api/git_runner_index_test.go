package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStagedCandidateIndexPreservesUnusualPaths(t *testing.T) {
	changed := map[string]struct{}{
		"space and (parens).txt": {},
		"line\nbreak\tname.txt":  {},
		"deleted.txt":            {},
	}
	index := []byte("100644 aaaaaaaa 0\tspace and (parens).txt\x00" +
		"100644 bbbbbbbb 0\tline\nbreak\tname.txt\x00")
	entries, err := parseStagedCandidateIndex(index, changed)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries["space and (parens).txt"].oid != "aaaaaaaa" || entries["line\nbreak\tname.txt"].oid != "bbbbbbbb" {
		t.Fatalf("entries = %#v", entries)
	}
	if _, ok := entries["deleted.txt"]; ok {
		t.Fatal("deleted path unexpectedly appeared in the index candidate")
	}
}

func TestParseStagedCandidateIndexRefusesIncompleteEntries(t *testing.T) {
	tests := []struct {
		name  string
		entry []byte
	}{
		{name: "unmerged", entry: []byte("100644 aaaaaaaa 2\tconflict.txt\x00")},
		{name: "submodule", entry: []byte("160000 aaaaaaaa 0\tmodule\x00")},
		{name: "non utf8 path", entry: append([]byte("100644 aaaaaaaa 0\t"), append([]byte{0xff}, 0)...)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseStagedCandidateIndex(test.entry, map[string]struct{}{"conflict.txt": {}, "module": {}, string([]byte{0xff}): {}}); err == nil {
				t.Fatal("expected incomplete candidate error")
			}
		})
	}
}

func TestParseStagedCandidateBlobsPreservesBinaryBytesAndBounds(t *testing.T) {
	const oid = "aaaaaaaa"
	content := []byte{'a', 0, 'b', 'c'}
	input := append([]byte(fmt.Sprintf("%s blob %d\n", oid, len(content))), content...)
	input = append(input, '\n')
	got, err := parseStagedCandidateBlobs(input, []string{oid})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[oid], content) {
		t.Fatalf("blob = %v, want %v", got[oid], content)
	}
	if _, err := parseStagedCandidateBlobs([]byte(fmt.Sprintf("%s blob %d\n", oid, maxCandidateSecurityBytes+1)), []string{oid}); err == nil {
		t.Fatal("expected oversized candidate rejection")
	}
}

// TestExecGitRunnerReadOnlyLiveIndex is an opt-in integration probe for the
// exact host repository index. It is read-only and prints counts only; it
// never stages, commits, or logs candidate bytes or file names.
func TestExecGitRunnerReadOnlyLiveIndex(t *testing.T) {
	repo := os.Getenv("GCT_READONLY_INDEX_ROOT")
	if repo == "" {
		t.Skip("set GCT_READONLY_INDEX_ROOT for the read-only live index probe")
	}
	files, err := (&ExecGitRunner{}).ReadStagedCandidate(context.Background(), repo)
	if err != nil {
		t.Fatalf("read-only staged candidate probe failed: %v", err)
	}
	total := 0
	for _, content := range files {
		total += len(content)
	}
	if total > maxCandidateSecurityBytes {
		t.Fatalf("candidate reader returned %d bytes above its configured bound", total)
	}
	t.Logf("read %d changed staged blobs (%d bytes) without mutation", len(files), total)
}

func TestExecGitRunnerReadsMismatchedCandidateFromRealIndex(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is unavailable")
	}
	repo := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(git, args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
		}
	}
	runGit("init", "--quiet")
	runGit("config", "user.email", "candidate-test@example.invalid")
	runGit("config", "user.name", "Candidate Test")
	path := filepath.Join(repo, "config candidate.env")
	const stagedSecret = "AWS_ACCESS_KEY_ID=AKIA1234567890ABCDEF\n"
	const cleanCandidate = "MODE=demo\n"
	if err := os.WriteFile(path, []byte(stagedSecret), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "--", "config candidate.env")
	if err := os.WriteFile(path, []byte(cleanCandidate), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := &ExecGitRunner{}
	files, err := reader.ReadStagedCandidate(context.Background(), repo)
	if err != nil {
		t.Fatalf("read staged-secret candidate: %v", err)
	}
	if !bytes.Equal(files["config candidate.env"], []byte(stagedSecret)) {
		t.Fatal("candidate reader did not return the staged secret while the worktree was clean")
	}

	if err := os.WriteFile(path, []byte(cleanCandidate), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "--", "config candidate.env")
	if err := os.WriteFile(path, []byte(stagedSecret), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err = reader.ReadStagedCandidate(context.Background(), repo)
	if err != nil {
		t.Fatalf("read clean staged candidate: %v", err)
	}
	if !bytes.Equal(files["config candidate.env"], []byte(cleanCandidate)) {
		t.Fatal("candidate reader included a worktree-only secret")
	}
}
