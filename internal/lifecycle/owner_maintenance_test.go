package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/maintenance"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestOwnerMaintenanceStoppedBootstrapRequiresPhysicalExclusion(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown_%t", unknown), func(t *testing.T) {
			root, home := t.TempDir(), t.TempDir()
			writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "agent-manager"}, Lifecycle: scenario.Lifecycle{Setup: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "setup", Exec: []string{"bash", "-c", "printf setup > setup.txt"}}}}}})
			store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: home})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			old, err := store.CreateInstance(t.Context(), scenarioruntime.Instance{Scenario: "agent-manager", Status: scenarioruntime.StatusStarting})
			if err != nil {
				t.Fatal(err)
			}
			var runner *Runner
			proofs := 0
			runner = newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
				deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
					return ownerMaintenanceStanding{}, errors.New("owner offline")
				}
				deps.requireRuntimeAbsent = func(_ context.Context, ref maintenance.RuntimeScopeRef) error {
					proofs++
					if len(ref.InstanceIDs) != 1 || ref.InstanceIDs[0] != old.InstanceID {
						t.Fatalf("wrong bootstrap scope: %+v", ref)
					}
					if release, err := runner.tryAcquireScenarioLock("agent-manager"); err == nil {
						release()
						t.Error("bootstrap proof must hold lifecycle lock")
					}
					if unknown {
						return errors.New("managed process scope unknown")
					}
					return nil
				}
				deps.signalPID = func(int, bool) error { t.Fatal("bootstrap must not signal an executor"); return nil }
				deps.signalProcessGroup = func(int, bool) error { t.Fatal("bootstrap must not signal a group"); return nil }
			})
			result, err := runner.Start("agent-manager", StartOptions{ForceSetup: true})
			if unknown {
				if !errors.Is(err, ErrOwnerMaintenanceRequired) {
					t.Fatalf("unknown bootstrap admitted: %v", err)
				}
				if _, err := os.Stat(filepath.Join(root, "scenarios/agent-manager/setup.txt")); !os.IsNotExist(err) {
					t.Fatalf("unknown bootstrap setup effect: %v", err)
				}
			} else if err != nil || result.Scenario.Slug != "agent-manager" {
				t.Fatalf("positively stopped owner cannot bootstrap: %+v %v", result, err)
			}
			if proofs != 1 {
				t.Fatalf("physical exclusion reads=%d", proofs)
			}
			instances, err := store.ListInstances(t.Context(), scenarioruntime.InstanceFilter{Scenario: "agent-manager"})
			if err != nil {
				t.Fatal(err)
			}
			for _, instance := range instances {
				if instance.InstanceID == old.InstanceID {
					want := scenarioruntime.StatusStopped
					if unknown {
						want = scenarioruntime.StatusStarting
					}
					if instance.Status != want {
						t.Fatalf("old instance status=%s want=%s", instance.Status, want)
					}
				}
			}
		})
	}
}

const drainedOwnerFixture = `{"closed":true,"revision":7,"admitting":0,"remaining":0,"drained":true,"lifecycleInterlock":"scenario-lock-v1","inventory":{"remaining":0,"work":[],"executors":[]}}`

type bootstrapChangedGenerationStore struct {
	scenarioRuntimeStore
	t *testing.T
}

func (s bootstrapChangedGenerationStore) Close() error { return nil }
func (s bootstrapChangedGenerationStore) StopLease(context.Context, string, int64, string) (scenarioruntime.Instance, error) {
	return scenarioruntime.Instance{}, scenarioruntime.ErrStaleGeneration
}

func (s bootstrapChangedGenerationStore) ReleaseActivePortClaimsForInstance(context.Context, string) ([]scenarioruntime.PortClaim, error) {
	s.t.Fatal("generation change must refuse before claim release")
	return nil, nil
}

func TestOwnerMaintenanceBootstrapPinsGenerationBeforeClaimRelease(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	instance, err := store.CreateInstance(t.Context(), scenarioruntime.Instance{Scenario: "agent-manager", Status: scenarioruntime.StatusStarting})
	if err != nil {
		t.Fatal(err)
	}
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.runtimeRegistry = func(context.Context, string) (scenarioRuntimeStore, error) {
			return bootstrapChangedGenerationStore{scenarioRuntimeStore: store, t: t}, nil
		}
		deps.requireRuntimeAbsent = func(context.Context, maintenance.RuntimeScopeRef) error { return nil }
	})
	err = runner.retireAbsentOwner(t.Context(), scenario.Scenario{Slug: "agent-manager"}, registryRuntimeView{Present: true, Instance: instance})
	if !errors.Is(err, scenarioruntime.ErrStaleGeneration) || !errors.Is(err, ErrOwnerMaintenanceRequired) {
		t.Fatalf("changed generation admitted: %v", err)
	}
}

