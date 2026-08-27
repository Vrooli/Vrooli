package resources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestPythonWheelsCommandCarriesPinnedIndexAndHashMode(t *testing.T) {
	lockfile := filepath.Join(t.TempDir(), "requirements.lock")
	if err := os.WriteFile(lockfile, []byte("demo==1.0 --hash=sha256:"+strings.Repeat("a", 64)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command, err := pythonWheelsCommand(context.Background(), binaryfetch.ComposeStep{
		IndexURL: "https://wheels.example.test/simple", ExtraIndexURLs: []string{"https://mirror.example.test/simple"},
	}, "/artifact/lib", lockfile, "x86_64-unknown-linux-gnu")
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(command.Args, " ")
	for _, expected := range []string{"--no-deps", "--only-binary :all:", "--index-url https://wheels.example.test/simple", "--extra-index-url https://mirror.example.test/simple", "--require-hashes"} {
		if !strings.Contains(args, expected) {
			t.Errorf("command %q does not contain %q", args, expected)
		}
	}
}

func TestPythonWheelsCommandAllowsLockedSourceDistributionsWhenDeclared(t *testing.T) {
	lockfile := filepath.Join(t.TempDir(), "requirements.lock")
	if err := os.WriteFile(lockfile, []byte("demo==1.0 --hash=sha256:"+strings.Repeat("a", 64)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command, err := pythonWheelsCommand(context.Background(), binaryfetch.ComposeStep{AllowSDists: true}, "/artifact/lib", lockfile, "x86_64-unknown-linux-gnu")
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(command.Args, " ")
	if strings.Contains(args, "--only-binary :all:") {
		t.Fatalf("source distributions were not allowed in command %q", args)
	}
	if !strings.Contains(args, "--no-deps") {
		t.Fatalf("command %q must install only the complete locked closure", args)
	}
}

func TestPythonWheelsCommandRejectsUnhashedThirdPartyIndex(t *testing.T) {
	lockfile := filepath.Join(t.TempDir(), "requirements.lock")
	if err := os.WriteFile(lockfile, []byte("demo==1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := pythonWheelsCommand(context.Background(), binaryfetch.ComposeStep{IndexURL: "https://wheels.example.test/simple"}, "/artifact/lib", lockfile, "x86_64-unknown-linux-gnu")
	if err == nil || !strings.Contains(err.Error(), "hash-pinned") {
		t.Fatalf("unhashed indexed lockfile error = %v", err)
	}
}
