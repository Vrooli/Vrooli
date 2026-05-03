package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DecodeJSON parses the request body into a value of type T. The
// decoder is configured with DisallowUnknownFields so the API rejects
// payloads carrying fields the schema hasn't caught up to — strict
// input by default, per the greenfield rule.
//
// Returns the zero value plus a wrapped error on parse failure;
// handlers translate that error into a 400 ErrorEnvelope via
// WriteError(w, 400, CodeInvalidRequest, err.Error()).
//
// Scenarios that need leniency (forwarding raw blobs, accepting
// future-version payloads) skip this helper and reach for
// json.NewDecoder directly — the pattern stays opt-in rather than
// opt-out.
func DecodeJSON[T any](r *http.Request) (T, error) {
	var v T
	if r.Body == nil {
		return v, fmt.Errorf("decode JSON: empty request body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("decode JSON: %w", err)
	}
	return v, nil
}
