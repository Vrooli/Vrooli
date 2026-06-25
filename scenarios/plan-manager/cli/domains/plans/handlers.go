package plans

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the generated PlansService client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client plansconnect.PlansServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: plansconnect.NewPlansServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListPlans(context.Background(), connect.NewRequest(&plansv1.ListPlansRequest{
		Status:          planStatusFlag(ctx.Flag("status")),
		IncludeArchived: ctx.BoolFlag("include-archived"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list plans", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no plans response")
	}
	results := make([]string, 0, len(resp.Msg.Plans))
	for _, p := range resp.Msg.Plans {
		results = append(results, formatPlan(p))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d plan(s).", len(resp.Msg.Plans))},
		ResultsHeading: "Plans",
		Results:        results,
		RetrievalHints: []string{
			"`plans get <id>` — show a plan",
			"`plans render <id>` — render the markdown view",
			"`plans create --title <title>` — create a plan",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get plan %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Plan == nil {
		return fmt.Errorf("server returned no plan")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched plan %s.", resp.Msg.Plan.Id)},
		ResultsHeading: "Plan",
		Results:        planDetail(resp.Msg.Plan),
	})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	plan := &sharedv1.Plan{
		Title:            ctx.Flag("title"),
		Slug:             ctx.Flag("slug"),
		Purpose:          ctx.Flag("purpose"),
		Scope:            ctx.Flag("scope"),
		Constraints:      ctx.Flag("constraints"),
		NonGoals:         ctx.Flag("non-goals"),
		DefinitionOfDone: ctx.Flag("dod"),
	}
	resp, err := h.client.CreatePlan(context.Background(), connect.NewRequest(&plansv1.CreatePlanRequest{Plan: plan}))
	if err != nil {
		return cliapp.WrapAPIError("create plan", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Created")
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	plan := &sharedv1.Plan{
		Id:               ctx.Positional("id"),
		Title:            ctx.Flag("title"),
		Purpose:          ctx.Flag("purpose"),
		Scope:            ctx.Flag("scope"),
		Constraints:      ctx.Flag("constraints"),
		NonGoals:         ctx.Flag("non-goals"),
		DefinitionOfDone: ctx.Flag("dod"),
	}
	resp, err := h.client.UpdatePlan(context.Background(), connect.NewRequest(&plansv1.UpdatePlanRequest{Plan: plan}))
	if err != nil {
		return cliapp.WrapAPIError("update plan", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Updated")
}

func (h *handlers) archive(ctx cliapp.RunContext) error {
	resp, err := h.client.ArchivePlan(context.Background(), connect.NewRequest(&plansv1.ArchivePlanRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("archive plan", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Archived")
}

func (h *handlers) render(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RenderMarkdown(context.Background(), connect.NewRequest(&plansv1.RenderMarkdownRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("render plan %q", id), err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Rendered plan %s.", id)},
		ResultsHeading: "Markdown",
		Results:        []string{resp.Msg.GetMarkdown()},
	})
}

func (h *handlers) graph(ctx cliapp.RunContext) error {
	resp, err := h.client.GetGraph(context.Background(), connect.NewRequest(&plansv1.GetGraphRequest{PlanId: ctx.Flag("plan")}))
	if err != nil {
		return cliapp.WrapAPIError("get plan graph", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Edges))
	for _, e := range resp.Msg.Edges {
		results = append(results, fmt.Sprintf("%s --%s--> %s", e.FromPlanId, e.Kind, e.ToPlanId))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d edge(s).", len(resp.Msg.Edges))},
		ResultsHeading: "Plan graph",
		Results:        results,
	})
}

func (h *handlers) link(ctx cliapp.RunContext) error {
	resp, err := h.client.LinkSupersession(context.Background(), connect.NewRequest(&plansv1.LinkSupersessionRequest{
		SupersedingPlanId: ctx.Positional("superseding"),
		SupersededPlanId:  ctx.Positional("superseded"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("link supersession", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Linked supersession on")
}

func (h *handlers) importPlan(ctx cliapp.RunContext) error {
	resp, err := h.client.ImportPlan(context.Background(), connect.NewRequest(&plansv1.ImportPlanRequest{
		SourcePath: ctx.Flag("source"),
		Markdown:   ctx.Flag("markdown"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("import plan", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Imported")
}

func (h *handlers) migrate(ctx cliapp.RunContext) error {
	resp, err := h.client.MigratePlan(context.Background(), connect.NewRequest(&plansv1.MigratePlanRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("migrate plan", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Migrated")
}

// splitCSV splits a comma-separated flag value into a trimmed, non-empty list.
// An empty/whitespace value yields nil so an unset list flag leaves the field
// empty rather than carrying a single blank entry.
func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (h *handlers) phaseAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddPhase(context.Background(), connect.NewRequest(&plansv1.AddPhaseRequest{
		PlanId: ctx.Positional("plan"),
		Phase: &sharedv1.Phase{
			Title:           ctx.Flag("title"),
			Intent:          ctx.Flag("intent"),
			Acceptance:      ctx.Flag("acceptance"),
			RequiredReading: splitCSV(ctx.Flag("required-reading")),
			Reminders:       splitCSV(ctx.Flag("reminders")),
			BaselineScope:   splitCSV(ctx.Flag("baseline-scope")),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("add phase", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Added phase to")
}

func (h *handlers) phaseUpdate(ctx cliapp.RunContext) error {
	resp, err := h.client.UpdatePhase(context.Background(), connect.NewRequest(&plansv1.UpdatePhaseRequest{
		PlanId: ctx.Positional("plan"),
		Phase: &sharedv1.Phase{
			Id:              ctx.Positional("phase"),
			Title:           ctx.Flag("title"),
			Intent:          ctx.Flag("intent"),
			Acceptance:      ctx.Flag("acceptance"),
			Status:          phaseStatusFlag(ctx.Flag("status")),
			RequiredReading: splitCSV(ctx.Flag("required-reading")),
			Reminders:       splitCSV(ctx.Flag("reminders")),
			BaselineScope:   splitCSV(ctx.Flag("baseline-scope")),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("update phase", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Updated phase on")
}

func (h *handlers) templateList(ctx cliapp.RunContext) error {
	resp, err := h.client.ListTemplates(context.Background(), connect.NewRequest(&plansv1.ListTemplatesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list templates", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Templates))
	for _, t := range resp.Msg.Templates {
		results = append(results, fmt.Sprintf("%s — %s [%s]", t.Id, t.Name, t.Surface))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d template(s).", len(resp.Msg.Templates))},
		ResultsHeading: "Templates",
		Results:        results,
		RetrievalHints: []string{"`template new <id> --title <title>` — start a plan from a template"},
	})
}

func (h *handlers) templateNew(ctx cliapp.RunContext) error {
	resp, err := h.client.CreateFromTemplate(context.Background(), connect.NewRequest(&plansv1.CreateFromTemplateRequest{
		TemplateId: ctx.Positional("template"),
		Title:      ctx.Flag("title"),
		Slug:       ctx.Flag("slug"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create from template", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Created")
}

func (h *handlers) renderMutation(ctx cliapp.RunContext, p *sharedv1.Plan, verb string) error {
	if p == nil {
		return fmt.Errorf("server returned no plan")
	}
	return cliapp.RenderProtoMutation(ctx, &plansv1.GetPlanResponse{Plan: p}, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s plan %s.", verb, p.Id)},
		Changes: planDetail(p),
		NextCommand: []string{
			fmt.Sprintf("`plans get %s` — show this plan", p.Id),
			fmt.Sprintf("`plans render %s` — render the markdown view", p.Id),
		},
	})
}

func formatPlan(p *sharedv1.Plan) string {
	if p == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [%s, phases=%d]", p.Slug, p.Title, planStatusLabel(p.Status), len(p.Phases))
}

func planDetail(p *sharedv1.Plan) []string {
	if p == nil {
		return []string{"(nil)"}
	}
	out := []string{
		fmt.Sprintf("id: %s", p.Id),
		fmt.Sprintf("slug: %s", p.Slug),
		fmt.Sprintf("title: %s", p.Title),
		fmt.Sprintf("status: %s", planStatusLabel(p.Status)),
	}
	if p.ContentHash != "" {
		out = append(out, fmt.Sprintf("content-hash: %s", p.ContentHash))
	}
	for i, ph := range p.Phases {
		out = append(out, fmt.Sprintf("  phase %d: %s [%s] (id=%s)", i+1, ph.Title, phaseStatusLabel(ph.Status), ph.Id))
	}
	return out
}

// --- flag → proto enum helpers (unknown values fall through to UNSPECIFIED, a
// no-op filter for reads). ---

func planStatusFlag(s string) sharedv1.PlanStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft":
		return sharedv1.PlanStatus_PLAN_STATUS_DRAFT
	case "active":
		return sharedv1.PlanStatus_PLAN_STATUS_ACTIVE
	case "complete":
		return sharedv1.PlanStatus_PLAN_STATUS_COMPLETE
	case "archived":
		return sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED
	default:
		return sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED
	}
}

func phaseStatusFlag(s string) sharedv1.PhaseStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "todo":
		return sharedv1.PhaseStatus_PHASE_STATUS_TODO
	case "active":
		return sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE
	case "done":
		return sharedv1.PhaseStatus_PHASE_STATUS_DONE
	case "blocked":
		return sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED
	default:
		return sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED
	}
}

func planStatusLabel(s sharedv1.PlanStatus) string {
	switch s {
	case sharedv1.PlanStatus_PLAN_STATUS_DRAFT:
		return "draft"
	case sharedv1.PlanStatus_PLAN_STATUS_ACTIVE:
		return "active"
	case sharedv1.PlanStatus_PLAN_STATUS_COMPLETE:
		return "complete"
	case sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED:
		return "archived"
	default:
		return "unspecified"
	}
}

func phaseStatusLabel(s sharedv1.PhaseStatus) string {
	switch s {
	case sharedv1.PhaseStatus_PHASE_STATUS_TODO:
		return "todo"
	case sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE:
		return "active"
	case sharedv1.PhaseStatus_PHASE_STATUS_DONE:
		return "done"
	case sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED:
		return "blocked"
	default:
		return "todo"
	}
}
