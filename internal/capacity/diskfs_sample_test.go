package capacity

import (
	"path/filepath"
	"testing"
)

func TestDiskJobSampleIdentityPreservesAdmissionFilesystem(t *testing.T) {
	root := t.TempDir()
	// Newly declared output paths must retain the physical filesystem of the
	// existing ancestor, without requiring a directory creation or disk write.
	for _, path := range []string{root, filepath.Join(root, "not-created", "output")} {
		_, want, _, _, err := diskPathIdentity(path)
		if err != nil {
			t.Skipf("native disk admission unavailable: %v", err)
		}
		got, probe, err := diskJobFilesystemIdentity(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("sample identity %q differs from admitted physical filesystem %q", got, want)
		}
		_, observed, _, err := diskFilesystem(probe)
		if err != nil || observed != want {
			t.Fatalf("sample probe %q does not identify admitted filesystem %q: got %q, error %v", probe, want, observed, err)
		}
	}
}
