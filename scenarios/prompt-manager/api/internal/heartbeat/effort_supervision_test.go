package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

type effortOwnerFake struct {
	rows     []EffortObservation
	quota    []*ampb.EffortQuotaObservation
	err      error
	calls    int
	pages    map[string]*EffortDiscovery
	receipts map[string]*SupervisionAssessment
}

func (o *effortOwnerFake) GetEffortAssessment(_ context.Context, id string) (*SupervisionAssessment, error) {
	return o.receipts[id], nil
}

func (o *effortOwnerFake) DiscoverEfforts(_ context.Context, limit int, cursor string) (*EffortDiscovery, error) {
	o.calls++
	if o.err != nil {
		return nil, o.err
	}
	if o.pages != nil {
		return o.pages[cursor], nil
	}
	return &EffortDiscovery{Efforts: append([]EffortObservation(nil), o.rows...), QuotaObservations: append([]*ampb.EffortQuotaObservation(nil), o.quota...), Coverage: "complete"}, nil
}

type effortQueueFake struct {
	status   TeamExecutionStatus
	enqueues int
}

func (q *effortQueueFake) Status(string) TeamExecutionStatus { return q.status }
func (q *effortQueueFake) Enqueue(_ context.Context, team, agent, profile string) (*EnqueueResult, error) {
	if memberOccupied(q.status, agent) {
		return nil, &MemberAlreadyQueuedError{TeamID: team, AgentID: agent}
	}
	q.enqueues++
	q.status.RunningAgentIDs = []string{agent}
	return &EnqueueResult{TeamID: team, AgentID: agent, Status: "started"}, nil
}
func (q *effortQueueFake) OnComplete(string, string) { q.status = TeamExecutionStatus{} }

type supervisionFixture struct {
	s       *StandingSupervisor
	owner   *effortOwnerFake
	agent   *mockAgentClient
	queue   *effortQueueFake
	cfg     *store.HeartbeatConfig
	now     time.Time
	prompts int
}

func newSupervisionFixture(t *testing.T) *supervisionFixture {
	t.Helper()
	f := &supervisionFixture{owner: &effortOwnerFake{}, queue: &effortQueueFake{}, now: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)}
	f.agent = newMockAgentClient().WithCreateTaskResponse(&Task{ID: "task-1"}).WithCreateRunResponse(&Run{ID: "wake-run-1", Status: "running"})
	f.agent.getRuns["wake-run-1"] = &Run{ID: "wake-run-1", Status: "running"}
	f.agent.listRunsResp = &ListRunsResponse{}
	f.cfg = &store.HeartbeatConfig{Enabled: true, ProfileKey: "qualified-role-profile", Supervision: &teamconfig.Supervision{DiscoveryLimit: 100, MaxEffortsPerWake: 1, MinWakeIntervalSeconds: 60}}
	f.cfg.Supervision.DiagnosticAllowance = teamconfig.DiagnosticAllowance{MaxWakesPerWindow: 10, WindowSeconds: 3600, AccountingRef: "test:standing-supervision"}
	f.s = &StandingSupervisor{Owner: f.owner, State: FileSupervisionStateStore{Root: t.TempDir()}, Queue: f.queue, Agent: f.agent, Root: t.TempDir(), Now: func() time.Time { return f.now },
		Config: func(context.Context, string, string) (*store.HeartbeatConfig, error) { return f.cfg, nil },
		Prompt: func(context.Context, string, string) (string, error) { f.prompts++; return "team context", nil },
	}
	return f
}

func effort(id string) EffortObservation {
	return EffortObservation{ID: id, TargetRevision: "target-1", EvidenceRevision: "evidence-1", Freshness: "fresh", Eligible: true, BoardRef: "agent-manager effort board --effort-ref " + id}
}