func drainedOwner(t *testing.T) ownerMaintenanceStanding {
	t.Helper()
	var state ownerMaintenanceStanding
	if err := json.Unmarshal([]byte(drainedOwnerFixture), &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestOwnerMaintenanceRefusesUnsafeEvidence(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, tc := range []struct {
		name string
		edit func(*ownerMaintenanceStanding)
		err  error
	}{
		{name: "admission_open", edit: func(s *ownerMaintenanceStanding) { s.Closed = false }},
		{name: "unobserved_admissions", edit: func(s *ownerMaintenanceStanding) { s.Admitting = nil }},
		{name: "admission_in_progress", edit: func(s *ownerMaintenanceStanding) { s.Admitting = intPtr(1) }},
		{name: "unknown_remaining", edit: func(s *ownerMaintenanceStanding) { s.Remaining = nil }},
		{name: "active_work", edit: func(s *ownerMaintenanceStanding) { s.Remaining = intPtr(1) }},
		{name: "negative_remaining", edit: func(s *ownerMaintenanceStanding) { s.Remaining = intPtr(-1) }},
		{name: "not_drained", edit: func(s *ownerMaintenanceStanding) { s.Drained = false }},
		{name: "unfenced_revision", edit: func(s *ownerMaintenanceStanding) { s.Revision = 0 }},
		{name: "resume_not_interlocked", edit: func(s *ownerMaintenanceStanding) { s.LifecycleInterlock = "" }},
		{name: "missing_inventory", edit: func(s *ownerMaintenanceStanding) { s.Inventory = nil }},
		{name: "unknown_physical_scope", edit: func(s *ownerMaintenanceStanding) { s.Inventory.Unknown = []string{"descendants unavailable"} }},
		{name: "terminal_run_live_executor", edit: func(s *ownerMaintenanceStanding) {
			s.Inventory.Executors = []json.RawMessage{json.RawMessage(`{"id":"terminal-run","alive":true}`)}
		}},
		{name: "workflow_still_active", edit: func(s *ownerMaintenanceStanding) {
			s.Inventory.Work = []json.RawMessage{json.RawMessage(`{"kind":"workflow"}`)}
		}},
		{name: "inventory_count_missing", edit: func(s *ownerMaintenanceStanding) { s.Inventory.Remaining = nil }},
		{name: "owner_unavailable", err: errors.New("owner offline")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := drainedOwner(t)
			if tc.edit != nil {
				tc.edit(&state)
			}
			r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) { return state, tc.err }}}
			if _, err := r.requireOwnerMaintenance(t.Context(), scenario.Scenario{Slug: "agent-manager"}, 0); !errors.Is(err, ErrOwnerMaintenanceRequired) {
				t.Fatalf("unsafe observation admitted: %v", err)
			}
		})
	}
}

func TestOwnerMaintenanceProtectsCurrentExecutor(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "identified-test-executor")
	r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
		t.Fatal("current executor protection must precede even a false empty inventory")
		return drainedOwner(t), nil
	}}}
	if _, err := r.requireOwnerMaintenance(t.Context(), scenario.Scenario{Slug: "agent-manager"}, 0); !errors.Is(err, ErrOwnerMaintenanceRequired) {
		t.Fatalf("self-maintenance admitted: %v", err)
	}
}

func TestOwnerMaintenanceProtectsLegacyTaggedExecutor(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, key := range []string{"VROOLI_RUN_ID", "CODEX_AGENT_TAG", "OPENCODE_AGENT_TAG"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "retained-legacy-executor")
			r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
				t.Fatal("legacy executor protection must precede false-empty inventory")
				return drainedOwner(t), nil
			}}}
			if _, err := r.requireOwnerMaintenance(t.Context(), scenario.Scenario{Slug: "agent-manager"}, 0); !errors.Is(err, ErrOwnerMaintenanceRequired) {
				t.Fatalf("legacy executor admitted: %v", err)
			}
		})
	}
}

