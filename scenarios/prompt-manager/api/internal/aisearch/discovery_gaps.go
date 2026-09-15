package aisearch

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"prompt-manager/internal/store"
)

// discoveryMissThreshold is the top-result score below which a discover call is
// treated as a miss (in addition to the zero-results case). Tunable.
const discoveryMissThreshold = 0.45

// DiscoveryMissStore is the telemetry seam: the production implementation is
// *store.DiscoveryMissStore; tests inject a fake so capture and gaps are
// deterministic without a real filesystem home.
type DiscoveryMissStore interface {
	Append(miss store.DiscoveryMiss) error
	ReadSince(window time.Duration) ([]store.DiscoveryMiss, error)
}

// SetDiscoveryMissStore wires the discovery-miss telemetry sink/source. When
// unset, capture is a no-op and gaps return empty.
func (s *Service) SetDiscoveryMissStore(missStore DiscoveryMissStore) {
	s.missStore = missStore
}

// DiscoveryGapCluster is a group of near-duplicate missed queries within the
// requested window. Count is window-relative, never an all-time absolute.
type DiscoveryGapCluster struct {
	Query    string   `json:"query"` // normalized representative
	Count    int      `json:"count"`
	LastSeen string   `json:"lastSeen"`
	Types    []string `json:"types,omitempty"`
	Examples []string `json:"examples,omitempty"`
}

// recordDiscoveryMiss appends a miss record when a discover call returned
// nothing useful. It is guarded and log-and-continue: a telemetry failure must
// never fail or slow the discover response.
func (s *Service) recordDiscoveryMiss(ctx context.Context, resp *DiscoverResponse, discoverType, complexity string) {
	if s.missStore == nil || resp == nil {
		return
	}
	topScore := 0.0
	for _, result := range resp.Results {
		if result.Score > topScore {
			topScore = result.Score
		}
	}
	if len(resp.Results) > 0 && topScore >= discoveryMissThreshold {
		return // a useful result exists — not a miss
	}
	typeFilter := discoverType
	if typeFilter == "" {
		typeFilter = "skill"
	}
	miss := store.DiscoveryMiss{
		Query:       resp.Query,
		Type:        typeFilter,
		TopScore:    topScore,
		ResultCount: len(resp.Results),
		Complexity:  complexity,
		Caller:      CallerFrom(ctx),
	}
	if err := s.missStore.Append(miss); err != nil {
		log.Printf("[aisearch] discovery-miss append failed (continuing): %v", err)
	}
}

// DiscoveryGaps reads the miss window, optionally filters by type, clusters
// near-duplicate queries (normalized), and returns clusters sorted by count.
func (s *Service) DiscoveryGaps(window time.Duration, typeFilter string) ([]DiscoveryGapCluster, error) {
	if s.missStore == nil {
		return []DiscoveryGapCluster{}, nil
	}
	misses, err := s.missStore.ReadSince(window)
	if err != nil {
		return nil, err
	}
	// An empty filter means "all types" — do NOT normalize it, since
	// normalizeDiscoverType("") returns "skill" and would wrongly exclude
	// action/all misses.
	if typeFilter != "" {
		typeFilter = normalizeDiscoverType(typeFilter)
	}

	type accumulator struct {
		query    string
		count    int
		lastSeen string
		types    map[string]bool
		examples []string
	}
	groups := map[string]*accumulator{}
	for _, miss := range misses {
		if typeFilter != "" && typeFilter != "all" && normalizeDiscoverType(miss.Type) != typeFilter {
			continue
		}
		key := normalizeQuery(miss.Query)
		if key == "" {
			continue
		}
		group, ok := groups[key]
		if !ok {
			group = &accumulator{query: key, types: map[string]bool{}}
			groups[key] = group
		}
		group.count++
		if miss.At > group.lastSeen {
			group.lastSeen = miss.At
		}
		if t := normalizeDiscoverType(miss.Type); t != "" {
			group.types[t] = true
		}
		if raw := strings.TrimSpace(miss.Query); raw != "" && len(group.examples) < 3 && !containsString(group.examples, raw) {
			group.examples = append(group.examples, raw)
		}
	}

	clusters := make([]DiscoveryGapCluster, 0, len(groups))
	for _, group := range groups {
		types := make([]string, 0, len(group.types))
		for t := range group.types {
			types = append(types, t)
		}
		sort.Strings(types)
		clusters = append(clusters, DiscoveryGapCluster{
			Query:    group.query,
			Count:    group.count,
			LastSeen: group.lastSeen,
			Types:    types,
			Examples: group.examples,
		})
	}
	sort.SliceStable(clusters, func(i, j int) bool {
		if clusters[i].Count != clusters[j].Count {
			return clusters[i].Count > clusters[j].Count
		}
		if clusters[i].LastSeen != clusters[j].LastSeen {
			return clusters[i].LastSeen > clusters[j].LastSeen
		}
		return clusters[i].Query < clusters[j].Query
	})
	return clusters, nil
}

func normalizeQuery(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(query))), " ")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// parseSinceWindow parses a "since" query value such as "7d", "24h", "30m", or
// "1h30m" into a duration. Empty defaults to 7 days. Supports a leading day
// component (Go's time.ParseDuration does not understand "d").
func parseSinceWindow(raw string) (time.Duration, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return 7 * 24 * time.Hour, nil
	}
	var days int
	if idx := strings.IndexByte(value, 'd'); idx >= 0 {
		dayPart := value[:idx]
		parsed, err := parseInt(dayPart)
		if err != nil {
			return 0, fmt.Errorf("invalid since window %q", raw)
		}
		days = parsed
		value = strings.TrimSpace(value[idx+1:])
	}
	total := time.Duration(days) * 24 * time.Hour
	if value != "" {
		rest, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("invalid since window %q", raw)
		}
		total += rest
	}
	if total <= 0 {
		return 0, fmt.Errorf("since window must be positive: %q", raw)
	}
	return total, nil
}

func parseInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty")
	}
	result := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a number")
		}
		result = result*10 + int(r-'0')
	}
	return result, nil
}

// callerFromRequest derives a best-effort caller attribution from the request:
// the structured X-Vrooli-Attribution header when present and decodable, else
// the User-Agent, else "".
func callerFromRequest(r *http.Request) string {
	if raw := strings.TrimSpace(r.Header.Get("X-Vrooli-Attribution")); raw != "" {
		if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
			var info store.AttributionInfo
			if json.Unmarshal(decoded, &info) == nil && info.Kind != "" {
				caller := info.Kind
				if info.MemberID != nil && *info.MemberID != "" {
					caller += "/" + *info.MemberID
				} else if info.SourceSkillID != nil && *info.SourceSkillID != "" {
					caller += "/" + *info.SourceSkillID
				}
				return caller
			}
		}
		return "attribution"
	}
	if ua := strings.TrimSpace(r.UserAgent()); ua != "" {
		return ua
	}
	return ""
}

// callerContextKey carries best-effort caller attribution from the HTTP handler
// into DiscoverTyped so misses record who searched.
type callerContextKey struct{}

// WithCaller attaches a caller attribution string to the context.
func WithCaller(ctx context.Context, caller string) context.Context {
	if strings.TrimSpace(caller) == "" {
		return ctx
	}
	return context.WithValue(ctx, callerContextKey{}, caller)
}

// CallerFrom returns the caller attribution attached by WithCaller, or "".
func CallerFrom(ctx context.Context) string {
	if value, ok := ctx.Value(callerContextKey{}).(string); ok {
		return value
	}
	return ""
}
