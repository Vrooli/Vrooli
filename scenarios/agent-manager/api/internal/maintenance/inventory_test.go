package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

func TestInventoryCountsFalseFailedPhysicalExecutorAndPreservesRows(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('usi','failed',315576,315576,'none'),('pending','pending',0,0,'none'),('parked','parked',0,0,'none'),('done','complete',0,0,'none');
INSERT INTO workflow_executions VALUES('workflow','running'),('history','succeeded');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{ExecutorRef: ref, Alive: true, HasChildren: true}, nil
	})
	before, err := observer.Observe(t.Context())
	if err != nil || before.Remaining == nil || *before.Remaining != 4 || len(before.Executors) != 1 || before.Executors[0].Status != "failed" || !before.Executors[0].HasChildren || before.ControlPlaneGap == "" {
		t.Fatalf("false terminal work disappeared: %+v %v", before, err)
	}
	_, _ = gate.Enter(t.Context(), "owner", "rollout")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	standing, err := gate.Wait(ctx, observer.Remaining, 0)
	if err == nil || standing.Drained {
		t.Fatalf("cancelled drain=%+v %v", standing, err)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM runs WHERE id='usi'`).Scan(&status); err != nil || status != "failed" {
		t.Fatalf("inventory changed accounting: %q %v", status, err)
	}
}

func TestInventoryPagesControlPlaneReferencesBeyondPerRequestLimit(t *testing.T) {
	const count = 65537
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory-large.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, execution_mode TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<65537)
INSERT INTO runs SELECT printf('00000000-0000-0000-0000-%012x',x),'failed','codec_pipe',x,x,'none' FROM n;`)
	if err != nil {
		t.Fatal(err)
	}
	reads, hydrated, totalRefs := 0, 0, 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(_ context.Context, refs []ExecutorRef) ([]*domain.Run, error) {
			hydrated++
			out := make([]*domain.Run, 0, len(refs))
			for _, ref := range refs {
				id, parseErr := uuid.Parse(ref.ID)
				if parseErr != nil {
					return nil, parseErr
				}
				out = append(out, &domain.Run{ID: id, RunnerPID: ref.PID, RunnerPGID: ref.PGID})
			}
			return out, nil
		},
		read: func(_ context.Context, input []byte) ([]byte, error) {
			reads++
			var request struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &request); err != nil || len(request.Executors) != count || len(input) > executorScopeMaxBytes {
				t.Fatalf("unbounded control-plane request: refs=%d bytes=%d err=%v", len(request.Executors), len(input), err)
			}
			totalRefs += len(request.Executors)
			var report strings.Builder
			report.WriteString(`{"schemaVersion":"executor-scope-v1","complete":true,"executors":[`)
			for i, ref := range request.Executors {
				if i > 0 {
					report.WriteByte(',')
				}
				fmt.Fprintf(&report, `{"runId":%q,"state":"absent","pids":[],"reasons":[]}`, ref.RunID)
			}
			report.WriteString(`]}`)
			return []byte(report.String()), nil
		},
	}
	state, err := NewBatchedInventory(db, owner.Observe).Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 0 || len(state.Unknown) != 0 || totalRefs != count || reads != 1 || hydrated != 1 {
		t.Fatalf("full inventory was not covered by bounded pages: refs=%d reads=%d hydrated=%d state=%+v err=%v", totalRefs, reads, hydrated, state, err)
	}
}