func TestRecoveryFirstRestartRequiresCompleteIdentifiedExecutorInventory(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, tc := range []struct {
		name string
		edit func(*ownerMaintenanceStanding)
		want bool
	}{
		{name: "identified active executor", edit: func(s *ownerMaintenanceStanding) {
			s.Remaining = intPtr(1)
			s.Inventory.Remaining = intPtr(1)
			s.Inventory.Executors = []json.RawMessage{json.RawMessage(`{"id":"run-1","alive":true}`)}
		}, want: true},
		{name: "unknown physical scope", edit: func(s *ownerMaintenanceStanding) {
			s.Remaining = intPtr(1)
			s.Inventory.Remaining = intPtr(1)
			s.Inventory.Executors = []json.RawMessage{json.RawMessage(`{"id":"run-1","alive":true}`)}
			s.Inventory.Unknown = []string{"descendants unavailable"}
		}},
		{name: "missing identity", edit: func(s *ownerMaintenanceStanding) {
			s.Remaining = intPtr(1)
			s.Inventory.Remaining = intPtr(1)
			s.Inventory.Executors = []json.RawMessage{json.RawMessage(`{"alive":true}`)}
		}},
		{name: "inactive executor", edit: func(s *ownerMaintenanceStanding) {
			s.Remaining = intPtr(1)
			s.Inventory.Remaining = intPtr(1)
			s.Inventory.Executors = []json.RawMessage{json.RawMessage(`{"id":"run-1","alive":false}`)}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := drainedOwner(t)
			tc.edit(&state)
			r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) { return state, nil }}}
			_, err := r.allowRecoveryFirstRestart(t.Context(), scenario.Scenario{Slug: "agent-manager"})
			if tc.want && err != nil {
				t.Fatalf("valid recovery-first inventory refused: %v", err)
			}
			if !tc.want && err == nil {
				t.Fatal("unsafe recovery-first inventory admitted")
			}
		})
	}
}

func TestRecoveryFirstRestartProtectsIdentifiedCurrentExecutor(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "identified-test-executor")
	r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
		t.Fatal("current executor protection must precede owner observation")
		return drainedOwner(t), nil
	}}}
	if _, err := r.allowRecoveryFirstRestart(t.Context(), scenario.Scenario{Slug: "agent-manager"}); err == nil {
		t.Fatal("recovery-first restart admitted the current executor")
	}
}

func TestRecoveryFirstRestartAcceptsTypedStartupRecoveryUnavailable(t *testing.T) {
	state := drainedOwner(t)
	state.Remaining = nil
	state.Inventory.Remaining = nil
	state.Inventory.Executors = nil
	state.Inventory.Unknown = []string{"startup recovery degraded"}
	payload, err := json.Marshal(struct {
		State ownerMaintenanceStanding `json:"state"`
		Error string                   `json:"error"`
	}{State: state, Error: "startup recovery is not ready"})
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
		return ownerMaintenanceStanding{}, &ownerAdmissionReadError{status: http.StatusServiceUnavailable, body: payload}
	}}}
	if revision, err := r.allowRecoveryFirstRestart(t.Context(), scenario.Scenario{Slug: "agent-manager"}); err != nil || revision != state.Revision {
		t.Fatalf("typed startup recovery response revision=%d err=%v, want revision %d and admission", revision, err, state.Revision)
	}
}

func TestRecoveryFirstRestartRejectsUnrelatedUnavailableOwner(t *testing.T) {
	r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
		return ownerMaintenanceStanding{}, &ownerAdmissionReadError{status: http.StatusServiceUnavailable, body: []byte(`{"error":"database unavailable"}`)}
	}}}
	if _, err := r.allowRecoveryFirstRestart(t.Context(), scenario.Scenario{Slug: "agent-manager"}); err == nil {
		t.Fatal("unrelated unavailable owner response admitted recovery-first restart")
	}
}

