//go:build windows

package shelltest

import (
	"os/exec"
	"testing"
)

func TestStubBinUsesWindowsCommandFile(t *testing.T) {
	path := StubBin(t, "shelltest-windows-stub", 0, "windows fixture")
	if len(path) < 4 || path[len(path)-4:] != ".cmd" {
		t.Fatalf("StubBin path = %q, want .cmd", path)
	}
	output, err := exec.Command("shelltest-windows-stub").CombinedOutput()
	if err != nil || string(output) != "windows fixture" {
		t.Fatalf("stub = %q, %v", output, err)
	}
}
