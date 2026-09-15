package assignments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"
	assignmentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments/assignments_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client.
type handlers struct {
	core   *cliapp.ScenarioApp
	client assignmentsconnect.AssignmentsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: assignmentsconnect.NewAssignmentsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListAssignments(context.Background(), connect.NewRequest(&assignmentsv1.ListAssignmentsRequest{
		BrandId: ctx.Flag("brand-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list assignments", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no assignments response")
	}
	results := make([]string, 0, len(resp.Msg.Assignments))
	for _, a := range resp.Msg.Assignments {
		results = append(results, formatAssignment(a))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d assignment(s).", len(resp.Msg.Assignments))},
		ResultsHeading: "Assignments",
		Results:        results,
		RetrievalHints: []string{
			"`assignments status <scenario>` — show one scenario's branding",
			"`assignments assign --brand-id <id> --scenario <name>` — assign a brand",
		},
	})
}

func (h *handlers) assign(ctx cliapp.RunContext) error {
	resp, err := h.client.AssignBrand(context.Background(), connect.NewRequest(&assignmentsv1.AssignBrandRequest{
		BrandId:      ctx.Flag("brand-id"),
		ScenarioName: ctx.Flag("scenario"),
		Elements:     splitElements(ctx.Flag("elements")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("assign brand", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Assignment == nil {
		return fmt.Errorf("server returned no assignment")
	}
	a := resp.Msg.Assignment
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Assigned brand %s to %s (brand v%d).", a.BrandId, a.ScenarioName, a.BrandVersion)},
		Changes: []string{formatAssignment(a)},
		NextCommand: []string{
			fmt.Sprintf("`assignments status %s` — confirm the assignment", a.ScenarioName),
			fmt.Sprintf("`assignments unassign %s` — remove it", a.ScenarioName),
		},
	})
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetScenarioStatus(context.Background(), connect.NewRequest(&assignmentsv1.GetScenarioStatusRequest{
		ScenarioName: scenario,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get status for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return fmt.Errorf("server returned no status")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Branding status for %s.", scenario)},
		ResultsHeading: "Status",
		Results:        []string{formatStatus(resp.Msg.Status)},
	})
}

func (h *handlers) unassign(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	_, err := h.client.UnassignScenario(context.Background(), connect.NewRequest(&assignmentsv1.UnassignScenarioRequest{
		ScenarioName: scenario,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("unassign %q", scenario), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, &assignmentsv1.UnassignScenarioResponse{}, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Unassigned brand from %s (idempotent).", scenario)},
		NextCommand: []string{"`assignments list` — show remaining assignments"},
	})
}

// splitElements parses a comma-separated --elements flag into a trimmed,
// empties-dropped slice. Empty input yields nil so the server stores no
// elements.
func splitElements(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func formatAssignment(a *assignmentsv1.Assignment) string {
	if a == nil {
		return "(nil)"
	}
	applied := ""
	if a.AppliedAt != nil {
		applied = a.AppliedAt.AsTime().Format(time.RFC3339)
	}
	elements := "—"
	if len(a.Elements) > 0 {
		elements = strings.Join(a.Elements, ",")
	}
	return fmt.Sprintf("%s → %s [brand v%d elements=%s applied=%s]", a.ScenarioName, a.BrandId, a.BrandVersion, elements, applied)
}

func formatStatus(s *assignmentsv1.ScenarioStatus) string {
	if s == nil {
		return "(nil)"
	}
	if !s.HasBrand {
		return fmt.Sprintf("%s — no brand assigned", s.Scenario)
	}
	elements := "—"
	if len(s.Elements) > 0 {
		elements = strings.Join(s.Elements, ",")
	}
	return fmt.Sprintf("%s — brand %s [v%d elements=%s]", s.Scenario, s.BrandId, s.BrandVersion, elements)
}
