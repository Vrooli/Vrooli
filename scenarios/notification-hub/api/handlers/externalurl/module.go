// Package externalurl exposes browser-openable scenario URLs to trusted UI
// callers without requiring them to know host ports or tunnel domains.
package externalurl

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"

	"notification-hub/internal/module"
)

const allowedScenario = "web-console"

type resolver func(context.Context, string) (string, error)

type response struct {
	Scenario string `json:"scenario"`
	URL      string `json:"url"`
}

// Module returns the read-only external URL route. Resolver injection keeps
// route behavior testable without Tunnel Manager or host discovery.
func Module() module.Module { return moduleWithResolver(discovery.ResolveExternalURLForNotice) }

func moduleWithResolver(resolve resolver) module.Module {
	return module.Module{Name: "externalurl", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/external-url", func(w http.ResponseWriter, req *http.Request) {
			scenario := strings.TrimSpace(req.URL.Query().Get("scenario"))
			out := response{Scenario: scenario}
			if scenario == allowedScenario && resolve != nil {
				if url, err := resolve(req.Context(), scenario); err == nil {
					out.URL = strings.TrimSpace(url)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}
