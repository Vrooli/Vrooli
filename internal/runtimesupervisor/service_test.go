package runtimesupervisor

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/testenv"
)

// stubListenerSnapshot pins the snapshot seam so ticks read controlled
// listener evidence instead of the live host's TCP table. Returns a pointer
// to the capture count.
func stubListenerSnapshot(t *testing.T, snapshot network.TCPListenerSnapshot, onCapture func()) *int {
	t.Helper()
	original := captureListenerSnapshotFn
	t.Cleanup(func() { captureListenerSnapshotFn = original })
	captures := 0
	captureListenerSnapshotFn = func() network.TCPListenerSnapshot {
		captures++
		if onCapture != nil {
			onCapture()
		}
		return snapshot
	}
	return &captures
}

type fakeHostProvider struct {
	snapshot hostsession.Snapshot
}

type fakePressureProvider struct{ state PressureState }

func (p *fakePressureProvider) Snapshot(context.Context) PressureState { return p.state }

func (p fakeHostProvider) Current(context.Context, string) (hostsession.Snapshot, error) {
	return p.snapshot, nil
}

func TestModeFromEnvDefaultsToAuto(t *testing.T) {
	t.Setenv(ModeEnv, "")
	if got := ModeFromEnv(); got != ModeAuto {
		t.Fatalf("ModeFromEnv() = %q, want %q", got, ModeAuto)
	}
	t.Setenv(ModeEnv, ModeOff)
	if got := ModeFromEnv(); got != ModeOff {
		t.Fatalf("ModeFromEnv(off) = %q, want %q", got, ModeOff)
	}
}

func TestServiceEnsureStartedRefusesLivePeerWithActionablePID(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	peerPID := 43210
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{SupervisorID: "peer", HostBootID: "boot", HostSessionID: "session", PID: &peerPID}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	svc := New(Config{DBPath: dbPath, SupervisorID: "new", HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}}, PIDRunning: func(pid int) bool { return pid == peerPID }})
	defer svc.Close()
	err = svc.ensureStarted(ctx)
	if !errors.Is(err, scenarioruntime.ErrSupervisorAlreadyRunning) || !strings.Contains(err.Error(), "43210") || !strings.Contains(err.Error(), "kill -TERM 43210") {
		t.Fatalf("ensureStarted error = %v", err)
	}
}

func TestServiceEnsureStartedTakeoverStopsPeerAndClaimsDatabase(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	peerPID := 43211
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{SupervisorID: "peer", HostBootID: "boot", HostSessionID: "session", PID: &peerPID}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	stopped := 0
	svc := New(Config{DBPath: dbPath, SupervisorID: "new", Takeover: true, HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}}, PIDRunning: func(pid int) bool { return pid == peerPID }, StopPIDFunc: func(pid int) error { stopped = pid; return nil }})
	defer svc.Close()
	if err := svc.ensureStarted(ctx); err != nil {
		t.Fatalf("ensureStarted takeover: %v", err)
	}
	if stopped != peerPID || svc.session.SupervisorID != "new" {
		t.Fatalf("stopped=%d session=%+v", stopped, svc.session)
	}
	running, err := svc.store.ListSupervisorSessions(ctx, scenarioruntime.SupervisorSessionFilter{Statuses: []string{scenarioruntime.SupervisorStatusRunning}})
	if err != nil || len(running) != 1 || running[0].SupervisorID != "new" {
		t.Fatalf("running sessions = %+v, err=%v", running, err)
	}
}

func TestServiceBuildIdentityFallsBackFromUnknownFingerprint(t *testing.T) {
	svc := New(Config{BuildIdentity: "unknown", Version: "dev-version"})
	defer svc.Close()
	if got := svc.buildIdentity(); got != "dev-version" {
		t.Fatalf("build identity = %q, want dev-version", got)
	}
}

