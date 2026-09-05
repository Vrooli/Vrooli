package sources

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/sources"
	sourcesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/sources/sources_v1connect"
)

type handlers struct {
	client sourcesconnect.SourcesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: sourcesconnect.NewSourcesServiceClient(client, base)}
}
func (h *handlers) listCall(_ cliapp.OperationContext) (*sourcesv1.ListAdaptersResponse, error) {
	response, err := h.client.ListAdapters(context.Background(), connect.NewRequest(&sourcesv1.ListAdaptersRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list source adapters", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, response *sourcesv1.ListAdaptersResponse) cliapp.ListReport {
	rows := make([]string, 0, len(response.Adapters))
	for _, adapter := range response.Adapters {
		rows = append(rows, fmt.Sprintf("%s tier=%d enabled=%t disabled_reason=%q", adapter.AdapterId, adapter.RiskTier, adapter.Enabled, adapter.DisabledReason))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d adapter(s).", len(rows))}, ResultsHeading: "Adapters", Results: rows}
}
func (h *handlers) enableCall(ctx cliapp.OperationContext) (*sourcesv1.SetAdapterEnabledResponse, error) {
	enabled := ctx.Flag("enabled") == "true"
	response, err := h.client.SetAdapterEnabled(context.Background(), connect.NewRequest(&sourcesv1.SetAdapterEnabledRequest{AdapterId: ctx.Positional("adapter-id"), Enabled: enabled}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set adapter enabled", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) enableReport(_ cliapp.OperationContext, response *sourcesv1.SetAdapterEnabledResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Adapter %s enabled=%t", response.Adapter.AdapterId, response.Adapter.Enabled)}}
}
func (h *handlers) importCall(ctx cliapp.OperationContext) (*sourcesv1.ImportArchiveResponse, error) {
	content, err := os.ReadFile(ctx.Flag("file"))
	if err != nil {
		return nil, fmt.Errorf("read import file: %w", err)
	}
	response, err := h.client.ImportArchive(context.Background(), connect.NewRequest(&sourcesv1.ImportArchiveRequest{AdapterId: ctx.Flag("adapter-id"), Content: content}))
	if err != nil {
		return nil, cliapp.WrapAPIError("import archive", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) importReport(_ cliapp.OperationContext, response *sourcesv1.ImportArchiveResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Import created=%d duplicated=%d failed=%d", response.Result.Created, response.Result.Duplicated, response.Result.Failed)}}
}
