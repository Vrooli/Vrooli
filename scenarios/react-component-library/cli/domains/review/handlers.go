package review

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review/review_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the generated
// Connect-Go client. Mirrors cli/domains/adoptions/.
type handlers struct {
	core   *cliapp.ScenarioApp
	client reviewconnect.ReviewServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: reviewconnect.NewReviewServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) findingsCall(ctx cliapp.OperationContext) (*reviewv1.ListFindingsResponse, error) {
	status, err := parseFindingStatus(ctx.Flag("status"))
	if err != nil {
		return nil, err
	}
	severity, err := parseFindingSeverity(ctx.Flag("severity"))
	if err != nil {
		return nil, err
	}
	req := &reviewv1.ListFindingsRequest{
		Status:      status,
		Severity:    severity,
		SubjectKind: ctx.Flag("subject-kind"),
		Query:       ctx.Flag("query"),
	}
	if req.Limit, err = parseNonNegativeInt32(ctx.Flag("limit"), "--limit"); err != nil {
		return nil, err
	}
	if req.Offset, err = parseNonNegativeInt32(ctx.Flag("offset"), "--offset"); err != nil {
		return nil, err
	}
	resp, err := h.client.ListFindings(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no findings")
	}
	return resp.Msg, nil
}

func (h *handlers) findingsReport(_ cliapp.OperationContext, msg *reviewv1.ListFindingsResponse) cliapp.ListReport {
	findings := msg.GetFindings()
	rows := make([]string, 0, len(findings))
	for _, finding := range findings {
		rows = append(rows, formatFinding(finding))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d finding(s) of %d in the filtered queue.", len(findings), msg.GetTotal())},
		ResultsHeading: "Findings",
		Results:        rows,
		RetrievalHints: []string{
			"`review finding <id>` — inspect one finding and its dispositions",
			"`review summary` — read queue counts by severity and status",
		},
	}
}