func TestServiceStatusExposesDurableRecoveryContract(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if _, err := store.UpsertRecoveryPolicy(ctx, scenarioruntime.RecoveryPolicy{
		Scenario: "critical-api", Critical: true, Enabled: true, DependencyTier: 1, RetryBudget: 2,
	}); err != nil {
		t.Fatalf("UpsertRecoveryPolicy: %v", err)
	}
	if _, err := store.CreatePressureEpoch(ctx, scenarioruntime.PressureEpoch{EpochID: "epoch-1", Source: "test"}); err != nil {
		t.Fatalf("CreatePressureEpoch: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc := New(Config{DBPath: dbPath, Clock: clk, HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}}})
	defer svc.Close()
	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(report.RecoveryPolicies) != 1 || report.RecoveryPolicies[0].Scenario != "critical-api" {
		t.Fatalf("RecoveryPolicies = %#v", report.RecoveryPolicies)
	}
	if len(report.PressureEpochs) != 1 || report.PressureEpochs[0].EpochID != "epoch-1" {
		t.Fatalf("PressureEpochs = %#v", report.PressureEpochs)
	}
}

func TestServiceRecoveryWaitsForPressureClearAndRestoresOnlyDeclaredCriticalWorkload(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	for _, scenarioName := range []string{"critical-api", "ordinary-api"} {
		if _, err := store.CreateInstance(ctx, scenarioruntime.Instance{Scenario: scenarioName, Status: scenarioruntime.StatusExpired}); err != nil {
			t.Fatalf("CreateInstance(%s): %v", scenarioName, err)
		}
	}
	if _, err := store.UpsertRecoveryPolicy(ctx, scenarioruntime.RecoveryPolicy{Scenario: "critical-api", Critical: true, Enabled: true, RetryBudget: 1}); err != nil {
		t.Fatalf("UpsertRecoveryPolicy: %v", err)
	}
	if _, err := store.UpsertRecoveryPolicy(ctx, scenarioruntime.RecoveryPolicy{Scenario: "ordinary-api", Critical: false, Enabled: true, RetryBudget: 1}); err != nil {
		t.Fatalf("UpsertRecoveryPolicy ordinary: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	pressure := &fakePressureProvider{state: PressureState{Known: true, UnderPressure: true, ObservedAt: clk.Now(), Source: "fixture"}}
	var launches []RecoveryLaunchRequest
	svc := New(Config{
		DBPath: dbPath, SupervisorID: "sup-recovery", Clock: clk,
		HostProvider:     fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}},
		PressureProvider: pressure, RecoveryQuietPeriod: time.Minute,
		RecoveryLaunch: func(_ context.Context, request RecoveryLaunchRequest) error {
			launches = append(launches, request)
			return nil
		},
	})
	defer svc.Close()
	first, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick(pressure): %v", err)
	}
	if first.Recovery.Gated != 1 || len(launches) != 0 {
		t.Fatalf("pressure tick = %#v launches=%#v, want gated no launch", first.Recovery, launches)
	}
	pressure.state.UnderPressure = false
	clk.Advance(30 * time.Second)
	pressure.state.ObservedAt = clk.Now()
	second, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick(initial clear): %v", err)
	}
	if second.Recovery.Gated != 1 || len(launches) != 0 {
		t.Fatalf("initial clear tick = %#v launches=%#v, want gated no launch", second.Recovery, launches)
	}
	epochs, err := svc.store.ListPressureEpochs(ctx, 1)
	if err != nil || len(epochs) != 1 || epochs[0].Status != scenarioruntime.PressureEpochGated {
		t.Fatalf("pressure epoch after clear = %#v err=%v, want gated", epochs, err)
	}
	clk.Advance(time.Minute)
	pressure.state.ObservedAt = clk.Now()
	third, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick(stable clear): %v", err)
	}
	if third.Recovery.Restored != 1 || len(launches) != 1 || launches[0].Scenario != "critical-api" {
		t.Fatalf("stable clear = %#v launches=%#v, want only critical-api restored", third.Recovery, launches)
	}
	epochs, err = svc.store.ListPressureEpochs(ctx, 1)
	if err != nil || len(epochs) != 1 || epochs[0].Status != scenarioruntime.PressureEpochCleared {
		t.Fatalf("pressure epoch after dispatch = %#v err=%v, want cleared", epochs, err)
	}
	// The observed expired lease remains until lifecycle reconciliation sees the
	// restart. A later tick must not replay an already accepted launch.
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatalf("Tick(duplicate clear): %v", err)
	}
	if len(launches) != 1 {
		t.Fatalf("launches after duplicate tick = %#v, want exactly one", launches)
	}
}

