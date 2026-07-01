package plans

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	repocontract "github.com/vrooli/repo-contract-go"

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
		Workspace:       workspaceScopeFromFlag(ctx.Flag("workspace")),
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
	resp, err := h.client.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: id, Workspace: workspaceScopeFromFlag(ctx.Flag("workspace"))}))
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
	resp, err := h.client.ArchivePlan(context.Background(), connect.NewRequest(&plansv1.ArchivePlanRequest{Id: ctx.Positional("id"), Workspace: workspaceScopeFromFlag(ctx.Flag("workspace"))}))
	if err != nil {
		return cliapp.WrapAPIError("archive plan", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Archived")
}

func (h *handlers) render(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RenderMarkdown(context.Background(), connect.NewRequest(&plansv1.RenderMarkdownRequest{Id: id, Workspace: workspaceScopeFromFlag(ctx.Flag("workspace"))}))
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

func (h *handlers) depend(ctx cliapp.RunContext) error {
	resp, err := h.client.LinkDependency(context.Background(), connect.NewRequest(&plansv1.LinkDependencyRequest{
		DependingPlanId:  ctx.Positional("depending"),
		DependencyPlanId: ctx.Positional("dependency"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("link dependency", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Linked dependency on")
}

func (h *handlers) importPlan(ctx cliapp.RunContext) error {
	resp, err := h.client.ImportPlan(context.Background(), connect.NewRequest(&plansv1.ImportPlanRequest{
		SourcePath: ctx.Flag("source"),
		Markdown:   ctx.Flag("markdown"),
		Workspace:  workspaceScopeFromFlag(ctx.Flag("workspace")),
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

func (h *handlers) reconcile(ctx cliapp.RunContext) error {
	resp, err := h.client.ReconcilePlans(context.Background(), connect.NewRequest(&plansv1.ReconcilePlansRequest{
		DryRun:                 ctx.BoolFlag("dry-run"),
		RepairMirrors:          ctx.BoolFlag("repair-mirrors"),
		AdoptLegacy:            ctx.BoolFlag("adopt-legacy"),
		CleanupAdoptedSources:  ctx.BoolFlag("cleanup-adopted-sources"),
		IncludeArchived:        ctx.BoolFlag("include-archived"),
		IncludeArchivedLegacy:  ctx.BoolFlag("include-archived-legacy"),
		ConflictPolicy:         reconcileConflictPolicyFlag(ctx.Flag("conflict-policy")),
		SourceRuntimeHomePlans: ctx.BoolFlag("source-runtime-home-plans"),
		SourceDocsPlans:        ctx.BoolFlag("source-docs-plans"),
		SourceRepoPlans:        ctx.BoolFlag("source-repo-plans"),
		Workspace:              workspaceScopeFromFlag(ctx.Flag("workspace")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile plans", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetItems()))
	for _, item := range resp.Msg.GetItems() {
		results = append(results, formatReconcileItem(item))
	}
	mode := "Applied"
	if resp.Msg.GetDryRun() {
		mode = "Dry run"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s inspected %d item(s).", mode, len(resp.Msg.GetItems()))},
		ResultsHeading: "Reconcile results",
		Results:        results,
		RetrievalHints: []string{
			"`plans reconcile --dry-run` — preview mirror repair and legacy adoption",
			"`plans reconcile --repair-mirrors` — repair rendered markdown mirrors",
		},
	})
}

func workspaceScopeFromFlag(raw string) *plansv1.WorkspaceScope {
	root := strings.TrimSpace(raw)
	if root == "" {
		resolved, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return nil
		}
		root = resolved
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return &plansv1.WorkspaceScope{Root: filepath.Clean(root)}
	}
	return &plansv1.WorkspaceScope{Root: filepath.Clean(abs)}
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

// splitLinesOrCSV parses a list flag for the canonical phase fields: it splits on
// newlines when the value is multi-line (so a step can contain commas), and falls
// back to comma-separated otherwise. Used by direct phase add/update so the direct
// CLI reaches full phase parity with the authoring wizard.
func splitLinesOrCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.Contains(raw, "\n") {
		var out []string
		for _, line := range strings.Split(raw, "\n") {
			if v := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-")); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return splitCSV(raw)
}

func parsePhaseContext(raw string) []*sharedv1.RelevantContextItem {
	entries := splitCSV(raw)
	if len(entries) == 0 {
		return nil
	}
	out := make([]*sharedv1.RelevantContextItem, 0, len(entries))
	for _, entry := range entries {
		item := parseContextEntry(entry)
		item.Scope = sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_PHASE
		if item.RepeatPolicy == sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED {
			item.RepeatPolicy = sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY
		}
		if item.Source == sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_UNSPECIFIED {
			item.Source = sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED
		}
		if item.Status == sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_UNSPECIFIED {
			item.Status = sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY
		}
		out = append(out, item)
	}
	return out
}

func parseContextEntry(entry string) *sharedv1.RelevantContextItem {
	item := &sharedv1.RelevantContextItem{Required: true}
	fields := strings.Split(entry, ";")
	hasKV := false
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		hasKV = true
		switch strings.TrimSpace(strings.ToLower(key)) {
		case "kind":
			item.Kind = parseContextKind(value)
		case "label":
			item.Label = strings.TrimSpace(value)
		case "reason":
			item.Reason = strings.TrimSpace(value)
		case "instruction":
			item.Instruction = strings.TrimSpace(value)
		case "command":
			item.Command = strings.TrimSpace(value)
			if len(item.Argv) == 0 {
				item.Argv = strings.Fields(item.Command)
			}
		case "argv":
			item.Argv = splitFields(value, "|")
		case "target":
			item.Target = strings.TrimSpace(value)
		case "required":
			item.Required = strings.TrimSpace(strings.ToLower(value)) != "false"
		case "repeat":
			item.RepeatPolicy = parseRepeatPolicy(value)
		}
	}
	if !hasKV {
		item.Kind = sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE
		item.Instruction = strings.TrimSpace(entry)
		return item
	}
	if item.Kind == sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_UNSPECIFIED {
		switch {
		case item.Command != "":
			item.Kind = sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND
		case item.Target != "":
			item.Kind = sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC
		default:
			item.Kind = sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE
		}
	}
	return item
}

func splitFields(raw, sep string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, sep)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseContextKind(raw string) sharedv1.RelevantContextKind {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "skill":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SKILL
	case "doc":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC
	case "command":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND
	case "search":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SEARCH
	case "code_ref":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_CODE_REF
	case "req_ref":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_REQ_REF
	case "note":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE
	default:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_UNSPECIFIED
	}
}

func parseRepeatPolicy(raw string) sharedv1.RelevantContextRepeatPolicy {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "once_per_execution":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ONCE_PER_EXECUTION
	case "on_resume":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME
	case "every_phase":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_EVERY_PHASE
	case "phase_entry", "":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY
	case "as_needed":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_AS_NEEDED
	default:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED
	}
}

func (h *handlers) phaseAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddPhase(context.Background(), connect.NewRequest(&plansv1.AddPhaseRequest{
		PlanId: ctx.Positional("plan"),
		Phase: &sharedv1.Phase{
			Title:           ctx.Flag("title"),
			Intent:          ctx.Flag("intent"),
			AffectedAreas:   splitLinesOrCSV(ctx.Flag("affected-areas")),
			Steps:           splitLinesOrCSV(ctx.Flag("steps")),
			ExpectedOutputs: splitLinesOrCSV(ctx.Flag("expected-outputs")),
			Validation:      ctx.Flag("validation"),
			Acceptance:      ctx.Flag("acceptance"),
			RisksHazards:    splitLinesOrCSV(ctx.Flag("risks-hazards")),
			HandoffNotes:    ctx.Flag("handoff-notes"),
			RelevantContext: parsePhaseContext(ctx.Flag("context")),
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
			AffectedAreas:   splitLinesOrCSV(ctx.Flag("affected-areas")),
			Steps:           splitLinesOrCSV(ctx.Flag("steps")),
			ExpectedOutputs: splitLinesOrCSV(ctx.Flag("expected-outputs")),
			Validation:      ctx.Flag("validation"),
			Acceptance:      ctx.Flag("acceptance"),
			RisksHazards:    splitLinesOrCSV(ctx.Flag("risks-hazards")),
			HandoffNotes:    ctx.Flag("handoff-notes"),
			Status:          phaseStatusFlag(ctx.Flag("status")),
			RelevantContext: parsePhaseContext(ctx.Flag("context")),
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
	if mirror := p.GetMirror(); mirror != nil && mirror.GetPath() != "" {
		out = append(out, fmt.Sprintf("mirror path: %s", mirror.GetPath()))
		if status := mirrorStatusLabel(mirror.GetStatus()); status != "" && status != "fresh" {
			out = append(out, fmt.Sprintf("mirror status: %s", status))
		}
	}
	// Work posture is autofilled; surface it so a reviewer sees it without the
	// full rendered markdown.
	if label := workPostureLabel(p.WorkPosture); label != "" {
		out = append(out, fmt.Sprintf("work posture: %s", label))
	}
	out = appendField(out, "problem/need", p.ProblemStatement)
	out = appendField(out, "target outcome", p.TargetOutcome)
	out = appendField(out, "technical approach", p.TechnicalApproach)
	out = appendField(out, "validation strategy", p.ValidationStrategy)
	if p.GetImportProvenance() != nil {
		out = append(out, fmt.Sprintf("imported from: %s", p.GetImportProvenance().GetSourcePath()))
	}
	if n := len(p.GetPreservedLegacySections()); n > 0 {
		out = append(out, fmt.Sprintf("preserved legacy sections: %d", n))
	}
	for i, ph := range p.Phases {
		out = append(out, fmt.Sprintf("  phase %d: %s [%s] (id=%s)", i+1, ph.Title, phaseStatusLabel(ph.Status), ph.Id))
		if len(ph.GetSteps()) > 0 {
			out = append(out, fmt.Sprintf("    steps: %d · validation: %s", len(ph.GetSteps()), truncateOneLine(ph.GetValidation(), 60)))
		}
	}
	out = append(out, "(run `plan-manager plans render "+p.Slug+"` for the full markdown review artifact)")
	return out
}

func formatReconcileItem(item *plansv1.ReconcilePlanItem) string {
	if item == nil {
		return "(nil)"
	}
	parts := []string{reconcileActionLabel(item.GetAction())}
	if item.GetSlug() != "" {
		parts = append(parts, item.GetSlug())
	} else if item.GetPlanId() != "" {
		parts = append(parts, item.GetPlanId())
	}
	if item.GetTitle() != "" {
		parts = append(parts, "— "+truncateOneLine(item.GetTitle(), 72))
	}
	if item.GetSourcePath() != "" {
		parts = append(parts, "source="+item.GetSourcePath())
	}
	if mirror := item.GetMirror(); mirror != nil && mirror.GetPath() != "" {
		parts = append(parts, "mirror="+mirror.GetPath())
	}
	if item.GetSourceUntouched() {
		parts = append(parts, "source untouched")
	}
	if item.GetSourceCleanupPlanned() {
		parts = append(parts, "source cleanup planned")
	}
	if item.GetSourceRemoved() {
		parts = append(parts, "source removed")
	}
	if item.GetError() != "" {
		parts = append(parts, "error="+truncateOneLine(item.GetError(), 120))
	}
	return strings.Join(parts, " ")
}

// appendField appends a "label: value" detail line only when value is non-empty,
// truncated to one readable line (the full text lives in the rendered markdown).
func appendField(out []string, label, value string) []string {
	if strings.TrimSpace(value) == "" {
		return out
	}
	return append(out, fmt.Sprintf("%s: %s", label, truncateOneLine(value, 100)))
}

func truncateOneLine(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}

func workPostureLabel(p sharedv1.WorkPosture) string {
	switch p {
	case sharedv1.WorkPosture_WORK_POSTURE_GREENFIELD:
		return "greenfield"
	case sharedv1.WorkPosture_WORK_POSTURE_BROWNFIELD:
		return "brownfield"
	default:
		return ""
	}
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

func mirrorStatusLabel(s sharedv1.RenderedMirrorStatus) string {
	switch s {
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_FRESH:
		return "fresh"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_MISSING:
		return "missing"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_STALE:
		return "stale"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_WRITE_FAILED:
		return "write_failed"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_UNKNOWN:
		return "unknown"
	default:
		return ""
	}
}

func reconcileConflictPolicyFlag(s string) plansv1.ReconcileConflictPolicy {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "report_only":
		return plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_REPORT_ONLY
	case "skip_existing", "":
		return plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_SKIP_EXISTING
	default:
		return plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_UNSPECIFIED
	}
}

func reconcileActionLabel(a plansv1.ReconcileAction) string {
	switch a {
	case plansv1.ReconcileAction_RECONCILE_ACTION_ALREADY_CANONICAL:
		return "already_canonical"
	case plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_FRESH:
		return "mirror_fresh"
	case plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_REPAIR_NEEDED:
		return "mirror_repair_needed"
	case plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_REPAIRED:
		return "mirror_repaired"
	case plansv1.ReconcileAction_RECONCILE_ACTION_IMPORT_PLANNED:
		return "import_planned"
	case plansv1.ReconcileAction_RECONCILE_ACTION_IMPORTED:
		return "imported"
	case plansv1.ReconcileAction_RECONCILE_ACTION_SKIPPED_DUPLICATE:
		return "skipped_duplicate"
	case plansv1.ReconcileAction_RECONCILE_ACTION_PARSE_FAILED:
		return "parse_failed"
	case plansv1.ReconcileAction_RECONCILE_ACTION_CONFLICT:
		return "conflict"
	default:
		return "unspecified"
	}
}
