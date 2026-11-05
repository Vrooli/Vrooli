package cliinstall

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
	publiccapacity "github.com/vrooli/vrooli/packages/capacity"
)

type admissionTestBroker struct {
	lease   publiccapacity.Lease
	verdict publiccapacity.Verdict
	err     error
}

func (b admissionTestBroker) Acquire(context.Context, string, int64, int64) (publiccapacity.Lease, publiccapacity.Verdict, error) {
	return b.lease, b.verdict, b.err
}
func (admissionTestBroker) Close() error { return nil }

type admissionTestLease struct {
	heartbeatErr error
	releaseErr   error
	releaseCheck func(context.Context) error
	heartbeats   atomic.Int32
	releases     atomic.Int32
}

func (l *admissionTestLease) Heartbeat(context.Context) error {
	l.heartbeats.Add(1)
	return l.heartbeatErr
}
func (l *admissionTestLease) Release(ctx context.Context) error {
	l.releases.Add(1)
	if l.releaseCheck != nil {
		l.releaseErr = l.releaseCheck(ctx)
	}
	return l.releaseErr
}

func withAdmissionFactory(t *testing.T, factory func(context.Context) (cliAdmissionBroker, error), interval time.Duration) {
	t.Helper()
	oldFactory := openCLIBuildBroker
	oldInterval := cliBuildHeartbeatInterval
	openCLIBuildBroker = factory
	cliBuildHeartbeatInterval = func() time.Duration { return interval }
	t.Cleanup(func() {
		openCLIBuildBroker = oldFactory
		cliBuildHeartbeatInterval = oldInterval
	})
}

func TestCLIInstallAdmissionDenialDoesNotStartInstaller(t *testing.T) {
	withAdmissionFactory(t, func(context.Context) (cliAdmissionBroker, error) {
		return admissionTestBroker{verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictDeny, Reason: "fixture capacity denial"}}, nil
	}, time.Millisecond)

	started := false
	err := withCLIInstallAdmission(context.Background(), "scenario-demo", func(context.Context) error {
		started = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "fixture capacity denial") {
		t.Fatalf("withCLIInstallAdmission() error = %v, want capacity denial", err)
	}
	if started {
		t.Fatal("denied CLI install started compiler work")
	}
}

func TestCLIInstallAdmissionCancelsWorkAndReleasesLease(t *testing.T) {
	lease := &admissionTestLease{}
	withAdmissionFactory(t, func(context.Context) (cliAdmissionBroker, error) {
		return admissionTestBroker{lease: lease, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}, nil
	}, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- withCLIInstallAdmission(ctx, "scenario-demo", func(workCtx context.Context) error {
			close(started)
			<-workCtx.Done()
			return workCtx.Err()
		})
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("withCLIInstallAdmission() error = %v, want context.Canceled", err)
	}
	if got := lease.releases.Load(); got != 1 {
		t.Fatalf("lease releases = %d, want 1", got)
	}
}

