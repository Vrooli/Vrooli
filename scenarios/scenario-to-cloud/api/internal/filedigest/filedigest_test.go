package filedigest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256File(t *testing.T) {
	const content = "scenario-to-cloud file digest\n"
	path := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, size, err := SHA256File(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "11b7503cb11a96e256eb4fa7d832b3b0b804f2a280f38068c83dfb2d1d66d84b" || size != int64(len(content)) {
		t.Fatalf("SHA256File() = (%q, %d), want known digest and size %d", got, size, len(content))
	}
}

func TestSHA256FileReturnsOpenError(t *testing.T) {
	if _, _, err := SHA256File(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing file did not return an error")
	}
}
