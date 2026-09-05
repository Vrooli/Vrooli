package metadata

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	metadatav1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/metadata"
	metadataconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/metadata/metadata_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/ogmeta"
)

type connectHandler struct {
	metadataconnect.UnimplementedMetadataServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return metadataconnect.NewMetadataServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) FetchOpenGraph(ctx context.Context, req *connect.Request[metadatav1.FetchOpenGraphRequest]) (*connect.Response[metadatav1.OpenGraphMetadata], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Get, http.MethodGet, "/og-metadata?url="+url.QueryEscape(req.Msg.GetUrl()), nil, nil, &metadatav1.OpenGraphMetadata{})
}
