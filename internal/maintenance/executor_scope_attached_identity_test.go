package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

func TestReadScopeGoneChecksBatchExactUnitsAndRetainReappearingScopes(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{}, map[int]scopeIdentity{})
	scopeSessionCgroupPath = func(unit string) (string, error) {
		return "/user.slice/user-1000.slice/vrooli-agents.slice/" + unit, nil
	}
	occupancyCalls := map[string]int{}
	scopeEditorScopeOccupancy = func(ref platformgo.ScopeRef) (platformgo.Occupancy, error) {
		occupancyCalls[ref.Name]++
		if ref.Name == "live" {
			return platformgo.Occupancy{}, nil
		}
		return platformgo.Occupancy{}, os.ErrNotExist
	}
	var batched []platformgo.ScopeRef
	scopeEditorScopeProcessesBatch = func(_ context.Context, refs []platformgo.ScopeRef) []platformgo.ScopeProcessObservation {
		batched = append([]platformgo.ScopeRef(nil), refs...)
		return []platformgo.ScopeProcessObservation{{}, {PIDs: []int{77}}}
	}
	results := readScopeGoneChecks(t.Context(), []scopeGoneCheck{{runID: "live-run", unit: "live.scope"}, {runID: "gone-run", unit: "gone.scope"}, {runID: "reappeared-run", unit: "reappeared.scope"}})
	if len(batched) != 2 || batched[0].Name != "gone" || batched[1].Name != "reappeared" {
		t.Fatalf("missing exact scopes were not batched in stable order: %+v", batched)
	}
	if results["live-run"].gone || results["live-run"].err != nil {
		t.Fatalf("present scope did not remain present: %+v", results["live-run"])
	}
	if !results["gone-run"].gone || results["gone-run"].err != nil || occupancyCalls["gone"] != 2 {
		t.Fatalf("gone scope lacked exact final absence recheck: result=%+v occupancy=%d", results["gone-run"], occupancyCalls["gone"])
	}
	if results["reappeared-run"].gone || results["reappeared-run"].err != nil {
		t.Fatalf("scope with a live member was accepted as gone: %+v", results["reappeared-run"])
	}
}

