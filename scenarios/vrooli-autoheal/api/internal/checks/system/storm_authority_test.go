package system

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

const (
	agentScopePath      = "/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/vrooli-agent-abc.scope"
	supervisorScopePath = "/user.slice/user-1000.slice/user@1000.service/app.slice/vrooli-runtime-supervisor.service"
)

type fakeStorm struct {
	frozen    map[string]bool
	decisions []StormDecision
	gateOpen  bool
	gateErr   string
	freezeErr error
}

func newFakeStorm(mode string) (*StormAuthority, *fakeStorm) {
	fake := &fakeStorm{frozen: map[string]bool{}, gateOpen: true}
	clock := time.Date(2026, 9, 2, 10, 14, 0, 0, time.UTC)
	authority := &StormAuthority{
		Mode: mode,
		Gate: func(context.Context) (string, bool, string) {
			if !fake.gateOpen {
				return "epoch-1", false, fake.gateErr
			}
			return "epoch-1", true, ""
		},
		Record: func(_ context.Context, decision StormDecision) error {
			fake.decisions = append(fake.decisions, decision)
			return nil
		},
		Freeze: func(ref platformgo.ScopeRef) error {
			if fake.freezeErr != nil {
				return fake.freezeErr
			}
			if _, ok := AgentScopeName(ref.Path); !ok {
				return errors.New("platform: refusing to freeze " + ref.Path)
			}
			fake.frozen[ref.Path] = true
			return nil
		},
		Thaw:       func(ref platformgo.ScopeRef) error { delete(fake.frozen, ref.Path); return nil },
		Frozen:     func(ref platformgo.ScopeRef) (bool, error) { return fake.frozen[ref.Path], nil },
		WorkingDir: func(int) (string, error) { return "/home/op/Vrooli", nil },
		Now:        func() time.Time { return clock },
	}
	return authority, fake
}

func writeStormReport(t *testing.T, findings []string, scope string, at time.Time) string {
	t.Helper()
	report := map[string]any{
		"captured_at": at,
		"findings":    findings,
		"evidence":    map[string][]string{"fork-rate": {"/proc/stat"}},
		"attribution": map[string]any{
			"state": "read", "reason": "",
			"by_children": []map[string]any{{"pid": 4242, "name": "claude", "children": 300, "delta": 280, "scope": scope}},
			"by_delta":    []map[string]any{{"pid": 4242, "name": "claude", "children": 300, "delta": 280, "scope": scope}},
		},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "last-report.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func stormCheck(t *testing.T, authority *StormAuthority, findings []string, scope string) (*EmergencyWatchdogReportCheck, checks.Result) {
	t.Helper()
	now := time.Date(2026, 9, 2, 10, 15, 0, 0, time.UTC)
	path := writeStormReport(t, findings, scope, now.Add(-time.Minute))
	check := NewEmergencyWatchdogReportCheck(WithReportPath(path), WithReportClock(func() time.Time { return now }), WithStormAuthority(authority))
	return check, check.Run(context.Background())
}

// [REQ:STORM-002] The action targets only a scope under vrooli-agents.slice.
func TestContainStormRefusesPidOutsideAgentSlice(t *testing.T) {
	authority, fake := newFakeStorm(ContainStormAutomatic)
	check, result := stormCheck(t, authority, []string{"fork-rate: 2481.0 forks/s exceeds SB16 bar"}, supervisorScopePath)
	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s", result.Status)
	}
	actions := check.RecoveryActions(&result)
	if len(actions) != 1 || actions[0].Available {
		t.Fatalf("a supervisor scope must not make contain-storm available: %+v", actions)
	}
	outcome := check.ExecuteAction(context.Background(), ContainStormActionID)
	if outcome.Success || !strings.Contains(outcome.Error, "no sustained finding names an agent session scope") {
		t.Fatalf("execute = %+v", outcome)
	}
	// Even a direct call with a forged target refuses on the path.
	_, err := authority.Contain(context.Background(), StormTarget{Finding: "fork-rate", PID: 1, Name: "vrooli-runtime-supervisor", ScopePath: supervisorScopePath})
	if err == nil || !strings.Contains(err.Error(), "not an agent session scope") || len(fake.frozen) != 0 {
		t.Fatalf("supervisor was not refused: err=%v frozen=%v", err, fake.frozen)
	}
}