func TestActivityInventoryRoutesAttachedZeroPIDRunToExactOwnerProof(t *testing.T) {
	id := uuid.New()
	attached := &domain.Run{
		ID: id, ExecutionMode: domain.ExecutionModeAttached,
		HarnessKind: "claude-code", HarnessSessionID: "harness-session",
		WebConsoleSessionID: "web-session", Tag: "attached-run",
	}
	_, db := openGate(t, filepath.Join(t.TempDir(), "activity-attached-zero-pid.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT,status TEXT,execution_mode TEXT,runner_pid INTEGER,runner_pgid INTEGER,finalization_status TEXT,tag TEXT,started_at TEXT,ended_at TEXT,harness_kind TEXT,harness_session_id TEXT,web_console_session_id TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
INSERT INTO runs VALUES(?, 'running','attached',0,0,'none','attached-run',NULL,NULL,'claude-code','harness-session','web-session');`, id.String())
	if err != nil {
		t.Fatal(err)
	}
	proof := cliutil.AgentSessionBindingProof{
		ProofVersion: cliutil.AgentSessionBindingProofVersion,
		RunID:        id.String(), HarnessSessionID: attached.HarnessSessionID,
		LeaseSessionID: "lease-alias", WebConsoleSessionID: attached.WebConsoleSessionID,
		HarnessKind: attached.HarnessKind, OwnerKind: "web-console-session",
		Binding: cliutil.ProcessBinding{PID: 555, UID: 1000, Start: "12345", Boot: "faad40a2-2026-4c3b-8b36-efc2d641ced6", Cgroup: "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/tmux-spawn-test.scope"},
	}
	proofReads, scopeReads := 0, 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(_ context.Context, refs []ExecutorRef) ([]*domain.Run, error) {
			if len(refs) != 1 || refs[0].ID != id.String() || refs[0].PID != 0 || refs[0].PGID != 0 {
				t.Fatalf("attached zero-PID row not included in hydration: %+v", refs)
			}
			return []*domain.Run{attached}, nil
		},
		readBinding: func(_ context.Context, request attachedBindingRequest) (cliutil.AgentSessionBindingProof, error) {
			proofReads++
			if request != (attachedBindingRequest{RunID: id.String(), HarnessSessionID: attached.HarnessSessionID, WebConsoleSessionID: attached.WebConsoleSessionID, HarnessKind: attached.HarnessKind}) {
				t.Fatalf("unexpected owner proof request: %+v", request)
			}
			return proof, nil
		},
		read: func(_ context.Context, input []byte) ([]byte, error) {
			scopeReads++
			var request struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &request); err != nil || len(request.Executors) != 1 {
				t.Fatalf("invalid physical proof request: %s err=%v", input, err)
			}
			ref := request.Executors[0]
			if ref.PID != 0 || ref.PGID != 0 || ref.BindingProof == nil || ref.BindingProof.LeaseSessionID != "lease-alias" {
				t.Fatalf("exact attached proof was not forwarded: %+v", ref)
			}
			return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1","identityVersion":"attached-key-v1","complete":true,"executors":[{"runId":%q,"state":"present","pids":[556],"reasons":[]}]}`, id.String())), nil
		},
	}
	state, err := NewBatchedInventory(db, owner.Observe).ObserveActivity(t.Context())
	if err != nil || proofReads != 1 || scopeReads != 1 || state.Remaining == nil || *state.Remaining != 1 || len(state.Unknown) != 0 {
		t.Fatalf("attached inventory did not defer to exact proof: state=%+v proofReads=%d scopeReads=%d err=%v", state, proofReads, scopeReads, err)
	}
}

