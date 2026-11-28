package contentdesk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts/artifacts_v1connect"
	ledgerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger/ledger_v1connect"
)

type staticResolver struct{ url string }

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type artifactsHandler struct {
	artifactsconnect.UnimplementedArtifactsServiceHandler
}

func (artifactsHandler) RecordReleaseOutcome(_ context.Context, request *connect.Request[artifactsv1.RecordReleaseOutcomeRequest]) (*connect.Response[artifactsv1.RecordReleaseOutcomeResponse], error) {
	if request.Msg.ReceiptId != "release-1" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&artifactsv1.RecordReleaseOutcomeResponse{PublishRecordId: "record-1"}), nil
}

type ledgerHandler struct {
	ledgerconnect.UnimplementedLedgerServiceHandler
}

func (ledgerHandler) IngestMetricSample(_ context.Context, request *connect.Request[ledgerv1.IngestMetricSampleRequest]) (*connect.Response[ledgerv1.IngestMetricSampleResponse], error) {
	if request.Msg.SampleId != "sample-1" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&ledgerv1.IngestMetricSampleResponse{SampleId: request.Msg.SampleId, Accepted: true}), nil
}

func TestClientDeliversTypedReleaseAndMetric(t *testing.T) {
	artifactsPath, artifactsService := artifactsconnect.NewArtifactsServiceHandler(artifactsHandler{})
	ledgerPath, ledgerService := ledgerconnect.NewLedgerServiceHandler(ledgerHandler{})
	mux := http.NewServeMux()
	mux.Handle(artifactsPath, artifactsService)
	mux.Handle(ledgerPath, ledgerService)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := &Client{resolver: staticResolver{url: server.URL}, http: server.Client()}
	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	require.NoError(t, client.DeliverRelease(context.Background(), ReleaseOutcome{ReceiptID: "release-1", DraftID: "draft-1", Status: "published", PlatformPostID: "post-1", PublishedURL: "https://example.test/post-1", PublishedAt: now}))
	require.NoError(t, client.DeliverMetric(context.Background(), MetricSample{ID: "sample-1", ReleaseID: "release-1", DraftID: "draft-1", Metric: "impressions", Value: 2, ObservedAt: now}))
}