// [REQ:STORM-002] A closed gate (unreadable recovery coordination) refuses.
func TestContainStormRefusesWhenGateClosed(t *testing.T) {
	authority, fake := newFakeStorm(ContainStormAutomatic)
	fake.gateOpen, fake.gateErr = false, "runtime recovery ownership unavailable: registry locked"
	check, _ := stormCheck(t, authority, []string{"fork-rate: 2481.0 forks/s exceeds SB16 bar"}, agentScopePath)
	outcome := check.ExecuteAction(context.Background(), ContainStormActionID)
	if outcome.Success || !strings.Contains(outcome.Error, "registry locked") || len(fake.frozen) != 0 {
		t.Fatalf("closed gate did not refuse: %+v frozen=%v", outcome, fake.frozen)
	}
	if len(fake.decisions) != 1 || fake.decisions[0].State != StormDecisionRefused {
		t.Fatalf("refusal must leave a decision row: %+v", fake.decisions)
	}
}

// [REQ:STORM-002] A sustained agent-scope finding is frozen, recorded with the
// idempotency key <epoch>/<scope>/contain-storm, and the next run names the
// thaw command on the finding.
func TestContainStormFreezesScopeAndRecordsDecision(t *testing.T) {
	authority, fake := newFakeStorm(ContainStormAutomatic)
	check, result := stormCheck(t, authority, []string{"fork-rate: 2481.0 forks/s exceeds SB16 bar"}, agentScopePath)
	actions := check.RecoveryActions(&result)
	if len(actions) != 1 || !actions[0].Available || actions[0].Dangerous {
		t.Fatalf("automatic mode must offer a safe, available action: %+v", actions)
	}
	outcome := check.ExecuteAction(context.Background(), ContainStormActionID)
	if !outcome.Success || !fake.frozen[agentScopePath] {
		t.Fatalf("freeze did not happen: %+v frozen=%v", outcome, fake.frozen)
	}
	if len(fake.decisions) != 1 || fake.decisions[0].IdempotencyKey != "epoch-1/vrooli-agent-abc.scope/contain-storm" || fake.decisions[0].State != StormDecisionContained {
		t.Fatalf("decision = %+v", fake.decisions)
	}
	if fake.decisions[0].Target.WorkingDir != "/home/op/Vrooli" || !strings.Contains(outcome.Message, "vrooli agent thaw vrooli-agent-abc") {
		t.Fatalf("decision lacks the tree or the thaw command: %+v / %s", fake.decisions[0].Target, outcome.Message)
	}
	again := check.Run(context.Background())
	findings := findingsFromDetails(again.Details)
	if len(findings) != 1 || findings[0]["contained"] != true || !strings.Contains(findings[0]["reason"].(string), "vrooli agent thaw vrooli-agent-abc") {
		t.Fatalf("next run did not annotate containment: %+v", findings)
	}
	if actions := check.RecoveryActions(&again); actions[0].Available {
		t.Fatal("a frozen scope must not be offered again")
	}
}

// [REQ:STORM-002] Attribution without a sustained finding is not a target: the
// watchdog lists a finding only after the authored sustain.
func TestContainStormRequiresSustainedFinding(t *testing.T) {
	authority, fake := newFakeStorm(ContainStormAutomatic)
	check, result := stormCheck(t, authority, nil, agentScopePath)
	if result.Status != checks.StatusOK {
		t.Fatalf("status = %s", result.Status)
	}
	if actions := check.RecoveryActions(&result); actions[0].Available {
		t.Fatalf("no finding, yet the action is available: %+v", actions)
	}
	if outcome := check.ExecuteAction(context.Background(), ContainStormActionID); outcome.Success || len(fake.frozen) != 0 {
		t.Fatalf("froze without a sustained finding: %+v", outcome)
	}
}

