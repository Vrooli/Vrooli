// Responsibility: read-only clients for persisted health snapshots, the health
// audit history, and the typed event log.
package main

import (
	"net/url"
	"strconv"

	"github.com/vrooli/cli-core/cliutil"
)

// HealthAuditService talks to /api/v1/health/{models,runners,audit}.
type HealthAuditService struct{ api *cliutil.APIClient }

// AuditQuery mirrors the audit endpoint's filters; empty fields are omitted.
type AuditQuery struct {
	Scope  string
	Runner string
	Model  string
	Status string
	Since  string
	Until  string
	Limit  int
}

func (s *HealthAuditService) GetModels() ([]byte, error) {
	return s.api.Get("/api/v1/health/models", nil)
}

func (s *HealthAuditService) GetRunners() ([]byte, error) {
	return s.api.Get("/api/v1/health/runners", nil)
}

func (s *HealthAuditService) QueryAudit(q AuditQuery) ([]byte, error) {
	values := url.Values{}
	for key, value := range map[string]string{"scope": q.Scope, "runner": q.Runner, "model": q.Model, "status": q.Status, "since": q.Since, "until": q.Until} {
		if value != "" {
			values.Set(key, value)
		}
	}
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	return s.api.Get("/api/v1/health/audit", values)
}

// EventsService talks to /api/v1/events.
type EventsService struct{ api *cliutil.APIClient }

// EventsQuery mirrors the events endpoint's filters; empty fields are omitted.
type EventsQuery struct {
	Run   string
	Type  string
	Since string
	Limit int
}

func (s *EventsService) List(q EventsQuery) ([]byte, error) {
	values := url.Values{}
	for key, value := range map[string]string{"run": q.Run, "type": q.Type, "since": q.Since} {
		if value != "" {
			values.Set(key, value)
		}
	}
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	return s.api.Get("/api/v1/events", values)
}
