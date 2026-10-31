//go:build !windows

package agentinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// npmPrefix lays out what `npm install --prefix <root>` leaves behind: the
// executable under node_modules/.bin and nothing in <root>/bin.
func npmPrefix(t *testing.T, binary string) (prefix, binDir, source string) {
	t.Helper()
	prefix = t.TempDir()
	binDir = filepath.Join(prefix, "bin")
	if err := os.MkdirAll(filepath.Join(prefix, "node_modules", ".bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source = filepath.Join(prefix, "node_modules", ".bin", binary)
	if err := os.WriteFile(source, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return prefix, binDir, source
}

// TestNPMInstallLinksTheExecutableWhereTheResourceLooks pins the false
// success: npm put opencode under node_modules/.bin, the resource and the
// node's capability probe looked in bin, and the install still reported
// success.
func TestNPMInstallLinksTheExecutableWhereTheResourceLooks(t *testing.T) {
	prefix, binDir, source := npmPrefix(t, "opencode")
	if err := linkNPMBinary(prefix, "opencode", binDir); err != nil {
		t.Fatalf("linkNPMBinary() error = %v", err)
	}
	got, err := os.Readlink(filepath.Join(binDir, "opencode"))
	if err != nil || got != source {
		t.Fatalf("bin/opencode -> %q (%v), want %q", got, err, source)
	}
	if err := linkNPMBinary(prefix, "opencode", binDir); err != nil {
		t.Fatalf("re-running the link must be a no-op, got %v", err)
	}
}

func TestNPMInstallReplacesAStaleLink(t *testing.T) {
	prefix, binDir, source := npmPrefix(t, "opencode")
	if err := os.Symlink(filepath.Join(prefix, "old", "opencode"), filepath.Join(binDir, "opencode")); err != nil {
		t.Fatal(err)
	}
	if err := linkNPMBinary(prefix, "opencode", binDir); err != nil {
		t.Fatalf("linkNPMBinary() error = %v", err)
	}
	if got, _ := os.Readlink(filepath.Join(binDir, "opencode")); got != source {
		t.Fatalf("bin/opencode -> %q, want %q", got, source)
	}
}

func TestNPMInstallRefusesToReplaceSomeoneElsesFile(t *testing.T) {
	prefix, binDir, _ := npmPrefix(t, "opencode")
	owned := filepath.Join(binDir, "opencode")
	if err := os.WriteFile(owned, []byte("operator's own build"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := linkNPMBinary(prefix, "opencode", binDir)
	if err == nil || !strings.Contains(err.Error(), "refusing to replace") {
		t.Fatalf("linkNPMBinary() error = %v, want a refusal", err)
	}
	if data, _ := os.ReadFile(owned); string(data) != "operator's own build" {
		t.Fatal("an existing file was overwritten")
	}
}

func TestNPMInstallThatProducedNoExecutableFails(t *testing.T) {
	prefix := t.TempDir()
	binDir := filepath.Join(prefix, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := linkNPMBinary(prefix, "opencode", binDir)
	if err == nil || !strings.Contains(err.Error(), "produced no opencode executable") {
		t.Fatalf("linkNPMBinary() error = %v, want a failure naming the missing executable", err)
	}
}
