package checks

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/testenv"
)

// stubGate answers every restart question with the same verdict.
type stubGate struct{ verdict GateVerdict }

func (g *stubGate) AllowsAutoHealRestart(context.Context, string, string) GateVerdict {
	return g.verdict
}

// suspendableRegistry builds a non-scenario check that fails every heal, so
// three ticks suspend it. The check result is driven by the returned pointer.
func suspendableRegistry(t *testing.T, status Status) (*Registry, *fixedClock, *mockHealableCheck, *healIncidentReporterSpy) {
	t.Helper()
	reporter := &healIncidentReporterSpy{}
	reg, clock := newIncidentEscalationRegistry(reporter, ActionResult{Success: false, Error: "start failed"})
	check := &mockHealableCheck{
		id:            "resource-qdrant",
		result:        Result{CheckID: "resource-qdrant", Status: status, Message: "qdrant stopped"},
		actions:       []RecoveryAction{{ID: "start", Available: true}},
		executeResult: ActionResult{Success: false, Error: "start failed"},
	}
	reg.Unregister("resource-qdrant")
	reg.Register(check)
	return reg, clock, check, reporter
}

func suspendNonScenarioCheck(t *testing.T, reg *Registry, clock *fixedClock) {
	t.Helper()
	for i := 0; i < 3; i++ {
		reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
		clock.Advance(10 * time.Minute)
	}
	if tracker, _ := reg.GetHealTracker("resource-qdrant"); !tracker.IsSuspended() {
		t.Fatalf("fixture did not suspend: %+v", tracker)
	}
}

func TestSuspensionIsLoudAndClearsWhenAnyCheckIsHealthy(t *testing.T) {
	reg, clock, check, _ := suspendableRegistry(t, StatusCritical)
	suspendNonScenarioCheck(t, reg, clock)

	reg.RunAll(context.Background(), true)
	summary := reg.GetSummary()
	if summary.SuspendedCount != 1 {
		t.Fatalf("summary suspended = %d, want 1", summary.SuspendedCount)
	}
	result, _ := reg.GetResult("resource-qdrant")
	if suspended, _ := result.Details["autoHealSuspended"].(bool); !suspended || !strings.Contains(result.Message, "suspended") {
		t.Fatalf("suspended check carries no signal: %q %v", result.Message, result.Details)
	}

	check.result = Result{CheckID: "resource-qdrant", Status: StatusOK}
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	if tracker, _ := reg.GetHealTracker("resource-qdrant"); tracker.IsSuspended() {
		t.Fatalf("suspension survived a healthy result: %+v", tracker)
	}
	reg.RunAll(context.Background(), true)
	if got := reg.GetSummary().SuspendedCount; got != 0 {
		t.Fatalf("summary suspended after recovery = %d, want 0", got)
	}
}

func TestSuspensionOlderThanBoundIsReprobedOnceThenResuspended(t *testing.T) {
	reg, clock, check, _ := suspendableRegistry(t, StatusCritical)
	suspendNonScenarioCheck(t, reg, clock)
	before := len(check.executedActions)

	clock.Advance(SuspensionMaxAge - 2*time.Hour)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	if len(check.executedActions) != before {
		t.Fatalf("suspension inside the bound was re-probed")
	}

	suspendedAt := time.Time{}
	if tracker, _ := reg.GetHealTracker("resource-qdrant"); tracker != nil {
		suspendedAt = tracker.SuspendedAt
	}
	clock.Advance(3 * time.Hour)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	if len(check.executedActions) != before+1 {
		t.Fatalf("stale suspension was not re-probed: %d actions, want %d", len(check.executedActions), before+1)
	}
	tracker, _ := reg.GetHealTracker("resource-qdrant")
	if !tracker.IsSuspended() || !tracker.SuspendedAt.After(suspendedAt) {
		t.Fatalf("failed re-probe did not re-suspend with a fresh timestamp: %+v", tracker)
	}
}

func heldWarningRegistry(t *testing.T, gate *stubGate) (*Registry, *fixedClock, *mockHealableCheck, *healIncidentReporterSpy) {
	t.Helper()
	reporter := &healIncidentReporterSpy{}
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(1000, 0)}
	reg.SetClock(clock)
	reg.SetHealIncidentReporter(reporter)
	reg.SetRecoveryOwnershipGate(gate)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-demo": true},
		autoHealOn:     map[string]string{"scenario-demo": AutoHealOnWarningCritical},
	})
	check := &mockHealableCheck{
		id:            "scenario-demo",
		result:        Result{CheckID: "scenario-demo", Status: StatusWarning, Message: "demo degraded"},
		actions:       []RecoveryAction{{ID: "restart", Available: true, Dangerous: true}},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(check)
	return reg, clock, check, reporter
}

