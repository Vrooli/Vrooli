package httpx_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/health"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"

	"{{SCENARIO_ID}}/internal/httpx"
)

func TestDecodeProtoJSON(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":"hello","body":"world"}`))
		got, err := httpx.DecodeProtoJSON[*notesv1.CreateNoteRequest](req)
		require.NoError(t, err)
		require.Equal(t, "hello", got.Title)
		require.Equal(t, "world", got.Body)
	})

	t.Run("accepts proto-name request fields", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"uptime_seconds":12.5}`))
		got, err := httpx.DecodeProtoJSON[*healthv1.Response](req)
		require.NoError(t, err)
		require.Equal(t, 12.5, got.UptimeSeconds)
	})

	t.Run("accepts lowerCamel request fields", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"uptimeSeconds":9}`))
		got, err := httpx.DecodeProtoJSON[*healthv1.Response](req)
		require.NoError(t, err)
		require.Equal(t, 9.0, got.UptimeSeconds)
	})

	t.Run("malformed json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":`))
		_, err := httpx.DecodeProtoJSON[*notesv1.CreateNoteRequest](req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "decode proto JSON")
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":"x","mystery":"y"}`))
		_, err := httpx.DecodeProtoJSON[*notesv1.CreateNoteRequest](req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "mystery")
	})

	t.Run("empty body errors", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(``))
		_, err := httpx.DecodeProtoJSON[*notesv1.CreateNoteRequest](req)
		require.Error(t, err)
	})
}
