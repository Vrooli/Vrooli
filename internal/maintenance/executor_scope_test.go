package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

func scopeFixture(t *testing.T, table map[int]processTableEntry, identities map[int]scopeIdentity) {
	t.Helper()
	oldTable, oldIdentity, oldBirths, oldListeners := scopeProcessTable, scopeReadIdentity, scopeProcessBirths, scopeListeners
	oldLeases, oldMembers, oldGone, oldBoot := scopeEditorLeases, scopeEditorScopeMembers, scopeEditorScopeGone, scopeCurrentBootID
	oldCgroupPath, oldOccupancy, oldScopeProcesses, oldLeaseBySession := scopeSessionCgroupPath, scopeEditorScopeOccupancy, scopeEditorScopeProcesses, scopeEditorLeaseBySession
	t.Cleanup(func() {
		scopeProcessTable, scopeReadIdentity, scopeProcessBirths, scopeListeners = oldTable, oldIdentity, oldBirths, oldListeners
		scopeEditorLeases, scopeEditorScopeMembers, scopeEditorScopeGone, scopeCurrentBootID = oldLeases, oldMembers, oldGone, oldBoot
		scopeSessionCgroupPath, scopeEditorScopeOccupancy, scopeEditorScopeProcesses, scopeEditorLeaseBySession = oldCgroupPath, oldOccupancy, oldScopeProcesses, oldLeaseBySession
	})
	scopeProcessTable = func(context.Context) (map[int]processTableEntry, error) { return table, nil }
	scopeReadIdentity = func(entry processTableEntry) (scopeIdentity, error) { return identities[entry.PID], nil }
	scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
		return map[int]supervision.ProcessInfo{}, nil
	}
	scopeEditorLeases = func(context.Context, string) ([]scenarioruntime.EditorLease, error) { return nil, nil }
	scopeEditorScopeMembers = func(string) ([]int, error) { return nil, nil }
	scopeEditorScopeGone = func(context.Context, string) (bool, error) { return false, nil }
	scopeCurrentBootID = func() string { return "boot-current" }
	scopeListeners = func() network.TCPListenerSnapshot { return network.TCPListenerSnapshot{Known: true} }
}

func TestExecutorScopeExactAttachedLeaseProvesRunWithoutEnvironmentTag(t *testing.T) {
	started := time.Unix(1000, 0).UTC()
	created := started.Add(time.Second)
	scopeFixture(t, map[int]processTableEntry{42: {PID: 42, PGID: 42}}, map[int]scopeIdentity{42: {}})
	scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
		return map[int]supervision.ProcessInfo{42: {PID: 42, StartedAt: created}}, nil
	}
	scopeEditorLeases = func(context.Context, string) ([]scenarioruntime.EditorLease, error) {
		return []scenarioruntime.EditorLease{{SessionID: "run-1", PID: 42, HostBootID: "boot-current", Scope: "vrooli-agent-claude-code-abc", Status: scenarioruntime.EditorLeaseActive, CreatedAt: created, Owner: scenarioruntime.EditorLeaseOwner{Kind: "web-console-session", Ref: "web-console-1"}}}, nil
	}
	scopeEditorScopeMembers = func(unit string) ([]int, error) {
		if unit != "vrooli-agent-claude-code-abc.scope" {
			t.Fatalf("scope unit=%q", unit)
		}
		return []int{42}, nil
	}
	ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, PGID: 42, StartedAt: &started}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
	if err != nil || report.Executors[0].State != "present" || len(report.Executors[0].PIDs) != 1 || report.Executors[0].PIDs[0] != 42 {
		t.Fatalf("attached lease failed to prove exact run: report=%+v err=%v", report, err)
	}
}

