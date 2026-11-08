package emergencywatchdog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatchdogStampIncludesSharedModulesAndEmbeddedSafeguardPolicy(t *testing.T) {
	root := t.TempDir()
	write := func(name, data string) {
		t.Helper()
		file := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("cmd/vrooli-watchdog/main.go", "package main\n")
	write("internal/safeguards/policy.go", "package safeguards\nimport _ \"embed\"\n//go:embed safeguard.json\nvar policy string\n")
	write("internal/safeguards/safeguard.json", "initial policy")
	previous, err := expectedWatchdogVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct{ file, data string }{
		{"go.mod", "module example.test/root\n"},
		{"go.sum", "dependency checksum\n"},
		{"packages/cli-core/cli.go", "package cli\n"},
		{"packages/proto/gen/go/common.go", "package common\n"},
		{"internal/safeguards/safeguard.json", "updated policy"},
	} {
		write(change.file, change.data)
		next, err := expectedWatchdogVersion(root)
		if err != nil || next == previous {
			t.Fatalf("watchdog ignored %s: %v", change.file, err)
		}
		previous = next
	}
}
