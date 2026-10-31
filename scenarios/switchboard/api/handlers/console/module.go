// Package console exposes the operator console's read model and its few
// owner-authenticated writes. Reads are unauthenticated operational probes,
// like /api/v1/channels and /health; every write resolves the owner through
// the same bearer verification as the channels module (see identity.Subject).
//
// Auth decision: the browser console obtains its owner credential through the
// same-origin login facade at POST /api/v1/session/login, which forwards to
// scenario-authenticator's typed AccountsService and relays the issued token.
// This mirrors device-sync-hub's IdentityService: the browser never calls the
// authenticator cross-origin and this API never mints or bypasses credentials.
package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/owneridentity"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"

	"switchboard/internal/agents"
	"switchboard/internal/authoring"
	channelcore "switchboard/internal/channels"
	"switchboard/internal/console"
	"switchboard/internal/contacts"
	"switchboard/internal/gates"
	"switchboard/internal/identity"
	"switchboard/internal/module"
)

// ProfileSource is the by-reference agent roster (prompt-manager).
type ProfileSource interface {
	List(context.Context) ([]agents.Profile, error)
	Get(context.Context, string) (agents.Profile, error)
}

type Deps struct {
	Queries   console.Queries
	Registry  *channelcore.Registry
	Facts     channelcore.HostFacts
	Contacts  *contacts.Store
	Bindings  *agents.Store
	Profiles  ProfileSource
	Authoring *authoring.Service
	CreatedID func(context.Context, authoring.Draft) string
	Identity  owneridentity.Validator
	// AuthURL is scenario-authenticator's base URL for the login facade. Empty
	// disables the facade with a 503 that says so.
	AuthURL    string
	HTTPClient connect.HTTPClient
	Now        func() time.Time
}

type handler struct{ d Deps }

func Module(d Deps) module.Module {
	h := &handler{d: d}
	return module.Module{Name: "console", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/overview", h.overview).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/threads", h.listThreads).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/threads/{id}", h.getThread).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/contacts", h.listContacts).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/contacts/{id}", h.getContact).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/contacts/{id}", h.updateContact).Methods(http.MethodPut)
		r.HandleFunc("/api/v1/agents", h.listAgents).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/agents", h.createAgent).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/agents/draft", h.draftAgent).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/agents/{id}", h.getAgent).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/gates", h.listGates).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/session/login", h.login).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/session", h.session).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func (h *handler) now() time.Time {
	if h.d.Now != nil {
		return h.d.Now()
	}
	return time.Now()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error, reason string) {
	writeJSON(w, status, map[string]string{"error": err.Error(), "reason": reason})
}

// --- overview -------------------------------------------------------------

type channelHealth struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	Accent       string `json:"accent"`
	Availability string `json:"availability"`
	Reason       string `json:"reason"`
	Implemented  bool   `json:"implemented"`
	Friction     int    `json:"friction"`
	Bindings     int64  `json:"bindings"`
	Threads      int64  `json:"threads"`
}

func (h *handler) channelHealth(ctx context.Context) ([]channelHealth, error) {
	bindings, threads, err := h.d.Queries.ChannelCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]channelHealth, 0)
	if h.d.Registry == nil {
		return out, nil
	}
	for _, l := range h.d.Registry.List(ctx, h.d.Facts) {
		out = append(out, channelHealth{ID: l.Descriptor.ID, DisplayName: l.Descriptor.DisplayName, Accent: l.Descriptor.Accent, Availability: string(l.Availability), Reason: l.Reason, Implemented: l.Implemented, Friction: l.Descriptor.Setup.Friction, Bindings: bindings[l.Descriptor.ID], Threads: threads[l.Descriptor.ID]})
	}
	return out, nil
}

func (h *handler) overview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pending, err := h.d.Queries.Gates(ctx, string(gates.Pending), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "gates")
		return
	}
	refusals, err := h.d.Queries.Refusals(ctx, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "refusals")
		return
	}
	channelsOut, err := h.channelHealth(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "channels")
		return
	}
	threads, err := h.d.Queries.ListThreads(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "threads")
		return
	}
	pressure := make([]console.ThreadBudget, 0)
	for _, t := range threads {
		if t.Budget.Pressure() {
			pressure = append(pressure, t.Budget)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"generated_at": h.now().UTC().Format(time.RFC3339),
		"gates":        pending,
		"refusals":     refusals,
		"channels":     channelsOut,
		"budget":       map[string]any{"threads_under_pressure": pressure},
	})
}

