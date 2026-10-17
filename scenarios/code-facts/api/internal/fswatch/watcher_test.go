package fswatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherSignalsNestedWrite(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "packages", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := newWithInterval([]string{filepath.Join(root, "packages")}, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte("package demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not signal nested write")
	}
}

func TestIgnoredRejectsStateAndDependencyPaths(t *testing.T) {
	ignored := []string{
		"scenarios/x/data/y.db-wal",
		"scenarios/x/data/y.db",
		"scenarios/x/data/y.db-shm",
		"scenarios/x/data",
		"scenarios/x/data/nested/state.json",
		"resources/twilio/data/node_modules/twilio-cli/src/index.js",
		"scenarios/x/ui/node_modules/react/index.js",
		"scenarios/x/ui/node_modules",
		"scenarios/x/ui/.pnpm/lock",
		"scenarios/x/ui/dist/bundle.js",
		"scenarios/x/ui/build/index.html",
		"scenarios/x/api/coverage/cover.out",
		"scenarios/x/tmp/scratch.go",
		"scenarios/x/.git/index",
		"scenarios/x/api/.mypy_cache/3.11/cache.0.db",
		"scenarios/web-console/web-console.db",
		"scenarios/x/api/service.log",
		"scenarios/x/api/state.sqlite",
		"scenarios/x/api/state.sqlite3-wal",
		"packages/proto/.verify-123/output.go",
	}
	for _, path := range ignored {
		if !Ignored(path) {
			t.Errorf("Ignored(%q) = false, want true", path)
		}
	}
	accepted := []string{
		"scenarios/x/api/main.go",
		"scenarios/x/api",
		"scenarios/x/ui/src/App.tsx",
		"scenarios/x/api/internal/data/loader.go",
		"packages/proto/schemas/facts.proto",
		"cmd/vrooli/main.go",
		"internal/fswatch/watcher_linux.go",
		"resources/postgres/cli/postgres.sh",
		"scenarios/x/api/database.go",
		"scenarios/x/api/handlers/logger.go",
	}
	for _, path := range accepted {
		if Ignored(path) {
			t.Errorf("Ignored(%q) = true, want false", path)
		}
	}
}

func TestWatcherDoesNotSignalIgnoredPaths(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	dataDir := filepath.Join(scenarios, "demo", "data")
	modules := filepath.Join(scenarios, "demo", "ui", "node_modules", "pkg")
	source := filepath.Join(scenarios, "demo", "api")
	for _, dir := range []string{dataDir, modules, source} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	w, err := newWithInterval([]string{scenarios}, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(dataDir, "state.db-wal"), []byte("wal"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modules, "index.js"), []byte("module.exports = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "trace.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.Events():
		t.Fatal("watcher signalled for ignored data/node_modules/log writes")
	case <-time.After(400 * time.Millisecond):
	}
	if err := os.WriteFile(filepath.Join(source, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not signal source write after ignored writes")
	}
}

func TestWatcherWatchesDirectoriesCreatedLater(t *testing.T) {
	root := t.TempDir()
	packages := filepath.Join(root, "packages")
	if err := os.MkdirAll(packages, 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := newWithInterval([]string{packages}, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	created := filepath.Join(packages, "later")
	if err := os.Mkdir(created, 0o755); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not signal directory creation")
	}
	// Give the loop time to install the watch on the new directory before
	// writing into it; the event for the mkdir may still be in flight.
	time.Sleep(200 * time.Millisecond)
	drain(w)
	if err := os.WriteFile(filepath.Join(created, "source.go"), []byte("package later"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not signal write inside a directory created after start")
	}
}

func drain(w *Watcher) {
	for {
		select {
		case <-w.Events():
		default:
			return
		}
	}
}
