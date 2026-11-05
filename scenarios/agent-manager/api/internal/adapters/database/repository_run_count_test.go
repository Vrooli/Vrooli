package database

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Count describes the same population as List, with paging ignored.
func TestRunCountMatchesListFiltersIgnoringPaging(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repos := NewRepositories(db, logrus.New())
	tasks := map[string]*domain.Task{}
	for _, scope := range []string{"scenarios/agent-manager", "scenarios/other"} {
		task := &domain.Task{ID: uuid.New(), Title: scope, ScopePath: scope, Status: domain.TaskStatusQueued}
		if err := repos.Tasks.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
		tasks[scope] = task
	}
	for i, seed := range []struct {
		scope  string
		status domain.RunStatus
		tag    string
	}{
		{"scenarios/agent-manager", domain.RunStatusRunning, "eco-1"},
		{"scenarios/agent-manager", domain.RunStatusRunning, "eco-2"},
		{"scenarios/other", domain.RunStatusRunning, "other-1"},
		{"scenarios/agent-manager", domain.RunStatusComplete, "eco-3"},
	} {
		run := &domain.Run{ID: uuid.New(), TaskID: tasks[seed.scope].ID, Tag: seed.tag + "-" + string(rune('a'+i)), Status: seed.status, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	running := domain.RunStatusRunning
	for _, tc := range []struct {
		name   string
		filter repository.RunListFilter
		want   int
	}{
		{"all", repository.RunListFilter{}, 4},
		{"status ignores paging", repository.RunListFilter{Status: &running, ListFilter: repository.ListFilter{Limit: 1, Offset: 1}}, 3},
		{"tag prefix", repository.RunListFilter{TagPrefix: "eco-"}, 3},
		{"scope prefix joins tasks", repository.RunListFilter{Status: &running, ScopePrefix: "scenarios/agent-manager"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repos.Runs.Count(ctx, tc.filter)
			if err != nil || got != tc.want {
				t.Fatalf("count=%d err=%v want %d", got, err, tc.want)
			}
			unpaged := tc.filter
			unpaged.ListFilter = repository.ListFilter{}
			listed, err := repos.Runs.List(ctx, unpaged)
			if err != nil || len(listed) != got {
				t.Fatalf("list=%d err=%v, count=%d must describe the same population", len(listed), err, got)
			}
		})
	}
	if _, err := repos.Runs.Count(ctx, repository.RunListFilter{RunnerPIDs: []int{1}}); err == nil {
		t.Fatal("invalid runner PID filter must be refused by Count like List")
	}
}
