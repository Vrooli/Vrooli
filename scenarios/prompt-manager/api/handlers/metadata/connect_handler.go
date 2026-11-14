package metadata

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	metadatav1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/metadata"
	metadataconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/metadata/metadata_v1connect"

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
	meta, err := h.legacy.Fetch(ctx, req.Msg.GetUrl())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&metadatav1.OpenGraphMetadata{Url: meta.URL, Title: meta.Title, Description: meta.Description, Image: meta.Image, SiteName: meta.SiteName, Type: meta.Type, Favicon: meta.Favicon}), nil
}