func TestExecutorScopeProvesExactScopedChildOfAttachedLauncher(t *testing.T) {
	for _, tc := range []struct {
		name          string
		legacy        bool
		leaseAlias    bool
		badProvenance bool
		wantState     string
	}{
		{name: "schema16_binding_proves_spawn_and_wait_child", wantState: "present"},
		{name: "legacy_owner_proof_proves_live_child_only", legacy: true, wantState: "present"},
		{name: "legacy_run_id_fallback_accepts_exact_proof_lease_alias", legacy: true, leaseAlias: true, wantState: "present"},
		{name: "binding_evidence_mismatch_stays_unknown", badProvenance: true, wantState: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			boot := uuid.NewString()
			unit := "vrooli-agent-claude-session"
			scopePath := "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/" + unit + ".scope"
			wrapperCgroup := "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/tmux-spawn-test.scope"
			started := time.Unix(1000, 0).UTC()
			wrapperStart, childStart := "1001", "1002"
			wrapperPID, childPID := 42, 43
			scopeFixture(t, map[int]processTableEntry{
				wrapperPID: {PID: wrapperPID, PPID: 1, PGID: wrapperPID, State: "S", Cgroup: wrapperCgroup},
				childPID:   {PID: childPID, PPID: wrapperPID, PGID: wrapperPID, State: "S", Cgroup: "0::" + scopePath},
			}, map[int]scopeIdentity{})
			scopeCurrentBootID = func() string { return boot }
			birth := started.Add(time.Second)
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
				return map[int]supervision.ProcessInfo{
					wrapperPID: {PID: wrapperPID, StartedAt: birth},
					childPID:   {PID: childPID, StartedAt: birth.Add(time.Second)},
				}, nil
			}
			scopeReadProcessBinding = func(pid int) (cliutil.ProcessBinding, error) {
				switch pid {
				case wrapperPID:
					return cliutil.ProcessBinding{PID: pid, UID: 1000, Start: wrapperStart, Boot: boot, Cgroup: wrapperCgroup}, nil
				case childPID:
					return cliutil.ProcessBinding{PID: pid, UID: 1000, Start: childStart, Boot: boot, Cgroup: "0::" + scopePath}, nil
				default:
					return cliutil.ProcessBinding{}, errors.New("unknown pid")
				}
			}
			scopeReadProcessFact = func(pid int) (agentsession.ProcessFact, error) {
				switch pid {
				case wrapperPID:
					return agentsession.ProcessFact{PID: pid, PPID: 1, StartTime: wrapperStart}, nil
				case childPID:
					return agentsession.ProcessFact{PID: pid, PPID: wrapperPID, StartTime: childStart}, nil
				default:
					return agentsession.ProcessFact{}, errors.New("unknown pid")
				}
			}
			scopeEditorScopeMembers = func(got string) ([]int, error) {
				if got != unit+".scope" {
					t.Fatalf("scope members queried for %q", got)
				}
				return []int{childPID}, nil
			}
			scopeSessionCgroupPath = func(got string) (string, error) {
				if got != unit+".scope" {
					t.Fatalf("unexpected session unit %q", got)
				}
				return scopePath, nil
			}

			ref := ExecutorScopeRef{
				RunID: "run-1", Tag: "attached-tag", LegacyTag: "legacy-tag", HarnessSessionID: "claude-session", HarnessKind: "claude-code", StartedAt: &started,
			}
			proof := cliutil.AgentSessionBindingProof{
				ProofVersion: cliutil.AgentSessionBindingProofVersion, RunID: ref.RunID,
				HarnessSessionID: ref.HarnessSessionID, LeaseSessionID: ref.HarnessSessionID,
				WebConsoleSessionID: "web-session-1", HarnessKind: ref.HarnessKind, OwnerKind: "web-console-session",
				Binding: cliutil.ProcessBinding{PID: wrapperPID, UID: 1000, Start: wrapperStart, Boot: boot, Cgroup: wrapperCgroup},
			}
			ref.BindingProof = &proof
			lease := scenarioruntime.EditorLease{
				BindingID: "binding-1", SessionID: ref.HarnessSessionID, Harness: ref.HarnessKind,
				PID: wrapperPID, HostBootID: boot, Scope: unit, Status: scenarioruntime.EditorLeaseActive,
				Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindWebConsoleSession, Ref: proof.WebConsoleSessionID, Source: agentsession.SourceLaunch, EvidenceJSON: `{"read_from_pid":42,"web_console_session_id":"web-session-1"}`},
			}
			if tc.legacy {
				lease.BindingID = ""
			}
			if tc.leaseAlias {
				proof.LeaseSessionID = "legacy-lease-alias"
				lease.SessionID = proof.LeaseSessionID
			}
			var provenance scenarioruntime.EditorLeaseProvenance
			if !tc.legacy {
				provenance = scenarioruntime.EditorLeaseProvenance{
					BindingID: lease.BindingID, SessionID: lease.SessionID, State: scenarioruntime.EditorBindingActive,
					ContainmentState: scenarioruntime.EditorContainmentEstablished, IntendedScope: lease.Scope,
					Scope: "cgroup:" + scopePath, Method: "systemd-run", PID: wrapperPID, UID: 1000,
					HostBootID: boot, ProcessStart: wrapperStart, EvidencePID: childPID, EvidenceUID: 1000,
					EvidenceBootID: boot, EvidenceProcessStart: childStart,
				}
				if tc.badProvenance {
					provenance.EvidencePID = childPID + 1
				}
			}
			scopeEditorLeaseBindings = func(context.Context, string, []string) ([]scenarioruntime.EditorLease, error) {
				return []scenarioruntime.EditorLease{lease}, nil
			}
			scopeEditorLeaseProvenanceBatch = func(_ context.Context, _ string, ids []string) (map[string]scenarioruntime.EditorLeaseProvenance, map[string]error) {
				if tc.legacy {
					return map[string]scenarioruntime.EditorLeaseProvenance{}, map[string]error{}
				}
				return map[string]scenarioruntime.EditorLeaseProvenance{lease.BindingID: provenance}, map[string]error{}
			}
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
			if err != nil || len(report.Executors) != 1 || report.Executors[0].State != tc.wantState || report.Complete != (tc.wantState == "present") {
				t.Fatalf("scoped child proof report=%+v err=%v", report, err)
			}
			if tc.wantState == "present" && (len(report.Executors[0].PIDs) != 1 || report.Executors[0].PIDs[0] != childPID) {
				t.Fatalf("exact contained executor root not returned: %+v", report.Executors[0])
			}
		})
	}
}

