package incidents

import (
	"context"
	"testing"
	"time"
	"vrooli-autoheal/internal/checks"
)

type memoryStore struct {
	inputs []UpsertInput
}

func (m *memoryStore) UpsertIncident(ctx context.Context, input UpsertInput) (*Incident, error) {
	m.inputs = append(m.inputs, input)
	return &Incident{ID: "inc_test", Fingerprint: input.Fingerprint, Type: input.Type, Severity: input.Severity, Status: StatusOpen}, nil
}

func (m *memoryStore) ListIncidents(ctx context.Context, filters ListFilters) (*ListResponse, error) {
	return nil, nil
}

func (m *memoryStore) GetIncident(ctx context.Context, id string) (*Incident, error) {
	return nil, nil
}

func (m *memoryStore) ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]Observation, error) {
	return nil, nil
}

func (m *memoryStore) UpdateIncidentStatus(ctx context.Context, incidentID string, status Status, note string) (*Incident, error) {
	return nil, nil
}

func TestUpsertFromCheckResultCreatesHostIntegrityIncident(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	incident, created, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID:   "host-runtime-integrity",
		Status:    checks.StatusCritical,
		Message:   "runtime failed",
		Timestamp: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		Details: map[string]any{
			"recommendations": []string{"inspect runtime"},
		},
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if !created {
		t.Fatal("expected incident to be created")
	}
	if incident.Type != TypeHostIntegrity {
		t.Fatalf("incident type = %s, want host_integrity", incident.Type)
	}
	if len(store.inputs) != 1 {
		t.Fatalf("upsert count = %d, want 1", len(store.inputs))
	}
	if store.inputs[0].Severity != SeverityCritical {
		t.Fatalf("severity = %s, want critical", store.inputs[0].Severity)
	}
}

func TestUpsertFromCheckResultIgnoresOKResult(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	_, created, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "host-runtime-integrity",
		Status:  checks.StatusOK,
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if created {
		t.Fatal("did not expect incident for OK result")
	}
}

func TestUpsertFromCheckResultUsesLatestUncleanBootForFingerprint(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	_, created, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "system-boot-history",
		Status:  checks.StatusCritical,
		Message: "unclean boot",
		Details: map[string]any{
			"latestUncleanBootId": "boot-a",
		},
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if !created {
		t.Fatal("expected incident to be created")
	}
	want := Fingerprint(string(TypeUncleanBoot), "system-boot-history", "boot-a")
	if store.inputs[0].Fingerprint != want {
		t.Fatalf("fingerprint = %s, want %s", store.inputs[0].Fingerprint, want)
	}
}
