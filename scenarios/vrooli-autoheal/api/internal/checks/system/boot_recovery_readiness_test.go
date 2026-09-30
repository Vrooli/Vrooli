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

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// readinessFixture is the `vrooli setup status --json --phase readiness`
// output captured on the development host on 2026-09-02, saved with the
// phase-10 evidence. The test reads the copy kept next to the code so it does
// not depend on an operator directory.
func readinessFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "setup-status-readiness.json"))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return data
}

const (
	loopSHA      = "8756c44057b5e55994f6699ce994662defd32366fb97c582e8360086e4f43c29"
	fixtureNowS  = "2026-09-02T18:12:00Z"
	preflightOut = `{"at":"2026-09-02T18:11:00Z","ok":true,"checks":[{"name":"cli-resolves","status":"ok"},{"name":"toolchain","status":"ok"}]}`
)

// healthyProbes reproduces the host as captured: every precondition holds.
func healthyProbes(t *testing.T) BootRecoveryProbes {
	t.Helper()
	now, _ := time.Parse(time.RFC3339, fixtureNowS)
	tick := now.Add(-50 * time.Second)
	loopStatus, _ := json.Marshal(map[string]any{"last_tick_at": tick, "state": "healthy", "binary_sha256": loopSHA, "pid": 116451})
	fixture := readinessFixture(t)
	return BootRecoveryProbes{
		SetupStatus:  func(context.Context) ([]byte, error) { return fixture, nil },
		LoopSelfTest: func(context.Context) ([]byte, int, error) { return []byte(preflightOut), 0, nil },
		UnitState: func(_ context.Context, _ string) (UnitState, error) {
			return UnitState{ActiveState: "active", NRestarts: "0", Result: "success", RestartsKnown: true}, nil
		},
		LoopStatus:       func() ([]byte, error) { return loopStatus, nil },
		LoopBinarySHA256: func() (string, error) { return loopSHA, nil },
		Lingering:        func(context.Context, string) (bool, error) { return true, nil },
		Username:         func() (string, error) { return "tester", nil },
		AgentScopes:      func(context.Context) (int, error) { return 2, nil },
		GOOS:             "linux",
		Now:              func() time.Time { return now },
	}
}

func preconditionState(t *testing.T, r checks.Result, name string) (string, string) {
	t.Helper()
	list, ok := r.Details["preconditions"].([]map[string]any)
	if !ok {
		t.Fatalf("preconditions = %#v", r.Details["preconditions"])
	}
	for _, p := range list {
		if p["name"] == name {
			return p["state"].(string), p["reason"].(string)
		}
	}
	t.Fatalf("precondition %q not reported in %v", name, list)
	return "", ""
}

// [REQ:BOOT-RECOVERY-001] The healthy host lists every precondition by name
// and reports ok.
func TestReadinessIsOkWithSevenNamedPreconditions(t *testing.T) {
	r := NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(healthyProbes(t))).Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Fatalf("status = %s: %s", r.Status, r.Message)
	}
	for _, name := range []string{PreconditionSafeguards, PreconditionLoopPreflight, PreconditionUnitActive, PreconditionLoopHeartbeat, PreconditionLingering, PreconditionValidator, PreconditionContainment} {
		if state, reason := preconditionState(t, r, name); state != PreconditionOK || reason == "" {
			t.Errorf("%s = %s (%q), want ok with a reason", name, state, reason)
		}
	}
	if got := len(r.Details["preconditions"].([]map[string]any)); got != 7 {
		t.Errorf("preconditions = %d, want 7", got)
	}
	if r.Details["remediation"] != "vrooli setup" {
		t.Errorf("remediation = %v", r.Details["remediation"])
	}
}

