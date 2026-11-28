package platform

import (
	"errors"
	"os"
	"testing"
)

func TestProcessWorkingDirectoryIdentityIgnoresSpellingAndRejectsAnotherDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	same, err := ProcessWorkingDirectoryMatches(os.Getpid(), cwd+string(os.PathSeparator))
	if errors.Is(err, ErrUnsupported) {
		t.Skip("native directory identity unavailable")
	}
	if err != nil || !same {
		t.Fatalf("self directory identity match=%v err=%v", same, err)
	}
	other, err := ProcessWorkingDirectoryMatches(os.Getpid(), t.TempDir())
	if err != nil || other {
		t.Fatalf("foreign directory match=%v err=%v", other, err)
	}
}
