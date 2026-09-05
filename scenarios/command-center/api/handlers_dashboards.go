package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
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
	integrations := map[string]capabilityStateView{}
	for _, state := range s.integrationSnapshot(ctx, false).States {
		integrations[state.ID] = capabilityStateView{Status: string(state.Status), ReasonCode: state.ReasonCode, FeatureStatus: state.FeatureStatus}
	}
	teamDeclarations := s.teamDeclarations(ctx)
	for i := range out {
		m := &out[i]
		m.Value = nil
		m.Rows = nil
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
		env, err := s.fetchObservation(ctx, src, path, m.Source.TTLSeconds)
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
		sources[name] = metadata
		if env.Data == nil {
			m.TrustReason = unavailableReason(err)
			continue
		}
		selectorID := first(m.Source.Selector, m.Source.Select)
		pick, known := selectors[selectorID]
		panelPick, panelKnown := panelSelectors[selectorID]
		if !known && !panelKnown {
			m.Trust = TrustUntrusted
			m.TrustReason = "no selector named " + selectorID
			continue
		}
		var payload any
		if json.Unmarshal(env.Data, &payload) != nil {
			m.Trust = TrustUntrusted
			m.TrustReason = "source returned a body that is not JSON"
			continue
		}
		if version := sourceContractVersion(env.Data); version != "" && m.Source.ContractVersion != "" && version != m.Source.ContractVersion {
			m.Trust = TrustUntrusted
			m.TrustReason = "producer contract version " + version + " does not match expected " + m.Source.ContractVersion
			continue
		}
		if unit := sourceUnit(env.Data, selectorID); unit != "" && !sameUnit(unit, m.Source.ExpectedUnit, m.Unit) {
			m.Trust = TrustUntrusted
			m.TrustReason = "producer unit " + unit + " does not match expected " + first(m.Source.ExpectedUnit, m.Unit)
			continue
		}
		if strings.EqualFold(m.Kind, "panel") {
			if !panelKnown {
				m.Trust = TrustUntrusted
				m.TrustReason = "no panel selector named " + selectorID
				continue
			}
			rows, found := panelPick(payload)
			if !found {
				m.Trust = TrustUntrusted
				m.TrustReason = "panel selector " + selectorID + " found no rows in the source payload"
				continue
			}
			if !plausiblePanelRows(rows, 6, true) {
				m.Trust = TrustUntrusted
				m.TrustReason = "panel selector " + selectorID + " returned implausible rows"
				continue
			}
			m.Rows = rows
			if env.ObservationAt == nil {
				m.Trust = TrustUntrusted
				m.TrustReason = "producer did not supply observation time"
				continue
			}
			observed := env.ObservationAt.UTC()
			m.ObservedAt = &observed
			now := time.Now().UTC()
			age := now.Sub(observed)
			switch {
			case err == nil && !observed.After(now) && age <= time.Duration(m.TTLSeconds)*time.Second:
				m.Trust = TrustValid
			case observed.After(now):
				m.Trust = TrustUntrusted
				m.TrustReason = "producer observation time is in the future"
			case age <= time.Duration(m.TTLSeconds*2)*time.Second:
				m.Trust = TrustCached
			default:
				m.Trust = TrustUnavailable
				m.TrustReason = "producer observation is stale"
			}
			continue
		}
		value, found := pick(payload)
		if !found {
			m.Trust = TrustUntrusted
			m.TrustReason = "selector " + selectorID + " found no number in the source payload"
			continue
		}
		unit := first(m.Unit, m.Source.ExpectedUnit)
		sampleSize := producerSampleSize(payload, selectorID)
		if !plausibleMetricValue(value, unit, sampleSize) {
			m.Trust = TrustUntrusted
			m.TrustReason = "selector " + selectorID + " returned an implausible value for unit " + unit
			continue
		}
		m.Value = value
		if env.ObservationAt == nil {
			m.Trust = TrustUntrusted
			m.TrustReason = "producer did not supply observation time"
			continue
		}
		observed := env.ObservationAt.UTC()
		m.ObservedAt = &observed
		now := time.Now().UTC()
		age := now.Sub(observed)
		switch {
		case err == nil && !observed.After(now) && age <= time.Duration(m.TTLSeconds)*time.Second:
			m.Trust = TrustValid
		case observed.After(now):
			m.Trust = TrustUntrusted
			m.TrustReason = "producer observation time is in the future"
		default:
			// The source stopped answering, or the cached observation aged
			// past its TTL. Either way the last good reading is served with
			// its age, and coverage is untouched.
			m.Trust = TrustCached
			m.TrustReason = cachedReason(age, time.Duration(m.TTLSeconds)*time.Second, err)
		}
		if m.TrendPolicy != nil && m.TrendPolicy.Enabled && m.Trust == TrustValid && m.ObservedAt != nil {
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

func plausiblePanelRows(rows []PanelRow, limit int, exhaustive bool) bool {
	if len(rows) == 0 || len(rows) > limit {
		return false
	}
	total := 0.0
	for _, row := range rows {
		if row.Key == "" || row.Label == "" || row.Value < 0 || row.Share < 0 || row.Share > 1 || math.IsNaN(row.Share) {
			return false
		}
		total += row.Share
	}
	return !exhaustive || math.Abs(total-1) <= 0.02
}

func (s *Server) readingOrigin(binding SourceBinding) (string, string, string) {
	origin := first(binding.Origin, "local")
	if spec, ok := s.registry.Origins[origin]; ok {
		return origin, first(spec.Environment, "local"), first(spec.Display, origin)
	}
	return origin, "local", origin
}

const plausiblePercentSampleFloor = 100

// plausibleMetricValue rejects values that are structurally inconsistent with
// the declared unit. A sub-1 percent value with a meaningful sample is more
// likely to be a 0..1 ratio accidentally labelled as a percent.
func plausibleMetricValue(value float64, unit string, sampleSize int) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	switch strings.ToLower(unit) {
	case "count", "integer", "usd", "currency", "seconds", "minutes":
		return value >= 0
	case "percent", "%":
		if value < 0 || value > 100 {
			return false
		}
		return !(value > 0 && value < 1 && sampleSize >= plausiblePercentSampleFloor)
	default:
		return true
	}
}

func producerSampleSize(payload any, selector string) int {
	root, ok := payload.(map[string]any)
	if !ok {
		return 0
	}
	for _, key := range []string{"sample_size", "sampleSize"} {
		if value, ok := root[key].(float64); ok && value >= 0 {
			return int(value)
		}
	}
	if samples, ok := root["sample_sizes"].(map[string]any); ok {
		if value, ok := samples[selector].(float64); ok && value >= 0 {
			return int(value)
		}
	}
	return 0
}

func cachedReason(age, ttl time.Duration, err error) string {
	if err != nil && age <= ttl {
		return err.Error()
	}
	return fmt.Sprintf("observation age %s exceeds TTL %s", age.Round(time.Millisecond), ttl)
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
	if strings.Contains(binding, "offer-desk") {
		return SourceOffer
	}
	if strings.Contains(binding, "deployment-manager") {
		return SourceDeploy
	}
	return SourceNone
}

func (s *Server) fetchObservation(ctx context.Context, src UpstreamSource, path string, ttlSeconds int) (Envelope, error) {
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
	env.ObservationAt = observationTime(raw)
	ttl := TTLFor(src)
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}
	s.cache.Put(key, env, ttl)
	return env, nil
}

