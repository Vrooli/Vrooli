package console

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
	"switchboard/internal/authoring"
	channelcore "switchboard/internal/channels"
	"switchboard/internal/console"
	"switchboard/internal/contacts"
	"switchboard/internal/dispatch"
	"switchboard/internal/gates"
	"switchboard/internal/ingress"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
)

type fakeProfiles struct{ list []agents.Profile }

func (f fakeProfiles) List(context.Context) ([]agents.Profile, error) { return f.list, nil }
func (f fakeProfiles) Get(_ context.Context, id string) (agents.Profile, error) {
	for _, p := range f.list {
		if p.ID == id {
			return p, nil
		}
	}
	return agents.Profile{}, agents.ErrProfileNotFound
}

type fakeWriter struct{ drafts []authoring.Draft }

func (w *fakeWriter) WriteAgent(d authoring.Draft) error { w.drafts = append(w.drafts, d); return nil }

func fixture(t *testing.T) (*mux.Router, *dispatch.Processor, *contacts.Store, *fakeWriter) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fixture.json"), []byte(`{"kind":"channel","schemaVersion":1,"id":"fixture","displayName":"Fixture","transport":"fixture","supports":{"text":true},"limits":{"maxTextBytes":1000},"setup":{"friction":1},"cost":"free","accent":"#112233"}`), 0o644))
	registry, err := channelcore.Load(dir)
	require.NoError(t, err)
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(threads.Schema), apidb.SchemaProviderFunc(agents.Schema), apidb.SchemaProviderFunc(gates.Schema), apidb.SchemaProviderFunc(contacts.Schema)))
	threadStore := threads.NewStore(db)
	contactStore := contacts.NewStore(db)
	bindingStore := agents.NewStore(db)
	sent := []channelcore.Outbound{}
	processor := &dispatch.Processor{
		Ingress: ingress.New(), Threads: threadStore, Runner: dispatch.AgentManagerRunner{}, Grant: trust.Grant{Scopes: []string{"read"}},
		Send: dispatch.PersistingReply(threadStore, func(_ context.Context, out channelcore.Outbound) error { sent = append(sent, out); return nil }, func(channelcore.Outbound) string { return "helper" }),
	}
	writer := &fakeWriter{}
	profiles := fakeProfiles{list: []agents.Profile{{ID: "helper", DisplayName: "Helper", Status: "active", Tags: []string{}, Grant: agents.CapabilityGrant{Scopes: []string{"read", "owner"}}, GrantSource: "descriptor"}}}
	router := mux.NewRouter()
	Module(Deps{
		Queries: console.Queries{DB: db, Registry: registry, Contacts: contactStore}, Registry: registry, Facts: channelcore.HostFacts{},
		Contacts: contactStore, Bindings: bindingStore, Profiles: profiles, Authoring: authoring.New(writer),
	}).Mount(router)
	_, err = bindingStore.Create(context.Background(), agents.Binding{AgentID: "helper", ChannelID: "fixture", Address: "alice", ThreadKey: "room-1"})
	require.NoError(t, err)
	_, err = bindingStore.Create(context.Background(), agents.Binding{AgentID: "ghost", ChannelID: "fixture", Address: "bob", ThreadKey: "room-2"})
	require.NoError(t, err)
	return router, processor, contactStore, writer
}

func get(t *testing.T, router *mux.Router, path string, into any) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if into != nil {
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), into), resp.Body.String())
	}
	return resp.Code
}

