package orchestration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/modules"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	coreidentity "github.com/vrooli/api-core/identity"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

// failMarker in a task description makes the synthetic runner fail that run.
const failMarker = "FAIL-ME"

// nestedOwnerWindow is the fixture owner's authority lifetime.
const nestedOwnerWindow = 30 * time.Hour

// nestedRunFixture launches real Agent Manager runs through CreateRun with a
// runner that holds every run live until released, so each run's identity
// token can create the next level of children.
type nestedRunFixture struct {
	t        *testing.T
	o        *Orchestrator
	ctx      context.Context
	owner    context.Context
	profile  *domain.AgentProfile
	root     string
	launches chan runner.ExecuteRequest
	release  func()
}

func newNestedRunFixture(t *testing.T, maxChildDepth int) *nestedRunFixture {
	t.Helper()
	ctx := context.Background()
	o := newDeclarationOrchestrator(t)
	newCurrentModelPolicyFixtureOption(t)(o)
	o.identitySecret = []byte("disposable-nested-lineage-secret")
	o.runStateRoot = t.TempDir()
	o.runWaitPoll = 5 * time.Millisecond
	root := t.TempDir()
	o.config.DefaultProjectRoot = root
	o.config.RequireSandboxByDefault = false
	t.Cleanup(o.dispatcher.Close)

	store, err := agentconfig.NewOrchestrationSettingsStore(filepath.Join(t.TempDir(), "orchestration.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := store.Get()
	settings.RunExecution.MaxChildDepth = maxChildDepth
	if err := store.Update(settings); err != nil {
		t.Fatal(err)
	}
	WithOrchestrationSettings(store)(o)

	launches := make(chan runner.ExecuteRequest, 8)
	hold := make(chan struct{})
	var released sync.Once
	release := func() { released.Do(func() { close(hold) }) }
	t.Cleanup(release)
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		select {
		case launches <- req:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if strings.Contains(req.Prompt+req.SystemPrompt, failMarker) {
			return nil, errors.New("synthetic nested run failure")
		}
		select {
		case <-hold:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "synthetic nested run"}}, nil
	}
	o.runners = runner.NewRegistry()
	if err := o.runners.Register(&launchInfoStub{MockRunner: mock, args: []string{"--synthetic-in-process"}, runtimeVersion: "synthetic-in-process/1"}); err != nil {
		t.Fatal(err)
	}
	// The owner's authority outlives one run identity token (24h), as a
	// long effort's operator credential does.
	owner := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "fixture-owner", Scopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}, ExpiresAt: time.Now().Add(nestedOwnerWindow).Truncate(time.Second)}
	o.ownerIdentity = createOwnerVerifier(func(_ context.Context, token string) (coreidentity.Principal, error) {
		if token != "disposable-nested-owner" {
			return coreidentity.Principal{}, fmt.Errorf("unexpected fixture owner credential")
		}
		return owner, nil
	})
	profile, err := o.CreateProfile(ctx, &domain.AgentProfile{Name: "synthetic nested", ProfileKey: "nested-lineage", RoleRef: "code.default", DeclaredScopes: owner.Scopes, MaxTurns: 5, Timeout: time.Minute, NetworkAccess: domain.NetworkAccessNone, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}})
	if err != nil {
		t.Fatal(err)
	}
	return &nestedRunFixture{t: t, o: o, ctx: ctx, owner: coreidentity.WithPrincipal(ctx, owner), profile: profile, root: root, launches: launches, release: release}
}

func (f *nestedRunFixture) task(title string) *domain.Task {
	f.t.Helper()
	created, err := f.o.CreateTask(f.ctx, &domain.Task{Title: title, Description: "synthetic in-process only: " + title, ScopePath: f.root, ProjectRoot: f.root})
	if err != nil {
		f.t.Fatal(err)
	}
	return created
}

// launched waits for the next run to reach its runner.
func (f *nestedRunFixture) launched() runner.ExecuteRequest {
	f.t.Helper()
	select {
	case req := <-f.launches:
		return req
	case <-time.After(10 * time.Second):
		f.t.Fatal("synthetic nested run did not launch")
		return runner.ExecuteRequest{}
	}
}

// createRoot creates a root run as the verified human owner.
func (f *nestedRunFixture) createRoot(charter string, refs ...*eventpb.WorkReference) (*domain.Run, runner.ExecuteRequest) {
	f.t.Helper()
	task := f.task("nested root")
	run, err := f.o.CreateRun(f.owner, CreateRunRequest{TaskID: task.ID, AgentProfileID: &f.profile.ID, PreferredRunner: "claude-code", OwnerToken: "disposable-nested-owner", IdempotencyKey: "nested-root", CharterPath: charter, WorkReferences: refs})
	if err != nil {
		f.t.Fatal(err)
	}
	return run, f.launched()
}

