package agenttranscript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256FingerprintsBytesAndExactFilePrefixes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(path, []byte("first-second"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, want := SHA256Bytes([]byte("first")), SHA256BytesPrefix([]byte("first-second"), 5); got != want {
		t.Fatalf("prefix digest = %q, want %q", got, want)
	}
	got, err := SHA256File(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := SHA256Bytes([]byte("first-second")); got != want {
		t.Fatalf("full file digest = %q, want %q", got, want)
	}
	if got, err := SHA256FilePrefixExact(path, 5); err != nil || got != SHA256Bytes([]byte("first")) {
		t.Fatalf("exact prefix digest = %q, err = %v", got, err)
	}
	if _, err := SHA256FilePrefixExact(path, 100); err == nil {
		t.Fatal("short file unexpectedly satisfied exact prefix")
	}
	if _, err := SHA256FilePrefixExact(path, -1); err == nil {
		t.Fatal("negative prefix was accepted")
	}
}
