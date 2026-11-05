package lifecycle

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/scenario"
	publiccapacity "github.com/vrooli/vrooli/packages/capacity"
)

type buildAdmissionFakeBroker struct {
	lease     publiccapacity.Lease
	verdict   publiccapacity.Verdict
	err       error
	onAcquire func() (publiccapacity.Lease, error)
	closed    atomic.Int32
}

func (b *buildAdmissionFakeBroker) Acquire(context.Context, string, int64, int64) (publiccapacity.Lease, publiccapacity.Verdict, error) {
	if b.onAcquire != nil {
		lease, err := b.onAcquire()
		return lease, b.verdict, err
	}
	return b.lease, b.verdict, b.err
}

func (b *buildAdmissionFakeBroker) Close() error {
	b.closed.Add(1)
	return nil
}

type buildAdmissionFakeLease struct {
	heartbeats atomic.Int32
	releases   atomic.Int32
	heartErr   error
	releaseErr error
	onRelease  func() error
}

func (l *buildAdmissionFakeLease) Heartbeat(context.Context) error {
	l.heartbeats.Add(1)
	return l.heartErr
}

func (l *buildAdmissionFakeLease) Release(context.Context) error {
	l.releases.Add(1)
	if l.onRelease != nil {
		return l.onRelease()
	}
	return l.releaseErr
}

func TestBuildDeclaredComponentsFreshSkipsAdmission(t *testing.T) {
	installTestBuilder(t)
	item := testBuildScenario(t)
	var factoryCalls atomic.Int32
	runner := &Runner{Root: item.Path, Home: t.TempDir(), buildAdmissionFactory: func(context.Context, string) (buildAdmissionBroker, error) {
		factoryCalls.Add(1)
		return nil, errors.New("fresh component must not open the ledger")
	}}
	count, err := runner.buildDeclaredComponents(context.Background(), item, nil, io.Discard, false, map[string]artifactVerdict{})
	if err != nil || count != 0 {
		t.Fatalf("fresh build = %d, %v; want skipped without error", count, err)
	}
	if got := factoryCalls.Load(); got != 0 {
		t.Fatalf("opened capacity broker %d times for fresh component", got)
	}
}

func TestBuildDeclaredComponentsDeniedAdmissionRunsNoCommand(t *testing.T) {
	installTestBuilder(t)
	item := testBuildScenario(t)
	broker := &buildAdmissionFakeBroker{verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictDeny, Reason: "test budget exhausted"}}
	runner := &Runner{Root: item.Path, Home: t.TempDir(), buildAdmissionFactory: func(context.Context, string) (buildAdmissionBroker, error) { return broker, nil }}
	_, err := runner.buildDeclaredComponents(context.Background(), item, nil, io.Discard, false, map[string]artifactVerdict{"api": {Stale: true}})
	if err == nil || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("denied build error = %v, want capacity denial", err)
	}
	if matches, _ := filepath.Glob(filepath.Join(item.Path, "api.artifact")); len(matches) != 0 {
		t.Fatalf("denied build wrote output: %v", matches)
	}
	if broker.closed.Load() != 1 {
		t.Fatalf("broker closes = %d, want 1", broker.closed.Load())
	}
}

func TestStaleComponentLeaseCoversInstallAndEveryBuildTarget(t *testing.T) {
	root := t.TempDir()
	item := scenario.Scenario{
		Slug: "lease-build-test",
		Path: root,
		Manifest: scenario.ServiceManifest{Components: map[string]scenario.Component{
			"api": {Role: "api", Build: scenario.ComponentBuild{Kind: "lease_test_builder", Dir: ".", Outputs: []scenario.ComponentBuildOutput{{Output: "api-secondary.artifact"}}}},
		}},
	}
	marker := filepath.Join(t.TempDir(), "claim-held")
	events := filepath.Join(t.TempDir(), "commands")
	script := filepath.Join(t.TempDir(), "builder.sh")
	scriptText := "#!/bin/sh\n"
	scriptText += "test -f '" + marker + "' || exit 71\n"
	scriptText += "printf '%s\\n' \"$1\" >> '" + events + "'\n"
	scriptText += "if [ \"$1\" != install ]; then printf ok > \"$1\"; fi\n"
	if err := os.WriteFile(script, []byte(scriptText), 0o755); err != nil {
		t.Fatal(err)
	}
	previous, existed := builderRegistry["lease_test_builder"]
	builderRegistry["lease_test_builder"] = BuilderSpec{
		Kind:                   "lease_test_builder",
		DefaultOutput:          "{dir}/{component}.artifact",
		MemoryReservationBytes: 1,
		Install:                []string{script, "install"},
		Build:                  []string{script, "{output}"},
	}
	t.Cleanup(func() {
		if existed {
			builderRegistry["lease_test_builder"] = previous
		} else {
			delete(builderRegistry, "lease_test_builder")
		}
	})
	lease := &buildAdmissionFakeLease{onRelease: func() error { return os.Remove(marker) }}
	broker := &buildAdmissionFakeBroker{
		verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant},
		onAcquire: func() (publiccapacity.Lease, error) {
			if err := os.WriteFile(marker, []byte("held"), 0o600); err != nil {
				return nil, err
			}
			return lease, nil
		},
	}
	runner := &Runner{Root: root, Home: t.TempDir(), buildAdmissionFactory: func(context.Context, string) (buildAdmissionBroker, error) { return broker, nil }}
	count, err := runner.buildDeclaredComponents(context.Background(), item, nil, io.Discard, false, map[string]artifactVerdict{"api": {Stale: true}})
	if err != nil || count != 3 {
		t.Fatalf("build result = %d, %v; want install and two targets under one lease", count, err)
	}
	content, err := os.ReadFile(events)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(content))
	if len(lines) != 3 || lines[0] != "install" {
		t.Fatalf("commands under lease = %q; want install and two build targets", content)
	}
	if lease.releases.Load() != 1 {
		t.Fatalf("lease releases = %d; want one after all targets", lease.releases.Load())
	}
}

