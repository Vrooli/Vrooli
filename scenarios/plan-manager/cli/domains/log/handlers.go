package log

import (
	"context"
	"fmt"
	"strings"

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
		Evidence:        splitCSV(ctx.Flag("evidence")),
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
		Evidence:        splitCSV(ctx.Flag("evidence")),
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
		Evidence:        splitCSV(ctx.Flag("evidence")),
		SourceCommand:   ctx.Flag("source-command"),
		IdempotencyKey:  ctx.Flag("idempotency-key"),
		RunId:           ctx.Flag("run-id"),
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
		Evidence:        splitCSV(ctx.Flag("evidence")),
		SourceCommand:   ctx.Flag("source-command"),
		IdempotencyKey:  ctx.Flag("idempotency-key"),
		RunId:           ctx.Flag("run-id"),
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
		Evidence:        splitCSV(ctx.Flag("evidence")),
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
		AddEvidence: splitCSV(ctx.Flag("add-evidence")),
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
	changes := []string{entryLine(e)}
	if e.GetSyncStatus() != sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL && e.GetSyncStatus() != sharedv1.LogSyncStatus_LOG_SYNC_STATUS_UNSPECIFIED {
		changes = append(changes, fmt.Sprintf("downstream sync: %s", syncStatusLabel(e.GetSyncStatus())))
	}
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

func entryDetail(e *sharedv1.LogEntry) []string {
	if e == nil {
		return nil
	}
	out := make([]string, 0, 6)
	if e.GetPhaseId() != "" {
		out = append(out, "phase: "+e.GetPhaseId())
	}
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
	if step == nil || strings.TrimSpace(step.GetStepKind()) == "" {
		return nil
	}
	out := []string{fmt.Sprintf("Current Step (%s): %s", step.GetStepKind(), step.GetSummary())}
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
