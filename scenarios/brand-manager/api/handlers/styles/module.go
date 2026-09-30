package styles

import (
	"log"

	"brand-manager/internal/module"
	internalstyles "brand-manager/internal/styles"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	stylesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/styles/styles_v1connect"
)

// Module returns the styles domain's contribution to the API.
func Module(svc *internalstyles.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := stylesconnect.NewStylesServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "styles",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the styles schema.
func Schema() string { return internalstyles.Schema() }

// Endpoints describes the styles module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "styles_list_styles",
		Path:        stylesconnect.StylesServiceListContainerStylesProcedure,
		Method:      "POST",
		Summary:     "List container styles",
		Description: "Returns every container style ordered by name.",
		Category:    "styles",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"styles": "array<ContainerStyle>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
	},
	{
		ID:          "styles_get_style",
		Path:        stylesconnect.StylesServiceGetContainerStyleProcedure,
		Method:      "POST",
		Summary:     "Get a container style by id",
		Description: "Returns the container style matching the request id.",
		Category:    "styles",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"style": "ContainerStyle"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No style with that id exists"}},
	},
	{
		ID:          "styles_create_style",
		Path:        stylesconnect.StylesServiceCreateContainerStyleProcedure,
		Method:      "POST",
		Summary:     "Create a container style",
		Description: "Persists a new container style. Name is required.",
		Category:    "styles",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"style": "ContainerStyle"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing name"}},
	},
	{
		ID:          "styles_update_style",
		Path:        stylesconnect.StylesServiceUpdateContainerStyleProcedure,
		Method:      "POST",
		Summary:     "Update a container style (partial)",
		Description: "Merges non-zero fields onto the stored container style.",
		Category:    "styles",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"style": "ContainerStyle"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No style with that id exists"}},
	},
	{
		ID:          "styles_list_lines",
		Path:        stylesconnect.StylesServiceListProductLinesProcedure,
		Method:      "POST",
		Summary:     "List product lines",
		Description: "Returns every product line ordered by name.",
		Category:    "styles",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"lines": "array<ProductLine>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
	},
	{
		ID:          "styles_create_line",
		Path:        stylesconnect.StylesServiceCreateProductLineProcedure,
		Method:      "POST",
		Summary:     "Create a product line",
		Description: "Persists a new product line. Name is required.",
		Category:    "styles",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"line": "ProductLine"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing name"}},
	},
}
