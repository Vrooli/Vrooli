package audits

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	auditsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits"
	auditsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits/audits_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client auditsconnect.AuditsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: auditsconnect.NewAuditsServiceClient(httpClient, baseURL),
	}
}

// auditPollInterval is how often the CLI polls an audit for its terminal state.
// RunSnapshotAudit is async (returns immediately), so the CLI owns the wait —
// each poll is a fast GetAudit.
const auditPollInterval = 2 * time.Second

// auditPollDeadline caps how long the CLI waits for an audit to reach a terminal
// state. An audit restores AND captures a whole target, so it can take many
// minutes on large targets; this generous ceiling only guards against a wedged
// backend (the worker always reaches terminal, and reconciliation closes
// orphans).
const auditPollDeadline = 6 * time.Hour

func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.RunSnapshotAudit(context.Background(), connect.NewRequest(&auditsv1.RunSnapshotAuditRequest{
		TargetId:            ctx.Flag("target"),
		DestinationId:       ctx.Flag("destination"),
		SnapshotId:          ctx.Flag("snapshot"),
		IncludeContentHash:  !ctx.BoolFlag("no-content-hash"),
		IncludeSqliteChecks: !ctx.BoolFlag("no-sqlite-checks"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("run snapshot audit", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Audit == nil {
		return fmt.Errorf("server returned no audit")
	}
	final, err := h.pollToTerminal(resp.Msg.Audit.Id)
	if err != nil {
		return cliapp.WrapAPIError("run snapshot audit", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, &auditsv1.RunSnapshotAuditResponse{Audit: final}, cliapp.MutationReport{
		Result:  []string{auditVerdictLine(final)},
		Changes: auditEvidence(final),
		NextCommand: []string{
			fmt.Sprintf("`audits get %s` — show the full audit record", final.Id),
			fmt.Sprintf("`audits list --target %s` — prior audits for this target", final.TargetId),
		},
	})
}

// pollToTerminal polls GetAudit until the record reaches a terminal state.
func (h *handlers) pollToTerminal(id string) (*auditsv1.Audit, error) {
	deadline := time.Now().Add(auditPollDeadline)
	for {
		resp, err := h.client.GetAudit(context.Background(), connect.NewRequest(&auditsv1.GetAuditRequest{Id: id}))
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Msg == nil || resp.Msg.Audit == nil {
			return nil, fmt.Errorf("server returned no audit for %q", id)
		}
		if isTerminalAudit(resp.Msg.Audit.Status) {
			return resp.Msg.Audit, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("audit %q did not reach a terminal state within %s (still %s)", id, auditPollDeadline, auditStatusLabel(resp.Msg.Audit.Status))
		}
		time.Sleep(auditPollInterval)
	}
}

func isTerminalAudit(s auditsv1.AuditStatus) bool {
	switch s {
	case auditsv1.AuditStatus_AUDIT_STATUS_COMPLETED, auditsv1.AuditStatus_AUDIT_STATUS_FAILED:
		return true
	default:
		return false
	}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetAudit(context.Background(), connect.NewRequest(&auditsv1.GetAuditRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get audit %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Audit == nil {
		return fmt.Errorf("server returned no audit")
	}
	results := append([]string{auditVerdictLine(resp.Msg.Audit)}, auditEvidence(resp.Msg.Audit)...)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched audit %s.", resp.Msg.Audit.Id)},
		ResultsHeading: "Audit",
		Results:        results,
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListAudits(context.Background(), connect.NewRequest(&auditsv1.ListAuditsRequest{
		TargetId: ctx.Flag("target"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list audits", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no audits response")
	}
	results := make([]string, 0, len(resp.Msg.Audits))
	for _, a := range resp.Msg.Audits {
		results = append(results, formatAuditLine(a))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d audit(s).", len(resp.Msg.Audits))},
		ResultsHeading: "Audits",
		Results:        results,
		RetrievalHints: []string{
			"`audits get <id>` — show a single audit with full evidence",
			"`audits run --target <id> --destination <id> --snapshot <id>` — run a new audit",
		},
	})
}

// auditVerdictLine is the one-line operator status: matched / mismatched (drift
// explained or not) / failed.
func auditVerdictLine(a *auditsv1.Audit) string {
	if a == nil {
		return "(nil)"
	}
	if a.Status == auditsv1.AuditStatus_AUDIT_STATUS_FAILED {
		return fmt.Sprintf("Audit FAILED: %s — %s", a.Id, a.Error)
	}
	if a.Comparison == nil {
		return fmt.Sprintf("Audit %s: %s (no comparison)", auditStatusLabel(a.Status), a.Id)
	}
	if a.Comparison.Matches {
		return fmt.Sprintf("Audit PASSED: %s — restored snapshot matches live by all generic signals.", a.Id)
	}
	if a.Comparison.LiveNewerThanSnapshot {
		return fmt.Sprintf("Audit DIFF (drift): %s — live changed after the snapshot; differences are explainable as drift.", a.Id)
	}
	return fmt.Sprintf("Audit DIFF: %s — restored snapshot differs from live and live is NOT newer than the snapshot (investigate).", a.Id)
}

// auditEvidence renders the generic findings + evidence lines for an audit.
func auditEvidence(a *auditsv1.Audit) []string {
	if a == nil {
		return nil
	}
	var out []string
	out = append(out, fmt.Sprintf("Restorable: %t", a.Restorable))
	if a.Comparison != nil && len(a.Comparison.Mismatches) > 0 {
		out = append(out, "Mismatches:")
		for _, m := range a.Comparison.Mismatches {
			out = append(out, "  - "+m)
		}
	}
	out = append(out, "Live: "+inventoryLine(a.Live))
	out = append(out, "Snapshot: "+inventoryLine(a.Snapshot))
	for _, s := range sqliteLines(a.Snapshot) {
		out = append(out, "Snapshot SQLite "+s)
	}
	return out
}

func inventoryLine(inv *auditsv1.InventorySummary) string {
	if inv == nil {
		return "(none)"
	}
	line := fmt.Sprintf("files=%d dirs=%d symlinks=%d bytes=%d pathHash=%s",
		inv.Files, inv.Directories, inv.Symlinks, inv.RegularBytes, shortHash(inv.PathListSha256))
	if inv.TreeContentSha256 != "" {
		line += " contentHash=" + shortHash(inv.TreeContentSha256)
	}
	if len(inv.UnreadablePaths) > 0 {
		line += fmt.Sprintf(" unreadable=%d", len(inv.UnreadablePaths))
	}
	return line
}

func sqliteLines(inv *auditsv1.InventorySummary) []string {
	if inv == nil {
		return nil
	}
	var out []string
	for _, s := range inv.Sqlite {
		out = append(out, fmt.Sprintf("%s: integrity=%s tables=%d pages=%d schema=%s",
			s.Path, s.IntegrityStatus, s.TableCount, s.PageCount, shortHash(s.SchemaSha256)))
	}
	return out
}

func formatAuditLine(a *auditsv1.Audit) string {
	if a == nil {
		return "(nil)"
	}
	verdict := auditStatusLabel(a.Status)
	if a.Status == auditsv1.AuditStatus_AUDIT_STATUS_COMPLETED && a.Comparison != nil {
		if a.Comparison.Matches {
			verdict = "match"
		} else if a.Comparison.LiveNewerThanSnapshot {
			verdict = "diff(drift)"
		} else {
			verdict = "diff"
		}
	}
	requested := ""
	if a.RequestedAt != nil {
		requested = a.RequestedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — target=%s destination=%s snapshot=%s verdict=%s requested=%s",
		a.Id, a.TargetId, a.DestinationId, a.SnapshotId, verdict, requested)
}

func auditStatusLabel(s auditsv1.AuditStatus) string {
	switch s {
	case auditsv1.AuditStatus_AUDIT_STATUS_REQUESTED:
		return "requested"
	case auditsv1.AuditStatus_AUDIT_STATUS_RUNNING:
		return "running"
	case auditsv1.AuditStatus_AUDIT_STATUS_COMPLETED:
		return "completed"
	case auditsv1.AuditStatus_AUDIT_STATUS_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

// shortHash trims a sha256 hex to its first 12 chars for human-readable output;
// the full hash is in --json.
func shortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return strings.ToLower(h[:12])
}