func (f *supervisionFixture) tick(t *testing.T) *SupervisionState {
	t.Helper()
	s, err := f.s.Tick(context.Background(), "supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *supervisionFixture) dispatch(t *testing.T) {
	t.Helper()
	if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err != nil {
		t.Fatal(err)
	}
}

func (f *supervisionFixture) receipt(t *testing.T, disposition string) {
	t.Helper()
	state, err := f.s.State.Load("supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if f.owner.receipts == nil {
		f.owner.receipts = map[string]*SupervisionAssessment{}
	}
	for _, row := range state.Pending.Efforts {
		f.owner.receipts[row.ID] = &SupervisionAssessment{ID: "assessment-" + state.Pending.ID, WakeID: state.Pending.ID, RunID: "wake-run-1", Disposition: disposition, TargetRevisions: map[string]string{row.ID: row.TargetRevision}}
	}
}

func TestStandingSupervisorSelectsHigherConsequenceBeforeFairnessTieBreak(t *testing.T) {
	f := newSupervisionFixture(t)
	low, high := effort("effort:low"), effort("effort:high")
	low.Priority, high.Priority = 1, 9
	f.owner.rows = []EffortObservation{low, high}
	state := f.tick(t)
	if state.Pending == nil || len(state.Pending.Efforts) != 1 || state.Pending.Efforts[0].ID != high.ID {
		t.Fatalf("higher-priority changed effort was not selected: %+v", state.Pending)
	}
}

func TestStandingSupervisorRetainsSharedQuotaOnceInWake(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.rows = []EffortObservation{effort("effort:quota")}
	f.owner.quota = []*ampb.EffortQuotaObservation{{Provider: "openai", Pool: "primary", Window: "5h", Standing: "available", Provenance: "codex:native", EvidenceRef: "quota:shared"}}
	state := f.tick(t)
	if state.Pending == nil || len(state.Pending.QuotaObservations) != 1 || state.Pending.QuotaObservations[0].GetEvidenceRef() != "quota:shared" {
		t.Fatalf("shared quota observation was dropped from durable wake: %+v", state.Pending)
	}
}

func TestStandingSupervisorIdleArrivalChangeAndRetirement(t *testing.T) {
	// [REQ:ES-09] [REQ:ES-10] [REQ:ES-13]
	f := newSupervisionFixture(t)
	for i := 0; i < 3; i++ {
		f.tick(t)
		f.now = f.now.Add(time.Minute)
	}
	if f.prompts != 0 || f.queue.enqueues != 0 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("empty discovery bought inference")
	}
	f.owner.rows = []EffortObservation{effort("arbitrary/mandate")}
	first := f.tick(t)
	if first.Pending == nil || first.Pending.Efforts[0].ID != "arbitrary/mandate" {
		t.Fatal("new arbitrary effort was not admitted")
	}
	firstID := first.Pending.ID
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.agent.getRuns["wake-run-1"].Status = "complete"
	f.now = f.now.Add(time.Minute)
	f.tick(t)
	f.tick(t)
	if len(f.agent.createRunCalls) != 1 || f.queue.enqueues != 1 {
		t.Fatal("unchanged observation admitted another wake")
	}
	f.owner.rows[0].ChangeGeneration = "quota-owner-eligible-v2"
	next := f.tick(t)
	if next.Pending == nil || next.Pending.ID == firstID {
		t.Fatal("owner change did not make a distinct wake")
	}
	f.owner.rows[0].Retired, f.owner.rows[0].Eligible = true, false
	f.dispatch(t)
	if len(f.agent.createRunCalls) != 1 {
		t.Fatal("retired queued effort dispatched")
	}
	if state := f.tick(t); state.Pending != nil {
		t.Fatal("retired watch was not fenced")
	}
	f.owner.rows = append(f.owner.rows, effort("another:bounded-task"))
	if state := f.tick(t); state.Pending == nil || state.Pending.Efforts[0].ID != "another:bounded-task" {
		t.Fatal("individual retirement stopped the standing service")
	}
}

func TestStandingSupervisorProtectsSupervisorNotSubjectAndOwnerWaits(t *testing.T) {
	for _, status := range []string{"queued", "running", "parked", "awaiting_capacity", "unknown-new-owner-state"} {
		t.Run(status, func(t *testing.T) {
			f := newSupervisionFixture(t)
			row := effort("productive-orchestrator")
			row.WaitRef = "subject-test-producer-wait"
			f.owner.rows = []EffortObservation{row}
			f.tick(t)
			f.dispatch(t)
			f.agent.getRuns["wake-run-1"].Status = status
			f.owner.rows[0].EvidenceRevision = "fresh-acceptance-failure"
			f.now = f.now.Add(24 * time.Hour)
			state := f.tick(t)
			if state.Pending == nil || state.Pending.RunID != "wake-run-1" || len(f.agent.createRunCalls) != 1 || f.queue.enqueues != 1 {
				t.Fatal("occupied supervisor admitted overlap")
			}
		})
	}
}