// Given an attached run whose immutable harness key differs from its run UUID,
// when maintenance inspects physical execution, then the exact native binding
// is present. A missing proof must never be converted into an absent executor.
func TestExecutorScopeAttachedHarnessIdentity(t *testing.T) {
	for _, tc := range []struct {
		name                                                                                                                         string
		change                                                                                                                       func(*ExecutorScopeRef, *scenarioruntime.EditorLease)
		legacy, duplicate, registryError, missing, rebind, ownerChange, readError, finalBirthChange, scopeError, missingMember, gone bool
		want, reason                                                                                                                 string
	}{
		{name: "canonical_launcher_key_proves_shared_backend_present", want: "present"},
		{name: "missing_canonical_key_allows_exact_legacy_key", legacy: true, want: "present"},
		{name: "canonical_wrong_pid_never_falls_back_to_valid_legacy", change: func(_ *ExecutorScopeRef, l *scenarioruntime.EditorLease) { l.PID = 43 }, legacy: true, want: "unknown", reason: "persisted PID or harness"},
		{name: "canonical_wrong_harness_never_falls_back", change: func(_ *ExecutorScopeRef, l *scenarioruntime.EditorLease) { l.Harness = "codex" }, legacy: true, want: "unknown", reason: "persisted PID or harness"},
		{name: "canonical_prior_boot_is_not_current_binding", change: func(_ *ExecutorScopeRef, l *scenarioruntime.EditorLease) { l.HostBootID = "boot-old" }, want: "unknown", reason: "lacks run identity or reuse proof"},
		{name: "canonical_stopped_lease_with_live_member_is_present", change: func(_ *ExecutorScopeRef, l *scenarioruntime.EditorLease) {
			l.Status = scenarioruntime.EditorLeaseStopped
		}, want: "present"},
		{name: "current_native_presence_does_not_invent_lease_time", change: func(_ *ExecutorScopeRef, l *scenarioruntime.EditorLease) { l.CreatedAt = time.Time{} }, want: "present"},
		{name: "legitimate_reattach_preserves_session_creation_time", change: func(_ *ExecutorScopeRef, l *scenarioruntime.EditorLease) { l.CreatedAt = l.CreatedAt.Add(-time.Second) }, want: "present"},
		{name: "unavailable_registry_is_not_absence", registryError: true, want: "unknown", reason: "registry unavailable"},
		{name: "missing_exact_lease_is_not_absence", missing: true, want: "unknown", reason: "could not be resolved"},
		{name: "duplicate_canonical_key_is_conflicting", duplicate: true, want: "unknown", reason: "could not be resolved"},
		{name: "scope_unavailable_is_not_absence", scopeError: true, want: "unknown", reason: "membership could not be read"},
		{name: "pid_outside_exact_native_scope_is_unknown", missingMember: true, want: "unknown", reason: "outside the recorded native session scope"},
		{name: "lease_rebind_during_scope_proof_is_unknown", rebind: true, want: "unknown", reason: "changed or became unavailable"},
		{name: "owner_rebind_during_scope_proof_is_unknown", ownerChange: true, want: "unknown", reason: "changed or became unavailable"},
		{name: "lease_reread_failure_is_unknown", readError: true, want: "unknown", reason: "changed or became unavailable"},
		{name: "pid_reused_during_scope_proof_is_unknown", finalBirthChange: true, want: "unknown", reason: "birth changed or became unavailable"},
		{name: "planned_scope_gone_cannot_prove_historical_containment", gone: true, want: "unknown", reason: "containment provenance missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := time.Unix(1000, 0).UTC()
			created := started.Add(time.Second)
			table := map[int]processTableEntry{42: {PID: 42, PGID: 40}}
			if tc.gone {
				table = map[int]processTableEntry{}
			}
			scopeFixture(t, table, map[int]scopeIdentity{})
			birthReads := 0
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
				birthReads++
				birth := created
				if tc.finalBirthChange && birthReads > 2 {
					birth = birth.Add(time.Second)
				}
				return map[int]supervision.ProcessInfo{42: {PID: 42, StartedAt: birth}}, nil
			}
			ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, StartedAt: &started, HarnessSessionID: "opencode-abc", HarnessKind: "opencode"}
			lease := scenarioruntime.EditorLease{SessionID: "opencode-abc", Harness: "opencode", PID: 42, HostBootID: "boot-current", Scope: "vrooli-agent-opencode-abc", Status: scenarioruntime.EditorLeaseActive, CreatedAt: created, Owner: scenarioruntime.EditorLeaseOwner{Kind: "web-console-session", Ref: "web-session-1", Source: "launch", EvidenceJSON: `{"read_from_pid":42}`}}
			legacy := lease
			legacy.SessionID = ref.RunID
			if tc.change != nil {
				tc.change(&ref, &lease)
			}
			rows := []scenarioruntime.EditorLease{lease}
			if tc.legacy {
				if tc.change == nil {
					rows = nil
				}
				rows = append(rows, legacy)
			}
			if tc.duplicate {
				rows = append(rows, lease)
			}
			if tc.missing {
				rows = nil
			}
			leaseReads := 0
			scopeEditorLeaseBindings = func(context.Context, string, []string) ([]scenarioruntime.EditorLease, error) {
				leaseReads++
				if tc.registryError {
					return nil, errors.New("registry unavailable: permission denied")
				}
				if tc.readError && leaseReads > 1 {
					return nil, errors.New("read unavailable")
				}
				currentRows := append([]scenarioruntime.EditorLease(nil), rows...)
				if leaseReads > 1 {
					for i := range currentRows {
						if tc.rebind {
							currentRows[i].PID = 43
						}
						if tc.ownerChange {
							currentRows[i].Owner.Ref = "other-instance"
						}
					}
				}
				return currentRows, nil
			}
			scopeEditorScopeMembers = func(unit string) ([]int, error) {
				if unit != "vrooli-agent-opencode-abc.scope" {
					t.Fatalf("unexpected native scope %q", unit)
				}
				if tc.scopeError {
					return nil, errors.New("scope unavailable")
				}
				if tc.missingMember {
					return []int{43}, nil
				}
				return []int{42}, nil
			}
			scopeEditorScopeGone = func(context.Context, string) (bool, error) { return tc.gone, nil }
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
			if err != nil || len(report.Executors) != 1 || report.Executors[0].State != tc.want || report.Complete != (tc.want != "unknown") {
				t.Fatalf("report=%+v err=%v", report, err)
			}
			if report.IdentityVersion != "attached-key-v1" {
				t.Fatalf("missing identity protocol capability: %+v", report)
			}
			evidence := report.Executors[0]
			if tc.reason != "" && !strings.Contains(strings.Join(evidence.Reasons, "\n"), tc.reason) {
				t.Fatalf("reasons %q lack %q", evidence.Reasons, tc.reason)
			}
			if tc.registryError && !strings.Contains(strings.Join(evidence.Reasons, " "), "permission denied") {
				t.Fatalf("lease registry cause was discarded: %+v", evidence)
			}
			if tc.gone && !tc.legacy {
				reason := strings.Join(evidence.Reasons, " ")
				for _, want := range []string{`"owner":"web-console-session:web-session-1"`, `"session":"opencode-abc"`, `"containment":"missing"`, `"scope":"vrooli-agent-opencode-abc"`} {
					if !strings.Contains(reason, want) {
						t.Fatalf("attached owner diagnostic omitted %q: %s", want, reason)
					}
				}
				if evidence.State != "unknown" {
					t.Fatalf("missing containment provenance must remain unknown: %+v", evidence)
				}
			}
			if tc.want == "present" && (len(evidence.PIDs) != 1 || evidence.PIDs[0] != 42) {
				t.Fatalf("missing positive exact PID: %+v", evidence)
			}
			if tc.want == "absent" && (len(evidence.PIDs) != 0 || len(evidence.Reasons) != 0) {
				t.Fatalf("invalid absence: %+v", evidence)
			}
		})
	}
}

