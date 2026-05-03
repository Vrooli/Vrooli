// Package httpx holds the production HTTP-boundary helpers every
// handler in this scenario shares: the typed error envelope writer
// (errors.go) and the strict request-decode helper (decode.go).
//
// Distinct from internal/testutil/httpx — that sibling holds the
// test-side LiveServer harness. Both packages live under the same
// "httpx" name because they're both HTTP-boundary helpers; Go imports
// by full path, so the duplicate base name is ergonomic, not a clash.
//
// Resist generalising into a god-helper grab bag. WriteError and
// DecodeJSON are the two patterns every handler shares; new helpers
// land in their own file when (and only when) they're proven shared.
package httpx

import (
	"log"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/errors"
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
// store.ErrNoteNotFound) to (status, code, message) tuples is the
// handler's responsibility — this writer just emits.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	envelope := &errorsv1.ErrorEnvelope{
		Code:    code,
		Message: message,
	}
	body, err := protojson.Marshal(envelope)
	if err != nil {
		// protojson.Marshal on a populated ErrorEnvelope cannot fail —
		// no oneofs, no recursion, no Any fields. If it does, the
		// process is in an unrecoverable state; log and fall back to a
		// minimal JSON literal so the client still sees *something*.
		log.Printf("httpx.WriteError: protojson marshal failed: %v", err)
		body = []byte(`{"code":"internal","message":"error envelope marshal failed"}`)
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
