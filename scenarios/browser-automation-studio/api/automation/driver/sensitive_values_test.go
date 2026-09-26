package driver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRedactSensitiveValues(t *testing.T) {
	for _, tc := range []struct {
		name       string
		attributes map[string]string
	}{
		{name: "password", attributes: map[string]string{"type": "password"}},
		{name: "hidden", attributes: map[string]string{"type": "hidden"}},
		{name: "one-time-code", attributes: map[string]string{"type": "text", "autocomplete": "section-login one-time-code"}},
		{name: "payment", attributes: map[string]string{"type": "text", "autocomplete": "cc-number"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const secret = "BAS_SYNTHETIC_API_SECRET_43bd"
			attrs := map[string]string{"value": secret, "data-secret": secret}
			for k, v := range tc.attributes {
				attrs[k] = v
			}
			action := &RecordedAction{
				ActionType:  "type",
				ElementMeta: &ElementMeta{TagName: "input", InnerText: secret, Attributes: attrs},
				Payload:     map[string]interface{}{"text": secret, "value": secret},
			}

			require.True(t, RedactSensitiveValues(action))
			require.Empty(t, action.ElementMeta.InnerText)
			require.NotContains(t, action.ElementMeta.Attributes, "value")
			require.NotContains(t, action.ElementMeta.Attributes, "data-secret")
			require.Equal(t, "", action.Payload["text"])
			require.Equal(t, "", action.Payload["value"])
		})
	}

	ordinary := &RecordedAction{
		ActionType:  "type",
		ElementMeta: &ElementMeta{TagName: "input", InnerText: "ordinary value", Attributes: map[string]string{"type": "text"}},
		Payload:     map[string]interface{}{"text": "ordinary value"},
	}
	require.False(t, RedactSensitiveValues(ordinary))
	require.Equal(t, "ordinary value", ordinary.Payload["text"])
}

func TestGetRecordedActionsRedactsBufferedTimelineAndKeepsLegacyEntrySafe(t *testing.T) {
	const secret = "BAS_SYNTHETIC_BUFFERED_SECRET_735e"
	entry := &bastimeline.TimelineEntry{
		Id: "synthetic-entry",
		Action: &basactions.ActionDefinition{
			Type:   basactions.ActionType_ACTION_TYPE_INPUT,
			Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{Selector: "#password", Value: secret}},
			Metadata: &basactions.ActionMetadata{ElementSnapshot: &basdomain.ElementMeta{
				TagName: "input", InnerText: secret, Attributes: map[string]string{"type": "password", "value": secret},
			}},
		},
	}
	entryJSON, err := protojson.Marshal(entry)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id": "synthetic-session",
			"entries":    []json.RawMessage{entryJSON},
		}))
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)

	response, err := client.GetRecordedActions(t.Context(), "synthetic-session")
	require.NoError(t, err)
	require.Len(t, response.Actions, 1)
	require.Empty(t, response.Actions[0].Payload["text"])
	require.NotContains(t, string(response.Entries[0]), secret)
	require.Empty(t, response.Actions[0].ElementMeta.InnerText)
	require.NotContains(t, response.Actions[0].ElementMeta.Attributes, "value")
}