func TestServiceTickAdoptsAndRenewsRunningInstance(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	clk.Advance(10 * time.Second)
	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Renewed != 1 || report.Unverified != 0 || report.Expired != 0 {
		t.Fatalf("report = %#v, want one renewed", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	after, err := check.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.SupervisorID != "sup-alpha" || after.OwnerKind != scenarioruntime.OwnerKindSupervisor {
		t.Fatalf("after supervision = %#v, want sup-alpha owner", after)
	}
	if after.HeartbeatDeadlineAt == nil || !after.HeartbeatDeadlineAt.Equal(clk.Now().Add(90*time.Second)) {
		t.Fatalf("deadline = %#v, want %s", after.HeartbeatDeadlineAt, clk.Now().Add(90*time.Second))
	}
	if after.ReconciliationStatus != string(scenarioruntime.ReconcileVerifiedRunning) {
		t.Fatalf("reconciliation = %q, want verified", after.ReconciliationStatus)
	}
}

func TestServiceTickPersistsListenerEvidenceForBoundClaims(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       18080,
		Status:     scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	listenerPID := 2468
	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PortListener: func(port int) scenarioruntime.ListenerEvidence {
			if port != 18080 {
				t.Fatalf("inspected port = %d, want 18080", port)
			}
			return scenarioruntime.ListenerEvidence{Known: true, Listening: true, PID: &listenerPID, ProcessLabel: "alpha-api"}
		},
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Renewed != 1 || report.Unverified != 0 || report.Expired != 0 {
		t.Fatalf("report = %#v, want one renewed", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	claims, err := check.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claims = %#v, want one", claims)
	}
	claim := claims[0]
	if claim.ListenerStatus != scenarioruntime.ListenerStatusListening {
		t.Fatalf("ListenerStatus = %q, want listening", claim.ListenerStatus)
	}
	if claim.LastListenerCheckAt == nil || !claim.LastListenerCheckAt.Equal(clk.Now()) {
		t.Fatalf("LastListenerCheckAt = %#v, want %s", claim.LastListenerCheckAt, clk.Now())
	}
	if claim.LastListenerSeenAt == nil || !claim.LastListenerSeenAt.Equal(clk.Now()) {
		t.Fatalf("LastListenerSeenAt = %#v, want %s", claim.LastListenerSeenAt, clk.Now())
	}
	if claim.ListenerPID == nil || *claim.ListenerPID != listenerPID || claim.ListenerProcessLabel != "alpha-api" {
		t.Fatalf("listener identity = pid %#v label %q, want %d alpha-api", claim.ListenerPID, claim.ListenerProcessLabel, listenerPID)
	}
}

// TestServiceTickRealSnapshotBranchConvertsAndCapturesOnce exercises the real
// tickPortListener branch (no cfg.PortListener override): exactly one snapshot
// capture per tick regardless of claim count, listening ports convert to
// Known/Listening evidence with PID+label attribution, and an unavailable
// snapshot degrades to Known:false (never false-"not listening").
func TestServiceTickRealSnapshotBranchConvertsAndCapturesOnce(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	for _, claim := range []scenarioruntime.PortClaim{
		{ClaimID: "claim-alpha-api", InstanceID: instance.InstanceID, Scenario: "alpha", PortName: "api", Port: 18080, Status: scenarioruntime.ClaimStatusBound},
		{ClaimID: "claim-alpha-ui", InstanceID: instance.InstanceID, Scenario: "alpha", PortName: "ui", Port: 18081, Status: scenarioruntime.ClaimStatusBound},
	} {
		if _, err := store.AcquirePortClaim(ctx, claim); err != nil {
			t.Fatalf("AcquirePortClaim(%s): %v", claim.ClaimID, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	captures := stubListenerSnapshot(t, network.TCPListenerSnapshot{
		Known: true,
		Tool:  "test",
		Ports: map[int][]network.SnapshotListener{
			18080: {{PID: 2468, Label: "alpha-api"}},
		},
	}, nil)

	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
	})
	defer svc.Close()
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if *captures != 1 {
		t.Fatalf("snapshot captures = %d, want exactly 1 per tick", *captures)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	claims, err := check.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	byID := map[string]scenarioruntime.PortClaim{}
	for _, c := range claims {
		byID[c.ClaimID] = c
	}
	api := byID["claim-alpha-api"]
	if api.ListenerStatus != scenarioruntime.ListenerStatusListening {
		t.Fatalf("api ListenerStatus = %q, want listening", api.ListenerStatus)
	}
	if api.ListenerPID == nil || *api.ListenerPID != 2468 || api.ListenerProcessLabel != "alpha-api" {
		t.Fatalf("api listener identity = pid %#v label %q, want 2468 alpha-api", api.ListenerPID, api.ListenerProcessLabel)
	}
	ui := byID["claim-alpha-ui"]
	if ui.ListenerStatus == scenarioruntime.ListenerStatusListening {
		t.Fatalf("ui ListenerStatus = %q, want not-listening (port absent from snapshot)", ui.ListenerStatus)
	}
}

// TestServiceTickCapturesSnapshotAfterClaimReads pins the freshness ordering:
// the snapshot must be captured AFTER the tick's store reads, so evidence is
// at least as fresh as the claim set. The stub binds a NEW claim at capture
// time; with correct ordering that claim is not part of this tick's read set
// and its listener evidence stays untouched. A refactor that hoists the
// capture above the claim reads would pick the claim up and stamp it against
// evidence that predates its bind — the false-expiry race.
func TestServiceTickCapturesSnapshotAfterClaimReads(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID: "claim-alpha-api", InstanceID: instance.InstanceID, Scenario: "alpha",
		PortName: "api", Port: 18080, Status: scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	stubListenerSnapshot(t, network.TCPListenerSnapshot{
		Known: true,
		Tool:  "test",
		Ports: map[int][]network.SnapshotListener{18080: nil},
	}, func() {
		if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
			ClaimID: "claim-alpha-late", InstanceID: instance.InstanceID, Scenario: "alpha",
			PortName: "late", Port: 19090, Status: scenarioruntime.ClaimStatusBound,
		}); err != nil {
			t.Errorf("AcquirePortClaim(late): %v", err)
		}
	})

	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
	})
	defer svc.Close()
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatalf("Tick() error = %v", err)
	}

	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	for _, claim := range claims {
		if claim.ClaimID != "claim-alpha-late" {
			continue
		}
		if claim.LastListenerCheckAt != nil || claim.ListenerStatus == scenarioruntime.ListenerStatusNotListening {
			t.Fatalf("claim bound during capture was stamped with pre-bind evidence: %#v — snapshot captured before the claim reads", claim)
		}
		return
	}
	t.Fatal("late claim missing; capture stub did not run")
}

