package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"command-center/upstream"
)

// errMissingObservationTime rejects producer responses that cannot be tied to
// a timestamped observation.
var errMissingObservationTime = errors.New("upstream response has no valid producer observation timestamp")

// qualifyMetricObservation owns payload validation and trust qualification for
// every dashboard reading kind. Fetch/cache orchestration stays in the handler.
func qualifyMetricObservation(entry MetricEntry, env Envelope, fetchErr error) (MetricEntry, bool) {
	m := entry
	m.Value, m.Rows, m.Ladder, m.ObservedAt, m.Trend = nil, nil, nil, nil, nil
	m.Trust, m.TrustReason = TrustUntrusted, ""
	selectorID := first(m.Source.Selector, m.Source.Select)
	pick, known := selectors[selectorID]
	panelPick, panelKnown := panelSelectors[selectorID]
	ladderPick, ladderKnown := ladderSelectors[selectorID]
	posturePick, postureKnown := postureSelectors[selectorID]
	if !known && !panelKnown && !ladderKnown && !postureKnown {
		m.TrustReason = "no selector named " + selectorID
		return m, false
	}
	var payload any
	if json.Unmarshal(env.Data, &payload) != nil {
		m.TrustReason = "source returned a body that is not JSON"
		return m, false
	}
	if version := sourceContractVersion(env.Data); version != "" && m.Source.ContractVersion != "" && version != m.Source.ContractVersion {
		m.TrustReason = "producer contract version " + version + " does not match expected " + m.Source.ContractVersion
		return m, false
	}
	if unit := sourceUnit(env.Data, selectorID); unit != "" && !sameUnit(unit, m.Source.ExpectedUnit, m.Unit) {
		m.TrustReason = "producer unit " + unit + " does not match expected " + first(m.Source.ExpectedUnit, m.Unit)
		return m, false
	}
	if strings.EqualFold(m.Kind, "ladder") {
		if !ladderKnown {
			m.TrustReason = "no ladder selector named " + selectorID
			return m, false
		}
		ladder, found := ladderPick(payload)
		if !found {
			m.TrustReason = "ladder selector " + selectorID + " found no plausible schedule in the source payload"
			return m, false
		}
		m.Ladder, m.Rows, m.Value = ladder, ladderRows(ladder), float64(len(ladder.Rungs))
		m.trustFromProducerTime(env, fetchErr)
		return m, m.Trust == TrustValid
	}
	if strings.EqualFold(m.Kind, "posture") {
		if !postureKnown {
			m.TrustReason = "no posture selector named " + selectorID
			return m, false
		}
		posture, found := posturePick(payload)
		if !found {
			m.TrustReason = "posture selector " + selectorID + " found no position in the source payload"
			return m, false
		}
		m.Value = posture
		m.trustFromProducerTime(env, fetchErr)
		return m, m.Trust == TrustValid
	}
	if strings.EqualFold(m.Kind, "panel") || strings.EqualFold(m.Kind, "funnel") || strings.EqualFold(m.Kind, "leaderboard") {
		if !panelKnown {
			m.TrustReason = "no panel selector named " + selectorID
			return m, false
		}
		rows, found := panelPick(payload)
		if !found {
			m.TrustReason = "panel selector " + selectorID + " found no rows in the source payload"
			return m, false
		}
		plausible := plausiblePanelRows(rows, 6, true)
		if selectorID == "goal_progress" {
			plausible = plausibleProgressRows(rows, 6)
		}
		if strings.EqualFold(m.Kind, "panel") && !plausible {
			m.TrustReason = "panel selector " + selectorID + " returned implausible rows"
			return m, false
		}
		m.Rows = rows
		var total float64
		for _, row := range rows {
			total += row.Value
		}
		m.Value = total
		m.trustFromProducerTime(env, fetchErr)
		return m, m.Trust == TrustValid
	}
	value, found := pick(payload)
	if !found {
		m.TrustReason = "selector " + selectorID + " found no number in the source payload"
		return m, false
	}
	unit := first(m.Unit, m.Source.ExpectedUnit)
	if !plausibleMetricValue(value, unit, producerSampleSize(payload, selectorID)) {
		m.TrustReason = "selector " + selectorID + " returned an implausible value for unit " + unit
		return m, false
	}
	m.Value = value
	m.trustFromProducerTime(env, fetchErr)
	return m, m.Trust == TrustValid
}

// trustFromProducerTime qualifies a structured reading (panel or ladder) by the
// same bounded producer-time policy used for scalar readings. A reading without
// producer time is never trusted.
func (m *MetricEntry) trustFromProducerTime(env Envelope, err error) {
	if env.ObservationAt == nil {
		m.Trust = TrustUntrusted
		m.TrustReason = "producer did not supply observation time"
		return
	}
	observed := env.ObservationAt.UTC()
	m.ObservedAt = &observed
	m.Trust, m.TrustReason = qualifyProducerTime(observed, time.Duration(m.TTLSeconds)*time.Second, err)
}

// qualifyProducerTime is the single trust policy for scalar and structured
// observations. Last-good values remain usable briefly, but become unavailable
// after two TTLs instead of looking cached forever.
func qualifyProducerTime(observed time.Time, ttl time.Duration, err error) (Trust, string) {
	now := time.Now().UTC()
	age := now.Sub(observed)
	switch {
	case err == nil && !observed.After(now) && age <= ttl:
		return TrustValid, ""
	case observed.After(now):
		return TrustUntrusted, "producer observation time is in the future"
	case age <= ttl*2:
		return TrustCached, cachedReason(age, ttl, err)
	default:
		return TrustUnavailable, "producer observation is stale"
	}
}

func plausiblePanelRows(rows []PanelRow, limit int, exhaustive bool) bool {
	if len(rows) == 0 {
		return exhaustive
	}
	if len(rows) > limit {
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

func plausibleProgressRows(rows []PanelRow, limit int) bool {
	if len(rows) > limit {
		return false
	}
	for _, row := range rows {
		if row.Key == "" || row.Label == "" || row.Value < 0 || row.Share < 0 || row.Share > 1 || math.IsNaN(row.Value) || math.IsNaN(row.Share) {
			return false
		}
	}
	return true
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