// [REQ:BOOT-RECOVERY-001] One failed precondition is critical and the
// message names it and the repair; the rest still report.
func TestReadinessIsCriticalWhenAnyPreconditionFails(t *testing.T) {
	cases := map[string]func(*BootRecoveryProbes){
		PreconditionUnitActive: func(p *BootRecoveryProbes) {
			p.UnitState = func(_ context.Context, unit string) (UnitState, error) {
				if unit == "vrooli-runtime-supervisor.service" {
					return UnitState{ActiveState: "active", NRestarts: "495", Result: "start-limit-hit", RestartsKnown: true}, nil
				}
				return UnitState{ActiveState: "active", NRestarts: "0", Result: "success", RestartsKnown: true}, nil
			}
		},
		PreconditionLoopPreflight: func(p *BootRecoveryProbes) {
			p.LoopSelfTest = func(context.Context) ([]byte, int, error) {
				return []byte(`{"ok":false,"checks":[{"name":"toolchain","status":"failed","reason":"go not found"}]}`), 3, nil
			}
		},
		PreconditionLoopHeartbeat: func(p *BootRecoveryProbes) {
			old := p.LoopBinarySHA256
			p.LoopBinarySHA256 = func() (string, error) { s, _ := old(); return "deadbeef" + s[8:], nil }
		},
		PreconditionLingering: func(p *BootRecoveryProbes) {
			p.Lingering = func(context.Context, string) (bool, error) { return false, nil }
		},
		PreconditionSafeguards: func(p *BootRecoveryProbes) {
			p.SetupStatus = func(context.Context) ([]byte, error) {
				return []byte(`{"version":"1","safeguards":[{"name":"autoheal_watchdog","applied":false,"execution_state":"pending","notes":["autoheal loop binary needs building"],"config":{"boot_policy":"dedicated"}},{"name":"runtime_supervisor","applied":true},{"name":"emergency_watchdog","applied":true}]}`), nil
			}
		},
		PreconditionValidator: func(p *BootRecoveryProbes) {
			p.SetupStatus = func(context.Context) ([]byte, error) {
				return []byte(`{"version":"1","safeguards":[{"name":"autoheal_watchdog","applied":true,"config":{"boot_policy":"dedicated"},"evidence":{"validator_verdict":{"state":"rejected","validator":"systemd-analyze verify","output":"Unknown key name 'Restat'"}}},{"name":"runtime_supervisor","applied":true,"evidence":{"validator_verdict":{"state":"accepted"}}},{"name":"emergency_watchdog","applied":true,"evidence":{"validator_verdict":{"state":"accepted"}}}]}`), nil
			}
		},
	}
	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			probes := healthyProbes(t)
			breakIt(&probes)
			r := NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(probes)).Run(context.Background())
			if r.Status != checks.StatusCritical {
				t.Fatalf("status = %s, want critical: %s", r.Status, r.Message)
			}
			if state, _ := preconditionState(t, r, name); state != PreconditionFailed {
				t.Errorf("%s state = %s, want failed", name, state)
			}
			if !strings.Contains(r.Message, name) || !strings.Contains(r.Message, "vrooli setup") {
				t.Errorf("message must name the precondition and the repair: %q", r.Message)
			}
			if r.Details["findingKey"] != name {
				t.Errorf("findingKey = %v, want %s", r.Details["findingKey"], name)
			}
		})
	}
}

// [REQ:BOOT-RECOVERY-001] A probe that cannot run is undetermined, never ok:
// "we could not ask" must not read as "the boot path works".
func TestReadinessIsUndeterminedNotOkWhenProbeCannotRun(t *testing.T) {
	probes := healthyProbes(t)
	probes.SetupStatus = func(context.Context) ([]byte, error) { return nil, errors.New("vrooli binary not found") }
	r := NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(probes)).Run(context.Background())
	if r.Status != checks.StatusUndetermined {
		t.Fatalf("status = %s, want undetermined: %s", r.Status, r.Message)
	}
	for _, name := range []string{PreconditionSafeguards, PreconditionLingering, PreconditionValidator} {
		if state, _ := preconditionState(t, r, name); state != PreconditionUndetermined {
			t.Errorf("%s = %s, want undetermined when setup status cannot be read", name, state)
		}
	}

	// A validator that could not run is unproven, which is undetermined.
	probes = healthyProbes(t)
	probes.SetupStatus = func(context.Context) ([]byte, error) {
		return []byte(`{"version":"1","safeguards":[{"name":"autoheal_watchdog","applied":true,"config":{"boot_policy":"shared"},"evidence":{"validator_verdict":{"state":"unavailable","validator":"systemd-analyze verify","output":"not on PATH"}}},{"name":"runtime_supervisor","applied":true,"evidence":{"validator_verdict":{"state":"accepted"}}},{"name":"emergency_watchdog","applied":true,"evidence":{"validator_verdict":{"state":"accepted"}}}]}`), nil
	}
	r = NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(probes)).Run(context.Background())
	if r.Status != checks.StatusUndetermined {
		t.Fatalf("status = %s, want undetermined for an unavailable validator: %s", r.Status, r.Message)
	}
	if state, _ := preconditionState(t, r, PreconditionLingering); state != PreconditionOK {
		t.Errorf("lingering under a shared policy = %s, want ok (not required)", state)
	}

	// A failed precondition outranks an undetermined one.
	probes = healthyProbes(t)
	probes.LoopStatus = func() ([]byte, error) { return nil, os.ErrNotExist }
	probes.Lingering = func(context.Context, string) (bool, error) { return false, nil }
	r = NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(probes)).Run(context.Background())
	if r.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical when one precondition failed and another is undetermined", r.Status)
	}
}

