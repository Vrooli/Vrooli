package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestBuildInstructionPayloadRejectsMissingTypedAction(t *testing.T) {
	_, err := buildInstructionPayload(contracts.CompiledInstruction{NodeID: "missing-action"})
	require.EqualError(t, err, "instruction missing-action has no typed action")
}

func TestBuildInstructionPayloadUsesTypedActionOnly(t *testing.T) {
	payload, err := buildInstructionPayload(contracts.CompiledInstruction{
		Index:  3,
		NodeID: "click-save",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_CLICK,
			Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{
				Selector: "[data-testid=save]",
			}},
		},
	})
	require.NoError(t, err)

	instruction, ok := payload["instruction"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, instruction, "type")
	require.NotContains(t, instruction, "params")
	action, ok := instruction["action"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), action["type"])
}

func TestBuildInstructionPayloadPreservesTypedActionVariants(t *testing.T) {
	for _, tc := range []struct {
		name       string
		action     *basactions.ActionDefinition
		payloadKey string
	}{
		{
			name: "navigate",
			action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{
				Navigate: &basactions.NavigateParams{Url: "https://example.test"},
			}},
			payloadKey: "navigate",
		},
		{
			name: "input",
			action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_INPUT, Params: &basactions.ActionDefinition_Input{
				Input: &basactions.InputParams{Selector: "#email", Value: "a@example.test"},
			}},
			payloadKey: "input",
		},
		{
			name: "subflow",
			action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SUBFLOW, Params: &basactions.ActionDefinition_Subflow{
				Subflow: &basactions.SubflowParams{Target: &basactions.SubflowParams_WorkflowPath{WorkflowPath: "actions/login.json"}},
			}},
			payloadKey: "subflow",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := buildInstructionPayload(contracts.CompiledInstruction{NodeID: tc.name, Action: tc.action})
			require.NoError(t, err)
			instruction := payload["instruction"].(map[string]any)
			action := instruction["action"].(map[string]any)
			require.Contains(t, action, tc.payloadKey)
			require.NotContains(t, instruction, "type")
			require.NotContains(t, instruction, "params")
		})
	}
}

func TestRecordingPullTransportRequiresMatchingAcknowledgement(t *testing.T) {
	for _, response := range []string{`{}`, `{"entry_ids":["different"]}`, `{"entry_ids":["entry"]}`} {
		t.Run(response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/session/session/record/actions/ack", r.URL.Path)
				var request struct {
					EntryIDs    []string `json:"entry_ids"`
					ExecutionID string   `json:"execution_id"`
					LeaseID     string   `json:"lease_id"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, []string{"entry"}, request.EntryIDs)
				assert.Equal(t, "owner", request.ExecutionID)
				assert.Equal(t, "lease", request.LeaseID)
				_, _ = w.Write([]byte(response))
			}))
			defer server.Close()
			client, err := NewClientWithURL(server.URL)
			require.NoError(t, err)
			err = client.AcknowledgeRecordedActions(context.Background(), "session", "owner", "lease", []string{"entry"})
			if response == `{"entry_ids":["entry"]}` {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestRecordingPullRejectsMalformedTimelineWithoutDestructiveRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Empty(t, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"session_id":"session","entries":[{"id":"entry","unknown_field":1}]}`))
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL)
	require.NoError(t, err)
	_, err = client.GetRecordedActions(context.Background(), "session")
	require.ErrorContains(t, err, "parse recorded entry")
}

func TestRecordingCommandsRejectMissingOwnershipBeforeHTTP(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	client, err := NewClientWithURL(srv.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	for _, identity := range [][2]string{{"", "lease"}, {"owner", " "}} {
		_, err = client.StartRecording(context.Background(), "session", identity[0], identity[1], &StartRecordingRequest{})
		require.Error(t, err)
		_, err = client.StopRecording(context.Background(), "session", identity[0], identity[1])
		require.Error(t, err)
		err = client.AcknowledgeRecordedActions(context.Background(), "session", identity[0], identity[1], []string{"entry"})
		require.Error(t, err)
	}
	_, err = client.StartRecording(context.Background(), "session", "owner", "lease", nil)
	require.Error(t, err)
	require.Zero(t, calls)
}

func TestInputRejectsInvalidEnvelopesBeforeHTTP(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	for _, identity := range [][2]string{{"", "lease"}, {"owner", " "}} {
		require.Error(t, client.ForwardInput(context.Background(), "s", identity[0], identity[1], []byte(`{"type":"pointer"}`)))
	}
	for _, body := range []string{"null", "[]", "invalid"} {
		require.Error(t, client.ForwardInput(context.Background(), "s", "owner", "lease", []byte(body)))
	}
	require.Zero(t, calls)
}

func TestNavigationCommandsRejectUnownedOrUnsupportedRequestsBeforeHTTP(t *testing.T) {
	calls := make(chan struct{}, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls <- struct{}{}; w.WriteHeader(200) }))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	for _, identity := range [][2]string{{"", "lease"}, {"owner", " "}} {
		_, err = client.Navigate(context.Background(), "session", identity[0], identity[1], &NavigateRequest{URL: "https://fixture.test"})
		require.Error(t, err)
		_, err = client.NavigateHistory(context.Background(), "session", identity[0], identity[1], HistoryReload, &HistoryNavigationRequest{})
		require.Error(t, err)
	}
	_, err = client.Navigate(context.Background(), "session", "owner", "lease", nil)
	require.Error(t, err)
	_, err = client.NavigateHistory(context.Background(), "session", "owner", "lease", HistoryBack, nil)
	require.Error(t, err)
	_, err = client.NavigateHistory(context.Background(), "session", "owner", "lease", "../../close", &HistoryNavigationRequest{})
	require.ErrorContains(t, err, "unsupported history navigation")
	require.Empty(t, calls)
}
