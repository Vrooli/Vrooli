package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"switchboard/internal/agents"
	channelcore "switchboard/internal/channels"
	"switchboard/internal/channels/adapters/inapp"
	"switchboard/internal/contacts"
	"switchboard/internal/dispatch"
	"switchboard/internal/gates"
	"switchboard/internal/ingress"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
)

type recordingRunner struct{ calls []string }

func (r *recordingRunner) Run(_ context.Context, agent string, _ []string, _ string) (string, error) {
	r.calls = append(r.calls, agent)
	return "done", nil
}

// [REQ:SWBD-P0-009] [REQ:SWBD-P0-010] [REQ:SWBD-P0-014]
func TestIngestResolvesRealTiersFromContactsAndDescriptorDefaults(t *testing.T) {
	dir := t.TempDir()
	// A console-shaped channel whose descriptor declares owner as the default tier, and an external one that does not.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "in-app.json"), []byte(`{"kind":"channel","schemaVersion":1,"id":"in-app","displayName":"App","transport":"websocket","supports":{"text":true},"limits":{"maxTextBytes":1000},"setup":{"friction":0},"cost":"free","trust":{"defaultTier":"owner"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "external.json"), []byte(`{"kind":"channel","schemaVersion":1,"id":"external","displayName":"External","transport":"poll","supports":{"text":true},"limits":{"maxTextBytes":1000},"setup":{"friction":2},"cost":"free"}`), 0o644))
	registry, err := channelcore.Load(dir, inapp.New())
	require.NoError(t, err)
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(threads.Schema), apidb.SchemaProviderFunc(agents.Schema), apidb.SchemaProviderFunc(gates.Schema), apidb.SchemaProviderFunc(contacts.Schema)))
	threadStore := threads.NewStore(db)
	runner := &recordingRunner{}
	processor := &dispatch.Processor{Ingress: ingress.New(), Threads: threadStore, Runner: runner, Grant: trust.Grant{Scopes: []string{"read"}}, Send: dispatch.PersistingReply(threadStore, nil, nil)}
	router := mux.NewRouter()
	Module(ModuleDeps{Registry: registry, DB: db, Processor: processor, Threads: threadStore, Contacts: contacts.NewStore(db)}).Mount(router)

	// Start an in-app thread through the adapter-owned endpoint.
	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/channels/in-app/threads", bytes.NewReader([]byte(`{"agent_id":"helper"}`)))
	startResp := httptest.NewRecorder()
	router.ServeHTTP(startResp, startReq)
	require.Equal(t, http.StatusCreated, startResp.Code, startResp.Body.String())
	var started struct {
		ThreadID  string `json:"thread_id"`
		ThreadKey string `json:"thread_key"`
		ChannelID string `json:"channel_id"`
	}
	require.NoError(t, json.Unmarshal(startResp.Body.Bytes(), &started))
	require.Equal(t, "in-app", started.ChannelID)
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM switchboard_threads WHERE id=?`, started.ThreadID).Scan(&count))
	require.Equal(t, 1, count, "the thread is listable before its first message")

	receive := func(e channelcore.Envelope) dispatch.Result {
		body, _ := json.Marshal(e)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/receive", bytes.NewReader(body))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var result dispatch.Result
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &result))
		return result
	}
	// The owner speaking from the console runs a turn.
	result := receive(channelcore.Envelope{ChannelID: "in-app", ThreadKey: started.ThreadKey, RemoteMessageID: "c1", SenderAddress: inapp.OwnerAddress, AuthorKind: channelcore.AuthorHuman, Text: "hello"})
	require.Equal(t, dispatch.OutcomeAccepted, result.Outcome)
	require.Equal(t, []string{"helper"}, runner.calls)
	var tier string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT tier FROM switchboard_contacts WHERE channel_id='in-app' AND address=?`, inapp.OwnerAddress).Scan(&tier))
	require.Equal(t, "owner", tier)

	// An unknown external sender is a stranger and is refused out loud, with no run.
	_, err = agents.NewStore(db).Create(context.Background(), agents.Binding{AgentID: "helper", ChannelID: "external", Address: "555", ThreadKey: "chat-9"})
	require.NoError(t, err)
	result = receive(channelcore.Envelope{ChannelID: "external", ThreadKey: "chat-9", RemoteMessageID: "x1", SenderAddress: "555", AuthorKind: channelcore.AuthorHuman, Text: "deploy prod"})
	require.Equal(t, dispatch.OutcomeRefused, result.Outcome)
	require.Contains(t, result.Reply, "stranger")
	require.Len(t, runner.calls, 1)

	// Unbound senders are still rejected before anything is recorded.
	body, _ := json.Marshal(channelcore.Envelope{ChannelID: "external", ThreadKey: "nowhere", RemoteMessageID: "x2", SenderAddress: "777", AuthorKind: channelcore.AuthorHuman})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/receive", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusForbidden, resp.Code)
}