// [REQ:SWBD-P0-017] [REQ:SWBD-P1-005]
func TestOverviewThreadsAndRefusalsRenderRealState(t *testing.T) {
	router, processor, contactStore, _ := fixture(t)
	ctx := context.Background()
	// A stranger reaches the agent: refused out loud, recorded, persisted on the thread.
	envelope := channelcore.Envelope{ChannelID: "fixture", ThreadKey: "room-1", RemoteMessageID: "m1", SenderAddress: "alice", AuthorKind: channelcore.AuthorHuman, Text: "run the deploy"}
	contact, err := contactStore.Seen(ctx, "fixture", "alice", "stranger")
	require.NoError(t, err)
	thread, err := processor.Threads.Upsert(ctx, envelope, false)
	require.NoError(t, err)
	require.NoError(t, contactStore.Join(ctx, thread.ID, contact.ID))
	result, err := processor.Process(ctx, envelope, trust.Stranger, trust.Stranger, "helper", false, true)
	require.NoError(t, err)
	require.Equal(t, dispatch.OutcomeRefused, result.Outcome)
	require.Contains(t, result.Reply, "Contacts")

	var overview struct {
		Refusals []console.Refusal `json:"refusals"`
		Channels []struct {
			ID       string `json:"id"`
			Accent   string `json:"accent"`
			Bindings int64  `json:"bindings"`
			Threads  int64  `json:"threads"`
		} `json:"channels"`
		Gates []gates.Gate `json:"gates"`
	}
	require.Equal(t, http.StatusOK, get(t, router, "/api/v1/overview", &overview))
	require.Len(t, overview.Refusals, 1)
	require.Equal(t, "Fixture", overview.Refusals[0].ChannelDisplayName)
	require.Equal(t, "#112233", overview.Refusals[0].ChannelAccent)
	require.Equal(t, "helper", overview.Refusals[0].AgentID)
	require.Equal(t, "fixture", overview.Channels[0].ID)
	require.EqualValues(t, 2, overview.Channels[0].Bindings)
	require.EqualValues(t, 1, overview.Channels[0].Threads)

	var list []console.Thread
	require.Equal(t, http.StatusOK, get(t, router, "/api/v1/threads", &list))
	require.Len(t, list, 1)
	require.Equal(t, "helper", list[0].AgentID)
	require.Equal(t, "Helper", list[0].AgentDisplayName)
	require.Equal(t, "stranger", list[0].CeilingTier)
	require.EqualValues(t, 2, list[0].MessageCount, "human message plus the spoken refusal")
	require.NotNil(t, list[0].LastMessage)
	require.Equal(t, "agent", list[0].LastMessage.AuthorKind)

	var detail struct {
		Thread       console.Thread         `json:"thread"`
		Messages     []console.Message      `json:"messages"`
		Participants []contacts.Participant `json:"participants"`
	}
	require.Equal(t, http.StatusOK, get(t, router, "/api/v1/threads/"+thread.ID, &detail))
	require.Len(t, detail.Messages, 2)
	require.Equal(t, "human", detail.Messages[0].AuthorKind)
	require.Equal(t, "agent", detail.Messages[1].AuthorKind)
	require.Len(t, detail.Participants, 1)
	require.Equal(t, http.StatusNotFound, get(t, router, "/api/v1/threads/nope", nil))
}

