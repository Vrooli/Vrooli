package devices

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"device-control/internal/identity"
)

// RelationEvidence is intentionally non-authoritative. It explains why two
// transport records may describe one appliance without changing identity
// ownership or merging durable records.
type RelationEvidence struct {
	Kind       string    `json:"kind"`
	Left       string    `json:"left"`
	Right      string    `json:"right"`
	Detail     string    `json:"detail"`
	Confidence float64   `json:"confidence"`
	ObservedAt time.Time `json:"observed_at"`
}

type CorrelationCandidate struct {
	ID          string             `json:"id"`
	LeftID      string             `json:"left_id"`
	RightID     string             `json:"right_id"`
	Confidence  float64            `json:"confidence"`
	Disposition string             `json:"disposition"`
	Evidence    []RelationEvidence `json:"evidence"`
	CreatedAt   time.Time          `json:"created_at"`
	ExpiresAt   time.Time          `json:"expires_at,omitempty"`
}

// NormalizeEndpoint strips transport-specific presentation differences while
// retaining the host as the only relation signal. A host is never a durable
// identity claim.
func NormalizeEndpoint(endpoint string) string {
	cleanHost := func(host string) string { return strings.TrimSuffix(strings.Trim(strings.TrimSpace(host), "[]"), ".") }
	endpoint = strings.TrimSpace(strings.ToLower(endpoint))
	if endpoint == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(endpoint); err == nil {
		return cleanHost(host)
	}
	if strings.HasPrefix(endpoint, "[") && strings.Contains(endpoint, "]") {
		return cleanHost(endpoint[1:strings.Index(endpoint, "]")])
	}
	if host, _, err := net.SplitHostPort("[" + endpoint + "]:0"); err == nil {
		return cleanHost(host)
	}
	return cleanHost(endpoint)
}

