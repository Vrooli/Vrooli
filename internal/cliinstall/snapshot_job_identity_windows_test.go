//go:build windows

package cliinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnapshotJobNamesUseLockFileIdentityAcrossDOSPathAliases(t *testing.T) {
	root := t.TempDir()
	snapshot := filepath.Join(root, "install-bundle.a1b2c3d4")
	if err := os.Mkdir(snapshot, 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "snapshots.lock"), filepath.Join(snapshot, ".lease")} {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}

	rootName, err := SnapshotRootJobName(root)
	if err != nil {
		t.Fatal(err)
	}
	leaseName, err := SnapshotLeaseJobName(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}

	rootAlias := windowsExtendedPath(t, root)
	snapshotAlias := filepath.Join(rootAlias, filepath.Base(snapshot))
	rootAliasName, err := SnapshotRootJobName(rootAlias)
	if err != nil {
		t.Fatal(err)
	}
	leaseAliasName, err := SnapshotLeaseJobName(rootAlias, snapshotAlias)
	if err != nil {
		t.Fatal(err)
	}
	if rootAliasName != rootName {
		t.Fatalf("root DOS alias changed witness name: %q != %q", rootAliasName, rootName)
	}
	if leaseAliasName != leaseName {
		t.Fatalf("lease DOS alias changed witness name: %q != %q", leaseAliasName, leaseName)
	}
	leaseNames, err := snapshotLeaseJobNames(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	leaseAliasNames, err := snapshotLeaseJobNames(rootAlias, snapshotAlias)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaseNames) != 2 || len(leaseAliasNames) != 2 || leaseNames[0] != leaseName || leaseAliasNames[0] != leaseName {
		t.Fatalf("file-ID witness differs across DOS aliases: %v versus %v", leaseNames, leaseAliasNames)
	}
}

func TestMarkerlessStaleSnapshotStillPrunesAndLegacyNameCanBlock(t *testing.T) {
	root := t.TempDir()
	snapshot := filepath.Join(root, "install-bundle.a1b2c3d4")
	if err := os.Mkdir(snapshot, 0700); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(snapshot, old, old); err != nil {
		t.Fatal(err)
	}

	names, err := snapshotLeaseJobNames(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 {
		t.Fatalf("markerless snapshot witness names=%v; want legacy path name only", names)
	}
	legacyName := names[0]
	state, err := inspectSnapshotJobNamesUsing(names, func(name string) (snapshotJobState, error) {
		if name != legacyName {
			t.Fatalf("probed %q, want legacy name %q", name, legacyName)
		}
		return snapshotJobActive, nil
	})
	if err != nil || state != snapshotJobActive {
		t.Fatalf("markerless legacy state=%v error=%v; want active", state, err)
	}

	if err := PruneInstallSnapshots(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(snapshot); !os.IsNotExist(err) {
		t.Fatalf("stale markerless snapshot remains after absent legacy witness: %v", err)
	}
}

func windowsExtendedPath(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(absolute, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(absolute, `\\`)
	}
	return `\\?\` + absolute
}