func TestStandingSupervisorRecoveryRetainsUncertainDispatch(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.rows = []EffortObservation{effort("effort-a")}
	f.tick(t)
	f.agent.createRunErr = errors.New("response lost after dispatch")
	if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err == nil {
		t.Fatal("expected lost response")
	}
	wakeRequest := f.agent.createRunCalls[0]
	if wakeRequest.IdempotencyKey == "" {
		t.Fatal("dispatch lacks stable wake identity")
	}
	// Simulate the ordinary queue dropping an uncertain entry and process restart.
	// The retained ordinary request is replayed once with the same key; the
	// allowance reservation and pending wake remain held while the owner result
	// is still unavailable.
	f.queue.status = TeamExecutionStatus{}
	f.s = &StandingSupervisor{Owner: f.owner, State: f.s.State, Queue: f.queue, Agent: f.agent, Config: f.s.Config, Prompt: f.s.Prompt, Root: f.s.Root, Now: f.s.Now}
	f.now = f.now.Add(time.Hour)
	if state := f.tick(t); state.Status != "uncertain" {
		t.Fatalf("lost dispatch status = %s", state.Status)
	}
	state := f.s.State
	stored, err := state.Load("supervisors", "leader")
	if err != nil || stored.Pending == nil || stored.Pending.DispatchReplayAttempts != 1 || len(f.agent.createRunCalls) != 2 || f.queue.enqueues != 1 {
		t.Fatalf("restart did not perform one fenced replay: state=%+v calls=%d enqueues=%d err=%v", stored, len(f.agent.createRunCalls), f.queue.enqueues, err)
	}
	f.agent.listRunsResp = &ListRunsResponse{Runs: []*Run{{ID: "wake-run-1", Tag: *wakeRequest.Tag}}}
	f.agent.getRunErr = errors.New("owner outage")
	if _, err := f.s.Tick(context.Background(), "supervisors", "leader"); err == nil {
		t.Fatal("outage not reported")
	}
	f.agent.getRunErr = nil
	f.agent.getRuns["wake-run-1"].Status = "complete"
	if state := f.tick(t); state.Pending != nil {
		t.Fatal("exact terminal owner result did not recover reservation")
	}
	if f.prompts != 1 {
		t.Fatal("recovery repeated orientation/inference")
	}
}

func TestStandingSupervisorUnavailableMissingAndConflictingSources(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.rows = []EffortObservation{effort("effort-a")}
	f.tick(t)
	f.owner.err = errors.New("AM unavailable")
	if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err == nil {
		t.Fatal("owner outage admitted dispatch")
	}
	f.owner.err = nil
	f.owner.rows = nil
	f.dispatch(t)
	state := f.tick(t)
	if state.Efforts["effort-a"].Retired || state.Pending != nil {
		t.Fatal("removal inferred successful retirement or kept stale wake")
	}
	f.owner.rows = []EffortObservation{effort("conflict"), effort("conflict")}
	if _, err := f.s.Tick(context.Background(), "supervisors", "leader"); err == nil {
		t.Fatal("conflicting IDs accepted")
	}
	f.owner.rows = []EffortObservation{{ID: "malformed", Eligible: true}}
	if _, err := f.s.Tick(context.Background(), "supervisors", "leader"); err == nil {
		t.Fatal("malformed eligible source accepted")
	}
	if len(f.agent.createRunCalls) != 0 || f.prompts != 0 {
		t.Fatal("unavailable evidence bought inference")
	}
}

