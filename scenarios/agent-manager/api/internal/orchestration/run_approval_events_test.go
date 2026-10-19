package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

func approvalTestDiff(sandboxID uuid.UUID, unified string) *sandbox.DiffResult {
	sum := sha256.Sum256([]byte(unified))
	return &sandbox.DiffResult{SandboxID: sandboxID, UnifiedDiff: unified, PatchSHA256: hex.EncodeToString(sum[:]), Stats: sandbox.DiffStats{FilesChanged: 3, TotalBytes: int64(len(unified))}}
}

func approvalTestEventID(t *testing.T, o *Orchestrator, run *domain.Run, diff *sandbox.DiffResult) string {
	t.Helper()
	event, err := o.runApprovalEvent(context.Background(), run, diff)
	if err != nil {
		t.Fatal(err)
	}
	return operatordecision.EventID(event)
}

func TestRunApprovalGenerationNormalizesOnlyCompletedArchiveMetadata(t *testing.T) {
	sandboxID := uuid.New()
	run := &domain.Run{ID: uuid.New(), SandboxID: &sandboxID, EndedAt: verifiedEmptyTestTimePtr(time.Unix(100, 0).UTC())}
	live := approvalTestDiff(sandboxID, "known sandbox patch")
	live.ArchiveState = ""
	complete := *live
	complete.ArchiveState = sandbox.ArchiveStateComplete
	want, err := runApprovalGeneration(run, live)
	if err != nil {
		t.Fatal(err)
	}
	got, err := runApprovalGeneration(run, &complete)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("live and completed archive generations differ: %s != %s", got, want)
	}
	notCaptured := *live
	notCaptured.ArchiveState = sandbox.ArchiveStateNotCaptured
	unknown := *live
	unknown.ArchiveState = "future_state"
	for name, diff := range map[string]*sandbox.DiffResult{"not_captured": &notCaptured, "unknown": &unknown} {
		digest, err := runApprovalGeneration(run, diff)
		if err != nil {
			t.Fatal(err)
		}
		if digest == want {
			t.Fatalf("%s archive state collapsed into live generation", name)
		}
	}
	changed := *live
	changed.PatchSHA256 = approvalTestDiff(sandboxID, "different contents").PatchSHA256
	digest, err := runApprovalGeneration(run, &changed)
	if err != nil {
		t.Fatal(err)
	}
	if digest == want {
		t.Fatal("changed patch reused prior generation")
	}
}

func verifiedEmptyTestTimePtr(value time.Time) *time.Time { return &value }

func TestRunApprovalUnknownEmptyLookingDiffDoesNotClaimZeroFiles(t *testing.T) {
	sandboxID := uuid.New()
	o := &Orchestrator{webConsoleUIBase: "http://localhost:21238"}
	run := &domain.Run{ID: uuid.New(), SandboxID: &sandboxID, Label: "unknown empty diff"}
	diff := &sandbox.DiffResult{SandboxID: sandboxID, PatchSHA256: hex.EncodeToString(make([]byte, sha256.Size)), Files: nil, Stats: sandbox.DiffStats{}, EvidenceComplete: false}
	event, err := o.runApprovalEvent(context.Background(), run, diff)
	if err != nil {
		t.Fatal(err)
	}
	question := event.Payload["question"].(string)
	details := event.Payload["technical_details"].(string)
	if strings.Contains(question, "0 files") || strings.Contains(details, "0 files") || !strings.Contains(question, "file count is unavailable") || !strings.Contains(details, "file count is unavailable") {
		t.Fatalf("unknown empty-looking diff was presented as verified zero: question=%q details=%q", question, details)
	}
}

