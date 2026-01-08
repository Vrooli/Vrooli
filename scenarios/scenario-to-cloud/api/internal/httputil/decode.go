package httputil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// DecodeJSON decodes JSON from the given reader with a byte limit.
// It enforces strict parsing: unknown fields cause an error, and
// multiple JSON values are rejected.
func DecodeJSON[T any](r io.Reader, maxBytes int64) (T, error) {
	var zero T
	if r == nil {
		return zero, errors.New("missing request body")
	}
	body := io.LimitReader(r, maxBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&zero); err != nil {
		return zero, err
	}
	if err := dec.Decode(&struct{}{}); err == nil {
		return zero, errors.New("unexpected extra JSON values")
	} else if !errors.Is(err, io.EOF) {
		return zero, err
	}
	return zero, nil
}

// DecodeRequestBody decodes a JSON request body into the provided pointer.
// Returns false and writes an error response if decoding fails.
func DecodeRequestBody[T any](w http.ResponseWriter, r *http.Request, dest *T) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		WriteAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Invalid request body",
			Hint:    err.Error(),
		})
		return false
	}
	return true
}