// DetectCorrelationCandidates compares independent observations. It only
// emits candidates; callers must use Store.Merge with an owner assertion or a
// shared hardware claim to create a durable composition.
func DetectCorrelationCandidates(records []Record, now time.Time) []CorrelationCandidate {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	physical := make([]Record, 0, len(records))
	for _, record := range records {
		if record.Kind == "physical" || record.Kind == "emulator" {
			physical = append(physical, record)
		}
	}
	var out []CorrelationCandidate
	for i := 0; i < len(physical); i++ {
		for j := i + 1; j < len(physical); j++ {
			left, right := physical[i], physical[j]
			if left.ID == right.ID || identity.ClaimsMatch(left.Claims, right.Claims) {
				continue
			}
			evidence := relationEvidence(left, right, now)
			if len(evidence) == 0 {
				continue
			}
			confidence := 0.0
			for _, item := range evidence {
				if item.Confidence > confidence {
					confidence = item.Confidence
				}
			}
			if len(evidence) > 1 && confidence < 0.85 {
				confidence += 0.1
			}
			out = append(out, CorrelationCandidate{ID: "candidate:" + pairID(left.ID, right.ID), LeftID: left.ID, RightID: right.ID, Confidence: minConfidence(confidence), Disposition: "unconfirmed", Evidence: evidence, CreatedAt: now})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func relationEvidence(left, right Record, now time.Time) []RelationEvidence {
	var out []RelationEvidence
	for _, leftEndpoint := range recordEndpoints(left) {
		leftHost := NormalizeEndpoint(leftEndpoint)
		if leftHost == "" {
			continue
		}
		for _, rightEndpoint := range recordEndpoints(right) {
			if leftHost == NormalizeEndpoint(rightEndpoint) {
				out = append(out, RelationEvidence{Kind: "same-host", Left: leftEndpoint, Right: rightEndpoint, Detail: "independent transports observed at the same normalized host", Confidence: 0.72, ObservedAt: now})
				return appendNameAndModelEvidence(out, left, right, now)
			}
		}
	}
	return appendNameAndModelEvidence(out, left, right, now)
}

func recordEndpoints(record Record) []string {
	endpoints := []string{record.Endpoint}
	for _, profile := range record.Transports {
		endpoints = append(endpoints, profile.Endpoint)
		endpoints = append(endpoints, profile.Endpoints...)
	}
	return uniqueStrings(endpoints)
}

func appendNameAndModelEvidence(out []RelationEvidence, left, right Record, now time.Time) []RelationEvidence {
	leftName, rightName := strings.TrimSpace(strings.ToLower(left.Name)), strings.TrimSpace(strings.ToLower(right.Name))
	if leftName != "" && leftName == rightName {
		out = append(out, RelationEvidence{Kind: "same-friendly-name", Left: left.Name, Right: right.Name, Detail: "friendly names match; not a durable identity claim", Confidence: 0.48, ObservedAt: now})
	}
	leftModel, rightModel := strings.TrimSpace(strings.ToLower(left.Model)), strings.TrimSpace(strings.ToLower(right.Model))
	if leftModel != "" && leftModel == rightModel {
		out = append(out, RelationEvidence{Kind: "same-model", Left: left.Model, Right: right.Model, Detail: "models match; not a durable identity claim", Confidence: 0.4, ObservedAt: now})
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func pairID(left, right string) string {
	if left > right {
		left, right = right, left
	}
	return fmt.Sprintf("%s:%s", left, right)
}

func minConfidence(value float64) float64 {
	if value > 0.99 {
		return 0.99
	}
	return value
}

func (s *Store) CorrelationCandidates(now time.Time) []CorrelationCandidate {
	return DetectCorrelationCandidates(s.List(), now)
}

// ListActionable excludes retained historical rows from normal selection. A
// current observation is always actionable even when its health is degraded;
// callers need to see that typed failure rather than silently lose the target.
func (s *Store) ListActionable(now time.Time, maxAge time.Duration) []Record {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if maxAge <= 0 {
		maxAge = 15 * time.Minute
	}
	out := make([]Record, 0)
	for _, record := range s.List() {
		if record.Kind != "physical" && record.Kind != "emulator" && record.Kind != "desktop" {
			continue
		}
		if record.LastSeenAt.IsZero() || now.Sub(record.LastSeenAt) <= maxAge {
			out = append(out, record)
		}
	}
	return out
}

func (s *Store) Diagnostics(now time.Time, maxAge time.Duration) []Record {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if maxAge <= 0 {
		maxAge = 15 * time.Minute
	}
	out := make([]Record, 0)
	for _, record := range s.List() {
		if !record.LastSeenAt.IsZero() && now.Sub(record.LastSeenAt) <= maxAge {
			continue
		}
		out = append(out, record)
	}
	return out
}

// AggregateHealth projects the worst current transport condition onto the
// logical device. A logical target remains visible when one transport is
// unreachable, but its health explains the limiting transport and reason.
func AggregateHealth(record Record, now time.Time, maxAge time.Duration) (string, string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if maxAge <= 0 {
		maxAge = 15 * time.Minute
	}
	bestRank := healthRank(record.Health)
	best := normalizeHealth(record.Health)
	reason := strings.TrimSpace(record.HealthReason)
	if best == "" {
		best = normalizeHealth(record.Status)
		bestRank = healthRank(best)
	}
	for _, transport := range record.Transports {
		health := normalizeHealth(transport.Health)
		if health == "" {
			health = "available"
		}
		if !transport.ObservedAt.IsZero() && now.Sub(transport.ObservedAt) > maxAge && healthRank("stale") > healthRank(health) {
			health = "stale"
		}
		if healthRank(health) > bestRank {
			bestRank = healthRank(health)
			best = health
			reason = strings.TrimSpace(transport.HealthReason)
			if reason == "" {
				reason = transport.Name + " transport is " + health
			}
		}
	}
	if best == "" {
		best = "unknown"
	}
	if reason == "" && best != "available" {
		reason = "logical device health is " + best
	}
	return best, reason
}

func normalizeHealth(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "healthy", "ready", "available":
		return "available"
	case "degraded":
		return "degraded"
	case "stale":
		return "stale"
	case "unreachable", "offline", "unavailable":
		return "unreachable"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func healthRank(value string) int {
	switch normalizeHealth(value) {
	case "available":
		return 0
	case "degraded":
		return 1
	case "stale":
		return 2
	case "unreachable":
		return 3
	default:
		return 1
	}
}
