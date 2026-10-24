// Package preferences mounts the plain-REST notification-preferences surface:
// the per-recipient catalog read and the preference upsert the hub's own
// Settings UI drives. It deliberately mirrors the proto-free operator-config
// REST style (see internal/integrations.Handler) rather than Connect-RPC, so
// there are no proto changes and no generated-client dependency.
//
// Read-only catalog and preference views use the installation's configured
// recipient. Preference writes still resolve a verified owner identity. The
// module mounts behind optional owner-context middleware, which does not itself
// reject unauthenticated reads.
package preferences

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	identity "notification-hub/internal/identity"
	"notification-hub/internal/module"
	"notification-hub/internal/preferences"

	"github.com/gorilla/mux"
)

type handler struct {
	store     *preferences.Storage
	verifier  identity.Verifier
	recipient func(context.Context) string
}

// Module wires the preferences REST surface with no identity verifier; useful
// for isolated handler tests that inject the legacy subject header.
func Module(store *preferences.Storage) module.Module {
	return ModuleWithVerifier(store, nil)
}

// ModuleWithVerifier wires the preferences REST surface with owner verification
// for writes. Read-only preference views have no configured recipient unless
// ModuleWithRecipientResolver is used.
func ModuleWithVerifier(store *preferences.Storage, verifier identity.Verifier) module.Module {
	return ModuleWithRecipientResolver(store, nil, verifier)
}

// ModuleWithRecipientResolver scopes read-only preference views to the
// installation's configured recipient while writes continue to use verified
// identity.Subject.
func ModuleWithRecipientResolver(store *preferences.Storage, recipient func(context.Context) string, verifier identity.Verifier) module.Module {
	h := &handler{store: store, verifier: verifier, recipient: recipient}
	return module.Module{Name: "preferences", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/notifications/catalog", h.getCatalog).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/notifications/preferences", h.getPreferences).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/notifications/preferences", h.putPreference).Methods(http.MethodPut)
	}}
}

func (h *handler) subject(r *http.Request) (string, error) {
	return identity.Subject(r.Context(), r.Header, h.verifier)
}

func (h *handler) configuredRecipient(ctx context.Context) string {
	if h.recipient == nil {
		return ""
	}
	return strings.TrimSpace(h.recipient(ctx))
}

// categoryView is one category under a source in the catalog response.
type categoryView struct {
	Category          string `json:"category"`
	DisplayName       string `json:"display_name"`
	Description       string `json:"description"`
	Declared          bool   `json:"declared"`
	DefaultEnabled    bool   `json:"default_enabled"`
	DefaultMinUrgency string `json:"default_min_urgency"`
}

// sourceView is one notification-producing scenario and its categories.
type sourceView struct {
	SourceScenario string         `json:"source_scenario"`
	DisplayName    string         `json:"display_name"`
	Icon           string         `json:"icon"`
	DeepLinkBase   string         `json:"deep_link_base"`
	Declared       bool           `json:"declared"`
	Categories     []categoryView `json:"categories"`
}

// preferenceView is one stored per-recipient preference row. ScopeKey is empty
// for a global preference and set for a per-instance (e.g. per-session) override.
type preferenceView struct {
	SourceScenario  string `json:"source_scenario"`
	Category        string `json:"category"`
	ScopeKey        string `json:"scope_key"`
	Enabled         bool   `json:"enabled"`
	MinUrgency      string `json:"min_urgency"`
	ChannelOverride string `json:"channel_override"`
}

type catalogResponse struct {
	Sources     []sourceView     `json:"sources"`
	Preferences []preferenceView `json:"preferences"`
}

type preferencesResponse struct {
	Preferences []preferenceView `json:"preferences"`
}

type putPreferenceRequest struct {
	SourceScenario  string `json:"source_scenario"`
	Category        string `json:"category"`
	ScopeKey        string `json:"scope_key"`
	Enabled         bool   `json:"enabled"`
	MinUrgency      string `json:"min_urgency"`
	ChannelOverride string `json:"channel_override"`
}

func (h *handler) getCatalog(w http.ResponseWriter, r *http.Request) {
	recipient := h.configuredRecipient(r.Context())
	sources, err := h.store.ListCatalog(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	prefs, err := h.store.ListPreferences(r.Context(), recipient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, catalogResponse{Sources: sourceViews(sources), Preferences: preferenceViews(prefs)})
}

func (h *handler) getPreferences(w http.ResponseWriter, r *http.Request) {
	recipient := h.configuredRecipient(r.Context())
	prefs, err := h.store.ListPreferences(r.Context(), recipient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, preferencesResponse{Preferences: preferenceViews(prefs)})
}

func (h *handler) putPreference(w http.ResponseWriter, r *http.Request) {
	subject, err := h.subject(r)
	if err != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var body putPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid preference body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.SourceScenario) == "" {
		http.Error(w, "source_scenario is required", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := h.store.SetPreference(r.Context(), subject, body.SourceScenario, body.Category, body.Enabled, body.MinUrgency, body.ChannelOverride, body.ScopeKey, now); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, preferenceView{
		SourceScenario:  strings.TrimSpace(body.SourceScenario),
		Category:        strings.TrimSpace(body.Category),
		ScopeKey:        strings.TrimSpace(body.ScopeKey),
		Enabled:         body.Enabled,
		MinUrgency:      strings.TrimSpace(body.MinUrgency),
		ChannelOverride: strings.TrimSpace(body.ChannelOverride),
	})
}

func sourceViews(sources []preferences.Source) []sourceView {
	out := make([]sourceView, 0, len(sources))
	for _, src := range sources {
		view := sourceView{
			SourceScenario: src.SourceScenario,
			DisplayName:    src.DisplayName,
			Icon:           src.Icon,
			DeepLinkBase:   src.DeepLinkBase,
			Declared:       src.Declared,
			Categories:     make([]categoryView, 0, len(src.Categories)),
		}
		for _, cat := range src.Categories {
			view.Categories = append(view.Categories, categoryView{
				Category:          cat.Category,
				DisplayName:       cat.DisplayName,
				Description:       cat.Description,
				Declared:          cat.Declared,
				DefaultEnabled:    cat.DefaultEnabled,
				DefaultMinUrgency: cat.DefaultMinUrgency,
			})
		}
		out = append(out, view)
	}
	return out
}

func preferenceViews(prefs []preferences.Preference) []preferenceView {
	out := make([]preferenceView, 0, len(prefs))
	for _, p := range prefs {
		out = append(out, preferenceView{
			SourceScenario:  p.SourceScenario,
			Category:        p.Category,
			ScopeKey:        p.ScopeKey,
			Enabled:         p.Enabled,
			MinUrgency:      p.MinUrgency,
			ChannelOverride: p.ChannelOverride,
		})
	}
	return out
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
