package goals

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	_ "modernc.org/sqlite"
	g "personal-planner/internal/goals"
)

func TestConnectMeasuredGoalCreateListRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(g.Schema()); err != nil {
		t.Fatal(err)
	}

	h := NewConnectHandler(Deps{Service: g.NewService(g.NewSQLiteRepository(db, schedule.NewFake(time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC))))})
	ctx := context.Background()
	cases := []struct {
		name, targetDate, recurrence, period string
		amount                               int64
	}{
		{name: "one-time", targetDate: "2026-11-30", recurrence: "one_time", period: "once", amount: 100000},
		{name: "monthly", targetDate: "2026-12-31", recurrence: "recurring", period: "month", amount: 100000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.CreateGoal(ctx, connect.NewRequest(&v.CreateGoalRequest{
				Title: tc.name, ProgressMethod: g.ProgressManual, TargetBasisPoints: 10000, TargetDate: tc.targetDate,
				Measure: &v.GoalMeasure{Unit: "currency", TargetAmountMinor: tc.amount, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: tc.recurrence, Period: tc.period},
			}))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	_, err = h.CreateGoal(ctx, connect.NewRequest(&v.CreateGoalRequest{Title: "generic", ProgressMethod: g.ProgressManual, TargetBasisPoints: 10000}))
	if err != nil {
		t.Fatal(err)
	}

	listed, err := h.ListGoals(ctx, connect.NewRequest(&v.ListGoalsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.Goals) != 3 {
		t.Fatalf("listed %d goals, want 3", len(listed.Msg.Goals))
	}
	byTitle := make(map[string]*v.Goal, len(listed.Msg.Goals))
	for _, goal := range listed.Msg.Goals {
		byTitle[goal.Title] = goal
	}
	for _, tc := range cases {
		goal := byTitle[tc.name]
		if goal == nil {
			t.Fatalf("missing %s goal", tc.name)
		}
		m := goal.Measure
		if m == nil || m.Unit != "currency" || m.TargetAmountMinor != tc.amount || m.CurrencyCode != "USD" || m.Basis != "after_costs_before_tax" || m.Recurrence != tc.recurrence || m.Period != tc.period || goal.TargetDate != tc.targetDate {
			t.Errorf("%s projection lost measure/date: goal=%+v measure=%+v", tc.name, goal, m)
		}
		if goal.ProgressState != "unknown" || goal.ProgressBasisPoints != nil {
			t.Errorf("%s progress should be unknown without a numeric value: state=%q value=%v", tc.name, goal.ProgressState, goal.ProgressBasisPoints)
		}
	}
	generic := byTitle["generic"]
	if generic == nil || generic.Measure != nil || generic.TargetDate != "" || generic.ProgressState != "" || generic.ProgressBasisPoints == nil || *generic.ProgressBasisPoints != 0 {
		t.Errorf("generic goal compatibility changed: %+v", generic)
	}
}
