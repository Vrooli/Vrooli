// Package ledger mounts the append-oriented publish-history Connect surface.
package ledger

import (
	"context"
	"fmt"
	"time"

	internalledger "content-desk/internal/ledger"
	"content-desk/internal/module"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	ledgerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger/ledger_v1connect"
)

type handler struct{ repo internalledger.Repository }

var _ ledgerconnect.LedgerServiceHandler = handler{}

func (h handler) ListPublishRecords(ctx context.Context, _ *connect.Request[ledgerv1.ListPublishRecordsRequest]) (*connect.Response[ledgerv1.ListPublishRecordsResponse], error) {
	records, err := h.repo.ListPublishHistory(ctx, 100)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &ledgerv1.ListPublishRecordsResponse{}
	for _, record := range records {
		response.PublishRecords = append(response.PublishRecords, &ledgerv1.PublishRecord{Id: record.ID, DraftId: record.DraftID, PublishedUrl: record.PublishedURL, PlatformPostId: record.PlatformPostID})
	}
	return connect.NewResponse(response), nil
}

func (h handler) ListContaminatedPublishRecords(ctx context.Context, request *connect.Request[ledgerv1.ListContaminatedPublishRecordsRequest]) (*connect.Response[ledgerv1.ListContaminatedPublishRecordsResponse], error) {
	records, err := h.repo.ContaminatedByClaim(ctx, request.Msg.ClaimId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &ledgerv1.ListContaminatedPublishRecordsResponse{}
	for _, record := range records {
		response.PublishRecords = append(response.PublishRecords, publishRecordMessage(record))
	}
	return connect.NewResponse(response), nil
}

func (h handler) ListCoverage(ctx context.Context, request *connect.Request[ledgerv1.ListCoverageRequest]) (*connect.Response[ledgerv1.ListCoverageResponse], error) {
	cells, err := h.repo.Coverage(ctx, time.Duration(request.Msg.StaleAfterDays)*24*time.Hour)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &ledgerv1.ListCoverageResponse{}
	for _, cell := range cells {
		response.Cells = append(response.Cells, &ledgerv1.CoverageCell{CampaignId: cell.CampaignID, Lane: cell.Lane, Channel: cell.Channel, Sku: cell.SKU, PublishCount: int32(cell.PublishCount), LastPublishedAt: cell.LastPublishedAt.Format(time.RFC3339Nano), Stale: cell.Stale})
	}
	return connect.NewResponse(response), nil
}

func (h handler) IngestMetricSample(ctx context.Context, request *connect.Request[ledgerv1.IngestMetricSampleRequest]) (*connect.Response[ledgerv1.IngestMetricSampleResponse], error) {
	observedAt, err := time.Parse(time.RFC3339Nano, request.Msg.ObservedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse observed_at: %w", err))
	}
	sample, err := h.repo.IngestMetricSample(ctx, internalledger.MetricSample{SampleID: request.Msg.SampleId, ReleaseID: request.Msg.ReleaseId, DraftID: request.Msg.DraftId, Metric: request.Msg.Metric, Value: request.Msg.Value, ObservedAt: observedAt})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&ledgerv1.IngestMetricSampleResponse{SampleId: sample.SampleID, Accepted: true}), nil
}

func (h handler) ListRemediations(ctx context.Context, request *connect.Request[ledgerv1.ListRemediationsRequest]) (*connect.Response[ledgerv1.ListRemediationsResponse], error) {
	remediations, err := h.repo.ListRemediations(ctx, request.Msg.PublishRecordId, request.Msg.OpenOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &ledgerv1.ListRemediationsResponse{}
	for _, remediation := range remediations {
		response.Remediations = append(response.Remediations, remediationMessage(remediation))
	}
	return connect.NewResponse(response), nil
}

func (h handler) CreateRemediation(ctx context.Context, request *connect.Request[ledgerv1.CreateRemediationRequest]) (*connect.Response[ledgerv1.CreateRemediationResponse], error) {
	remediation, err := h.repo.CreateRemediation(ctx, internalledger.Remediation{PublishRecordID: request.Msg.PublishRecordId, Kind: request.Msg.Kind, Note: request.Msg.Note})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&ledgerv1.CreateRemediationResponse{Remediation: remediationMessage(remediation)}), nil
}

func (h handler) ResolveRemediation(ctx context.Context, request *connect.Request[ledgerv1.ResolveRemediationRequest]) (*connect.Response[ledgerv1.ResolveRemediationResponse], error) {
	remediation, err := h.repo.ResolveRemediation(ctx, request.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&ledgerv1.ResolveRemediationResponse{Remediation: remediationMessage(remediation)}), nil
}

func publishRecordMessage(record internalledger.PublishRecord) *ledgerv1.PublishRecord {
	return &ledgerv1.PublishRecord{Id: record.ID, DraftId: record.DraftID, PublishedUrl: record.PublishedURL, PlatformPostId: record.PlatformPostID}
}

func remediationMessage(remediation internalledger.Remediation) *ledgerv1.Remediation {
	message := &ledgerv1.Remediation{Id: remediation.ID, PublishRecordId: remediation.PublishRecordID, Kind: remediation.Kind, Status: remediation.Status, Note: remediation.Note, CreatedAt: remediation.CreatedAt.Format(time.RFC3339Nano)}
	if !remediation.ResolvedAt.IsZero() {
		message.ResolvedAt = remediation.ResolvedAt.Format(time.RFC3339Nano)
	}
	return message
}

func Module(db *database.RoutedDB) module.Module {
	path, h := ledgerconnect.NewLedgerServiceHandler(handler{repo: internalledger.NewSQLiteRepository(db)})
	return module.Module{Name: "ledger", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalledger.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "ledger_list", Path: ledgerconnect.LedgerServiceListPublishRecordsProcedure, Method: "POST", Summary: "List publish records", Category: "ledger"},
	{ID: "ledger_contamination", Path: ledgerconnect.LedgerServiceListContaminatedPublishRecordsProcedure, Method: "POST", Summary: "List published records contaminated by a claim", Category: "ledger"},
	{ID: "ledger_coverage", Path: ledgerconnect.LedgerServiceListCoverageProcedure, Method: "POST", Summary: "List publish coverage cells and staleness", Category: "ledger"},
	{ID: "ledger_ingest_metric", Path: ledgerconnect.LedgerServiceIngestMetricSampleProcedure, Method: "POST", Summary: "Ingest idempotent Channel Manager metric sample", Category: "ledger"},
	{ID: "ledger_list_remediations", Path: ledgerconnect.LedgerServiceListRemediationsProcedure, Method: "POST", Summary: "List active or historical contamination remediations", Category: "ledger"},
	{ID: "ledger_create_remediation", Path: ledgerconnect.LedgerServiceCreateRemediationProcedure, Method: "POST", Summary: "Create a remediation for a published record", Category: "ledger"},
	{ID: "ledger_resolve_remediation", Path: ledgerconnect.LedgerServiceResolveRemediationProcedure, Method: "POST", Summary: "Resolve an open remediation without deleting history", Category: "ledger"},
}
