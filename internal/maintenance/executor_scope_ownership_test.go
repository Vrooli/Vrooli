package maintenance

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

const fixtureServiceScope = "/user.slice/user-1000.slice/user@1000.service/vrooli-services.slice/vrooli-service-fixture-start-api-63c8cc8849ebccb544a3c011e973a716.scope"

type serviceScopeFixture struct {
	controller       *Controller
	instance         scenarioruntime.Instance
	ref              scenarioruntime.ProcessRef
	table            map[int]processTableEntry
	identities       map[int]scopeIdentity
	births           map[int]supervision.ProcessInfo
	executor         ExecutorScopeRef
	skipRegistration bool
	beforeCheck      func()
}

func newServiceScopeFixture(t *testing.T) *serviceScopeFixture {
	t.Helper()
	if hostsession.CurrentBootID() == "" {
		t.Skip("native service ownership requires kernel boot identity")
	}
	start, end, birth := time.Unix(1000, 0), time.Unix(3000, 0), time.Unix(2000, 0)
	pid, pgid := 20, 20
	return &serviceScopeFixture{
		controller: &Controller{Home: t.TempDir()},
		instance:   scenarioruntime.Instance{InstanceID: "instance", Scenario: "fixture", Variant: "live", Status: scenarioruntime.StatusRunning, OwnerKind: scenarioruntime.OwnerKindLifecycle, HostBootID: hostsession.CurrentBootID(), StartedAt: start},
		ref:        scenarioruntime.ProcessRef{InstanceID: "instance", PID: &pid, PGID: &pgid, Step: "start-api", ProcessID: "fixture-start-api", Status: "running", StartedAt: birth, HostBootID: hostsession.CurrentBootID()},
		table:      map[int]processTableEntry{20: {PID: 20, PPID: 1, PGID: 20, Cgroup: fixtureServiceScope}, 30: {PID: 30, PPID: 20, PGID: 30, Cgroup: fixtureServiceScope}},
		identities: map[int]scopeIdentity{20: {Scenario: "fixture", Variant: "live", InstanceID: "instance", Tags: []string{"current", "legacy"}, TagLabels: map[string]string{"VROOLI_AGENT_TAG": "current", "OPENCODE_AGENT_TAG": "legacy"}}, 30: {Scenario: "fixture", Variant: "live", InstanceID: "instance", Tags: []string{"current", "legacy"}, TagLabels: map[string]string{"VROOLI_AGENT_TAG": "current", "OPENCODE_AGENT_TAG": "legacy"}}},
		births:     map[int]supervision.ProcessInfo{20: {PID: 20, StartedAt: birth}, 30: {PID: 30, StartedAt: birth.Add(time.Second)}},
		executor:   ExecutorScopeRef{RunID: "run", Tag: "current", LegacyTag: "legacy", PID: 10, PGID: 10, StartedAt: &start, EndedAt: &end},
	}
}

func (f *serviceScopeFixture) check(t *testing.T, want string) {
	t.Helper()
	store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: f.controller.Home})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.CreateInstance(t.Context(), f.instance); err != nil {
		t.Fatal(err)
	}
	if !f.skipRegistration {
		if _, err = store.AddProcessRef(t.Context(), f.ref); err != nil {
			t.Fatal(err)
		}
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	scopeFixture(t, f.table, f.identities)
	scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) { return f.births, nil }
	if f.beforeCheck != nil {
		f.beforeCheck()
	}
	report, err := f.controller.ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{f.executor}})
	if err != nil || len(report.Executors) != 1 || report.Executors[0].State != want || report.Complete != (want != "unknown") {
		t.Fatalf("want %s; report=%+v error=%v", want, report, err)
	}
}