func TestExecutorScopeGoneLeaderRequiresExactEstablishedContainment(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*scenarioruntime.EditorLeaseProvenance)
		missing bool
		want    string
	}{
		{name: "exact_established_scope_proves_gone", want: "absent"},
		{name: "reused_process_identity_is_unknown", change: func(p *scenarioruntime.EditorLeaseProvenance) { p.EvidenceBootID = "old-boot" }, want: "unknown"},
		{name: "wrong_actual_scope_is_unknown", change: func(p *scenarioruntime.EditorLeaseProvenance) {
			p.Scope = "cgroup:/user.slice/vrooli-agents.slice/vrooli-agent-opencode-other.scope"
		}, want: "unknown"},
		{name: "wrong_binding_scope_is_unknown", change: func(p *scenarioruntime.EditorLeaseProvenance) { p.IntendedScope = "vrooli-agent-opencode-other" }, want: "unknown"},
		{name: "missing_provenance_is_unknown", missing: true, want: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := time.Unix(1000, 0).UTC()
			created := started.Add(time.Second)
			scopeFixture(t, map[int]processTableEntry{}, map[int]scopeIdentity{})
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
				return map[int]supervision.ProcessInfo{}, nil
			}
			ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, StartedAt: &started, HarnessSessionID: "opencode-abc", HarnessKind: "opencode"}
			lease := scenarioruntime.EditorLease{SessionID: "opencode-abc", Harness: "opencode", PID: 42, HostBootID: "boot-current", Scope: "vrooli-agent-opencode-abc", BindingID: "binding-1", Status: scenarioruntime.EditorLeaseActive, CreatedAt: created}
			scopeEditorLeaseBindings = func(context.Context, string, []string) ([]scenarioruntime.EditorLease, error) {
				return []scenarioruntime.EditorLease{lease}, nil
			}
			provenance := scenarioruntime.EditorLeaseProvenance{
				BindingID: "binding-1", SessionID: lease.SessionID, PID: lease.PID, UID: 1000,
				HostBootID: "boot-current", ProcessStart: "1001", IntendedScope: lease.Scope,
				ContainmentState: scenarioruntime.EditorContainmentEstablished, Scope: "cgroup:/user.slice/vrooli-agents.slice/vrooli-agent-opencode-abc.scope",
				Method: "systemd", EvidencePID: 43, EvidenceUID: 1000, EvidenceBootID: "boot-current", EvidenceProcessStart: "1002",
			}
			if tc.change != nil {
				tc.change(&provenance)
			}
			scopeEditorLeaseProvenance = func(_ context.Context, _ string, id string) (scenarioruntime.EditorLeaseProvenance, error) {
				if id != lease.BindingID {
					t.Fatalf("provenance lookup binding id=%q", id)
				}
				if tc.missing {
					return scenarioruntime.EditorLeaseProvenance{}, scenarioruntime.ErrNotFound
				}
				return provenance, nil
			}
			scopeEditorScopeGone = func(_ context.Context, unit string) (bool, error) {
				if unit != "vrooli-agent-opencode-abc.scope" {
					t.Fatalf("scope-gone probe used unverified scope %q", unit)
				}
				return true, nil
			}
			scopeSessionCgroupPath = func(unit string) (string, error) {
				if unit != "vrooli-agent-opencode-abc.scope" {
					t.Fatalf("canonical path requested for unverified unit %q", unit)
				}
				return "/user.slice/vrooli-agents.slice/vrooli-agent-opencode-abc.scope", nil
			}
			scopeEditorScopeMembers = func(string) ([]int, error) { return nil, nil }
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
			if err != nil || len(report.Executors) != 1 || report.Executors[0].State != tc.want || report.Complete != (tc.want == "absent") {
				t.Fatalf("report=%+v err=%v", report, err)
			}
		})
	}
}

