// Package terminal owns the descriptor for the terminal multipart
// upload endpoint. The handler continues to live in api/upload_handler.go;
// this package only exposes the canonical metadata so gen-endpoints can
// validate the route under the RESTException rule. Multipart uploads
// stay REST even post-Connect because Connect's wire format does not
// natively carry multipart payloads — the upload semantics are inherent
// to the transport, not just incidentally tied to HTTP.
//
// Both routes are owned by handlers/terminal's Module() in module.go;
// this file is the canonical metadata source for gen-endpoints.
package terminal

import "web-console/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "terminal_get_screen",
		Path:        "/vrooli.web_console.v1.terminal.TerminalService/GetScreen",
		Method:      "POST",
		Summary:     "Read the decoded screen of a session",
		Description: "Connect-RPC. Returns the cell grid, cursor, alt-buffer flag, scrollback line count, and plain-text rendering of the active screen.",
		Category:    "terminal",
	},
	{
		ID:          "terminal_send_input",
		Path:        "/vrooli.web_console.v1.terminal.TerminalService/SendInput",
		Method:      "POST",
		Summary:     "Send programmatic input to a session",
		Description: "Connect-RPC. Accepts text, named keys (resolved via the active KeyMap), or raw bytes; routes through the single Session SendInput seam.",
		Category:    "terminal",
	},
	{
		ID:          "terminal_wait_idle",
		Path:        "/vrooli.web_console.v1.terminal.TerminalService/WaitIdle",
		Method:      "POST",
		Summary:     "Block until a session is idle",
		Description: "Connect-RPC. Blocks until the PTY has produced no output for quiet_window, or until timeout elapses, or until the session exits.",
		Category:    "terminal",
	},
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
	{
		ID:          "terminal_ws",
		Path:        "/api/v1/sessions/{id}/ws",
		Method:      "GET",
		Summary:     "Terminal WebSocket bridge",
		Description: "WebSocket upgrade endpoint for xterm.js. Streams stdin/stdout/resize/keepalive frames between the browser and the per-session PTY (or tmux pane). Migration to Connect server-streaming is deferred to the final streams phase.",
		Category:    "terminal",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonStreamUpgrade,
			Note:   "xterm.js requires a raw WebSocket upgrade — Connect-RPC cannot express that handshake. Stays REST until a Connect server-streaming RPC replaces it.",
		},
	},
}
