package cliinstall

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrepareHubBundlePublishesInRecoveryOrder(t *testing.T) {
	injected := errors.New("injected failure")
	snapshot := filepath.Join(t.TempDir(), "snapshot")
	pointer := filepath.Join(filepath.Dir(snapshot), "current")
	want := []string{"prepare", filepath.Join(snapshot, ".lease"), filepath.Join(snapshot, ".ready"), pointer}
	for failure := 0; failure <= len(want); failure++ {
		var calls []string
		call := func(name string) error {
			calls = append(calls, name)
			if len(calls) == failure {
				return injected
			}
			return nil
		}
		err := prepareHubBundle("source", snapshot, pointer, func(_, _ string) error { return call("prepare") }, func(path string, data []byte, mode os.FileMode) error {
			if path == pointer && string(data) != snapshot+"\n" {
				t.Fatalf("locator = %q", data)
			}
			if mode != 0600 {
				t.Fatalf("mode = %v", mode)
			}
			return call(path)
		})
		count := len(want)
		if failure > 0 {
			count = failure
			if !errors.Is(err, injected) {
				t.Fatalf("failure %d: %v", failure, err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(calls, want[:count]) {
			t.Fatalf("failure %d: calls %v", failure, calls)
		}
	}
}

func TestPrepareHubBundleCreatesResumableSnapshot(t *testing.T) {
	source, parent := t.TempDir(), t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, pointer := filepath.Join(parent, "snapshot"), filepath.Join(parent, "current")
	if err := PrepareHubBundle(source, snapshot, pointer); err != nil {
		t.Fatal(err)
	}
	locator, err := os.ReadFile(pointer)
	if err != nil || string(locator) != snapshot+"\n" {
		t.Fatalf("locator %q: %v", locator, err)
	}
	if _, err := os.Stat(filepath.Join(snapshot, ".ready")); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(snapshot, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(manifest))
	if err := VerifyPreparedBundle(snapshot, digest); err != nil {
		t.Fatal(err)
	}
}
