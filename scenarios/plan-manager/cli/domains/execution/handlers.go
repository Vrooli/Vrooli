package execution

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	executionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution/execution_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client executionconnect.ExecutionServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: executionconnect.NewExecutionServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	resp, err := h.client.Start(context.Background(), connect.NewRequest(&executionv1.StartRequest{
		PlanId: ctx.Positional("plan"), RunId: ctx.Flag("run-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("start execution", err, nil)
	}
	e := resp.Msg.GetExecution()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Started execution %s on plan %s.", e.GetId(), e.GetPlanId())},
		Changes: []string{
			fmt.Sprintf("current phase: %s", orNone(e.GetCurrentPhaseId())),
			fmt.Sprintf("run id: %s", orNone(e.GetRunId())),
		},
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.GetStatus(context.Background(), connect.NewRequest(&executionv1.GetStatusRequest{
		ExecutionId: ctx.Positional("execution"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get status", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        contextSummary(resp.Msg.GetExecution(), resp.Msg.GetContext()),
		ResultsHeading: "Phase context",
		Results:        append(contextLines(resp.Msg.GetContext()), formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) context(ctx cliapp.RunContext) error {
	resp, err := h.client.GetContext(context.Background(), connect.NewRequest(&executionv1.GetContextRequest{
		ExecutionId: ctx.Positional("execution"),
		PhaseId:     ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get context", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        contextSummary(resp.Msg.GetExecution(), resp.Msg.GetContext()),
		ResultsHeading: "Setup context",
		Results:        append(contextLines(resp.Msg.GetContext()), formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) resume(ctx cliapp.RunContext) error {
	resp, err := h.client.Resume(context.Background(), connect.NewRequest(&executionv1.ResumeRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		RunId:           ctx.Flag("run-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("resume execution", err, nil)
	}
	e := resp.Msg.GetExecution()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Resumed execution %s on plan %s.", e.GetId(), e.GetPlanId()),
			fmt.Sprintf("Current phase: %s.", orNone(e.GetCurrentPhaseId())),
		},
		ResultsHeading: "Setup context",
		Results:        append(contextLines(resp.Msg.GetContext()), formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) next(ctx cliapp.RunContext) error {
	resp, err := h.client.GetNext(context.Background(), connect.NewRequest(&executionv1.GetNextRequest{
		ExecutionId: ctx.Positional("execution"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get next", err, nil)
	}
	summary := "Advanced to the next actionable phase."
	if resp.Msg.GetComplete() {
		summary = "No actionable phase remains — the run is complete."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Phase context",
		Results:        append(contextLines(resp.Msg.GetContext()), formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) transition(ctx cliapp.RunContext) error {
	resp, err := h.client.TransitionPhase(context.Background(), connect.NewRequest(&executionv1.TransitionPhaseRequest{
		ExecutionId: ctx.Positional("execution"),
		PhaseId:     ctx.Positional("phase"),
		ToStatus:    phaseStatusFlag(ctx.Flag("status")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("transition phase", err, nil)
	}
	e := resp.Msg.GetExecution()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Transitioned phase %s to %s.", ctx.Positional("phase"), phaseStatusLabel(phaseStatusFlag(ctx.Flag("status"))))},
		Changes: append([]string{
			fmt.Sprintf("plan status: %s", planStatusLabel(resp.Msg.GetPlan().GetStatus())),
			fmt.Sprintf("current phase: %s", orNone(e.GetCurrentPhaseId())),
		}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) decisionAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.RecordDecision(context.Background(), connect.NewRequest(&executionv1.RecordDecisionRequest{
		ExecutionId: ctx.Positional("execution"),
		PhaseId:     ctx.Flag("phase"),
		Summary:     ctx.Flag("summary"),
		Detail:      ctx.Flag("detail"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("record decision", err, nil)
	}
	d := resp.Msg.GetDecision()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Recorded decision %s.", d.GetId())},
		Changes:     append([]string{d.GetSummary()}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) findingAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.RecordFinding(context.Background(), connect.NewRequest(&executionv1.RecordFindingRequest{
		ExecutionId: ctx.Positional("execution"),
		PhaseId:     ctx.Flag("phase"),
		Title:       ctx.Flag("title"),
		Detail:      ctx.Flag("detail"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("record finding", err, nil)
	}
	f := resp.Msg.GetFinding()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Filed candidate finding %s (triage %s).", f.GetId(), triageLabel(f.GetTriage()))},
		Changes:     append([]string{f.GetTitle()}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) complete(ctx cliapp.RunContext) error {
	tokens, err := parseInt64Flag(ctx.Flag("tokens"))
	if err != nil {
		return cliapp.WrapAPIError("complete", fmt.Errorf("invalid --tokens: %w", err), nil)
	}
	iterations, err := parseInt32Flag(ctx.Flag("iterations"))
	if err != nil {
		return cliapp.WrapAPIError("complete", fmt.Errorf("invalid --iterations: %w", err), nil)
	}
	resp, err := h.client.Complete(context.Background(), connect.NewRequest(&executionv1.CompleteRequest{
		ExecutionId: ctx.Positional("execution"),
		Tokens:      tokens,
		Iterations:  iterations,
	}))
	if err != nil {
		return cliapp.WrapAPIError("complete", err, nil)
	}
	ho := resp.Msg.GetHandoff()
	results := make([]string, 0, len(resp.Msg.GetNudges()))
	for _, n := range resp.Msg.GetNudges() {
		mark := "needs attention"
		if n.GetSatisfied() {
			mark = "satisfied"
		}
		results = append(results, fmt.Sprintf("[%s] %s — %s", n.GetKind(), mark, n.GetMessage()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Completed execution %s (completeness %s).", ho.GetExecutionId(), completenessLabel(ho.GetCompleteness())),
			fmt.Sprintf("Resume point: %s.", orNone(ho.GetResumePhaseId())),
		},
		ResultsHeading: "Completion nudges",
		Results:        append(results, formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) handoff(ctx cliapp.RunContext) error {
	resp, err := h.client.GetHandoff(context.Background(), connect.NewRequest(&executionv1.GetHandoffRequest{
		ExecutionId: ctx.Positional("execution"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get handoff", err, nil)
	}
	ho := resp.Msg.GetHandoff()
	results := make([]string, 0)
	for _, d := range ho.GetDecisions() {
		results = append(results, fmt.Sprintf("decision: %s", d.GetSummary()))
	}
	for _, f := range ho.GetCandidateFindings() {
		results = append(results, fmt.Sprintf("candidate finding: %s", f.GetTitle()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Handoff for execution %s (completeness %s, staleness %s).", ho.GetExecutionId(), completenessLabel(ho.GetCompleteness()), stalenessLabel(ho.GetStaleness())),
			fmt.Sprintf("Resume point: %s.", orNone(ho.GetResumePhaseId())),
		},
		ResultsHeading: "Captured state",
		Results:        append(results, formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) findings(ctx cliapp.RunContext) error {
	resp, err := h.client.ListCandidateFindings(context.Background(), connect.NewRequest(&executionv1.ListCandidateFindingsRequest{
		ExecutionId: ctx.Flag("exec"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list candidate findings", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetFindings()))
	for _, f := range resp.Msg.GetFindings() {
		results = append(results, fmt.Sprintf("%s — %s (triage %s)", f.GetId(), f.GetTitle(), triageLabel(f.GetTriage())))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d candidate finding(s) awaiting triage.", len(resp.Msg.GetFindings()))},
		ResultsHeading: "Candidate findings",
		Results:        results,
		RetrievalHints: append(formatRecommendedActions(resp.Msg.GetStep()), formatStep(resp.Msg.GetStep())...),
	})
}

func (h *handlers) triage(ctx cliapp.RunContext) error {
	resp, err := h.client.TriageFinding(context.Background(), connect.NewRequest(&executionv1.TriageFindingRequest{
		FindingId: ctx.Positional("finding"),
		Triage:    triageFlag(ctx.Flag("status")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("triage finding", err, nil)
	}
	f := resp.Msg.GetFinding()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Finding %s triaged to %s.", f.GetId(), triageLabel(f.GetTriage()))},
		Changes:     append([]string{f.GetTitle()}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) velocity(ctx cliapp.RunContext) error {
	resp, err := h.client.GetVelocity(context.Background(), connect.NewRequest(&executionv1.GetVelocityRequest{
		PlanId: ctx.Positional("plan"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get velocity", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetPoints()))
	for _, p := range resp.Msg.GetPoints() {
		results = append(results, fmt.Sprintf("%s — %ds wall, %d tokens, %d iterations (%s)",
			p.GetRecordedAt(), p.GetWallTimeSeconds(), p.GetTokens(), p.GetIterations(), completenessLabel(p.GetCompleteness())))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d velocity point(s) for plan %s.", len(resp.Msg.GetPoints()), ctx.Positional("plan"))},
		ResultsHeading: "Velocity series",
		Results:        results,
	})
}

// --- rendering + parsing helpers ---

func contextSummary(e *executionv1.Execution, c *executionv1.PhaseContext) []string {
	out := []string{
		fmt.Sprintf("Execution %s on plan %s (completeness %s).", e.GetId(), e.GetPlanId(), completenessLabel(c.GetCompleteness())),
		fmt.Sprintf("Resume point: %s; staleness: %s.", orNone(c.GetResumePhaseId()), stalenessLabel(c.GetStaleness())),
	}
	if cur := c.GetCurrentPhase(); cur != nil {
		out = append(out, fmt.Sprintf("Current phase: %s (%s).", cur.GetTitle(), phaseStatusLabel(cur.GetStatus())))
	}
	return out
}

func contextLines(c *executionv1.PhaseContext) []string {
	out := make([]string, 0)
	for _, item := range c.GetRelevantContext() {
		out = append(out, formatRelevantContext(item)...)
	}
	for _, r := range c.GetRequiredReading() {
		out = append(out, "read: "+r)
	}
	for _, r := range c.GetReminders() {
		out = append(out, "reminder: "+r)
	}
	if next := c.GetNextPhase(); next != nil {
		out = append(out, "next phase: "+next.GetTitle())
	}
	if lv := c.GetLastValidation(); lv != nil {
		out = append(out, fmt.Sprintf("last validation: %s", lv.GetDetail()))
	}
	return out
}

func formatRelevantContext(item *sharedv1.RelevantContextItem) []string {
	if item == nil {
		return nil
	}
	label := firstNonEmpty(item.GetLabel(), item.GetTarget(), item.GetCommand(), item.GetInstruction(), "context")
	line := fmt.Sprintf("context[%s]: %s", relevantContextKindLabel(item.GetKind()), label)
	if item.GetRequired() {
		line += " (required)"
	}
	out := []string{line}
	if item.GetReason() != "" {
		out = append(out, "  reason: "+item.GetReason())
	}
	if item.GetInstruction() != "" && item.GetInstruction() != label {
		out = append(out, "  instruction: "+item.GetInstruction())
	}
	if cmd := relevantContextCommand(item); cmd != "" {
		out = append(out, "  command: "+cmd)
	}
	return out
}

func relevantContextCommand(item *sharedv1.RelevantContextItem) string {
	if item.GetCommand() != "" {
		return item.GetCommand()
	}
	if len(item.GetArgv()) > 0 {
		return shellCommand(item.GetArgv())
	}
	return ""
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
		return "unspecified"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatStep(step *sharedv1.GuidedStep) []string {
	if step == nil || strings.TrimSpace(step.GetStepKind()) == "" {
		return nil
	}
	out := []string{fmt.Sprintf("Current Step (%s): %s", step.GetStepKind(), step.GetSummary())}
	for _, input := range step.GetRequiredInputs() {
		out = append(out, "Required input: "+input)
	}
	for _, item := range step.GetInstructions() {
		out = append(out, "- "+item)
	}
	for _, action := range step.GetNextActions() {
		if action.GetKind() == sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED {
			continue
		}
		out = append(out, fmt.Sprintf("%s: `%s` — %s", actionKindLabel(action.GetKind()), shellCommand(action.GetArgv()), action.GetReason()))
	}
	return out
}

func formatRecommendedActions(step *sharedv1.GuidedStep) []string {
	if step == nil {
		return nil
	}
	out := make([]string, 0, 1)
	for _, action := range step.GetNextActions() {
		if action.GetKind() != sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED {
			continue
		}
		out = append(out, fmt.Sprintf("`%s` — %s", shellCommand(action.GetArgv()), action.GetReason()))
	}
	return out
}

func actionKindLabel(kind sharedv1.NextActionKind) string {
	switch kind {
	case sharedv1.NextActionKind_NEXT_ACTION_KIND_ALTERNATIVE:
		return "Alternative"
	case sharedv1.NextActionKind_NEXT_ACTION_KIND_OPTIONAL:
		return "Optional"
	case sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOVERY:
		return "Recovery"
	default:
		return "Action"
	}
}

func shellCommand(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '<' || r == '>' || r == '[' || r == ']' || r == ':' || r == ';' || r == '|' || r == '&'
	}) < 0 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

func parseInt64Flag(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

func parseInt32Flag(s string) (int32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
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

func triageFlag(s string) sharedv1.FindingTriage {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "candidate":
		return sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE
	case "promoted", "promote":
		return sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED
	case "dismissed", "dismiss":
		return sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED
	default:
		return sharedv1.FindingTriage_FINDING_TRIAGE_UNSPECIFIED
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
		return "unspecified"
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

func completenessLabel(c sharedv1.Completeness) string {
	switch c {
	case sharedv1.Completeness_COMPLETENESS_FULL:
		return "full"
	case sharedv1.Completeness_COMPLETENESS_PARTIAL:
		return "partial"
	default:
		return "unspecified"
	}
}

func triageLabel(t sharedv1.FindingTriage) string {
	switch t {
	case sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE:
		return "candidate"
	case sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED:
		return "promoted"
	case sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED:
		return "dismissed"
	default:
		return "unspecified"
	}
}

func stalenessLabel(s sharedv1.StalenessTier) string {
	switch s {
	case sharedv1.StalenessTier_STALENESS_TIER_FRESH:
		return "fresh"
	case sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE:
		return "lightly_stale"
	case sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE:
		return "definitely_stale"
	default:
		return "unknown"
	}
}