func TestExecutorScopePartialHostSnapshotCannotProveAttachedLeaderGone(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{}, map[int]scopeIdentity{})
	ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, HarnessSessionID: "claude-session", HarnessKind: "claude-code"}
	lease := scenarioruntime.EditorLease{
		SessionID: ref.HarnessSessionID, Harness: ref.HarnessKind, PID: ref.PID,
		HostBootID: "boot-current", Scope: "vrooli-agent-claude-session", Status: scenarioruntime.EditorLeaseActive,
		BindingID: "binding-1", CreatedAt: time.Unix(1000, 0).UTC(),
		Owner: scenarioruntime.EditorLeaseOwner{Kind: "web-console-session", Ref: "web-session-1"},
	}
	scopeEditorLeaseBindings = func(context.Context, string, []string) ([]scenarioruntime.EditorLease, error) {
		return []scenarioruntime.EditorLease{lease}, nil
	}
	before := map[int]supervision.ProcessInfo{42: {PID: 42, StartedAt: time.Unix(1000, 0).UTC()}}
	matched, uncertain, reasons, err := (&Controller{}).attachedExecutorLeases(
		t.Context(), []ExecutorScopeRef{ref}, loadScopeLeasePage(t.Context(), "", []ExecutorScopeRef{ref}), map[int]processTableEntry{}, map[int]scopeIdentity{}, map[int]error{}, before, before,
		nil, nil, errors.New("process identity unavailable"),
	)
	if err != nil || matched[ref.RunID] != 0 || !uncertain[ref.RunID] || !strings.Contains(reasons[ref.RunID], "host process identity inventory is incomplete") || !strings.Contains(reasons[ref.RunID], "web-session-1 and its tmux pane") {
		t.Fatalf("partial host snapshot was mistaken for leader absence: matched=%v uncertain=%v reasons=%v err=%v", matched, uncertain, reasons, err)
	}
}

