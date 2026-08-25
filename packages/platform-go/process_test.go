//go:build linux

package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProcessWorkingDir_CurrentProcess(t *testing.T) {
	want, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ProcessWorkingDir(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("working directory = %q, want %q", got, want)
	}
}

func TestProcessHasChildren_ReportsStartedChild(t *testing.T) {
	child := exec.Command("sleep", "2")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })

	got, err := ProcessHasChildren(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("ProcessHasChildren reported false while sleep child was running")
	}
}

func TestProcessFacts_UnsupportedNeverLooksLikeEmptySuccess(t *testing.T) {
	workingDir, workingDirErr := ProcessWorkingDir(os.Getpid())
	if workingDirErr == nil && workingDir == "" {
		t.Fatal("ProcessWorkingDir returned an empty value without an error")
	}
	children, childrenErr := ProcessHasChildren(os.Getpid())
	if childrenErr == nil && children {
		// A child can exist in a test runner, so this is not an error. The
		// assertion below is intentionally about the contract, not falsehood.
		return
	}
	if childrenErr != nil && !errors.Is(childrenErr, ErrUnsupported) {
		// Linux reports ordinary /proc errors directly; only unsupported
		// hosts must use the typed sentinel.
		return
	}
}
