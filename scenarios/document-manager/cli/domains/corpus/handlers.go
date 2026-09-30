package corpus

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/corpus"
	corpusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/corpus/corpus_v1connect"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type handlers struct {
	client corpusconnect.CorpusServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: corpusconnect.NewCorpusServiceClient(httpClient, baseURL)}
}

func privacy(raw string) documentpb.PrivacyClass {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "public":
		return documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC
	case "confidential":
		return documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	case "secret":
		return documentpb.PrivacyClass_PRIVACY_CLASS_SECRET
	default:
		return documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL
	}
}

func limit(raw string) (int32, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid limit %q: must be a non-negative 32-bit integer", raw)
	}
	return int32(value), nil // #nosec G115 -- ParseInt enforces the int32 range above.
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*corpusv1.CreateCollectionResponse, error) {
	federated, _ := strconv.ParseBool(ctx.Flag("federated"))
	resp, err := h.client.CreateCollection(context.Background(), connect.NewRequest(&corpusv1.CreateCollectionRequest{Name: ctx.Flag("name"), DefaultPrivacyClass: privacy(ctx.Flag("privacy")), Federated: federated}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create collection", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, r *corpusv1.CreateCollectionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created collection " + r.Collection.Id}}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*corpusv1.GetCollectionResponse, error) {
	resp, err := h.client.GetCollection(context.Background(), connect.NewRequest(&corpusv1.GetCollectionRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, r *corpusv1.GetCollectionResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{r.Collection.Name}, Results: []string{r.Collection.Id}}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*corpusv1.ListCollectionsResponse, error) {
	limit, err := limit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListCollections(context.Background(), connect.NewRequest(&corpusv1.ListCollectionsRequest{Limit: limit}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, r *corpusv1.ListCollectionsResponse) cliapp.ListReport {
	out := make([]string, 0, len(r.Collections))
	for _, c := range r.Collections {
		out = append(out, fmt.Sprintf("%s %s", c.Id, c.Name))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d collection(s).", len(out))}, Results: out}
}

func (h *handlers) addDocumentCall(ctx cliapp.OperationContext) (*corpusv1.AddDocumentResponse, error) {
	resp, err := h.client.AddDocument(context.Background(), connect.NewRequest(&corpusv1.AddDocumentRequest{CollectionId: ctx.Positional("collection_id"), DocumentHash: ctx.Positional("document_hash"), PrivacyClass: privacy(ctx.Flag("privacy"))}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) addDocumentReport(_ cliapp.OperationContext, r *corpusv1.AddDocumentResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Added " + r.Document.DocumentHash}}
}

func (h *handlers) listDocumentsCall(ctx cliapp.OperationContext) (*corpusv1.ListDocumentsResponse, error) {
	limit, err := limit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListDocuments(context.Background(), connect.NewRequest(&corpusv1.ListDocumentsRequest{CollectionId: ctx.Positional("collection_id"), Limit: limit}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) listDocumentsReport(_ cliapp.OperationContext, r *corpusv1.ListDocumentsResponse) cliapp.ListReport {
	out := make([]string, 0, len(r.Documents))
	for _, d := range r.Documents {
		out = append(out, fmt.Sprintf("%s %s", d.DocumentHash, d.PrivacyClass.String()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d document(s).", len(out))}, Results: out}
}

func (h *handlers) exportCall(ctx cliapp.OperationContext) (*corpusv1.ExportResponse, error) {
	resp, err := h.client.Export(context.Background(), connect.NewRequest(&corpusv1.ExportRequest{CollectionId: ctx.Positional("collection_id")}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) exportReport(_ cliapp.OperationContext, r *corpusv1.ExportResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{r.Format}, Results: []string{string(r.ArchiveJson)}}
}

func (h *handlers) importCall(ctx cliapp.OperationContext) (*corpusv1.ImportResponse, error) {
	data, err := os.ReadFile(ctx.Flag("file"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Import(context.Background(), connect.NewRequest(&corpusv1.ImportRequest{ArchiveJson: data}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) importReport(_ cliapp.OperationContext, r *corpusv1.ImportResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Imported %d document(s) into %s", r.DocumentsImported, r.Collection.Id)}}
}

func (h *handlers) pruneCall(ctx cliapp.OperationContext) (*corpusv1.PruneResponse, error) {
	dryRun, _ := strconv.ParseBool(ctx.Flag("dry-run"))
	resp, err := h.client.Prune(context.Background(), connect.NewRequest(&corpusv1.PruneRequest{DryRun: dryRun}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) pruneReport(_ cliapp.OperationContext, r *corpusv1.PruneResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Prune dry-run=%t reclaimed=%d", r.DryRun, r.ReclaimedBytes)}}
}
