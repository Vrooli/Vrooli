package documents

import (
	documentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/documents/documents_v1connect"
	"persona/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "documents_bind", Path: documentsconnect.DocumentsServiceBindDocumentProcedure, Method: "POST", Summary: "Bind a document-manager document", Category: "documents"},
	{ID: "documents_list", Path: documentsconnect.DocumentsServiceListBindingsProcedure, Method: "POST", Summary: "List document bindings", Category: "documents"},
	{ID: "documents_release", Path: documentsconnect.DocumentsServiceReleaseIntoHandoffProcedure, Method: "POST", Summary: "Release a document into a named handoff", Category: "documents"},
}
