package cliinstall

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrepareBundleDurabilityOrderAndFailures(t *testing.T) {
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	parent := t.TempDir()
	output := filepath.Join(parent, "snapshot")
	var expected []string
	for _, name := range bundleNames {
		expected = append(expected, filepath.Join(output, name))
	}
	expected = append(expected, filepath.Join(output, "SHA256SUMS"), output, parent)
	var calls []string
	syncPath := func(path string) error {
		if _, err := os.Stat(filepath.Join(output, ".lease")); err != nil {
			t.Fatalf("bundle lease missing before first image sync: %v", err)
		}
		calls = append(calls, path)
		return syncPreparedPath(path)
	}
	if err := prepareBundle(source, output, syncPath); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("sync order = %v; want %v", calls, expected)
	}
	if err := os.RemoveAll(output); err != nil {
		t.Fatal(err)
	}
	for i := range expected {
		calls = nil
		injected := errors.New("injected sync failure")
		err := prepareBundle(source, output, func(path string) error {
			calls = append(calls, path)
			if len(calls) == i+1 {
				return injected
			}
			return syncPreparedPath(path)
		})
		if !errors.Is(err, injected) {
			t.Fatalf("sync %d: got %v", i, err)
		}
		if !reflect.DeepEqual(calls, expected[:i+1]) {
			t.Fatalf("sync %d: calls %v", i, calls)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("sync %d left incomplete output: %v", i, err)
		}
	}
}