// --- threads --------------------------------------------------------------

func (h *handler) decorate(ctx context.Context, threads []console.Thread) {
	if h.d.Profiles == nil {
		return
	}
	profiles, err := h.d.Profiles.List(ctx)
	if err != nil {
		return
	}
	names := make(map[string]string, len(profiles))
	for _, p := range profiles {
		names[p.ID] = p.DisplayName
	}
	for i := range threads {
		threads[i].AgentDisplayName = names[threads[i].AgentID]
	}
}

func (h *handler) listThreads(w http.ResponseWriter, r *http.Request) {
	threads, err := h.d.Queries.ListThreads(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "threads")
		return
	}
	h.decorate(r.Context(), threads)
	writeJSON(w, http.StatusOK, threads)
}

func (h *handler) getThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	thread, err := h.d.Queries.GetThread(ctx, id)
	if errors.Is(err, console.ErrNotFound) {
		writeError(w, http.StatusNotFound, err, "thread")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "thread")
		return
	}
	list := []console.Thread{thread}
	h.decorate(ctx, list)
	thread = list[0]
	messages, err := h.d.Queries.Messages(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "messages")
		return
	}
	participants := []contacts.Participant{}
	if h.d.Contacts != nil {
		if participants, err = h.d.Contacts.Roster(ctx, id); err != nil {
			writeError(w, http.StatusInternalServerError, err, "roster")
			return
		}
	}
	threadGates, err := h.d.Queries.Gates(ctx, "", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "gates")
		return
	}
	runID, err := h.d.Queries.RunID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "run")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"thread": thread, "messages": messages, "participants": participants, "gates": threadGates, "run_id": runID})
}

// --- contacts -------------------------------------------------------------

type roomView struct {
	contacts.Room
	ChannelDisplayName string `json:"channel_display_name"`
	ChannelAccent      string `json:"channel_accent"`
}

func (h *handler) rooms(ctx context.Context, contactID string) ([]roomView, error) {
	rooms, err := h.d.Contacts.Rooms(ctx, contactID)
	if err != nil {
		return nil, err
	}
	out := make([]roomView, 0, len(rooms))
	for _, room := range rooms {
		view := roomView{Room: room, ChannelDisplayName: room.ChannelID}
		if h.d.Registry != nil {
			if d, ok := h.d.Registry.Get(room.ChannelID); ok {
				view.ChannelDisplayName, view.ChannelAccent = d.DisplayName, d.Accent
			}
		}
		out = append(out, view)
	}
	return out, nil
}

type contactView struct {
	contacts.Contact
	ChannelDisplayName string `json:"channel_display_name"`
	ChannelAccent      string `json:"channel_accent"`
}

func (h *handler) contactView(c contacts.Contact) contactView {
	view := contactView{Contact: c, ChannelDisplayName: c.ChannelID}
	if h.d.Registry != nil {
		if d, ok := h.d.Registry.Get(c.ChannelID); ok {
			view.ChannelDisplayName, view.ChannelAccent = d.DisplayName, d.Accent
		}
	}
	return view
}

func (h *handler) listContacts(w http.ResponseWriter, r *http.Request) {
	if h.d.Contacts == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("contact storage unavailable"), "contacts")
		return
	}
	list, err := h.d.Contacts.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "contacts")
		return
	}
	out := make([]contactView, 0, len(list))
	for _, c := range list {
		out = append(out, h.contactView(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) getContact(w http.ResponseWriter, r *http.Request) {
	if h.d.Contacts == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("contact storage unavailable"), "contacts")
		return
	}
	id := mux.Vars(r)["id"]
	c, err := h.d.Contacts.Get(r.Context(), id)
	if errors.Is(err, contacts.ErrNotFound) {
		writeError(w, http.StatusNotFound, err, "contact")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "contact")
		return
	}
	rooms, err := h.rooms(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "rooms")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contact": h.contactView(c), "rooms": rooms})
}

