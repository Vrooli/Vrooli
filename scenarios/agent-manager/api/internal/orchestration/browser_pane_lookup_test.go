package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
)

func TestListRunsByWebConsoleThreadUsesExactRepositoryAssociation(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	taskID := uuid.New()
	if err := repos.Tasks.Create(context.Background(), &domain.Task{ID: taskID, Title: "pane lookup fixture", ScopePath: "src/"}); err != nil {
		t.Fatal(err)
	}
	for _, selector := range [][2]string{{"web-a", "thread-a"}, {"web-a", "thread-b"}, {"web-b", "thread-a"}} {
		id := uuid.New()
		run := &domain.Run{ID: id, TaskID: taskID, Tag: id.String(), RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeInteractive, WebConsoleSessionID: selector[0], SessionID: selector[1], Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
		if err := repos.Runs.Create(context.Background(), run); err != nil {
			t.Fatal(err)
		}
	}
	owner := &Orchestrator{runs: repos.Runs}
	got, err := owner.ListRunsByWebConsoleThread(context.Background(), "web-a", "thread-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].WebConsoleSessionID != "web-a" || got[0].SessionID != "thread-a" {
		t.Fatalf("exact selector returned %#v", got)
	}
	got, err = owner.ListRunsByWebConsoleThread(context.Background(), "web-a", "")
	if err != nil || len(got) != 0 {
		t.Fatalf("empty selector: got=%d err=%v", len(got), err)
	}
}
