package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/supervision"
)

func scopeFixture(t *testing.T, table map[int]processTableEntry, identities map[int]scopeIdentity) {
	t.Helper()
	oldTable, oldIdentity, oldBirths, oldListeners := scopeProcessTable, scopeReadIdentity, scopeProcessBirths, scopeListeners
	t.Cleanup(func() {
		scopeProcessTable, scopeReadIdentity, scopeProcessBirths, scopeListeners = oldTable, oldIdentity, oldBirths, oldListeners
	})
	scopeProcessTable = func(context.Context) (map[int]processTableEntry, error) { return table, nil }
	scopeReadIdentity = func(entry processTableEntry) (scopeIdentity, error) { return identities[entry.PID], nil }
	scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
		return map[int]supervision.ProcessInfo{}, nil
	}
	scopeListeners = func() network.TCPListenerSnapshot { return network.TCPListenerSnapshot{Known: true} }
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
