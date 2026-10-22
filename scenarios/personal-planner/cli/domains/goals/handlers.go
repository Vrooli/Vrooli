package goals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals/goals_v1connect"
)

type handlers struct {
	client     gc.GoalsServiceClient
	httpClient connect.HTTPClient
	baseURL    string
}

func newHandlers(c *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(c)
	return &handlers{client: gc.NewGoalsServiceClient(httpClient, base), httpClient: httpClient, baseURL: base}
}

type goalDTO struct {
	ID                  string          `json:"id"`
	Title               string          `json:"title"`
	Purpose             string          `json:"purpose"`
	Status              string          `json:"status"`
	ProgressMethod      string          `json:"progress_method"`
	ProgressBasisPoints *int64          `json:"progress_basis_points,omitempty"`
	ProgressState       string          `json:"progress_state,omitempty"`
	TargetBasisPoints   int64           `json:"target_basis_points"`
	Revision            int64           `json:"revision"`
	TargetDate          string          `json:"target_date,omitempty"`
	Measure             *goalMeasureDTO `json:"measure,omitempty"`
}

type goalListDTO struct {
	Goals []goalDTO `json:"goals"`
}

type goalMeasureDTO struct {
	Unit              string `json:"unit"`
	TargetAmountMinor int64  `json:"target_amount_minor"`
	CurrencyCode      string `json:"currency_code"`
	Basis             string `json:"basis"`
	Recurrence        string `json:"recurrence"`
	Period            string `json:"period"`
}

func (h *handlers) listCall(_ cliapp.OperationContext) (goalListDTO, error) {
	var response goalListDTO
	if err := h.request(http.MethodGet, nil, &response); err != nil {
		return goalListDTO{}, cliapp.WrapAPIError("list goals", err, nil)
	}
	return response, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, response goalListDTO) cliapp.MutationReport {
	rows := make([]string, 0, len(response.Goals))
	for _, goal := range response.Goals {
		rows = append(rows, formatGoal(goal))
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Found %d goal(s).", len(response.Goals))}, Changes: rows}
}

func (h *handlers) createCall(c cliapp.OperationContext) (goalDTO, error) {
	targetText := c.Flag("target-percent")
	if targetText == "" {
		targetText = "100"
	}
	target, e := strconv.ParseInt(targetText, 10, 64)
	if e != nil {
		return goalDTO{}, fmt.Errorf("target-percent must be an integer: %w", e)
	}
	method := c.Flag("progress-method")
	if method == "" {
		method = "manual"
	}
	var measure *goalMeasureDTO
	measureFlags := []string{"measure-unit", "target-amount-minor", "currency-code", "measure-basis", "recurrence", "period"}
	for _, name := range measureFlags {
		if strings.TrimSpace(c.Flag(name)) != "" {
			measure = &goalMeasureDTO{Unit: c.Flag("measure-unit"), CurrencyCode: c.Flag("currency-code"), Basis: c.Flag("measure-basis"), Recurrence: c.Flag("recurrence"), Period: c.Flag("period")}
			if amount := c.Flag("target-amount-minor"); amount != "" {
				measure.TargetAmountMinor, e = strconv.ParseInt(amount, 10, 64)
				if e != nil {
					return goalDTO{}, fmt.Errorf("target-amount-minor must be an integer: %w", e)
				}
			}
			break
		}
	}
	var response struct {
		Goal goalDTO `json:"goal"`
	}
	body := map[string]any{"title": c.Flag("title"), "purpose": c.Flag("purpose"), "progress_method": method, "target_basis_points": target * 100, "target_date": c.Flag("target-date")}
	if measure != nil {
		body["measure"] = measure
	}
	if err := h.request(http.MethodPost, body, &response); err != nil {
		return goalDTO{}, cliapp.WrapAPIError("create goal", err, nil)
	}
	if response.Goal.ID == "" {
		return goalDTO{}, fmt.Errorf("server returned no goal")
	}
	return response.Goal, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, goal goalDTO) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created goal %s.", goal.ID)}, Changes: []string{formatGoal(goal)}, NextCommand: []string{"`goals list` — inspect goals"}}
}

func formatGoal(goal goalDTO) string {
	if goal.Measure != nil {
		amount := float64(goal.Measure.TargetAmountMinor) / 100
		return fmt.Sprintf("%s — %s [%s %s %s; %s/%s; target %s; progress unknown]", goal.ID, goal.Title, goal.Measure.CurrencyCode, fmt.Sprintf("%.2f", amount), goal.Measure.Basis, goal.Measure.Recurrence, goal.Measure.Period, goal.TargetDate)
	}
	progress := int64(0)
	if goal.ProgressBasisPoints != nil {
		progress = *goal.ProgressBasisPoints
	}
	return fmt.Sprintf("%s — %s [%d%%]", goal.ID, goal.Title, progress/100)
}

func (h *handlers) request(method string, body any, output any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, strings.TrimRight(h.baseURL, "/")+"/api/v1/goals", reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := h.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("API returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	return json.NewDecoder(response.Body).Decode(output)
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
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded %d%% progress for %s.", m.Goal.GetProgressBasisPoints()/100, m.Goal.Id)}}
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
