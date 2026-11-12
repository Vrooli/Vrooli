package sessionlease

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func provenanceRecorder(t *testing.T) (Recorder, func(context.Context) (*scenarioruntime.SQLiteStore, error), map[int]cliutil.ProcessBinding) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime.db")
	open := func(ctx context.Context) (*scenarioruntime.SQLiteStore, error) {
		return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: path})
	}
	processes := map[int]cliutil.ProcessBinding{
		100: {PID: 100, UID: 1000, Boot: "boot-a", Start: "123", Cgroup: "0::/launcher"},
		200: {PID: 200, UID: 1000, Boot: "boot-a", Start: "456", Cgroup: "0::/planned-scope.scope"},
	}
	return Recorder{Open: open, Owner: func() scenarioruntime.EditorLeaseOwner { return scenarioruntime.EditorLeaseOwner{} }, Observe: func(pid int) (cliutil.ProcessBinding, error) {
		p, ok := processes[pid]
		if !ok {
			return p, errors.New("process absent")
		}
		return p, nil
	}}, open, processes
}

func TestBindingRecorderPersistsOnlySuccessfulNativeContainment(t *testing.T) {
	for _, tc := range []struct {
		name   string
		state  string
		change func(*cliutil.EditorLeaseContainment, map[int]cliutil.ProcessBinding)
	}{
		{name: "successful native child", state: scenarioruntime.EditorContainmentEstablished},
		{name: "planned method and scope", state: scenarioruntime.EditorContainmentUnavailable, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) {
			e.Session.BeforeExec = false
		}},
		{name: "unknown mechanism", state: scenarioruntime.EditorContainmentUnavailable, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) {
			e.Session.Method = "unknown"
		}},
		{name: "wrong intended scope", state: scenarioruntime.EditorContainmentUnavailable, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) {
			e.Session.Scope = "cgroup:/other.scope"
		}},
		{name: "child post-start masquerades as self", state: scenarioruntime.EditorContainmentUnavailable, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) {
			e.Session.Method = "cgroup-write"
		}},
		{name: "post-start placement", state: scenarioruntime.EditorContainmentUnavailable, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) {
			e.Session.BeforeExec = false
			e.Session.ProofFailure = "post-start root move"
		}},
		{name: "returned scope with failure", state: scenarioruntime.EditorContainmentFailed, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) {
			e.Failure = "placement failed"
		}},
		{name: "unsupported native primitive", state: scenarioruntime.EditorContainmentUnavailable, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) {
			e.Failure = "unsupported"
			e.Unavailable = true
		}},
		{name: "explicit unavailable overrides success-shaped data", state: scenarioruntime.EditorContainmentUnavailable, change: func(e *cliutil.EditorLeaseContainment, _ map[int]cliutil.ProcessBinding) { e.Unavailable = true }},
		{name: "payload pid reuse", state: scenarioruntime.EditorContainmentUnavailable, change: func(_ *cliutil.EditorLeaseContainment, p map[int]cliutil.ProcessBinding) {
			v := p[200]
			v.Start = "999"
			p[200] = v
		}},
		{name: "payload exited", state: scenarioruntime.EditorContainmentUnavailable, change: func(_ *cliutil.EditorLeaseContainment, p map[int]cliutil.ProcessBinding) { delete(p, 200) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r, open, processes := provenanceRecorder(t)
			id, err := r.BeginBinding(ctx, cliutil.EditorLeaseRecord{SessionID: "session", PID: 100, Scope: "planned-scope"})
			if err != nil {
				t.Fatal(err)
			}
			evidence := cliutil.EditorLeaseContainment{Session: cliutil.ContainedSession{Scope: "cgroup:/planned-scope.scope", Method: "systemd-run", BeforeExec: true, Process: processes[200]}}
			if tc.change != nil {
				tc.change(&evidence, processes)
			}
			if err := r.RecordContainment(ctx, "session", id, evidence); err != nil {
				t.Fatal(err)
			}
			store, err := open(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			got, err := store.GetEditorLeaseProvenance(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if got.ContainmentState != tc.state || got.ProcessStart != "123" || got.PID != 100 || got.IntendedScope != "planned-scope" {
				t.Fatalf("binding=%+v", got)
			}
			if tc.state == scenarioruntime.EditorContainmentEstablished {
				if got.Scope != "cgroup:/planned-scope.scope" || got.Method != "systemd-run" || got.EvidencePID != 200 || got.EvidenceProcessStart != "456" {
					t.Fatalf("evidence=%+v", got)
				}
			} else if got.Scope != "" || got.Method != "" || got.EvidencePID != 0 || got.Failure == "" {
				t.Fatalf("failure promoted to success: %+v", got)
			}
		})
	}
}