func TestExecutorScopeReregisteredLeaseUsesCurrentProcessAndScope(t *testing.T) {
	for _, tc := range []struct {
		name       string
		members    []int
		want       string
		wantReason string
	}{
		{name: "refreshed_lease_and_current_scope", members: []int{43}, want: "present"},
		{name: "refreshed_lease_pid_outside_current_scope", members: []int{44}, want: "unknown", wantReason: "leased PID is not a member of the recorded native session scope"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			path, err := scenarioruntime.DefaultDBPath(home)
			if err != nil {
				t.Fatal(err)
			}
			registeredAt := time.Unix(1000, 0).UTC()
			restartedAt := registeredAt.Add(30 * time.Second)
			fakeClock := clock.NewFake(registeredAt)
			store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: home, DBPath: path, Clock: fakeClock})
			if err != nil {
				t.Fatal(err)
			}
			original, err := store.CreateEditorLease(t.Context(), scenarioruntime.EditorLease{
				SessionID: "run-1", Harness: "claude-code", PID: 42, HostBootID: "boot-old",
				Scope: "vrooli-agent-claude-code-old", Owner: scenarioruntime.EditorLeaseOwner{Kind: "web-console-session", Ref: "web-console-1"},
			}, time.Minute)
			if err != nil {
				_ = store.Close()
				t.Fatal(err)
			}
			fakeClock.Set(restartedAt)
			refreshed, err := store.CreateEditorLease(t.Context(), scenarioruntime.EditorLease{
				SessionID: "run-1", Harness: "claude-code", PID: 43, HostBootID: "boot-new",
				Scope: "vrooli-agent-claude-code-new", Owner: scenarioruntime.EditorLeaseOwner{Kind: "web-console-session", Ref: "web-console-1"},
			}, time.Minute)
			if err != nil {
				_ = store.Close()
				t.Fatal(err)
			}
			if !refreshed.CreatedAt.Equal(original.CreatedAt) || refreshed.PID != 43 || refreshed.Scope != "vrooli-agent-claude-code-new" || refreshed.HostBootID != "boot-new" {
				_ = store.Close()
				t.Fatalf("same-session re-registration did not retain creation time while refreshing owner fields: original=%+v refreshed=%+v", original, refreshed)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}

			scopeFixture(t, map[int]processTableEntry{43: {PID: 43, PGID: 43}}, map[int]scopeIdentity{})
			scopeCurrentBootID = func() string { return "boot-new" }
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
				return map[int]supervision.ProcessInfo{43: {PID: 43, StartedAt: restartedAt}}, nil
			}
			scopeEditorLeases = func(ctx context.Context, _ string) ([]scenarioruntime.EditorLease, error) {
				return listScopeEditorLeases(ctx, home)
			}
			scopeEditorScopeMembers = func(unit string) ([]int, error) {
				if unit != "vrooli-agent-claude-code-new.scope" {
					t.Fatalf("queried stale or unexpected scope unit %q", unit)
				}
				return tc.members, nil
			}
			started := restartedAt.Add(-time.Second)
			report, err := (&Controller{Home: home}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy", StartedAt: &started}}})
			if err != nil || len(report.Executors) != 1 || report.Executors[0].State != tc.want || report.Complete != (tc.want == "present") {
				t.Fatalf("unexpected refreshed-session evidence: report=%+v err=%v", report, err)
			}
			if tc.wantReason != "" && !strings.Contains(strings.Join(report.Executors[0].Reasons, "\n"), tc.wantReason) {
				t.Fatalf("reasons=%q, want %q", report.Executors[0].Reasons, tc.wantReason)
			}
		})
	}
}

