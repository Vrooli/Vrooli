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

func TestHealIncidentLifecyclePreservesErrorAndResolvesMatchingAction(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	const lastError = "Runtime error: start qdrant: managed-service health check did not pass before startup timeout"
	if err := service.OpenHealIncident(context.Background(), "resource-qdrant", "start", lastError, 3); err != nil {
		t.Fatalf("OpenHealIncident() error = %v", err)
	}
	if len(store.inputs) != 1 {
		t.Fatalf("upsert count = %d, want one", len(store.inputs))
	}
	input := store.inputs[0]
	if input.SourceCheckID != "resource-qdrant" || input.Evidence["actionId"] != "start" || input.Evidence["lastError"] != lastError {
		t.Fatalf("incident input lost identity/error: %+v", input)
	}

	store.list = []Incident{{
		ID: "inc_qdrant", Type: TypeAutohealFailure, Status: StatusOpen,
		SourceCheckIDs: []string{"resource-qdrant"},
		Evidence:       map[string]any{"actionId": "start"},
	}}
	if err := service.ResolveHealIncident(context.Background(), "resource-qdrant", "start"); err != nil {
		t.Fatalf("ResolveHealIncident() error = %v", err)
	}
	if len(store.updateIDs) != 1 || store.updateIDs[0] != "inc_qdrant" || store.updates[0] != StatusResolved {
		t.Fatalf("updates = %#v/%#v, want inc_qdrant/resolved", store.updateIDs, store.updates)
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

// A captured panic must become a critical unclean-boot incident identified by
// what faulted, so one recurring defect stays one incident.
func TestPanicEvidenceBecomesAnIdentifiedIncident(t *testing.T) {
	base := checks.Result{
		CheckID: "system-panic-evidence",
		Status:  checks.StatusCritical,
		Message: "Kernel panic captured by kdump: kernel BUG at fs/iomap/buffered-io.c:1061!",
		Details: map[string]interface{}{
			"latestStamp":  "202608191459",
			"latestReason": "kernel BUG at fs/iomap/buffered-io.c:1061!",
			"latestComm":   "kopia",
		},
	}

	rule, ok := classifyResult(base)
	if !ok {
		t.Fatal("a captured panic must classify as an incident")
	}
	if rule.incidentType != TypeUncleanBoot {
		t.Errorf("type = %v, want unclean_boot", rule.incidentType)
	}
	if rule.severity != SeverityCritical {
		t.Errorf("severity = %v, want critical", rule.severity)
	}

	// The same defect crashing again is the same incident, even at a new stamp.
	repeat := base
	repeat.Details = map[string]interface{}{
		"latestStamp":  "202608200800",
		"latestReason": "kernel BUG at fs/iomap/buffered-io.c:1061!",
		"latestComm":   "kopia",
	}
	repeatRule, _ := classifyResult(repeat)
	if repeatRule.fingerprint != rule.fingerprint {
		t.Error("the same panic at a later time must dedupe into one incident")
	}

	// A different panic is a different incident.
	other := base
	other.Details = map[string]interface{}{
		"latestStamp":  "202608200900",
		"latestReason": "Oops: general protection fault",
		"latestComm":   "node",
	}
	otherRule, _ := classifyResult(other)
	if otherRule.fingerprint == rule.fingerprint {
		t.Error("a different panic must not collapse into the existing incident")
	}
}

// A coverage gap is a warning about observability, not a claim that the host
// crashed.
func TestPanicEvidenceCoverageGapIsAWarning(t *testing.T) {
	rule, ok := classifyResult(checks.Result{
		CheckID: "system-panic-evidence",
		Status:  checks.StatusWarning,
		Details: map[string]interface{}{
			"coverageGap":       true,
			"coverageGapReason": "crashdump_export_missing",
		},
	})
	if !ok {
		t.Fatal("a coverage gap must classify as an incident")
	}
	if rule.severity != SeverityWarning {
		t.Errorf("severity = %v, want warning", rule.severity)
	}
	if rule.incidentType != TypeAutohealFailure {
		t.Errorf("type = %v, want autoheal_failure", rule.incidentType)
	}
}
