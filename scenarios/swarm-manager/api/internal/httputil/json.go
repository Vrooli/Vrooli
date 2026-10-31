package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DecodeJSONStrict decodes a single JSON object from the request body and
// rejects unknown fields or trailing data.
func DecodeJSONStrict(r *http.Request, target any) error {
	return DecodeJSONStrictBounded(r, target, 0)
}

// DecodeJSONStrictBounded is DecodeJSONStrict with a maximum body size.
// A zero or negative `maxBytes` disables the limit (behaves like
// DecodeJSONStrict). Use a positive limit as defense-in-depth against
// oversized or malformed bodies on user-facing endpoints.
func DecodeJSONStrictBounded(r *http.Request, target any, maxBytes int64) error {
	defer r.Body.Close()

	body := r.Body
	if maxBytes > 0 {
		body = http.MaxBytesReader(nil, body, maxBytes)
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("unexpected trailing JSON content")
	}

	return nil
}
