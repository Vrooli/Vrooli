package cliinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func TestSnapshotLeaseAdmissionPolicy(t *testing.T) {
	probeFailure := errors.New("job query denied")
	tests := []struct {
		name        string
		leaseLocked bool
		state       snapshotJobState
		probeErr    error
		want        bool
		wantErr     bool
	}{
		{name: "lease unavailable", leaseLocked: false, state: snapshotJobActive, want: false},
		{name: "no job", leaseLocked: true, state: snapshotJobAbsent, want: true},
		{name: "idle job", leaseLocked: true, state: snapshotJobIdle, want: true},
		{name: "active job", leaseLocked: true, state: snapshotJobActive, want: false},
		{name: "query failure", leaseLocked: true, state: snapshotJobAbsent, probeErr: probeFailure, wantErr: true},
		{name: "unknown state", leaseLocked: true, state: snapshotJobState(99), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := snapshotLeaseAdmission(tt.leaseLocked, tt.state, tt.probeErr)
			if got != tt.want {
				t.Fatalf("admitted=%v; want %v", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v; wantErr=%v", err, tt.wantErr)
			}
			if tt.probeErr != nil && !errors.Is(err, tt.probeErr) {
				t.Fatalf("error %v does not preserve probe failure %v", err, tt.probeErr)
			}
		})
	}
}

func TestSnapshotLeasePrunePolicy(t *testing.T) {
	probeFailure := errors.New("job query denied")
	tests := []struct {
		name     string
		state    snapshotJobState
		probeErr error
		want     bool
		wantErr  bool
	}{
		{name: "job absent", state: snapshotJobAbsent, want: true},
		{name: "job drained", state: snapshotJobIdle, want: true},
		{name: "active job", state: snapshotJobActive, want: false},
		{name: "query failure", state: snapshotJobAbsent, probeErr: probeFailure, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := snapshotLeasePruneAllowed(tt.state, tt.probeErr)
			if got != tt.want || (err != nil) != tt.wantErr {
				t.Fatalf("allowed=%v error=%v; want allowed=%v wantErr=%v", got, err, tt.want, tt.wantErr)
			}
			if tt.probeErr != nil && !errors.Is(err, tt.probeErr) {
				t.Fatalf("error %v does not preserve probe failure %v", err, tt.probeErr)
			}
		})
	}
}

func TestSnapshotJobWitnessProbeChecksLegacyNameAndFailsClosed(t *testing.T) {
	names := []string{"file-id-name", "legacy-path-name"}
	probed := make([]string, 0, len(names))
	state, err := inspectSnapshotJobNamesUsing(names, func(name string) (snapshotJobState, error) {
		probed = append(probed, name)
		if name == "legacy-path-name" {
			return snapshotJobActive, nil
		}
		return snapshotJobAbsent, nil
	})
	if err != nil || state != snapshotJobActive {
		t.Fatalf("state=%v error=%v; want active legacy witness", state, err)
	}
	if len(probed) != 2 || probed[0] != names[0] || probed[1] != names[1] {
		t.Fatalf("probed names=%v; want %v", probed, names)
	}

	probeFailure := errors.New("legacy query denied")
	state, err = inspectSnapshotJobNamesUsing(names, func(name string) (snapshotJobState, error) {
		if name == "legacy-path-name" {
			return snapshotJobAbsent, probeFailure
		}
		return snapshotJobActive, nil
	})
	if state != snapshotJobAbsent || !errors.Is(err, probeFailure) {
		t.Fatalf("state=%v error=%v; want fail-closed query error", state, err)
	}
}

