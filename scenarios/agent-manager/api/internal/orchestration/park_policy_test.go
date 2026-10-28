package orchestration

import (
	"context"
	"errors"
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

type fakeGoalHomeParks struct {
	home          *goalhome.Home
	err           error
	notApplicable bool
}

func TestGoalHomeParkCannotIgnoreNewFeedbackWithUnchangedSignals(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := fixtures.NewTask(t)
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	parent := fixtures.NewRun(t, task.ID, uuid.Nil, fixtures.WithRunStatus(domain.RunStatusRunning))
	if err := repos.Runs.Create(ctx, parent); err != nil {
		t.Fatal(err)
	}
	child := fixtures.NewRun(t, task.ID, uuid.Nil, fixtures.WithRunStatus(domain.RunStatusRunning))
	child.ParentRunID = &parent.ID
	if err := repos.Runs.Create(ctx, child); err != nil {
		t.Fatal(err)
	}
	home := &goalhome.Home{Queue: goalhome.ParseQueue([]byte("# Status\n\n## Handoff\n\nWaiting.\n")),
		Feedback: goalhome.Feedback{Entries: []goalhome.FeedbackEntry{{ID: "NEW-REPAIR", Status: "open", Source: "supervisor"}}}}
	o := New(repos.Profiles, repos.Tasks, repos.Runs)
	o.SetGoalHomeParkSource(fakeGoalHomeParks{home: home})
	t.Setenv(parkEnforceEnv, "")
	for _, producer := range []string{ProducerChildren, ProducerGCT, ProducerOperatorDecision} {
		if got := o.checkGoalHomePark(ctx, parent, producer, nil); !strings.Contains(got, "feedback-unacknowledged") || !strings.Contains(got, "NEW-REPAIR") {
			t.Fatalf("raw %s park bypassed feedback obligation: %q", producer, got)
		}
	}
	home.Queue = goalhome.ParseQueue([]byte("# Status\n\n## Handoff\n\nNEW-REPAIR: amended scope; child continues.\n"))
	if got := o.checkGoalHomePark(ctx, parent, ProducerChildren, nil); got != "" {
		t.Fatalf("acknowledged feedback must allow child wait: %q", got)
	}
}

func (f fakeGoalHomeParks) GoalHomeForOrchestrator(context.Context, *domain.Run) (*goalhome.Home, string, bool, error) {
	return f.home, "scenarios/example/docs/internal/goal", !f.notApplicable, f.err
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
	readyAndWaiting := goalhome.ParseQueue([]byte("# Queue\n\n## Needs operator\n\n- [decision X-1] Wait for owner? Options: wait, continue.\n\n## Next\n\n- [ready] Independent slice\n\n## Handoff\n\nContinue the independent slice.\n"))
	o.SetGoalHomeParkSource(fakeGoalHomeParks{home: &goalhome.Home{Queue: readyAndWaiting}})
	if refusal := o.checkGoalHomePark(ctx, orchestrator, ProducerChildren, &twelveHours); !strings.Contains(refusal, goalhome.ParkRuleAdmissible) {
		t.Fatalf("ready independent work must take precedence over an operator hold: %q", refusal)
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

func TestLiveDirectChildrenTracksExecutionNotApprovalStatus(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := fixtures.NewTask(t)
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	parent := fixtures.NewRun(t, task.ID, uuid.Nil, fixtures.WithRunStatus(domain.RunStatusRunning))
	if err := repos.Runs.Create(ctx, parent); err != nil {
		t.Fatal(err)
	}
	parentID := parent.ID
	cases := []struct {
		status   domain.RunStatus
		wantLive bool
	}{
		{status: domain.RunStatusNeedsReview, wantLive: false},
		{status: domain.RunStatusPending, wantLive: true},
		{status: domain.RunStatusStarting, wantLive: true},
		{status: domain.RunStatusRunning, wantLive: true},
		{status: domain.RunStatusParked, wantLive: true},
	}
	want := make(map[string]bool, len(cases))
	for _, tc := range cases {
		child := fixtures.NewRun(t, task.ID, uuid.Nil, fixtures.WithRunStatus(tc.status))
		child.ParentRunID = &parentID
		if err := repos.Runs.Create(ctx, child); err != nil {
			t.Fatal(err)
		}
		want[child.ID.String()] = tc.wantLive
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs)
	live, err := o.liveDirectChildren(ctx, parent)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(live))
	for _, id := range live {
		got[id] = true
	}
	if len(got) != 4 {
		t.Fatalf("execution cohort should contain exactly four children, got %v", live)
	}
	for id, expected := range want {
		if got[id] != expected {
			t.Fatalf("child %s live=%t, want %t", id, got[id], expected)
		}
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

func TestGoalHomeParkRefusesWhenMandatorySteeringCannotBeRead(t *testing.T) {
	for _, source := range []fakeGoalHomeParks{
		{err: errors.New("incomplete LOG evidence")},
		{err: errors.New("effort repository unavailable"), notApplicable: true},
		{},
	} {
		o := &Orchestrator{goalHomeParks: source}
		for _, producer := range []string{ProducerChildren, ProducerGCT, ProducerOperatorDecision} {
			if got := o.checkGoalHomePark(t.Context(), &domain.Run{ID: uuid.New()}, producer, nil); !strings.Contains(got, "park refused") {
				t.Fatalf("%s bypassed unreadable required steering: %q", producer, got)
			}
		}
	}
}
