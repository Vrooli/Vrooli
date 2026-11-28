package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"command-center/internal/trends"
	"command-center/upstream"
)

// dashboardResponse is the shape returned by GET /api/v1/dashboards/{id}.
type dashboardResponse struct {
	Dashboard   string                    `json:"dashboard"`
	GeneratedAt time.Time                 `json:"generated_at"`
	Metrics     []MetricEntry             `json:"metrics"`
	Sources     map[string]sourceMetadata `json:"sources"`
}

func numericReading(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	case float32:
		return float64(v), !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

type sourceMetadata struct {
	FromCache         bool              `json:"from_cache"`
	StalenessTS       *time.Time        `json:"staleness_ts,omitempty"`
	IntegrationID     string            `json:"integration_id,omitempty"`
	IntegrationStatus string            `json:"integration_status,omitempty"`
	IntegrationReason string            `json:"integration_reason_code,omitempty"`
	FeatureStatus     map[string]string `json:"feature_status,omitempty"`
	Origin            string            `json:"origin,omitempty"`
	OriginEnv         string            `json:"origin_env,omitempty"`
	OriginDisplay     string            `json:"origin_display,omitempty"`
}

type capabilityStateView struct {
	Status        string
	ReasonCode    string
	FeatureStatus map[string]string
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
	setpoint, err := s.readSetpoint()
	if err != nil {
		writeSetpointReadError(w, err)
		return
	}

	readings, sources := s.readings(r.Context(), entries)
	applySetpoint(readings, setpoint)

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
	observations := s.fetchObservations(ctx, entries)
	sources := map[string]sourceMetadata{}
	integrations := map[string]capabilityStateView{}
	for _, state := range s.integrationSnapshot(ctx, false).States {
		integrations[state.ID] = capabilityStateView{Status: string(state.Status), ReasonCode: state.ReasonCode, FeatureStatus: state.FeatureStatus}
	}
	teamDeclarations := s.teamDeclarations(ctx)
	for i := range out {
		m := &out[i]
		m.Value = nil
		m.Rows = nil
		m.Ladder = nil
		m.ObservedAt = nil
		m.Trend = nil
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
		m.Origin, m.OriginEnv, m.OriginDisplay = s.readingOrigin(m.Source)
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
		observation := observations[m.ID]
		env, err := observation.envelope, observation.err
		name := string(src)
		if name == "" {
			name = "none"
		}
		integrationID := first(m.Source.IntegrationID, strings.TrimPrefix(strings.TrimPrefix(m.Source.Binding, "scenario:"), "resource:"), name)
		state := integrations[integrationID]
		metadata := sourceMetadata{IntegrationID: integrationID, IntegrationStatus: state.Status, IntegrationReason: state.ReasonCode, FeatureStatus: state.FeatureStatus}
		metadata.Origin, metadata.OriginEnv, metadata.OriginDisplay = s.readingOrigin(m.Source)
		if env.Data != nil {
			metadata.FromCache, metadata.StalenessTS = env.FromCache, env.StalenessTS
		}
		if env.Data == nil {
			m.TrustReason = unavailableReason(err)
			sources[name] = metadata
			continue
		}
		candidate, cacheable := qualifyMetricObservation(*m, env, err)
		key := observationCacheKey(src, path, m.ID)
		if cacheable && observation.commit {
			candidate, env = commitQualifiedObservation(s.cache, key, *m, candidate, env, time.Duration(m.Source.TTLSeconds)*time.Second)
			metadata.FromCache, metadata.StalenessTS = env.FromCache, env.StalenessTS
		} else if !cacheable && !env.FromCache && err == nil {
			// A timestamped response can still violate this reading's contract.
			// Keep the previous qualified value and expose its degraded trust.
			s.cache.MarkStale(key, time.Now().UTC())
			if previous, _, ok := s.cache.Get(key); ok {
				candidate, _ = qualifyMetricObservation(*m, previous, errors.New(candidate.TrustReason))
				metadata.FromCache, metadata.StalenessTS = true, previous.StalenessTS
			}
		}
		sources[name] = metadata
		*m = candidate
		if m.Trust == TrustValid && m.TrendPolicy != nil && m.TrendPolicy.Enabled && m.ObservedAt != nil {
			value, ok := numericReading(m.Value)
			if ok && s.trendStore != nil {
				_ = s.trendStore.Record(ctx, trends.Observation{MetricID: m.ID, Source: string(src), Value: value, Observed: *m.ObservedAt})
				trend, trendErr := s.trendStore.Trend(ctx, m.ID, string(src), *m.TrendPolicy, *m.ObservedAt)
				if trendErr == nil {
					m.Trend = &trend
				}
			}
		}
	}
	s.joinPredictions(out)
	return out, sources
}

func (s *Server) readingOrigin(binding SourceBinding) (string, string, string) {
	if binding.IntegrationID == "landing-page-business-suite" {
		switch origin := strings.ToLower(strings.TrimSpace(os.Getenv("COMMAND_CENTER_LPBS_ORIGIN"))); origin {
		case "local":
			if spec, ok := s.registry.Origins["local"]; ok {
				return "local", first(spec.Environment, "local"), first(spec.Display, "Local instance")
			}
			return "local", "local", "Local instance"
		case "override":
			return "override", "override", "Local LPBS override"
		case "production":
			if spec, ok := s.registry.Origins["production"]; ok {
				return "production", first(spec.Environment, "production"), first(spec.Display, "vrooli.com")
			}
		}
	}
	origin := first(binding.Origin, "local")
	if spec, ok := s.registry.Origins[origin]; ok {
		return origin, first(spec.Environment, "local"), first(spec.Display, origin)
	}
	return origin, "local", origin
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
	if strings.Contains(binding, "offer-desk") {
		return SourceOffer
	}
	if strings.Contains(binding, "deployment-manager") {
		return SourceDeploy
	}
	return SourceNone
}

func observationCacheKey(src UpstreamSource, path, metricID string) string {
	return string(src) + ":" + path + ":" + metricID
}

func (s *Server) clientFor(src UpstreamSource) upstream.Client {
	if provider, ok := s.providers[src]; ok {
		return provider.client()
	}
	return nil
}

func defaultPathFor(src UpstreamSource) string {
	defaults := map[UpstreamSource]string{
		SourceSwarm:  "/api/v1/stats",
		SourceVrooli: "/scenarios",
		SourceLPBS:   "/api/v1/admin/dashboard/summary",
		SourceOffer:  "/vrooli.offer_desk.v1.offers.ReleaseLadderService/GetReleaseLadder",
		SourceDeploy: "/api/v1/readiness/state",
	}
	if path, ok := defaults[src]; ok {
		return path
	}
	return "/"
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
