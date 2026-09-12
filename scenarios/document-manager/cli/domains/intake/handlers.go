package intake

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	intakev1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/intake"
	intakeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/intake/intake_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type handlers struct {
	client intakeconnect.IntakeServiceClient
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
	return &handlers{client: intakeconnect.NewIntakeServiceClient(httpClient, baseURL)}
}

func (h *handlers) ingestCall(ctx cliapp.OperationContext) (*intakev1.IngestResponse, error) {
	data, err := os.ReadFile(ctx.Flag("file"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Ingest(context.Background(), connect.NewRequest(&intakev1.IngestRequest{Content: data, SourceName: ctx.Flag("source"), CollectionId: ctx.Flag("collection"), PrivacyClass: privacy(ctx.Flag("privacy"))}))
	if err != nil {
		return nil, cliapp.WrapAPIError("ingest", err, nil)
	}
	return resp.Msg, nil
}

func privacy(raw string) sharedv1.PrivacyClass {
	switch raw {
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

func (h *handlers) ingestReport(_ cliapp.OperationContext, r *intakev1.IngestResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Ingested %s (duplicate=%t).", r.Document.Id, r.Duplicate)}}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*intakev1.GetDocumentResponse, error) {
	resp, err := h.client.GetDocument(context.Background(), connect.NewRequest(&intakev1.GetDocumentRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, r *intakev1.GetDocumentResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Document " + r.Document.Id}, Results: []string{r.Document.DetectedMime}}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*intakev1.ListDocumentsResponse, error) {
	limit, err := limit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListDocuments(context.Background(), connect.NewRequest(&intakev1.ListDocumentsRequest{Limit: limit}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, r *intakev1.ListDocumentsResponse) cliapp.ListReport {
	out := []string{}
	for _, d := range r.Documents {
		out = append(out, fmt.Sprintf("%s %s", d.Id, d.DetectedMime))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d document(s).", len(out))}, Results: out}
}

func (h *handlers) sourcesCall(_ cliapp.OperationContext) (*intakev1.ListSourcesResponse, error) {
	resp, err := h.client.ListSources(context.Background(), connect.NewRequest(&intakev1.ListSourcesRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) sourcesReport(_ cliapp.OperationContext, r *intakev1.ListSourcesResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d source(s).", len(r.Sources))}, Results: r.Sources}
}

func (h *handlers) watchCall(ctx cliapp.OperationContext) (*intakev1.ConfigureWatchResponse, error) {
	enabled, _ := strconv.ParseBool(ctx.Flag("enabled"))
	resp, err := h.client.ConfigureWatch(context.Background(), connect.NewRequest(&intakev1.ConfigureWatchRequest{Path: ctx.Flag("path"), Enabled: enabled}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) watchReport(_ cliapp.OperationContext, r *intakev1.ConfigureWatchResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Watch %s enabled=%t", r.Path, r.Enabled)}}
}

func (h *handlers) verdictCall(ctx cliapp.OperationContext) (*intakev1.GetTypeVerdictResponse, error) {
	resp, err := h.client.GetTypeVerdict(context.Background(), connect.NewRequest(&intakev1.GetTypeVerdictRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) verdictReport(_ cliapp.OperationContext, r *intakev1.GetTypeVerdictResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Stored type verdict"}, Results: []string{fmt.Sprintf("%s %s %.2f", r.DetectedMime, r.PdfType, r.Confidence)}}
}
