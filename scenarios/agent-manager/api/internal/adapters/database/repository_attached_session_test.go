package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// A re-attached harness session keeps several live runs; quota readings must
// name the earliest one, the run whose physical presence the inventory
// probes. Terminal runs and other harnesses never match.
func TestAttachedSessionRunResolverNamesTheEarliestLiveRunOfTheSession(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	base := time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC)
	create := func(status domain.RunStatus, kind, session string, pid int, offset time.Duration) uuid.UUID {
		t.Helper()
		started := base.Add(offset)
		run := &domain.Run{ID: uuid.New(), Tag: "attached-" + uuid.NewString()[:8], RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeAttached,
			HarnessKind: kind, HarnessSessionID: session, Status: status, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone,
			RunnerPID: pid, StartedAt: &started, RequestedScopes: []string{}}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run.ID
	}
	create(domain.RunStatusComplete, "claude-code", "claude-code-a", 4100, -time.Hour) // finished earlier attach
	earliest := create(domain.RunStatusRunning, "claude-code", "claude-code-a", 4200, 0)
	create(domain.RunStatusRunning, "claude-code", "claude-code-a", 4200, 30*time.Minute) // re-attach
	create(domain.RunStatusRunning, "codex", "claude-code-a", 4300, -2*time.Hour)         // other harness
	resolve := AttachedSessionRunResolver(db)
	if got := resolve(ctx, "claude-code", "claude-code-a"); got != earliest.String() {
		t.Fatalf("resolved %q; want earliest live run %s", got, earliest)
	}
	for _, tc := range [][2]string{{"claude-code", "claude-code-missing"}, {"", "claude-code-a"}, {"claude-code", ""}} {
		if got := resolve(ctx, tc[0], tc[1]); got != "" {
			t.Fatalf("resolve(%q,%q)=%q; want unresolved", tc[0], tc[1], got)
		}
	}
}
