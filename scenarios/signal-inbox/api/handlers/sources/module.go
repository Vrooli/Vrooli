package sources

import (
	"signal-inbox/internal/module"
	internal "signal-inbox/internal/sources"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	sourcesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/sources/sources_v1connect"
)

func Module(service *internal.Service) module.Module {
	path, handler := sourcesconnect.NewSourcesServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "sources", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		router.Handle("/api/v1/sources/archive", NewArchiveUploadHandler(service)).Methods("POST")
	}, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "sources_list_adapters", Path: sourcesconnect.SourcesServiceListAdaptersProcedure, Method: "POST", Summary: "List source adapters", Description: "Lists adapter risk tiers and durable enablement state.", Category: "sources", Request: &module.Schema{Type: "ListAdaptersRequest"}, Response: &module.Schema{Type: "ListAdaptersResponse"}},
	{ID: "sources_set_enabled", Path: sourcesconnect.SourcesServiceSetAdapterEnabledProcedure, Method: "POST", Summary: "Set adapter enabled", Description: "Explicitly enables or disables one adapter; higher-risk adapters ship disabled.", Category: "sources", Request: &module.Schema{Type: "SetAdapterEnabledRequest"}, Response: &module.Schema{Type: "SetAdapterEnabledResponse"}},
	{ID: "sources_import_archive", Path: sourcesconnect.SourcesServiceImportArchiveProcedure, Method: "POST", Summary: "Import archive", Description: "Imports an operator-supplied tier-zero archive through the shared journal capture path.", Category: "sources", Request: &module.Schema{Type: "ImportArchiveRequest"}, Response: &module.Schema{Type: "ImportArchiveResponse"}},
	{ID: "sources_upload_archive", Path: "/api/v1/sources/archive", Method: "POST", Summary: "Upload and import archive", Description: "Uploads a bounded tier-zero archive without Connect message-size limits, then imports it through the shared journal capture path.", Category: "sources", Request: &module.Schema{Type: "multipart/form-data"}, Response: &module.Schema{Type: "ImportArchiveResponse"}, RESTException: &module.RESTException{Reason: module.RESTReasonMultipartUpload, Note: "large archive bytes cannot be represented in a bounded Connect request", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "multipart/form-data", Conformance: "transport_only"}, Response: module.RESTPayload{ProtoFullName: "vrooli.signal_inbox.v1.sources.ImportArchiveResponse", Transport: "json", Conformance: "protojson"}}}},
}
