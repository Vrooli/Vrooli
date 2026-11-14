// Package httpx holds the production HTTP-boundary helpers REST handlers
// in this scenario share: the typed error envelope writer and the proto
// response writer for deliberate REST exceptions.
//
// Distinct from internal/testutil/httpx — that sibling holds the
// test-side LiveServer harness. Both packages live under the same
// "httpx" name because they're both HTTP-boundary helpers; Go imports
// by full path, so the duplicate base name is ergonomic, not a clash.
//
// Resist generalising into a god-helper grab bag. Connect-RPC owns
// proto-typed unary JSON. Helpers here are for deliberate REST edges
// such as multipart uploads.
package httpx

import (
	"client-rehearsal/internal/logx"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/shared"
)

// Canonical error codes. Handlers reach for these constants rather than
// open-coding the strings so the wire-side vocabulary stays narrow.
// Add new codes here as the API surface grows; document each one.
const (
	// CodeInvalidRequest covers all client-side parse / validation
	// failures (malformed JSON, missing required field, unknown field).
	CodeInvalidRequest = "invalid_request"

	// CodeNotFound is the canonical 404 code. Use when a resource
	// identifier resolves to nothing rather than for "endpoint missing".
	CodeNotFound = "not_found"

	// CodeInternal is the catch-all 500 code. Pair with a server-side
	// log line carrying the underlying error; the wire message stays
	// human-safe.
	CodeInternal = "internal"
)

// WriteError serialises a proto-typed ErrorEnvelope as the response
// body and sets the status code. Content-Type is set to application/json
// because protojson.Marshal returns canonical JSON, not protobuf binary.
//
// Handlers reach for WriteError on every non-2xx path so the wire
// vocabulary stays consistent. Translation from typed sentinels (e.g.,
// notes.ErrNoteNotFound, notes.ErrInvalidNote) to (status, code,
// message) tuples is the handler's responsibility — this writer just
// emits.
func WriteError(w http.ResponseWriter, status int, code, message string, logger logx.Logger) {
	if logger == nil {
		panic("httpx.WriteError requires Logger")
	}
	envelope := &errorsv1.ErrorEnvelope{
		Code:    code,
		Message: message,
	}
	body, err := protojson.Marshal(envelope)
	if err != nil {
		logger.Printf("httpx.WriteError: protojson marshal failed: %v", err)
		body = []byte(`{"code":"internal","message":"error envelope marshal failed"}`)
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// WriteProto serialises msg as proto JSON for REST endpoints whose response
// metadata is still proto-typed. Connect-RPC handlers do not use this helper.
func WriteProto(w http.ResponseWriter, status int, msg proto.Message, logger logx.Logger) {
	if logger == nil {
		panic("httpx.WriteProto requires Logger")
	}
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		logger.Printf("httpx.WriteProto: protojson marshal failed: %v", err)
		WriteError(w, http.StatusInternalServerError, CodeInternal, "response marshal failed", logger)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