// createChild creates a child of parent with the parent's own run identity,
// exactly as an agent inside the parent run does.
func (f *nestedRunFixture) createChild(parent *domain.Run, launch runner.ExecuteRequest, key, charter string) (*domain.Run, error) {
	task := f.task("nested child " + key)
	return f.o.CreateRun(f.ctx, CreateRunRequest{TaskID: task.ID, AgentProfileID: &f.profile.ID, PreferredRunner: "claude-code", ParentRunID: &parent.ID, RunIdentityToken: launch.Environment["VROOLI_AGENT_IDENTITY_TOKEN"], RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: key, CharterPath: charter})
}

func TestNestedChildRunsRecordDepthAndRefuseBeyondTheCap(t *testing.T) {
	f := newNestedRunFixture(t, 2)
	charter := filepath.Join(t.TempDir(), "CHARTER.md")
	if err := os.WriteFile(charter, []byte("# Charter\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	root, rootLaunch := f.createRoot("")
	if root.Depth != 0 || strings.Contains(rootLaunch.Prompt+rootLaunch.SystemPrompt, "Agent Manager lineage") {
		t.Fatalf("an unchartered root has depth 0 and no lineage header: depth=%d prompt=%q", root.Depth, rootLaunch.Prompt)
	}
	child, err := f.createChild(root, rootLaunch, "nested-child", "")
	if err != nil {
		t.Fatal(err)
	}
	childLaunch := f.launched()
	grandchild, err := f.createChild(child, childLaunch, "nested-grandchild", charter)
	if err != nil {
		t.Fatalf("grandchild through the real admission path: %v", err)
	}
	grandchildLaunch := f.launched()

	for _, want := range []struct {
		run    *domain.Run
		depth  int
		parent *uuid.UUID
	}{{root, 0, nil}, {child, 1, &root.ID}, {grandchild, 2, &child.ID}} {
		saved, err := f.o.runs.Get(f.ctx, want.run.ID)
		if err != nil || saved == nil {
			t.Fatalf("read %s: %v", want.run.ID, err)
		}
		if saved.Depth != want.depth || want.run.Depth != want.depth {
			t.Fatalf("run %s depth persisted=%d returned=%d, want %d", saved.ID, saved.Depth, want.run.Depth, want.depth)
		}
		if (saved.ParentRunID == nil) != (want.parent == nil) || (want.parent != nil && *saved.ParentRunID != *want.parent) {
			t.Fatalf("run %s parent = %v, want %v", saved.ID, saved.ParentRunID, want.parent)
		}
	}
	if child.ConversationID == "" || child.ConversationID != root.ConversationID || grandchild.ConversationID != root.ConversationID {
		t.Fatalf("children inherit the root conversation: root=%q child=%q grandchild=%q", root.ConversationID, child.ConversationID, grandchild.ConversationID)
	}

	childPrompt := childLaunch.Prompt + childLaunch.SystemPrompt
	for _, want := range []string{"you are run " + child.ID.String(), "depth 1 of max 2", "a child of run " + root.ID.String(), "agent-manager task get " + child.TaskID.String(), "--parent-run-id " + child.ID.String()} {
		if !strings.Contains(childPrompt, want) {
			t.Fatalf("child prompt lacks %q:\n%s", want, childPrompt)
		}
	}
	grandchildPrompt := grandchildLaunch.Prompt + grandchildLaunch.SystemPrompt
	for _, want := range []string{"depth 2 of max 2", "Charter: " + charter, "after any context compaction", "at the maximum depth"} {
		if !strings.Contains(grandchildPrompt, want) {
			t.Fatalf("grandchild prompt lacks %q:\n%s", want, grandchildPrompt)
		}
	}
	if saved, _ := f.o.runs.Get(f.ctx, grandchild.ID); saved.CharterPath != charter {
		t.Fatalf("charter path persisted = %q, want %q", saved.CharterPath, charter)
	}

	_, err = f.createChild(grandchild, grandchildLaunch, "nested-great-grandchild", "")
	var refusal *domain.ChildDepthExceededError
	if !errors.As(err, &refusal) || refusal.Depth != 3 || refusal.MaxDepth != 2 || refusal.ParentRunID != grandchild.ID {
		t.Fatalf("a child beyond maxChildDepth must be refused with the typed depth error, got %v", err)
	}
	if domain.GetErrorCode(err) != domain.ErrCodePolicyChildDepth || !strings.Contains(err.Error(), "runExecution.maxChildDepth") {
		t.Fatalf("depth refusal must name its code and setting: %v", err)
	}
	if saved, _ := f.o.runs.GetByIdempotencyKey(f.ctx, "nested-great-grandchild"); saved != nil {
		t.Fatal("a refused child must not be persisted")
	}

	children, err := f.o.ListRuns(f.ctx, RunListOptions{ParentRunID: &child.ID})
	if err != nil || len(children) != 1 || children[0].ID != grandchild.ID || children[0].Depth != 2 {
		t.Fatalf("listing children of %s = %+v, %v; want only the grandchild at depth 2", child.ID, children, err)
	}
}

func TestChildCreationRefusesAnUnusableCharterBeforeAdmission(t *testing.T) {
	f := newNestedRunFixture(t, 3)
	root, rootLaunch := f.createRoot("")
	for name, charter := range map[string]string{
		"relative":  "CHARTER.md",
		"missing":   filepath.Join(t.TempDir(), "absent.md"),
		"directory": t.TempDir(),
	} {
		_, err := f.createChild(root, rootLaunch, "charter-"+name, charter)
		if err == nil || domain.GetErrorCode(err) != domain.ErrCodeValidationField || !strings.Contains(err.Error(), "charter") {
			t.Fatalf("%s charter must be refused as a charter validation error, got %v", name, err)
		}
		if saved, _ := f.o.runs.GetByIdempotencyKey(f.ctx, "charter-"+name); saved != nil {
			t.Fatalf("%s charter refusal persisted a run", name)
		}
	}
}

func TestRunTreeWaitsAndOrphansAcrossNestedRuns(t *testing.T) {
	f := newNestedRunFixture(t, 3)
	root, rootLaunch := f.createRoot("")
	child, err := f.createChild(root, rootLaunch, "tree-child", "")
	if err != nil {
		t.Fatal(err)
	}
	childLaunch := f.launched()
	grandchild, err := f.createChild(child, childLaunch, "tree-grandchild", "")
	if err != nil {
		t.Fatal(err)
	}
	f.launched()

	tree, err := f.o.GetRunTree(f.ctx, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tree.NodeCount != 3 || tree.Truncated || tree.MaxChildDepth != 3 {
		t.Fatalf("tree = %d nodes truncated=%t max=%d, want 3, false, 3", tree.NodeCount, tree.Truncated, tree.MaxChildDepth)
	}
	if len(tree.Root.Children) != 1 || tree.Root.Children[0].Run.ID != child.ID || len(tree.Root.Children[0].Children) != 1 {
		t.Fatalf("tree shape does not follow lineage: %+v", tree.Root)
	}
	leaf := tree.Root.Children[0].Children[0]
	if leaf.Run.ID != grandchild.ID || leaf.Run.Depth != 2 || leaf.Run.ResolvedConfig == nil || leaf.Run.ResolvedConfig.RunnerType != domain.RunnerTypeClaudeCode || leaf.Orphaned {
		t.Fatalf("grandchild node = %+v", leaf.Run)
	}

	// A wait on a live run returns timed_out without changing it.
	waited, err := f.o.WaitRun(f.ctx, child.ID, 30*time.Millisecond)
	if err != nil || waited.Outcome != RunWaitTimedOut || waited.Run.Status.IsSettled() {
		t.Fatalf("wait on a live child = %+v, %v; want timed_out", waited, err)
	}
	since := time.Now()
	stillLive, err := f.o.WaitRunChildren(f.ctx, root.ID, &since, 30*time.Millisecond)
	if err != nil || stillLive.WakeReason != ChildrenWakeTimer || len(stillLive.Active) != 1 || len(stillLive.Ended) != 0 {
		t.Fatalf("children wait with a live child = %+v, %v; want timer with one active", stillLive, err)
	}

	// Stopping the root never stops its children: they are reported orphaned.
	if err := f.o.StopRun(f.ctx, root.ID); err != nil {
		t.Fatal(err)
	}
	stopped, err := f.o.WaitRun(f.ctx, root.ID, 5*time.Second)
	if err != nil || stopped.Outcome != RunWaitTerminal {
		t.Fatalf("wait on the stopped root = %+v, %v; want terminal", stopped, err)
	}
	tree, err = f.o.GetRunTree(f.ctx, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	orphan := tree.Root.Children[0]
	if !orphan.Run.Status.IsLive() || !orphan.Orphaned || orphan.Children[0].Orphaned {
		t.Fatalf("live child of a terminal root must be orphaned (and only it): child=%s orphaned=%t grandchild orphaned=%t", orphan.Run.Status, orphan.Orphaned, orphan.Children[0].Orphaned)
	}

	f.release()
	ended, err := f.o.WaitRunChildren(f.ctx, child.ID, &since, 5*time.Second)
	if err != nil || ended.WakeReason != ChildrenWakeEnded || len(ended.Ended) != 1 || ended.Ended[0].RunID != grandchild.ID {
		t.Fatalf("children wait after the grandchild ends = %+v, %v; want child_run_ended naming it", ended, err)
	}
	if _, err := f.o.WaitRun(f.ctx, uuid.New(), time.Second); domain.GetErrorCode(err) != domain.ErrCodeNotFoundRun {
		t.Fatalf("waiting on an unknown run must report not found, got %v", err)
	}
	if _, err := f.o.WaitRun(f.ctx, child.ID, MaxRunWaitTimeout+time.Second); domain.GetErrorCode(err) != domain.ErrCodeValidationField {
		t.Fatalf("a wait beyond the server bound must be refused, got %v", err)
	}
}

// enrollDelivery attaches a running delivery commission so effort lineage and
// worker admission go through the real owner.
func (f *nestedRunFixture) enrollDelivery(effortRef string) *eventpb.WorkReference {
	f.t.Helper()
	db, err := sqlx.Connect("sqlite", filepath.Join(f.t.TempDir(), "delivery.db"))
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(f.ctx, db, modules.AllSchemas()...); err != nil {
		f.t.Fatal(err)
	}
	service := supervision.NewEffortService(supervision.NewRepository(db), nil, supervision.EffortDiscoveryConfig{})
	service.SetRunRegistry(f.o.runs)
	f.o.SetDeliveryEfforts(service)
	if _, err := service.Enroll(f.ctx, &pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{EffortRef: effortRef, TargetRevision: "selected-1", Delivery: &pb.EffortDelivery{CommissionId: "nested-commission", BudgetSeconds: 3600, MaxWorkers: 2, State: "running"}}, IdempotencyKey: "nested-start"}, supervision.EffortActor{ID: "fixture-owner", Operator: true}); err != nil {
		f.t.Fatal(err)
	}
	return &eventpb.WorkReference{Kind: "effort", Id: effortRef, Revision: "selected-1", Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
}

// Resuming a failed child replaces it in place: same parent, depth, charter,
// conversation and commissioned worker lineage, even when the parent has since
// stopped and the depth cap was lowered below the child's depth.
func TestResumingAFailedChildKeepsItsPlaceInTheTree(t *testing.T) {
	f := newNestedRunFixture(t, 3)
	ref := f.enrollDelivery("effort:nested-resume")
	charter := filepath.Join(t.TempDir(), "CHARTER.md")
	if err := os.WriteFile(charter, []byte("# Charter\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, rootLaunch := f.createRoot("", ref)
	task := f.task("nested child " + failMarker)
	child, err := f.o.CreateRun(f.ctx, CreateRunRequest{TaskID: task.ID, AgentProfileID: &f.profile.ID, PreferredRunner: "claude-code", ParentRunID: &root.ID, RunIdentityToken: rootLaunch.Environment["VROOLI_AGENT_IDENTITY_TOKEN"], RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: "nested-failing-child", CharterPath: charter})
	if err != nil {
		t.Fatal(err)
	}
	f.launched()
	failed, err := f.o.WaitRun(f.ctx, child.ID, 10*time.Second)
	if err != nil || failed.Run.Status != domain.RunStatusFailed {
		t.Fatalf("synthetic child did not fail: %+v %v", failed, err)
	}
	if err := f.o.StopRun(f.ctx, root.ID); err != nil {
		t.Fatal(err)
	}
	settings := f.o.orchestrationSettings.Get()
	settings.RunExecution.MaxChildDepth = 0
	if err := f.o.orchestrationSettings.Update(settings); err != nil {
		t.Fatal(err)
	}

	resumed, err := f.o.ResumeFromFailedRun(f.owner, ResumeFromFailedRunRequest{RunID: child.ID})
	if err != nil {
		t.Fatalf("resume of a failed child through the real path: %v", err)
	}
	f.launched()
	saved, err := f.o.runs.Get(f.ctx, resumed.ID)
	if err != nil || saved == nil {
		t.Fatal(err)
	}
	if saved.ParentRunID == nil || *saved.ParentRunID != root.ID || saved.Depth != 1 || saved.CharterPath != charter || saved.ConversationID != child.ConversationID {
		t.Fatalf("resumed child left its place: parent=%v depth=%d charter=%q conversation=%q", saved.ParentRunID, saved.Depth, saved.CharterPath, saved.ConversationID)
	}
	worker := false
	for _, r := range saved.WorkReferences {
		worker = worker || (r.GetId() == ref.Id && r.GetRelationship() == "worker")
	}
	if !worker || saved.ResolvedConfig.DeliveryCommissions[ref.Id] != "nested-commission" {
		t.Fatalf("resumed child lost its commissioned worker lineage: refs=%v commissions=%v", saved.WorkReferences, saved.ResolvedConfig.DeliveryCommissions)
	}
	tree, err := f.o.GetRunTree(f.ctx, root.ID)
	if err != nil || len(tree.Root.Children) != 2 {
		t.Fatalf("the root's tree must hold the failed child and its replacement: %+v %v", tree, err)
	}
}

// A child's authority is its parent's owner window, not the remaining life of
// the token the parent held when it created the child: a segment owner started
// late in its parent's turn keeps its identity for its own hours of work.
func TestChildAuthorityFollowsTheOwnerWindowNotTheParentToken(t *testing.T) {
	f := newNestedRunFixture(t, 3)
	root, rootLaunch := f.createRoot("")
	rootToken, err := identity.VerifyToken(rootLaunch.Environment["VROOLI_AGENT_IDENTITY_TOKEN"], f.o.identitySecret)
	if err != nil {
		t.Fatal(err)
	}
	if root.OwnerExpiresAt == nil || !time.Unix(rootToken.ExpiresAt, 0).Before(*root.OwnerExpiresAt) {
		t.Fatalf("fixture: the root token (%v) must expire before the owner window (%v)", time.Unix(rootToken.ExpiresAt, 0), root.OwnerExpiresAt)
	}
	// Twenty hours into the root's turn its token has four hours left.
	late := time.Now().Add(20 * time.Hour)
	f.o.clock = func() time.Time { return late }
	child, err := f.createChild(root, rootLaunch, "late-child", "")
	if err != nil {
		t.Fatal(err)
	}
	childLaunch := f.launched()
	if child.OwnerExpiresAt == nil || !child.OwnerExpiresAt.Equal(*root.OwnerExpiresAt) {
		t.Fatalf("child authority = %v, want the root owner window %v", child.OwnerExpiresAt, root.OwnerExpiresAt)
	}
	childToken, err := identity.VerifyTokenSignature(childLaunch.Environment["VROOLI_AGENT_IDENTITY_TOKEN"], f.o.identitySecret)
	if err != nil {
		t.Fatal(err)
	}
	if remaining := time.Unix(childToken.ExpiresAt, 0).Sub(late); remaining < 8*time.Hour {
		t.Fatalf("an eight-hour segment owner started at hour 20 keeps its identity for only %s", remaining)
	}
	grandchild, err := f.createChild(child, childLaunch, "late-grandchild", "")
	if err != nil {
		t.Fatalf("the child could not create its own child: %v", err)
	}
	f.launched()
	if grandchild.OwnerExpiresAt == nil || !grandchild.OwnerExpiresAt.Equal(*root.OwnerExpiresAt) {
		t.Fatalf("grandchild authority = %v, want the root owner window %v", grandchild.OwnerExpiresAt, root.OwnerExpiresAt)
	}
}

func TestChildAuthorityExpiryForParentsWithAndWithoutAnOwnerWindow(t *testing.T) {
	now := time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	parentToken := &identity.Claims{ExpiresAt: now.Add(4 * time.Hour).Unix()}
	window := now.Add(30 * time.Hour)
	if got := childAuthorityExpiry(&domain.Run{OwnerExpiresAt: &window}, parentToken, now); !got.Equal(window) {
		t.Fatalf("a windowed parent passes its window on, got %v", got)
	}
	// A standing-grant parent has no window; its child must not inherit the
	// four hours its parent's current token has left.
	if got := childAuthorityExpiry(&domain.Run{}, parentToken, now); !got.Equal(now.Add(identity.DefaultTTL)) {
		t.Fatalf("a parent without a window gives one identity lifetime from creation, got %v", got)
	}
}