func TestOwnerMaintenanceBeforeLifecycleEffects(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, operation := range []string{"stop", "restart", "setup", "dependency_rebuild"} {
		t.Run(operation, func(t *testing.T) {
			root, home := t.TempDir(), t.TempDir()
			writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
				Service:   scenario.ServiceMetadata{Name: "agent-manager"},
				Lifecycle: scenario.Lifecycle{Setup: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "setup-marker", Exec: []string{"bash", "-c", "printf setup >> setup.txt"}}}}},
			})
			reads := 0
			var runner *Runner
			runner = newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
				deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
					reads++
					release, err := runner.tryAcquireScenarioLock("agent-manager")
					if err == nil {
						release()
						t.Error("owner precondition was read outside the target lifecycle lock")
					} else if !errors.Is(err, ErrScenarioBusy) {
						t.Errorf("unexpected lock error: %v", err)
					}
					return ownerMaintenanceStanding{}, errors.New("owner inventory unavailable")
				}
			})
			item, err := runner.loadScenario("agent-manager", "")
			if err != nil {
				t.Fatal(err)
			}
			switch operation {
			case "stop":
				err = runner.Stop("agent-manager", StopOptions{})
			case "restart":
				_, err = runner.Restart("agent-manager", StartOptions{})
			case "setup":
				_, err = runner.RunPhaseDetailed("agent-manager", "setup", PhaseOptions{})
			case "dependency_rebuild":
				err = runner.rebuildDependencyArtifactsContext(t.Context(), item, registryRuntimeView{})
			}
			if !errors.Is(err, ErrOwnerMaintenanceRequired) || reads == 0 {
				t.Fatalf("%s did not refuse unknown admission before effects: reads=%d err=%v", operation, reads, err)
			}
			if _, err := os.Stat(filepath.Join(item.Path, "setup.txt")); !os.IsNotExist(err) {
				t.Fatalf("refused operation ran setup: %v", err)
			}
			release, err := runner.tryAcquireScenarioLock("agent-manager")
			if err != nil {
				t.Fatalf("refused operation leaked lock: %v", err)
			}
			release()
		})
	}
}

func TestOwnerMaintenanceHTTPContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
		ok   bool
	}{
		{"drained", http.StatusOK, drainedOwnerFixture, true},
		{"old_status", http.StatusOK, `{"closed":true,"revision":7,"admitting":0}`, false},
		{"missing_endpoint", http.StatusNotFound, "", false},
		{"unavailable", http.StatusServiceUnavailable, "", false},
		{"redirect", http.StatusTemporaryRedirect, "", false},
		{"malformed", http.StatusOK, `{`, false},
		{"multiple_values", http.StatusOK, drainedOwnerFixture + `{}`, false},
		{"oversized", http.StatusOK, strings.Repeat(" ", 256<<10) + drainedOwnerFixture, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(cliutil.EnvIdentityToken, "")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/api/v1/maintenance/admission" || req.Header.Get("Authorization") != "" {
					t.Errorf("unexpected owner request: %s %s credential=%t", req.Method, req.URL.Path, req.Header.Get("Authorization") != "")
				}
				w.Header().Set("Location", "/must-not-follow")
				w.WriteHeader(tc.code)
				_, _ = fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			port := server.Listener.Addr().(*net.TCPAddr).Port
			r := &Runner{deps: lifecycleDeps{readOwnerMaintenance: func(ctx context.Context, _ scenario.Scenario) (ownerMaintenanceStanding, error) {
				return fetchOwnerMaintenance(ctx, port)
			}}}
			_, err := r.requireOwnerMaintenance(t.Context(), scenario.Scenario{Slug: "agent-manager"}, 0)
			if (err == nil) != tc.ok {
				t.Fatalf("admitted=%t want=%t: %v", err == nil, tc.ok, err)
			}
		})
	}
}

