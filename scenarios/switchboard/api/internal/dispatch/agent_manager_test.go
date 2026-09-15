package dispatch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"switchboard/internal/channels"
	"switchboard/internal/threads"
)

// [REQ:SWBD-P0-012] [REQ:SWBD-P1-007]
func TestAgentManagerRunnerCreatesTaskAndRun(t *testing.T) {
	var paths []string
	var runBody map[string]any
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		authorization = r.Header.Get("Authorization")
		if r.URL.Path == "/api/v1/tasks" {
			_, _ = w.Write([]byte(`{"task":{"id":"task-1"}}`))
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&runBody)
		_, _ = w.Write([]byte(`{"run":{"id":"run-1"}}`))
	}))
	defer server.Close()
	run, err := (AgentManagerRunner{BaseURL: server.URL}).Run(context.Background(), "agent-1", []string{"read"}, "hello")
	require.NoError(t, err)
	require.Equal(t, "agent-manager run accepted: run-1", run)
	require.Equal(t, []string{"/api/v1/tasks", "/api/v1/runs"}, paths)
	inline, ok := runBody["inline_config"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []any{"read"}, inline["allowed_tools"])
	require.Empty(t, authorization)
}

// [REQ:SWBD-P0-012]
func TestAgentManagerRunnerForwardsRequestBearerWithoutPersistingIt(t *testing.T) {
	var authorization []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = append(authorization, r.Header.Get("Authorization"))
		if r.URL.Path == "/api/v1/tasks" {
			_, _ = w.Write([]byte(`{"task":{"id":"task-1"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"run":{"id":"run-1"}}`))
	}))
	defer server.Close()
	ctx := WithAuthorization(context.Background(), "Bearer owner-token")
	_, err := (AgentManagerRunner{BaseURL: server.URL}).Run(ctx, "agent-1", []string{"read"}, "hello")
	require.NoError(t, err)
	require.Equal(t, []string{"Bearer owner-token", "Bearer owner-token"}, authorization)
}

func TestAgentManagerRunnerFailsClosedOnMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"task": map[string]any{}})
	}))
	defer server.Close()
	_, err := (AgentManagerRunner{BaseURL: server.URL}).Run(context.Background(), "agent-1", nil, "hello")
	require.ErrorContains(t, err, "no id")
}

func TestAgentManagerRunnerContinuesDurableThreadRun(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/tasks":
			_, _ = w.Write([]byte(`{"task":{"id":"task-1"}}`))
		case "/api/v1/runs":
			_, _ = w.Write([]byte(`{"run":{"id":"run-1"}}`))
		case "/api/v1/runs/run-1/events":
			_, _ = w.Write([]byte(`{"events":[]}`))
		default:
			_, _ = w.Write([]byte(`{"success":true}`))
		}
	}))
	defer server.Close()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(threads.Schema)))
	store := threads.NewStore(database)
	runner := AgentManagerRunner{BaseURL: server.URL, Threads: store}
	require.NoError(t, func() error {
		_, err := runner.RunConversation(context.Background(), "agent-1", []string{"read"}, "telegram", "chat-1", "first")
		return err
	}())
	require.NoError(t, func() error {
		_, err := runner.RunConversation(context.Background(), "agent-1", []string{"read"}, "telegram", "chat-1", "second")
		return err
	}())
	require.Equal(t, []string{"/api/v1/tasks", "/api/v1/runs", "/api/v1/runs/run-1/events", "/api/v1/runs/run-1/continue"}, paths)
}

func TestAgentManagerRunnerDeliversAssistantEventOnSameThread(t *testing.T) {
	replies := make(chan channels.Outbound, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tasks":
			_, _ = w.Write([]byte(`{"task":{"id":"task-1"}}`))
		case "/api/v1/runs":
			_, _ = w.Write([]byte(`{"run":{"id":"run-1"}}`))
		case "/api/v1/runs/run-1/events":
			_, _ = w.Write([]byte(`{"events":[{"sequence":1,"message":{"role":"assistant","content":"hello from the agent"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	runner := AgentManagerRunner{BaseURL: server.URL, Wait: time.Second, Send: func(ctx context.Context, out channels.Outbound) error {
		replies <- out
		return nil
	}}
	reply, err := runner.RunConversation(context.Background(), "agent-1", []string{"read"}, "telegram", "chat-1", "hello")
	require.NoError(t, err)
	require.Empty(t, reply)
	select {
	case out := <-replies:
		require.Equal(t, "telegram", out.ChannelID)
		require.Equal(t, "chat-1", out.ThreadKey)
		require.Equal(t, "hello from the agent", out.Text)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for assistant reply")
	}
}