func TestExecutorScopeAttachedLeaseMismatchRemainsUnknown(t *testing.T) {
	for _, tc := range []struct {
		name        string
		boot        string
		leaseStatus string
		identity    scopeIdentity
		member      int
		changeBirth bool
		runOffset   time.Duration
		scope       string
		scopeErr    bool
		want        string
		complete    bool
		wantReason  string
	}{
		{name: "missing_scope_membership", boot: "boot-current", member: 43, want: "unknown", wantReason: "leased PID is not a member of the recorded native session scope"},
		{name: "reused_pid_changed_birth", boot: "boot-current", member: 42, changeBirth: true, want: "unknown", wantReason: "leased PID birth is unavailable or changed during sampling"},
		{name: "conflicting_environment_run", boot: "boot-current", identity: scopeIdentity{RunID: "another-run"}, member: 42, want: "unknown", wantReason: "leased process carries a conflicting run identity"},
		{name: "run_interval_mismatch", boot: "boot-current", member: 42, runOffset: 5 * time.Second, want: "unknown", wantReason: "leased PID birth is outside the recorded Agent Manager run interval"},
		{name: "unsupported_scope", boot: "boot-current", member: 42, want: "unknown", scope: "not-an-agent-session", wantReason: "attached-session lease scope is not a supported session scope"},
		{name: "scope_membership_unavailable", boot: "boot-current", member: 42, scopeErr: true, want: "unknown", wantReason: "attached-session native scope membership could not be read"},
		{name: "prior_boot_lease", boot: "boot-old", member: 42, want: "absent", complete: true},
		{name: "stopped_lease", boot: "boot-current", leaseStatus: scenarioruntime.EditorLeaseStopped, member: 42, want: "absent", complete: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			created := time.Unix(1001, 0).UTC()
			scopeFixture(t, map[int]processTableEntry{42: {PID: 42, PGID: 42}}, map[int]scopeIdentity{42: tc.identity})
			calls := 0
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
				calls++
				birth := created
				if tc.changeBirth && calls > 1 {
					birth = created.Add(time.Second)
				}
				return map[int]supervision.ProcessInfo{42: {PID: 42, StartedAt: birth}}, nil
			}
			scopeEditorLeases = func(context.Context, string) ([]scenarioruntime.EditorLease, error) {
				status := tc.leaseStatus
				if status == "" {
					status = scenarioruntime.EditorLeaseActive
				}
				scope := tc.scope
				if scope == "" {
					scope = "vrooli-agent-claude-code-abc"
				}
				return []scenarioruntime.EditorLease{{SessionID: "run-1", PID: 42, HostBootID: tc.boot, Scope: scope, Status: status, CreatedAt: created}}, nil
			}
			scopeEditorScopeMembers = func(string) ([]int, error) {
				if tc.scopeErr {
					return nil, errors.New("injected native scope error")
				}
				return []int{tc.member}, nil
			}
			pid, pgid := 42, 42
			if tc.complete {
				pid, pgid = 0, 0
			}
			ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: pid, PGID: pgid}
			if tc.runOffset != 0 {
				startedAt := created.Add(tc.runOffset)
				ref.StartedAt = &startedAt
			}
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
			if err != nil || report.Executors[0].State != tc.want || report.Complete != tc.complete {
				t.Fatalf("unsafe attached-lease evidence: report=%+v err=%v", report, err)
			}
			if tc.wantReason != "" && !strings.Contains(strings.Join(report.Executors[0].Reasons, "\n"), tc.wantReason) {
				t.Fatalf("reasons=%q, want diagnostic %q", report.Executors[0].Reasons, tc.wantReason)
			}
			if report.Executors[0].State == "absent" && len(report.Executors[0].Reasons) != 0 {
				t.Fatalf("consumer rejects absent verdict with reasons: %q", report.Executors[0].Reasons)
			}
		})
	}
}

