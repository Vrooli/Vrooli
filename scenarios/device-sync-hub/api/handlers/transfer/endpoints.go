package transfer

import (
	"device-sync-hub/internal/module"

	transferconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer/transfer_v1connect"
)

// Endpoints is the machine-readable description of the transfer module's public
// surface. The four Connect RPC paths reference generated *Procedure constants
// (renaming an RPC in transfer.proto breaks this file at compile time); the two
// byte edges are documented REST exceptions (multipart upload, binary download)
// whose response metadata stays proto-typed. The global parity test asserts
// every transfer RPC has exactly one entry here.
//
// Every authed endpoint here is gated by the DEVICE token (X-Device-Token), not
// the owner JWT — transfer is device-to-device within the owner's trust group.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "transfer_create_text",
		Path:        transferconnect.TransferServiceCreateTextItemProcedure,
		Method:      "POST",
		Summary:     "Send a text snippet",
		Description: "Stores a text snippet as an item visible to the trust group (or directed to one device). Device-token authed.",
		Category:    "transfer",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"text":             "string (required)",
			"name":             "string (optional label)",
			"retention":        "Retention (live|held|pinned; default held)",
			"target_device_id": "string (empty = broadcast)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"item": "Item"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or untrusted device token"},
			{Status: 400, Code: "invalid_argument", Description: "Empty text or invalid target device"},
			{Status: 429, Code: "resource_exhausted", Description: "Storage quota exceeded"},
		},
		Examples: []module.Example{
			{Name: "Send text", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.transfer.TransferService/CreateTextItem -H 'X-Device-Token: <token>' -H 'Content-Type: application/json' -d '{\"text\":\"hello\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub transfer send-text", Args: []string{"<text>"}},
	},
	{
		ID:          "transfer_list",
		Path:        transferconnect.TransferServiceListItemsProcedure,
		Method:      "POST",
		Summary:     "List received items",
		Description: "Returns the items the calling device may pull (broadcast, directed-to-it, or originated by it), newest first, optionally filtered by query/kind.",
		Category:    "transfer",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"query": "string (optional substring filter)",
			"kind":  "ItemKind (optional; unspecified = both)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"items": "array<Item>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or untrusted device token"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List items", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.transfer.TransferService/ListItems -H 'X-Device-Token: <token>' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub transfer list"},
	},
	{
		ID:          "transfer_get",
		Path:        transferconnect.TransferServiceGetItemProcedure,
		Method:      "POST",
		Summary:     "Get an item by id",
		Description: "Returns one item's metadata if the calling device may see it.",
		Category:    "transfer",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"item": "Item"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or untrusted device token"},
			{Status: 404, Code: "not_found", Description: "No visible item with that id"},
		},
		Examples: []module.Example{
			{Name: "Get item", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.transfer.TransferService/GetItem -H 'X-Device-Token: <token>' -d '{\"id\":\"<item-id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub transfer get", Args: []string{"<id>"}},
	},
	{
		ID:          "transfer_delete",
		Path:        transferconnect.TransferServiceDeleteItemProcedure,
		Method:      "POST",
		Summary:     "Delete an item",
		Description: "Removes an item and its blob. Any trusted device of the owner may delete.",
		Category:    "transfer",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"id": "string (deleted id)"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or untrusted device token"},
			{Status: 404, Code: "not_found", Description: "No such item for this owner"},
		},
		Examples: []module.Example{
			{Name: "Delete item", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.transfer.TransferService/DeleteItem -H 'X-Device-Token: <token>' -d '{\"id\":\"<item-id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub transfer delete", Args: []string{"<id>"}},
	},
	{
		ID:          "transfer_upload",
		Path:        "/api/v1/transfer/items",
		Method:      "POST",
		Summary:     "Upload a file",
		Description: "Uploads opaque file bytes via the documented REST multipart exception and returns proto-typed Item metadata. Device-token authed; quota-checked; generates an image thumbnail when applicable.",
		Category:    "transfer",
		Request: &module.Schema{Type: "multipart/form-data", Properties: map[string]string{
			"file":             "file (required)",
			"name":             "string (optional; defaults to filename)",
			"retention":        "string (live|held|pinned; default held)",
			"target_device_id": "string (optional; empty = broadcast)",
		}},
		Response: &module.Schema{Type: "UploadItemResponse", Properties: map[string]string{"item": "Item"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or untrusted device token"},
			{Status: 400, Code: "invalid_request", Description: "Missing/empty file or invalid multipart upload"},
			{Status: 413, Code: "quota_exceeded", Description: "Upload would exceed the owner or device storage quota"},
			{Status: 500, Code: "internal", Description: "BlobStore or metadata persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Upload file", Curl: "curl -X POST http://localhost:${API_PORT}/api/v1/transfer/items -H 'X-Device-Token: <token>' -F file=@./photo.png -F retention=held"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub transfer upload", Args: []string{"--file", "<path>"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Multipart/form-data request transport cannot be expressed in proto. The response payload is the proto-typed UploadItemResponse message.",
			ProtoPayloads: module.ProtoPayloads{
				Request:  module.RESTPayload{Transport: "multipart/form-data", Conformance: "transport_only"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.device_sync_hub.v1.transfer.UploadItemResponse", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.device_sync_hub.v1.errors.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
	{
		ID:          "transfer_download",
		Path:        "/api/v1/transfer/items/{id}/content",
		Method:      "GET",
		Summary:     "Download an item's bytes",
		Description: "Streams an item's content with its original filename (Content-Disposition). For text items the body is the snippet; ?thumb=1 streams the image thumbnail. Pulling a Live item marks it delivered (purged on the next sweep). Device-token authed.",
		Category:    "transfer",
		Response: &module.Schema{Type: "application/octet-stream", Properties: map[string]string{
			"body": "raw bytes (original filename via Content-Disposition)",
		}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or untrusted device token"},
			{Status: 404, Code: "not_found", Description: "No visible item with that id, or no thumbnail"},
		},
		Examples: []module.Example{
			{Name: "Download file", Curl: "curl -OJ http://localhost:${API_PORT}/api/v1/transfer/items/<item-id>/content -H 'X-Device-Token: <token>'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub transfer download", Args: []string{"<id>", "--out", "<path>"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Direct GET (no Connect client): response body is opaque streamed bytes with the original filename — no proto message can carry it. Metadata is served proto-typed via the GetItem Connect RPC.",
			ProtoPayloads: module.ProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "none", Conformance: "none"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.device_sync_hub.v1.errors.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}
