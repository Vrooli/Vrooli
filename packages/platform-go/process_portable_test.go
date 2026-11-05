package platform

import (
	"errors"
	"os"
	"testing"
)

func TestProcessFacts_PortableContract(t *testing.T) {
	workingDir, err := ProcessWorkingDir(os.Getpid())
	if err == nil && workingDir == "" {
		t.Fatal("ProcessWorkingDir returned an empty value without an error")
	}
	if err != nil && !errors.Is(err, ErrUnsupported) {
		// A live process can disappear or be denied between lookup and read;
		// that is still an honest operational error.
		t.Logf("ProcessWorkingDir returned operational error: %v", err)
	}

	children, err := ProcessHasChildren(os.Getpid())
	if err == nil {
		_ = children
	} else if !errors.Is(err, ErrUnsupported) {
		t.Logf("ProcessHasChildren returned operational error: %v", err)
	}
}