func TestExecutorScopeMissingAttachedLeaderRequiresExactGoneScope(t *testing.T) {
	created := time.Unix(1001, 0).UTC()
	ended := created.Add(time.Minute)
	for _, tc := range []struct {
		name         string
		boot         string
		scope        string
		gone         bool
		leaseChanged bool
		goneErr      error
		ended        bool
		want         string
		wantReason   string
	}{
		{name: "ended_run_exact_scope_gone", gone: true, ended: true, want: "absent"},
		{name: "running_run_exact_scope_gone", gone: true, want: "absent"},
		{name: "prior_boot_lease", boot: "boot-old", want: "absent"},
		{name: "same_session_reattached_during_gone_proof", gone: true, leaseChanged: true, want: "unknown", wantReason: "lease changed during native scope proof"},
		{name: "empty_but_scope_exists", want: "unknown", wantReason: "not positively gone/inactive"},
		{name: "gone_proof_error", goneErr: errors.New("unit state unavailable"), want: "unknown", wantReason: "native scope gone state could not be verified"},
		{name: "missing_boot_metadata", boot: "", scope: "vrooli-agent-claude-code-abc", want: "unknown", wantReason: "missing required boot"},
		{name: "unsupported_scope", boot: "boot-current", scope: "not-an-agent-session", want: "unknown", wantReason: "not a supported session scope"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scopeFixture(t, map[int]processTableEntry{43: {PID: 43, PGID: 43}}, map[int]scopeIdentity{})
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
				return map[int]supervision.ProcessInfo{43: {PID: 43, StartedAt: created}}, nil
			}
			boot := tc.boot
			if boot == "" && tc.name != "missing_boot_metadata" {
				boot = "boot-current"
			}
			scope := tc.scope
			if scope == "" {
				scope = "vrooli-agent-claude-code-abc"
			}
			lease := scenarioruntime.EditorLease{SessionID: "run-1", PID: 42, HostBootID: boot, Scope: scope, Status: scenarioruntime.EditorLeaseActive, CreatedAt: created}
			scopeEditorLeases = func(context.Context, string) ([]scenarioruntime.EditorLease, error) {
				return []scenarioruntime.EditorLease{lease}, nil
			}
			scopeEditorLeaseBySession = func(_ context.Context, _ string, sessionID string) (scenarioruntime.EditorLease, error) {
				if sessionID != lease.SessionID {
					t.Fatalf("revalidated unexpected session %q", sessionID)
				}
				if tc.leaseChanged {
					lease.PID++
				}
				return lease, nil
			}
			scopeEditorScopeGone = func(_ context.Context, unit string) (bool, error) {
				if unit != "vrooli-agent-claude-code-abc.scope" {
					t.Fatalf("queried unexpected unit %q", unit)
				}
				return tc.gone, tc.goneErr
			}
			ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", StartedAt: ptrTime(created)}
			if tc.ended {
				ref.EndedAt = &ended
			}
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
			if err != nil || report.Executors[0].State != tc.want || report.Complete != (tc.want != "unknown") {
				t.Fatalf("missing leader classification: report=%+v err=%v", report, err)
			}
			if tc.wantReason != "" && !strings.Contains(strings.Join(report.Executors[0].Reasons, "\n"), tc.wantReason) {
				t.Fatalf("reasons=%q, want %q", report.Executors[0].Reasons, tc.wantReason)
			}
			// Agent Manager accepts absence only when no positive PID or
			// diagnostic reason accompanies the verdict.
			if report.Executors[0].State == "absent" && len(report.Executors[0].Reasons) != 0 {
				t.Fatalf("consumer rejects absent verdict with reasons: %q", report.Executors[0].Reasons)
			}
		})
	}
}

