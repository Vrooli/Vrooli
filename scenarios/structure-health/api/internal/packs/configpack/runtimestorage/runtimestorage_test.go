package runtimestorage

import "testing"

func TestCheckRepoLocalRuntimeStorageFlagsRepoLocalPatterns(t *testing.T) {
	violations := CheckRepoLocalRuntimeStorage([]byte(`package main
import "os"
func save() error {
	return os.WriteFile("../data/tasks/task.json", []byte("x"), 0o644)
}`), "api/main.go", "example")
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if got := violations[0].Message; got == "" {
		t.Fatal("expected violation message")
	}
}

func TestCheckRepoLocalRuntimeStorageFlagsShellRepoRootPatterns(t *testing.T) {
	violations := CheckRepoLocalRuntimeStorage([]byte(`#!/usr/bin/env bash
mkdir -p "${APP_ROOT}/data/session-profiles"`), "cli/start.sh", "example")
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

func TestCheckRepoLocalRuntimeStorageAllowsStoragePackageUsage(t *testing.T) {
	violations := CheckRepoLocalRuntimeStorage([]byte(`package main
import "github.com/vrooli/api-core/storage"
func save(store *storage.Storage) (string, error) {
	return store.Path(storage.ClassData, "tasks/task.json")
}`), "api/main.go", "example")
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestCheckRepoLocalRuntimeStorageFlagsLegacyHomeScopedStorage(t *testing.T) {
	violations := CheckRepoLocalRuntimeStorage([]byte(`#!/usr/bin/env bash
mkdir -p "${HOME}/.qdrant/data"
cp output.json "$HOME/.minio/cache/result.json"
echo ~/.whisper/uploads`), "cli/cache.sh", "example")
	if len(violations) != 3 {
		t.Fatalf("expected 3 violations, got %d", len(violations))
	}
}
