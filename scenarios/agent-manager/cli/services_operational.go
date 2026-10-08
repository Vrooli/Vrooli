// Service helpers for the operational stats, health audit, and typed
// events HTTP surface introduced in Phase 3.

package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// OperationalService talks to /api/v1/stats/operational and the
// dedicated /api/v1/stats/fallback endpoint. Responses are surfaced as
// raw JSON so the CLI commands can render their own human-readable
// tables without coupling the wire shape to a dedicated proto.
type OperationalService struct {
	api *cliutil.APIClient
}

// GetOperational fetches the typed-event stats by category. Empty
// category resolves to "summary" on the server side.
func (s *OperationalService) GetOperational(category string) ([]byte, error) {
	q := url.Values{}
	if category != "" {
		q.Set("category", category)
	}
	return s.api.Get("/api/v1/stats/operational", q)
}

// GetFallback fetches the dedicated fallback insights view.
func (s *OperationalService) GetFallback() ([]byte, error) {
	return s.api.Get("/api/v1/stats/fallback", nil)
}

// HealthAuditService talks to /api/v1/health/{models,runners,audit}.
type HealthAuditService struct {
	api *cliutil.APIClient
}

// GetModels returns the flat models snapshot.
func (s *HealthAuditService) GetModels() ([]byte, error) {
	return s.api.Get("/api/v1/health/models", nil)
}

// GetRunners returns the flat runners snapshot.
func (s *HealthAuditService) GetRunners() ([]byte, error) {
	return s.api.Get("/api/v1/health/runners", nil)
}

// AuditQuery filters audit history. All fields optional. Scope must be
// "model" (default) or "runner".
type AuditQuery struct {
	Scope  string
	Runner string
	Model  string
	Status string
	Since  string
	Until  string
	Limit  int
}

// QueryAudit pages through the audit history.
func (s *HealthAuditService) QueryAudit(q AuditQuery) ([]byte, error) {
	values := url.Values{}
	if q.Scope != "" {
		values.Set("scope", q.Scope)
	}
	if q.Runner != "" {
		values.Set("runner", q.Runner)
	}
	if q.Model != "" {
		values.Set("model", q.Model)
	}
	if q.Status != "" {
		values.Set("status", q.Status)
	}
	if q.Since != "" {
		values.Set("since", q.Since)
	}
	if q.Until != "" {
		values.Set("until", q.Until)
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	return s.api.Get("/api/v1/health/audit", values)
}

// EventsService talks to /api/v1/events.
type EventsService struct {
	api *cliutil.APIClient
}

// EventsQuery filters typed-event listings.
type EventsQuery struct {
	Run   string
	Type  string
	Since string
	Limit int
}

// List returns typed events according to the query.
func (s *EventsService) List(q EventsQuery) ([]byte, error) {
	values := url.Values{}
	if q.Run != "" {
		values.Set("run", q.Run)
	}
	if q.Type != "" {
		values.Set("type", q.Type)
	}
	if q.Since != "" {
		values.Set("since", q.Since)
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	return s.api.Get("/api/v1/events", values)
}

// prettyPrintJSON re-marshals raw JSON bytes with two-space indent for
// human-readable CLI output. Falls back to the input bytes on parse
// failure so the user always gets something useful.
func prettyPrintJSON(body []byte) []byte {
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return body
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return body
	}
	return out
}