// Live shape of 2026-11-20: 123k terminal attached rows retained harness PIDs.
// Sending them grew the request past the owner limit and left every activity
// observation incomplete. Terminal attached rows are not AM executors; live
// attached rows and managed terminal residue are still observed.
func TestActivityInventorySkipsTerminalAttachedHistory(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "activity-attached-history.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT,status TEXT,execution_mode TEXT,runner_pid INTEGER,runner_pgid INTEGER,finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<500)
INSERT INTO runs SELECT 'attached-done-'||x, CASE x%3 WHEN 0 THEN 'complete' WHEN 1 THEN 'needs_review' ELSE 'cancelled' END,'attached',700000+x,0,'none' FROM n;
INSERT INTO runs VALUES('attached-live','running','attached',4242,0,'none'),('managed-residue','failed','codec_pipe',315576,315576,'none'),('attached-finalizing','complete','attached',4343,0,'pending');`)
	if err != nil {
		t.Fatal(err)
	}
	var probed []string
	observer := NewBatchedInventory(db, func(_ context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
		out := make([]ExecutorEvidence, 0, len(refs))
		for _, ref := range refs {
			probed = append(probed, ref.ID)
			out = append(out, ExecutorEvidence{ExecutorRef: ref, Alive: ref.ID == "attached-live"})
		}
		return out, nil
	})
	state, err := observer.ObserveActivity(t.Context())
	if err != nil || !state.PhysicalComplete || len(state.Unknown) != 0 {
		t.Fatalf("activity inventory incomplete: state=%+v err=%v", state, err)
	}
	if strings.Join(probed, ",") != "attached-finalizing,attached-live,managed-residue" {
		t.Fatalf("probed=%v; want live attached, finalizing attached and managed residue only", probed)
	}
	if strings.Join(state.PhysicalRunIDs, ",") != "attached-live" || state.ObservedExecutors != 1 {
		t.Fatalf("physical runs=%v executors=%d", state.PhysicalRunIDs, state.ObservedExecutors)
	}
}

// One harness session re-attached three times keeps three live runs with one
// process. The control plane proves the PID only inside the earliest run's
// interval, so only that run is probed; the session counts as one executor.
func TestControlPlaneProbesOneRunPerReattachedSession(t *testing.T) {
	base := time.Date(2026, 11, 19, 0, 43, 29, 0, time.UTC)
	runAt := func(offset time.Duration, session string, pid int) *domain.Run {
		started := base.Add(offset)
		return &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeAttached, HarnessKind: "opencode", HarnessSessionID: session, RunnerPID: pid, StartedAt: &started, Tag: "attached-" + session}
	}
	first, second, third := runAt(0, "opencode-a", 1271209), runAt(7*time.Hour, "opencode-a", 1271209), runAt(14*time.Hour, "opencode-a", 1271209)
	other := runAt(time.Hour, "codex-b", 521025)
	runs := []*domain.Run{third, second, other, first}
	refs := make([]ExecutorRef, 0, len(runs))
	for _, run := range runs {
		refs = append(refs, ExecutorRef{WorkRef: WorkRef{ID: run.ID.String(), Kind: "run", Status: string(run.Status)}, PID: run.RunnerPID})
	}
	var sent []string
	owner := &ControlPlaneObserver{
		lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) { return runs, nil },
		read: func(_ context.Context, input []byte) ([]byte, error) {
			var request struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &request); err != nil {
				t.Fatal(err)
			}
			items := make([]string, 0, len(request.Executors))
			for _, ref := range request.Executors {
				sent = append(sent, ref.RunID)
				items = append(items, fmt.Sprintf(`{"runId":%q,"state":"present","pids":[%d],"reasons":[]}`, ref.RunID, ref.PID))
			}
			return []byte(`{"schemaVersion":"executor-scope-v1","identityVersion":"attached-key-v1","complete":true,"executors":[` + strings.Join(items, ",") + `]}`), nil
		},
	}
	evidence, err := owner.Observe(t.Context(), refs)
	if err != nil || len(evidence) != 4 {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
	if len(sent) != 2 || !((sent[0] == first.ID.String() && sent[1] == other.ID.String()) || (sent[1] == first.ID.String() && sent[0] == other.ID.String())) {
		t.Fatalf("sent=%v; want the earliest run of the re-attached session and the other session", sent)
	}
	alive := map[string]bool{}
	for _, item := range evidence {
		alive[item.ID] = item.Alive
	}
	if !alive[first.ID.String()] || !alive[other.ID.String()] || alive[second.ID.String()] || alive[third.ID.String()] {
		t.Fatalf("alive=%v; re-attached duplicates must not add executors", alive)
	}
}

func TestInventoryUnknownPhysicalEvidenceCannotDrain(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('usi','failed',315576,315576,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(context.Context, ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{}, errors.New("executor scope unavailable")
	})
	state, err := observer.Observe(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) == 0 {
		t.Fatalf("unknown became empty: %+v %v", state, err)
	}
}

func TestActivityInventoryIdentifiesLogicalRunsWithoutPersistedProcessIdentity(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "activity-unobserved.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
INSERT INTO runs VALUES('run-alpha','running',0,0,'none'),('run-bravo','starting',NULL,0,'none'),('run-charlie','running',0,NULL,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	reader := NewInventory(db, func(context.Context, ExecutorRef) (ExecutorEvidence, error) {
		t.Fatal("run without a persisted process identity reached physical observation")
		return ExecutorEvidence{}, nil
	})
	state, err := reader.ObserveActivity(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) != 1 {
		t.Fatalf("missing physical identities became known activity: state=%+v err=%v", state, err)
	}
	diagnostic := state.Unknown[0]
	for _, want := range []string{"3 logical native run records", "lack persisted PID/PGID", "need owner reconciliation", "run-alpha", "run-bravo", "run-charlie", "omitted_ids=0", "total activity is unknown"} {
		if !strings.Contains(diagnostic, want) {
			t.Errorf("unobserved-run diagnostic missing %q: %s", want, diagnostic)
		}
	}
}

func TestActivityInventoryBoundsUnobservedRunIDDiagnosticsAndReportsOmissionCount(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "activity-unobserved-overflow.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<70)
INSERT INTO runs SELECT printf('run-%03d',x),'running',0,0,'none' FROM n;
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	reader := NewInventory(db, func(context.Context, ExecutorRef) (ExecutorEvidence, error) {
		t.Fatal("run without a persisted process identity reached physical observation")
		return ExecutorEvidence{}, nil
	})
	state, err := reader.ObserveActivity(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) != 1 {
		t.Fatalf("overflowed identity diagnostics became complete or unbounded: state=%+v err=%v", state, err)
	}
	diagnostic := state.Unknown[0]
	for _, want := range []string{"70 logical native run records", "need owner reconciliation", "run-001", "run-064", "omitted_ids=6", "total activity is unknown"} {
		if !strings.Contains(diagnostic, want) {
			t.Errorf("bounded unobserved-run diagnostic missing %q: %s", want, diagnostic)
		}
	}
	if strings.Contains(diagnostic, "run-065") {
		t.Fatalf("diagnostic exceeded the 64-ID bound: %s", diagnostic)
	}
}

func TestInventoryBatchFailureReportsOwnerCauseWithoutMissingRowFlood(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
INSERT INTO runs VALUES('one','failed',315576,315576,'none'),('two','failed',315577,315577,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	want := "control-plane executor-scope requires at most 65536 references"
	reader := NewBatchedInventory(db, func(context.Context, []ExecutorRef) ([]ExecutorEvidence, error) {
		return nil, errors.New(want)
	})
	state, err := reader.Observe(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) != 1 || state.Unknown[0] != want {
		t.Fatalf("batch owner failure became row-level omission noise: state=%+v err=%v", state, err)
	}
}

func TestInventoryCompletedAndRestingReviewHistoryAllowsDrain(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<6000)
INSERT INTO runs SELECT 'run-'||x,CASE WHEN x%2=0 THEN 'complete' ELSE 'needs_review' END,0,0,'none' FROM n;
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	observer := NewInventory(db, func(context.Context, ExecutorRef) (ExecutorEvidence, error) {
		calls++
		return ExecutorEvidence{}, errors.New("no executor should need observation")
	})
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 0 || calls != 0 {
		t.Fatalf("resting history blocks drain: %+v calls=%d err=%v", state, calls, err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	standing, err := gate.Wait(ctx, observer.Remaining, time.Second)
	if err != nil || !standing.Drained {
		t.Fatalf("resting history did not drain: %+v %v", standing, err)
	}
	if _, err := gate.Admit(t.Context()); !errors.Is(err, ErrClosed) {
		t.Fatalf("future admission opened after drain: %v", err)
	}
	var retained int
	if err := db.QueryRow(`SELECT COUNT(*) FROM runs`).Scan(&retained); err != nil || retained != 6000 {
		t.Fatalf("history altered: count=%d err=%v", retained, err)
	}
}

func TestInventoryRetainedPIDsRequireOwnerProofAndDoNotTruncateLiveTail(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<2500)
INSERT INTO runs SELECT 'old-'||x,'complete',x,x,'none' FROM n;
INSERT INTO runs VALUES('zz-review','needs_review',9001,9001,'none'),('zz-usi','failed',315576,315576,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	liveUSI := true
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		calls++
		// This is an injected complete control-plane exclusion proof, not a
		// missing-root heuristic. USI stays alive beyond the first result page.
		return ExecutorEvidence{ExecutorRef: ref, Alive: liveUSI && ref.ID == "zz-usi", HasChildren: liveUSI && ref.ID == "zz-usi"}, nil
	})
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 1 || calls != 2502 || len(state.Executors) != 1 || state.Executors[0].PID != 315576 {
		t.Fatalf("physical tail lost or history blocked: %+v calls=%d err=%v", state, calls, err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	standing, err := gate.Wait(ctx, func(ctx context.Context) (int, error) {
		count, err := observer.Remaining(ctx)
		cancel()
		return count, err
	}, time.Second)
	if err == nil || standing.Drained {
		t.Fatalf("live false-failed executor allowed drain: %+v %v", standing, err)
	}
	// Only a subsequent complete owner proof permits the retained PIDs to drain.
	liveUSI = false
	ctx, done := context.WithTimeout(t.Context(), 5*time.Second)
	defer done()
	standing, err = gate.Wait(ctx, observer.Remaining, time.Second)
	if err != nil || !standing.Drained {
		t.Fatalf("excluded historic executor scopes block drain: %+v %v", standing, err)
	}
}

func TestInventoryRestingReviewStillCountsExecutorOrFinalization(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('live-review','needs_review',123,123,'none'),('pending-finalization','needs_review',0,0,'pending'),('running-finalization','needs_review',0,0,'running');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{ExecutorRef: ref, Alive: true}, nil
	})
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 3 {
		t.Fatalf("review executor/finalization excluded: %+v %v", state, err)
	}
}

func TestInventoryRetainedReviewExecutorRequiresCompleteProof(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('review','needs_review',123,123,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{ExecutorRef: ref, Alive: false}, errors.New("root absent; descendant scope not excluded")
	})
	state, err := observer.Observe(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) != 1 {
		t.Fatalf("resting review relaxed physical exclusion: %+v %v", state, err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	standing, err := gate.Wait(t.Context(), observer.Remaining, time.Second)
	if err == nil || standing.Drained {
		t.Fatalf("unknown review executor scope drained: %+v %v", standing, err)
	}
}

func TestInventoryInterruptedProofDoesNotReportEmpty(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('history','complete',123,123,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		cancel()
		return ExecutorEvidence{ExecutorRef: ref}, nil
	})
	state, err := observer.Observe(ctx)
	if !errors.Is(err, context.Canceled) || state.Remaining != nil {
		t.Fatalf("interrupted exclusion became empty: %+v %v", state, err)
	}
}

func TestInventoryImportedHistoryIsNotOwnedExecution(t *testing.T) {
	for _, batched := range []bool{false, true} {
		t.Run(fmt.Sprintf("batched=%v", batched), func(t *testing.T) {
			testImportedHistoryIsNotOwnedExecution(t, batched)
		})
	}
}

func testImportedHistoryIsNotOwnedExecution(t *testing.T, batched bool) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, execution_mode TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<1000)
INSERT INTO runs SELECT 'imported-'||x,CASE WHEN x%2=0 THEN 'unknown' ELSE 'running' END,'imported',900000+x,900000+x,'none' FROM n;`)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	physical := func(ctx context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		calls++
		if ref.ID != "managed-usi" {
			return ExecutorEvidence{}, fmt.Errorf("foreign imported executor entered owner scope: %s", ref.ID)
		}
		return ExecutorEvidence{ExecutorRef: ref, Alive: true, HasChildren: true, PIDs: []int{315576, 567605}}, nil
	}
	observer := NewInventory(db, physical)
	if batched {
		observer = NewBatchedInventory(db, func(ctx context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
			if len(refs) != 1 {
				return nil, fmt.Errorf("imported history consumed root batch capacity: %d references", len(refs))
			}
			evidence, err := physical(ctx, refs[0])
			return []ExecutorEvidence{evidence}, err
		})
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 0 || len(state.Work) != 0 || calls != 0 {
		t.Fatalf("read-only history blocks drain: %+v %v", state, err)
	}
	standing, err := gate.Wait(t.Context(), observer.Remaining, time.Second)
	if err != nil || !standing.Drained {
		t.Fatalf("history blocked drain: %+v %v", standing, err)
	}
	// The mode exclusion is not a general unknown-status exemption. Legacy and
	// managed unknown runs and managed terminal executor residue stay visible.
	_, err = db.Exec(`INSERT INTO runs VALUES('managed','unknown','codec_pipe',0,0,'none'),('legacy','unknown',NULL,0,0,'none'),('managed-usi','failed','codec_pipe',315576,315576,'skipped');`)
	if err != nil {
		t.Fatal(err)
	}
	state, err = observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 3 || len(state.Executors) != 1 || calls != 1 || state.Executors[0].PID != 315576 {
		t.Fatalf("owned unknown or managed terminal residue erased: %+v calls=%d err=%v", state, calls, err)
	}
	// Imported active finalization contradicts the read-only ownership contract.
	// Keep that contradiction explicit without inspecting a foreign historical PID.
	_, err = db.Exec(`INSERT INTO runs VALUES('finalization-import-pending','unknown','imported',999998,999998,'pending'),('finalization-import-running','unknown','imported',999999,999999,'running');`)
	if err != nil {
		t.Fatal(err)
	}
	state, err = observer.Observe(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) != 2 || len(state.Executors) != 1 || calls != 2 {
		t.Fatalf("contradictory finalization became exclusion or foreign proof: %+v calls=%d err=%v", state, calls, err)
	}
	for _, status := range []string{"pending", "running"} {
		if !strings.Contains(strings.Join(state.Unknown, "\n"), "finalization-import-"+status+": imported read-only run claims active finalization") {
			t.Fatalf("missing explicit ownership contradiction: %+v", state)
		}
	}
	var retained int
	if err := db.QueryRow(`SELECT COUNT(*) FROM runs WHERE runner_pid > 0`).Scan(&retained); err != nil || retained != 1003 {
		t.Fatalf("history modified: count=%d err=%v", retained, err)
	}
}

