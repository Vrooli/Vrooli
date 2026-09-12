package capabilities

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMacOSQualificationNamesEveryUnmetRequirement(t *testing.T) {
	unmet := MacOSQualification(MacOSCandidate{})
	for _, want := range []string{"executable", "signed", "checksum", "smoke"} {
		found := false
		for _, item := range unmet {
			if contains(item, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unmet requirements %v do not name %q", unmet, want)
		}
	}
}

func TestMacOSQualificationAcceptsMatchingEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "speech-server")
	if err := os.WriteFile(path, []byte("native server"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := fileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	if unmet := MacOSQualification(MacOSCandidate{Path: path, Signed: true, SHA256: digest, SmokePassed: true}); len(unmet) != 0 {
		t.Fatalf("qualification = %v, want no unmet requirements", unmet)
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