// TestRunApprovalEventShape pins the emitted event to notification-hub's
// *.decision_requested.v1 producer contract (decisionFacts): a non-empty
// question, >=2 options, a recommended option key, and a run_id correlation.
func TestRunApprovalEventShape(t *testing.T) {
	runID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	o := &Orchestrator{
		webConsoleUIBase: "http://localhost:21238/",
		clock:            func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	sandboxID := uuid.New()
	run := &domain.Run{ID: runID, SandboxID: &sandboxID, ChangedFiles: 3, Label: "Prepare release notes"}

	ev, err := o.runApprovalEvent(context.Background(), run, approvalTestDiff(sandboxID, "diff v1"))
	if err != nil {
		t.Fatal(err)
	}

	if ev.EventType != RunApprovalDecisionEventType {
		t.Fatalf("event type = %q, want %q", ev.EventType, RunApprovalDecisionEventType)
	}
	if ev.Source != "agent-manager" {
		t.Fatalf("source = %q, want agent-manager", ev.Source)
	}
	if !strings.HasSuffix(ev.EventType, ".decision_requested.v1") {
		t.Fatalf("event type %q must ride the generic decision suffix", ev.EventType)
	}

	p := ev.Payload
	if q, _ := p["question"].(string); strings.TrimSpace(q) == "" || !strings.Contains(q, "3 files") {
		t.Fatalf("question missing or wrong file count: %q", p["question"])
	}
	if subject, _ := p["subject"].(string); !strings.Contains(subject, "Prepare release notes") {
		t.Fatalf("subject lacks the persisted run label: %q", subject)
	}
	if big, _ := p["big_picture"].(string); !strings.Contains(big, "Prepare release notes") {
		t.Fatalf("big_picture lacks the persisted run label: %q", big)
	}
	if p["recommended"] != "approve" {
		t.Fatalf("recommended = %v, want approve", p["recommended"])
	}
	if p["reversible"] != true {
		t.Fatalf("reversible = %v, want true", p["reversible"])
	}
	if big, _ := p["big_picture"].(string); !strings.Contains(big, "canonical repository") {
		t.Fatalf("big_picture lacks the practical effect: %q", big)
	}
	if details, _ := p["technical_details"].(string); !strings.Contains(details, "3 files") || !strings.Contains(details, "sandbox diff") || !strings.Contains(details, "not broader task or goal acceptance") {
		t.Fatalf("technical_details lacks bounded review context: %q", details)
	}
	if _, hasDefault := p["default_answer"]; hasDefault {
		t.Fatalf("run approval must not set a default_answer (never auto-apply)")
	}

	opts, ok := p["options"].([]any)
	if !ok || len(opts) != 2 {
		t.Fatalf("options = %v, want 2 entries", p["options"])
	}
	keys := map[string]bool{}
	for _, o := range opts {
		m, _ := o.(map[string]any)
		key, _ := m["key"].(string)
		if key == "" || m["label"] == "" {
			t.Fatalf("option missing key/label: %v", o)
		}
		keys[key] = true
	}
	if !keys["approve"] || !keys["reject"] {
		t.Fatalf("options must include approve and reject, got %v", keys)
	}

	corr, ok := p["correlation"].(map[string]any)
	if !ok || corr["run_id"] != runID.String() {
		t.Fatalf("correlation.run_id = %v, want %s", p["correlation"], runID.String())
	}
	if url, _ := p["context_url"].(string); !strings.Contains(url, runID.String()) {
		t.Fatalf("context_url %q must reference the run id", p["context_url"])
	}
}

// TestRunApprovalEventOmitsLinkWithoutUIBase ensures a missing web-console base
// drops context_url rather than emitting an empty link.
func TestRunApprovalEventOmitsLinkWithoutUIBase(t *testing.T) {
	// No override and no resolvable base: context_url must be omitted.
	restore := resolveExternalURLForNotice
	resolveExternalURLForNotice = func(context.Context, string) (string, error) { return "", nil }
	t.Cleanup(func() { resolveExternalURLForNotice = restore })

	o := &Orchestrator{clock: func() time.Time { return time.Unix(1000, 0).UTC() }}
	sandboxID := uuid.New()
	diff := approvalTestDiff(sandboxID, "diff")
	diff.Stats.FilesChanged = 1
	ev, err := o.runApprovalEvent(context.Background(), &domain.Run{ID: uuid.New(), SandboxID: &sandboxID, ChangedFiles: 1}, diff)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ev.Payload["context_url"]; ok {
		t.Fatalf("context_url must be omitted when no UI base is resolved")
	}
	if q, _ := ev.Payload["question"].(string); !strings.Contains(q, "1 file") || strings.Contains(q, "1 files") {
		t.Fatalf("singular file count wrong: %q", q)
	}
}

// TestRunApprovalEventOccurredIsDeterministicForNeedsReview pins that a
// needs_review run (no ApprovedAt yet) derives Occurred from the stable EndedAt
// transition time, so two builds across a redelivery produce the SAME EventID —
// what makes the hub deduplicate instead of raising a duplicate ask. It also
// pins that a moving wall clock does NOT change the EventID.
func TestRunApprovalEventOccurredIsDeterministicForNeedsReview(t *testing.T) {
	runID := uuid.MustParse("22222222-3333-4444-5555-666666666666")
	ended := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	sandboxID := uuid.New()
	run := &domain.Run{ID: runID, SandboxID: &sandboxID, ChangedFiles: 2, EndedAt: &ended}
	diff := approvalTestDiff(sandboxID, "diff")
	diff.Generated = time.Unix(111, 0)
	diffLater := *diff
	diffLater.Generated = time.Unix(222, 0)

	// Two Orchestrators with DIFFERENT wall clocks — EventID must be identical.
	o1 := &Orchestrator{webConsoleUIBase: "http://localhost:21238/", clock: func() time.Time { return time.Unix(1000, 0).UTC() }}
	o2 := &Orchestrator{webConsoleUIBase: "http://localhost:21238/", clock: func() time.Time { return time.Unix(9999, 0).UTC() }}
	ev1, err := o1.runApprovalEvent(context.Background(), run, diff)
	if err != nil {
		t.Fatal(err)
	}
	ev2, err := o2.runApprovalEvent(context.Background(), run, &diffLater)
	if err != nil {
		t.Fatal(err)
	}
	if !ev1.Occurred.Equal(ended) {
		t.Fatalf("Occurred must be the stable EndedAt, got %v want %v", ev1.Occurred, ended)
	}
	if operatordecision.EventID(ev1) != operatordecision.EventID(ev2) {
		t.Fatalf("EventID must be deterministic across wall clocks for a needs_review run: %s != %s", operatordecision.EventID(ev1), operatordecision.EventID(ev2))
	}
}

func TestRunApprovalResolutionRequiresCurrentPendingGeneration(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "review generation", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending, ChangedFiles: 3, EndedAt: &ended}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	provider := mocks.NewFakeSandboxProvider()
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return approvalTestDiff(id, "current sandbox contents"), nil
	}
	approveCalls := 0
	var appliedPatchSHA256 string
	provider.ApproveFunc = func(_ context.Context, req sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
		approveCalls++
		appliedPatchSHA256 = req.ExpectedPatchSHA256
		return &sandbox.ApproveResult{Success: true, Applied: 3}, nil
	}
	publisher := &runReviewEventPublisher{}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider), WithApprovalEventPublisher(publisher), WithWebConsoleUIBase("http://localhost:21238"))
	current, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	currentRef := approvalTestEventID(t, o, current, approvalTestDiff(sandboxID, "current sandbox contents"))
	stale := *current
	stale.ChangedFiles = 2
	staleRef := approvalTestEventID(t, o, &stale, approvalTestDiff(sandboxID, "current sandbox contents"))
	if staleRef == currentRef {
		t.Fatal("fixture did not create distinct decision generations")
	}

	base := supervision.DecisionResolution{State: "answered", Answer: "approve", AnsweredBy: "verified-human", SourceEventType: RunApprovalDecisionEventType, Correlation: map[string]string{"run_id": run.ID.String()}}
	for _, tc := range []struct {
		name        string
		ref         string
		actor       string
		wantErr     bool
		wantApprove int
	}{
		{name: "stale generation is ignored", ref: staleRef, actor: "verified-human", wantApprove: 0},
		{name: "changed sandbox contents are a different generation", actor: "verified-human", wantApprove: 0},
		{name: "missing generation is ignored", ref: "", actor: "verified-human", wantApprove: 0},
		{name: "current generation requires approver", ref: currentRef, actor: "", wantErr: true, wantApprove: 0},
		{name: "current pending generation applies", ref: currentRef, actor: "verified-human", wantApprove: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Recreate the pending run for each resolution so each case is isolated.
			caseRun := *run
			caseRun.ID = uuid.New()
			caseRun.TaskID = task.ID
			caseRun.SandboxID = &sandboxID
			caseRun.ApprovalState = domain.ApprovalStatePending
			if err := repos.Runs.Create(ctx, &caseRun); err != nil {
				t.Fatal(err)
			}
			persisted, err := repos.Runs.Get(ctx, caseRun.ID)
			if err != nil {
				t.Fatal(err)
			}
			ref := tc.ref
			if tc.name == "current pending generation applies" || tc.name == "current generation requires approver" {
				ref = approvalTestEventID(t, o, persisted, approvalTestDiff(sandboxID, "current sandbox contents"))
			} else if tc.name == "stale generation is ignored" {
				old := *persisted
				old.ChangedFiles = 2
				ref = approvalTestEventID(t, o, &old, approvalTestDiff(sandboxID, "current sandbox contents"))
			} else if tc.name == "changed sandbox contents are a different generation" {
				oldDiff := approvalTestDiff(sandboxID, "contents from the earlier ask")
				ref = approvalTestEventID(t, o, persisted, oldDiff)
			}
			res := base
			res.Correlation = map[string]string{"run_id": caseRun.ID.String()}
			res.RequesterRef = ref
			res.AnsweredBy = tc.actor
			handled, err := o.ResolveRunApprovalFromHub(ctx, res)
			if !handled || (err != nil) != tc.wantErr {
				t.Fatalf("handled=%v err=%v, want handled and wantErr=%v", handled, err, tc.wantErr)
			}
			if approveCalls != tc.wantApprove {
				t.Fatalf("approve calls=%d want %d", approveCalls, tc.wantApprove)
			}
			if tc.name == "current pending generation applies" && appliedPatchSHA256 != approvalTestDiff(sandboxID, "current sandbox contents").PatchSHA256 {
				t.Fatalf("approval patch precondition=%q want current diff digest", appliedPatchSHA256)
			}
			if tc.wantApprove == 0 {
				got, err := repos.Runs.Get(ctx, caseRun.ID)
				if err != nil {
					t.Fatal(err)
				}
				if got.Status != domain.RunStatusNeedsReview || got.ApprovalState != domain.ApprovalStatePending {
					t.Fatalf("run changed after refused resolution: status=%s approval=%s", got.Status, got.ApprovalState)
				}
			}
		})
	}
}

