package ledger

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	ledgerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger/ledger_v1connect"
)

const GroupName = "ledger"

type handlers struct {
	client ledgerconnect.LedgerServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: ledgerconnect.NewLedgerServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*ledgerv1.ListPublishRecordsResponse, error) {
	response, err := h.client.ListPublishRecords(context.Background(), connect.NewRequest(&ledgerv1.ListPublishRecordsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list publish records", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no publish records response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *ledgerv1.ListPublishRecordsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.PublishRecords))
	for _, record := range message.PublishRecords {
		results = append(results, fmt.Sprintf("%s — draft=%s url=%s post=%s", record.Id, record.DraftId, record.PublishedUrl, record.PlatformPostId))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d publish record(s).", len(message.PublishRecords))}, ResultsHeading: "Publish records", Results: results}
}

func (h *handlers) contaminationCall(ctx cliapp.OperationContext) (*ledgerv1.ListContaminatedPublishRecordsResponse, error) {
	response, err := h.client.ListContaminatedPublishRecords(context.Background(), connect.NewRequest(&ledgerv1.ListContaminatedPublishRecordsRequest{ClaimId: ctx.Positional("claim-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list contaminated publish records", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no contamination response")
	}
	return response.Msg, nil
}
func (h *handlers) contaminationReport(_ cliapp.OperationContext, message *ledgerv1.ListContaminatedPublishRecordsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.PublishRecords))
	for _, record := range message.PublishRecords {
		results = append(results, fmt.Sprintf("%s — draft=%s url=%s", record.Id, record.DraftId, record.PublishedUrl))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d contaminated publish record(s).", len(message.PublishRecords))}, ResultsHeading: "Contaminated publish records", Results: results}
}

func (h *handlers) coverageCall(ctx cliapp.OperationContext) (*ledgerv1.ListCoverageResponse, error) {
	days, err := strconv.ParseInt(ctx.Flag("stale-after-days"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parse stale-after-days: %w", err)
	}
	response, err := h.client.ListCoverage(context.Background(), connect.NewRequest(&ledgerv1.ListCoverageRequest{StaleAfterDays: int32(days)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list coverage", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no coverage response")
	}
	return response.Msg, nil
}
func (h *handlers) coverageReport(_ cliapp.OperationContext, message *ledgerv1.ListCoverageResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Cells))
	for _, cell := range message.Cells {
		results = append(results, fmt.Sprintf("campaign=%s lane=%s channel=%s sku=%s posts=%d stale=%t", cell.CampaignId, cell.Lane, cell.Channel, cell.Sku, cell.PublishCount, cell.Stale))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d coverage cell(s).", len(message.Cells))}, ResultsHeading: "Coverage", Results: results}
}

func (h *handlers) remediationsCall(ctx cliapp.OperationContext) (*ledgerv1.ListRemediationsResponse, error) {
	openOnly := false
	if raw := ctx.Flag("open-only"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil { return nil, fmt.Errorf("parse open-only: %w", err) }
		openOnly = parsed
	}
	response, err := h.client.ListRemediations(context.Background(), connect.NewRequest(&ledgerv1.ListRemediationsRequest{PublishRecordId: ctx.Flag("publish-record-id"), OpenOnly: openOnly}))
	if err != nil { return nil, cliapp.WrapAPIError("list remediations", err, nil) }
	if response == nil || response.Msg == nil { return nil, fmt.Errorf("server returned no remediations response") }
	return response.Msg, nil
}
func (h *handlers) remediationsReport(_ cliapp.OperationContext, message *ledgerv1.ListRemediationsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Remediations))
	for _, remediation := range message.Remediations { results = append(results, fmt.Sprintf("%s — %s %s (%s)", remediation.Id, remediation.Kind, remediation.Status, remediation.PublishRecordId)) }
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d remediation(s).", len(message.Remediations))}, ResultsHeading: "Remediations", Results: results}
}

func (h *handlers) remediateCall(ctx cliapp.OperationContext) (*ledgerv1.CreateRemediationResponse, error) {
	response, err := h.client.CreateRemediation(context.Background(), connect.NewRequest(&ledgerv1.CreateRemediationRequest{PublishRecordId: ctx.Flag("publish-record-id"), Kind: ctx.Flag("kind"), Note: ctx.Flag("note")}))
	if err != nil { return nil, cliapp.WrapAPIError("create remediation", err, nil) }
	if response == nil || response.Msg == nil || response.Msg.Remediation == nil { return nil, fmt.Errorf("server returned no remediation") }
	return response.Msg, nil
}
func (h *handlers) remediateReport(_ cliapp.OperationContext, message *ledgerv1.CreateRemediationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded remediation %s.", message.Remediation.Id)}}
}

func (h *handlers) resolveRemediationCall(ctx cliapp.OperationContext) (*ledgerv1.ResolveRemediationResponse, error) {
	response, err := h.client.ResolveRemediation(context.Background(), connect.NewRequest(&ledgerv1.ResolveRemediationRequest{Id: ctx.Positional("id")}))
	if err != nil { return nil, cliapp.WrapAPIError("resolve remediation", err, nil) }
	if response == nil || response.Msg == nil || response.Msg.Remediation == nil { return nil, fmt.Errorf("server returned no resolved remediation") }
	return response.Msg, nil
}
func (h *handlers) resolveRemediationReport(_ cliapp.OperationContext, message *ledgerv1.ResolveRemediationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Resolved remediation %s.", message.Remediation.Id)}}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"LedgerService.ListPublishRecords":             cliapp.ProtoList(h.listCall, h.listReport),
		"LedgerService.ListContaminatedPublishRecords": cliapp.ProtoList(h.contaminationCall, h.contaminationReport),
		"LedgerService.ListCoverage":                   cliapp.ProtoList(h.coverageCall, h.coverageReport),
		"LedgerService.ListRemediations":               cliapp.ProtoList(h.remediationsCall, h.remediationsReport),
		"LedgerService.CreateRemediation":              cliapp.ProtoMutation(h.remediateCall, h.remediateReport),
		"LedgerService.ResolveRemediation":             cliapp.ProtoMutation(h.resolveRemediationCall, h.resolveRemediationReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ledger: load from manifest: %w", err)
	}
	return group, nil
}
