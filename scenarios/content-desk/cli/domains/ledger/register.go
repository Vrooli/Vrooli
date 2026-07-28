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

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"LedgerService.ListPublishRecords":             cliapp.ProtoList(h.listCall, h.listReport),
		"LedgerService.ListContaminatedPublishRecords": cliapp.ProtoList(h.contaminationCall, h.contaminationReport),
		"LedgerService.ListCoverage":                   cliapp.ProtoList(h.coverageCall, h.coverageReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ledger: load from manifest: %w", err)
	}
	return group, nil
}
