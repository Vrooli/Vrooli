package shelltest

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestFakeRecordsCallsAndReturnsScriptedResult(t *testing.T) {
	fake := &Fake{
		Paths: map[string]string{"probe": "/tmp/probe"},
		Results: map[string]Result{
			"probe --version": {Output: []byte("1.2\n")},
		},
	}
	if path, err := fake.LookPath("probe"); err != nil || path != "/tmp/probe" {
		t.Fatalf("LookPath() = %q, %v", path, err)
	}
	output, err := fake.Run(context.Background(), "probe", "--version")
	if err != nil || string(output) != "1.2\n" {
		t.Fatalf("Run() = %q, %v", output, err)
	}
	if got := fake.Calls(); !reflect.DeepEqual(got, []string{"probe --version"}) {
		t.Fatalf("Calls() = %#v", got)
	}
	if _, err := fake.Run(context.Background(), "missing"); err == nil {
		t.Fatal("missing scripted command returned nil error")
	}
	if _, err := fake.LookPath("missing"); !errors.Is(err, exec.ErrNotFound) && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing LookPath() error = %v", err)
	}
}

func TestStubBin(t *testing.T) {
	path := StubBin(t, "shelltest-stub", 3, "fixture output")
	if path == "" {
		t.Fatal("StubBin returned empty path")
	}
	output, err := exec.Command("shelltest-stub").CombinedOutput()
	if err == nil || string(output) != "fixture output" {
		t.Fatalf("stub = %q, %v", output, err)
	}
}