func TestBindingRecorderRejectsLauncherReuseAndFencesRebinding(t *testing.T) {
	ctx := context.Background()
	r, open, processes := provenanceRecorder(t)
	record := cliutil.EditorLeaseRecord{SessionID: "session", PID: 100, Scope: "planned-scope"}
	id, err := r.BeginBinding(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	evidence := cliutil.EditorLeaseContainment{Session: cliutil.ContainedSession{Scope: "cgroup:/planned-scope.scope", Method: "systemd-run", BeforeExec: true, Process: processes[200]}}
	original := processes[100]
	changed := original
	changed.Start = "999"
	processes[100] = changed
	if err := r.RecordContainment(ctx, "session", id, evidence); err == nil {
		t.Fatal("launcher reuse accepted")
	}
	processes[100] = original
	second, err := r.BeginBinding(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	if second == id {
		t.Fatal("reused binding ID")
	}
	if err := r.RecordContainment(ctx, "session", id, evidence); err != nil {
		t.Fatal(err)
	}
	if err := r.StopBinding(ctx, "session", id, "old launcher finished"); err != nil {
		t.Fatal(err)
	}
	if err := r.HeartbeatBinding(ctx, "session", id); err != nil {
		t.Fatal(err)
	}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	old, err := store.GetEditorLeaseProvenance(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.GetEditorLease(ctx, "session")
	if err != nil {
		t.Fatal(err)
	}
	if old.State != scenarioruntime.EditorBindingSuperseded || old.ContainmentState != scenarioruntime.EditorContainmentIntended || current.BindingID != second || current.Status != scenarioruntime.EditorLeaseActive {
		t.Fatalf("old=%+v current=%+v", old, current)
	}
	// A launcher that captured only the old recorder interface must not close
	// or renew a newer native attempt for the same session key.
	if err := r.Heartbeat(ctx, "session"); !errors.Is(err, scenarioruntime.ErrNotFound) {
		t.Fatalf("legacy heartbeat reached bound attempt: %v", err)
	}
	if err := r.Stop(ctx, "session", "late legacy stop"); !errors.Is(err, scenarioruntime.ErrNotFound) {
		t.Fatalf("legacy stop reached bound attempt: %v", err)
	}
	unchanged, err := store.GetEditorLease(ctx, "session")
	if err != nil || unchanged.BindingID != second || unchanged.Status != scenarioruntime.EditorLeaseActive || !unchanged.HeartbeatDeadlineAt.Equal(current.HeartbeatDeadlineAt) {
		t.Fatalf("legacy callback changed replacement: %+v error=%v", unchanged, err)
	}
	if err := r.RecordContainment(ctx, "session", second, evidence); err != nil {
		t.Fatal(err)
	}
	if err := r.StopBinding(ctx, "session", second, "session ended"); err != nil {
		t.Fatal(err)
	}
	terminal, err := store.GetEditorLeaseProvenance(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.State != scenarioruntime.EditorBindingTerminal || terminal.ContainmentState != scenarioruntime.EditorContainmentEstablished {
		t.Fatalf("terminal=%+v", terminal)
	}
}

func TestBindingRecorderDoesNotPromoteLegacyOrUnavailableIdentity(t *testing.T) {
	ctx := context.Background()
	r, open, processes := provenanceRecorder(t)
	if err := r.Create(ctx, cliutil.EditorLeaseRecord{SessionID: "legacy", PID: 100, Scope: "planned", ContainmentMethod: "systemd-run"}); err != nil {
		t.Fatal(err)
	}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	legacy, err := store.GetEditorLease(ctx, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.BindingID != "" {
		t.Fatalf("historical promotion: %+v", legacy)
	}
	bindings, err := store.ListEditorLeaseProvenance(ctx, "legacy")
	if err != nil || len(bindings) != 0 {
		t.Fatalf("history=%+v err=%v", bindings, err)
	}
	delete(processes, 100)
	if _, err := r.BeginBinding(ctx, cliutil.EditorLeaseRecord{SessionID: "unavailable", PID: 100}); !errors.Is(err, cliutil.ErrEditorLeaseBindingUnavailable) {
		t.Fatalf("missing process accepted: %v", err)
	}
}
