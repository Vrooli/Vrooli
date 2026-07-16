package execution

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"plan-manager/cli/internal/statusconv"
	"plan-manager/cli/internal/steprender"

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

func (h *handlers) continueExecution(ctx cliapp.RunContext) error {
	resp, err := h.client.ContinueExecution(context.Background(), connect.NewRequest(&executionv1.ContinueExecutionRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		RunId:           ctx.Flag("run-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("continue execution", err, nil)
	}
	e := resp.Msg.GetExecution()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Continuing execution %s on plan %s.", e.GetId(), e.GetPlanId()),
			fmt.Sprintf("Current phase: %s.", orNone(e.GetCurrentPhaseId())),
		},
		ResultsHeading: "Recommended action",
		Results:        append(contextLines(resp.Msg.GetContext()), formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) abandon(ctx cliapp.RunContext) error {
	resp, err := h.client.AbandonExecution(context.Background(), connect.NewRequest(&executionv1.AbandonExecutionRequest{
		ExecutionId: ctx.Positional("execution"), Reason: ctx.Flag("reason"), Actor: ctx.Flag("actor"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("abandon execution", err, nil)
	}
	e := resp.Msg.GetExecution()
	result := fmt.Sprintf("Abandoned execution %s.", e.GetId())
	if resp.Msg.GetAlreadyAbandoned() {
		result = fmt.Sprintf("Execution %s was already abandoned; existing terminal record retained.", e.GetId())
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{result},
		Changes: []string{
			"lifecycle state: " + e.GetLifecycleState(),
			"reason: " + e.GetAbandonedReason(),
			"actor: " + e.GetAbandonedBy(),
			"abandoned at: " + e.GetAbandonedAt(),
		},
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) syncBaseline(ctx cliapp.RunContext) error {
	resp, err := h.client.SyncBaseline(context.Background(), connect.NewRequest(&executionv1.SyncBaselineRequest{ExecutionId: ctx.Positional("execution")}))
	if err != nil {
		return cliapp.WrapAPIError("synchronize baseline", err, nil)
	}
	state := resp.Msg.GetExecution().GetBaselineSet()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Baseline %s synchronized as %s.", state.GetName(), state.GetStatus())},
		ResultsHeading: "Next action", Results: formatStep(resp.Msg.GetStep()), RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) amendScope(ctx cliapp.RunContext) error {
	resp, err := h.client.AmendScope(context.Background(), connect.NewRequest(&executionv1.AmendScopeRequest{ExecutionId: ctx.Positional("execution"), PhaseId: ctx.Flag("phase"), Member: commaSeparated(ctx.Flag("members")), Author: ctx.Flag("author"), Reason: ctx.Flag("reason")}))
	if err != nil {
		return cliapp.WrapAPIError("amend scope", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Recorded scope amendment and invalidated prior phase evidence."}, Changes: formatStep(resp.Msg.GetStep()), NextCommand: formatRecommendedActions(resp.Msg.GetStep())})
}

func (h *handlers) adoptBaseline(ctx cliapp.RunContext) error {
	resp, err := h.client.AdoptBaseline(context.Background(), connect.NewRequest(&executionv1.AdoptBaselineRequest{ExecutionId: ctx.Positional("execution"), Mode: ctx.Flag("mode"), Name: ctx.Flag("name"), Member: commaSeparated(ctx.Flag("members")), Path: commaSeparated(ctx.Flag("paths")), Reason: ctx.Flag("reason")}))
	if err != nil {
		return cliapp.WrapAPIError("adopt baseline", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Recorded explicit legacy baseline adoption state."}, Changes: formatStep(resp.Msg.GetStep()), NextCommand: formatRecommendedActions(resp.Msg.GetStep())})
}

func (h *handlers) repairSourceScope(ctx cliapp.RunContext) error {
	resp, err := h.client.RepairSourceScope(context.Background(), connect.NewRequest(&executionv1.RepairSourceScopeRequest{ExecutionId: ctx.Positional("execution"), Path: commaSeparated(ctx.Flag("paths")), Reason: ctx.Flag("reason")}))
	if err != nil {
		return cliapp.WrapAPIError("repair baseline source scope", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Re-estimated replacement source evidence scope."}, Changes: formatStep(resp.Msg.GetStep()), NextCommand: formatRecommendedActions(resp.Msg.GetStep())})
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
		ToStatus:    statusconv.PhaseStatusFlag(ctx.Flag("status")),
		ValidationOverride: &executionv1.ValidationOverride{
			Reason: ctx.Flag("validation-override-reason"),
		},
		FeedbackOverride: &executionv1.FeedbackOverride{
			Reason: ctx.Flag("feedback-override-reason"),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("transition phase", err, nil)
	}
	e := resp.Msg.GetExecution()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Transitioned phase %s to %s.", ctx.Positional("phase"), statusconv.PhaseStatusLabel(statusconv.PhaseStatusFlag(ctx.Flag("status"))))},
		Changes: append([]string{
			fmt.Sprintf("plan status: %s", statusconv.PlanStatusLabel(resp.Msg.GetPlan().GetStatus())),
			fmt.Sprintf("current phase: %s", orNone(e.GetCurrentPhaseId())),
		}, formatStep(resp.Msg.GetStep())...),
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

func (h *handlers) partialHandoff(ctx cliapp.RunContext) error {
	tokens, err := parseInt64Flag(ctx.Flag("tokens"))
	if err != nil {
		return cliapp.WrapAPIError("partial handoff", err, nil)
	}
	iterations, err := parseInt32Flag(ctx.Flag("iterations"))
	if err != nil {
		return cliapp.WrapAPIError("partial handoff", err, nil)
	}
	resp, err := h.client.PartialHandoff(context.Background(), connect.NewRequest(&executionv1.PartialHandoffRequest{ExecutionId: ctx.Positional("execution"), Tokens: tokens, Iterations: iterations}))
	if err != nil {
		return cliapp.WrapAPIError("partial handoff", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded partial handoff for execution %s; it remains incomplete.", resp.Msg.GetHandoff().GetExecutionId())}, Changes: formatStep(resp.Msg.GetStep()), NextCommand: formatRecommendedActions(resp.Msg.GetStep())})
}

func commaSeparated(raw string) []string {
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
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
	for _, e := range ho.GetLogEntries() {
		line := fmt.Sprintf("%s: %s", logEntryTypeLabel(e.GetType()), e.GetTitle())
		if e.GetType() == sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT || e.GetType() == sharedv1.LogEntryType_LOG_ENTRY_TYPE_RECORD {
			line += fmt.Sprintf(" (sync %s)", logSyncStatusLabel(e.GetSyncStatus()))
		}
		results = append(results, line)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: append([]string{
			fmt.Sprintf("Handoff for execution %s (completeness %s, staleness %s).", ho.GetExecutionId(), completenessLabel(ho.GetCompleteness()), stalenessLabel(ho.GetStaleness())),
			fmt.Sprintf("Resume point: %s.", orNone(ho.GetResumePhaseId())),
		}, logSummaryLines(ho.GetLogSummary())...),
		ResultsHeading: "Captured log entries",
		Results:        append(results, formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
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
		out = append(out, fmt.Sprintf("Current phase: %s (%s).", cur.GetTitle(), statusconv.PhaseStatusLabel(cur.GetStatus())))
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
	if cp := c.GetFeedbackCheckpoint(); cp != nil && cp.GetPhaseId() != "" {
		out = append(out, fmt.Sprintf("feedback checkpoint: %s", cp.GetSummary()))
		out = append(out, fmt.Sprintf("  phase feedback: %d decisions, %d findings, %d bugs, %d records, %d notes",
			cp.GetDecisions(), cp.GetFindings(), cp.GetBugReports(), cp.GetRecords(), cp.GetNotes()))
		if cp.GetPendingSync()+cp.GetFailedSync() > 0 {
			out = append(out, fmt.Sprintf("  feedback sync: %d pending, %d failed", cp.GetPendingSync(), cp.GetFailedSync()))
		}
	}
	out = append(out, logSummaryLines(c.GetLogSummary())...)
	return out
}

// logSummaryLines renders the compact plan-log roll-up so a resumed agent sees
// decisions/findings/bugs/records and any degraded downstream sync at a glance.
func logSummaryLines(s *sharedv1.LogSummary) []string {
	if s == nil || s.GetTotal() == 0 {
		return nil
	}
	out := []string{fmt.Sprintf("log ledger: %d entries (%d decisions, %d findings, %d bugs, %d records, %d notes)",
		s.GetTotal(), s.GetDecisions(), s.GetFindings(), s.GetBugReports(), s.GetRecords(), s.GetNotes())}
	if s.GetCandidateFindings() > 0 {
		out = append(out, fmt.Sprintf("  %d candidate finding(s) awaiting triage/promotion", s.GetCandidateFindings()))
	}
	if s.GetPendingSync()+s.GetFailedSync() > 0 {
		out = append(out, fmt.Sprintf("  downstream sync: %d pending, %d failed — retry with `plan-manager log sync <id>`", s.GetPendingSync(), s.GetFailedSync()))
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
		return steprender.ShellCommand(item.GetArgv())
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
	return steprender.StepLines(step)
}

func formatRecommendedActions(step *sharedv1.GuidedStep) []string {
	return steprender.RecommendedActions(step)
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

func logEntryTypeLabel(t sharedv1.LogEntryType) string {
	switch t {
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_DECISION:
		return "decision"
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_FINDING:
		return "finding"
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT:
		return "bug_report"
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_RECORD:
		return "record"
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_NOTE:
		return "note"
	default:
		return "unspecified"
	}
}

func logSyncStatusLabel(s sharedv1.LogSyncStatus) string {
	switch s {
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL:
		return "local"
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_PENDING:
		return "pending"
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_SYNCED:
		return "synced"
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_FAILED:
		return "sync_failed"
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
