package incidents

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

type memoryStore struct {
	inputs    []UpsertInput
	list      []Incident
	updates   []Status
	updateIDs []string
}

func (m *memoryStore) UpsertIncident(ctx context.Context, input UpsertInput) (*Incident, error) {
	m.inputs = append(m.inputs, input)
	return &Incident{ID: "inc_test", Fingerprint: input.Fingerprint, Type: input.Type, Severity: input.Severity, Status: StatusOpen}, nil
}

func (m *memoryStore) ListIncidents(ctx context.Context, filters ListFilters) (*ListResponse, error) {
	var out []Incident
	for _, incident := range m.list {
		if filters.Status != "" && incident.Status != filters.Status {
			continue
		}
		out = append(out, incident)
	}
	return &ListResponse{Incidents: out, Total: len(out), Filters: filters}, nil
}

func (m *memoryStore) GetIncident(ctx context.Context, id string) (*Incident, error) {
	return nil, nil
}

func (m *memoryStore) ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]Observation, error) {
	return nil, nil
}

func (m *memoryStore) UpdateIncidentStatus(ctx context.Context, incidentID string, status Status, note string) (*Incident, error) {
	m.updateIDs = append(m.updateIDs, incidentID)
	m.updates = append(m.updates, status)
	return &Incident{ID: incidentID, Status: status}, nil
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

func TestUpsertFromCheckResultClassifiesPstoreCoverageGapSeparately(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	_, created, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "system-pstore-evidence",
		Status:  checks.StatusWarning,
		Message: "coverage gap",
		Details: map[string]any{
			"coverageGap":       true,
			"coverageGapReason": "direct_pstore_permission_denied_export_missing",
		},
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if !created {
		t.Fatal("expected incident to be created")
	}
	input := store.inputs[0]
	if input.Type != TypeAutohealFailure {
		t.Fatalf("type = %s, want autoheal_failure", input.Type)
	}
	if input.Title != "Crash artifact coverage gap detected" {
		t.Fatalf("title = %q", input.Title)
	}
	if input.Diagnosis == "Persistent kernel crash evidence was found or could not be inspected" {
		t.Fatalf("diagnosis still conflates evidence and coverage: %q", input.Diagnosis)
	}
}

func TestUpsertFromCheckResultAutoResolvesRecoveredHostIntegrityIncident(t *testing.T) {
	store := &memoryStore{list: []Incident{{
		ID:             "inc_1",
		Type:           TypeHostIntegrity,
		Status:         StatusOpen,
		SourceCheckIDs: []string{"host-kernel-module-drift"},
	}}}
	service := NewService(store)

	_, created, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "host-kernel-module-drift",
		Status:  checks.StatusOK,
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if created {
		t.Fatal("OK result should not create an incident")
	}
	if len(store.updates) != 1 || store.updates[0] != StatusResolved {
		t.Fatalf("updates = %#v, want one resolved update", store.updates)
	}
}

func TestUpsertFromCheckResultDoesNotAutoResolveCrashArtifactIncident(t *testing.T) {
	store := &memoryStore{list: []Incident{{
		ID:             "inc_1",
		Type:           TypeUncleanBoot,
		Status:         StatusOpen,
		SourceCheckIDs: []string{"system-pstore-evidence"},
	}}}
	service := NewService(store)

	_, _, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "system-pstore-evidence",
		Status:  checks.StatusOK,
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if len(store.updates) != 0 {
		t.Fatalf("updates = %#v, want no auto-resolve for crash artifact incident", store.updates)
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

func TestUpsertFromCheckResultAddsTypedRemediationCandidate(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	_, created, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "host-kernel-module-drift",
		Status:  checks.StatusCritical,
		Message: "missing module package",
		Details: map[string]any{
			"bootId": "boot-a",
			"evidence": []map[string]any{{
				"kind":            "missing_nvidia_module_package",
				"expectedPackage": "linux-modules-nvidia-580-open-6.17.0-23-generic",
				"runningKernel":   "6.17.0-23-generic",
				"candidate":       map[string]any{"available": true},
				"severity":        "critical",
			}},
		},
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if !created {
		t.Fatal("expected incident to be created")
	}
	input := store.inputs[0]
	if input.Diagnosis != "NVIDIA driver stack unavailable for the running kernel" {
		t.Fatalf("diagnosis = %q", input.Diagnosis)
	}
	if len(input.EvidenceItems) != 1 || input.EvidenceItems[0].Kind != "missing_nvidia_module_package" {
		t.Fatalf("evidence items = %#v", input.EvidenceItems)
	}
	if len(input.RemediationCandidates) != 1 {
		t.Fatalf("remediation candidates = %#v, want one", input.RemediationCandidates)
	}
	if input.RemediationCandidates[0].ID != "ubuntu-nvidia-kernel-module-mismatch" {
		t.Fatalf("remediation candidate id = %q", input.RemediationCandidates[0].ID)
	}
	if input.RemediationCandidates[0].Applicability != "applicable" {
		t.Fatalf("remediation candidate applicability = %q", input.RemediationCandidates[0].Applicability)
	}
}

func TestUpsertFromCheckResultBlocksNVIDIARemediationWithoutPackageCandidate(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	_, created, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "host-kernel-module-drift",
		Status:  checks.StatusWarning,
		Message: "missing module package without candidate",
		Details: map[string]any{
			"evidence": []map[string]any{{
				"kind":            "missing_nvidia_module_package",
				"expectedPackage": "linux-modules-nvidia-580-open-6.17.0-23-generic",
				"runningKernel":   "6.17.0-23-generic",
				"candidate":       map[string]any{"available": false},
			}},
		},
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if !created {
		t.Fatal("expected incident to be created")
	}
	input := store.inputs[0]
	if len(input.RemediationCandidates) != 1 {
		t.Fatalf("remediation candidates = %#v, want one blocked candidate", input.RemediationCandidates)
	}
	if input.RemediationCandidates[0].Applicability != "blocked" {
		t.Fatalf("remediation candidate applicability = %q, want blocked", input.RemediationCandidates[0].Applicability)
	}
}