func TestRunApprovalResolutionDoesNotTurnCrashReviewIntoApproval(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "crash review", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStateNone, ChangedFiles: 1, EndedAt: &ended, StopReason: domain.RunStopReasonCrash}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	provider := mocks.NewFakeSandboxProvider()
	approveCalls := 0
	provider.ApproveFunc = func(context.Context, sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
		approveCalls++
		return &sandbox.ApproveResult{Success: true}, nil
	}
	publisher := &runReviewEventPublisher{}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider), WithApprovalEventPublisher(publisher), WithWebConsoleUIBase("http://localhost:21238"))
	ref := approvalTestEventID(t, o, run, approvalTestDiff(sandboxID, "crash sandbox contents"))
	handled, err := o.ResolveRunApprovalFromHub(ctx, supervision.DecisionResolution{State: "answered", Answer: "approve", AnsweredBy: "verified-human", SourceEventType: RunApprovalDecisionEventType, RequesterRef: ref, Correlation: map[string]string{"run_id": run.ID.String()}})
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if approveCalls != 0 {
		t.Fatalf("crash review triggered %d approvals", approveCalls)
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusNeedsReview || got.ApprovalState != domain.ApprovalStateNone {
		t.Fatalf("crash review changed: status=%s approval=%s", got.Status, got.ApprovalState)
	}
}

