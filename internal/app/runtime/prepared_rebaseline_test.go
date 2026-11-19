package runtimeapp

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestPreparedRebaselineStatePathsUseSharedHomeInterlock(t *testing.T) {
	home := t.TempDir()
	stateDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := preparedRebaselineStatePathsForHome(home)
	if err != nil {
		t.Fatal(err)
	}
	transitions := filepath.Join(stateDir, "root-rebaseline")
	want := preparedRebaselineStatePaths{
		transitions: transitions,
		pause:       filepath.Join(transitions, "pause.json"),
		progress:    filepath.Join(transitions, "progress.json"),
		lock:        filepath.Join(transitions, "transition.lock"),
	}
	if paths != want {
		t.Fatalf("prepared rebaseline state paths = %+v, want %+v", paths, want)
	}
}

func TestPreparePreparedRebaselineRecoveryPreservesRetainedPathOnFailure(t *testing.T) {
	wantErr := errors.New("injected recovery failure")
	for _, tc := range []struct {
		name            string
		retainedPath    string
		retainErr       error
		missingWatchdog bool
		badProgress     bool
		wantPath        bool
		wantProgress    bool
	}{
		{name: "retain failed before path known", retainErr: wantErr},
		{name: "retain path survives sync uncertainty", retainedPath: "retained-bundle", retainErr: wantErr, wantPath: true},
		{name: "watchdog retain failed after retention", retainedPath: "retained-bundle", missingWatchdog: true, wantPath: true},
		{name: "progress write failed after retention", retainedPath: "retained-bundle", badProgress: true, wantPath: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stagedWatchdog := filepath.Join(dir, "watchdog")
			watchdogSHA := "qualified-digest"
			if !tc.missingWatchdog {
				data := []byte("qualified watchdog")
				if err := os.WriteFile(stagedWatchdog, data, 0700); err != nil {
					t.Fatal(err)
				}
				watchdogSHA = fmtHash(data)
			}
			progressPath := filepath.Join(dir, "progress.json")
			if tc.badProgress {
				parentFile := filepath.Join(dir, "not-a-directory")
				if err := os.WriteFile(parentFile, []byte("file"), 0600); err != nil {
					t.Fatal(err)
				}
				progressPath = filepath.Join(parentFile, "progress.json")
			}
			e := rebaselineEffects{
				retainBundle: func(string, string, string) (string, error) {
					return tc.retainedPath, tc.retainErr
				},
			}
			request := RebaselineRequest{BundleSHA: "bundle-sha"}
			prepared, err := preparePreparedRebaselineRecovery(context.Background(), e, request, dir,
				RebaselineAdmission{CandidateSHA: "candidate", WatchdogSHA: watchdogSHA}, filepath.Join(dir, "candidate"),
				request.BundleSHA, stagedWatchdog, filepath.Join(dir, "watchdog-pending"), progressPath, watchdogSHA,
				newPreparedRebaselineProgress("candidate", watchdogSHA, request.BundleSHA, "source"))
			if err == nil {
				t.Fatal("injected failure unexpectedly succeeded")
			}
			if tc.wantPath && prepared.BundleDir != tc.retainedPath {
				t.Fatalf("retained recovery path = %q, want %q", prepared.BundleDir, tc.retainedPath)
			}
			if !tc.wantPath && prepared.BundleDir != "" {
				t.Fatalf("claimed unknown retained recovery path %q after pre-retention failure", prepared.BundleDir)
			}
			if prepared.RecoveryProgressWritten != tc.wantProgress {
				t.Fatalf("recovery progress written = %t, want %t", prepared.RecoveryProgressWritten, tc.wantProgress)
			}
		})
	}
}

func TestPreparePreparedRebaselineRecoveryMarksProgressAfterDurableWrite(t *testing.T) {
	dir := t.TempDir()
	watchdog := filepath.Join(dir, "watchdog")
	data := []byte("qualified watchdog")
	if err := os.WriteFile(watchdog, data, 0700); err != nil {
		t.Fatal(err)
	}
	const bundleSHA = "bundle-sha"
	request := RebaselineRequest{BundleSHA: bundleSHA}
	prepared, err := preparePreparedRebaselineRecovery(context.Background(), rebaselineEffects{
		retainBundle: func(string, string, string) (string, error) { return "retained", nil },
	}, request, dir, RebaselineAdmission{}, filepath.Join(dir, "candidate"), bundleSHA,
		watchdog, filepath.Join(dir, "pending-watchdog"), filepath.Join(dir, "progress.json"), fmtHash(data),
		newPreparedRebaselineProgress("candidate", fmtHash(data), bundleSHA, "source"))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BundleDir != "retained" || !prepared.RecoveryProgressWritten {
		t.Fatalf("successful recovery preparation = %+v", prepared)
	}
}

func fmtHash(data []byte) string {
	digest := sha256.Sum256(data)
	return fmt.Sprintf("%x", digest)
}