// [REQ:STORM-002] Thawing reverses the freeze and the next run drops the
// containment so the incident can resolve with the finding.
func TestThawReversesAndResolvesIncident(t *testing.T) {
	authority, fake := newFakeStorm(ContainStormAutomatic)
	check, _ := stormCheck(t, authority, []string{"fork-rate: 2481.0 forks/s exceeds SB16 bar"}, agentScopePath)
	if outcome := check.ExecuteAction(context.Background(), ContainStormActionID); !outcome.Success {
		t.Fatal(outcome.Error)
	}
	if err := authority.Thaw(ScopeRefForPath(agentScopePath)); err != nil {
		t.Fatal(err)
	}
	result := check.Run(context.Background())
	if _, ok := result.Details["containment"]; ok || len(check.ContainedScopes()) != 0 {
		t.Fatalf("thawed scope still reported as contained: %v", result.Details["containment"])
	}
	if actions := check.RecoveryActions(&result); !actions[0].Available {
		t.Fatal("after a thaw a still-sustained finding must be containable again")
	}
	// A scope that exited (its cgroup is gone) is not contained either.
	if outcome := check.ExecuteAction(context.Background(), ContainStormActionID); !outcome.Success {
		t.Fatal(outcome.Error)
	}
	authority.Frozen = func(platformgo.ScopeRef) (bool, error) {
		return false, errors.New("open cgroup.freeze: no such file or directory")
	}
	if result := check.Run(context.Background()); result.Details["containment"] != nil || len(check.ContainedScopes()) != 0 {
		t.Fatalf("a vanished scope still reported as contained: %v", result.Details["containment"])
	}
	_ = fake
}

// [REQ:STORM-002] propose_only keeps the action for the operator: Dangerous,
// so the registry's auto-heal selection never picks it.
func TestProposeOnlyModeNeverFreezes(t *testing.T) {
	authority, fake := newFakeStorm(ContainStormProposeOnly)
	check, result := stormCheck(t, authority, []string{"fork-rate: 2481.0 forks/s exceeds SB16 bar"}, agentScopePath)
	actions := check.RecoveryActions(&result)
	if len(actions) != 1 || !actions[0].Available || !actions[0].Dangerous {
		t.Fatalf("propose_only must offer a Dangerous action: %+v", actions)
	}
	registry := checks.NewRegistry(&platform.Capabilities{Platform: platform.Linux})
	registry.Register(check)
	healed := registry.RunAutoHeal(context.Background(), []checks.Result{result})
	if len(fake.frozen) != 0 {
		t.Fatalf("auto-heal froze in propose_only mode: %+v", healed)
	}
	if outcome := check.ExecuteAction(context.Background(), ContainStormActionID); !outcome.Success || !fake.frozen[agentScopePath] {
		t.Fatalf("operator execution must still freeze: %+v", outcome)
	}
}

func TestAgentScopeNameAcceptsOnlyAgentScopes(t *testing.T) {
	if name, ok := AgentScopeName(agentScopePath); !ok || name != "vrooli-agent-abc.scope" {
		t.Fatalf("agent scope = %q %v", name, ok)
	}
	for _, path := range []string{supervisorScopePath, "/vrooli-agents.slice", "/user.slice/vrooli-agents.slice/vrooli-agent-x", "", "/sys/fs/cgroup/../vrooli-agents.slice/vrooli-agent-y.scope"} {
		if _, ok := AgentScopeName(path); ok && !strings.Contains(path, "/vrooli-agents.slice/vrooli-agent-y.scope") {
			t.Fatalf("%q accepted", path)
		}
	}
}

// A storm decision is a runtime_recovery_decisions row the supervisor and the
// CLI read from the same registry; the idempotency key makes a repeat one row.
func TestStormDecisionRowRoundTrips(t *testing.T) {
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: filepath.Join(t.TempDir(), "runtime.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	epochID, err := EnsureStormEpoch(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := EnsureStormEpoch(ctx, store); err != nil || again != epochID {
		t.Fatalf("anchor epoch not reused: %q %q %v", epochID, again, err)
	}
	epochs, _ := store.ListPressureEpochs(ctx, 1)
	if len(epochs) != 1 || epochs[0].Status != scenarioruntime.PressureEpochCleared {
		t.Fatalf("anchor epoch must be cleared so it gates nothing: %+v", epochs)
	}
	decision := StormDecision{EpochID: epochID, State: StormDecisionContained, Reason: "froze x", IdempotencyKey: epochID + "/vrooli-agent-x.scope/contain-storm", Target: StormTarget{ScopeName: "vrooli-agent-x.scope", PID: 7}, At: time.Now()}
	for i := 0; i < 2; i++ {
		if err := RecordStormDecision(ctx, store, decision); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.ListRecoveryDecisions(ctx, scenarioruntime.RecoveryDecisionFilter{Scenario: stormDecisionScenario, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].State != StormDecisionContained || !strings.Contains(rows[0].DetailsJSON, "vrooli-agent-x.scope") {
		t.Fatalf("rows = %+v", rows)
	}
}