func TestScopeEditorScopeGoneUsesExactNamedCgroupAndUnitProbe(t *testing.T) {
	oldPath, oldOccupancy, oldProcesses := scopeSessionCgroupPath, scopeEditorScopeOccupancy, scopeEditorScopeProcesses
	t.Cleanup(func() {
		scopeSessionCgroupPath, scopeEditorScopeOccupancy, scopeEditorScopeProcesses = oldPath, oldOccupancy, oldProcesses
	})
	const unit = "vrooli-agent-codex-test.scope"
	const cgroup = "/user.slice/vrooli-agents.slice/" + unit
	scopeSessionCgroupPath = func(got string) (string, error) {
		if got != unit {
			t.Fatalf("resolved unexpected scope unit %q", got)
		}
		return cgroup, nil
	}
	checks := 0
	occupancyCalls := 0
	assertExactRef := func(ref platformgo.ScopeRef) {
		t.Helper()
		if ref.Kind != platformgo.ScopeKindCgroup || ref.Path != cgroup || ref.Name != "vrooli-agent-codex-test" {
			t.Fatalf("scope probe ref=%+v, want exact named cgroup %q", ref, cgroup)
		}
	}
	scopeEditorScopeOccupancy = func(ref platformgo.ScopeRef) (platformgo.Occupancy, error) {
		assertExactRef(ref)
		checks++
		occupancyCalls++
		return platformgo.Occupancy{}, os.ErrNotExist
	}
	scopeEditorScopeProcesses = func(_ context.Context, ref platformgo.ScopeRef) ([]int, error) {
		assertExactRef(ref)
		checks++
		return nil, nil // platform owner confirms the exact unit is not found/inactive
	}
	gone, err := scopeEditorScopeGone(t.Context(), unit)
	if err != nil || !gone || checks != 3 || occupancyCalls != 2 {
		t.Fatalf("exact gone proof = (%t, %v), checks=%d occupancy=%d", gone, err, checks, occupancyCalls)
	}

	// A cgroup recreated as empty between the first occupancy and unit probes
	// must not be accepted as gone.
	checks = 0
	occupancyCalls = 0
	scopeEditorScopeOccupancy = func(ref platformgo.ScopeRef) (platformgo.Occupancy, error) {
		assertExactRef(ref)
		checks++
		occupancyCalls++
		if occupancyCalls == 1 {
			return platformgo.Occupancy{}, os.ErrNotExist
		}
		return platformgo.Occupancy{}, nil
	}
	scopeEditorScopeProcesses = func(_ context.Context, ref platformgo.ScopeRef) ([]int, error) {
		assertExactRef(ref)
		checks++
		return nil, nil
	}
	gone, err = scopeEditorScopeGone(t.Context(), unit)
	if err != nil || gone || checks != 3 || occupancyCalls != 2 {
		t.Fatalf("recreated empty scope proof = (%t, %v), checks=%d occupancy=%d", gone, err, checks, occupancyCalls)
	}

	// A positive native member sample cannot be reclassified as gone, even
	// when the occupancy probe said the cgroup path was absent.
	checks = 0
	occupancyCalls = 0
	scopeEditorScopeOccupancy = func(ref platformgo.ScopeRef) (platformgo.Occupancy, error) {
		assertExactRef(ref)
		checks++
		occupancyCalls++
		return platformgo.Occupancy{}, os.ErrNotExist
	}
	scopeEditorScopeProcesses = func(_ context.Context, ref platformgo.ScopeRef) ([]int, error) {
		assertExactRef(ref)
		checks++
		return []int{43}, nil
	}
	gone, err = scopeEditorScopeGone(t.Context(), unit)
	if err != nil || gone || checks != 2 {
		t.Fatalf("surviving orphan member proof = (%t, %v), checks=%d", gone, err, checks)
	}

	// A readable cgroup with no members is still an existing scope, not a gone one.
	checks = 0
	occupancyCalls = 0
	scopeEditorScopeOccupancy = func(ref platformgo.ScopeRef) (platformgo.Occupancy, error) {
		assertExactRef(ref)
		checks++
		return platformgo.Occupancy{}, nil
	}
	scopeEditorScopeProcesses = func(context.Context, platformgo.ScopeRef) ([]int, error) {
		t.Fatal("existing empty scope must not take the gone-unit probe")
		return nil, nil
	}
	gone, err = scopeEditorScopeGone(t.Context(), unit)
	if err != nil || gone || checks != 1 {
		t.Fatalf("existing empty scope proof = (%t, %v), checks=%d", gone, err, checks)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestListScopeEditorLeasesReadsCanonicalRegistryReadOnly(t *testing.T) {
	home := t.TempDir()
	path, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		t.Fatal(err)
	}
	store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: home, DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	lease := scenarioruntime.EditorLease{SessionID: "run-1", Harness: "claude-code", PID: 42, HostBootID: "boot-current", Scope: "vrooli-agent-claude-code-abc", Owner: scenarioruntime.EditorLeaseOwner{Kind: "web-console-session", Ref: "web-console-1"}}
	if _, err := store.CreateEditorLease(t.Context(), lease, time.Minute); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	leases, err := listScopeEditorLeases(t.Context(), home)
	if err != nil || len(leases) != 1 || leases[0].SessionID != "run-1" || leases[0].Owner.Ref != "web-console-1" {
		t.Fatalf("read-only runtime lease listing: leases=%+v err=%v", leases, err)
	}
	current, err := getScopeEditorLeaseBySession(t.Context(), home, "run-1")
	if err != nil || current.SessionID != "run-1" || current.Status != scenarioruntime.EditorLeaseActive || current.PID != 42 {
		t.Fatalf("read-only exact lease revalidation: lease=%+v err=%v", current, err)
	}
}

func TestExecutorScopeHistoricalAndPhysicalEvidence(t *testing.T) {
	for _, tc := range []struct {
		name       string
		table      map[int]processTableEntry
		identities map[int]scopeIdentity
		want       string
	}{
		{"exited_historical_root_and_group", map[int]processTableEntry{}, nil, "absent"},
		{"numeric_pid_is_not_identity", map[int]processTableEntry{10: {PID: 10, PGID: 10}}, nil, "unknown"},
		{"leader_exited_group_survives", map[int]processTableEntry{20: {PID: 20, PGID: 10}}, nil, "unknown"},
		{"terminal_run_still_executes", map[int]processTableEntry{10: {PID: 10}}, map[int]scopeIdentity{10: {RunID: "run-1"}}, "present"},
		{"detached_tagged_descendant", map[int]processTableEntry{20: {PID: 20, PPID: 1, PGID: 20}}, map[int]scopeIdentity{20: {Tags: []string{"current"}}}, "present"},
		{"detached_legacy_descendant", map[int]processTableEntry{20: {PID: 20, PPID: 1, PGID: 20}}, map[int]scopeIdentity{20: {Tags: []string{"legacy"}}}, "present"},
		{"zombie_cannot_execute", map[int]processTableEntry{10: {PID: 10, State: "Z"}}, map[int]scopeIdentity{10: {RunID: "run-1"}}, "absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scopeFixture(t, tc.table, tc.identities)
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 10, PGID: 10}}})
			if err != nil || len(report.Executors) != 1 || report.Executors[0].State != tc.want || report.Complete != (tc.want != "unknown") {
				t.Fatalf("evidence=%+v err=%v", report, err)
			}
		})
	}
}