func (h *handler) updateContact(w http.ResponseWriter, r *http.Request) {
	if h.d.Contacts == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("contact storage unavailable"), "contacts")
		return
	}
	if _, err := identity.Subject(r.Context(), r.Header, h.d.Identity); err != nil {
		writeError(w, http.StatusUnauthorized, err, "owner credential required to change a trust tier")
		return
	}
	var body struct {
		Tier        *string `json:"tier"`
		DisplayName *string `json:"display_name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid contact update"), "body")
		return
	}
	if body.Tier == nil && body.DisplayName == nil {
		writeError(w, http.StatusBadRequest, errors.New("tier or display_name is required"), "body")
		return
	}
	c, changes, err := h.d.Contacts.Update(r.Context(), mux.Vars(r)["id"], body.Tier, body.DisplayName)
	if errors.Is(err, contacts.ErrNotFound) {
		writeError(w, http.StatusNotFound, err, "contact")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err, "tier")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contact": h.contactView(c), "affected_rooms": changes})
}

// --- agents ---------------------------------------------------------------

type bindingView struct {
	ID                 string `json:"id"`
	ChannelID          string `json:"channel_id"`
	ChannelDisplayName string `json:"channel_display_name"`
	ChannelAccent      string `json:"channel_accent"`
	Address            string `json:"address"`
	ThreadKey          string `json:"thread_key"`
	Live               bool   `json:"live"`
}

type agentView struct {
	agents.Profile
	Broken   string           `json:"broken,omitempty"`
	Bindings []bindingView    `json:"bindings"`
	Grant    agents.GrantView `json:"grant"`
	Activity console.Activity `json:"activity"`
}

func (h *handler) bindingViews(ctx context.Context, agentID string) ([]bindingView, error) {
	out := make([]bindingView, 0)
	if h.d.Bindings == nil {
		return out, nil
	}
	records, err := h.d.Bindings.List(ctx, agentID)
	if err != nil {
		return nil, err
	}
	for _, rec := range records {
		view := bindingView{ID: rec.ID, ChannelID: rec.ChannelID, ChannelDisplayName: rec.ChannelID, Address: rec.Address, ThreadKey: rec.ThreadKey}
		if h.d.Registry != nil {
			if d, ok := h.d.Registry.Get(rec.ChannelID); ok {
				view.ChannelDisplayName, view.ChannelAccent = d.DisplayName, d.Accent
				state, _ := channelcore.Evaluate(d, h.d.Facts)
				_, implemented := h.d.Registry.Adapter(d.ID)
				view.Live = state == channelcore.Available && implemented
			}
		}
		out = append(out, view)
	}
	return out, nil
}

func (h *handler) view(ctx context.Context, p agents.Profile, broken string) (agentView, error) {
	bindings, err := h.bindingViews(ctx, p.ID)
	if err != nil {
		return agentView{}, err
	}
	activity, err := h.d.Queries.AgentActivity(ctx, p.ID)
	if err != nil {
		return agentView{}, err
	}
	if p.GrantSource == "" {
		p.Grant, p.GrantSource = agents.DefaultGrant, "default"
	}
	return agentView{Profile: p, Broken: broken, Bindings: bindings, Grant: p.GrantView(), Activity: activity}, nil
}

func (h *handler) listAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	source := map[string]any{"ok": true, "reason": ""}
	var profiles []agents.Profile
	if h.d.Profiles == nil {
		source["ok"], source["reason"] = false, "prompt-manager is not configured"
	} else if list, err := h.d.Profiles.List(ctx); err != nil {
		source["ok"], source["reason"] = false, err.Error()
	} else {
		profiles = list
	}
	seen := make(map[string]struct{}, len(profiles))
	out := make([]agentView, 0, len(profiles))
	for _, p := range profiles {
		seen[p.ID] = struct{}{}
		view, err := h.view(ctx, p, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err, "agents")
			return
		}
		out = append(out, view)
	}
	// D7: an agent that is bound here but missing from prompt-manager renders
	// with its reason instead of being filtered out.
	if h.d.Bindings != nil {
		records, err := h.d.Bindings.List(ctx, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err, "bindings")
			return
		}
		for _, rec := range records {
			if _, ok := seen[rec.AgentID]; ok {
				continue
			}
			seen[rec.AgentID] = struct{}{}
			reason := "agent profile not found in prompt-manager"
			if source["ok"] == false {
				reason = "agent profile unavailable: " + fmt.Sprint(source["reason"])
			}
			view, err := h.view(ctx, agents.Profile{ID: rec.AgentID, DisplayName: rec.AgentID, Status: "unknown", Tags: []string{}}, reason)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err, "agents")
				return
			}
			out = append(out, view)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"source": source, "agents": out})
}

func (h *handler) getAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	var profile agents.Profile
	broken := ""
	if h.d.Profiles == nil {
		profile, broken = agents.Profile{ID: id, DisplayName: id, Status: "unknown", Tags: []string{}}, "prompt-manager is not configured"
	} else if p, err := h.d.Profiles.Get(ctx, id); err != nil {
		bindings, berr := h.bindingViews(ctx, id)
		if berr != nil || len(bindings) == 0 {
			if errors.Is(err, agents.ErrProfileNotFound) {
				writeError(w, http.StatusNotFound, err, "agent")
				return
			}
			writeError(w, http.StatusBadGateway, err, "prompt-manager")
			return
		}
		profile, broken = agents.Profile{ID: id, DisplayName: id, Status: "unknown", Tags: []string{}}, err.Error()
	} else {
		profile = p
	}
	view, err := h.view(ctx, profile, broken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "agent")
		return
	}
	log, err := h.d.Queries.AgentActivityLog(ctx, id, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "activity")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		agentView
		ActivityLog []console.ActivityEntry `json:"activity_log"`
	}{view, log})
}

type draftBody struct {
	ID              string   `json:"id,omitempty"`
	DisplayName     string   `json:"display_name"`
	Description     string   `json:"description"`
	Scopes          []string `json:"scopes"`
	OwnerOnlyScopes []string `json:"owner_only_scopes"`
}

func (h *handler) draftAgent(w http.ResponseWriter, r *http.Request) {
	if h.d.Authoring == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent authoring unavailable"), "authoring")
		return
	}
	var body struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid draft request"), "body")
		return
	}
	d, err := h.d.Authoring.Draft(body.Description)
	if err != nil {
		writeError(w, http.StatusBadRequest, err, "description")
		return
	}
	writeJSON(w, http.StatusOK, draftBody{DisplayName: d.DisplayName, Description: d.Description, Scopes: d.Scopes, OwnerOnlyScopes: []string{}})
}

func (h *handler) createAgent(w http.ResponseWriter, r *http.Request) {
	if h.d.Authoring == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent authoring unavailable"), "authoring")
		return
	}
	if _, err := identity.Subject(r.Context(), r.Header, h.d.Identity); err != nil {
		writeError(w, http.StatusUnauthorized, err, "owner credential required to create an agent")
		return
	}
	var body draftBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid agent draft"), "body")
		return
	}
	draft := authoring.Draft{ID: strings.TrimSpace(body.ID), DisplayName: strings.TrimSpace(body.DisplayName), Description: strings.TrimSpace(body.Description), Scopes: body.Scopes, OwnerOnlyScopes: body.OwnerOnlyScopes}
	if len(draft.Scopes) == 0 {
		draft.Scopes = agents.DefaultGrant.Scopes
	}
	if err := h.d.Authoring.Confirm(draft); err != nil {
		writeError(w, http.StatusBadGateway, err, "prompt-manager create")
		return
	}
	id := draft.ID
	if h.d.CreatedID != nil {
		id = h.d.CreatedID(r.Context(), draft)
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "display_name": draft.DisplayName})
}

// --- gates ------------------------------------------------------------------

func (h *handler) listGates(w http.ResponseWriter, r *http.Request) {
	list, err := h.d.Queries.Gates(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")), strings.TrimSpace(r.URL.Query().Get("thread_id")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err, "gates")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// --- session ----------------------------------------------------------------

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.d.AuthURL) == "" {
		writeError(w, http.StatusServiceUnavailable, errors.New("scenario-authenticator is unavailable"), "the owner sign-in facade needs scenario-authenticator running")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.Email) == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, errors.New("email and password are required"), "body")
		return
	}
	client := h.d.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	accounts := accountsconnect.NewAccountsServiceClient(client, strings.TrimRight(h.d.AuthURL, "/"))
	resp, err := accounts.Login(r.Context(), connect.NewRequest(&accountsv1.LoginRequest{Email: strings.TrimSpace(body.Email), Password: body.Password}))
	if err != nil {
		switch connect.CodeOf(err) {
		case connect.CodeUnauthenticated, connect.CodePermissionDenied, connect.CodeNotFound:
			writeError(w, http.StatusUnauthorized, errors.New("invalid credentials"), "login")
		case connect.CodeInvalidArgument:
			writeError(w, http.StatusBadRequest, err, "login")
		default:
			writeError(w, http.StatusBadGateway, err, "scenario-authenticator")
		}
		return
	}
	tokens := resp.Msg.GetTokens()
	if tokens == nil || tokens.GetAccessToken() == "" {
		writeError(w, http.StatusBadGateway, errors.New("authenticator returned no token"), "scenario-authenticator")
		return
	}
	out := map[string]any{"token": tokens.GetAccessToken(), "refresh_token": tokens.GetRefreshToken()}
	if account := resp.Msg.GetAccount(); account != nil {
		out["subject"], out["email"] = account.GetId(), account.GetEmail()
	}
	writeJSON(w, http.StatusOK, out)
}

// session reports who the bearer credential belongs to, so the console can
// show the signed-in owner and know whether writes will be accepted.
func (h *handler) session(w http.ResponseWriter, r *http.Request) {
	// An anonymous caller is a normal state for this probe, not a failure:
	// the console asks "who am I" before deciding whether to offer sign-in.
	subject, err := identity.Subject(r.Context(), r.Header, h.d.Identity)
	authenticated := err == nil && strings.TrimSpace(subject) != ""
	out := map[string]any{"authenticated": authenticated, "login_available": strings.TrimSpace(h.d.AuthURL) != ""}
	if authenticated {
		out["subject"] = subject
	}
	writeJSON(w, http.StatusOK, out)
}

func restJSON(hasRequest bool) *module.RESTException {
	request := module.RESTPayload{Transport: "none", Conformance: "none"}
	if hasRequest {
		request = module.RESTPayload{Transport: "json", Conformance: "external_shape"}
	}
	return &module.RESTException{Reason: module.RESTReasonOpsProbe, ProtoPayloads: &module.RESTProtoPayloads{
		Request: request, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
	}}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "console_overview", Path: "/api/v1/overview", Method: http.MethodGet, Summary: "Operator overview", Description: "Pending gates, recent refusals, channel health, and threads under budget pressure.", Category: "console", RESTException: restJSON(false)},
	{ID: "console_threads_list", Path: "/api/v1/threads", Method: http.MethodGet, Summary: "List threads", Description: "Every thread across every channel, newest activity first.", Category: "console", RESTException: restJSON(false)},
	{ID: "console_threads_get", Path: "/api/v1/threads/{id}", Method: http.MethodGet, Summary: "Read a thread", Description: "Thread, transcript, roster, gates, and run link.", Category: "console", RESTException: restJSON(false)},
	{ID: "console_contacts_list", Path: "/api/v1/contacts", Method: http.MethodGet, Summary: "List contacts", Category: "console", RESTException: restJSON(false)},
	{ID: "console_contacts_get", Path: "/api/v1/contacts/{id}", Method: http.MethodGet, Summary: "Read a contact and its rooms", Category: "console", RESTException: restJSON(false)},
	{ID: "console_contacts_update", Path: "/api/v1/contacts/{id}", Method: http.MethodPut, Summary: "Change a contact's tier or name", Description: "Owner-authenticated. Reports every room whose ceiling moved.", Category: "console", RESTException: restJSON(true)},
	{ID: "console_agents_list", Path: "/api/v1/agents", Method: http.MethodGet, Summary: "Agent roster by reference", Description: "Profiles from prompt-manager joined with bindings, grants, and activity.", Category: "console", RESTException: restJSON(false)},
	{ID: "console_agents_create", Path: "/api/v1/agents", Method: http.MethodPost, Summary: "Confirm an agent draft", Description: "Owner-authenticated. Writes the profile through prompt-manager's typed create.", Category: "console", RESTException: restJSON(true)},
	{ID: "console_agents_draft", Path: "/api/v1/agents/draft", Method: http.MethodPost, Summary: "Draft an agent", Description: "Typed draft from a description; nothing is written.", Category: "console", RESTException: restJSON(true)},
	{ID: "console_agents_get", Path: "/api/v1/agents/{id}", Method: http.MethodGet, Summary: "Read one agent with its activity log", Category: "console", RESTException: restJSON(false)},
	{ID: "console_gates_list", Path: "/api/v1/gates", Method: http.MethodGet, Summary: "List capability gates", Description: "Filter with ?status=pending and ?thread_id=.", Category: "console", RESTException: restJSON(false)},
	{ID: "console_session_login", Path: "/api/v1/session/login", Method: http.MethodPost, Summary: "Owner sign-in facade", Description: "Forwards to scenario-authenticator and relays the issued owner token.", Category: "console", RESTException: restJSON(true)},
	{ID: "console_session", Path: "/api/v1/session", Method: http.MethodGet, Summary: "Current owner session", Category: "console", RESTException: restJSON(false)},
}