func TestOwnerMaintenanceRestartDrainedAndRevisionRevalidated(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprintf("revision_changed_%t", changed), func(t *testing.T) {
			root, home := t.TempDir(), t.TempDir()
			writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
				Service:   scenario.ServiceMetadata{Name: "agent-manager"},
				Lifecycle: scenario.Lifecycle{Setup: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "setup-marker", Exec: []string{"bash", "-c", "printf setup >> setup.txt"}}}}},
			})
			state, reads := drainedOwner(t), 0
			var runner *Runner
			runner = newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
				deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
					reads++
					release, err := runner.tryAcquireScenarioLock("agent-manager")
					if err == nil {
						release()
						t.Error("explicit resume could overlap maintenance")
					} else if !errors.Is(err, ErrScenarioBusy) {
						t.Errorf("unexpected lock error: %v", err)
					}
					return state, nil
				}
				deps.enforceHostRequirements = func(vrooliruntime.Options) (hostreqkit.Report, error) {
					// A stale preflight receipt must not authorize Stop, even if
					// a misbehaving owner reports another closed, empty revision.
					if changed {
						state.Revision++
					}
					return hostreqkit.Report{}, nil
				}
			})
			result, err := runner.Restart("agent-manager", StartOptions{ForceSetup: true})
			if changed {
				if !errors.Is(err, ErrOwnerMaintenanceRequired) || !strings.Contains(err.Error(), "revision changed") {
					t.Fatalf("changed revision authorized restart: %v", err)
				}
				if _, err := os.Stat(filepath.Join(root, "scenarios/agent-manager/setup.txt")); !os.IsNotExist(err) {
					t.Fatalf("changed revision caused setup effects: %v", err)
				}
			} else {
				if err != nil || result.Scenario.Slug != "agent-manager" {
					t.Fatalf("drained restart failed: result=%+v err=%v", result, err)
				}
				if data, err := os.ReadFile(filepath.Join(root, "scenarios/agent-manager/setup.txt")); err != nil || string(data) != "setup" {
					t.Fatalf("drained restart did not execute setup exactly once: %q %v", data, err)
				}
				if !state.Closed || state.Revision != 7 {
					t.Fatalf("restart changed owner admission: %+v", state)
				}
			}
			if reads != 2 {
				t.Fatalf("owner reads=%d want preflight and immediately before Stop", reads)
			}
		})
	}
}

func TestOwnerMaintenanceAdmissionChangesAtLockAcquisition(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "agent-manager"}})
	state := drainedOwner(t)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) { return state, nil }
	})
	originalLock := lockFileFn
	t.Cleanup(func() { lockFileFn = originalLock })
	lockFileFn = func(file *os.File, nonBlocking bool) (func(), error) {
		release, err := originalLock(file, nonBlocking)
		if err == nil {
			// An earlier empty read would be stale by the time Stop can act.
			state.Closed = false
			state.Admitting = intPtr(1)
		}
		return release, err
	}
	if err := runner.Stop("agent-manager", StopOptions{}); !errors.Is(err, ErrOwnerMaintenanceRequired) {
		t.Fatalf("pre-lock empty observation authorized Stop: %v", err)
	}
}

func TestOwnerMaintenanceDependencyRefreshPreservesOwner(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, bestEffort := range []bool{false, true} {
		t.Run(fmt.Sprintf("best_effort_%t", bestEffort), func(t *testing.T) {
			root, home := t.TempDir(), t.TempDir()
			writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
				Service:   scenario.ServiceMetadata{Name: "agent-manager"},
				Lifecycle: scenario.Lifecycle{Setup: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "setup-marker", Exec: []string{"bash", "-c", "printf setup >> setup.txt"}}}}},
			})
			store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: home})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			_, err = store.CreateInstance(t.Context(), scenarioruntime.Instance{Scenario: "agent-manager", Status: scenarioruntime.StatusRunning})
			if err != nil {
				t.Fatal(err)
			}
			reads := 0
			runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
				deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
					reads++
					state := drainedOwner(t)
					state.Remaining = intPtr(1)
					return state, nil
				}
			})
			policy := scenario.DependencyStartupPolicyMustStart
			if bestEffort {
				policy = scenario.DependencyStartupPolicyTryStart
			}
			consumer := scenario.Scenario{Slug: "swarm-manager", Manifest: scenario.ServiceManifest{
				Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{"agent-manager": {Enabled: true, Required: !bestEffort, StartupPolicy: policy}}},
			}}
			failed, err := runner.ensureDependency(t.Context(), consumer, StartOptions{BestEffort: bestEffort}, newStartSession(t.Context()), 0, "agent-manager", 1)
			if bestEffort {
				if err != nil || failed != "agent-manager" {
					t.Fatalf("unsafe optional dependency must be reported as deferred: failed=%s err=%v", failed, err)
				}
			} else if !errors.Is(err, ErrOwnerMaintenanceRequired) {
				t.Fatalf("unsafe required dependency must refuse: %v", err)
			}
			if reads != 1 {
				t.Fatalf("dependency did not consult its execution owner: reads=%d", reads)
			}
			instances, err := store.ListInstances(t.Context(), scenarioruntime.InstanceFilter{Scenario: "agent-manager"})
			if err != nil || len(instances) != 1 || instances[0].Status != scenarioruntime.StatusRunning {
				t.Fatalf("dependency refresh changed its owner runtime: %+v %v", instances, err)
			}
			if _, err := os.Stat(filepath.Join(root, "scenarios/agent-manager/setup.txt")); !os.IsNotExist(err) {
				t.Fatalf("dependency refresh rebuilt active owner: %v", err)
			}
		})
	}
}