func TestExecutorScopePIDReuseRequiresStableBirthAndNoMatchingLabels(t *testing.T) {
	start, end := time.Unix(1000, 0), time.Unix(2000, 0)
	for _, tc := range []struct {
		name                   string
		birth                  time.Time
		change, member, tagged bool
		want                   string
	}{
		{"new_pid_after_run", time.Unix(3000, 0), false, false, false, "absent"},
		{"preexisting_pid", time.Unix(500, 0), false, false, false, "absent"},
		{"possible_original_pid", time.Unix(1500, 0), false, false, false, "unknown"},
		{"birth_changed_midread", time.Unix(3000, 0), true, false, false, "unknown"},
		{"new_child_not_group_reuse", time.Unix(3000, 0), false, true, false, "unknown"},
		{"late_tag_still_positive", time.Unix(3000, 0), false, false, true, "present"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pid := 10
			if tc.member {
				pid = 20
			}
			ids := map[int]scopeIdentity{}
			if tc.tagged {
				ids[pid] = scopeIdentity{Tags: []string{"current"}}
			}
			scopeFixture(t, map[int]processTableEntry{pid: {PID: pid, PGID: 10}}, ids)
			calls := 0
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
				calls++
				birth := tc.birth
				if tc.change && calls > 1 {
					birth = birth.Add(time.Second)
				}
				return map[int]supervision.ProcessInfo{pid: {PID: pid, StartedAt: birth}}, nil
			}
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 10, PGID: 10, StartedAt: &start, EndedAt: &end}}})
			if err != nil || report.Executors[0].State != tc.want {
				t.Fatalf("report=%+v err=%v", report, err)
			}
		})
	}
}

func TestExecutorScopeUnknownIsNotEmptyAndDoesNotExposeSecrets(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{20: {PID: 20, PGID: 20}}, nil)
	scopeReadIdentity = func(processTableEntry) (scopeIdentity, error) {
		return scopeIdentity{}, errors.New("SECRET_ENVIRONMENT=value")
	}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 10}}})
	data, _ := json.Marshal(report)
	if err != nil || report.Complete || report.Executors[0].State != "unknown" || strings.Contains(string(data), "SECRET") {
		t.Fatalf("report=%s err=%v", data, err)
	}
}

