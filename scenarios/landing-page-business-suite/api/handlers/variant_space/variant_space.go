// Package variant_space owns verbatim variant-space Connect transport.
package variant_space

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

type Handler struct {
	json func() []byte
}

func NewHandler(json func() []byte) *Handler {
	return &Handler{json: json}
}

func (h *Handler) GetVariantSpace(context.Context, *connect.Request[lpbsv1.GetVariantSpaceRequest]) (*connect.Response[lpbsv1.GetVariantSpaceResponse], error) {
	return connect.NewResponse(&lpbsv1.GetVariantSpaceResponse{RawJson: append([]byte(nil), h.json()...)}), nil
}

// RegisterRoutes mounts VariantSpaceService at its generated Connect path.
func RegisterRoutes(router *mux.Router, json func() []byte) {
	path, handler := lpbsconnect.NewVariantSpaceServiceHandler(NewHandler(json))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