func TestStandingSupervisorFairnessAndBoundedHealthySampling(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.rows = []EffortObservation{effort("a"), effort("b")}
	f.cfg.Supervision.HealthySampleIntervalSeconds, f.cfg.Supervision.MaxHealthySamplesPerWake = 120, 1
	if f.tick(t).Pending.Efforts[0].ID != "a" {
		t.Fatal("unstable selection")
	}
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.agent.getRuns["wake-run-1"].Status = "complete"
	f.now = f.now.Add(time.Minute)
	if f.tick(t).Pending.Efforts[0].ID != "b" {
		t.Fatal("second effort starved")
	}
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.now = f.now.Add(2 * time.Minute)
	state := f.tick(t)
	if state.Pending == nil || len(state.Pending.Efforts) != 1 || len(state.Pending.SampledEffortIDs) != 1 || state.Pending.SampledEffortIDs[0] != "a" {
		t.Fatal("due healthy sample not bounded or selected")
	}
	f.dispatch(t)
	f.receipt(t, "sample")
	f.now = f.now.Add(2 * time.Minute)
	state = f.tick(t)
	if state.Pending == nil || state.Pending.SampledEffortIDs[0] != "b" {
		t.Fatal("healthy sampling starved the second effort")
	}
	if !strings.Contains(f.agent.createTaskCalls[2].Description, "healthySampleEffortIds") {
		t.Fatal("sampling reason missing from context")
	}
}

func TestStandingSupervisorDiagnosticSampleDoesNotRequireSteeringGrant(t *testing.T) {
	f := newSupervisionFixture(t)
	row := effort("observation-without-business-grant")
	row.ObservationOnly, row.Freshness = true, "fresh"
	f.owner.rows = []EffortObservation{row}
	f.cfg.Supervision.HealthySampleIntervalSeconds, f.cfg.Supervision.MaxHealthySamplesPerWake = 120, 1
	f.tick(t)
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.agent.getRuns["wake-run-1"].Status = "complete"
	f.now = f.now.Add(time.Minute)
	f.tick(t)
	f.now = f.now.Add(3 * time.Minute)
	state := f.tick(t)
	if state.Pending == nil || len(state.Pending.SampledEffortIDs) != 1 || !state.Pending.Efforts[0].ObservationOnly {
		t.Fatal("diagnostic allowance did not admit due observation-only sample without granting steering", state)
	}
}

func TestStandingSupervisorRecoveryFollowupIsBoundedAndNotHealthySampling(t *testing.T) {
	f := newSupervisionFixture(t)
	row := effort("any-recovery")
	row.RecoveryVerificationPending = true
	f.owner.rows = []EffortObservation{row}
	f.cfg.Supervision.DiagnosticAllowance.MaxWakesPerWindow = 2
	f.tick(t)
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.agent.getRuns["wake-run-1"].Status = "complete"
	if state := f.tick(t); state.Pending != nil {
		t.Fatal("pending verification bypassed cooldown", state)
	}
	f.now = f.now.Add(time.Minute)
	state := f.tick(t)
	if state.Pending == nil || len(state.Pending.SampledEffortIDs) != 0 || !state.Pending.Efforts[0].RecoveryVerificationPending {
		t.Fatal("unchanged recovery lost follow-up or fabricated healthy sample", state)
	}
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.now = f.now.Add(time.Minute)
	if state := f.tick(t); state.Pending != nil || state.Status != "allowance-wait" {
		t.Fatal("recovery follow-up bypassed cumulative allowance", state)
	}
}

func TestStandingSupervisorRecoveryFollowupStopsAtOwnerWaitOrInvalidAuthority(t *testing.T) {
	for _, variant := range []string{"owner-wait", "observation-only", "stale", "withdrawn"} {
		t.Run(variant, func(t *testing.T) {
			f := newSupervisionFixture(t)
			row := effort("any-recovery")
			row.RecoveryVerificationPending = true
			f.owner.rows = []EffortObservation{row}
			f.tick(t)
			f.dispatch(t)
			f.receipt(t, "quiet")
			f.agent.getRuns["wake-run-1"].Status = "complete"
			f.tick(t)
			switch variant {
			case "owner-wait":
				row.RecoveryVerificationPending = false
			case "observation-only":
				row.ObservationOnly = true
			case "stale":
				row.Freshness = "stale"
			case "withdrawn":
				row.Retired, row.Eligible = true, false
			}
			f.owner.rows = []EffortObservation{row}
			f.now = f.now.Add(time.Minute)
			if state := f.tick(t); state.Pending != nil {
				t.Fatal("closed or unauthorized verification kept buying unchanged wakes", state)
			}
		})
	}
}