func TestExecutorScopeBoundsAndMidReadBirth(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{}, nil)
	for _, refs := range [][]ExecutorScopeRef{nil, make([]ExecutorScopeRef, ExecutorScopeMaxReferences+1), {{RunID: "run-1"}}, {{RunID: "run-1", Tag: "tag", LegacyTag: "legacy", PID: -1}}} {
		if _, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: refs}); err == nil {
			t.Fatal("invalid request accepted")
		}
	}
	calls := 0
	scopeProcessTable = func(context.Context) (map[int]processTableEntry, error) {
		calls++
		if calls == 1 {
			return map[int]processTableEntry{1: {PID: 1}}, nil
		}
		return map[int]processTableEntry{1: {PID: 1}, 20: {PID: 20}}, nil
	}
	scopeReadIdentity = func(entry processTableEntry) (scopeIdentity, error) {
		if entry.PID == 20 {
			return scopeIdentity{RunID: "run-1"}, nil
		}
		return scopeIdentity{}, nil
	}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "tag", LegacyTag: "legacy"}}})
	if err != nil || report.Executors[0].State != "present" {
		t.Fatalf("birth escaped exclusion: %+v %v", report, err)
	}
}

func TestExecutorScopeIgnoresUnrelatedHostProcessChurn(t *testing.T) {
	oldTable, oldIdentity := scopeProcessTable, scopeReadIdentity
	t.Cleanup(func() { scopeProcessTable, scopeReadIdentity = oldTable, oldIdentity })
	calls := 0
	scopeProcessTable = func(context.Context) (map[int]processTableEntry, error) {
		calls++
		// A changing unrelated process must not make an otherwise provable
		// historical executor impossible to exclude.
		return map[int]processTableEntry{100 + calls: {PID: 100 + calls, PGID: 100 + calls}}, nil
	}
	scopeReadIdentity = func(processTableEntry) (scopeIdentity, error) { return scopeIdentity{}, nil }
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 10, PGID: 10}}})
	if err != nil || !report.Complete || report.Executors[0].State != "absent" {
		t.Fatalf("unrelated process churn blocked exclusion: report=%+v err=%v", report, err)
	}
}

func TestExecutorScopeBatchSharesOneBoundedObservation(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{1: {PID: 1}}, nil)
	reads := 0
	scopeReadIdentity = func(processTableEntry) (scopeIdentity, error) { reads++; return scopeIdentity{}, nil }
	refs := make([]ExecutorScopeRef, 128)
	for i := range refs {
		refs[i] = ExecutorScopeRef{RunID: fmt.Sprintf("run-%d", i), Tag: "tag", LegacyTag: "legacy"}
	}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: refs})
	if err != nil || !report.Complete || len(report.Executors) != 128 || reads != 2 {
		t.Fatalf("reads=%d report=%+v err=%v", reads, report, err)
	}
}

func TestExecutorScopeCancellationCannotProveEmptyHost(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{}, nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	report, err := (&Controller{}).ExecutorScope(ctx, ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "tag", LegacyTag: "legacy"}}})
	if err != nil || report.Complete || report.Executors[0].State != "unknown" {
		t.Fatalf("cancellation manufactured absence: %+v %v", report, err)
	}
}

func TestExecutorScopeManagedHistoryCapacity(t *testing.T) {
	for _, count := range []int{6000, 16384} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			refs := make([]ExecutorScopeRef, count)
			for i := range refs {
				refs[i] = ExecutorScopeRef{RunID: fmt.Sprintf("managed-run-%d", i), Tag: fmt.Sprintf("tag-%d", i), LegacyTag: fmt.Sprintf("legacy-%d", i), PID: 100000 + i, PGID: 100000 + i}
			}
			table, identities := map[int]processTableEntry{}, map[int]scopeIdentity{}
			for pid := 1; pid <= 6000; pid++ {
				table[pid] = processTableEntry{PID: pid, PGID: pid}
				identities[pid] = scopeIdentity{RunID: fmt.Sprintf("unrelated-%d", pid)}
			}
			// A detached terminal executor at the tail must not disappear behind
			// historical count caps; a surviving unlabelled group stays unknown.
			table[90000] = processTableEntry{PID: 90000, PGID: 90000}
			identities[90000] = scopeIdentity{Tags: []string{refs[count-1].LegacyTag}}
			table[90001] = processTableEntry{PID: 90001, PGID: refs[count-2].PGID}
			scopeFixture(t, table, identities)
			reads := 0
			scopeReadIdentity = func(entry processTableEntry) (scopeIdentity, error) { reads++; return identities[entry.PID], nil }
			started := time.Now()
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: refs})
			if err != nil || len(report.Executors) != count {
				t.Fatalf("normal managed history rejected: count=%d result=%d err=%v", count, len(report.Executors), err)
			}
			for i, evidence := range report.Executors {
				want := "absent"
				if i == count-2 {
					want = "unknown"
				}
				if i == count-1 {
					want = "present"
				}
				if evidence.RunID != refs[i].RunID || evidence.State != want {
					t.Fatalf("lost history/physical evidence at %d: %+v want=%s", i, evidence, want)
				}
			}
			if report.Complete || reads != 2*len(table) {
				t.Fatalf("incomplete scope accepted or per-reference scans: complete=%t reads=%d processes=%d", report.Complete, reads, len(table))
			}
			t.Logf("%d references / %d host processes / %d identity reads in %s", count, len(table), reads, time.Since(started))
		})
	}
}