func TestSnapshotRootLockWaitsOutsideLockWhileRootJobIsActive(t *testing.T) {
	root := t.TempDir()
	probes, sleeps := 0, 0
	file, release, err := acquireSnapshotRootLockUsing(root, func(string) (snapshotJobState, error) {
		probes++
		if probes == 1 {
			return snapshotJobActive, nil
		}
		return snapshotJobAbsent, nil
	}, func(delay time.Duration) {
		sleeps++
		if delay != snapshotLeaseRetryDelay(0) {
			t.Errorf("first retry delay=%s; want %s", delay, snapshotLeaseRetryDelay(0))
		}
		unlock, err := platform.AcquireFileLock(filepath.Join(root, "snapshots.lock"))
		if err != nil {
			t.Errorf("root lock remained held while waiting for root job: %v", err)
			return
		}
		unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	defer release()
	if probes != 2 || sleeps != 1 {
		t.Fatalf("probes=%d sleeps=%d; want 2 and 1", probes, sleeps)
	}
}

func TestLeaseClaimRejectsActiveJobWithoutRunningCallback(t *testing.T) {
	root := t.TempDir()
	snapshot := makeSnapshot(t, root, "rebaseline-hub", "active001", ".ready")
	callbackCalled := false
	probeOpen := false
	err := WithSnapshotLock(root, func() error {
		acquired, err := tryWithSnapshotLeaseFileUnderRootUsing(root, snapshot, func(*os.File) error {
			callbackCalled = true
			return nil
		}, func(string, string) (snapshotJobState, error) {
			probeOpen = true
			defer func() { probeOpen = false }()
			return snapshotJobActive, nil
		})
		if err != nil {
			return err
		}
		if acquired {
			return errors.New("lease admitted while named job was active")
		}
		if probeOpen {
			return errors.New("job probe was left open after busy result")
		}
		if callbackCalled {
			return errors.New("lease callback ran while named job was active")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBlockingLeaseClaimDropsRootLockBeforeRetry(t *testing.T) {
	root := t.TempDir()
	snapshot := makeSnapshot(t, root, "install-bundle", "retry0001", "")
	probes, sleeps, callbacks := 0, 0, 0
	err := withSnapshotLeaseFileUsing(root, snapshot, func(*os.File) error {
		callbacks++
		return nil
	}, func(string, string) (snapshotJobState, error) {
		probes++
		if probes == 1 {
			return snapshotJobActive, nil
		}
		return snapshotJobAbsent, nil
	}, func(delay time.Duration) {
		sleeps++
		if delay != snapshotLeaseRetryDelay(0) {
			t.Errorf("first retry delay=%s; want %s", delay, snapshotLeaseRetryDelay(0))
		}
		release, err := platform.AcquireFileLock(filepath.Join(root, "snapshots.lock"))
		if err != nil {
			t.Errorf("root lock remained held while backing off: %v", err)
			return
		}
		release()
	})
	if err != nil {
		t.Fatal(err)
	}
	if probes != 2 || sleeps != 1 || callbacks != 1 {
		t.Fatalf("probes=%d sleeps=%d callbacks=%d; want 2, 1, 1", probes, sleeps, callbacks)
	}
}

func TestSnapshotLeaseJobNameIsStableAndBoundToRoot(t *testing.T) {
	root := t.TempDir()
	snapshot := filepath.Join(root, "install-bundle.a1b2c3d4")
	if err := os.Mkdir(snapshot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapshot, ".lease"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	first, err := SnapshotLeaseJobName(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SnapshotLeaseJobName(root, filepath.Join(root, ".", "install-bundle.a1b2c3d4"))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("same lease path produced different job names: %q and %q", first, second)
	}
	aliasRoot := filepath.Join(t.TempDir(), "root-alias")
	if err := os.Symlink(root, aliasRoot); err == nil {
		alias, aliasErr := SnapshotLeaseJobName(aliasRoot, filepath.Join(aliasRoot, filepath.Base(snapshot)))
		if aliasErr != nil {
			t.Fatal(aliasErr)
		}
		if alias != first {
			t.Fatalf("symlink alias changed job name: %q and %q", alias, first)
		}
	}
	if _, err := SnapshotLeaseJobName(root, filepath.Join(root, "..", "outside")); err == nil {
		t.Fatal("outside snapshot unexpectedly received a job name")
	}
}

func TestSnapshotRootJobNameIsStable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "snapshots.lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	first, err := SnapshotRootJobName(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SnapshotRootJobName(filepath.Join(root, "."))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("same root produced different job names: %q and %q", first, second)
	}
	relative, err := filepath.Rel(".", root)
	if err == nil {
		third, err := SnapshotRootJobName(relative)
		if err != nil {
			t.Fatal(err)
		}
		if third != first {
			t.Fatalf("relative root changed job name: %q and %q", third, first)
		}
	}
	alias := filepath.Join(t.TempDir(), "root-alias")
	if err := os.Symlink(root, alias); err == nil {
		aliased, err := SnapshotRootJobName(alias)
		if err != nil {
			t.Fatal(err)
		}
		if aliased != first {
			t.Fatalf("symlink root changed job name: %q and %q", aliased, first)
		}
	}
}
