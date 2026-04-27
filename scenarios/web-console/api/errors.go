package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// DOC: docs/internal/ERROR-SEMANTICS.md
// DOC: docs/internal/SEAMS.md#axis-3-error-codes--recovery-api--ui

/**
 * ╔════════════════════════════════════════════════════════════════╗
 * ║  ERROR CATEGORIES — Read before modifying                      ║
 * ║                                                                ║
 * ║  Each category has a specific recovery path. Changing these    ║
 * ║  affects UI error states and automated retry logic.            ║
 * ║                                                                ║
 * ║  Categories (4):                                               ║
 * ║    validation — User can fix input and retry                   ║
 * ║    resource_limit — User must free resources, then retry       ║
 * ║    dependency — Transient; system/agent should retry           ║
 * ║    internal — Unexpected; escalate or report                   ║
 * ║                                                                ║
 * ║  To add a new error code:                                      ║
 * ║  1. Assign it to exactly one category above                    ║
 * ║  2. Define its recovery action in the handler                  ║
 * ║  3. Update docs/internal/ERROR-SEMANTICS.md                    ║
 * ║  4. Add a test in session_handlers_test.go                     ║
 * ╚════════════════════════════════════════════════════════════════╝
 */

// ErrorResponse is the JSON body returned for all error responses.
// Clients can rely on this shape for programmatic error handling.
type ErrorResponse struct {
	Error    string `json:"error"`
	Code     string `json:"code,omitempty"`
	Category string `json:"category,omitempty"`
	Recovery string `json:"recovery,omitempty"`
	Retry    bool   `json:"retry,omitempty"`
}

// appError bundles all fields needed to produce a structured error response.
// Handlers create these to separate error semantics from HTTP mechanics.
type appError struct {
	Status   int
	Code     string
	Category string
	Message  string
	Recovery string
	Retry    bool
}

