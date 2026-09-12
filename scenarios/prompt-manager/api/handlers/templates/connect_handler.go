package templates

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	templatesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/templates"
	templatesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/templates/templates_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/templates"
)

type connectHandler struct {
	templatesconnect.UnimplementedTemplatesServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return templatesconnect.NewTemplatesServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListAgentFileTemplates(ctx context.Context, req *connect.Request[templatesv1.ListAgentFileTemplatesRequest]) (*connect.Response[templatesv1.ListAgentFileTemplatesResponse], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.ListAgentFileTemplates, http.MethodGet, "/agent-file-templates", nil, nil, &templatesv1.ListAgentFileTemplatesResponse{})
}
