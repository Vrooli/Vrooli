package maintenance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

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
		{name: "pid_outside_exact_native_scope_is_unknown", missingMember: true, want: "unknown", reason: "not a member"},
		{name: "lease_rebind_during_scope_proof_is_unknown", rebind: true, want: "unknown", reason: "changed or became unavailable"},
		{name: "owner_rebind_during_scope_proof_is_unknown", ownerChange: true, want: "unknown", reason: "changed or became unavailable"},
		{name: "lease_reread_failure_is_unknown", readError: true, want: "unknown", reason: "changed or became unavailable"},
		{name: "pid_reused_during_scope_proof_is_unknown", finalBirthChange: true, want: "unknown", reason: "birth changed or became unavailable"},
		{name: "planned_scope_gone_cannot_prove_historical_containment", gone: true, want: "unknown", reason: "established containment provenance"},
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
			lease := scenarioruntime.EditorLease{SessionID: "opencode-abc", Harness: "opencode", PID: 42, HostBootID: "boot-current", Scope: "vrooli-agent-opencode-abc", Status: scenarioruntime.EditorLeaseActive, CreatedAt: created, Owner: scenarioruntime.EditorLeaseOwner{Kind: "scenario-instance", Ref: "web-console-instance", Source: "launch", EvidenceJSON: `{"read_from_pid":42}`}}
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
			scopeEditorLeaseBindings = func(context.Context, string, []string) ([]scenarioruntime.EditorLease, error) {
				if tc.registryError {
					return nil, errors.New("registry unavailable")
				}
				return rows, nil
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
			reread := scopeEditorLeaseBySession
			scopeEditorLeaseBySession = func(ctx context.Context, home, key string) (scenarioruntime.EditorLease, error) {
				current, err := reread(ctx, home, key)
				if tc.readError {
					return current, errors.New("read unavailable")
				}
				if tc.rebind {
					current.PID = 43
				}
				if tc.ownerChange {
					current.Owner.Ref = "other-instance"
				}
				return current, err
			}
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
			if tc.want == "present" && (len(evidence.PIDs) != 1 || evidence.PIDs[0] != 42) {
				t.Fatalf("missing positive exact PID: %+v", evidence)
			}
			if tc.want == "absent" && (len(evidence.PIDs) != 0 || len(evidence.Reasons) != 0) {
				t.Fatalf("invalid absence: %+v", evidence)
			}
		})
	}
}

func TestExecutorScopeRejectsPartialOrMalformedAttachedIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, session, harness string
		pid                    int
	}{
		{"missing_session", "", "opencode", 42}, {"missing_harness", "opencode-abc", "", 42},
		{"blank_session", " ", "opencode", 42}, {"trimmed_session", " opencode-abc", "opencode", 42},
		{"blank_harness", "opencode-abc", " ", 42}, {"trimmed_harness", "opencode-abc", "opencode ", 42},
		{"missing_pid", "opencode-abc", "opencode", 0}, {"init_pid", "opencode-abc", "opencode", 1},
		{"oversized_session", strings.Repeat("x", 513), "opencode", 42}, {"oversized_harness", "opencode-abc", strings.Repeat("x", 101), 42},
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
	scopeEditorLeaseBySession = getScopeEditorLeaseBySession
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