// Error codes → categories and recovery hints.
var errorCatalog = map[string]appError{
	"invalid_body": {
		Status:   http.StatusBadRequest,
		Code:     "invalid_body",
		Category: "validation",
		Message:  "Request body is not valid JSON",
		Recovery: "Check the request body format and try again",
	},
	"session_limit_reached": {
		Status:   http.StatusTooManyRequests,
		Code:     "session_limit_reached",
		Category: "resource_limit",
		Message:  "Maximum number of concurrent sessions reached. Close an existing session and try again.",
		Recovery: "Close an unused terminal session, then retry",
		Retry:    true,
	},
	"pty_spawn_failed": {
		Status:   http.StatusInternalServerError,
		Code:     "pty_spawn_failed",
		Category: "dependency",
		Message:  "Failed to start terminal process. The configured shell may not be available.",
		Recovery: "Check the configured shell path or server logs",
	},
	"internal_error": {
		Status:   http.StatusInternalServerError,
		Code:     "internal_error",
		Category: "internal",
		Message:  "An unexpected error occurred",
		Recovery: "Retry the request; if the problem persists, check server logs",
		Retry:    true,
	},
	"session_not_found": {
		Status:   http.StatusNotFound,
		Code:     "session_not_found",
		Category: "validation",
		Message:  "Session not found",
		Recovery: "The session may have ended. Open a new terminal.",
	},
	"profile_not_found": {
		Status:   http.StatusNotFound,
		Code:     "profile_not_found",
		Category: "validation",
		Message:  "Shortcut profile not found",
		Recovery: "The profile may have been deleted. Refresh the profile list.",
	},
	"session_terminated": {
		Status:   http.StatusGone,
		Code:     "session_terminated",
		Category: "dependency",
		Message:  "Session has terminated",
		Recovery: "The terminal process exited. Open a new terminal.",
	},
	"ai_provider_unavailable": {
		Status:   http.StatusServiceUnavailable,
		Code:     "ai_provider_unavailable",
		Category: "dependency",
		Message:  "AI command generation is currently unavailable",
		Recovery: "Check that Ollama is running or OPENROUTER_API_KEY is set",
		Retry:    true,
	},
	"invalid_policy": {
		Status:   http.StatusBadRequest,
		Code:     "invalid_policy",
		Category: "validation",
		Message:  "Invalid expiration policy",
		Recovery: "Use mode 'never', 'preset' (with 1h/8h/24h), or 'custom' (with a Go duration like 30m)",
	},
	"backend_unavailable": {
		Status:   http.StatusUnprocessableEntity,
		Code:     "backend_unavailable",
		Category: "dependency",
		Message:  "Requested session backend is not available",
		Recovery: "Use 'standard' backend or install tmux",
	},
	"backend_unknown": {
		Status:   http.StatusUnprocessableEntity,
		Code:     "backend_unknown",
		Category: "validation",
		Message:  "Unknown session backend",
		Recovery: "Use 'standard' or 'persistent'",
	},
	"voice_unavailable": {
		Status:   http.StatusServiceUnavailable,
		Code:     "voice_unavailable",
		Category: "dependency",
		Message:  "Voice transcription is currently unavailable",
		Recovery: "Ensure Whisper is running (resource whisper on port 8090)",
		Retry:    true,
	},
	"voice_transcribe_failed": {
		Status:   http.StatusBadGateway,
		Code:     "voice_transcribe_failed",
		Category: "dependency",
		Message:  "Voice transcription request failed",
		Recovery: "Try again. If the problem persists, check Whisper logs.",
		Retry:    true,
	},
	"speaker_verification_unavailable": {
		Status:   http.StatusServiceUnavailable,
		Code:     "speaker_verification_unavailable",
		Category: "dependency",
		Message:  "Speaker verification is currently unavailable",
		Recovery: "Ensure the speaker-verification resource is running and healthy",
		Retry:    true,
	},
	"speaker_verification_failed": {
		Status:   http.StatusBadGateway,
		Code:     "speaker_verification_failed",
		Category: "dependency",
		Message:  "Speaker verification request failed",
		Recovery: "Retry the request; if the problem persists, check speaker-verification logs",
		Retry:    true,
	},
	"speaker_enrollment_failed": {
		Status:   http.StatusBadGateway,
		Code:     "speaker_enrollment_failed",
		Category: "dependency",
		Message:  "Speaker enrollment failed",
		Recovery: "Retry enrollment in a quieter environment or check speaker-verification logs",
		Retry:    true,
	},
	"speaker_profile_clear_failed": {
		Status:   http.StatusConflict,
		Code:     "speaker_profile_clear_failed",
		Category: "validation",
		Message:  "Failed to clear the active speaker profile",
		Recovery: "Refresh settings and try again",
	},
	"upload_too_large": {
		Status:   http.StatusRequestEntityTooLarge,
		Code:     "upload_too_large",
		Category: "validation",
		Message:  "File exceeds maximum upload size",
		Recovery: "Reduce the file size and try again (max 20MB)",
	},
	"group_not_found": {
		Status:   http.StatusNotFound,
		Code:     "group_not_found",
		Category: "validation",
		Message:  "Tab group not found",
		Recovery: "The group may have been deleted. Refresh the workspace.",
	},
	"pane_not_found": {
		Status:   http.StatusNotFound,
		Code:     "pane_not_found",
		Category: "validation",
		Message:  "Workspace pane not found",
		Recovery: "The pane may have been removed. Refresh the workspace.",
	},
	"invalid_upload_type": {
		Status:   http.StatusBadRequest,
		Code:     "invalid_upload_type",
		Category: "validation",
		Message:  "Unsupported file type",
		Recovery: "Upload an image file (PNG, JPEG, GIF, WebP, or SVG)",
	},
	"file_reference_invalid": {
		Status:   http.StatusBadRequest,
		Code:     "file_reference_invalid",
		Category: "validation",
		Message:  "Invalid file reference",
		Recovery: "Use a valid file path and try again",
	},
	"file_reference_not_found": {
		Status:   http.StatusNotFound,
		Code:     "file_reference_not_found",
		Category: "validation",
		Message:  "Referenced file not found",
		Recovery: "Confirm the file path is correct relative to the project or current session directory",
	},
	"file_reference_not_allowed": {
		Status:   http.StatusForbidden,
		Code:     "file_reference_not_allowed",
		Category: "validation",
		Message:  "Referenced path is outside allowed roots",
		Recovery: "Open files from the active project workspace only",
	},
	"file_reference_too_large": {
		Status:   http.StatusRequestEntityTooLarge,
		Code:     "file_reference_too_large",
		Category: "validation",
		Message:  "Referenced file is too large to preview",
		Recovery: "Open a smaller file or inspect it directly in the terminal",
	},
	"file_reference_unresolvable": {
		Status:   http.StatusUnprocessableEntity,
		Code:     "file_reference_unresolvable",
		Category: "validation",
		Message:  "Referenced file could not be resolved",
		Recovery: "Use an absolute path or a path relative to the active session directory",
	},
	"file_reference_not_previewable": {
		Status:   http.StatusUnsupportedMediaType,
		Code:     "file_reference_not_previewable",
		Category: "validation",
		Message:  "Referenced file cannot be previewed",
		Recovery: "Open a markdown or text/code file instead",
	},
	"unauthorized": {
		Status:   http.StatusUnauthorized,
		Code:     "unauthorized",
		Category: "validation",
		Message:  "Invalid or missing authentication token",
		Recovery: "Provide a valid authentication token",
	},
	"tts_unavailable": {
		Status:   http.StatusServiceUnavailable,
		Code:     "tts_unavailable",
		Category: "dependency",
		Message:  "Kokoro TTS is not available",
		Recovery: "Ensure the Kokoro resource is running",
		Retry:    true,
	},
	"tts_input_required": {
		Status:   http.StatusBadRequest,
		Code:     "tts_input_required",
		Category: "validation",
		Message:  "TTS input text is required",
		Recovery: "Provide non-empty text in the 'input' field",
	},
	"tts_input_too_long": {
		Status:   http.StatusBadRequest,
		Code:     "tts_input_too_long",
		Category: "validation",
		Message:  "TTS input exceeds maximum length",
		Recovery: "Reduce input to 5000 characters or fewer",
	},
	"tts_invalid_format": {
		Status:   http.StatusBadRequest,
		Code:     "tts_invalid_format",
		Category: "validation",
		Message:  "Unsupported audio response format",
		Recovery: "Use mp3, wav, opus, or flac",
	},
	"tts_synthesis_failed": {
		Status:   http.StatusBadGateway,
		Code:     "tts_synthesis_failed",
		Category: "dependency",
		Message:  "TTS synthesis request failed",
		Recovery: "Try again; if the problem persists, check Kokoro logs",
		Retry:    true,
	},
	"tts_voice_list_failed": {
		Status:   http.StatusBadGateway,
		Code:     "tts_voice_list_failed",
		Category: "dependency",
		Message:  "Failed to list TTS voices",
		Recovery: "Try again; if the problem persists, check Kokoro logs",
		Retry:    true,
	},
	"tts_auto_disabled": {
		Status:   http.StatusConflict,
		Code:     "tts_auto_disabled",
		Category: "validation",
		Message:  "Auto-TTS is disabled",
		Recovery: "Enable Auto-speak AI responses in Settings before retrying",
	},
	"tts_target_missing": {
		Status:   http.StatusNotFound,
		Code:     "tts_target_missing",
		Category: "validation",
		Message:  "No terminal session was available for TTS routing",
		Recovery: "Keep the owning Claude or Codex terminal open so web-console can route the TTS candidate back to it",
	},
	"tts_hook_not_registered": {
		Status:   http.StatusServiceUnavailable,
		Code:     "tts_hook_not_registered",
		Category: "dependency",
		Message:  "Claude Stop hook is not registered",
		Recovery: "Restart web-console through its lifecycle or re-register the TTS hook",
	},
	"tts_browser_audio_blocked": {
		Status:   http.StatusConflict,
		Code:     "tts_browser_audio_blocked",
		Category: "validation",
		Message:  "Browser audio is blocked until the page receives a user interaction",
		Recovery: "Click or press a key in the page, then retry playback",
	},
	"tts_playback_failed": {
		Status:   http.StatusBadGateway,
		Code:     "tts_playback_failed",
		Category: "dependency",
		Message:  "TTS playback failed",
		Recovery: "Retry playback or switch to another backend",
		Retry:    true,
	},
	"tts_fallback_used": {
		Status:   http.StatusOK,
		Code:     "tts_fallback_used",
		Category: "dependency",
		Message:  "A fallback TTS backend was used",
		Recovery: "Review backend status to restore the preferred backend if needed",
	},
}

