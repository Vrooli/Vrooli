package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/discovery"
)

// scenarioLinkResponse is the browser-openable URL of another scenario,
// resolved relative to the origin this browser used: a tunnel subdomain when
// the request arrived through one, a localhost port otherwise.
type scenarioLinkResponse struct {
	Scenario string `json:"scenario"`
	URL      string `json:"url"`
}

// handleExternalURL answers a Web Console UI feature that needs a link a person
// can open to reach another scenario's interface — e.g. the Notifications tab
// pointing to Notification Hub. The link must work from the operator's phone, so
// it has to be the scenario's public tunnel URL, not a localhost port.
//
// It resolves through ResolveExternalURLForNotice, which asks Tunnel Manager for
// the scenario's public URL and falls back to a localhost port only when the
// scenario is not exposed. Deriving from the inbound request host does NOT work
// here: the Cloudflare tunnel rewrites Host to the local origin before the
// request reaches Web Console, so a host-based resolver always sees loopback and
// yields localhost. Tunnel Manager is the authoritative source either way.
//
// An unresolvable scenario (not installed, not exposed, no UI port) yields an
// empty url rather than an error: the caller renders plain text, not a dead link.
func handleExternalURL(w http.ResponseWriter, r *http.Request) {
	scenario := strings.TrimSpace(r.URL.Query().Get("scenario"))
	resp := scenarioLinkResponse{Scenario: scenario}
	if scenario != "" {
		if url, err := discovery.ResolveExternalURLForNotice(r.Context(), scenario); err == nil {
			resp.URL = url
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
