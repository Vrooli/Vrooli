package runtimesupervisor

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/testenv"
	"github.com/vrooli/vrooli/internal/tuning"
)

// The periodic reap is off unless the operator turns it on: with no
// configured interval and no override, the supervisor never reaps.
func TestSupervisorAgentReapIsOffByDefault(t *testing.T) {
	t.Setenv(tuning.EnvironmentVariable("AgentOrphanReapInterval"), "")
	if tuning.AgentOrphanReapInterval() != 0 {
		t.Fatalf("compiled default reap interval = %s, want 0 (off)", tuning.AgentOrphanReapInterval())
	}
	calls := 0
	svc := New(Config{
		Clock: testenv.NewClock(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)),
		AgentOrphanReap: func(context.Context, agentsession.ReapRequest) ([]agentsession.Result, error) {
			calls++
			return nil, nil
		},
	})
	if report := svc.reapOrphanedAgentSessions(context.Background()); report.Ran || calls != 0 {
		t.Fatalf("report %+v, %d reaps; want none", report, calls)
	}
}

// Enabled, it runs at most once per interval and counts what it stopped.
func TestSupervisorAgentReapRunsOncePerInterval(t *testing.T) {
	clk := testenv.NewClock(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	var requests []agentsession.ReapRequest
	svc := New(Config{
		Clock: clk, HomeDir: "/home/op", AgentOrphanReapInterval: time.Hour,
		AgentOrphanReap: func(_ context.Context, request agentsession.ReapRequest) ([]agentsession.Result, error) {
			requests = append(requests, request)
			return []agentsession.Result{{Outcome: agentsession.OutcomeStopped}, {Outcome: agentsession.OutcomeKept}, {Outcome: agentsession.OutcomeFailed}}, nil
		},
	})
	first := svc.reapOrphanedAgentSessions(context.Background())
	if !first.Ran || first.Stopped != 1 || first.Failed != 1 {
		t.Fatalf("first reap = %+v, want ran with 1 stopped and 1 failed", first)
	}
	clk.Advance(30 * time.Minute)
	if again := svc.reapOrphanedAgentSessions(context.Background()); again.Ran {
		t.Fatal("reaped again inside the interval")
	}
	clk.Advance(31 * time.Minute)
	if later := svc.reapOrphanedAgentSessions(context.Background()); !later.Ran {
		t.Fatal("did not reap after the interval elapsed")
	}
	if len(requests) != 2 || requests[0].DryRun || requests[0].Home != "/home/op" || requests[0].Initiator != "runtime-supervisor" ||
		requests[0].Policy.OlderThan != agentsession.DefaultReapOlderThan {
		t.Fatalf("requests = %+v", requests)
	}
}