// writeJSON encodes data as a JSON response with the given HTTP status code.
// Logs encoding errors rather than propagating them, since the response is
// already partially written by the time encoding fails.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJSON: encode response: %v", err)
	}
}

// writeAppError writes a structured JSON error response from an appError.
func writeAppError(w http.ResponseWriter, ae appError) {
	writeJSON(w, ae.Status, ErrorResponse{
		Error:    ae.Message,
		Code:     ae.Code,
		Category: ae.Category,
		Recovery: ae.Recovery,
		Retry:    ae.Retry,
	})
}

// writeCatalogError writes a structured JSON error response. It looks up the
// given code in errorCatalog for the HTTP status, category, and recovery hint.
// The message parameter overrides the catalog default (useful for adding context
// like session IDs). Unknown codes fall back to HTTP 500 with generic recovery.
func writeCatalogError(w http.ResponseWriter, code, message string) {
	ae, ok := errorCatalog[code]
	if !ok {
		ae = appError{
			Status:   http.StatusInternalServerError,
			Category: "internal",
			Recovery: "Retry the request; if the problem persists, check server logs",
		}
	}
	ae.Code = code
	ae.Message = message
	writeAppError(w, ae)
}

// decodeJSON reads r.Body into dst, writing an "invalid_body" error and
// returning false when decoding fails. Callers should return on false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
			writeCatalogError(w, "invalid_body", "Request body is not valid JSON")
			return false
		}
	}
	return true
}

// lookupSession extracts the {id} path variable, looks up the session, and
// writes a "session_not_found" error if it doesn't exist. Returns nil when
// the session was not found (callers should return immediately).
func (s *Server) lookupSession(w http.ResponseWriter, r *http.Request) *Session {
	id := mux.Vars(r)["id"]
	sess, ok := s.sessions.Get(id)
	if !ok {
		writeCatalogError(w, "session_not_found",
			"Session "+sanitizeID(id)+" not found")
		return nil
	}
	return sess
}

// sanitizeID truncates and cleans an ID for safe inclusion in user-facing messages.
// Prevents log injection and overly long values in error responses.
func sanitizeID(id string) string {
	// Strip control characters
	clean := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, id)
	if len(clean) > 40 {
		clean = clean[:40] + "..."
	}
	return clean
}
