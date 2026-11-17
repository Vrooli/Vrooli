package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliinstall"
)

func TestInstallProjectBundleOwnsSnapshotLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name       string
		installErr error
	}{
		{name: "success"},
		{name: "installer failure", installErr: errors.New("activation refused")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, source := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("SUDO_USER", "")
			for _, name := range []string{"vrooli", "vrooli-atomic-install", "vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog", "vrooli-watchdog"} {
				if err := os.WriteFile(filepath.Join(source, name), []byte("prepared image: "+name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			bin, pointer, err := ownedRuntimePaths()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			calls := 0
			var preparedSnapshot string
			err = installProjectBundle(&stderr, source, bin, pointer, "/source", func(lock *os.File, root, bundle, gotBin, manifest, sourceRoot string) error {
				calls++
				preparedSnapshot = bundle
				if lock == nil || root == "" || gotBin != bin || sourceRoot != "/source" || len(manifest) != 64 {
					t.Fatalf("install arguments = lock %t root %q bin %q manifest %q source %q", lock != nil, root, gotBin, manifest, sourceRoot)
				}
				if err := cliinstall.VerifyPreparedBundle(bundle, manifest); err != nil {
					t.Fatalf("bundle was not verified before install: %v", err)
				}
				if _, err := os.Stat(filepath.Join(bundle, ".lease")); err != nil {
					t.Fatalf("snapshot lease was not present during install: %v", err)
				}
				return tc.installErr
			})
			if !errors.Is(err, tc.installErr) {
				t.Fatalf("install error = %v, want %v", err, tc.installErr)
			}
			if calls != 1 {
				t.Fatalf("installer calls = %d, want 1", calls)
			}
			if _, err := os.Stat(preparedSnapshot); !os.IsNotExist(err) {
				t.Fatalf("prepared snapshot remains after lease release: stat error = %v", err)
			}
			if strings.Contains(stderr.String(), "cleanup failed") {
				t.Fatalf("cleanup warning: %s", stderr.String())
			}
		})
	}
}

func TestInstallProjectBundleRejectsRuntimeDestinationOverridesBeforePreparation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	calls := 0
	err := installProjectBundle(&bytes.Buffer{}, t.TempDir(), filepath.Join(t.TempDir(), "wrong-bin"), "", "/source", func(*os.File, string, string, string, string, string) error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "destination overrides") || calls != 0 {
		t.Fatalf("invalid destination override: err=%v install calls=%d", err, calls)
	}
}
