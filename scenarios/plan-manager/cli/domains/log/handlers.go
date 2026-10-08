package log

import (
	"context"
	"fmt"
	"strings"

	"plan-manager/cli/internal/steprender"

	"connectrpc.com/connect"

	logv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log"
	logconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log/log_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client logconnect.LogServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: logconnect.NewLogServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) decisionAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddDecision(context.Background(), connect.NewRequest(&logv1.AddDecisionRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		Title:           ctx.Flag("title"),
		Detail:          ctx.Flag("detail"),
		Evidence:        evidenceFlags(ctx, "evidence", "evidence-item"),
		SourceCommand:   ctx.Flag("source-command"),
		IdempotencyKey:  ctx.Flag("idempotency-key"),
		RunId:           ctx.Flag("run-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("record decision", err, nil)
	}
	return renderAdd(ctx, resp.Msg)
}

func (h *handlers) findingAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddFinding(context.Background(), connect.NewRequest(&logv1.AddFindingRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		Title:           ctx.Flag("title"),
		Detail:          ctx.Flag("detail"),
		Severity:        severityFlag(ctx.Flag("severity")),
		Evidence:        evidenceFlags(ctx, "evidence", "evidence-item"),
		SourceCommand:   ctx.Flag("source-command"),
		IdempotencyKey:  ctx.Flag("idempotency-key"),
		RunId:           ctx.Flag("run-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("record finding", err, nil)
	}
	return renderAdd(ctx, resp.Msg)
}

func (h *handlers) bugAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddBug(context.Background(), connect.NewRequest(&logv1.AddBugRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		Title:           ctx.Flag("title"),
		Detail:          ctx.Flag("detail"),
		Severity:        severityFlag(ctx.Flag("severity")),
		Evidence:        evidenceFlags(ctx, "evidence", "evidence-item"),
		SourceCommand:   ctx.Flag("source-command"),
		IdempotencyKey:  ctx.Flag("idempotency-key"),
		RunId:           ctx.Flag("run-id"),
		SignalType:      ctx.Flag("signal-type"),
		ReportSeverity:  ctx.Flag("report-severity"),
		Repro:           splitCSV(ctx.Flag("repro")),
		Expected:        ctx.Flag("expected"),
		Actual:          ctx.Flag("actual"),
		Description:     ctx.Flag("description"),
		HonestyFlags:    splitCSV(ctx.Flag("honesty-flags")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("file bug", err, nil)
	}
	return renderAdd(ctx, resp.Msg)
}

func (h *handlers) recordAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddRecord(context.Background(), connect.NewRequest(&logv1.AddRecordRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		Title:           ctx.Flag("title"),
		Detail:          ctx.Flag("detail"),
		Evidence:        evidenceFlags(ctx, "evidence", "evidence-item"),
		SourceCommand:   ctx.Flag("source-command"),
		IdempotencyKey:  ctx.Flag("idempotency-key"),
		RunId:           ctx.Flag("run-id"),
		Kind:            ctx.Flag("kind"),
		Scenario:        ctx.Flag("scenario"),
		Trigger:         ctx.Flag("trigger"),
		Approach:        ctx.Flag("approach"),
		RecordEvidence:  ctx.Flag("record-evidence"),
		Outcome:         ctx.Flag("outcome"),
		CreatedBy:       ctx.Flag("created-by"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("capture record", err, nil)
	}
	return renderAdd(ctx, resp.Msg)
}

func (h *handlers) noteAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddNote(context.Background(), connect.NewRequest(&logv1.AddNoteRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		Title:           ctx.Flag("title"),
		Detail:          ctx.Flag("detail"),
		Evidence:        evidenceFlags(ctx, "evidence", "evidence-item"),
		SourceCommand:   ctx.Flag("source-command"),
		IdempotencyKey:  ctx.Flag("idempotency-key"),
		RunId:           ctx.Flag("run-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("record note", err, nil)
	}
	return renderAdd(ctx, resp.Msg)
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListEntries(context.Background(), connect.NewRequest(&logv1.ListEntriesRequest{
		PlanOrExecution: ctx.Positional("plan-or-execution"),
		PhaseId:         ctx.Flag("phase"),
		Type:            entryTypeFlag(ctx.Flag("type")),
		Triage:          triageFlag(ctx.Flag("triage")),
		SyncStatus:      syncStatusFlag(ctx.Flag("sync-status")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list log entries", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetEntries()))
	for _, e := range resp.Msg.GetEntries() {
		results = append(results, entryLine(e))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summaryLines(resp.Msg.GetSummary()),
		ResultsHeading: "Log entries",
		Results:        append(results, formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetEntry(context.Background(), connect.NewRequest(&logv1.GetEntryRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get log entry", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{entryLine(resp.Msg.GetEntry())},
		ResultsHeading: "Entry detail",
		Results:        append(entryDetail(resp.Msg.GetEntry()), formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	resp, err := h.client.UpdateEntry(context.Background(), connect.NewRequest(&logv1.UpdateEntryRequest{
		Id:          ctx.Positional("id"),
		Title:       ctx.Flag("title"),
		Detail:      ctx.Flag("detail"),
		Severity:    severityFlag(ctx.Flag("severity")),
		Triage:      triageFlag(ctx.Flag("triage")),
		AddEvidence: evidenceFlags(ctx, "add-evidence", "add-evidence-item"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("update log entry", err, nil)
	}
	e := resp.Msg.GetEntry()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated %s entry %s.", entryTypeLabel(e.GetType()), e.GetId())},
		Changes:     append([]string{entryLine(e)}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) reassign(ctx cliapp.RunContext) error {
	resp, err := h.client.ReassignEntry(context.Background(), connect.NewRequest(&logv1.ReassignEntryRequest{
		Id:      ctx.Positional("id"),
		PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("reassign log entry", err, nil)
	}
	e := resp.Msg.GetEntry()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Reassigned %s entry %s.", entryTypeLabel(e.GetType()), e.GetId())},
		Changes: append([]string{
			entryLine(e),
			scopeLine(e),
		}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) promote(ctx cliapp.RunContext) error {
	resp, err := h.client.PromoteEntry(context.Background(), connect.NewRequest(&logv1.PromoteEntryRequest{
		Id:       ctx.Positional("id"),
		ToType:   entryTypeFlag(ctx.Flag("to")),
		Title:    ctx.Flag("title"),
		Detail:   ctx.Flag("detail"),
		Severity: severityFlag(ctx.Flag("severity")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("promote log entry", err, nil)
	}
	e := resp.Msg.GetEntry()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Promoted finding %s to %s entry %s.", resp.Msg.GetSource().GetId(), entryTypeLabel(e.GetType()), e.GetId())},
		Changes: append([]string{
			entryLine(e),
			fmt.Sprintf("downstream sync: %s", syncStatusLabel(e.GetSyncStatus())),
		}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) sync(ctx cliapp.RunContext) error {
	resp, err := h.client.SyncEntry(context.Background(), connect.NewRequest(&logv1.SyncEntryRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("sync log entry", err, nil)
	}
	e := resp.Msg.GetEntry()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Sync for entry %s is now %s.", e.GetId(), syncStatusLabel(e.GetSyncStatus()))},
		Changes: append([]string{
			downstreamLine(e.GetDownstream()),
		}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

// --- rendering helpers ---

func renderAdd(ctx cliapp.RunContext, msg *logv1.AddEntryResponse) error {
	e := msg.GetEntry()
	result := fmt.Sprintf("Recorded %s entry %s.", entryTypeLabel(e.GetType()), e.GetId())
	if msg.GetDeduplicated() {
		result = fmt.Sprintf("Returned existing %s entry %s (deduplicated).", entryTypeLabel(e.GetType()), e.GetId())
	}
	changes := []string{entryLine(e), scopeLine(e)}
	if e.GetSyncStatus() != sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL && e.GetSyncStatus() != sharedv1.LogSyncStatus_LOG_SYNC_STATUS_UNSPECIFIED {
		changes = append(changes, fmt.Sprintf("downstream sync: %s", syncStatusLabel(e.GetSyncStatus())))
	}
	changes = append(changes, captureLines(e.GetCapture())...)
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:      []string{result},
		Changes:     append(changes, formatStep(msg.GetStep())...),
		NextCommand: formatRecommendedActions(msg.GetStep()),
	})
}

func summaryLines(s *sharedv1.LogSummary) []string {
	if s == nil {
		return nil
	}
	out := []string{fmt.Sprintf("%d entries: %d decisions, %d findings, %d bug reports, %d records, %d notes.",
		s.GetTotal(), s.GetDecisions(), s.GetFindings(), s.GetBugReports(), s.GetRecords(), s.GetNotes())}
	if s.GetPendingSync()+s.GetFailedSync() > 0 {
		out = append(out, fmt.Sprintf("downstream sync: %d pending, %d failed.", s.GetPendingSync(), s.GetFailedSync()))
	}
	return out
}

func entryLine(e *sharedv1.LogEntry) string {
	if e == nil {
		return ""
	}
	line := fmt.Sprintf("%s [%s] %s", e.GetId(), entryTypeLabel(e.GetType()), e.GetTitle())
	if e.GetType() == sharedv1.LogEntryType_LOG_ENTRY_TYPE_FINDING {
		line += fmt.Sprintf(" (triage %s)", triageLabel(e.GetTriage()))
	}
	if e.GetType() == sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT || e.GetType() == sharedv1.LogEntryType_LOG_ENTRY_TYPE_RECORD {
		line += fmt.Sprintf(" (sync %s)", syncStatusLabel(e.GetSyncStatus()))
	}
	return line
}

func scopeLine(e *sharedv1.LogEntry) string {
	if e == nil {
		return ""
	}
	parts := []string{"scope: plan " + orNone(e.GetPlanId())}
	if e.GetExecutionId() != "" {
		parts = append(parts, "execution "+e.GetExecutionId())
	}
	if e.GetPhaseId() != "" {
		parts = append(parts, "phase "+e.GetPhaseId())
	} else {
		parts = append(parts, "phase (plan-wide)")
	}
	return strings.Join(parts, ", ")
}

func entryDetail(e *sharedv1.LogEntry) []string {
	if e == nil {
		return nil
	}
	out := make([]string, 0, 6)
	out = append(out, scopeLine(e))
	if e.GetDetail() != "" {
		out = append(out, "detail: "+e.GetDetail())
	}
	for _, ev := range e.GetEvidence() {
		out = append(out, "evidence: "+ev)
	}
	if e.GetPromotedFromId() != "" {
		out = append(out, "promoted from finding: "+e.GetPromotedFromId())
	}
	if d := e.GetDownstream(); d != nil && (d.GetReference() != "" || d.GetDetail() != "") {
		out = append(out, downstreamLine(d))
	}
	out = append(out, captureLines(e.GetCapture())...)
	return out
}

func downstreamLine(d *sharedv1.DownstreamRef) string {
	if d == nil {
		return "downstream: (none)"
	}
	ref := d.GetReference()
	if ref == "" {
		ref = "(pending)"
	}
	line := fmt.Sprintf("downstream %s/%s: %s", orNone(d.GetSystem()), orNone(d.GetKind()), ref)
	if d.GetDetail() != "" {
		line += " — " + d.GetDetail()
	}
	return line
}

func captureLines(c *sharedv1.CaptureDisposition) []string {
	if c == nil || c.GetState() == "" {
		return nil
	}
	lines := []string{"capture disposition: " + c.GetState()}
	if c.GetDraftId() != "" {
		lines = append(lines, "private draft: "+c.GetDraftId())
	}
	if len(c.GetNeeds()) > 0 {
		lines = append(lines, "needs: "+strings.Join(c.GetNeeds(), ", "))
	}
	for _, invalid := range c.GetInvalid() {
		lines = append(lines, "invalid "+invalid.GetField()+": "+invalid.GetMessage())
	}
	if len(c.GetNextAction()) > 0 {
		lines = append(lines, "repair: "+strings.Join(c.GetNextAction(), " "))
	}
	return lines
}

// evidenceFlags merges the comma-separated evidence flag with its repeatable
// per-item variant.
//
// The CSV flag splits on every comma with no quoting or escaping, so any
// locator containing a comma is silently torn into two entries. The phase
// commands already solved this by pairing a plural CSV flag with a repeatable
// singular one; evidence simply never got the singular. Values from the
// repeatable flag are preserved verbatim.
func evidenceFlags(ctx cliapp.RunContext, csvFlag, itemFlag string) []string {
	var out []string
	if ctx.FlagDeclared(itemFlag) {
		for _, value := range ctx.FlagValues(itemFlag) {
			if strings.TrimSpace(value) != "" {
				out = append(out, strings.TrimSpace(value))
			}
		}
	}
	if ctx.FlagDeclared(csvFlag) && ctx.FlagProvided(csvFlag) {
		out = append(out, splitCSV(ctx.Flag(csvFlag))...)
	}
	return out
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
}

// --- step rendering (mirrors the per-domain CLI steering shape) ---

func formatStep(step *sharedv1.GuidedStep) []string {
	return steprender.StepLines(step)
}

func formatRecommendedActions(step *sharedv1.GuidedStep) []string {
	return steprender.RecommendedActions(step)
}

// --- enum flag parsers + labels ---

func severityFlag(s string) sharedv1.LogSeverity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "info":
		return sharedv1.LogSeverity_LOG_SEVERITY_INFO
	case "low":
		return sharedv1.LogSeverity_LOG_SEVERITY_LOW
	case "medium":
		return sharedv1.LogSeverity_LOG_SEVERITY_MEDIUM
	case "high":
		return sharedv1.LogSeverity_LOG_SEVERITY_HIGH
	case "critical":
		return sharedv1.LogSeverity_LOG_SEVERITY_CRITICAL
	default:
		return sharedv1.LogSeverity_LOG_SEVERITY_UNSPECIFIED
	}
}

func entryTypeFlag(s string) sharedv1.LogEntryType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "decision":
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_DECISION
	case "finding":
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_FINDING
	case "bug", "bug_report", "bug-report":
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT
	case "record":
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_RECORD
	case "note":
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_NOTE
	default:
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_UNSPECIFIED
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

func syncStatusFlag(s string) sharedv1.LogSyncStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "local":
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL
	case "pending":
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_PENDING
	case "synced":
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_SYNCED
	case "failed", "sync_failed", "sync-failed":
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_FAILED
	default:
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_UNSPECIFIED
	}
}

func entryTypeLabel(t sharedv1.LogEntryType) string {
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

func syncStatusLabel(s sharedv1.LogSyncStatus) string {
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
