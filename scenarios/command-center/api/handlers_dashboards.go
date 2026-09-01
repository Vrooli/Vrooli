package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

	readings, sources := s.readings(r.Context(), entries)

	writeJSON(w, http.StatusOK, dashboardResponse{
		Dashboard:   id,
		GeneratedAt: time.Now().UTC(),
		Metrics:     readings,
		Sources:     sources,
	})
}

// readings joins authored coverage with an upstream observation. Coverage is
// never changed by a fetch result; only trust and the value are live fields.
func (s *Server) readings(ctx context.Context, entries []MetricEntry) ([]MetricEntry, map[string]sourceMetadata) {
	out := make([]MetricEntry, len(entries))
	copy(out, entries)
	sources := map[string]sourceMetadata{}
	teamDeclarations := loadTeamDeclarations()
	for i := range out {
		m := &out[i]
		m.Value = nil
		m.ObservedAt = nil
		m.Trust = TrustUnavailable
		if m.Empirical == "" {
			m.Empirical = EmpiricalNone
		}
		if m.Source.Binding == "" {
			m.Source.Binding = string(m.UpstreamSource)
		}
		if m.Source.TTLSeconds == 0 {
			m.Source.TTLSeconds = m.TTLSeconds
		}
		if m.Source.TTLSeconds == 0 {
			m.Source.TTLSeconds = 60
		}
		m.TTLSeconds = m.Source.TTLSeconds
		decl := teamDeclarations[m.Source.Team]
		m.Source.InstrumentStatus = first(m.Source.InstrumentStatus, decl["status"], "partial")
		m.Source.InstrumentArchetype = first(m.Source.InstrumentArchetype, decl["archetype"], "production-ledger")
		if effectiveCoverage(*m) != CoverageNow && effectiveCoverage(*m) != CoverageInReach {
			continue
		}
		m.Coverage = effectiveCoverage(*m)
		src := m.UpstreamSource
		if src == "" {
			src = sourceFromBinding(m.Source.Binding)
		}
		path := m.Source.Read
		if path == "" {
			path = defaultPathFor(src)
		}
		env, err := s.primeSinglePath(ctx, src, path, m.Source.TTLSeconds)
		name := string(src)
		if name == "" {
			name = "none"
		}
		if env.Data != nil {
			sources[name] = sourceMetadata{FromCache: env.FromCache, StalenessTS: env.StalenessTS}
		}
		if env.Data == nil {
			m.TrustReason = unavailableReason(err)
			continue
		}
		pick, known := selectors[m.Source.Select]
		if !known {
			if m.Coverage == CoverageNow {
				m.TrustReason = "no selector named " + m.Source.Select
			}
			continue
		}
		var payload any
		if json.Unmarshal(env.Data, &payload) != nil {
			m.TrustReason = "source returned a body that is not JSON"
			continue
		}
		value, found := pick(payload)
		if !found {
			m.TrustReason = "selector " + m.Source.Select + " found no number in the source payload"
			continue
		}
		m.Value = value
		observed := env.CachedAt
		m.ObservedAt = &observed
		switch {
		case err == nil && time.Since(observed) <= time.Duration(m.TTLSeconds)*time.Second:
			m.Trust = TrustValid
		default:
			// The source stopped answering, or the cached observation aged
			// past its TTL. Either way the last good reading is served with
			// its age, and coverage is untouched.
			m.Trust = TrustCached
			m.TrustReason = unavailableReason(err)
		}
	}
	s.joinPredictions(out)
	return out, sources
}

func unavailableReason(err error) string {
	if err == nil {
		return "observation is older than its TTL"
	}
	if errors.Is(err, upstream.ErrNotAvailable) {
		return "source is not reachable or does not expose this surface"
	}
	return err.Error()
}

func sourceFromBinding(binding string) UpstreamSource {
	if strings.Contains(binding, "swarm") {
		return SourceSwarm
	}
	if strings.Contains(binding, "vrooli") {
		return SourceVrooli
	}
	if strings.Contains(binding, "lpbs") || strings.Contains(binding, "landing") {
		return SourceLPBS
	}
	return SourceNone
}

// primeSources fetches the upstream payloads required by the given entries
// and returns per-source cache metadata. Fetch errors never fail the
// request — the metric renders in gap-mode instead.
func (s *Server) primeSources(ctx context.Context, entries []MetricEntry) map[string]sourceMetadata {
	seen := make(map[UpstreamSource]struct{})
	out := map[string]sourceMetadata{}
	for _, e := range entries {
		if effectiveCoverage(e) != CoverageNow && effectiveCoverage(e) != CoverageInReach {
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
	return s.primeSinglePath(ctx, src, defaultPathFor(src), int(TTLFor(src)/time.Second))
}

func (s *Server) primeSinglePath(ctx context.Context, src UpstreamSource, path string, ttlSeconds int) (Envelope, error) {
	client := s.clientFor(src)
	if client == nil {
		return Envelope{}, upstream.ErrNotAvailable
	}
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
	ttl := TTLFor(src)
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}
	s.cache.Put(key, env, ttl)
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
