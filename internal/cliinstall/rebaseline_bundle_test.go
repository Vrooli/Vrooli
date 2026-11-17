package cliinstall

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
)

func TestRetainRebaselineBundleRecovery(t *testing.T) {
	home := t.TempDir()
	source := filepath.Join(t.TempDir(), "source")
	inputs := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(inputs, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := PrepareBundle(inputs, source); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(source, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("%x", sha256.Sum256(data))
	failure := errors.New("interrupted")
	calls := 0
	_, err = retainRebaselineBundle(home, source, manifest, func(src, dst, expected string, mode os.FileMode) (string, error) {
		calls++
		if calls == 2 {
			return "", failure
		}
		return copyRootCLIImage(src, dst, expected, mode)
	}, config.SyncOwnedDirectoryChain)
	if !errors.Is(err, failure) {
		t.Fatalf("expected interrupted copy, got %v", err)
	}
	dest, err := RebaselineBundlePath(home, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("partial bundle published: %v", err)
	}
	// Simulate a process killed before deferred staging cleanup.
	if err := os.MkdirAll(dest+".staging", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest+".staging", "partial"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := RetainRebaselineBundle(home, source, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got != dest {
		t.Fatalf("path = %s", got)
	}
	if _, err := os.Stat(dest + ".staging"); !os.IsNotExist(err) {
		t.Fatalf("staging remains: %v", err)
	}
	if _, err := RetainRebaselineBundle(home, dest, manifest); err != nil {
		t.Fatalf("resume retained source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, bundleNames[0]), []byte("tampered"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := RetainRebaselineBundle(home, source, manifest); err == nil {
		t.Fatal("accepted corrupt retained bundle")
	}
	if err := RemoveRebaselineBundle(home, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("bundle remains: %v", err)
	}
}

func TestRetainRebaselineBundleReturnsVisiblePathWhenParentSyncFails(t *testing.T) {
	home := t.TempDir()
	source := filepath.Join(t.TempDir(), "source")
	inputs := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(inputs, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := PrepareBundle(inputs, source); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(source, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("%x", sha256.Sum256(data))
	wantErr := errors.New("parent sync uncertain")
	syncCalls := 0
	got, err := retainRebaselineBundle(home, source, manifest, copyRootCLIImage, func(string) error {
		syncCalls++
		if syncCalls == 3 {
			return wantErr
		}
		return nil
	})
	dest, pathErr := RebaselineBundlePath(home, manifest)
	if pathErr != nil {
		t.Fatal(pathErr)
	}
	if !errors.Is(err, wantErr) || got != dest {
		t.Fatalf("retained path = %q, error = %v; want path %q and sync error", got, err, dest)
	}
	if err := VerifyPreparedBundle(got, manifest); err != nil {
		t.Fatalf("visible retained bundle is not recoverable: %v", err)
	}
}

func TestRetainLegacyAndWatchdogBundles(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%t", legacy), func(t *testing.T) {
			source := t.TempDir()
			names := bundleNames
			if legacy {
				names = legacyBundleNames
			}
			var manifestData []byte
			for _, name := range names {
				data := []byte(name)
				if err := os.WriteFile(filepath.Join(source, name), data, 0700); err != nil {
					t.Fatal(err)
				}
				manifestData = append(manifestData, []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name))...)
			}
			if err := os.WriteFile(filepath.Join(source, "SHA256SUMS"), manifestData, 0600); err != nil {
				t.Fatal(err)
			}
			manifest := fmt.Sprintf("%x", sha256.Sum256(manifestData))
			retained, err := RetainRebaselineBundle(t.TempDir(), source, manifest)
			if err != nil {
				t.Fatal(err)
			}
			watchdog, err := PreparedBundleWatchdogSHA(retained, manifest)
			if err != nil || (watchdog == "") != legacy {
				t.Fatalf("watchdog digest %q, error %v", watchdog, err)
			}
			if !legacy {
				if err := os.WriteFile(filepath.Join(retained, "vrooli-watchdog"), []byte("tampered"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := VerifyPreparedBundle(retained, manifest); err == nil {
					t.Fatal("accepted tampered watchdog")
				}
			}
		})
	}
}