func TestExecutorScopeScopesUnreadableHostIdentitiesToAffectedPID(t *testing.T) {
	started := time.Unix(1000, 0).UTC()
	created := started.Add(time.Second)
	ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, PGID: 42, StartedAt: &started, HarnessSessionID: "opencode-abc", HarnessKind: "opencode"}
	lease := scenarioruntime.EditorLease{
		SessionID: ref.HarnessSessionID, Harness: ref.HarnessKind, PID: ref.PID,
		HostBootID: "boot-current", Scope: "vrooli-agent-opencode-abc", Status: scenarioruntime.EditorLeaseActive,
		BindingID: "binding-1", CreatedAt: created,
	}
	provenance := scenarioruntime.EditorLeaseProvenance{
		BindingID: lease.BindingID, SessionID: lease.SessionID, PID: lease.PID, UID: 1000,
		HostBootID: lease.HostBootID, ProcessStart: "1001", IntendedScope: lease.Scope,
		ContainmentState: scenarioruntime.EditorContainmentEstablished,
		Scope:            "cgroup:/user.slice/vrooli-agents.slice/vrooli-agent-opencode-abc.scope",
		Method:           "systemd", EvidencePID: 43, EvidenceUID: 1000, EvidenceBootID: "boot-current", EvidenceProcessStart: "1002",
	}

	for _, tc := range []struct {
		name         string
		pid          int
		want         string
		wantComplete bool
	}{
		{name: "unrelated_unreadable_pid_does_not_block_exact_gone_proof", pid: 99, want: "absent", wantComplete: true},
		{name: "unreadable_attached_pid_remains_unknown", pid: 42, want: "unknown", wantComplete: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			startAt := started
			table := map[int]processTableEntry{tc.pid: {PID: tc.pid, PGID: tc.pid}}
			births := map[int]supervision.ProcessInfo{}
			if tc.pid == ref.PID {
				births[tc.pid] = supervision.ProcessInfo{PID: tc.pid, StartedAt: created}
			}
			scopeFixture(t, table, map[int]scopeIdentity{})
			scopeReadIdentity = func(entry processTableEntry) (scopeIdentity, error) {
				if entry.PID == tc.pid {
					return scopeIdentity{}, errors.New("secret environment read failure")
				}
				return scopeIdentity{}, nil
			}
			scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) { return births, nil }
			scopeEditorLeaseBindings = func(context.Context, string, []string) ([]scenarioruntime.EditorLease, error) {
				return []scenarioruntime.EditorLease{lease}, nil
			}
			scopeEditorLeaseProvenance = func(_ context.Context, _ string, bindingID string) (scenarioruntime.EditorLeaseProvenance, error) {
				if bindingID != lease.BindingID {
					t.Fatalf("unexpected binding provenance lookup %q", bindingID)
				}
				return provenance, nil
			}
			scopeSessionCgroupPath = func(unit string) (string, error) {
				if unit != "vrooli-agent-opencode-abc.scope" {
					t.Fatalf("unexpected scope unit %q", unit)
				}
				return "/user.slice/vrooli-agents.slice/vrooli-agent-opencode-abc.scope", nil
			}
			scopeEditorScopeGone = func(context.Context, string) (bool, error) { return true, nil }
			ref.StartedAt = &startAt
			report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
			if err != nil || report.Complete != tc.wantComplete || report.Executors[0].State != tc.want {
				t.Fatalf("identity failure escaped its PID boundary: report=%+v err=%v", report, err)
			}
			if strings.Contains(strings.Join(report.Executors[0].Reasons, " "), "secret") {
				t.Fatal("process environment read error leaked into the report")
			}
			if tc.want == "unknown" && !strings.Contains(strings.Join(report.Executors[0].Reasons, " "), "leased PID process identity is unreadable") {
				t.Fatalf("target PID uncertainty lacked a scoped reason: %+v", report.Executors[0])
			}
			if tc.want == "absent" && len(report.Executors[0].Reasons) != 0 {
				t.Fatalf("verified absence carries contradictory reasons: %+v", report.Executors[0])
			}
		})
	}
}

func TestExecutorScopeFullProcessTableFailureRemainsGloballyUnknown(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{}, map[int]scopeIdentity{})
	scopeProcessTable = func(context.Context) (map[int]processTableEntry, error) {
		return nil, errors.New("process table unavailable")
	}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, PGID: 42}}})
	if err != nil || report.Complete || report.Executors[0].State != "unknown" {
		t.Fatalf("full process table failure was not globally unknown: report=%+v err=%v", report, err)
	}
}

func TestExecutorScopeLoadsLeaseSnapshotBeforeProcessObservationUsesBudget(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{}, map[int]scopeIdentity{})
	executorScopeObservationTimeout = 25 * time.Millisecond
	var leaseRead atomic.Int32
	scopeEditorLeaseBindings = func(ctx context.Context, _ string, _ []string) ([]scenarioruntime.EditorLease, error) {
		if err := ctx.Err(); err != nil {
			t.Fatalf("lease lookup started after the process observation budget expired: %v", err)
		}
		leaseRead.Add(1)
		// Model a bounded registry read that finishes before expensive host
		// process inspection consumes the rest of the observation budget.
		time.Sleep(5 * time.Millisecond)
		return nil, nil
	}
	scopeProcessBirths = func(ctx context.Context) (map[int]supervision.ProcessInfo, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	started := time.Now()
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "tag", LegacyTag: "legacy"}}})
	if err != nil || report.Complete || report.Executors[0].State != "unknown" {
		t.Fatalf("expired host observation budget was not represented as unknown evidence: report=%+v err=%v", report, err)
	}
	if got := leaseRead.Load(); got != 1 {
		t.Fatalf("lease snapshots started/completed before host scan %d times, want 1", got)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("short-budget regression took unexpectedly long: %s", elapsed)
	}
}

