package enrichment

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	enrichmentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/enrichment"
	enrichmentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/enrichment/enrichment_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type handlers struct {
	client enrichmentconnect.EnrichmentServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: enrichmentconnect.NewEnrichmentServiceClient(httpClient, baseURL)}
}

func privacy(raw string) sharedv1.PrivacyClass {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "public":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC
	case "confidential":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	case "secret":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET
	default:
		return sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL
	}
}

func (h *handlers) enrichCall(ctx cliapp.OperationContext) (*enrichmentv1.EnrichResponse, error) {
	resp, err := h.client.Enrich(context.Background(), connect.NewRequest(&enrichmentv1.EnrichRequest{DocumentHash: ctx.Flag("document-hash"), Text: ctx.Flag("text"), PrivacyClass: privacy(ctx.Flag("privacy"))}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
func (h *handlers) enrichReport(_ cliapp.OperationContext, r *enrichmentv1.EnrichResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Enrichment status=%s", r.Status)}}
}

func (h *handlers) embedCall(ctx cliapp.OperationContext) (*enrichmentv1.EmbedResponse, error) {
	resp, err := h.client.Embed(context.Background(), connect.NewRequest(&enrichmentv1.EmbedRequest{DocumentHash: ctx.Flag("document-hash"), UnitId: ctx.Flag("unit-id"), Text: ctx.Flag("text"), PrivacyClass: privacy(ctx.Flag("privacy")), Role: ctx.Flag("role")}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
func (h *handlers) embedReport(_ cliapp.OperationContext, r *enrichmentv1.EmbedResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Embedding status=%s dimension=%d", r.Status, r.Dimension)}}
}
