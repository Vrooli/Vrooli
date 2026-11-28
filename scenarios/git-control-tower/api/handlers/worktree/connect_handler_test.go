package worktree_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	hworktree "git-control-tower/handlers/worktree"
	dworktree "git-control-tower/internal/worktree"
	"git-control-tower/internal/worktree/mocks"

	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

// All tests in this file run against a httptest.Server wired to the
// Connect handler with FAKE inspector/mutator. NO real git is invoked.

func newTestServer(t *testing.T, insp dworktree.Inspector, mut dworktree.Mutator, audit ...func(hworktree.MutationAuditEvent)) (worktreeconnect.WorktreeServiceClient, func()) {
	t.Helper()
	svc := dworktree.NewService(insp, mut)
	deps := hworktree.Deps{Service: svc}
	if len(audit) > 0 {
		deps.Audit = audit[0]
	}
	path, handler := hworktree.NewHandler(deps)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	client := worktreeconnect.NewWorktreeServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

func TestHandlerAuditsSuccessfulMutations(t *testing.T) {
	var events []hworktree.MutationAuditEvent
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{{Path: "/repo", IsMain: true}, {Path: "/tmp/feature", Name: "feature", Branch: "feature"}}}
	mut := &mocks.FakeMutator{PruneResult: dworktree.PruneResult{PrunedPaths: []string{"old-a", "old-b"}}}
	client, cleanup := newTestServer(t, insp, mut, func(event hworktree.MutationAuditEvent) { events = append(events, event) })
	defer cleanup()

	if _, err := client.CreateWorktree(context.Background(), connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/repo", NewWorktreePath: "/tmp/new", Source: &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"}})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RemoveWorktree(context.Background(), connect.NewRequest(&worktreev1.RemoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature"})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.LockWorktree(context.Background(), connect.NewRequest(&worktreev1.LockWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature", Reason: "pinned"})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UnlockWorktree(context.Background(), connect.NewRequest(&worktreev1.UnlockWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature"})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.MoveWorktree(context.Background(), connect.NewRequest(&worktreev1.MoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature", NewWorktreePath: "/tmp/moved"})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.PruneWorktrees(context.Background(), connect.NewRequest(&worktreev1.PruneWorktreesRequest{RepoPath: "/repo", Reason: "expired"})); err != nil {
		t.Fatal(err)
	}

	wantOps := []string{hworktree.AuditWorktreeCreated, hworktree.AuditWorktreeRemoved, hworktree.AuditWorktreeLocked, hworktree.AuditWorktreeUnlocked, hworktree.AuditWorktreeMoved, hworktree.AuditWorktreePruned}
	if len(events) != len(wantOps) {
		t.Fatalf("audit events = %d, want %d: %+v", len(events), len(wantOps), events)
	}
	for i, op := range wantOps {
		if events[i].Operation != op || events[i].RepoPath != "/repo" || events[i].DryRun {
			t.Fatalf("event %d = %+v, want operation %q on /repo without dry-run", i, events[i], op)
		}
	}
	if events[0].Paths[0] != "/tmp/new" || events[0].Branch != "feature" || events[0].Mode != "existing_branch" {
		t.Fatalf("create event lost source details: %+v", events[0])
	}
	if events[2].Reason != "pinned" || events[4].Paths[0] != "/tmp/feature" || events[4].Paths[1] != "/tmp/moved" || len(events[5].Paths) != 2 {
		t.Fatalf("audit details lost: lock=%+v move=%+v prune=%+v", events[2], events[4], events[5])
	}
}

