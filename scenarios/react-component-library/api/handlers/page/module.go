package page

import (
	"connectrpc.com/connect"
	"context"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	pageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page/page_v1connect"
	"react-component-library/internal/module"
	"react-component-library/internal/pageinspect"
)

type Handler struct{ Service pageinspect.Service }

func (h Handler) Inspect(ctx context.Context, req *connect.Request[pagev1.InspectRequest]) (*connect.Response[pagev1.InspectResponse], error) {
	result, err := h.Service.Inspect(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}
func Module(repoRoot string) module.Module {
	path, handler := pageconnect.NewPageServiceHandler(Handler{Service: pageinspect.Service{RepoRoot: repoRoot}})
	return module.Module{Name: "page", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: []module.EndpointDescriptor{
		{ID: "page_inspect", Path: pageconnect.PageServiceInspectProcedure, Method: "POST", Summary: "Inspect a running page and resolve its stamped nodes to library source", Category: "page"},
	}}
}