func TestExecutorScopeServiceExclusionRequiresCompleteOwnershipProof(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*serviceScopeFixture)
	}{
		{"missing_process_registration", func(f *serviceScopeFixture) { f.skipRegistration = true }},
		{"spoofed_instance", func(f *serviceScopeFixture) {
			identity := f.identities[20]
			identity.InstanceID = "unregistered"
			f.identities[20] = identity
		}},
		{"missing_instance", func(f *serviceScopeFixture) {
			identity := f.identities[20]
			identity.InstanceID = ""
			f.identities[20] = identity
		}},
		{"mismatched_scenario", func(f *serviceScopeFixture) { f.instance.Scenario = "different" }},
		{"mismatched_variant", func(f *serviceScopeFixture) { f.instance.Variant = "shadow" }},
		{"missing_live_variant", func(f *serviceScopeFixture) {
			identity := f.identities[20]
			identity.Variant = ""
			f.identities[20] = identity
		}},
		{"stopped_instance", func(f *serviceScopeFixture) { f.instance.Status = scenarioruntime.StatusStopped }},
		{"foreign_owner", func(f *serviceScopeFixture) { f.instance.OwnerKind = "agent" }},
		{"previous_boot_instance", func(f *serviceScopeFixture) { f.instance.HostBootID = "old-boot" }},
		{"previous_boot_registration", func(f *serviceScopeFixture) { f.ref.HostBootID = "old-boot" }},
		{"ended_registration", func(f *serviceScopeFixture) { f.ref.EndedAt = f.executor.EndedAt }},
		{"stopped_registration", func(f *serviceScopeFixture) { f.ref.Status = "exited" }},
		{"missing_registration_pid", func(f *serviceScopeFixture) { f.ref.PID = nil }},
		{"wrong_registration_pid", func(f *serviceScopeFixture) { pid := 99; f.ref.PID = &pid }},
		{"wrong_registration_group", func(f *serviceScopeFixture) { pgid := 99; f.ref.PGID = &pgid }},
		{"wrong_registration_step", func(f *serviceScopeFixture) { f.ref.Step = "start-ui" }},
		{"reused_registration_pid", func(f *serviceScopeFixture) { f.ref.StartedAt = f.ref.StartedAt.Add(-time.Hour) }},
		{"registration_after_birth", func(f *serviceScopeFixture) { f.ref.StartedAt = f.ref.StartedAt.Add(time.Hour) }},
		{"birth_unavailable", func(f *serviceScopeFixture) { delete(f.births, 20) }},
		{"scope_missing", func(f *serviceScopeFixture) { entry := f.table[20]; entry.Cgroup = ""; f.table[20] = entry }},
		{"scope_name_is_not_registration", func(f *serviceScopeFixture) {
			entry := f.table[20]
			entry.Cgroup = fixtureServiceScope + "-spoof"
			f.table[20] = entry
		}},
		{"agent_slice", func(f *serviceScopeFixture) {
			entry := f.table[20]
			entry.Cgroup = "/vrooli-agents.slice/" + "vrooli-service-fixture-start-api-63c8cc8849ebccb544a3c011e973a716.scope"
			f.table[20] = entry
		}},
		{"run_labelled_service_root", func(f *serviceScopeFixture) {
			identity := f.identities[20]
			identity.RunID = "other-run"
			f.identities[20] = identity
		}},
		{"child_independent_tag_key", func(f *serviceScopeFixture) { f.identities[30].TagLabels["CLAUDE_AGENT_TAG"] = "current" }},
		{"child_different_instance", func(f *serviceScopeFixture) {
			identity := f.identities[30]
			identity.InstanceID = "different"
			f.identities[30] = identity
		}},
		{"child_different_scope", func(f *serviceScopeFixture) { entry := f.table[30]; entry.Cgroup += "-other"; f.table[30] = entry }},
		{"child_missing_parent", func(f *serviceScopeFixture) { entry := f.table[30]; entry.PPID = 99; f.table[30] = entry }},
		{"child_birth_unavailable", func(f *serviceScopeFixture) { delete(f.births, 30) }},
		{"child_predates_registered_root", func(f *serviceScopeFixture) {
			f.births[30] = supervision.ProcessInfo{PID: 30, StartedAt: f.ref.StartedAt.Add(-time.Minute)}
		}},
		{"birth_changed_during_read", func(f *serviceScopeFixture) {
			f.beforeCheck = func() {
				calls := 0
				scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
					calls++
					if calls > 1 {
						return map[int]supervision.ProcessInfo{20: {PID: 20, StartedAt: time.Unix(2005, 0)}}, nil
					}
					return f.births, nil
				}
			}
		}},
		{"birth_read_failed", func(f *serviceScopeFixture) {
			f.beforeCheck = func() {
				scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) { return nil, errors.New("unavailable") }
			}
		}},
		{"live_execution", func(f *serviceScopeFixture) { f.executor.EndedAt = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) { f := newServiceScopeFixture(t); tc.change(f); f.check(t, "present") })
	}
}

func TestExecutorScopeServiceOwnershipDoesNotResolveNumericExecutorAmbiguity(t *testing.T) {
	for _, field := range []string{"pid", "group"} {
		t.Run(field, func(t *testing.T) {
			f := newServiceScopeFixture(t)
			if field == "pid" {
				f.executor.PID = 30
			} else {
				f.executor.PGID = 30
			}
			f.check(t, "unknown")
		})
	}
}

func TestExecutorScopeServiceOwnershipRetainsPositiveBeyondSampleLimit(t *testing.T) {
	f := newServiceScopeFixture(t)
	for pid := 40; pid < 240; pid++ {
		f.table[pid] = processTableEntry{PID: pid, PPID: 20, PGID: pid, Cgroup: fixtureServiceScope}
		f.identities[pid] = f.identities[30]
		f.births[pid] = supervision.ProcessInfo{PID: pid, StartedAt: f.ref.StartedAt.Add(time.Second)}
	}
	identity := f.identities[239]
	identity.TagLabels = map[string]string{"VROOLI_AGENT_TAG": "current", "OPENCODE_AGENT_TAG": "legacy", "CLAUDE_AGENT_TAG": "current"}
	f.identities[239] = identity
	f.check(t, "present")
}

