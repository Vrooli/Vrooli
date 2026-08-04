//go:build linux

package securestore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeSecretTool(t *testing.T, lookupExit int) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = lookup ]; then\n"
	if lookupExit == 0 {
		script += "  printf 'healthy-value\\n'\n  exit 0\n"
	} else {
		script += "  exit 1\n"
	}
	script += "fi\nexit 0\n"
	path := filepath.Join(dir, "secret-tool")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func withCollectionRunner(t *testing.T, runner func(string) (string, string, error)) {
	t.Helper()
	previous := runSecretServiceCommand
	runSecretServiceCommand = func(_ context.Context, args ...string) (string, string, error) {
		return runner(strings.Join(args, " "))
	}
	t.Cleanup(func() { runSecretServiceCommand = previous })
}

func TestSecretToolPartialCollectionLoadIsUnavailable(t *testing.T) {
	fakeSecretTool(t, 1)
	withCollectionRunner(t, func(command string) (string, string, error) {
		if strings.Contains(command, "Collections") {
			return "([objectpath '/org/freedesktop/secrets/collection/login'],)", "", nil
		}
		return "", "Error: object does not exist", errors.New("exit status 1")
	})
	_, err := (&secretToolStore{}).Get("vrooli.credentials.v1", "missing")
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "/org/freedesktop/secrets/collection/login") {
		t.Fatalf("Get() = %v, want unavailable with the collection path", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("partial collection load was reported as not found: %v", err)
	}
}

func TestSecretToolHealthyEmptyCollectionIsNotFound(t *testing.T) {
	fakeSecretTool(t, 1)
	withCollectionRunner(t, func(command string) (string, string, error) {
		if strings.Contains(command, "Collections") {
			return "([objectpath '/org/freedesktop/secrets/collection/login'],)", "", nil
		}
		return "'login'", "", nil
	})
	_, err := (&secretToolStore{}).Get("vrooli.credentials.v1", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() = %v, want not found for a healthy empty service", err)
	}
}

func TestSecretToolCollectionHealthRunsOnce(t *testing.T) {
	count := 0
	withCollectionRunner(t, func(command string) (string, string, error) {
		count++
		if strings.Contains(command, "Collections") {
			return "([objectpath '/org/freedesktop/secrets/collection/login'],)", "", nil
		}
		return "'login'", "", nil
	})
	store := &secretToolStore{}
	for range 6 {
		if err := store.collectionHealth(); err != nil {
			t.Fatal(err)
		}
	}
	if count != 2 {
		t.Fatalf("collection health calls = %d, want one service call and one collection call", count)
	}
}

func TestSecretToolSuccessfulGetDoesNotProbeCollections(t *testing.T) {
	fakeSecretTool(t, 0)
	called := false
	withCollectionRunner(t, func(string) (string, string, error) {
		called = true
		return "", "", errors.New("collection health should not run after a successful read")
	})
	value, err := (&secretToolStore{}).Get("vrooli.credentials.v1", "present")
	if err != nil || value != "healthy-value" {
		t.Fatalf("Get() = %q, %v", value, err)
	}
	if called {
		t.Fatal("successful Get ran the collection health probe")
	}
}