func TestRunApprovalResolutionCurrentGenerationCanReject(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "reject review", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending, ChangedFiles: 1, EndedAt: &ended}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	provider := mocks.NewFakeSandboxProvider()
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return approvalTestDiff(id, "current sandbox contents"), nil
	}
	rejectCalls, deleteCalls := 0, 0
	var rejected sandbox.RejectRequest
	provider.RejectReviewedFunc = func(_ context.Context, req sandbox.RejectRequest) error { rejected = req; rejectCalls++; return nil }
	provider.DeleteFunc = func(context.Context, uuid.UUID) error { deleteCalls++; return nil }
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider), WithWebConsoleUIBase("http://localhost:21238"))
	current, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	ref := approvalTestEventID(t, o, current, approvalTestDiff(sandboxID, "current sandbox contents"))
	handled, err := o.ResolveRunApprovalFromHub(ctx, supervision.DecisionResolution{State: "answered", Answer: "reject", AnsweredBy: "verified-human", SourceEventType: RunApprovalDecisionEventType, RequesterRef: ref, Correlation: map[string]string{"run_id": run.ID.String()}})
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if rejectCalls != 1 || deleteCalls != 1 {
		t.Fatalf("reject/delete calls=%d/%d, want 1/1", rejectCalls, deleteCalls)
	}
	if rejected.SandboxID != sandboxID || rejected.ExpectedPatchSHA256 != approvalTestDiff(sandboxID, "current sandbox contents").PatchSHA256 {
		t.Fatalf("reviewed reject request=%+v, want same sandbox and current patch digest", rejected)
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusFailed || got.ApprovalState != domain.ApprovalStateRejected {
		t.Fatalf("rejected run status=%s approval=%s", got.Status, got.ApprovalState)
	}
}