func (h *handlers) findingCall(ctx cliapp.OperationContext) (*reviewv1.GetFindingResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetFinding(context.Background(), connect.NewRequest(&reviewv1.GetFindingRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("get finding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetFinding() == nil {
		return nil, fmt.Errorf("server returned no finding %q", id)
	}
	return resp.Msg, nil
}

func (h *handlers) findingReport(_ cliapp.OperationContext, msg *reviewv1.GetFindingResponse) cliapp.ListReport {
	finding := msg.GetFinding()
	rows := []string{formatFinding(finding)}
	for _, disposition := range finding.GetDispositions() {
		rows = append(rows, fmt.Sprintf("  disposition %s decision=%s actor=%s at=%s reason=%s",
			disposition.GetId(), decisionLabel(disposition.GetDecision()), nonEmpty(disposition.GetActor(), "(unattributed)"),
			nonEmpty(disposition.GetAt(), "(unknown)"), nonEmpty(disposition.GetReason(), "(none)")))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Finding %s: %s / %s.", finding.GetId(), severityLabel(finding.GetSeverity()), statusLabel(finding.GetStatus()))},
		ResultsHeading: "Finding",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("`review disposition %s --decision fix --reason <reason>` — record a decision", finding.GetId()),
			fmt.Sprintf("`review assign %s --owner <owner>` — set the accountable owner", finding.GetId()),
		},
	}
}

func (h *handlers) dispositionCall(ctx cliapp.OperationContext) (*reviewv1.RecordDispositionResponse, error) {
	decision, err := parseDispositionDecision(ctx.Flag("decision"))
	if err != nil {
		return nil, err
	}
	if decision == reviewv1.DispositionDecision_DISPOSITION_DECISION_UNSPECIFIED {
		return nil, fmt.Errorf("--decision is required (acknowledge, accept-risk, fix, retire)")
	}
	resp, err := h.client.RecordDisposition(context.Background(), connect.NewRequest(&reviewv1.RecordDispositionRequest{
		Id:       ctx.Positional("id"),
		Decision: decision,
		Reason:   ctx.Flag("reason"),
		Actor:    ctx.Flag("actor"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record finding disposition", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetFinding() == nil {
		return nil, fmt.Errorf("server returned no finding after recording the disposition")
	}
	return resp.Msg, nil
}

func (h *handlers) dispositionReport(ctx cliapp.OperationContext, msg *reviewv1.RecordDispositionResponse) cliapp.MutationReport {
	finding := msg.GetFinding()
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Recorded %s disposition on finding %s.", decisionLabel(parseDecisionOrDefault(ctx.Flag("decision"))), finding.GetId())},
		Changes:     []string{formatFinding(finding)},
		NextCommand: []string{fmt.Sprintf("`review finding %s` — inspect the durable disposition history", finding.GetId())},
	}
}

func (h *handlers) assignCall(ctx cliapp.OperationContext) (*reviewv1.AssignFindingResponse, error) {
	owner := strings.TrimSpace(ctx.Flag("owner"))
	if owner == "" {
		return nil, fmt.Errorf("--owner is required to assign a finding")
	}
	resp, err := h.client.AssignFinding(context.Background(), connect.NewRequest(&reviewv1.AssignFindingRequest{
		Id:    ctx.Positional("id"),
		Owner: owner,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("assign finding", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetFinding() == nil {
		return nil, fmt.Errorf("server returned no finding after assignment")
	}
	return resp.Msg, nil
}

func (h *handlers) assignReport(_ cliapp.OperationContext, msg *reviewv1.AssignFindingResponse) cliapp.MutationReport {
	finding := msg.GetFinding()
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Assigned finding %s to %s.", finding.GetId(), nonEmpty(finding.GetOwner(), "(unassigned)"))},
		Changes:     []string{formatFinding(finding)},
		NextCommand: []string{"`review findings --status open` — continue triaging findings"},
	}
}

func (h *handlers) summaryCall(_ cliapp.OperationContext) (*reviewv1.GetReviewSummaryResponse, error) {
	resp, err := h.client.GetReviewSummary(context.Background(), connect.NewRequest(&reviewv1.GetReviewSummaryRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get review summary", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetSummary() == nil {
		return nil, fmt.Errorf("server returned no review summary")
	}
	return resp.Msg, nil
}

func (h *handlers) summaryReport(_ cliapp.OperationContext, msg *reviewv1.GetReviewSummaryResponse) cliapp.ListReport {
	summary := msg.GetSummary()
	return cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Review queue: %d open, %d blocker(s), %d high, %d acknowledged, %d accepted, %d fixed.",
			summary.GetOpen(), summary.GetBlockers(), summary.GetHigh(), summary.GetAcknowledged(), summary.GetAccepted(), summary.GetFixed())},
		ResultsHeading: "Counts",
		Results: []string{
			fmt.Sprintf("open: %d", summary.GetOpen()),
			fmt.Sprintf("blockers: %d", summary.GetBlockers()),
			fmt.Sprintf("high: %d", summary.GetHigh()),
			fmt.Sprintf("acknowledged: %d", summary.GetAcknowledged()),
			fmt.Sprintf("accepted: %d", summary.GetAccepted()),
			fmt.Sprintf("fixed: %d", summary.GetFixed()),
		},
		RetrievalHints: []string{
			"`review findings --severity blocker` — list the blocking queue",
			"`review findings --status open` — list everything still open",
		},
	}
}

func formatFinding(finding *reviewv1.Finding) string {
	if finding == nil {
		return "(nil finding)"
	}
	subject := finding.GetSubjectKind()
	if subject == "" {
		subject = "subject"
	}
	if finding.GetSubjectId() != "" {
		subject += "/" + finding.GetSubjectId()
	}
	line := fmt.Sprintf("%s [%s/%s] %s — %s",
		finding.GetId(), severityLabel(finding.GetSeverity()), statusLabel(finding.GetStatus()), subject,
		nonEmpty(finding.GetMessage(), "(no message)"))
	if gate := strings.TrimSpace(finding.GetGate()); gate != "" {
		line += " gate=" + gate
	}
	if owner := strings.TrimSpace(finding.GetOwner()); owner != "" {
		line += " owner=" + owner
	}
	if remediation := strings.TrimSpace(finding.GetRemediation()); remediation != "" {
		line += " | remediation: " + remediation
	}
	return line
}

func parseFindingStatus(raw string) (reviewv1.FindingStatus, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(raw)), "_", "-")
	normalized = strings.TrimPrefix(normalized, "finding-status-")
	switch normalized {
	case "", "unspecified", "any":
		return reviewv1.FindingStatus_FINDING_STATUS_UNSPECIFIED, nil
	case "open":
		return reviewv1.FindingStatus_FINDING_STATUS_OPEN, nil
	case "acknowledged":
		return reviewv1.FindingStatus_FINDING_STATUS_ACKNOWLEDGED, nil
	case "accepted":
		return reviewv1.FindingStatus_FINDING_STATUS_ACCEPTED, nil
	case "fixed":
		return reviewv1.FindingStatus_FINDING_STATUS_FIXED, nil
	case "retired":
		return reviewv1.FindingStatus_FINDING_STATUS_RETIRED, nil
	default:
		return 0, fmt.Errorf("--status must be one of open, acknowledged, accepted, fixed, retired (got %q)", raw)
	}
}

func parseFindingSeverity(raw string) (reviewv1.FindingSeverity, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(raw)), "_", "-")
	normalized = strings.TrimPrefix(normalized, "finding-severity-")
	switch normalized {
	case "", "unspecified", "any":
		return reviewv1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED, nil
	case "blocker":
		return reviewv1.FindingSeverity_FINDING_SEVERITY_BLOCKER, nil
	case "high":
		return reviewv1.FindingSeverity_FINDING_SEVERITY_HIGH, nil
	case "medium":
		return reviewv1.FindingSeverity_FINDING_SEVERITY_MEDIUM, nil
	case "low":
		return reviewv1.FindingSeverity_FINDING_SEVERITY_LOW, nil
	case "info":
		return reviewv1.FindingSeverity_FINDING_SEVERITY_INFO, nil
	default:
		return 0, fmt.Errorf("--severity must be one of blocker, high, medium, low, info (got %q)", raw)
	}
}

