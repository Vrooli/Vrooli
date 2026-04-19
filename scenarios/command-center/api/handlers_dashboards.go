package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"command-center/upstream"
)

// dashboardResponse is the shape returned by GET /api/v1/dashboards/{id}.
type dashboardResponse struct {
	Dashboard   string                    `json:"dashboard"`
	GeneratedAt time.Time                 `json:"generated_at"`
	Metrics     []MetricEntry             `json:"metrics"`
	Sources     map[string]sourceMetadata `json:"sources"`
}

type sourceMetadata struct {
	FromCache   bool       `json:"from_cache"`
	StalenessTS *time.Time `json:"staleness_ts,omitempty"`
}

func (s *Server) registerDashboardRoutes() {
	s.router.HandleFunc("/api/v1/dashboards/{id}", s.handleDashboard).Methods("GET")
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	entries := s.registry.Dashboard(id)
	if entries == nil {
		writeError(w, http.StatusNotFound, "dashboard_not_found",
			"Unknown dashboard id: "+id, nil)
		return
	}

	sources := s.primeSources(r.Context(), entries)

	writeJSON(w, http.StatusOK, dashboardResponse{
		Dashboard:   id,
		GeneratedAt: time.Now().UTC(),
		Metrics:     entries,
		Sources:     sources,
	})
}

// primeSources fetches the upstream payloads required by the given entries
// and returns per-source cache metadata. Fetch errors never fail the
// request — the metric renders in gap-mode instead.
func (s *Server) primeSources(ctx context.Context, entries []MetricEntry) map[string]sourceMetadata {
	seen := make(map[UpstreamSource]struct{})
	out := map[string]sourceMetadata{}
	for _, e := range entries {
		if e.DataSource != StatusLive && e.DataSource != StatusPartial {
			continue
		}
		if _, ok := seen[e.UpstreamSource]; ok {
			continue
		}
		seen[e.UpstreamSource] = struct{}{}

		env, err := s.primeSingle(ctx, e.UpstreamSource)
		switch {
		case errors.Is(err, upstream.ErrNotAvailable):
			out[string(e.UpstreamSource)] = sourceMetadata{FromCache: false, StalenessTS: ptrTime(time.Now().UTC())}
		case err != nil:
			out[string(e.UpstreamSource)] = sourceMetadata{FromCache: env.FromCache, StalenessTS: env.StalenessTS}
		default:
			out[string(e.UpstreamSource)] = sourceMetadata{FromCache: env.FromCache, StalenessTS: env.StalenessTS}
		}
	}
	return out
}

func (s *Server) primeSingle(ctx context.Context, src UpstreamSource) (Envelope, error) {
	client := s.clientFor(src)
	if client == nil {
		return Envelope{}, upstream.ErrNotAvailable
	}
	path := defaultPathFor(src)
	key := string(src) + ":" + path

	// Serve fresh cache when available.
	if env, fresh, ok := s.cache.Get(key); ok && fresh {
		return env, nil
	}

	raw, err := client.Fetch(ctx, path)
	if err != nil {
		s.cache.MarkStale(key, time.Now().UTC())
		if env, _, ok := s.cache.Get(key); ok {
			return env, err
		}
		return Envelope{}, err
	}

	env := Envelope{
		Data:      raw,
		CachedAt:  time.Now().UTC(),
		FromCache: false,
		Source:    string(src),
	}
	s.cache.Put(key, env, TTLFor(src))
	return env, nil
}

func (s *Server) clientFor(src UpstreamSource) upstream.Client {
	switch src {
	case SourceSwarm:
		return s.swarm
	case SourceVrooli:
		return s.vrooli
	case SourceLPBS:
		return s.lpbs
	default:
		return nil
	}
}

func defaultPathFor(src UpstreamSource) string {
	switch src {
	case SourceSwarm:
		return "/api/v1/stats"
	case SourceVrooli:
		return "/scenarios"
	case SourceLPBS:
		return "/api/v1/admin/dashboard/summary"
	default:
		return "/"
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// writeJSON serialises v to the response with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a standardised error body.
func writeError(w http.ResponseWriter, status int, code, message string, details any) {
	body := map[string]any{
		"error":   code,
		"message": message,
	}
	if details != nil {
		body["details"] = details
	}
	writeJSON(w, status, body)
}