func TestBuildAdmissionRenewalFailureCancelsWorkAndReleases(t *testing.T) {
	lease := &buildAdmissionFakeLease{heartErr: errors.New("stale claim generation")}
	broker := &buildAdmissionFakeBroker{lease: lease, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}
	runner := &Runner{Home: t.TempDir(), buildAdmissionFactory: func(context.Context, string) (buildAdmissionBroker, error) { return broker, nil }}
	workStarted := make(chan struct{})
	count, err := runner.withBuildAdmissionAtInterval(context.Background(), "scenario", "api", "run", 1, time.Millisecond, func(ctx context.Context) (int, error) {
		close(workStarted)
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if count != 0 || err == nil || !strings.Contains(err.Error(), "renewal failed") {
		t.Fatalf("admission result = %d, %v; want canceled work and renewal failure", count, err)
	}
	select {
	case <-workStarted:
	default:
		t.Fatal("build callback did not start")
	}
	if lease.heartbeats.Load() == 0 || lease.releases.Load() != 1 || broker.closed.Load() != 1 {
		t.Fatalf("heartbeat/release/close = %d/%d/%d; want >=1/1/1", lease.heartbeats.Load(), lease.releases.Load(), broker.closed.Load())
	}
}

func TestBuildAdmissionCallbackFailureReleasesWithIndependentContext(t *testing.T) {
	lease := &buildAdmissionFakeLease{}
	broker := &buildAdmissionFakeBroker{lease: lease, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}
	runner := &Runner{Home: t.TempDir(), buildAdmissionFactory: func(context.Context, string) (buildAdmissionBroker, error) { return broker, nil }}
	ctx, cancel := context.WithCancel(context.Background())
	count, err := runner.withBuildAdmission(ctx, "scenario", "ui", "run", 1, func(context.Context) (int, error) {
		cancel()
		return 2, context.Canceled
	})
	if count != 2 || !errors.Is(err, context.Canceled) {
		t.Fatalf("admission result = %d, %v; want callback cancellation", count, err)
	}
	if lease.releases.Load() != 1 || broker.closed.Load() != 1 {
		t.Fatalf("release/close = %d/%d; want 1/1", lease.releases.Load(), broker.closed.Load())
	}
}

func TestBuildAdmissionRejectsUnknownEstimateWithoutOpeningBroker(t *testing.T) {
	var factoryCalls atomic.Int32
	runner := &Runner{buildAdmissionFactory: func(context.Context, string) (buildAdmissionBroker, error) {
		factoryCalls.Add(1)
		return nil, errors.New("should not open")
	}}
	called := false
	_, err := runner.withBuildAdmission(context.Background(), "scenario", "api", "run", 0, func(context.Context) (int, error) {
		called = true
		return 1, nil
	})
	if err == nil || called || factoryCalls.Load() != 0 {
		t.Fatalf("unknown estimate result err=%v work=%t broker=%d; want fail closed before broker/command", err, called, factoryCalls.Load())
	}
}

func TestIndependentBuildInvocationsShareCapacityLedger(t *testing.T) {
	ledgerPath := filepath.Join(t.TempDir(), "capacity.db")
	first := testBuildRunnerWithCapacity(t, t.TempDir(), ledgerPath, 1<<30)
	second := testBuildRunnerWithCapacity(t, t.TempDir(), ledgerPath, 1<<30)
	started := make(chan struct{})
	finish := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, err := first.withBuildAdmission(context.Background(), "scenario", "api", "run-one", 1<<30, func(context.Context) (int, error) {
			close(started)
			<-finish
			return 1, nil
		})
		firstDone <- err
	}()
	<-started
	secondWorkCalled := false
	_, secondErr := second.withBuildAdmission(context.Background(), "scenario", "ui", "run-two", 1<<30, func(context.Context) (int, error) {
		secondWorkCalled = true
		return 1, nil
	})
	close(finish)
	if err := <-firstDone; err != nil {
		t.Fatalf("first admission: %v", err)
	}
	if secondErr == nil || secondWorkCalled || !strings.Contains(secondErr.Error(), "capacity") {
		t.Fatalf("second admission err=%v work=%t; want shared-ledger denial before build", secondErr, secondWorkCalled)
	}
}
