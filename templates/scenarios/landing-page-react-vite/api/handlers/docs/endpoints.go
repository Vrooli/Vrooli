package docs

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the docs module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "docs_get_tree", Path: landingconnect.DocsServiceGetDocsTreeProcedure, Method: "POST",
		Summary: "Get docs tree", Description: "Returns the markdown docs tree (admin).", Category: "docs",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"entries": "DocEntry[]"}},
	},
	{
		ID: "docs_get_content", Path: landingconnect.DocsServiceGetDocContentProcedure, Method: "POST",
		Summary: "Get doc content", Description: "Returns a single document's content by path (admin).", Category: "docs",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"path": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"content": "string", "title": "string"}},
	},
}