func TestRunApprovalResolutionRejectProviderFailureWithholdsDeleteAndStateChange(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "reject provider failure", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending, ChangedFiles: 1, EndedAt: &ended}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	provider := mocks.NewFakeSandboxProvider()
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return approvalTestDiff(id, "current sandbox contents"), nil
	}
	provider.RejectReviewedFunc = func(context.Context, sandbox.RejectRequest) error { return errors.New("patch changed before reject") }
	deleteCalls := 0
	provider.DeleteFunc = func(context.Context, uuid.UUID) error { deleteCalls++; return nil }
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider), WithWebConsoleUIBase("http://localhost:21238"))
	persisted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	ref := approvalTestEventID(t, o, persisted, approvalTestDiff(sandboxID, "current sandbox contents"))
	handled, err := o.ResolveRunApprovalFromHub(ctx, supervision.DecisionResolution{State: "answered", Answer: "reject", AnsweredBy: "reviewer", SourceEventType: RunApprovalDecisionEventType, RequesterRef: ref, Correlation: map[string]string{"run_id": run.ID.String()}})
	if !handled || err == nil {
		t.Fatalf("handled=%v err=%v, want handled error", handled, err)
	}
	if deleteCalls != 0 {
		t.Fatalf("provider rejection failure still deleted sandbox %d times", deleteCalls)
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusNeedsReview || got.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("provider rejection failure changed run: status=%s approval=%s", got.Status, got.ApprovalState)
	}
}

func TestRunApprovalReviewedRejectWithoutProviderLeavesRunPending(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "reject provider unavailable", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending, ChangedFiles: 1, EndedAt: &ended}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs)
	digest := approvalTestDiff(sandboxID, "current sandbox contents").PatchSHA256
	if err := o.RejectRunWithExpectedPatch(ctx, run.ID, "reviewer", "declined", digest); err == nil {
		t.Fatal("reviewed rejection succeeded without a sandbox provider")
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusNeedsReview || got.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("provider-unavailable rejection changed run: status=%s approval=%s", got.Status, got.ApprovalState)
	}
}