func observationTime(raw json.RawMessage) *time.Time {
	var v map[string]any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	for _, key := range []string{"observedAt", "observed_at", "observationTime", "timestamp", "generatedAt", "generated_at"} {
		if value, ok := v[key].(string); ok {
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				return ptrTime(t)
			}
		}
	}
	return nil
}

// sourceContractVersion and sourceUnit inspect optional metadata emitted by a
// producer envelope. Older compatibility projections may omit these fields;
// the binding's declared contract remains authoritative in that case. When a
// producer does emit metadata, disagreement is deterministic evidence of an
// incompatible observation and must not be presented as trustworthy.
func sourceContractVersion(raw json.RawMessage) string {
	var envelope struct {
		ContractVersion      string `json:"contractVersion"`
		ContractVersionSnake string `json:"contract_version"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return ""
	}
	return first(envelope.ContractVersion, envelope.ContractVersionSnake)
}

func sourceUnit(raw json.RawMessage, selector string) string {
	var envelope struct {
		Unit  string            `json:"unit"`
		Units map[string]string `json:"units"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return ""
	}
	if unit := strings.TrimSpace(envelope.Units[selector]); unit != "" {
		return unit
	}
	return strings.TrimSpace(envelope.Unit)
}

func sameUnit(actual string, expected ...string) bool {
	actual = strings.ToLower(strings.TrimSpace(actual))
	for _, candidate := range expected {
		if candidate != "" && actual == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
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
