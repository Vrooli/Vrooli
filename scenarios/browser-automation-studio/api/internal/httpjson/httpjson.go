package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/vrooli/browser-automation-studio/config"
)

// DefaultMaxBodyBytes is the fallback maximum request body size (1MB).
// The actual limit is read from config at runtime for configurability.
const DefaultMaxBodyBytes int64 = 1 << 20

// MaxBodyBytes returns the configured maximum request body size.
// Configurable via BAS_HTTP_MAX_BODY_BYTES (default: 1MB)
func MaxBodyBytes() int64 {
	cfg := config.Load()
	if cfg != nil && cfg.HTTP.MaxBodyBytes > 0 {
		return cfg.HTTP.MaxBodyBytes
	}
	return DefaultMaxBodyBytes
}

func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	return decode(w, r, dst, false)
}

func DecodeAllowEmpty(w http.ResponseWriter, r *http.Request, dst any) error {
	return decode(w, r, dst, true)
}

func decode(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) error {
	reader := http.MaxBytesReader(w, r.Body, MaxBodyBytes())
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if allowEmpty && errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}