func TestStandingSupervisorQueuedRecoveryFollowupRevalidatesOwnerState(t *testing.T) {
	for _, variant := range []string{"verified", "grant-revoked", "stale"} {
		t.Run(variant, func(t *testing.T) {
			f := newSupervisionFixture(t)
			row := effort("any-recovery")
			row.RecoveryVerificationPending = true
			f.owner.rows = []EffortObservation{row}
			f.tick(t)
			f.dispatch(t)
			f.receipt(t, "quiet")
			f.agent.getRuns["wake-run-1"].Status = "complete"
			f.tick(t)
			f.now = f.now.Add(time.Minute)
			if f.tick(t).Pending == nil {
				t.Fatal("follow-up not queued")
			}
			switch variant {
			case "verified":
				row.RecoveryVerificationPending = false
			case "grant-revoked":
				row.ObservationOnly = true
			case "stale":
				row.Freshness = "stale"
			}
			// Assessment/authority-only changes deliberately need not alter the
			// subject revision; dispatch must check the current eligibility too.
			f.owner.rows = []EffortObservation{row}
			f.dispatch(t)
			state, err := f.s.State.Load("supervisors", "leader")
			if err != nil || state.Pending != nil || len(f.agent.createRunCalls) != 1 || len(f.agent.createTaskCalls) != 1 {
				t.Fatal("closed queued follow-up bought another judgment", state, err)
			}
		})
	}
}

func TestStandingSupervisorPaginationRevalidatesSelectedPage(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.pages = map[string]*EffortDiscovery{
		"":       {NextCursor: "page-2", Coverage: "bounded"},
		"page-2": {Efforts: []EffortObservation{effort("later-page-effort")}, Coverage: "complete"},
	}
	f.tick(t)
	if f.tick(t).Pending == nil {
		t.Fatal("later discovery page not served")
	}
	f.dispatch(t)
	if len(f.agent.createRunCalls) != 1 {
		t.Fatal("dispatch failed to revalidate its own discovery page")
	}
}

func TestStandingSupervisorStorageAndDisableFence(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.rows = []EffortObservation{effort("effort-a")}
	f.tick(t)
	f.cfg.Enabled = false
	if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err == nil {
		t.Fatal("disabled heartbeat dispatched queued inference")
	}
	f.cfg.Enabled = true
	path := f.s.State.(FileSupervisionStateStore).path("supervisors", "leader")
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Tick(context.Background(), "supervisors", "leader"); err == nil {
		t.Fatal("corrupt reservation treated as empty")
	}
	if len(f.agent.createRunCalls) != 0 {
		t.Fatal("storage failure admitted effects")
	}
}

func TestStandingSupervisorDiagnosticAllowancePersistsAcrossRestart(t *testing.T) {
	f := newSupervisionFixture(t)
	f.cfg.Supervision.DiagnosticAllowance.MaxWakesPerWindow = 1
	f.owner.rows = []EffortObservation{effort("a"), effort("b")}
	f.tick(t)
	f.dispatch(t)
	f.agent.getRuns["wake-run-1"].Status = "complete"
	f.now = f.now.Add(time.Minute)
	state := f.tick(t)
	if state.Status != "allowance-wait" || state.WakesInWindow != 1 || state.WakeAttempts != 1 || state.Pending != nil {
		t.Fatalf("allowance did not block second wake: %+v", state)
	}
	if state.AccountingRef != "test:standing-supervision" || state.DiscoveryReads < 2 || state.LastWake.AccountingRef != state.AccountingRef {
		t.Fatal("shared observation/inference attribution missing")
	}
	f.s = &StandingSupervisor{Owner: f.owner, State: f.s.State, Queue: f.queue, Agent: f.agent, Config: f.s.Config, Prompt: f.s.Prompt, Root: f.s.Root, Now: f.s.Now}
	if state := f.tick(t); state.Status != "allowance-wait" {
		t.Fatal("restart reset allowance")
	}
	f.now = f.now.Add(-time.Hour)
	if state := f.tick(t); state.Status != "allowance-wait" {
		t.Fatal("clock rollback reset allowance")
	}
	f.now = state.AllowanceResumesAt
	if state := f.tick(t); state.Pending == nil || state.Pending.Efforts[0].ID != "b" {
		t.Fatal("allowance recovery failed to serve waiting effort")
	}
}

