package templates

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	templatesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/templates"
	templatesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/templates/templates_v1connect"

	domain "prompt-manager/internal/templates"
)

type connectHandler struct {
	templatesconnect.UnimplementedTemplatesServiceHandler
	store *domain.Store
}

func NewConnectMount(store *domain.Store) (string, http.Handler) {
	return templatesconnect.NewTemplatesServiceHandler(&connectHandler{store: store})
}

func (h *connectHandler) ListAgentFileTemplates(ctx context.Context, _ *connect.Request[templatesv1.ListAgentFileTemplatesRequest]) (*connect.Response[templatesv1.ListAgentFileTemplatesResponse], error) {
	items, err := h.store.ListAgentFileTemplates(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &templatesv1.ListAgentFileTemplatesResponse{Count: int32(len(items))}
	for _, item := range items {
		out.Templates = append(out.Templates, &templatesv1.AgentFileTemplate{Id: item.ID, Name: item.Name, Description: item.Description, FileName: item.FileName, Content: item.Content})
	}
	return connect.NewResponse(out), nil
}
