package assets

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the assets module's surface for codegen: three Connect
// RPCs plus the multipart upload and static file-serving REST exceptions.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "assets_list", Path: landingconnect.AssetsServiceListAssetsProcedure, Method: "POST",
		Summary: "List assets", Description: "Lists uploaded assets, optionally filtered by category (admin).", Category: "assets",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"category": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"assets": "Asset[]"}},
	},
	{
		ID: "assets_get", Path: landingconnect.AssetsServiceGetAssetProcedure, Method: "POST",
		Summary: "Get asset", Description: "Fetches a single asset by id (admin).", Category: "assets",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "int64"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"asset": "Asset"}},
	},
	{
		ID: "assets_delete", Path: landingconnect.AssetsServiceDeleteAssetProcedure, Method: "POST",
		Summary: "Delete asset", Description: "Deletes an asset and its files (admin).", Category: "assets",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "int64"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deleted": "bool"}},
	},
	{
		ID: "assets_upload", Path: uploadPath, Method: "POST",
		Summary: "Upload asset", Description: "Uploads an image via multipart form and generates derivatives (admin).", Category: "assets",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"file": "multipart", "category": "string", "alt_text": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"*": "created Asset"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Binary multipart/form-data upload; Connect-RPC does not carry raw file streams.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "multipart", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "landing_page_react_vite.v1.Asset", Transport: "json", Conformance: "protojson"},
			},
		},
	},
	{
		ID: "assets_serve", Path: "/api/v1/uploads/{path}", Method: "GET",
		Summary: "Serve upload", Description: "Serves uploaded file bytes publicly with long-lived cache headers.", Category: "assets",
		Response: &module.Schema{Type: "string", Properties: map[string]string{"content_type": "image/*"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonThirdPartyShape,
			Note:   "Static binary file server for uploaded media; raw bytes, not a Connect payload.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "binary", Conformance: "none"},
			},
		},
	},
}
