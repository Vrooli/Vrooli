package retrieval

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	retrievalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/retrieval"
	retrievalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/retrieval/retrieval_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type handlers struct {
	client retrievalconnect.RetrievalServiceClient
}

func limit(raw string) (int32, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid limit %q: must be a non-negative 32-bit integer", raw)
	}
	return int32(value), nil // #nosec G115 -- ParseInt enforces the int32 range above.
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: retrievalconnect.NewRetrievalServiceClient(httpClient, baseURL)}
}
func privacy(raw string) sharedv1.PrivacyClass {
	switch strings.ToLower(raw) {
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

func (h *handlers) queryCall(ctx cliapp.OperationContext) (*retrievalv1.QueryResponse, error) {
	limit, err := limit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	federated, _ := strconv.ParseBool(ctx.Flag("federated"))
	resp, err := h.client.Query(context.Background(), connect.NewRequest(&retrievalv1.QueryRequest{Text: ctx.Flag("text"), CollectionId: ctx.Flag("collection"), CallerMaxPrivacy: privacy(ctx.Flag("privacy")), Federated: federated, Limit: limit}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
func (h *handlers) queryReport(_ cliapp.OperationContext, r *retrievalv1.QueryResponse) cliapp.ListReport {
	return report(r.Results, r.Partial)
}
func (h *handlers) similarCall(ctx cliapp.OperationContext) (*retrievalv1.SimilarResponse, error) {
	limit, err := limit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Similar(context.Background(), connect.NewRequest(&retrievalv1.SimilarRequest{DocumentHash: ctx.Flag("document-hash"), CallerMaxPrivacy: privacy(ctx.Flag("privacy")), Limit: limit}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
func (h *handlers) similarReport(_ cliapp.OperationContext, r *retrievalv1.SimilarResponse) cliapp.ListReport {
	return report(r.Results, r.Partial)
}
func report(results []*retrievalv1.QueryResult, partial bool) cliapp.ListReport {
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, fmt.Sprintf("%s %.4f %s", r.DocumentHash, r.Score, r.AnchorUri))
	}
	status := "complete"
	if partial {
		status = "partial"
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s index: %d result(s)", status, len(out))}, Results: out}
}