// [REQ:BOOT-RECOVERY-001] The check can repair nothing, so it offers nothing.
func TestReadinessNeverOffersActions(t *testing.T) {
	var check checks.Check = NewBootRecoveryReadinessCheck()
	if _, ok := check.(checks.HealableCheck); ok {
		t.Fatal("system-boot-recovery-readiness must not implement HealableCheck")
	}
	if check.IntervalSeconds() != 3600 || check.Category() != checks.CategorySystem || check.ID() != "system-boot-recovery-readiness" {
		t.Fatalf("identity = %s/%s/%d", check.ID(), check.Category(), check.IntervalSeconds())
	}
}

// The heartbeat window and unit restarts are read from the sources the loop
// and systemd actually write, not from prose.
func TestReadinessReadsHeartbeatAndRestartsFromSources(t *testing.T) {
	probes := healthyProbes(t)
	now := probes.Now()
	stale, _ := json.Marshal(map[string]any{"last_tick_at": now.Add(-10 * time.Minute), "state": "healthy", "binary_sha256": loopSHA})
	probes.LoopStatus = func() ([]byte, error) { return stale, nil }
	r := NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(probes)).Run(context.Background())
	if state, reason := preconditionState(t, r, PreconditionLoopHeartbeat); state != PreconditionFailed || !strings.Contains(reason, "10m0s") {
		t.Errorf("stale heartbeat = %s (%q)", state, reason)
	}
}

// [REQ:STORM-002] The containment precondition reads the agent slice's
// readiness item: an inspection the safeguard could not run is undetermined,
// a slice that is not applied is failed, and the ok reason counts live scopes.
func TestReadinessContainmentPreconditionUndeterminedWithoutDelegation(t *testing.T) {
	rewrite := func(t *testing.T, mutate func(map[string]any)) []byte {
		t.Helper()
		var report map[string]any
		if err := json.Unmarshal(readinessFixture(t), &report); err != nil {
			t.Fatal(err)
		}
		for _, raw := range report["safeguards"].([]any) {
			item := raw.(map[string]any)
			if item["name"] == "agent_session_containment" {
				mutate(item)
			}
		}
		data, _ := json.Marshal(report)
		return data
	}
	probes := healthyProbes(t)
	undetermined := rewrite(t, func(item map[string]any) {
		item["applied"] = false
		item["evidence"] = map[string]any{"probe": "undetermined"}
		item["notes"] = []any{"undetermined: systemctl --user show vrooli-agents.slice: Failed to connect to bus"}
	})
	probes.SetupStatus = func(context.Context) ([]byte, error) { return undetermined, nil }
	r := NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(probes)).Run(context.Background())
	if state, reason := preconditionState(t, r, PreconditionContainment); state != PreconditionUndetermined || !strings.Contains(reason, "could not inspect") {
		t.Fatalf("containment = %s (%q), want undetermined", state, reason)
	}
	if r.Status != checks.StatusUndetermined {
		t.Fatalf("status = %s, want undetermined, not ok", r.Status)
	}

	notApplied := rewrite(t, func(item map[string]any) { item["applied"] = false })
	probes.SetupStatus = func(context.Context) ([]byte, error) { return notApplied, nil }
	r = NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(probes)).Run(context.Background())
	if state, _ := preconditionState(t, r, PreconditionContainment); state != PreconditionFailed || r.Status != checks.StatusCritical {
		t.Fatalf("not applied: containment = %s status = %s", state, r.Status)
	}

	healthy := healthyProbes(t)
	r = NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(healthy)).Run(context.Background())
	if state, reason := preconditionState(t, r, PreconditionContainment); state != PreconditionOK || !strings.Contains(reason, "2 live agent scope") {
		t.Fatalf("healthy containment = %s (%q)", state, reason)
	}
	healthy.AgentScopes = func(context.Context) (int, error) { return 0, errors.New("Failed to connect to bus") }
	r = NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(healthy)).Run(context.Background())
	if state, _ := preconditionState(t, r, PreconditionContainment); state != PreconditionUndetermined {
		t.Fatalf("uncountable scopes = %s, want undetermined", state)
	}
}