func TestServiceTickReconcilesLiveStartingInstance(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-starting",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusStarting,
		Phase:         "develop",
		ScopePath:     t.TempDir(),
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-starting-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       18081,
		Status:     scenarioruntime.ClaimStatusReserved,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	pid := 3579
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID:      "ref-starting-api",
		InstanceID: instance.InstanceID,
		PID:        &pid,
		Step:       "start-api",
		Status:     "running",
		HostBootID: "boot-current",
	}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	listenerPID := pid
	ready := true
	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning: func(got int) bool {
			return got == pid
		},
		PortListener: func(port int) scenarioruntime.ListenerEvidence {
			if port != 18081 {
				t.Fatalf("inspected port = %d, want 18081", port)
			}
			return scenarioruntime.ListenerEvidence{Known: true, Listening: true, PID: &listenerPID, ProcessLabel: "alpha-api"}
		},
		HealthProber: func(_ context.Context, got scenarioruntime.Instance, _ []scenarioruntime.PortClaim) scenarioruntime.HealthSnapshot {
			if got.InstanceID != instance.InstanceID {
				t.Fatalf("health instance = %s, want %s", got.InstanceID, instance.InstanceID)
			}
			now := clk.Now()
			return scenarioruntime.HealthSnapshot{
				InstanceID: got.InstanceID,
				Scenario:   got.Scenario,
				Status:     scenarioruntime.HealthStatusHealthy,
				Readiness:  &ready,
				CheckedAt:  &now,
			}
		},
	})
	defer svc.Close()

	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Renewed != 1 || report.Unverified != 0 || report.Expired != 0 {
		t.Fatalf("report = %#v, want reconciled instance renewed", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	after, err := check.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.Status != scenarioruntime.StatusRunning || after.SupervisorID != "sup-alpha" {
		t.Fatalf("after = %#v, want running supervised", after)
	}
	claims, err := check.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusBound {
		t.Fatalf("claims = %#v, want one bound claim", claims)
	}
}

func TestServiceTickExpiresPreviousBootInstance(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-old",
		HostSessionID: "session-old",
		SupervisorID:  "sup-alpha",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Expired != 1 || report.Renewed != 0 {
		t.Fatalf("report = %#v, want one expired", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	after, err := check.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.Status != scenarioruntime.StatusExpired {
		t.Fatalf("after.Status = %q, want expired", after.Status)
	}
}

func TestServiceTickRunsHealthProbesWithBoundedConcurrency(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := store.CreateLease(ctx, scenarioruntime.Instance{
			InstanceID:    "",
			Scenario:      "alpha",
			Status:        scenarioruntime.StatusRunning,
			HostBootID:    "boot-current",
			HostSessionID: "session-current",
		}, time.Minute); err != nil {
			t.Fatalf("CreateLease(%d): %v", i, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var mu sync.Mutex
	var inFlight int
	var maxInFlight int
	prober := func(_ context.Context, instance scenarioruntime.Instance, _ []scenarioruntime.PortClaim) scenarioruntime.HealthSnapshot {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
		now := clk.Now()
		ready := true
		return scenarioruntime.HealthSnapshot{
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			Status:     scenarioruntime.HealthStatusHealthy,
			Readiness:  &ready,
			CheckedAt:  &now,
		}
	}

	svc := New(Config{
		DBPath:               dbPath,
		SupervisorID:         "sup-alpha",
		Clock:                clk,
		HostProvider:         fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		HealthInterval:       time.Minute,
		MaxHealthConcurrency: 2,
		HealthProber:         prober,
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.HealthProbeCount != 8 {
		t.Fatalf("HealthProbeCount = %d, want 8", report.HealthProbeCount)
	}
	if maxInFlight > 2 {
		t.Fatalf("max concurrent health probes = %d, want <= 2", maxInFlight)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	instances, err := check.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: []string{scenarioruntime.StatusRunning}})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	for _, instance := range instances {
		snapshot, err := check.GetHealthSnapshot(ctx, instance.InstanceID)
		if err != nil {
			t.Fatalf("GetHealthSnapshot(%s): %v", instance.InstanceID, err)
		}
		if snapshot.Status != scenarioruntime.HealthStatusHealthy {
			t.Fatalf("snapshot.Status = %q, want healthy", snapshot.Status)
		}
	}
}

func TestServiceStatusReportsDeadWhenRecordedPIDIsMissing(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	pid := 424242
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  "sup-stale-pid",
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
		PID:           &pid,
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc := New(Config{
		DBPath:       dbPath,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning: func(got int) bool {
			if got != pid {
				t.Fatalf("PIDRunning(%d), want %d", got, pid)
			}
			return false
		},
	})
	defer svc.Close()

	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Status != StatusDead {
		t.Fatalf("report.Status = %q, want %q; report=%#v", report.Status, StatusDead, report)
	}
	if report.StatusReason == "" {
		t.Fatalf("report.StatusReason is empty, want PID remediation context")
	}
}

func TestServiceStatusReportsStaleWhenHeartbeatExpired(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	pid := 5151
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  "sup-expired",
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
		PID:           &pid,
	}, 10*time.Second); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	clk.Advance(11 * time.Second)

	svc := New(Config{
		DBPath:       dbPath,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning:   func(int) bool { return true },
	})
	defer svc.Close()

	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Status != StatusStale {
		t.Fatalf("report.Status = %q, want %q; report=%#v", report.Status, StatusStale, report)
	}
	if report.StatusReason == "" {
		t.Fatalf("report.StatusReason is empty, want heartbeat context")
	}
}

func TestServiceStatusReportsRunningOnlyForFreshLiveSession(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	pid := 6161
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  "sup-live",
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
		PID:           &pid,
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc := New(Config{
		DBPath:       dbPath,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning:   func(int) bool { return true },
	})
	defer svc.Close()

	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Status != scenarioruntime.SupervisorStatusRunning {
		t.Fatalf("report.Status = %q, want running; report=%#v", report.Status, report)
	}
	if report.StatusReason != "" {
		t.Fatalf("report.StatusReason = %q, want empty", report.StatusReason)
	}
}

// flakyStore wraps a real store and fails the first call of the tick's opening
// operation a fixed number of times, so Run's failure tolerance can be
// exercised without a hand-written fake of the whole Store surface.
type flakyStore struct {
	Store
	remainingFailures int
	ticks             int
}

func (f *flakyStore) HeartbeatSupervisorSession(ctx context.Context, supervisorID string, ttl time.Duration) (scenarioruntime.SupervisorSession, error) {
	f.ticks++
	if f.remainingFailures > 0 {
		f.remainingFailures--
		return scenarioruntime.SupervisorSession{}, errors.New("transient store failure")
	}
	return f.Store.HeartbeatSupervisorSession(ctx, supervisorID, ttl)
}

// A transient tick failure used to propagate out of Run and exit the process,
// so every scenario's lease expired because one store call blipped.
func TestRunSurvivesTransientTickFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	clk := testenv.NewClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")

	var flaky *flakyStore
	cfg := Config{
		DBPath:        dbPath,
		Clock:         clk,
		RenewInterval: time.Millisecond,
		HostProvider:  fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}},
		Stderr:        io.Discard,
		StoreFactory: func(ctx context.Context, c scenarioruntime.Config) (Store, error) {
			inner, err := scenarioruntime.NewSQLiteStore(ctx, c)
			if err != nil {
				return nil, err
			}
			flaky = &flakyStore{Store: inner, remainingFailures: MaxConsecutiveTickFailures - 1}
			return flaky, nil
		},
	}

	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg) }()

	deadline := time.After(10 * time.Second)
	for {
		if flaky != nil && flaky.remainingFailures == 0 && flaky.ticks > MaxConsecutiveTickFailures {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("Run exited on a transient failure: %v", err)
		case <-deadline:
			t.Fatal("timed out waiting for the supervisor to recover from transient failures")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

// Tolerance must be bounded: a supervisor that can never make progress should
// exit so its service manager replaces it with a fresh process.
func TestRunExitsAfterSustainedTickFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clk := testenv.NewClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")

	cfg := Config{
		DBPath:        dbPath,
		Clock:         clk,
		RenewInterval: time.Millisecond,
		HostProvider:  fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}},
		Stderr:        io.Discard,
		StoreFactory: func(ctx context.Context, c scenarioruntime.Config) (Store, error) {
			inner, err := scenarioruntime.NewSQLiteStore(ctx, c)
			if err != nil {
				return nil, err
			}
			return &flakyStore{Store: inner, remainingFailures: MaxConsecutiveTickFailures * 2}, nil
		},
	}

	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Run() returned nil, want the sustained failure surfaced")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not exit after sustained tick failures")
	}
}

// Taking over is the natural moment to clear predecessors that were killed
// before they could record their own shutdown.
func TestSupervisorStartupRetiresDeadPredecessorSessions(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	deadPID := 424242
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID: "sup-killed", HostBootID: "boot", HostSessionID: "session", PID: &deadPID,
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	clk.Advance(2 * time.Minute)

	svc := New(Config{
		DBPath:       dbPath,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}},
		PIDRunning:   func(pid int) bool { return pid != deadPID },
		Stderr:       io.Discard,
	})
	defer svc.Close()
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore(check): %v", err)
	}
	defer check.Close()
	running, err := check.ListSupervisorSessions(ctx, scenarioruntime.SupervisorSessionFilter{
		Statuses: []string{scenarioruntime.SupervisorStatusRunning},
	})
	if err != nil {
		t.Fatalf("ListSupervisorSessions: %v", err)
	}
	for _, session := range running {
		if session.SupervisorID == "sup-killed" {
			t.Fatal("a SIGKILLed predecessor is still reported as running")
		}
	}
	if len(running) != 1 {
		t.Fatalf("running sessions = %d, want only the live one", len(running))
	}
}
