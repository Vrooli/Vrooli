package jobs_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/jobs"
)

func TestRunScopeAtomicallyBoundsSubmissionsAndClosesDispatch(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "scope.db"))
	scopeStore, ok := repo.(jobs.RunScopeRepository)
	if !ok {
		t.Fatal("SQLite job owner does not implement durable run scopes")
	}
	scopes := jobs.NewRunScopeService(scopeStore, func() time.Time { return baseTime })
	policy, _ := json.Marshal(map[string]any{"funding_route": "local", "tokens_per_call": 4096})
	if err := scopes.Create(context.Background(), jobs.RunScope{ID: "creation-1", OwnerKind: "device_geometry_creation", Policy: policy, MaxChildren: 2, MaxProviderCalls: 2}); err != nil {
		t.Fatal(err)
	}
	svc := newService(repo)
	in := input()
	in.WorkflowScopeID = "creation-1"
	in.WorkflowProviderCalls = 1
	job, created, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "scope-child-1", Input: in})
	if err != nil || !created {
		t.Fatalf("first child created=%v err=%v", created, err)
	}
	replayed, created, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "scope-child-1", Input: in})
	if err != nil || created || replayed.ID != job.ID {
		t.Fatalf("idempotent replay created=%v job=%#v err=%v", created, replayed, err)
	}
	scope, err := scopes.Get(context.Background(), "creation-1")
	if err != nil || scope.ChildrenAdmitted != 1 || scope.ProviderCallsReserved != 1 {
		t.Fatalf("scope counters after replay %#v err=%v", scope, err)
	}
	closed, children, err := scopes.Close(context.Background(), scope.ID, scope.Version)
	if err != nil || closed.State != jobs.RunScopeClosing || len(children) != 1 || children[0].JobID != job.ID {
		t.Fatalf("close scope %#v children=%#v err=%v", closed, children, err)
	}
	if _, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "scope-child-2", Input: in}); err != jobs.ErrScopeAdmissionClosed {
		t.Fatalf("submit after close err=%v", err)
	}
	current, err := svc.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: current.Attempts[0].ID, AttemptNumber: 1, OwnerID: "test", ScopeID: "execution", ArtifactID: "worker", SHA256: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}, current.Version)
	if err != jobs.ErrScopeAdmissionClosed {
		t.Fatalf("start after close err=%v", err)
	}
	children, err = scopes.Children(context.Background(), scope.ID)
	if err != nil || len(children) != 1 {
		t.Fatalf("children=%#v err=%v", children, err)
	}
}

func TestRunScopeConcurrentAdmissionsCannotOverspend(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "scope-race.db"))
	scopeStore := repo.(jobs.RunScopeRepository)
	scopes := jobs.NewRunScopeService(scopeStore, func() time.Time { return baseTime })
	policy, _ := json.Marshal(map[string]any{"kind": "fixed"})
	if err := scopes.Create(context.Background(), jobs.RunScope{ID: "creation-race", OwnerKind: "device_geometry_creation", Policy: policy, MaxChildren: 1, MaxProviderCalls: 1}); err != nil {
		t.Fatal(err)
	}
	svc := newService(repo)
	inputs := []jobs.Input{input(), input()}
	for i := range inputs {
		inputs[i].WorkflowScopeID = "creation-race"
		inputs[i].WorkflowProviderCalls = 1
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := range inputs {
		go func(i int) {
			<-start
			_, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: string(rune('a' + i)), Input: inputs[i]})
			results <- err
		}(i)
	}
	close(start)
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatalf("exactly one admission should fit, got %v / %v", first, second)
	}
	winner, loser := first, second
	if winner != nil {
		winner, loser = second, first
	}
	if loser != jobs.ErrScopeAdmissionClosed {
		t.Fatalf("loser=%v, want closed/budget exceeded", loser)
	}
	scope, err := scopes.Get(context.Background(), "creation-race")
	if err != nil || scope.ChildrenAdmitted != 1 || scope.ProviderCallsReserved != 1 {
		t.Fatalf("race overspent scope %#v err=%v", scope, err)
	}
}