func TestStandingSupervisorSchedulerUsesCheapAdmission(t *testing.T) {
	f := newSupervisionFixture(t)
	scheduler := NewScheduler(nil, nil, supervisionConfigFake{f.cfg}, nil)
	scheduler.SetEffortSupervisor(f.s)
	scheduler.executeHeartbeat(context.Background(), "supervisors", "leader")
	if f.owner.calls != 1 || f.prompts != 0 || f.queue.enqueues != 0 {
		t.Fatal("scheduler did not use discovery-only idle")
	}
	f.owner.rows = []EffortObservation{effort("new-effort-after-start")}
	scheduler.executeHeartbeat(context.Background(), "supervisors", "leader")
	if f.queue.enqueues != 1 {
		t.Fatal("scheduler did not admit arriving effort")
	}
	f.cfg.Enabled = false
	scheduler.executeHeartbeat(context.Background(), "supervisors", "leader")
	if f.owner.calls != 2 {
		t.Fatal("disabled schedule ignored its lifecycle gate")
	}
}

func TestStandingSupervisorTerminalWithoutReceiptReopensWithinAllowance(t *testing.T) {
	for _, status := range []string{"failed", "cancelled", "complete"} {
		t.Run(status, func(t *testing.T) {
			f := newSupervisionFixture(t)
			f.owner.rows = []EffortObservation{effort("a")}
			f.cfg.Supervision.DiagnosticAllowance.MaxWakesPerWindow = 2
			f.tick(t)
			f.dispatch(t)
			f.agent.getRuns["wake-run-1"].Status = status
			state := f.tick(t)
			row := state.Efforts["a"]
			if state.Pending != nil || row.ServedRevision != "" || !row.LastSampleAt.IsZero() || row.Disposition != "unassessed-reopen" || state.WakesInWindow != 1 {
				t.Fatalf("terminal attempt fabricated coverage or lost its charge: %+v", state)
			}
			f.now = f.now.Add(time.Minute)
			if f.tick(t).Pending == nil {
				t.Fatal("unchanged unassessed evidence did not reopen after cooldown")
			}
			f.dispatch(t)
			state = f.tick(t)
			if state.Status != "allowance-wait" || state.WakesInWindow != 2 || state.Pending != nil {
				t.Fatal("recovery escaped the original allowance")
			}
		})
	}
}

func TestStandingSupervisorCoverageRequiresMatchingSampleReceipt(t *testing.T) {
	for _, receiptKind := range []string{"sample", "quiet", "wrong-wake", "wrong-run", "wrong-target", "missing"} {
		t.Run(receiptKind, func(t *testing.T) {
			f := newSupervisionFixture(t)
			f.owner.rows = []EffortObservation{effort("a")}
			f.cfg.Supervision.HealthySampleIntervalSeconds, f.cfg.Supervision.MaxHealthySamplesPerWake = 60, 1
			f.tick(t)
			f.dispatch(t)
			f.receipt(t, "quiet")
			f.agent.getRuns["wake-run-1"].Status = "complete"
			initial := f.tick(t).Efforts["a"]
			if initial.ServedRevision == "" || initial.AssessmentID == "" || !initial.LastSampleAt.IsZero() {
				t.Fatal("initial assessment/sample distinction lost")
			}
			f.now = f.now.Add(time.Minute)
			if len(f.tick(t).Pending.SampledEffortIDs) != 1 {
				t.Fatal("sample not selected")
			}
			f.dispatch(t)
			if receiptKind != "missing" {
				f.receipt(t, "sample")
				switch receiptKind {
				case "quiet":
					f.owner.receipts["a"].Disposition = "quiet"
				case "wrong-wake":
					f.owner.receipts["a"].WakeID = "other"
				case "wrong-run":
					f.owner.receipts["a"].RunID = "other"
				case "wrong-target":
					f.owner.receipts["a"].TargetRevisions["a"] = "other"
				}
			}
			state := f.tick(t)
			if state.Pending != nil {
				t.Fatal("terminal reservation not released")
			}
			if got := !state.Efforts["a"].LastSampleAt.IsZero(); got != (receiptKind == "sample") {
				t.Fatal("sample completion not backed by exact sample receipt")
			}
			if state.WakesInWindow != 2 {
				t.Fatal("receipt processing changed allowance charges")
			}
		})
	}
}