func TestExecutorScopeServiceOwnershipRetainsIndependentChildLabel(t *testing.T) {
	f := newServiceScopeFixture(t)
	identity := f.identities[30]
	identity.Tags = []string{"child-current", "legacy"}
	identity.TagLabels = map[string]string{"VROOLI_AGENT_TAG": "child-current", "OPENCODE_AGENT_TAG": "legacy"}
	f.identities[30] = identity
	f.executor.Tag = "child-current"
	f.check(t, "present")
}

func TestExecutorScopeSeparateLifecycleChildHasIndependentOwnership(t *testing.T) {
	f := newServiceScopeFixture(t)
	// The lifecycle may have been invoked below a still-running different
	// executor. Registration and the new native scope establish a new owner.
	f.table[40] = processTableEntry{PID: 40, PGID: 40, Cgroup: "/vrooli-agents.slice/live.scope"}
	f.identities[40] = scopeIdentity{RunID: "live-other"}
	entry := f.table[20]
	entry.PPID = 40
	f.table[20] = entry
	f.check(t, "absent")
}

func TestExecutorScopeUnreadableRegistryCannotGrantServiceExclusion(t *testing.T) {
	f := newServiceScopeFixture(t)
	scopeFixture(t, f.table, f.identities)
	store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: f.controller.Home})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	path, err := scenarioruntime.DefaultDBPath(f.controller.Home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid registry"), 0600); err != nil {
		t.Fatal(err)
	}
	scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) { return f.births, nil }
	absent := ExecutorScopeRef{RunID: "absent-run", Tag: "absent-tag", LegacyTag: "absent-legacy", EndedAt: f.executor.EndedAt}
	report, err := f.controller.ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{f.executor, absent}})
	if err != nil || !report.Complete || report.Executors[0].State != "present" || report.Executors[1].State != "absent" {
		t.Fatalf("report=%+v error=%v", report, err)
	}
	for _, evidence := range report.Executors {
		if len(evidence.Reasons) != 0 {
			t.Fatalf("optional proof failure changed complete physical evidence: %+v", evidence)
		}
	}
	// Already absent and non-ended references must not even query the broken
	// registry. Their physical classification needs no service exception.
	live := f.executor
	live.EndedAt = nil
	for _, reference := range []ExecutorScopeRef{absent, live} {
		inherited, err := f.controller.inheritedServiceTags(t.Context(), []ExecutorScopeRef{reference}, f.table, f.identities, f.births, f.births)
		if err != nil || len(inherited) != 0 {
			t.Fatalf("unnecessary optional registry lookup for %s: inherited=%v error=%v", reference.RunID, inherited, err)
		}
	}
}

func TestExecutorScopeRegisteredServiceOwnsInheritedLabels(t *testing.T) {
	f := newServiceScopeFixture(t)
	f.check(t, "absent")
}

func TestExecutorScopeRegisteredServiceRetainsActualRunDescendant(t *testing.T) {
	f := newServiceScopeFixture(t)
	identity := f.identities[30]
	identity.RunID = "run"
	f.identities[30] = identity
	f.check(t, "present")
}

func TestExecutorScopeServiceOwnershipCannotCrossLiveRunBoundary(t *testing.T) {
	f := newServiceScopeFixture(t)
	f.table[25] = processTableEntry{PID: 25, PPID: 20, PGID: 25, Cgroup: fixtureServiceScope}
	f.identities[25] = scopeIdentity{RunID: "live-run", Scenario: "fixture", Variant: "live", InstanceID: "instance"}
	f.births[25] = supervision.ProcessInfo{PID: 25, StartedAt: f.ref.StartedAt.Add(time.Second)}
	entry := f.table[30]
	entry.PPID = 25
	f.table[30] = entry
	f.check(t, "present")
}

func TestExecutorScopeServiceOwnershipSupportsRegisteredShadowVariant(t *testing.T) {
	f := newServiceScopeFixture(t)
	f.instance.Variant = "shadow"
	for pid, identity := range f.identities {
		identity.Variant = "shadow"
		f.identities[pid] = identity
		entry := f.table[pid]
		entry.Cgroup = "/user.slice/user-1000.slice/user@1000.service/vrooli-services.slice/vrooli-service-fixture-shadow-start-api-c1f8dafe99fce65fa4ff26a1456b32c5.scope"
		f.table[pid] = entry
	}
	f.check(t, "absent")
}
