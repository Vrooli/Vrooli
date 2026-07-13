package plans

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"plan-manager/cli/internal/statusconv"

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
		Status:          statusconv.PlanStatusFlag(ctx.Flag("status")),
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
	workspace := workspaceScopeFromFlag(ctx.Flag("workspace"))
	plan := &sharedv1.Plan{
		Title:            ctx.Flag("title"),
		Slug:             ctx.Flag("slug"),
		Purpose:          ctx.Flag("purpose"),
		Scope:            ctx.Flag("scope"),
		Constraints:      ctx.Flag("constraints"),
		NonGoals:         ctx.Flag("non-goals"),
		DefinitionOfDone: ctx.Flag("dod"),
	}
	if workspace != nil {
		plan.WorkspaceRoot = workspace.GetRoot()
	}
	resp, err := h.client.CreatePlan(context.Background(), connect.NewRequest(&plansv1.CreatePlanRequest{Plan: plan}))
	if err != nil {
		return cliapp.WrapAPIError("create plan", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Created")
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	got, err := h.client.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{
		Id:        id,
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get plan %q for update", id), err, nil)
	}
	plan := got.Msg.GetPlan()
	if plan == nil {
		return fmt.Errorf("server returned no plan")
	}
	applyStringFlag(ctx, "title", func(v string) { plan.Title = v })
	applyStringFlag(ctx, "purpose", func(v string) { plan.Purpose = v })
	applyStringFlag(ctx, "problem", func(v string) { plan.ProblemStatement = v })
	applyStringFlag(ctx, "outcome", func(v string) { plan.TargetOutcome = v })
	applyStringFlag(ctx, "scope", func(v string) { plan.Scope = v })
	applyStringFlag(ctx, "constraints", func(v string) { plan.Constraints = v })
	applyStringFlag(ctx, "non-goals", func(v string) { plan.NonGoals = v })
	applyStringFlag(ctx, "assumptions", func(v string) { plan.Assumptions = v })
	applyStringFlag(ctx, "technical-approach", func(v string) { plan.TechnicalApproach = v })
	applyStringFlag(ctx, "validation-strategy", func(v string) { plan.ValidationStrategy = v })
	applyStringFlag(ctx, "risks", func(v string) { plan.RisksHazards = v })
	applyStringFlag(ctx, "prohibited-approaches", func(v string) { plan.ProhibitedApproaches = v })
	applyStringFlag(ctx, "dod", func(v string) { plan.DefinitionOfDone = v })
	if values := splitPlanFlagValues(ctx.FlagValues("final-validation-command")); len(values) > 0 {
		plan.FinalValidationCommands = values
	}
	if values := splitPlanFlagValues(ctx.FlagValues("change-allow")); len(values) > 0 {
		ensureChangeBoundary(plan).AcceptanceAllow = values
	}
	if values := splitPlanFlagValues(ctx.FlagValues("change-deny")); len(values) > 0 {
		ensureChangeBoundary(plan).AcceptanceDeny = values
	}
	applyStringFlag(ctx, "operator-only", func(v string) { ensureChangeBoundary(plan).OperatorOnlyReason = v })
	if mode := strings.ToLower(strings.TrimSpace(ctx.Flag("baseline-mode"))); mode != "" {
		switch mode {
		case "legacy":
			plan.BaselineSet = &sharedv1.BaselineSetIntent{Compatibility: "legacy_anchor"}
		case "current":
			// The service derives the current intent from a change-boundary anchor.
			plan.BaselineSet = nil
		default:
			return fmt.Errorf("--baseline-mode must be legacy or current")
		}
	}
	applyStringFlag(ctx, "anchor-strategy", func(v string) { ensureRegressionAnchor(plan).Strategy = v })
	applyStringFlag(ctx, "anchor-scenario", func(v string) { ensureRegressionAnchor(plan).Scenario = v })
	applyStringFlag(ctx, "anchor-baseline", func(v string) { ensureRegressionAnchor(plan).BaselineName = v })
	applyStringFlag(ctx, "anchor-head-sha", func(v string) { ensureRegressionAnchor(plan).HeadSha = v })
	applyStringFlag(ctx, "anchor-captured-at", func(v string) { ensureRegressionAnchor(plan).CapturedAt = v })
	if values := splitPlanFlagValues(ctx.FlagValues("anchor-allow")); len(values) > 0 {
		ensureRegressionAnchor(plan).AllowlistPaths = values
	}
	if values := splitPlanFlagValues(ctx.FlagValues("anchor-command")); len(values) > 0 {
		ensureRegressionAnchor(plan).Commands = values
	}
	if ctx.BoolFlag("anchor-unavailable") {
		ensureRegressionAnchor(plan).Unavailable = true
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
	resp, err := h.client.RenderMarkdown(context.Background(), connect.NewRequest(&plansv1.RenderMarkdownRequest{
		Id:        id,
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
		Compact:   ctx.BoolFlag("compact"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("render plan %q", id), err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Rendered plan %s.", id)},
		ResultsHeading: "Markdown",
		Results:        []string{resp.Msg.GetMarkdown()},
	})
}

func (h *handlers) contextList(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRelevantContext(context.Background(), connect.NewRequest(&plansv1.ListRelevantContextRequest{
		Id:        ctx.Positional("id"),
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
		PhaseId:   ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list relevant context", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetItems()))
	for _, item := range resp.Msg.GetItems() {
		results = append(results, formatRelevantContextItem(item))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Relevant context item(s): %d.", len(resp.Msg.GetItems()))},
		ResultsHeading: "Relevant context",
		Results:        results,
	})
}

func (h *handlers) contextUpdate(ctx cliapp.RunContext) error {
	item := &sharedv1.RelevantContextItem{
		Id:           ctx.Positional("item"),
		Kind:         parseContextKind(ctx.Flag("kind")),
		Label:        ctx.Flag("label"),
		Reason:       ctx.Flag("reason"),
		Instruction:  ctx.Flag("instruction"),
		Command:      ctx.Flag("command"),
		Target:       ctx.Flag("target"),
		Required:     ctx.BoolFlag("required"),
		RepeatPolicy: parseRepeatPolicy(ctx.Flag("repeat")),
		Source:       sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED,
		Status:       sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY,
	}
	if item.Kind == sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_UNSPECIFIED {
		item.Kind = inferContextKind(item)
	}
	resp, err := h.client.UpdateRelevantContext(context.Background(), connect.NewRequest(&plansv1.UpdateRelevantContextRequest{
		Id:        ctx.Positional("id"),
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
		PhaseId:   ctx.Flag("phase"),
		ItemId:    ctx.Positional("item"),
		Item:      item,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update relevant context", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Updated context on")
}

func (h *handlers) contextRemove(ctx cliapp.RunContext) error {
	resp, err := h.client.RemoveRelevantContext(context.Background(), connect.NewRequest(&plansv1.RemoveRelevantContextRequest{
		Id:        ctx.Positional("id"),
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
		PhaseId:   ctx.Flag("phase"),
		ItemId:    ctx.Positional("item"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("remove relevant context", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Removed context from")
}

func (h *handlers) referenceList(ctx cliapp.RunContext) error {
	resp, err := h.client.ListReferences(context.Background(), connect.NewRequest(&plansv1.ListReferencesRequest{
		Id:        ctx.Positional("id"),
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
		PhaseId:   ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list references", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetReferences()))
	for _, ref := range resp.Msg.GetReferences() {
		results = append(results, formatReference(ref))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Reference(s): %d.", len(resp.Msg.GetReferences()))},
		ResultsHeading: "References",
		Results:        results,
	})
}

func (h *handlers) referenceUpdate(ctx cliapp.RunContext) error {
	resp, err := h.client.UpdateReference(context.Background(), connect.NewRequest(&plansv1.UpdateReferenceRequest{
		Id:          ctx.Positional("id"),
		Workspace:   workspaceScopeFromFlag(ctx.Flag("workspace")),
		PhaseId:     ctx.Flag("phase"),
		ReferenceId: ctx.Positional("reference"),
		Reference: &sharedv1.Reference{
			Id:     ctx.Positional("reference"),
			Kind:   parseReferenceKind(ctx.Flag("kind")),
			Target: ctx.Flag("target"),
			Future: ctx.BoolFlag("future"),
			Note:   ctx.Flag("note"),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("update reference", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Updated reference on")
}

func (h *handlers) referenceRemove(ctx cliapp.RunContext) error {
	resp, err := h.client.RemoveReference(context.Background(), connect.NewRequest(&plansv1.RemoveReferenceRequest{
		Id:          ctx.Positional("id"),
		Workspace:   workspaceScopeFromFlag(ctx.Flag("workspace")),
		PhaseId:     ctx.Flag("phase"),
		ReferenceId: ctx.Positional("reference"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("remove reference", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Removed reference from")
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
		SourceIntake:           ctx.BoolFlag("source-intake"),
		RetireSources:          ctx.BoolFlag("retire-sources"),
		IncludeArchived:        ctx.BoolFlag("include-archived"),
		IncludeArchivedSources: ctx.BoolFlag("include-archived-sources"),
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
			"`plans reconcile --dry-run` — preview mirror repair and source intake",
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

func inferContextKind(item *sharedv1.RelevantContextItem) sharedv1.RelevantContextKind {
	switch {
	case strings.TrimSpace(item.GetCommand()) != "":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND
	case strings.TrimSpace(item.GetTarget()) != "":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC
	default:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE
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

func parseReferenceKind(raw string) sharedv1.ReferenceKind {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "code", "code_ref", "":
		return sharedv1.ReferenceKind_REFERENCE_KIND_CODE
	case "doc":
		return sharedv1.ReferenceKind_REFERENCE_KIND_DOC
	case "req", "requirement":
		return sharedv1.ReferenceKind_REFERENCE_KIND_REQ
	default:
		return sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED
	}
}

func (h *handlers) phaseAdd(ctx cliapp.RunContext) error {
	validationScope, err := validationScopeFromFlag(ctx.Flag("validation-scope"))
	if err != nil {
		return err
	}
	resp, err := h.client.AddPhase(context.Background(), connect.NewRequest(&plansv1.AddPhaseRequest{
		PlanId:    ctx.Positional("plan"),
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
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
			ValidationScope: validationScope,
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("add phase", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Added phase to")
}

func (h *handlers) phaseUpdate(ctx cliapp.RunContext) error {
	validationScope, err := validationScopeFromFlag(ctx.Flag("validation-scope"))
	if err != nil {
		return err
	}
	if validationScope != nil && phaseUpdateOnlyChangesValidationScope(ctx) {
		return h.phaseValidationScope(ctx)
	}
	resp, err := h.client.UpdatePhase(context.Background(), connect.NewRequest(&plansv1.UpdatePhaseRequest{
		PlanId:    ctx.Positional("plan"),
		Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")),
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
			Status:          statusconv.PhaseStatusFlag(ctx.Flag("status")),
			RelevantContext: parsePhaseContext(ctx.Flag("context")),
			Reminders:       splitCSV(ctx.Flag("reminders")),
			BaselineScope:   splitCSV(ctx.Flag("baseline-scope")),
			ValidationScope: validationScope,
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("update phase", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Updated phase on")
}

func phaseUpdateOnlyChangesValidationScope(ctx cliapp.RunContext) bool {
	for _, name := range []string{
		"title", "intent", "affected-areas", "steps", "expected-outputs", "validation",
		"acceptance", "risks-hazards", "handoff-notes", "status", "context", "reminders", "baseline-scope",
	} {
		if strings.TrimSpace(ctx.Flag(name)) != "" {
			return false
		}
	}
	return true
}

// phaseValidationScope is the safe repair lane for an existing persisted phase.
// UpdatePhase is intentionally a full replacement API, so this command reads
// the canonical phase first, changes only its validation declaration, and sends
// the complete object back. It lets older imported plans become execution-grade
// without losing references or authored context.
func (h *handlers) phaseValidationScope(ctx cliapp.RunContext) error {
	scope, err := validationScopeFromFlag(ctx.Flag("validation-scope"))
	if err != nil {
		return err
	}
	if scope == nil {
		return fmt.Errorf("--validation-scope is required")
	}
	planID := ctx.Positional("plan")
	phaseID := ctx.Positional("phase")
	current, err := h.client.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: planID, Workspace: workspaceScopeFromFlag(ctx.Flag("workspace"))}))
	if err != nil {
		return cliapp.WrapAPIError("get plan before validation-scope repair", err, nil)
	}
	if current.Msg == nil || current.Msg.Plan == nil {
		return fmt.Errorf("server returned no plan for validation-scope repair")
	}
	var phase *sharedv1.Phase
	for _, candidate := range current.Msg.Plan.GetPhases() {
		if candidate.GetId() == phaseID {
			phase = candidate
			break
		}
	}
	if phase == nil {
		return fmt.Errorf("phase %q not found on plan %q", phaseID, planID)
	}
	phase.ValidationScope = scope
	resp, err := h.client.UpdatePhase(context.Background(), connect.NewRequest(&plansv1.UpdatePhaseRequest{PlanId: planID, Workspace: workspaceScopeFromFlag(ctx.Flag("workspace")), Phase: phase}))
	if err != nil {
		return cliapp.WrapAPIError("repair phase validation scope", err, nil)
	}
	return h.renderMutation(ctx, resp.Msg.GetPlan(), "Repaired validation scope on")
}

// validationScopeFromFlag exposes the phase validation declaration on the
// persisted-plan CLI. Without it, a rendered/imported multi-scenario plan can
// become execution-ineligible after the quality rule is introduced, with no
// supported repair path. A narrow scope uses `|` so shell-safe comma-separated
// list flags remain unambiguous.
func validationScopeFromFlag(raw string) (*sharedv1.ValidationScope, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "full_plan:") {
		rationale := strings.TrimSpace(raw[len("full_plan:"):])
		if rationale == "" {
			return nil, fmt.Errorf("validation scope full_plan requires a rationale")
		}
		return &sharedv1.ValidationScope{Mode: sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_FULL_PLAN, Rationale: rationale}, nil
	}
	if strings.HasPrefix(lower, "narrow:") {
		paths := strings.Split(strings.TrimSpace(raw[len("narrow:"):]), "|")
		allow := make([]string, 0, len(paths))
		for _, path := range paths {
			if path = strings.TrimSpace(path); path != "" {
				allow = append(allow, path)
			}
		}
		if len(allow) == 0 {
			return nil, fmt.Errorf("validation scope narrow requires one or more acceptance-allow paths")
		}
		return &sharedv1.ValidationScope{Mode: sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_NARROW, Boundary: &sharedv1.ChangeBoundary{AcceptanceAllow: allow}}, nil
	}
	return nil, fmt.Errorf("validation scope must be full_plan:<rationale> or narrow:<allow-glob>|<allow-glob>")
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
	return fmt.Sprintf("%s — %s [%s, phases=%d]", p.Slug, p.Title, statusconv.PlanStatusLabel(p.Status), len(p.Phases))
}

func planDetail(p *sharedv1.Plan) []string {
	if p == nil {
		return []string{"(nil)"}
	}
	out := []string{
		fmt.Sprintf("id: %s", p.Id),
		fmt.Sprintf("slug: %s", p.Slug),
		fmt.Sprintf("title: %s", p.Title),
		fmt.Sprintf("status: %s", statusconv.PlanStatusLabel(p.Status)),
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
		out = append(out, fmt.Sprintf("unmapped import sections: %d", n))
	}
	for i, ph := range p.Phases {
		out = append(out, fmt.Sprintf("  phase %d: %s [%s] (id=%s)", i+1, ph.Title, statusconv.PlanPhaseStatusLabel(ph.Status), ph.Id))
		if len(ph.GetSteps()) > 0 {
			out = append(out, fmt.Sprintf("    steps: %d · validation: %s", len(ph.GetSteps()), truncateOneLine(ph.GetValidation(), 60)))
		}
	}
	out = append(out, "(run `plan-manager plans render "+p.Slug+"` for the full markdown review artifact)")
	return out
}

func formatRelevantContextItem(item *sharedv1.RelevantContextItem) string {
	if item == nil {
		return "(nil)"
	}
	parts := []string{firstNonEmpty(item.GetId(), "(no-id)"), relevantContextKindLabel(item.GetKind())}
	if item.GetLabel() != "" {
		parts = append(parts, truncateOneLine(item.GetLabel(), 64))
	}
	if item.GetCommand() != "" {
		parts = append(parts, "command="+truncateOneLine(item.GetCommand(), 96))
	}
	if item.GetTarget() != "" {
		parts = append(parts, "target="+item.GetTarget())
	}
	if item.GetReason() != "" {
		parts = append(parts, "reason="+truncateOneLine(item.GetReason(), 96))
	}
	return strings.Join(parts, " ")
}

func relevantContextKindLabel(kind sharedv1.RelevantContextKind) string {
	switch kind {
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SKILL:
		return "skill"
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC:
		return "doc"
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND:
		return "command"
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SEARCH:
		return "search"
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_CODE_REF:
		return "code_ref"
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_REQ_REF:
		return "req_ref"
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE:
		return "note"
	default:
		return "unknown"
	}
}

func formatReference(ref *sharedv1.Reference) string {
	if ref == nil {
		return "(nil)"
	}
	parts := []string{firstNonEmpty(ref.GetId(), "(no-id)"), referenceKindLabel(ref.GetKind()), ref.GetTarget()}
	if ref.GetFuture() {
		parts = append(parts, "future")
	}
	if ref.GetNote() != "" {
		parts = append(parts, "note="+truncateOneLine(ref.GetNote(), 96))
	}
	return strings.Join(parts, " ")
}

func referenceKindLabel(kind sharedv1.ReferenceKind) string {
	switch kind {
	case sharedv1.ReferenceKind_REFERENCE_KIND_CODE:
		return "CODE"
	case sharedv1.ReferenceKind_REFERENCE_KIND_DOC:
		return "DOC"
	case sharedv1.ReferenceKind_REFERENCE_KIND_REQ:
		return "REQ"
	default:
		return "UNKNOWN"
	}
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
	if item.GetSourceRetirementPlanned() {
		parts = append(parts, "source retirement planned")
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

func applyStringFlag(ctx cliapp.RunContext, name string, apply func(string)) {
	values := ctx.FlagValues(name)
	if len(values) == 0 {
		return
	}
	apply(values[len(values)-1])
}

func splitPlanFlagValues(values []string) []string {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func ensureChangeBoundary(plan *sharedv1.Plan) *sharedv1.ChangeBoundary {
	if plan.ChangeBoundary == nil {
		plan.ChangeBoundary = &sharedv1.ChangeBoundary{}
	}
	return plan.ChangeBoundary
}

func ensureRegressionAnchor(plan *sharedv1.Plan) *sharedv1.RegressionAnchor {
	if plan.RegressionAnchor == nil {
		plan.RegressionAnchor = &sharedv1.RegressionAnchor{}
	}
	return plan.RegressionAnchor
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