func TestRunScopeConcurrentAdmissionStopsAtTwoChildBudget(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "scope-two-child-budget.db"))
	scopeStore := repo.(jobs.RunScopeRepository)
	scopes := jobs.NewRunScopeService(scopeStore, func() time.Time { return baseTime })
	policy, _ := json.Marshal(map[string]any{"funding_route": "local", "tokens_per_call": 4096})
	if err := scopes.Create(context.Background(), jobs.RunScope{ID: "creation-two-child", OwnerKind: "device_geometry_creation", Policy: policy, MaxChildren: 2, MaxProviderCalls: 2}); err != nil {
		t.Fatal(err)
	}
	svc := newService(repo)
	start := make(chan struct{})
	results := make(chan error, 3)
	for i := 0; i < 3; i++ {
		in := input()
		in.WorkflowScopeID, in.WorkflowProviderCalls = "creation-two-child", 1
		go func(i int, in jobs.Input) {
			<-start
			_, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: string(rune('a' + i)), Input: in})
			results <- err
		}(i, in)
	}
	close(start)
	accepted, rejected := 0, 0
	for i := 0; i < 3; i++ {
		err := <-results
		if err == nil {
			accepted++
		} else if err == jobs.ErrScopeAdmissionClosed {
			rejected++
		} else {
			t.Fatalf("unexpected admission error: %v", err)
		}
	}
	scope, err := scopes.Get(context.Background(), "creation-two-child")
	if err != nil || accepted != 2 || rejected != 1 || scope.ChildrenAdmitted != 2 || scope.ProviderCallsReserved != 2 {
		t.Fatalf("scope exceeded the two-call parent budget: accepted=%d rejected=%d scope=%#v err=%v", accepted, rejected, scope, err)
	}
}

func TestRunScopeCancelAndClaimHaveOneAtomicBoundary(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "scope-cancel-claim.db"))
	scopeStore := repo.(jobs.RunScopeRepository)
	scopes := jobs.NewRunScopeService(scopeStore, func() time.Time { return baseTime })
	policy, _ := json.Marshal(map[string]any{"funding_route": "local"})
	if err := scopes.Create(context.Background(), jobs.RunScope{ID: "creation-cancel-race", OwnerKind: "device_geometry_creation", Policy: policy, MaxChildren: 2, MaxProviderCalls: 2}); err != nil {
		t.Fatal(err)
	}
	svc := newService(repo)
	in := input()
	in.WorkflowScopeID, in.WorkflowProviderCalls = "creation-cancel-race", 1
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "cancel-race-child", Input: in})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	startResult := make(chan error, 1)
	closeResult := make(chan error, 1)
	go func() {
		<-start
		_, err := svc.Start(context.Background(), job.ID, executionOwner("protected", job.Attempts[0].ID, 1), job.Version)
		startResult <- err
	}()
	go func() {
		<-start
		_, _, err := scopes.Close(context.Background(), "creation-cancel-race", 2)
		closeResult <- err
	}()
	close(start)
	startErr, closeErr := <-startResult, <-closeResult
	if closeErr != nil {
		t.Fatalf("scope close lost its serialized transition: %v", closeErr)
	}
	if startErr != nil && startErr != jobs.ErrScopeAdmissionClosed {
		t.Fatalf("unexpected claim result: %v", startErr)
	}
	if startErr == jobs.ErrScopeAdmissionClosed {
		if _, err := svc.Start(context.Background(), job.ID, executionOwner("protected", job.Attempts[0].ID, 1), job.Version); err != jobs.ErrScopeAdmissionClosed {
			t.Fatalf("claim after close should stay blocked, got %v", err)
		}
	}
	second := input()
	second.WorkflowScopeID, second.WorkflowProviderCalls = "creation-cancel-race", 1
	if _, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "after-cancel", Input: second}); err != jobs.ErrScopeAdmissionClosed {
		t.Fatalf("child admission after cancel should stay blocked, got %v", err)
	}
}

func TestRunScopeAndChildrenResumeAfterSQLiteReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "scope-resume.db")
	db, repo := openRepo(t, dbPath)
	scopes := jobs.NewRunScopeService(repo.(jobs.RunScopeRepository), func() time.Time { return baseTime })
	policy, _ := json.Marshal(map[string]any{"funding_route": "local", "tokens_per_call": 4096})
	if err := scopes.Create(context.Background(), jobs.RunScope{ID: "creation-resume", OwnerKind: "device_geometry_creation", Policy: policy, MaxChildren: 2, MaxProviderCalls: 2}); err != nil {
		t.Fatal(err)
	}
	in := input()
	in.WorkflowScopeID, in.WorkflowProviderCalls = "creation-resume", 1
	child, _, err := newService(repo).Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "resume-child", Input: in})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	resumedDB, resumedRepo := openRepo(t, dbPath)
	defer resumedDB.Close()
	resumedScopes := jobs.NewRunScopeService(resumedRepo.(jobs.RunScopeRepository), func() time.Time { return baseTime.Add(time.Minute) })
	resumed, err := resumedScopes.Get(context.Background(), "creation-resume")
	children, childrenErr := resumedScopes.Children(context.Background(), "creation-resume")
	resumedChild, childErr := newService(resumedRepo).Get(context.Background(), child.ID)
	if err != nil || childrenErr != nil || childErr != nil || resumed.State != jobs.RunScopeActive || resumed.ChildrenAdmitted != 1 || resumed.ProviderCallsReserved != 1 || len(children) != 1 || children[0].JobID != child.ID || resumedChild.Input.WorkflowScopeID != resumed.ID {
		t.Fatalf("durable scope could not resume with its child: scope=%#v children=%#v child=%#v errors=%v/%v/%v", resumed, children, resumedChild, err, childrenErr, childErr)
	}
}