func TestHandlerAuditsDryRunsAndReportOnlyPrune(t *testing.T) {
	var events []hworktree.MutationAuditEvent
	mut := &mocks.FakeMutator{PruneResult: dworktree.PruneResult{PrunedPaths: []string{"would-remove"}}}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut, func(event hworktree.MutationAuditEvent) { events = append(events, event) })
	defer cleanup()
	ctx := context.Background()

	create := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/repo", NewWorktreePath: "/tmp/new", Source: &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"}})
	create.Header().Set(hworktree.DryRunHeader, "true")
	if _, err := client.CreateWorktree(ctx, create); err != nil {
		t.Fatal(err)
	}
	remove := connect.NewRequest(&worktreev1.RemoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature"})
	remove.Header().Set(hworktree.DryRunHeader, "true")
	if _, err := client.RemoveWorktree(ctx, remove); err != nil {
		t.Fatal(err)
	}
	lock := connect.NewRequest(&worktreev1.LockWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature"})
	lock.Header().Set(hworktree.DryRunHeader, "true")
	if _, err := client.LockWorktree(ctx, lock); err != nil {
		t.Fatal(err)
	}
	unlock := connect.NewRequest(&worktreev1.UnlockWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature"})
	unlock.Header().Set(hworktree.DryRunHeader, "true")
	if _, err := client.UnlockWorktree(ctx, unlock); err != nil {
		t.Fatal(err)
	}
	move := connect.NewRequest(&worktreev1.MoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/feature", NewWorktreePath: "/tmp/moved"})
	move.Header().Set(hworktree.DryRunHeader, "true")
	if _, err := client.MoveWorktree(ctx, move); err != nil {
		t.Fatal(err)
	}
	prune := connect.NewRequest(&worktreev1.PruneWorktreesRequest{RepoPath: "/repo"})
	prune.Header().Set(hworktree.DryRunHeader, "true")
	response, err := client.PruneWorktrees(ctx, prune)
	if err != nil || !response.Msg.DryRun {
		t.Fatalf("header prune dry-run = %v %v", response, err)
	}
	response, err = client.PruneWorktrees(ctx, connect.NewRequest(&worktreev1.PruneWorktreesRequest{RepoPath: "/repo", ReportOnly: true}))
	if err != nil || !response.Msg.DryRun || len(response.Msg.PrunedPaths) != 1 {
		t.Fatalf("report-only prune = %v %v", response, err)
	}

	if len(events) != 7 {
		t.Fatalf("dry-run audit events = %d, want 7: %+v", len(events), events)
	}
	for i, event := range events {
		if !event.DryRun || event.RepoPath != "/repo" {
			t.Fatalf("event %d = %+v, want dry-run for /repo", i, event)
		}
	}
	if events[6].Paths[0] != "would-remove" || len(mut.PruneCalls) != 1 || !mut.PruneCalls[0].ReportOnly {
		t.Fatalf("report-only should retain would-prune evidence without mutating: event=%+v calls=%+v", events[6], mut.PruneCalls)
	}
}

func TestHandlerDoesNotAuditRejectedMutation(t *testing.T) {
	var events []hworktree.MutationAuditEvent
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, &mocks.FakeMutator{}, func(event hworktree.MutationAuditEvent) { events = append(events, event) })
	defer cleanup()
	_, err := client.CreateWorktree(context.Background(), connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/repo", NewWorktreePath: "/tmp/new"}))
	if err == nil || len(events) != 0 {
		t.Fatalf("rejected mutation must not be audited as successful: err=%v events=%+v", err, events)
	}
}

func TestHandlerDoesNotAuditInvalidDryRuns(t *testing.T) {
	var events []hworktree.MutationAuditEvent
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, &mocks.FakeMutator{}, func(event hworktree.MutationAuditEvent) { events = append(events, event) })
	defer cleanup()
	ctx := context.Background()

	create := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/repo"})
	create.Header().Set(hworktree.DryRunHeader, "true")
	remove := connect.NewRequest(&worktreev1.RemoveWorktreeRequest{RepoPath: "/repo"})
	remove.Header().Set(hworktree.DryRunHeader, "true")
	lock := connect.NewRequest(&worktreev1.LockWorktreeRequest{RepoPath: "/repo"})
	lock.Header().Set(hworktree.DryRunHeader, "true")
	unlock := connect.NewRequest(&worktreev1.UnlockWorktreeRequest{RepoPath: "/repo"})
	unlock.Header().Set(hworktree.DryRunHeader, "true")
	move := connect.NewRequest(&worktreev1.MoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/tmp/old"})
	move.Header().Set(hworktree.DryRunHeader, "true")
	prune := connect.NewRequest(&worktreev1.PruneWorktreesRequest{})
	prune.Header().Set(hworktree.DryRunHeader, "true")

	checks := []struct {
		name string
		run  func() error
	}{
		{"create", func() error { _, err := client.CreateWorktree(ctx, create); return err }},
		{"remove", func() error { _, err := client.RemoveWorktree(ctx, remove); return err }},
		{"lock", func() error { _, err := client.LockWorktree(ctx, lock); return err }},
		{"unlock", func() error { _, err := client.UnlockWorktree(ctx, unlock); return err }},
		{"move", func() error { _, err := client.MoveWorktree(ctx, move); return err }},
		{"prune", func() error { _, err := client.PruneWorktrees(ctx, prune); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.run(); err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("invalid dry-run error = %v, want InvalidArgument", err)
			}
		})
	}
	if len(events) != 0 {
		t.Fatalf("invalid dry-runs must not be audited as successful: %+v", events)
	}
}

func TestHandlerList(t *testing.T) {
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{
		{Path: "/repo", Name: "repo", IsMain: true, Branch: "main"},
		{Path: "/tmp/feature", Name: "feature", Branch: "feature"},
	}}
	client, cleanup := newTestServer(t, insp, &mocks.FakeMutator{})
	defer cleanup()

	resp, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: "/repo"}))
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(resp.Msg.Worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(resp.Msg.Worktrees))
	}
	if resp.Msg.Worktrees[0].IsMain != true || resp.Msg.Worktrees[1].Branch != "feature" {
		t.Fatalf("unexpected payload: %+v", resp.Msg.Worktrees)
	}
}

