package main

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapptest"
)

// `integrations event-config-set` changes only the settings it is given: the
// others keep the values the API holds.
func TestIntegrationsEventConfigSetChangesOnlyWhatItIsGiven(t *testing.T) {
	var put map[string]any
	router := http.NewServeMux()
	router.HandleFunc("/api/v1/config/event-integration", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"events_api_base":"http://events.local","webhook_url":"http://hub.local/hook","pattern":"incident.**"}`))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(body, &put))
			_, _ = w.Write(body)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	cliapptest.NewAPIServer(t, router)

	app, err := NewApp()
	require.NoError(t, err)
	require.NoError(t, app.Run([]string{"integrations", "event-config-set", "--pattern", "backup.**"}))
	require.Equal(t, "backup.**", put["pattern"])
	require.Equal(t, "http://events.local", put["events_api_base"], "a setting the command did not name keeps its value")
	require.Equal(t, "http://hub.local/hook", put["webhook_url"])
}