func TestGateHoldIsVisibleCriticalAfterBoundAndOverriddenWhenAllowed(t *testing.T) {
	gate := &stubGate{verdict: GateVerdict{Class: GateClassEpoch, Reason: "epoch owns recovery",
		CriticalAfter: 10 * time.Minute, OverrideAfter: 15 * time.Minute}}
	reg, clock, check, reporter := heldWarningRegistry(t, gate)
	tick := func() []AutoHealResult {
		return reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	}

	tick()
	tick()
	result, _ := reg.GetResult("scenario-demo")
	if held, _ := result.Details["healHeld"].(bool); !held || result.Status != StatusWarning {
		t.Fatalf("held check = %s %v, want warning carrying healHeld", result.Status, result.Details)
	}
	if reg.GetSummary().HealHeldCount != 1 || len(check.executedActions) != 0 {
		t.Fatalf("held count=%d actions=%v", reg.GetSummary().HealHeldCount, check.executedActions)
	}

	clock.Advance(11 * time.Minute)
	tick()
	tick()
	result, _ = reg.GetResult("scenario-demo")
	if result.Status != StatusCritical || len(reporter.opened) != 1 || reporter.opened[0].actionID != holdIncidentAction {
		t.Fatalf("hold past its bound: status=%s incidents=%+v, want critical plus one incident", result.Status, reporter.opened)
	}
	if len(check.executedActions) != 0 {
		t.Fatalf("restart ran before the override bound")
	}

	clock.Advance(5 * time.Minute)
	results := tick()
	if len(check.executedActions) != 1 {
		t.Fatalf("restart did not proceed after the override bound: %+v", results)
	}

	check.result = Result{CheckID: "scenario-demo", Status: StatusOK}
	tick()
	if got := strings.Join(reporter.resolved, ","); !strings.Contains(got, "scenario-demo:"+holdIncidentAction) {
		t.Fatalf("hold incident not resolved on recovery: %v", reporter.resolved)
	}
	if reg.GetSummary().HealHeldCount != 0 {
		t.Fatalf("hold still counted after recovery")
	}
}

func TestOwnerMaintenanceHoldIsCriticalButNeverOverridden(t *testing.T) {
	gate := &stubGate{verdict: GateVerdict{Class: GateClassMaintenance, Reason: "owner maintenance is closed", CriticalAfter: 10 * time.Minute}}
	reg, clock, check, reporter := heldWarningRegistry(t, gate)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	clock.Advance(24 * time.Hour)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	result, _ := reg.GetResult("scenario-demo")
	if result.Status != StatusCritical || len(check.executedActions) != 0 || len(reporter.opened) != 1 {
		t.Fatalf("status=%s actions=%v incidents=%d", result.Status, check.executedActions, len(reporter.opened))
	}
}

func TestSchemaNewerThanBinaryIsCriticalNamingTheFixThenOverridden(t *testing.T) {
	home := t.TempDir()
	dbPath, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	testenv.StampSQLiteUserVersion(t, dbPath, scenarioruntime.SchemaVersion+1)
	gate := RuntimeRecoveryGate{HomeDir: home}
	verdict := gate.AllowsAutoHealRestart(context.Background(), "scenario-demo", "restart")
	if verdict.Allowed || verdict.Class != GateClassSchema || verdict.CriticalAfter != 0 || verdict.OverrideAfter <= 0 {
		t.Fatalf("verdict = %+v, want immediate critical with a bounded override", verdict)
	}
	if !strings.Contains(verdict.Reason, "vrooli scenario restart vrooli-autoheal") {
		t.Fatalf("reason does not name the fix: %q", verdict.Reason)
	}

	reg, clock, check, _ := heldWarningRegistry(t, &stubGate{})
	reg.SetRecoveryOwnershipGate(gate)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	reg.RunAll(context.Background(), true)
	result, _ := reg.GetResult("scenario-demo")
	if result.Status != StatusCritical || len(check.executedActions) != 0 {
		t.Fatalf("schema hold: status=%s actions=%v, want critical and deferred", result.Status, check.executedActions)
	}
	clock.Advance(gateSchemaOverride + time.Second)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	if len(check.executedActions) != 1 {
		t.Fatalf("restart stayed blocked by an unreadable-by-this-binary registry")
	}
}

