package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunRequiresCorpus(t *testing.T) {
	t.Setenv("FAKE_AGENT_CORPUS", "")
	var stdout, stderr bytes.Buffer
	if got := run(&stdout, &stderr); got != 2 {
		t.Fatalf("run() exit = %d, want 2", got)
	}
	if got := stderr.String(); got != "FAKE_AGENT_CORPUS is required\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRunReplaysCorpusAndWritesTagMarker(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus.jsonl")
	if err := os.WriteFile(corpus, []byte("{\"type\":\"message\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "marker")
	t.Setenv("FAKE_AGENT_CORPUS", corpus)
	t.Setenv("FAKE_AGENT_TAG_MARKER", marker)
	t.Setenv("REPLAY_AGENT_TAG", "run-tag")

	var stdout, stderr bytes.Buffer
	if got := run(&stdout, &stderr); got != 0 {
		t.Fatalf("run() exit = %d, stderr = %q", got, stderr.String())
	}
	if got := stdout.String(); got != "{\"type\":\"message\"}\n" {
		t.Fatalf("stdout = %q", got)
	}
	markerContents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(markerContents); got != "REPLAY_AGENT_TAG=run-tag" {
		t.Fatalf("marker = %q", got)
	}
}

func TestRunReturnsFailureForFailureCorpus(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus.jsonl")
	if err := os.WriteFile(corpus, []byte("{\"type\":\"error\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_AGENT_CORPUS", corpus)
	t.Setenv("FAKE_AGENT_TAG_MARKER", "")

	var stdout, stderr bytes.Buffer
	if got := run(&stdout, &stderr); got != 1 {
		t.Fatalf("run() exit = %d, stderr = %q", got, stderr.String())
	}
}
