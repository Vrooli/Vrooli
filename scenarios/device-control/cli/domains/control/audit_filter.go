package control

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// auditFilter narrows the audit record list to what a caller asked about, so
// reading "what did my last command record" is one bounded call instead of
// re-fetching every record through a different jq filter each time.
type auditFilter struct {
	device string
	actor  string
	since  time.Time
	limit  int
}

func parseAuditFilter(device, actor, since, limit string, now time.Time) (auditFilter, error) {
	filter := auditFilter{device: strings.TrimSpace(device), actor: strings.TrimSpace(actor)}
	if since = strings.TrimSpace(since); since != "" {
		if ago, err := time.ParseDuration(since); err == nil {
			if ago < 0 {
				return auditFilter{}, fmt.Errorf("--since duration must not be negative")
			}
			filter.since = now.Add(-ago)
		} else if at, err := time.Parse(time.RFC3339Nano, since); err == nil {
			filter.since = at
		} else {
			return auditFilter{}, fmt.Errorf("--since must be an RFC 3339 time or a duration such as 15m, got %q", since)
		}
	}
	if limit = strings.TrimSpace(limit); limit != "" {
		n, err := strconv.Atoi(limit)
		if err != nil || n <= 0 {
			return auditFilter{}, fmt.Errorf("--limit must be a positive integer, got %q", limit)
		}
		filter.limit = n
	}
	return filter, nil
}

func (f auditFilter) active() bool {
	return f.device != "" || f.actor != "" || !f.since.IsZero() || f.limit > 0
}

// apply filters the API's {"records": [...]} body and returns the same shape,
// newest record first.
func (f auditFilter) apply(body []byte) ([]byte, error) {
	var payload struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode audit records: %w", err)
	}
	kept := make([]map[string]any, 0, len(payload.Records))
	for _, record := range payload.Records {
		if f.device != "" && auditString(record, "device_id") != f.device {
			continue
		}
		if f.actor != "" && auditString(record, "actor") != f.actor {
			continue
		}
		if !f.since.IsZero() {
			created, ok := auditTime(record)
			if !ok || created.Before(f.since) {
				continue
			}
		}
		kept = append(kept, record)
	}
	sort.SliceStable(kept, func(i, j int) bool {
		a, _ := auditTime(kept[i])
		b, _ := auditTime(kept[j])
		return a.After(b)
	})
	if f.limit > 0 && len(kept) > f.limit {
		kept = kept[:f.limit]
	}
	return json.Marshal(map[string]any{"records": kept})
}

func auditString(record map[string]any, key string) string {
	value, _ := record[key].(string)
	return value
}

// auditTime is the record's creation time; older records carry only a
// zero created_at and a timestamp.
func auditTime(record map[string]any) (time.Time, bool) {
	for _, key := range []string{"created_at", "timestamp"} {
		if raw := auditString(record, key); raw != "" {
			if at, err := time.Parse(time.RFC3339Nano, raw); err == nil && !at.IsZero() && at.Year() > 1 {
				return at, true
			}
		}
	}
	return time.Time{}, false
}