func TestCLIInstallAdmissionWaitsForOwnedProcessGroupBeforeRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the platform process-group waiter is not qualified on Windows")
	}
	shellPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("POSIX shell fixture unavailable: %v", err)
	}
	marker := filepath.Join(t.TempDir(), "parent-pid")
	var groupID int
	var releaseCheckErr error
	lease := &admissionTestLease{releaseCheck: func(ctx context.Context) error {
		if groupID <= 0 {
			return errors.New("build process group was not captured")
		}
		releaseCheckErr = platform.WaitProcessGroupContext(ctx, groupID, 10*time.Millisecond)
		return releaseCheckErr
	}}
	withAdmissionFactory(t, func(context.Context) (cliAdmissionBroker, error) {
		return admissionTestBroker{lease: lease, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}, nil
	}, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		var cmd *exec.Cmd
		done <- withCLIInstallAdmission(ctx, "scenario-process-group", func(workCtx context.Context) error {
			cmd = exec.CommandContext(workCtx, shellPath, "-c", "echo $$ > \"$1\"; sleep 30 & wait", "cli-build-fixture", marker)
			return runOwnedCLICommand(workCtx, cmd)
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		data, readErr := os.ReadFile(marker)
		if readErr == nil {
			groupID, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil || groupID <= 0 {
				t.Fatalf("fixture process-group id = %q, err = %v", data, err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("owned build child did not start: %v", readErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("owned build cancellation error = %v, want context.Canceled", err)
	}
	if lease.releases.Load() != 1 {
		t.Fatalf("lease releases = %d, want one release after group exit", lease.releases.Load())
	}
	if releaseCheckErr != nil {
		t.Fatalf("lease release observed a live descendant: %v", releaseCheckErr)
	}
}

func TestCLIInstallAdmissionStopsDescendantsAfterSuccessfulRootExit(t *testing.T) {
	shellPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("shell fixture requires sh")
	}
	marker := filepath.Join(t.TempDir(), "descendant.pid")
	lease := &admissionTestLease{}
	var releaseCheckErr error
	lease.releaseCheck = func(ctx context.Context) error {
		data, readErr := os.ReadFile(marker)
		if readErr == nil {
			fields := strings.Fields(string(data))
			if len(fields) > 0 {
				groupID, parseErr := strconv.Atoi(fields[0])
				if parseErr == nil && groupID > 0 {
					releaseCheckErr = platform.WaitProcessGroupContext(ctx, groupID, 5*time.Millisecond)
				}
			}
		}
		return releaseCheckErr
	}
	withAdmissionFactory(t, func(context.Context) (cliAdmissionBroker, error) {
		return admissionTestBroker{lease: lease, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}, nil
	}, time.Hour)

	err = withCLIInstallAdmission(context.Background(), "scenario-root-exits-first", func(workCtx context.Context) error {
		cmd := exec.CommandContext(workCtx, shellPath, "-c", "sleep 30 & echo $$ $! > \"$1\"; exit 0", "cli-build-fixture", marker)
		return runOwnedCLICommand(workCtx, cmd)
	})
	if err == nil || !strings.Contains(err.Error(), "descendants remained") {
		t.Fatalf("successful root with a live descendant error = %v, want a failed install after stopping the group", err)
	}
	data, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("descendant marker was not written: %v", readErr)
	}
	fields := strings.Fields(string(data))
	if len(fields) != 2 {
		t.Fatalf("expected shell and descendant pids, got %q", data)
	}
	for _, field := range fields {
		pid, parseErr := strconv.Atoi(field)
		if parseErr != nil || pid <= 0 {
			t.Fatalf("fixture pid = %q, err = %v", field, parseErr)
		}
	}
	if releaseCheckErr != nil {
		t.Fatalf("lease release observed a live descendant: %v", releaseCheckErr)
	}
	if lease.releases.Load() != 1 {
		t.Fatalf("lease releases = %d, want one release after group exit", lease.releases.Load())
	}
}

func TestCLIInstallAdmissionRetainsLeaseWhenProcessGroupCleanupIsUnverified(t *testing.T) {
	lease := &admissionTestLease{}
	withAdmissionFactory(t, func(context.Context) (cliAdmissionBroker, error) {
		return admissionTestBroker{lease: lease, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}, nil
	}, time.Hour)
	err := withCLIInstallAdmission(context.Background(), "scenario-unverified-descendants", func(context.Context) error {
		return errCLIProcessGroupCleanupUnverified
	})
	if !errors.Is(err, errCLIProcessGroupCleanupUnverified) {
		t.Fatalf("cleanup-unverified error = %v, want sentinel", err)
	}
	if lease.releases.Load() != 0 {
		t.Fatalf("lease releases = %d, want no release when descendants may remain", lease.releases.Load())
	}
}

func TestCLIInstallAdmissionRenewalFailureCancelsBuildAndReleases(t *testing.T) {
	lease := &admissionTestLease{heartbeatErr: errors.New("renewal unavailable")}
	withAdmissionFactory(t, func(context.Context) (cliAdmissionBroker, error) {
		return admissionTestBroker{lease: lease, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}, nil
	}, time.Millisecond)

	buildCanceled := make(chan struct{})
	err := withCLIInstallAdmission(context.Background(), "scenario-demo", func(workCtx context.Context) error {
		<-workCtx.Done()
		close(buildCanceled)
		return workCtx.Err()
	})
	<-buildCanceled
	if err == nil || !strings.Contains(err.Error(), "renewal unavailable") {
		t.Fatalf("withCLIInstallAdmission() error = %v, want renewal error", err)
	}
	if got := lease.heartbeats.Load(); got == 0 {
		t.Fatal("lease was not renewed")
	}
	if got := lease.releases.Load(); got != 1 {
		t.Fatalf("lease releases = %d, want 1", got)
	}
}

func TestCLIInstallAdmissionsShareLedgerBudget(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	snapshot := hostinventory.Snapshot{Memory: hostinventory.Memory{
		TotalBytes:     700 << 20,
		AvailableBytes: 600 << 20,
	}}
	withAdmissionFactory(t, func(ctx context.Context) (cliAdmissionBroker, error) {
		store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath})
		if err != nil {
			return nil, err
		}
		return publiccapacity.NewBrokerWithSource(store, internalcapacity.StaticSource{Inventory: snapshot}), nil
	}, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- withCLIInstallAdmission(ctx, "scenario-first", func(workCtx context.Context) error {
			close(started)
			<-workCtx.Done()
			return workCtx.Err()
		})
	}()
	<-started

	secondStarted := false
	err := withCLIInstallAdmission(context.Background(), "scenario-second", func(context.Context) error {
		secondStarted = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "deny") {
		t.Fatalf("second shared-ledger admission error = %v, want capacity denial", err)
	}
	if secondStarted {
		t.Fatal("second build started despite insufficient shared-ledger headroom")
	}
	cancel()
	if err := <-firstDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("first build cleanup error = %v, want context.Canceled", err)
	}
}