func TestInventoryOnePhysicalBatchAcrossHistory(t *testing.T) {
	for _, references := range []int{2, 129, 17029, 65536, 65537} {
		t.Run(fmt.Sprintf("references=%d", references), func(t *testing.T) {
			_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
			db.SetMaxOpenConns(1)
			_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, execution_mode TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<300)
INSERT INTO runs SELECT 'active-'||x,'unknown','codec_pipe',0,0,'none' FROM n;`)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<?)
INSERT INTO runs SELECT 'executor-'||x,'failed','codec_pipe',x,x,'none' FROM n`, references); err != nil {
				t.Fatal(err)
			}
			calls := 0
			reader := NewBatchedInventory(db, func(ctx context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
				calls++
				var retained int
				if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs`).Scan(&retained); err != nil || retained != references+300 {
					t.Fatalf("SQL cursor was not closed before observer hydration: count=%d err=%v", retained, err)
				}
				if len(refs) != references {
					t.Fatalf("SQL pages fragmented root inventory: %d refs", len(refs))
				}
				var out []ExecutorEvidence
				for _, ref := range refs {
					out = append(out, ExecutorEvidence{ExecutorRef: ref})
				}
				return out, nil
			})
			state, err := reader.Observe(t.Context())
			if err != nil || calls != 1 || state.Remaining == nil || *state.Remaining != 300 {
				t.Fatalf("complete physical inventory lost logical work: calls=%d state=%+v err=%v", calls, state, err)
			}
		})
	}
}
