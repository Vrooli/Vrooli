package review

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review"
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review/review_v1connect"
)

type handlers struct{ client gc.ReviewServiceClient }

func newHandlers(c *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(c)
	return &handlers{client: gc.NewReviewServiceClient(httpClient, base)}
}

func (h *handlers) dailyCall(c cliapp.OperationContext) (*v.GetDailySummaryResponse, error) {
	r, e := h.client.GetDailySummary(context.Background(), connect.NewRequest(&v.GetDailySummaryRequest{LocalDate: c.Flag("date")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("get daily review", e, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Summary == nil {
		return nil, fmt.Errorf("server returned no daily review")
	}
	return r.Msg, nil
}

func (h *handlers) dailyReport(_ cliapp.OperationContext, m *v.GetDailySummaryResponse) cliapp.ListReport {
	s := m.Summary
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s: %d recorded active minutes across %d focus session(s).", s.LocalDate, s.RecordedActiveMinutes, s.FocusSessionCount)}, ResultsHeading: "Daily review", Results: []string{fmt.Sprintf("Active goals: %d; planned: %d min; unrecorded: unknown.", s.ActiveGoalCount, s.PlannedMinutes), s.CoverageNote}}
}

func (h *handlers) weeklyCall(c cliapp.OperationContext) (*v.GetWeeklySummaryResponse, error) {
	r, e := h.client.GetWeeklySummary(context.Background(), connect.NewRequest(&v.GetWeeklySummaryRequest{WeekStartLocalDate: c.Flag("week-start")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("get weekly review", e, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Summary == nil {
		return nil, fmt.Errorf("server returned no weekly review")
	}
	return r.Msg, nil
}

func (h *handlers) weeklyReport(_ cliapp.OperationContext, m *v.GetWeeklySummaryResponse) cliapp.ListReport {
	s := m.Summary
	results := make([]string, 0, len(s.Days)+1)
	for _, day := range s.Days {
		results = append(results, fmt.Sprintf("%s: planned %d min · recorded %d min · %d focus session(s)", day.LocalDate, day.PlannedMinutes, day.RecordedActiveMinutes, day.FocusSessionCount))
	}
	results = append(results, s.CoverageNote)
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Week of %s: %d planned minutes, %d recorded active minutes.", s.WeekStartLocalDate, s.PlannedMinutes, s.RecordedActiveMinutes)}, ResultsHeading: "Weekly review", Results: results}
}

func (h *handlers) reflectionCall(c cliapp.OperationContext) (*v.GetReflectionResponse, error) {
	r, e := h.client.GetReflection(context.Background(), connect.NewRequest(&v.GetReflectionRequest{LocalDate: c.Flag("date")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("get reflection", e, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Reflection == nil {
		return nil, fmt.Errorf("server returned no reflection")
	}
	return r.Msg, nil
}

func (h *handlers) reflectionReport(_ cliapp.OperationContext, m *v.GetReflectionResponse) cliapp.ListReport {
	if m.Reflection.Text == "" {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s: no reflection recorded.", m.Reflection.LocalDate)}, ResultsHeading: "Daily reflection", Results: []string{"Nothing has been written yet."}}
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Reflection for %s", m.Reflection.LocalDate)}, ResultsHeading: "Daily reflection", Results: []string{m.Reflection.Text, "Updated: " + m.Reflection.UpdatedAt}}
}

func (h *handlers) saveReflectionCall(c cliapp.OperationContext) (*v.SaveReflectionResponse, error) {
	r, e := h.client.SaveReflection(context.Background(), connect.NewRequest(&v.SaveReflectionRequest{LocalDate: c.Flag("date"), Text: c.Flag("text")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("save reflection", e, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Reflection == nil {
		return nil, fmt.Errorf("server returned no saved reflection")
	}
	return r.Msg, nil
}

func (h *handlers) saveReflectionReport(_ cliapp.OperationContext, m *v.SaveReflectionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved reflection for %s.", m.Reflection.LocalDate)}, Changes: []string{"The reflection is durable and will appear in Review."}}
}
