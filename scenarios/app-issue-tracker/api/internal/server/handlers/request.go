package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const (
	defaultMaxBodyBytes int64 = 10 << 20 // 10 MiB default; higher limits are declared per-handler when needed
)

var (
	ErrRequestTooLarge   = errors.New("request body too large")
	ErrInvalidJSON       = errors.New("invalid JSON payload")
	ErrUnknownFields     = errors.New("payload contains unknown fields")
	ErrUnexpectedPayload = errors.New("unexpected data after JSON payload")
)

// DecodeJSON reads the request body into dest while enforcing sane limits and
// consistent decoder behaviour across handlers.
func DecodeJSON(r *http.Request, dest interface{}, maxBytes int64) error {
	if r == nil || r.Body == nil {
		return io.EOF
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxBodyBytes
	}
	limited := &io.LimitedReader{R: r.Body, N: maxBytes + 1}
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		if limited.N <= 0 {
			return ErrRequestTooLarge
		}
		if errors.Is(err, io.EOF) {
			return ErrInvalidJSON
		}
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError
		if errors.As(err, &syntaxErr) || errors.As(err, &unmarshalTypeErr) {
			return ErrInvalidJSON
		}
		if strings.Contains(err.Error(), "unknown field") {
			return ErrUnknownFields
		}
		return err
	}

	if limited.N <= 0 {
		return ErrRequestTooLarge
	}

	// Ensure there is no trailing data after the first JSON value.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if limited.N <= 0 {
			return ErrRequestTooLarge
		}
		if err == nil {
			return ErrUnexpectedPayload
		}
		return ErrInvalidJSON
	}

	return nil
}