func TestExecutorScopeBatchesExactAttachedLeaseRevalidationAcrossReferencePages(t *testing.T) {
	const attachedCount = ExecutorScopeLeaseCheckBatchSize + 2
	refs := make([]ExecutorScopeRef, ExecutorScopeBatchSize+2)
	leases := make(map[string]scenarioruntime.EditorLease, attachedCount)
	provenance := make(map[string]scenarioruntime.EditorLeaseProvenance, attachedCount)
	for i := range refs {
		refs[i] = ExecutorScopeRef{RunID: fmt.Sprintf("run-%d", i), Tag: fmt.Sprintf("tag-%d", i), LegacyTag: fmt.Sprintf("legacy-%d", i)}
	}
	attachedStart := ExecutorScopeBatchSize - ExecutorScopeLeaseCheckBatchSize
	for i := attachedStart; i < len(refs); i++ {
		sessionID := fmt.Sprintf("opencode-session-%d", i)
		bindingID := fmt.Sprintf("binding-%d", i)
		scope := fmt.Sprintf("vrooli-agent-opencode-%d", i)
		ref := &refs[i]
		ref.HarnessSessionID, ref.HarnessKind, ref.PID = sessionID, "opencode", 10000+i
		lease := scenarioruntime.EditorLease{SessionID: sessionID, Harness: "opencode", PID: ref.PID, HostBootID: "boot-current", Scope: scope, BindingID: bindingID, Status: scenarioruntime.EditorLeaseActive}
		leases[sessionID] = lease
		provenance[bindingID] = scenarioruntime.EditorLeaseProvenance{
			BindingID: bindingID, SessionID: sessionID, PID: lease.PID, UID: 1000,
			HostBootID: "boot-current", ProcessStart: "1001", IntendedScope: scope,
			ContainmentState: scenarioruntime.EditorContainmentEstablished,
			Scope:            "cgroup:/user.slice/vrooli-agents.slice/" + scope + ".scope",
			Method:           "systemd", EvidencePID: lease.PID + 1, EvidenceUID: 1000, EvidenceBootID: "boot-current", EvidenceProcessStart: "1002",
		}
	}
	scopeFixture(t, map[int]processTableEntry{}, map[int]scopeIdentity{})
	leaseReadCalls := 0
	scopeEditorLeaseBindings = func(_ context.Context, _ string, keys []string) ([]scenarioruntime.EditorLease, error) {
		leaseReadCalls++
		out := make([]scenarioruntime.EditorLease, 0, len(keys))
		for _, key := range keys {
			if lease, ok := leases[key]; ok {
				out = append(out, lease)
			}
		}
		return out, nil
	}
	scopeEditorLeaseProvenance = func(_ context.Context, _ string, bindingID string) (scenarioruntime.EditorLeaseProvenance, error) {
		binding, ok := provenance[bindingID]
		if !ok {
			return scenarioruntime.EditorLeaseProvenance{}, scenarioruntime.ErrNotFound
		}
		return binding, nil
	}
	provenanceReadCalls := 0
	scopeEditorLeaseProvenanceBatch = func(_ context.Context, _ string, ids []string) (map[string]scenarioruntime.EditorLeaseProvenance, map[string]error) {
		provenanceReadCalls++
		if len(ids) > ExecutorScopeLeaseCheckBatchSize {
			t.Fatalf("provenance batch exceeded its bound: %d", len(ids))
		}
		items := make(map[string]scenarioruntime.EditorLeaseProvenance, len(ids))
		for _, id := range ids {
			items[id] = provenance[id]
		}
		return items, nil
	}
	scopeSessionCgroupPath = func(unit string) (string, error) {
		return "/user.slice/vrooli-agents.slice/" + unit, nil
	}
	var scopeGoneCalls atomic.Int32
	scopeEditorScopeGone = func(context.Context, string) (bool, error) {
		scopeGoneCalls.Add(1)
		return true, nil
	}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: refs})
	if err != nil || !report.Complete || len(report.Executors) != len(refs) {
		t.Fatalf("large attached inventory was incomplete: count=%d complete=%t err=%v", len(report.Executors), report.Complete, err)
	}
	for _, evidence := range report.Executors {
		if evidence.State != "absent" || len(evidence.PIDs) != 0 || len(evidence.Reasons) != 0 {
			t.Fatalf("exact gone proof was lost during batched revalidation: %+v", evidence)
		}
	}
	// Two reference pages each take one initial lease snapshot. The 128 proofs
	// on page one and two proofs on page two each use one bounded revalidation
	// snapshot, instead of opening/reading the lease registry once per proof.
	if want := 4; leaseReadCalls != want {
		t.Fatalf("lease registry reads=%d want %d for two initial pages plus two proof batches", leaseReadCalls, want)
	}
	if want := 2; provenanceReadCalls != want {
		t.Fatalf("provenance reads=%d want one bounded lookup per reference page", provenanceReadCalls)
	}
	if want := int32(attachedCount); scopeGoneCalls.Load() != want {
		t.Fatalf("exact native scope checks=%d want %d", scopeGoneCalls.Load(), want)
	}
	t.Logf("%d references / %d exact attached gone proofs / %d lease snapshots", len(refs), attachedCount, leaseReadCalls)
}

