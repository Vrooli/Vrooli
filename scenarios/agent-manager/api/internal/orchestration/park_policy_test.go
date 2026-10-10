package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/goalhome"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/fixtures"

	"github.com/google/uuid"
)

type fakeGoalHomeParks struct{ home *goalhome.Home }

func (f fakeGoalHomeParks) GoalHomeForOrchestrator(context.Context, *domain.Run) (*goalhome.Home, string, bool, error) {
	return f.home, "scenarios/example/docs/internal/goal", true, nil
}

func TestGoalHomeParkPolicyReportsByDefaultAndRefusesWhenEnforced(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := fixtures.NewTask(t)
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	orchestrator := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, ResolvedConfig: domain.DefaultRunConfig(), RunMode: domain.RunModeInPlace, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Runs.Create(ctx, orchestrator); err != nil {
		t.Fatal(err)
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events))
	idle := goalhome.ParseQueue([]byte("# Queue\n\n## Needs operator\n\n(none)\n\n## Handoff\n\nIdle.\n"))
	o.SetGoalHomeParkSource(fakeGoalHomeParks{home: &goalhome.Home{Queue: idle}})
	twelveHours := time.Now().Add(12 * time.Hour)

	t.Setenv(parkEnforceEnv, "")
	if refusal := o.checkGoalHomePark(ctx, orchestrator, ProducerChildren, &twelveHours); refusal != "" {
		t.Fatalf("report-only mode refused a park: %s", refusal)
	}
	if messages := parkPolicyMessages(t, o, orchestrator.ID, "park policy (report-only"); len(messages) != 1 || !strings.Contains(messages[0], goalhome.ParkRuleAdmissible) {
		t.Fatalf("report-only mode must record the would-be refusal once: %q", messages)
	}

	t.Setenv(parkEnforceEnv, "1")
	if refusal := o.checkGoalHomePark(ctx, orchestrator, ProducerChildren, &twelveHours); !strings.Contains(refusal, "admissible-slice") {
		t.Fatalf("an idle goal home must refuse a park with no live child or decision: %q", refusal)
	}
	if refusal := o.checkGoalHomePark(ctx, orchestrator, ProducerGCT, &twelveHours); refusal != "" {
		t.Fatalf("the policy applies only to the orchestrator's park on its children: %q", refusal)
	}
	waiting := goalhome.ParseQueue([]byte("# Queue\n\n## Needs operator\n\n- [decision X-1] Close or re-aim?\n\n## Handoff\n\nWaiting.\n"))
	o.SetGoalHomeParkSource(fakeGoalHomeParks{home: &goalhome.Home{Queue: waiting}})
	if refusal := o.checkGoalHomePark(ctx, orchestrator, ProducerChildren, &twelveHours); refusal != "" {
		t.Fatalf("a Needs operator item allows a long park: %q", refusal)
	}
	week := time.Now().Add(7 * 24 * time.Hour)
	if refusal := o.checkGoalHomePark(ctx, orchestrator, ProducerChildren, &week); !strings.Contains(refusal, "exceeds the 72h0m0s maximum") {
		t.Fatalf("a park past the 72h decision wait must be refused: %q", refusal)
	}
}

// parkPolicyMessages returns the run's log messages that contain text.
func parkPolicyMessages(t *testing.T, o *Orchestrator, runID uuid.UUID, text string) []string {
	t.Helper()
	all, err := o.events.Get(context.Background(), runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, evt := range all {
		if data, ok := evt.Data.(*domain.LogEventData); ok && strings.Contains(data.Message, text) {
			out = append(out, data.Message)
		}
	}
	return out
}
