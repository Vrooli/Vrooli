package agentinstall

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstallUsesPerCallPlatformDependencies(t *testing.T) {
	var created []string
	var command string
	deps := defaultPlatformDeps()
	deps.mkdirAll = func(path string, _ fs.FileMode) error {
		created = append(created, path)
		return nil
	}
	deps.commandOutput = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		command = name
		return nil, errors.New("injected command result")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	err = install(context.Background(), Spec{
		Binary:       "agentinstall-test-missing-binary",
		BinDir:       filepath.Join(home, ".local", "bin"),
		DataDir:      t.TempDir(),
		NPM:          "example/package",
		ResourceName: "example-resource",
	}, deps)
	if err == nil {
		t.Fatal("install unexpectedly succeeded")
	}
	if len(created) != 2 {
		t.Fatalf("mkdir calls = %v, want install and data directories", created)
	}
	if command != "scenario-dependency-analyzer" {
		t.Fatalf("command = %q", command)
	}
}

func TestInstalledVersionUsesInjectedCommand(t *testing.T) {
	called := false
	deps := defaultPlatformDeps()
	deps.commandCombined = func(_ context.Context, name string, args ...string) ([]byte, error) {
		called = name == "/fake/agent" && strings.Join(args, " ") == "--version"
		return []byte("agent 9.8.7\n"), nil
	}
	got, err := installedVersion(context.Background(), "/fake/agent", deps)
	if err != nil || !called || got != "agent 9.8.7\n" {
		t.Fatalf("installedVersion() = %q, %v, injected command called=%v", got, err, called)
	}
}

func TestDownloadThreadsInjectedPlatformDependencies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("payload")) }))
	defer server.Close()
	var written []byte
	deps := platformDeps{downloadTimeout: time.Second, writeOwnedFile: func(_ string, data []byte, _ fs.FileMode) error {
		written = append([]byte(nil), data...)
		return nil
	}}
	if err := downloadWithDeps(context.Background(), server.URL+"/agent", filepath.Join(t.TempDir(), "agent"), "", deps); err != nil {
		t.Fatal(err)
	}
	if string(written) != "payload" {
		t.Fatalf("injected writer received %q", written)
	}
}

func TestReleasePlatformUsesInjectedOSAndArchitecture(t *testing.T) {
	got := releasePlatformWith(platformDeps{goos: "plan9", goarch: "riscv64"})
	if got != "plan9_riscv64" {
		t.Fatalf("releasePlatformWith() = %q", got)
	}
}

func TestPublishLinkUsesInjectedOwnershipCommit(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, []byte("x"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "target")
	called := false
	deps := platformDeps{commitOwnedPath: func(staged, destination string) error {
		called = true
		if staged == "" || destination != target {
			t.Fatalf("commit paths = %q, %q", staged, destination)
		}
		return os.Rename(staged, destination)
	}}
	if err := publishLinkWithDeps(source, target, deps); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(target)
	if err != nil || !called || link != source {
		t.Fatalf("link=%q, err=%v, commit called=%v", link, err, called)
	}
}

func TestDefaultPlatformDependenciesAreComplete(t *testing.T) {
	deps := defaultPlatformDeps().withDefaults()
	if deps.loadManifest == nil || deps.resourceDataDir == nil || deps.commandOutput == nil || deps.commandCombined == nil || deps.homeDir == nil || deps.mkdirAll == nil || deps.commitOwnedPath == nil || deps.writeOwnedFile == nil || deps.ensureOwnedDir == nil || deps.acquireLock == nil {
		t.Fatal("default platform dependencies are incomplete")
	}
	if deps.permDir == 0 || deps.permPrivate == 0 || deps.permOwnerWrite == 0 || deps.controlTimeout <= 0 || deps.downloadTimeout <= 0 {
		t.Fatalf("default platform settings are incomplete: %+v", deps)
	}
}
