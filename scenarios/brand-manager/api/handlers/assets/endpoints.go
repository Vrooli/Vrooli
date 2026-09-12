package assets

import (
	"brand-manager/internal/module"

	assetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets/assets_v1connect"
)

// Endpoints is the machine-readable description of the assets module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in assets.proto breaks this file at compile
// time. The global parity test in modules/registry_test.go asserts every rpc
// has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "assets_list",
		Path:        assetsconnect.AssetsServiceListAssetsProcedure,
		Method:      "POST",
		Summary:     "List brand assets",
		Description: "Returns asset catalog entries ordered newest-uploaded first, optionally filtered to one brand.",
		Category:    "assets",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"brand_id": "string (optional filter; empty = all)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"assets": "array<Asset>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
		Examples:    []module.Example{{Name: "List assets", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assets.AssetsService/ListAssets -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "assets_upload",
		Path:        assetsconnect.AssetsServiceUploadAssetProcedure,
		Method:      "POST",
		Summary:     "Upload (or replace) a brand asset",
		Description: "Writes the file bytes and upserts the catalog row keyed by (brand_id, filename). Re-uploading the same filename replaces the bytes and keeps the asset id. The brand must exist, the mime type must be an allowed image type, and the payload must be non-empty and within 32 MiB.",
		Category:    "assets",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":  "string (required, must exist)",
			"filename":  "string (required, bare basename)",
			"mime_type": "string (optional; inferred from extension when empty)",
			"content":   "bytes (required, non-empty, <= 32 MiB)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"asset": "Asset"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing/invalid field, unknown brand, unsupported mime, or oversized payload"},
			{Status: 500, Code: "internal", Description: "Blob write or repository failure"},
		},
		Examples: []module.Example{{Name: "Upload asset", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assets.AssetsService/UploadAsset -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"filename\":\"logo.png\",\"content\":\"<base64>\"}'"}},
	},
	{
		ID:          "assets_get",
		Path:        assetsconnect.AssetsServiceGetAssetProcedure,
		Method:      "POST",
		Summary:     "Get an asset catalog entry by id",
		Description: "Returns the asset metadata matching the request id (no bytes; use DownloadAsset for bytes).",
		Category:    "assets",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"asset": "Asset"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No asset with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{{Name: "Get asset", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assets.AssetsService/GetAsset -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"}},
	},
	{
		ID:          "assets_download",
		Path:        assetsconnect.AssetsServiceDownloadAssetProcedure,
		Method:      "POST",
		Summary:     "Download an asset's bytes",
		Description: "Returns the stored file bytes plus its filename and mime type.",
		Category:    "assets",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"filename":  "string",
			"mime_type": "string",
			"content":   "bytes",
		}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No asset with that id exists"},
			{Status: 500, Code: "internal", Description: "Blob read or repository failure"},
		},
		Examples: []module.Example{{Name: "Download asset", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assets.AssetsService/DownloadAsset -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"}},
	},
	{
		ID:          "assets_delete",
		Path:        assetsconnect.AssetsServiceDeleteAssetProcedure,
		Method:      "POST",
		Summary:     "Delete an asset (idempotent)",
		Description: "Removes the catalog row and best-effort removes the on-disk file. Deleting a missing asset returns success.",
		Category:    "assets",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository write failure"}},
		Examples:    []module.Example{{Name: "Delete asset", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assets.AssetsService/DeleteAsset -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"}},
	},
}
