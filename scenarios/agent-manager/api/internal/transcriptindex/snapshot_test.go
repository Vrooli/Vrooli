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
	err = verifySnapshotWithHook(path, info, data, func() {
		if err := os.Rename(other, path); err != nil {
			t.Fatal(err)
		}
	})
	if err == nil {
		t.Fatal("path replacement during verification was accepted")
	}
}