func TestExecutorScopeSharedLabelsBoundResponseWithoutDroppingRuns(t *testing.T) {
	refs := make([]ExecutorScopeRef, ExecutorScopeMaxReferences)
	for i := range refs {
		refs[i] = ExecutorScopeRef{RunID: fmt.Sprintf("managed-%d", i), Tag: "shared", LegacyTag: "shared"}
	}
	table, identities := map[int]processTableEntry{}, map[int]scopeIdentity{}
	for pid := 1; pid <= 6000; pid++ {
		table[pid] = processTableEntry{PID: pid, PGID: pid}
		identities[pid] = scopeIdentity{Tags: []string{"shared", "shared"}}
	}
	scopeFixture(t, table, identities)
	start := time.Now()
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: refs})
	if err != nil || !report.Complete || len(report.Executors) != len(refs) {
		t.Fatalf("shared label inventory incomplete: count=%d complete=%t err=%v", len(report.Executors), report.Complete, err)
	}
	samples := 0
	for i, evidence := range report.Executors {
		if evidence.RunID != refs[i].RunID || evidence.State != "present" || len(evidence.PIDs) == 0 {
			t.Fatalf("shared positive lost at %d: %+v", i, evidence)
		}
		samples += len(evidence.PIDs)
	}
	data, err := json.Marshal(report)
	if err != nil || len(data) > ExecutorScopeMaxBytes || samples > executorScopeSampleBudget {
		t.Fatalf("response amplification: bytes=%d samples=%d err=%v", len(data), samples, err)
	}
	t.Logf("%d references / 6000 shared-tag processes: %s, %d response bytes, %d PID samples", len(refs), time.Since(start), len(data), samples)
}

func TestRuntimeScopeAbsentPreservesIndependentExecutor(t *testing.T) {
	for _, tc := range []struct {
		name     string
		entry    processTableEntry
		identity scopeIdentity
		listener network.TCPListenerSnapshot
		wantErr  bool
	}{
		{"only_independent_executor", processTableEntry{PID: 20, PGID: 20}, scopeIdentity{RunID: "run-1", InstanceID: "old"}, network.TCPListenerSnapshot{Known: true}, false},
		{"service_identity_survives", processTableEntry{PID: 20, PGID: 20}, scopeIdentity{InstanceID: "old"}, network.TCPListenerSnapshot{Known: true}, true},
		{"recorded_service_group_survives", processTableEntry{PID: 20, PGID: 10}, scopeIdentity{RunID: "run-1"}, network.TCPListenerSnapshot{Known: true}, true},
		{"listener_unknown", processTableEntry{PID: 20, PGID: 20}, scopeIdentity{RunID: "run-1"}, network.TCPListenerSnapshot{}, true},
		{"listener_survives", processTableEntry{PID: 20, PGID: 20}, scopeIdentity{RunID: "run-1"}, network.TCPListenerSnapshot{Known: true, Ports: map[int][]network.SnapshotListener{12345: nil}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scopeFixture(t, map[int]processTableEntry{20: tc.entry}, map[int]scopeIdentity{20: tc.identity})
			scopeListeners = func() network.TCPListenerSnapshot { return tc.listener }
			err := (&Controller{}).RequireRuntimeScopeAbsent(t.Context(), RuntimeScopeRef{Scenario: "agent-manager", Variant: "live", InstanceIDs: []string{"old"}, PIDs: []int{10}, PGIDs: []int{10}, Ports: []int{12345}})
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
