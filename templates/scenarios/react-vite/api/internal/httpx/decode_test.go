package httpx_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"{{SCENARIO_ID}}/internal/httpx"
)

type sampleBody struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// TestDecodeJSON pins the canonical request-decode contract:
//
//   - happy path returns the parsed value with a nil error
//   - malformed JSON surfaces a wrapped error so handlers can map it to
//     a 400 ErrorEnvelope without re-parsing the message
//   - unknown fields are rejected — the strict-input default that
//     prevents schema drift from going unnoticed
//   - an empty body errors out cleanly rather than producing the zero
//     value (which would be ambiguous with a legitimately-zero payload)
func TestDecodeJSON(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":"hello","body":"world"}`))
		got, err := httpx.DecodeJSON[sampleBody](req)
		require.NoError(t, err)
		require.Equal(t, "hello", got.Title)
		require.Equal(t, "world", got.Body)
	})

	t.Run("malformed json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":`))
		_, err := httpx.DecodeJSON[sampleBody](req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "decode JSON")
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":"x","mystery":"y"}`))
		_, err := httpx.DecodeJSON[sampleBody](req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "mystery")
	})

	t.Run("empty body errors", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(``))
		_, err := httpx.DecodeJSON[sampleBody](req)
		require.Error(t, err)
	})
}
