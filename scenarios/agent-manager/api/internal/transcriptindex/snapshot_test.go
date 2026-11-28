package transcriptindex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotVerificationRejectsRenameDuringRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.jsonl")
	data := []byte("{\"type\":\"test\"}\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	other := path + ".new"
	if err := os.WriteFile(other, data, 0600); err != nil {
		t.Fatal(err)
	}
	err = verifyPrefixWithHook(path, info, data, func() {
		if err := os.Rename(other, path); err != nil {
			t.Fatal(err)
		}
	})
	if err == nil {
		t.Fatal("path replacement during verification was accepted")
	}
}

// A live source grows between discovery and staging and ends in a line still
// being written: the snapshot takes what is there, stops at the last complete
// line, and verification accepts further growth but not a changed prefix.
func TestSnapshotStagesAGrowingSourceToItsLastCompleteLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	first := "{\"timestamp\":\"2026-11-20T22:20:52.811Z\",\"ordinal\":0,\"type\":\"session_meta\",\"payload\":{}}\n"
	if err := os.WriteFile(path, []byte(first), 0o600); err != nil {
		t.Fatal(err)
	}
	candidate, ok := Identify(Root{Path: filepath.Dir(path), Kind: RootManual, Harness: HarnessCodex}, path)
	if !ok {
		t.Fatal("identify")
	}
	second := "{\"timestamp\":\"2026-11-20T22:20:52.848Z\",\"ordinal\":1,\"type\":\"event_msg\",\"payload\":{}}\n"
	partial := "{\"timestamp\":\"2026-11-20T22:20:52.885Z\",\"ord"
	appendTo(t, path, second+partial)

	snapshot, err := snapshotTranscript(candidate)
	if err != nil {
		t.Fatalf("a source that grew after discovery must stage: %v", err)
	}
	if string(snapshot.data) != first+second || snapshot.extent != int64(len(first+second+partial)) {
		t.Fatalf("snapshot data=%q extent=%d", snapshot.data, snapshot.extent)
	}
	appendTo(t, path, "dinal\":2}\n")
	if err := snapshot.verify(); err != nil {
		t.Fatalf("growth after staging must verify: %v", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("X"), 3); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := snapshot.verify(); err == nil {
		t.Fatal("a changed prefix was verified")
	}
}

func appendTo(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
}
