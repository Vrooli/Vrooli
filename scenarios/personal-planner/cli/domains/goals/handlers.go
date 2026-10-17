package goals

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals/goals_v1connect"
)

type handlers struct{ client gc.GoalsServiceClient }

func newHandlers(c *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(c)
	return &handlers{client: gc.NewGoalsServiceClient(httpClient, base)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*v.ListGoalsResponse, error) {
	r, e := h.client.ListGoals(context.Background(), connect.NewRequest(&v.ListGoalsRequest{}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list goals", e, nil)
	}
	if r == nil || r.Msg == nil {
		return nil, fmt.Errorf("server returned no goals")
	}
	return r.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, m *v.ListGoalsResponse) cliapp.ListReport {
	out := []string{}
	for _, g := range m.Goals {
		out = append(out, fmt.Sprintf("%s — %s [%d%%]", g.Id, g.Title, g.ProgressBasisPoints/100))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d goal(s).", len(m.Goals))}, ResultsHeading: "Goals", Results: out}
}

func (h *handlers) createCall(c cliapp.OperationContext) (*v.CreateGoalResponse, error) {
	targetText := c.Flag("target-percent")
	if targetText == "" {
		targetText = "100"
	}
	target, e := strconv.ParseInt(targetText, 10, 64)
	if e != nil {
		return nil, fmt.Errorf("target-percent must be an integer: %w", e)
	}
	method := c.Flag("progress-method")
	if method == "" {
		method = "manual"
	}
	r, e := h.client.CreateGoal(context.Background(), connect.NewRequest(&v.CreateGoalRequest{Title: c.Flag("title"), Purpose: c.Flag("purpose"), ProgressMethod: method, TargetBasisPoints: target * 100}))
	if e != nil {
		return nil, cliapp.WrapAPIError("create goal", e, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Goal == nil {
		return nil, fmt.Errorf("server returned no goal")
	}
	return r.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, m *v.CreateGoalResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created goal %s.", m.Goal.Id)}, Changes: []string{fmt.Sprintf("%s — %s", m.Goal.Id, m.Goal.Title)}, NextCommand: []string{"`goals list` — inspect goals"}}
}

func (h *handlers) progressCall(c cliapp.OperationContext) (*v.UpdateGoalProgressResponse, error) {
	p, e := strconv.ParseInt(c.Flag("percent"), 10, 64)
	if e != nil {
		return nil, fmt.Errorf("percent must be an integer: %w", e)
	}
	r, e := strconv.ParseInt(c.Flag("revision"), 10, 64)
	if e != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", e)
	}
	resp, e := h.client.UpdateGoalProgress(context.Background(), connect.NewRequest(&v.UpdateGoalProgressRequest{Id: c.Positional("id"), ProgressBasisPoints: p * 100, ExpectedRevision: r}))
	if e != nil {
		return nil, cliapp.WrapAPIError("update goal progress", e, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Goal == nil {
		return nil, fmt.Errorf("server returned no goal")
	}
	return resp.Msg, nil
}

func (h *handlers) progressReport(_ cliapp.OperationContext, m *v.UpdateGoalProgressResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded %d%% progress for %s.", m.Goal.ProgressBasisPoints/100, m.Goal.Id)}}
}

func (h *handlers) milestonesCall(c cliapp.OperationContext) (*v.ListMilestonesResponse, error) {
	r, e := h.client.ListMilestones(context.Background(), connect.NewRequest(&v.ListMilestonesRequest{GoalId: c.Positional("goal-id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list milestones", e, nil)
	}
	if r == nil || r.Msg == nil {
		return nil, fmt.Errorf("server returned no milestones")
	}
	return r.Msg, nil
}

func (h *handlers) milestonesReport(_ cliapp.OperationContext, m *v.ListMilestonesResponse) cliapp.ListReport {
	results := make([]string, 0, len(m.Milestones))
	for _, milestone := range m.Milestones {
		results = append(results, fmt.Sprintf("%s — %s [%s]", milestone.Id, milestone.Title, milestone.Status))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d milestone(s).", len(m.Milestones))}, ResultsHeading: "Milestones", Results: results}
}

func (h *handlers) createMilestoneCall(c cliapp.OperationContext) (*v.CreateMilestoneResponse, error) {
	r, e := h.client.CreateMilestone(context.Background(), connect.NewRequest(&v.CreateMilestoneRequest{GoalId: c.Flag("goal-id"), Title: c.Flag("title"), Criteria: c.Flag("criteria"), DueDate: c.Flag("due-date"), LinkedWorkItemId: c.Flag("work-item-id"), PrerequisiteMilestoneIds: splitIDs(c.Flag("prerequisite-ids"))}))
	if e != nil {
		return nil, cliapp.WrapAPIError("create milestone", e, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Milestone == nil {
		return nil, fmt.Errorf("server returned no milestone")
	}
	return r.Msg, nil
}

func splitIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func (h *handlers) createMilestoneReport(_ cliapp.OperationContext, m *v.CreateMilestoneResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created milestone %s.", m.Milestone.Id)}, Changes: []string{m.Milestone.Title}, NextCommand: []string{"`goals milestones --goal-id <id>` — inspect milestones"}}
}

func (h *handlers) completeMilestoneCall(c cliapp.OperationContext) (*v.UpdateMilestoneStatusResponse, error) {
	revision, e := strconv.ParseInt(c.Flag("revision"), 10, 64)
	if e != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", e)
	}
	r, e := h.client.UpdateMilestoneStatus(context.Background(), connect.NewRequest(&v.UpdateMilestoneStatusRequest{Id: c.Positional("id"), Status: "complete", ExpectedRevision: revision}))
	if e != nil {
		return nil, cliapp.WrapAPIError("complete milestone", e, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Milestone == nil {
		return nil, fmt.Errorf("server returned no milestone")
	}
	return r.Msg, nil
}

func (h *handlers) completeMilestoneReport(_ cliapp.OperationContext, m *v.UpdateMilestoneStatusResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Completed milestone %s.", m.Milestone.Id)}}
}