// A wedged owner cannot answer its own admission fence; that must be a short,
// visible deferral that ends in a restart, never a standing denial (R6).
func TestHungOwnerAdmissionIsBoundedUnreadableHoldNotStandingDenial(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-release }))
	defer server.Close()
	defer close(release)
	port := server.Listener.Addr().(*net.TCPAddr).Port

	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	instance, err := store.CreateLease(context.Background(), scenarioruntime.Instance{
		Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant, Status: scenarioruntime.StatusRunning,
	}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.AcquirePortClaim(context.Background(), scenarioruntime.PortClaim{
		InstanceID: instance.InstanceID, Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant,
		PortName: "API_PORT", EnvVar: "API_PORT", Port: port,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindPortClaim(context.Background(), claim.ClaimID); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()

	gate := RuntimeRecoveryGate{HomeDir: home, CoordinatedScenario: "agent-manager"}
	verdict := gate.AllowsAutoHealRestart(context.Background(), "scenario-agent-manager", "restart")
	if verdict.Allowed || verdict.Class != GateClassUnreadable || verdict.OverrideAfter <= 0 || verdict.OverrideAfter > 5*time.Minute {
		t.Fatalf("hung owner verdict = %+v, want bounded unreadable hold", verdict)
	}

	reg, clock, check, _ := heldWarningRegistry(t, &stubGate{})
	reg.SetRecoveryOwnershipGate(&stubGate{verdict: verdict})
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	if len(check.executedActions) != 0 {
		t.Fatalf("restart ran inside the hold")
	}
	clock.Advance(verdict.OverrideAfter + time.Second)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	if len(check.executedActions) != 1 {
		t.Fatalf("hung owner still blocks its own repair after %s", verdict.OverrideAfter)
	}
}

func TestRecoveryGateCheckIsCriticalNamingTheFixWhenRegistryIsNewer(t *testing.T) {
	home := t.TempDir()
	dbPath, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	testenv.StampSQLiteUserVersion(t, dbPath, scenarioruntime.SchemaVersion+1)
	result := RecoveryGateCheck{HomeDir: home}.Run(context.Background())
	if result.Status != StatusCritical || !strings.Contains(result.Message, "vrooli scenario restart vrooli-autoheal") {
		t.Fatalf("result = %s %q, want critical naming the fix", result.Status, result.Message)
	}
	if ok := (RecoveryGateCheck{HomeDir: t.TempDir()}).Run(context.Background()); ok.Status != StatusOK {
		t.Fatalf("fresh registry = %s %q, want ok", ok.Status, ok.Message)
	}
}

// One failed probe never justifies restarting a scenario: a single timeout
// right after autoheal's own start restarted a healthy event bus into a stuck
// restart. Two consecutive failures spanning 30 s do.
func TestScenarioRestartNeedsConsecutiveFailedProbesSpacedApart(t *testing.T) {
	reg := newTestRegistry()
	reg.SetFailureConfirmation(ConfirmProbes, ConfirmSpan)
	clock := &fixedClock{current: time.Unix(5000, 0)}
	reg.SetClock(clock)
	check := &mockHealableCheck{
		id:            "scenario-demo",
		result:        Result{CheckID: "scenario-demo", Status: StatusCritical, Message: "health timed out"},
		actions:       []RecoveryAction{{ID: "restart", Available: true, Dangerous: true}},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{"scenario-demo": true}})
	probe := func(status Status) {
		check.result = Result{CheckID: "scenario-demo", Status: status, Message: "probe"}
		reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	}

	probe(StatusCritical) // the lone timeout
	if len(check.executedActions) != 0 {
		t.Fatal("restarted on a single failed probe")
	}
	clock.Advance(10 * time.Second)
	probe(StatusCritical) // second failure, too close to the first
	if len(check.executedActions) != 0 {
		t.Fatal("restarted on two probes less than 30s apart")
	}
	clock.Advance(time.Minute)
	probe(StatusOK) // a healthy probe resets the streak
	clock.Advance(time.Minute)
	probe(StatusCritical)
	if len(check.executedActions) != 0 {
		t.Fatal("a healthy probe in between did not reset the streak")
	}
	clock.Advance(40 * time.Second)
	probe(StatusCritical)
	if len(check.executedActions) != 1 {
		t.Fatalf("confirmed failure was not repaired: %v", check.executedActions)
	}
}

// A check whose severity never triggers a heal (a plain warning) must still
// lose a stale suspension; otherwise it stays suspended forever.
func TestStaleSuspensionLiftsEvenWhenSeverityNeverTriggersHeal(t *testing.T) {
	reg, clock, check, _ := suspendableRegistry(t, StatusCritical)
	suspendNonScenarioCheck(t, reg, clock)
	check.result = Result{CheckID: "resource-qdrant", Status: StatusWarning, Message: "degraded"}
	clock.Advance(SuspensionMaxAge + time.Hour)
	reg.RunAutoHeal(context.Background(), reg.RunAll(context.Background(), true))
	if tracker, _ := reg.GetHealTracker("resource-qdrant"); tracker.IsSuspended() {
		t.Fatalf("stale suspension survived on a non-triggering warning: %+v", tracker)
	}
}
