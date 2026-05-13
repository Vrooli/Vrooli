// Package terminal owns the descriptor for the terminal multipart
// upload endpoint. The handler continues to live in api/upload_handler.go;
// this package only exposes the canonical metadata so gen-endpoints can
// validate the route under the RESTException rule. Multipart uploads
// stay REST even post-Connect because Connect's wire format does not
// natively carry multipart payloads — the upload semantics are inherent
// to the transport, not just incidentally tied to HTTP.
//
// The terminal WebSocket route at /api/v1/sessions/{id}/ws is
// intentionally not registered here yet — it ships with the final
// streams phase alongside voice/events so the streaming pattern is
// decided once.
package terminal

import "web-console/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "terminal_upload",
		Path:        "/api/v1/sessions/{id}/upload",
		Method:      "POST",
		Summary:     "Upload an image to a terminal session",
		Description: "Multipart image upload that stores the image under the session workspace so the user can reference it from the terminal by path.",
		Category:    "terminal",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Multipart/form-data upload. Stays REST even post-Connect because Connect's wire format does not natively carry multipart payloads.",
		},
	},
}