func TestHandlerList_RequiresRepoPath(t *testing.T) {
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, &mocks.FakeMutator{})
	defer cleanup()

	_, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{}))
	if err == nil {
		t.Fatalf("expected error for empty repo_path")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v (%v)", connect.CodeOf(err), err)
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{{Path: "/repo", IsMain: true}}}
	client, cleanup := newTestServer(t, insp, &mocks.FakeMutator{})
	defer cleanup()

	_, err := client.GetWorktree(context.Background(), connect.NewRequest(&worktreev1.GetWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/missing",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestHandlerCreate_ExistingBranch(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.CreateWorktree(context.Background(), connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/feature",
		Source:          &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"},
	}))
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if resp.Msg.DryRun {
		t.Fatalf("expected DryRun=false")
	}
	if resp.Msg.Worktree.Branch != "feature" || resp.Msg.Worktree.Path != "/tmp/feature" {
		t.Fatalf("unexpected: %+v", resp.Msg.Worktree)
	}
	if len(mut.AddCalls) != 1 {
		t.Fatalf("expected one Add call, got %d", len(mut.AddCalls))
	}
}

func TestHandlerCreate_NewBranchWithStartPoint(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	_, err := client.CreateWorktree(context.Background(), connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/x",
		Source: &worktreev1.CreateWorktreeRequest_NewBranch{NewBranch: &worktreev1.NewBranchSpec{
			Name:       "topic",
			StartPoint: "main",
		}},
		Track: true,
	}))
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if len(mut.AddCalls) != 1 {
		t.Fatalf("expected one Add call")
	}
	got := mut.AddCalls[0]
	if got.NewBranchName != "topic" || got.NewBranchStart != "main" || !got.Track {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestHandlerCreate_DryRun_DoesNotCallMutator(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/feature",
		Source:          &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"},
	})
	req.Header().Set(hworktree.DryRunHeader, "true")

	resp, err := client.CreateWorktree(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if !resp.Msg.DryRun {
		t.Fatalf("expected DryRun=true, got %+v", resp.Msg)
	}
	if len(mut.AddCalls) != 0 {
		t.Fatalf("Mutator must NOT be called during dry-run, got %d calls", len(mut.AddCalls))
	}
}