func TestStandingSupervisorDeclinedSampleRespectsIntervalAcrossRestart(t *testing.T) {
	for _, variant := range []struct{ changed, legacy bool }{{}, {changed: true}, {legacy: true}} {
		t.Run(fmt.Sprintf("changed=%t/legacy=%t", variant.changed, variant.legacy), func(t *testing.T) {
			f := newSupervisionFixture(t)
			f.owner.rows = []EffortObservation{effort("stopped-subject")}
			f.cfg.Supervision.HealthySampleIntervalSeconds, f.cfg.Supervision.MaxHealthySamplesPerWake = 3600, 1
			f.tick(t)
			f.dispatch(t)
			f.receipt(t, "quiet")
			f.agent.getRuns["wake-run-1"].Status = "complete"
			f.tick(t)
			f.now = f.now.Add(time.Hour)
			if len(f.tick(t).Pending.SampledEffortIDs) != 1 {
				t.Fatal("due sample not selected")
			}
			f.dispatch(t)
			f.receipt(t, "unknown") // Stopped/unknown runtime cannot establish a healthy sample.
			state := f.tick(t)
			if !state.Efforts["stopped-subject"].LastSampleAt.IsZero() {
				t.Fatal("declined sample fabricated successful coverage")
			}
			if variant.legacy {
				row := state.Efforts["stopped-subject"]
				row.LastSampleAttemptAt = time.Time{}
				state.Efforts["stopped-subject"] = row
				if err := f.s.State.Save("supervisors", "leader", state); err != nil {
					t.Fatal(err)
				}
			}
			// Reconstruct the scheduler while retaining only its durable state store.
			old := f.s
			f.s = &StandingSupervisor{Owner: old.Owner, State: old.State, Queue: old.Queue, Agent: old.Agent, Config: old.Config, Prompt: old.Prompt, Root: old.Root, Now: old.Now}
			f.now = f.now.Add(5 * time.Minute)
			if variant.changed {
				f.owner.rows[0].EvidenceRevision = "new-resolution-evidence"
			}
			state = f.tick(t)
			if variant.changed {
				if state.Pending == nil || len(state.Pending.SampledEffortIDs) != 0 {
					t.Fatal("changed evidence must admit ordinary judgment without a sample claim")
				}
				return
			}
			if state.Pending != nil || len(f.agent.createRunCalls) != 2 {
				t.Fatal("unchanged declined sample reopened at recovery heartbeat")
			}
			f.now = f.now.Add(55 * time.Minute)
			if state = f.tick(t); state.Pending == nil || len(state.Pending.SampledEffortIDs) != 1 {
				t.Fatal("unfulfilled sample must remain due at the next sampling interval")
			}
		})
	}
}

func TestStandingSupervisorQuotaFailureKeepsChargeAndUnchangedEvidence(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.rows = []EffortObservation{effort("a")}
	f.cfg.Supervision.DiagnosticAllowance.MaxWakesPerWindow = 1
	f.tick(t)
	f.dispatch(t)
	f.agent.getRuns["wake-run-1"].Status = "failed"
	f.agent.getRuns["wake-run-1"].Error = "quota exhausted; owner retry eligibility remains unknown"
	f.now = f.now.Add(time.Minute)
	state := f.tick(t)
	if state.Pending != nil || state.Status != "allowance-wait" || state.WakesInWindow != 1 || state.Efforts["a"].ServedRevision != "" || state.LastWake.TerminalError == "" || state.LastWake.TerminalStatus != "failed" {
		t.Fatal("quota failure lost its owner evidence or charge")
	}
	if len(f.agent.createRunCalls) != 1 {
		t.Fatal("quota failure caused immediate replacement")
	}
}

type supervisionConfigFake struct{ cfg *store.HeartbeatConfig }

func (s supervisionConfigFake) GetHeartbeatConfig(context.Context, string, string) (*store.HeartbeatConfig, error) {
	return s.cfg, nil
}
