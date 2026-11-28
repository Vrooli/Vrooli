// Package httprest is the one JSON envelope for the scenario's REST surfaces.
//
// Authoring is Connect-RPC; REST exists only where the browser needs a native
// transport (binary image bytes, uploads, SSE). Those routes still answer a
// failure in the same shape Connect clients see, so one client-side reader can
// interpret any error:
//
//	{"error": {"code": "not_found", "message": "workspace not found"}}
//
// Keeping the writer here (instead of one copy per handler package) is what
// makes that a contract rather than a convention.
package httprest

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// WriteJSON writes one JSON body with its status. Encoding errors cannot be
// recovered after the header is sent, so they are deliberately ignored: a
// caller has already chosen a status and the payload is always a simple map or
// a struct this service built.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError writes the canonical error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

// EtagMatches reports whether an If-None-Match header names the given entity
// tag. It accepts the exact id, a quoted or weak form, and `*` for any existing
// entity, because browsers resend whatever was stored verbatim.
func EtagMatches(header, etag string) bool {
	for _, part := range strings.Split(header, ",") {
		candidate := strings.TrimSpace(part)
		candidate = strings.TrimPrefix(candidate, "W/")
		candidate = strings.Trim(candidate, `"`)
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

// DecodeBase64Data decodes the `dataBase64` field of a JSON image upload. It
// accepts a bare base64 string or a data URL, and reports a message the client
// can show rather than an opaque base64 error.
func DecodeBase64Data(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "data:") {
		if comma := strings.Index(value, ","); comma >= 0 {
			value = value[comma+1:]
		}
	}
	if value == "" {
		return nil, errors.New("dataBase64 is required")
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("dataBase64 is not valid base64")
	}
	return decoded, nil
}
