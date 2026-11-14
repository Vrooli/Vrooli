//go:build linux

package lifecycle

import (
	"errors"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestBackgroundServiceHandoffIsRequiredBeforePublication(t *testing.T) {
	for _, tc := range []struct {
		name         string
		placementErr error
		wantFailure  bool
	}{
		{"manager-unavailable", errors.New("fixture manager denied placement"), true},
		{"placed", nil, false},
		{"unsupported-platform", platform.ErrUnsupported, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, home := t.TempDir(), t.TempDir()
			writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
				Service: scenario.ServiceMetadata{Name: "alpha"},
				Lifecycle: scenario.Lifecycle{Develop: scenario.Phase{Steps: []scenario.PhaseStep{
					{Name: "start-api", Exec: []string{"/bin/sleep", "30"}, Background: true},
				}}},
			})
			pid, calls := 0, 0
			r := newLifecycleRunnerForTest(t, root, home, func(d *lifecycleDeps) {
				d.adoptIntoScope = func(spec platform.AdoptSpec) (platform.ScopeRef, string, error) {
					pid, calls = spec.PID, calls+1
					group, err := platform.ProcessGroupID(pid)
					if err != nil || group != pid {
						t.Fatalf("service is not its own detached group: pid=%d group=%d err=%v", pid, group, err)
					}
					if spec.Slice != platform.ServicesSlice || spec.Scope != scenarioruntime.ServiceScopeName("alpha", "start-api") {
						t.Fatalf("wrong service destination: %+v", spec)
					}
					return platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/fixture/service"}, platform.MethodTransientUnit, tc.placementErr
				}
			})
			t.Cleanup(func() {
				if pid > 0 {
					_ = platform.SignalProcessGroup(pid, true)
				}
			})
			item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
			if err != nil {
				t.Fatal(err)
			}
			started := time.Now()
			err = r.startTrackedProcess(item, "develop", item.Manifest.Lifecycle.Develop.Steps[0], map[string]string{})
			if calls != 1 || (err != nil) != tc.wantFailure {
				t.Fatalf("handoff calls=%d error=%v, want failure=%v", calls, err, tc.wantFailure)
			}
			if time.Since(started) > 5*time.Second {
				t.Fatal("launcher waited for daemon lifetime")
			}
			records, readErr := process.ReadScenarioRecords(home, "alpha")
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tc.wantFailure {
				if !errors.Is(err, tc.placementErr) {
					t.Fatalf("lost manager failure: %v", err)
				}
				if len(records) != 0 || process.IsPIDRunning(pid) {
					t.Fatalf("unplaced service survived or was published: pid=%d records=%+v", pid, records)
				}
			} else if len(records) != 1 || !process.IsPIDRunning(pid) {
				t.Fatalf("successful handoff lost detached service: pid=%d records=%+v", pid, records)
			}
		})
	}
}

func TestAgentManagerAPIHandoffSurvivesOccupiedPreviousScopes(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	const legacyScope = "vrooli-service-agent-manager-start-api-6acf5936c54e535eab5b633b80b28d84"
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "agent-manager"},
		Lifecycle: scenario.Lifecycle{Develop: scenario.Phase{Steps: []scenario.PhaseStep{
			{Name: "start-api", Exec: []string{"/bin/sleep", "30"}, Background: true},
		}}},
	})
	occupied := map[string]bool{legacyScope: true}
	var adopted []platform.AdoptSpec
	r := newLifecycleRunnerForTest(t, root, home, func(d *lifecycleDeps) {
		d.adoptIntoScope = func(spec platform.AdoptSpec) (platform.ScopeRef, string, error) {
			adopted = append(adopted, spec)
			if occupied[spec.Scope] {
				return platform.ScopeRef{}, "", errors.New("fixture scope is still occupied")
			}
			occupied[spec.Scope] = true
			return platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: spec.Scope}, platform.MethodTransientUnit, nil
		}
	})
	t.Cleanup(func() {
		for _, spec := range adopted {
			_ = platform.SignalProcessGroup(spec.PID, true)
		}
	})
	item, err := scenario.Load(root, "agent-manager", scenario.SandboxEnv{})
	if err != nil {
		t.Fatal(err)
	}
	for launch := range 2 {
		if err := r.startTrackedProcess(item, "develop", item.Manifest.Lifecycle.Develop.Steps[0], map[string]string{}); err != nil {
			t.Fatal(err)
		}
		records, err := process.ReadScenarioRecords(home, "agent-manager")
		if err != nil || len(records) != 1 || len(adopted) != launch+1 {
			t.Fatalf("launch %d: records=%+v adopted=%+v error=%v", launch, records, adopted, err)
		}
		spec := adopted[launch]
		if spec.Scope != legacyScope+"-"+records[0].Generation || spec.Slice != platform.ServicesSlice || !spec.Containment.ManagedOOMAvoid || spec.PID != records[0].PID {
			t.Fatalf("launch %d did not publish the adopted generation: spec=%+v record=%+v", launch, spec, records[0])
		}
	}
	if !process.IsPIDRunning(adopted[0].PID) {
		t.Fatal("placing the replacement interrupted the previous process")
	}
}