func parseDispositionDecision(raw string) (reviewv1.DispositionDecision, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(raw)), "_", "-")
	normalized = strings.TrimPrefix(normalized, "disposition-decision-")
	switch normalized {
	case "", "unspecified":
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_UNSPECIFIED, nil
	case "acknowledge":
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_ACKNOWLEDGE, nil
	case "accept-risk", "accept":
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_ACCEPT_RISK, nil
	case "fix":
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_FIX, nil
	case "retire":
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_RETIRE, nil
	default:
		return 0, fmt.Errorf("--decision must be one of acknowledge, accept-risk, fix, retire (got %q)", raw)
	}
}

func parseDecisionOrDefault(raw string) reviewv1.DispositionDecision {
	decision, err := parseDispositionDecision(raw)
	if err != nil {
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_UNSPECIFIED
	}
	return decision
}

func severityLabel(value reviewv1.FindingSeverity) string {
	switch value {
	case reviewv1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return "blocker"
	case reviewv1.FindingSeverity_FINDING_SEVERITY_HIGH:
		return "high"
	case reviewv1.FindingSeverity_FINDING_SEVERITY_MEDIUM:
		return "medium"
	case reviewv1.FindingSeverity_FINDING_SEVERITY_LOW:
		return "low"
	case reviewv1.FindingSeverity_FINDING_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

func statusLabel(value reviewv1.FindingStatus) string {
	switch value {
	case reviewv1.FindingStatus_FINDING_STATUS_OPEN:
		return "open"
	case reviewv1.FindingStatus_FINDING_STATUS_ACKNOWLEDGED:
		return "acknowledged"
	case reviewv1.FindingStatus_FINDING_STATUS_ACCEPTED:
		return "accepted"
	case reviewv1.FindingStatus_FINDING_STATUS_FIXED:
		return "fixed"
	case reviewv1.FindingStatus_FINDING_STATUS_RETIRED:
		return "retired"
	default:
		return "unspecified"
	}
}

func decisionLabel(value reviewv1.DispositionDecision) string {
	switch value {
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_ACKNOWLEDGE:
		return "acknowledge"
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_ACCEPT_RISK:
		return "accept-risk"
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_FIX:
		return "fix"
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_RETIRE:
		return "retire"
	default:
		return "unspecified"
	}
}

func parseNonNegativeInt32(raw, flag string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer (got %q)", flag, raw)
	}
	return int32(value), nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