func TestExecutorScopeRejectsPartialOrMalformedAttachedIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, session, harness string
		pid                    int
	}{
		{"missing_session", "", "opencode", 42},
		{"missing_harness", "opencode-abc", "", 42},
		{"blank_session", " ", "opencode", 42},
		{"trimmed_session", " opencode-abc", "opencode", 42},
		{"blank_harness", "opencode-abc", " ", 42},
		{"trimmed_harness", "opencode-abc", "opencode ", 42},
		{"missing_pid", "opencode-abc", "opencode", 0},
		{"init_pid", "opencode-abc", "opencode", 1},
		{"oversized_session", strings.Repeat("x", 513), "opencode", 42},
		{"oversized_harness", "opencode-abc", strings.Repeat("x", 101), 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scopeFixture(t, nil, nil)
			scopeProcessTable = func(context.Context) (map[int]processTableEntry, error) {
				t.Fatal("malformed identity reached host observation")
				return nil, nil
			}
			_, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: tc.pid, HarnessSessionID: tc.session, HarnessKind: tc.harness}}})
			if err == nil {
				t.Fatal("malformed attached identity was accepted")
			}
		})
	}
}

// Given a canonical key that appears while the legacy native scope is read,
// when the selected binding is revalidated, then the new canonical row wins.
func TestExecutorScopeCanonicalRegistrationDuringLegacyProofCannotBeBypassed(t *testing.T) {
	created := time.Unix(1001, 0).UTC()
	scopeFixture(t, map[int]processTableEntry{42: {PID: 42}}, map[int]scopeIdentity{})
	scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
		return map[int]supervision.ProcessInfo{42: {PID: 42, StartedAt: created}}, nil
	}
	legacy := scenarioruntime.EditorLease{SessionID: "run-1", Harness: "opencode", PID: 42, HostBootID: "boot-current", Scope: "vrooli-agent-opencode-old", Status: scenarioruntime.EditorLeaseActive, CreatedAt: created}
	rows := []scenarioruntime.EditorLease{legacy}
	scopeEditorLeaseBindings = func(context.Context, string, []string) ([]scenarioruntime.EditorLease, error) { return rows, nil }
	scopeEditorScopeMembers = func(string) ([]int, error) {
		canonical := legacy
		canonical.SessionID = "opencode-current"
		canonical.PID = 43
		rows = append(rows, canonical)
		return []int{42}, nil
	}
	ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, HarnessKind: "opencode", HarnessSessionID: "opencode-current"}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
	if err != nil || report.Complete || report.Executors[0].State != "unknown" {
		t.Fatalf("new canonical binding was bypassed: %+v, %v", report, err)
	}
}

// Given a stopped canonical lease and a plausible active legacy lease in the
// real registry, when the bounded owner snapshot is read, then stopped is not
// misrepresented as NotFound and the legacy binding cannot override it.
func TestExecutorScopeRealSnapshotKeepsStoppedCanonicalBinding(t *testing.T) {
	home := t.TempDir()
	path, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		t.Fatal(err)
	}
	store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: home, DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	lease := scenarioruntime.EditorLease{SessionID: "opencode-current", Harness: "opencode", PID: 43, HostBootID: "boot-current", Scope: "vrooli-agent-opencode-current"}
	if _, err = store.CreateEditorLease(t.Context(), lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err = store.StopEditorLease(t.Context(), lease.SessionID, "finished"); err != nil {
		t.Fatal(err)
	}
	lease.SessionID = "run-1"
	lease.PID = 42
	lease.Scope = "vrooli-agent-opencode-old"
	if _, err = store.CreateEditorLease(t.Context(), lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	scopeFixture(t, map[int]processTableEntry{42: {PID: 42}}, map[int]scopeIdentity{})
	born := time.Now().UTC().Add(-time.Second)
	scopeProcessBirths = func(context.Context) (map[int]supervision.ProcessInfo, error) {
		return map[int]supervision.ProcessInfo{42: {PID: 42, StartedAt: born}}, nil
	}
	scopeEditorLeaseBindings = listScopeEditorLeaseBindings
	scopeEditorScopeMembers = func(string) ([]int, error) {
		t.Fatal("conflicting canonical should not inspect legacy scope")
		return nil, nil
	}
	ref := ExecutorScopeRef{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 42, HarnessKind: "opencode", HarnessSessionID: "opencode-current"}
	report, err := (&Controller{Home: home}).ExecutorScope(t.Context(), ExecutorScopeRequest{Executors: []ExecutorScopeRef{ref}})
	if err != nil || report.Complete || report.Executors[0].State != "unknown" || !strings.Contains(strings.Join(report.Executors[0].Reasons, " "), "persisted PID or harness") {
		t.Fatalf("stopped canonical bypassed: %+v, %v", report, err)
	}
}
