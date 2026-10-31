// This file applies the goal-home park policy (DL-1) to agent-initiated
// parks, so a raw `run park` cannot bypass `agent-manager effort park`.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/goalhome"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
)

// parkEnforceEnv switches legacy duration/admissible-work checks from
// report-only to refusal. Unacknowledged steering always refuses a new park.
const parkEnforceEnv = "VROOLI_EFFORT_PARK_ENFORCE"

// GoalHomeParkSource finds the goal home of the effort a run orchestrates.
// applies is false when the run orchestrates no effort with a registered goal
// home; the park policy then does not apply.
type GoalHomeParkSource interface {
	GoalHomeForOrchestrator(ctx context.Context, run *domain.Run) (home *goalhome.Home, dir string, applies bool, err error)
}

// SetGoalHomeParkSource installs the goal-home reader for the park policy.
func (o *Orchestrator) SetGoalHomeParkSource(source GoalHomeParkSource) { o.goalHomeParks = source }

func parkPolicyEnforced() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(parkEnforceEnv))) {
	case "1", "true", "yes", "on", "enforce":
		return true
	}
	return false
}

// checkGoalHomePark applies the shared home policy to an orchestrator's child
// wait. New steering must be acknowledged; other rules retain their existing
// report/enforce setting. An unreadable applicable home cannot prove steering is handled.
func (o *Orchestrator) checkGoalHomePark(ctx context.Context, run *domain.Run, producer string, deadline *time.Time) string {
	if o.goalHomeParks == nil || run == nil {
		return ""
	}
	home, dir, applies, err := o.goalHomeParks.GoalHomeForOrchestrator(ctx, run)
	if err != nil {
		return fmt.Sprintf("park refused: cannot resolve current goal-home steering: %v", err)
	}
	if !applies {
		return ""
	}
	if home == nil {
		obs.Component("park-policy").Warn("goal home unreadable; steering cannot be checked", obs.KeyRunID, run.ID.String(), "goal_home", dir, obs.KeyError, fmt.Sprint(err))
		return fmt.Sprintf("park refused: cannot read current steering in goal home %s: %v", dir, err)
	}
	// Steering is independent of the producer being waited on. Only legacy
	// child-duration/admissible-work rules are specific to children waits.
	steering := goalhome.DecideFeedbackPark(home)
	if steering.Rule == "feedback-unacknowledged" {
		return fmt.Sprintf("park refused by the goal-home park policy (%s, goal home %s): %s", steering.Rule, dir, steering.Reason)
	}
	if producer != ProducerChildren {
		return ""
	}
	live, err := o.liveDirectChildren(ctx, run)
	if err != nil {
		obs.Component("park-policy").Warn("children unreadable; park policy not applied", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return ""
	}
	requested := DefaultParkTTL
	if deadline != nil {
		requested = deadline.Sub(o.now())
	}
	decision := goalhome.DecideHomePark(live, home, requested)
	if decision.Allowed {
		return ""
	}
	message := fmt.Sprintf("park refused by the goal-home park policy (%s, goal home %s): %s. Use `agent-manager effort park %s --run %s`.", decision.Rule, dir, decision.Reason, dir, run.ID)
	if decision.Rule == "feedback-unacknowledged" || parkPolicyEnforced() {
		return message
	}
	o.appendParkPolicyEvent(ctx, run, "park policy (report-only; "+parkEnforceEnv+" is off): "+message)
	return ""
}

func (o *Orchestrator) liveDirectChildren(ctx context.Context, run *domain.Run) ([]string, error) {
	children, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 1000}, ParentRunID: &run.ID})
	if err != nil {
		return nil, err
	}
	var live []string
	for _, child := range children {
		if child != nil && isParkExecutionChild(child.Status) {
			live = append(live, child.ID.String())
		}
	}
	return live, nil
}

func (o *Orchestrator) appendParkPolicyEvent(ctx context.Context, run *domain.Run, message string) {
	if o.events == nil {
		return
	}
	if err := o.appendAndBroadcastEvents(ctx, run.ID, domain.NewLogEvent(run.ID, "warn", message)); err != nil {
		obs.Component("park-policy").Warn("park policy event append failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

// isParkExecutionChild tracks runs that can still consume execution time.
// Needs-review runs remain nonterminal for approval/continuation, but their
// worker has exited; parked runs remain resumable and keep the backstop.
func isParkExecutionChild(status domain.RunStatus) bool {
	return status.IsActive() || status == domain.RunStatusParked
}
