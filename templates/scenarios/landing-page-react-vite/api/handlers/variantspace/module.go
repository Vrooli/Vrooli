package variantspace

import (
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internalvariantspace "landing-page-react-vite-api/internal/variantspace"
)

// Module returns the variant_space domain's contribution: the
// VariantSpaceService Connect-RPC handler mounted on the shared router.
func Module(space *internalvariantspace.Space, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewVariantSpaceServiceHandler(NewConnectHandler(Deps{Space: space, Logger: logger}))
	return module.Module{
		Name: "variant_space",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the variant space is file-backed and owns no tables.
func Schema() string { return "" }