const (
	accountingWorkflowID   = "391b1e6d-3b4b-40ef-9256-356cfb3183c9"
	accountingRunID        = "800ae00c-fb7d-4ae4-97ad-d737cd037fec"
	accountingTraceFixture = `{"execution":{"id":"391b1e6d-3b4b-40ef-9256-356cfb3183c9","status":"WORKFLOW_EXECUTION_STATUS_CANCELLING","terminal_reason":{"code":"cancelled"},"budget_usage":{"node_attempts":1}},"attempts":[{"id":"d866bb5c-d6f8-4902-93db-70c834893637","execution_id":"391b1e6d-3b4b-40ef-9256-356cfb3183c9","status":"dispatched","run_id":"800ae00c-fb7d-4ae4-97ad-d737cd037fec"}]}`
	accountingRunFixture   = `{"run":{"id":"800ae00c-fb7d-4ae4-97ad-d737cd037fec","status":"RUN_STATUS_CANCELLED","ended_at":"2026-09-12T12:00:00Z","finalization_status":"RUN_FINALIZATION_STATUS_NONE"}}`
)

func accountingOwner(t *testing.T) ownerMaintenanceStanding {
	t.Helper()
	s := drainedOwner(t)
	s.Drained = false
	s.Remaining, s.Inventory.Remaining = intPtr(1), intPtr(1)
	s.Inventory.Work = []json.RawMessage{json.RawMessage(`{"id":"` + accountingWorkflowID + `","kind":"workflow","status":"cancelling"}`)}
	return s
}

func TestOwnerMaintenanceAccountingOnlyRequiresCompleteProof(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	for _, tc := range []struct {
		name              string
		edit              func(*ownerMaintenanceStanding)
		proofErr          error
		revision          int64
		wantProof, wantOK bool
	}{
		{name: "retained_unknown_accounting", wantProof: true, wantOK: true},
		{name: "child_proof_unavailable", wantProof: true, proofErr: errors.New("original child missing")},
		{name: "revision_changed", revision: 8},
		{name: "live_executor", edit: func(s *ownerMaintenanceStanding) {
			s.Inventory.Executors = []json.RawMessage{json.RawMessage(`{"alive":true}`)}
		}},
		{name: "unknown_scope", edit: func(s *ownerMaintenanceStanding) { s.Inventory.Unknown = []string{"unreadable process"} }},
		{name: "truncated_inventory", edit: func(s *ownerMaintenanceStanding) { s.Remaining = intPtr(2); s.Inventory.Remaining = intPtr(2) }},
		{name: "nonworkflow", edit: func(s *ownerMaintenanceStanding) {
			s.Inventory.Work[0] = json.RawMessage(`{"id":"` + accountingWorkflowID + `","kind":"run","status":"cancelling"}`)
		}},
		{name: "active_workflow", edit: func(s *ownerMaintenanceStanding) {
			s.Inventory.Work[0] = json.RawMessage(`{"id":"` + accountingWorkflowID + `","kind":"workflow","status":"running"}`)
		}},
		{name: "duplicate_workflow", edit: func(s *ownerMaintenanceStanding) {
			s.Remaining = intPtr(2)
			s.Inventory.Remaining = intPtr(2)
			s.Inventory.Work = append(s.Inventory.Work, s.Inventory.Work[0])
		}},
		{name: "false_drained", edit: func(s *ownerMaintenanceStanding) { s.Drained = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := accountingOwner(t)
			if tc.edit != nil {
				tc.edit(&s)
			}
			calls := 0
			r := &Runner{deps: lifecycleDeps{
				readOwnerMaintenance: func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) { return s, nil },
				verifyOwnerAccounting: func(_ context.Context, _ scenario.Scenario, ids []string) error {
					calls++
					if len(ids) != 1 || ids[0] != accountingWorkflowID {
						t.Fatalf("wrong original identities: %v", ids)
					}
					return tc.proofErr
				},
			}}
			rev, err := r.requireOwnerMaintenance(t.Context(), scenario.Scenario{Slug: "agent-manager"}, tc.revision)
			if (err == nil) != tc.wantOK || (calls > 0) != tc.wantProof {
				t.Fatalf("revision=%d err=%v proof calls=%d", rev, err, calls)
			}
			if tc.wantOK && (rev != 7 || s.Drained || *s.Remaining != 1) {
				t.Fatal("accounting retention must not be rewritten as drainage")
			}
		})
	}
}

