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
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/shared"
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

// ErrorLogger receives serialization failures at this HTTP boundary. The
// caller owns the logger so tests and production can choose their sink.
type ErrorLogger interface {
	Errorf(format string, args ...any)
}

// ErrorLogFunc adapts a function-valued logger to ErrorLogger.
type ErrorLogFunc func(format string, args ...any)

func (f ErrorLogFunc) Errorf(format string, args ...any) {
	f(format, args...)
}

// WriteError serialises a proto-typed ErrorEnvelope as the response
// body and sets the status code. Content-Type is set to application/json
// because protojson.Marshal returns canonical JSON, not protobuf binary.
//
// Handlers reach for WriteError on every non-2xx path so the wire
// vocabulary stays consistent. Translation from typed sentinels (e.g.,
// notes.ErrNoteNotFound, notes.ErrInvalidNote) to (status, code,
// message) tuples is the handler's responsibility — this writer just
// emits.
func WriteError(w http.ResponseWriter, logger ErrorLogger, status int, code, message string) {
	envelope := &errorsv1.ErrorEnvelope{
		Code:    code,
		Message: message,
	}
	body, err := protojson.Marshal(envelope)
	if err != nil {
		logger.Errorf("httpx.WriteError: protojson marshal failed: %v", err)
		body = []byte(`{"code":"internal","message":"error envelope marshal failed"}`)
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// WriteProto serialises msg as proto JSON for REST endpoints whose response
// metadata is still proto-typed. Connect-RPC handlers do not use this helper.
func WriteProto(w http.ResponseWriter, logger ErrorLogger, status int, msg proto.Message) {
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		logger.Errorf("httpx.WriteProto: protojson marshal failed: %v", err)
		WriteError(w, logger, http.StatusInternalServerError, CodeInternal, "response marshal failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
