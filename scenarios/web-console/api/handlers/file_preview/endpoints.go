package file_preview

import (
	"web-console/internal/module"

	filepreviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview/file_preview_v1connect"
)

// Endpoints describes the file-preview module's public surface: two Connect
// RPCs (resolve + bounded text content) plus the opaque-id blob/range route.
// The blob route is a sanctioned REST exception (reason ops_probe): a byte-
// range stream consumed directly by native <img>/<video>/<audio>/<iframe>
// elements is browser-native and cannot be a Connect call — the same category
// as terminal_ws. Bytes never travel through Connect.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "file_preview_resolve",
		Path:        filepreviewconnect.FilePreviewServiceResolveProcedure,
		Method:      "POST",
		Summary:     "Resolve a path into a preview target",
		Description: "Translates a path string (with optional :line suffix or file:// scheme) into rich preview metadata: canonical path, classification, capability flags, and an opaque short-lived preview id the blob route serves bytes against.",
		Category:    "file_preview",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"preview_id":             "string",
				"resolved_path":          "string",
				"preview_kind":           "PreviewKind",
				"mime_type":              "string",
				"size_bytes":             "int64",
				"can_preview":            "bool",
				"can_download":           "bool",
				"supports_range":         "bool",
				"text_content_available": "bool",
				"blob_url":               "string",
				"warnings":               "[]string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing or malformed path"},
			{Status: 403, Code: "permission_denied", Description: "Referenced file is not readable"},
			{Status: 404, Code: "not_found", Description: "Session or file not found"},
			{Status: 412, Code: "failed_precondition", Description: "Referenced path is a special file (socket, FIFO, device) or otherwise not previewable"},
		},
	},
	{
		ID:          "file_preview_text",
		Path:        filepreviewconnect.FilePreviewServiceGetTextContentProcedure,
		Method:      "POST",
		Summary:     "Read bounded text content for a preview id",
		Description: "Returns up to 1 MiB of UTF-8 text for a text-kind preview (markdown/code/text/csv/diff). Media and images stream from the blob route instead.",
		Category:    "file_preview",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"resolved_path": "string",
				"preview_kind":  "PreviewKind",
				"mime_type":     "string",
				"content":       "string",
				"truncated":     "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing session_id or preview_id"},
			{Status: 404, Code: "not_found", Description: "Unknown/expired preview id or missing file"},
			{Status: 412, Code: "failed_precondition", Description: "Preview kind has no inline text content"},
		},
	},
	{
		ID:          "file_preview_list_directory",
		Path:        filepreviewconnect.FilePreviewServiceListDirectoryProcedure,
		Method:      "POST",
		Summary:     "List one page of a resolved directory",
		Description: "Returns a bounded, sorted page of a directory previously resolved by Resolve, keyed by its opaque preview id. Entries are classified from their extension alone so a page costs no file reads. Pages are capped (1000 max, 200 default) and the scan stops at 50000 entries; size and date sorts are limited to 5000 entries and downgrade to a name sort with a warning above that. The continuation token pins the directory mtime, so a directory that changes mid-walk aborts rather than silently skipping entries.",
		Category:    "file_preview",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"resolved_path":   "string",
				"parent_path":     "string",
				"entries":         "[]DirectoryEntry",
				"total_entries":   "int32",
				"truncated":       "bool",
				"next_page_token": "string",
				"effective_sort":  "DirectorySort",
				"warnings":        "[]string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing session_id/preview_id, or a page token that does not match the requested directory, sort, or filter"},
			{Status: 403, Code: "permission_denied", Description: "Directory is not readable"},
			{Status: 404, Code: "not_found", Description: "Unknown/expired preview id or the directory no longer exists"},
			{Status: 409, Code: "aborted", Description: "Directory changed between pages; the listing must be reloaded"},
			{Status: 412, Code: "failed_precondition", Description: "Preview id does not name a directory"},
		},
	},
	{
		ID:          "file_preview_blob",
		Path:        "/api/v1/sessions/{id}/file-previews/{previewId}/blob",
		Method:      "GET",
		Summary:     "Stream the bytes of a resolved preview",
		Description: "Serves the file bytes for an opaque, session-bound preview id with HTTP Range support (206/Content-Range/Accept-Ranges, 416 on bad range) and HEAD. Content-Type comes from the resolved metadata; no-store + nosniff. Re-stats the file and returns 409 if it changed since resolve. Consumed directly by native <img>/<video>/<audio>/<iframe>.",
		Category:    "file_preview",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonStreamUpgrade,
			Note:   "Byte-range blob stream consumed by native browser media elements without a generated client — the same browser-native-transport category as terminal_ws. Connect's wire format cannot express opaque byte streaming with Range. The opaque preview id (never a raw path) is issued by FilePreviewService.Resolve and bound to the session.",
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "Unknown/expired preview id or session mismatch"},
			{Status: 409, Code: "conflict", Description: "File changed since the preview was resolved; reopen to refresh"},
			{Status: 416, Code: "range_not_satisfiable", Description: "Requested byte range is invalid"},
		},
	},
}