func TestHandlerCreate_DryRun_StillValidates(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	// Multiple sources is invalid; dry-run must surface InvalidArgument.
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/x",
		Source:          &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"},
	})
	// Manually set Commit to violate Mode() (can't through oneof normally — but we can pass a different oneof).
	// Use empty source instead — the validation rejects it.
	req.Msg.Source = nil
	req.Header().Set(hworktree.DryRunHeader, "true")
	_, err := client.CreateWorktree(context.Background(), req)
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestHandlerRemove_DryRun(t *testing.T) {
	mut := &mocks.FakeMutator{}
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{
		{Path: "/repo", IsMain: true}, {Path: "/tmp/feature", Name: "feature"},
	}}
	client, cleanup := newTestServer(t, insp, mut)
	defer cleanup()

	req := connect.NewRequest(&worktreev1.RemoveWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/feature",
	})
	req.Header().Set(hworktree.DryRunHeader, "true")
	resp, err := client.RemoveWorktree(context.Background(), req)
	if err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}
	if !resp.Msg.DryRun {
		t.Fatalf("expected DryRun=true")
	}
	if len(mut.RemoveCalls) != 0 {
		t.Fatalf("Mutator must NOT be called during dry-run")
	}
}

func TestHandlerRemove_RefusesMain(t *testing.T) {
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{{Path: "/repo", Name: "repo", IsMain: true}}}
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, insp, mut)
	defer cleanup()

	_, err := client.RemoveWorktree(context.Background(), connect.NewRequest(&worktreev1.RemoveWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/repo", Force: true,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestHandlerLock(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.LockWorktree(context.Background(), connect.NewRequest(&worktreev1.LockWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/feature", Reason: "agents pinned",
	}))
	if err != nil {
		t.Fatalf("LockWorktree: %v", err)
	}
	if !resp.Msg.Worktree.Locked || resp.Msg.Worktree.LockReason != "agents pinned" {
		t.Fatalf("unexpected: %+v", resp.Msg.Worktree)
	}
}

func TestHandlerUnlock(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	_, err := client.UnlockWorktree(context.Background(), connect.NewRequest(&worktreev1.UnlockWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/feature",
	}))
	if err != nil {
		t.Fatalf("UnlockWorktree: %v", err)
	}
}

func TestHandlerMove(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.MoveWorktree(context.Background(), connect.NewRequest(&worktreev1.MoveWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/old", NewWorktreePath: "/tmp/new",
	}))
	if err != nil {
		t.Fatalf("MoveWorktree: %v", err)
	}
	if resp.Msg.Worktree.Path != "/tmp/new" {
		t.Fatalf("unexpected: %+v", resp.Msg.Worktree)
	}
}

func TestHandlerPrune(t *testing.T) {
	mut := &mocks.FakeMutator{PruneResult: dworktree.PruneResult{PrunedPaths: []string{"old1", "old2"}}}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.PruneWorktrees(context.Background(), connect.NewRequest(&worktreev1.PruneWorktreesRequest{
		RepoPath: "/repo",
	}))
	if err != nil {
		t.Fatalf("PruneWorktrees: %v", err)
	}
	if len(resp.Msg.PrunedPaths) != 2 {
		t.Fatalf("expected 2 pruned, got %v", resp.Msg.PrunedPaths)
	}
}

func TestHandlerPrune_DryRun(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	req := connect.NewRequest(&worktreev1.PruneWorktreesRequest{RepoPath: "/repo"})
	req.Header().Set(hworktree.DryRunHeader, "true")
	resp, err := client.PruneWorktrees(context.Background(), req)
	if err != nil {
		t.Fatalf("PruneWorktrees: %v", err)
	}
	if !resp.Msg.DryRun {
		t.Fatalf("expected DryRun")
	}
	if len(mut.PruneCalls) != 0 {
		t.Fatalf("Mutator must not be called during dry-run")
	}
}