func TestOwnerMaintenanceAccountingOnlyOriginalChildHTTPProof(t *testing.T) {
	for _, tc := range []struct {
		name, trace, run string
		code             int
		ok               bool
	}{
		{name: "cancelled_unknown_accounting", ok: true},
		{name: "completed_unknown_accounting", run: strings.Replace(accountingRunFixture, "RUN_STATUS_CANCELLED", "RUN_STATUS_COMPLETE", 1), ok: true},
		{name: "running_child", run: strings.Replace(accountingRunFixture, "RUN_STATUS_CANCELLED", "RUN_STATUS_RUNNING", 1)},
		{name: "no_ended_at", run: strings.Replace(accountingRunFixture, `,"ended_at":"2026-09-12T12:00:00Z"`, "", 1)},
		{name: "child_finalizing", run: strings.Replace(accountingRunFixture, "RUN_FINALIZATION_STATUS_NONE", "RUN_FINALIZATION_STATUS_PENDING", 1)},
		{name: "missing_finalization_observation", run: strings.Replace(accountingRunFixture, `,"finalization_status":"RUN_FINALIZATION_STATUS_NONE"`, "", 1)},
		{name: "wrong_child", run: strings.Replace(accountingRunFixture, accountingRunID, accountingWorkflowID, 1)},
		{name: "truncated_attempts", trace: strings.Replace(accountingTraceFixture, `"node_attempts":1`, `"node_attempts":2`, 1)},
		{name: "missing_dispatch", trace: strings.Replace(accountingTraceFixture, `"run_id":"`+accountingRunID+`"`, `"run_id":""`, 1)},
		{name: "nested_workflow", trace: strings.Replace(accountingTraceFixture, `"status":"dispatched"`, `"status":"dispatched","child_execution_id":"`+accountingWorkflowID+`"`, 1)},
		{name: "active_workflow", trace: strings.Replace(accountingTraceFixture, "WORKFLOW_EXECUTION_STATUS_CANCELLING", "WORKFLOW_EXECUTION_STATUS_RUNNING", 1)},
		{name: "not_cancelled", trace: strings.Replace(accountingTraceFixture, `"code":"cancelled"`, `"code":"unknown"`, 1)},
		{name: "uncertain_attempt", trace: strings.Replace(accountingTraceFixture, `"status":"dispatched"`, `"status":"dispatching"`, 1)},
		{name: "wrong_attempt_owner", trace: strings.Replace(accountingTraceFixture, `"execution_id":"`+accountingWorkflowID+`"`, `"execution_id":"`+accountingRunID+`"`, 1)},
		{name: "malformed", trace: "{"},
		{name: "oversized", trace: strings.Repeat(" ", 256<<10) + accountingTraceFixture},
		{name: "redirect", code: http.StatusTemporaryRedirect},
		{name: "missing_owner", code: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.trace == "" {
				tc.trace = accountingTraceFixture
			}
			if tc.run == "" {
				tc.run = accountingRunFixture
			}
			if tc.code == 0 {
				tc.code = http.StatusOK
			}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodGet || r.Header.Get("Authorization") != "" {
					t.Fatal("proof must be a credential-free read")
				}
				w.Header().Set("Location", "/must-not-follow")
				w.WriteHeader(tc.code)
				switch r.URL.Path {
				case "/api/v1/workflow-executions/" + accountingWorkflowID + "/trace":
					if r.URL.RawQuery != "limit=1" {
						t.Error("journal read must be bounded")
					}
					fmt.Fprint(w, tc.trace)
				case "/api/v1/runs/" + accountingRunID:
					fmt.Fprint(w, tc.run)
				default:
					t.Errorf("unscoped owner read: %s", r.URL.Path)
				}
			}))
			defer server.Close()
			err := verifyAccountingOnlyOwner(t.Context(), server.Listener.Addr().(*net.TCPAddr).Port, []string{accountingWorkflowID})
			if (err == nil) != tc.ok {
				t.Fatalf("proof admitted=%t want=%t: %v", err == nil, tc.ok, err)
			}
			if tc.ok && requests != 2 {
				t.Fatalf("expected one trace and one original child read; got %d", requests)
			}
		})
	}
}