// [REQ:SWBD-P0-009] [REQ:SWBD-P0-010] [REQ:SWBD-P1-001]
func TestContactsTierChangeRequiresOwnerAndReportsRooms(t *testing.T) {
	router, processor, contactStore, _ := fixture(t)
	ctx := context.Background()
	envelope := channelcore.Envelope{ChannelID: "fixture", ThreadKey: "room-1", RemoteMessageID: "m1", SenderAddress: "alice", AuthorKind: channelcore.AuthorHuman}
	contact, _ := contactStore.Seen(ctx, "fixture", "alice", "stranger")
	thread, _ := processor.Threads.Upsert(ctx, envelope, false)
	require.NoError(t, contactStore.Join(ctx, thread.ID, contact.ID))

	var list []map[string]any
	require.Equal(t, http.StatusOK, get(t, router, "/api/v1/contacts", &list))
	require.Len(t, list, 1)
	require.Equal(t, "Fixture", list[0]["channel_display_name"])

	body, _ := json.Marshal(map[string]string{"tier": "trusted"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/contacts/"+contact.ID, bytes.NewReader(body))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusUnauthorized, resp.Code, "tier changes are owner-only")

	req = httptest.NewRequest(http.MethodPut, "/api/v1/contacts/"+contact.ID, bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Identity-Subject", "owner-1")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var updated struct {
		Contact       contacts.Contact         `json:"contact"`
		AffectedRooms []contacts.CeilingChange `json:"affected_rooms"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &updated))
	require.Equal(t, "trusted", updated.Contact.Tier)
	require.Len(t, updated.AffectedRooms, 1)
	require.Equal(t, "trusted", updated.AffectedRooms[0].NewCeiling)

	bad, _ := json.Marshal(map[string]string{"tier": "vip"})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/contacts/"+contact.ID, bytes.NewReader(bad))
	req.Header.Set("X-Vrooli-Identity-Subject", "owner-1")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	var one struct {
		Rooms []map[string]any `json:"rooms"`
	}
	require.Equal(t, http.StatusOK, get(t, router, "/api/v1/contacts/"+contact.ID, &one))
	require.Len(t, one.Rooms, 1)
	require.Equal(t, "#112233", one.Rooms[0]["channel_accent"])
}

// [REQ:SWBD-P0-008] [REQ:SWBD-P0-011]
func TestAgentsRosterIsByReferenceAndKeepsBrokenReferences(t *testing.T) {
	router, _, _, writer := fixture(t)
	var roster struct {
		Source struct {
			OK bool `json:"ok"`
		} `json:"source"`
		Agents []struct {
			ID       string `json:"id"`
			Broken   string `json:"broken"`
			Bindings []struct {
				ChannelAccent string `json:"channel_accent"`
			} `json:"bindings"`
			Grant struct {
				Scopes    []string `json:"scopes"`
				OwnerOnly []string `json:"owner_only"`
				Source    string   `json:"source"`
			} `json:"grant"`
		} `json:"agents"`
	}
	require.Equal(t, http.StatusOK, get(t, router, "/api/v1/agents", &roster))
	require.True(t, roster.Source.OK)
	require.Len(t, roster.Agents, 2)
	require.Equal(t, "helper", roster.Agents[0].ID)
	require.Equal(t, []string{"owner"}, roster.Agents[0].Grant.OwnerOnly)
	require.Equal(t, "descriptor", roster.Agents[0].Grant.Source)
	require.Equal(t, "#112233", roster.Agents[0].Bindings[0].ChannelAccent)
	require.Equal(t, "ghost", roster.Agents[1].ID)
	require.NotEmpty(t, roster.Agents[1].Broken)
	require.Equal(t, "default", roster.Agents[1].Grant.Source)

	var one map[string]any
	require.Equal(t, http.StatusOK, get(t, router, "/api/v1/agents/helper", &one))
	require.Contains(t, one, "activity_log")
	require.Equal(t, http.StatusNotFound, get(t, router, "/api/v1/agents/missing", nil))

	draftReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/draft", bytes.NewReader([]byte(`{"description":"Concierge for the household calendar"}`)))
	draftResp := httptest.NewRecorder()
	router.ServeHTTP(draftResp, draftReq)
	require.Equal(t, http.StatusOK, draftResp.Code)
	var draft map[string]any
	require.NoError(t, json.Unmarshal(draftResp.Body.Bytes(), &draft))
	require.Equal(t, "Concierge", draft["display_name"])

	create := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(draftResp.Body.Bytes()))
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, create)
	require.Equal(t, http.StatusUnauthorized, createResp.Code)
	create = httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(draftResp.Body.Bytes()))
	create.Header.Set("X-Vrooli-Identity-Subject", "owner-1")
	createResp = httptest.NewRecorder()
	router.ServeHTTP(createResp, create)
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	require.Len(t, writer.drafts, 1)
	require.Equal(t, []string{"read"}, writer.drafts[0].Scopes)

	login := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader([]byte(`{"email":"a@b.c","password":"x"}`)))
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, login)
	require.Equal(t, http.StatusServiceUnavailable, loginResp.Code, "no authenticator configured in this fixture")
}
