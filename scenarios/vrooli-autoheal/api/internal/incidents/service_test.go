package incidents

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

type memoryStore struct {
	inputs    []UpsertInput
	list      []Incident
	updates   []Status
	updateIDs []string
	pending   []OpeningPublication
}

type recordingPublisher struct {
	eventType string
	payload   map[string]any
	events    []string
	count     int
	err       error
}

func (p *recordingPublisher) Publish(_ context.Context, eventType string, payload map[string]any) error {
	p.eventType = eventType
	p.payload = payload
	p.events = append(p.events, eventType)
	p.count++
	return p.err
}

func (m *memoryStore) UpsertIncident(ctx context.Context, input UpsertInput) (*Incident, error) {
	m.inputs = append(m.inputs, input)
	eventCount := 1
	newLifecycle := true
	for _, previous := range m.list {
		if previous.Fingerprint != input.Fingerprint {
			continue
		}
		eventCount = previous.EventCount
		newLifecycle = previous.Status == StatusResolved
		if newLifecycle {
			eventCount++
		}
		break
	}
	incident := &Incident{
		ID: "inc_test_" + input.Fingerprint, Fingerprint: input.Fingerprint, Type: input.Type, Severity: input.Severity,
		Status: StatusOpen, Title: input.Title, Summary: input.Summary,
		EventCount: eventCount, SourceCheckIDs: []string{input.SourceCheckID},
		RemediationCandidates: input.RemediationCandidates,
	}
	if newLifecycle {
		m.pending = append(m.pending, OpeningPublication{IncidentID: incident.ID, Generation: incident.EventCount, EventType: "incident.opened.v1", Payload: OpeningEventFacts(incident)})
	}
	return incident, nil
}

func TestDisabledEventBusLeavesOpeningPublicationPending(t *testing.T) {
	store := &memoryStore{pending: []OpeningPublication{{IncidentID: "inc-test", Generation: 1, EventType: "incident.opened.v1", Payload: map[string]any{"incident_id": "inc-test"}}}}
	service := NewService(store)
	service.SetEventPublisher(EventBusPublisher{})

	service.publishPendingOpeningPublications(context.Background())
	if len(store.pending) != 1 {
		t.Fatalf("disabled event bus cleared pending publication; want it retained")
	}
}

func TestUpsertAllFromCheckResultDrainsAtMostOnePublicationBatch(t *testing.T) {
	store := &memoryStore{}
	publisher := &recordingPublisher{}
	service := NewService(store)
	service.SetEventPublisher(publisher)
	findings := make([]any, pendingOpeningPublicationBatch+5)
	for i := range findings {
		findings[i] = map[string]any{"name": fmt.Sprintf("finding-%03d", i), "reason": "active"}
	}
	result := checks.Result{
		CheckID: "system-emergency-watchdog-report", Status: checks.StatusCritical,
		Details: map[string]any{"findings": findings},
	}

	count, err := service.UpsertAllFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("UpsertAllFromCheckResult() error = %v", err)
	}
	if count != len(findings) {
		t.Fatalf("created incidents = %d, want %d", count, len(findings))
	}
	if publisher.count != pendingOpeningPublicationBatch || len(store.pending) != 5 {
		t.Fatalf("one bounded drain published/left pending %d/%d; want %d/5", publisher.count, len(store.pending), pendingOpeningPublicationBatch)
	}
}

func TestUpsertFromCheckResultsDrainsOneBatchOnTickPath(t *testing.T) {
	store := &memoryStore{}
	publisher := &recordingPublisher{}
	service := NewService(store)
	service.SetEventPublisher(publisher)
	findings := make([]any, pendingOpeningPublicationBatch+5)
	for i := range findings {
		findings[i] = map[string]any{"name": fmt.Sprintf("tick-finding-%03d", i), "reason": "active"}
	}
	result := checks.Result{
		CheckID: "system-emergency-watchdog-report", Status: checks.StatusCritical,
		Details: map[string]any{"findings": findings},
	}

	count, err := service.UpsertFromCheckResults(context.Background(), []checks.Result{result})
	if err != nil {
		t.Fatalf("UpsertFromCheckResults() error = %v", err)
	}
	if count != len(findings) {
		t.Fatalf("created incidents = %d, want %d", count, len(findings))
	}
	if publisher.count != pendingOpeningPublicationBatch || len(store.pending) != 5 {
		t.Fatalf("one Tick-path drain published/left pending %d/%d; want %d/5", publisher.count, len(store.pending), pendingOpeningPublicationBatch)
	}
}

func TestCriticalApprovalFollowsOpeningPublicationAttempt(t *testing.T) {
	store := &memoryStore{}
	publisher := &recordingPublisher{}
	service := NewService(store)
	service.SetEventPublisher(publisher)

	_, _, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "host-runtime-integrity", Status: checks.StatusCritical, Message: "runtime failed",
		Details: map[string]any{"evidence": []map[string]any{{"kind": "runtime_not_callable", "service": "runtime"}}},
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if len(publisher.events) < 2 || publisher.events[0] != "incident.opened.v1" || publisher.events[1] != RemediationDecisionEventType {
		t.Fatalf("publication order = %v; want opening attempt before remediation decision", publisher.events)
	}
}

func TestUpsertFromCheckResultPublishesWireSafeIncidentFacts(t *testing.T) {
	store := &memoryStore{}
	publisher := &recordingPublisher{}
	service := NewService(store)
	service.SetEventPublisher(publisher)

	_, _, err := service.UpsertFromCheckResult(context.Background(), checks.Result{
		CheckID: "host-runtime-integrity",
		Status:  checks.StatusCritical,
		Message: "runtime failed",
	})
	if err != nil {
		t.Fatalf("UpsertFromCheckResult() error = %v", err)
	}
	if publisher.eventType != "incident.opened.v1" {
		t.Fatalf("event type = %q", publisher.eventType)
	}
	if _, ok := publisher.payload["severity"].(string); !ok {
		t.Fatalf("severity payload type = %T, want string", publisher.payload["severity"])
	}
	if _, ok := publisher.payload["status"].(string); !ok {
		t.Fatalf("status payload type = %T, want string", publisher.payload["status"])
	}
}

func TestUpsertFromCheckResultRetriesFailedPublicationForOpenIncident(t *testing.T) {
	store := &memoryStore{}
	publisher := &recordingPublisher{err: errors.New("temporary publish failure")}
	service := NewService(store)
	service.SetEventPublisher(publisher)
	result := checks.Result{CheckID: "host-runtime-integrity", Status: checks.StatusCritical, Message: "runtime failed"}

	first, _, err := service.UpsertFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("first upsert error = %v", err)
	}
	if first == nil || first.EventCount != 1 {
		t.Fatalf("first SQLite-faithful event count = %+v, want open event count 1", first)
	}
	if publisher.count != 1 || publisher.eventType != "incident.opened.v1" {
		t.Fatalf("first publication = %d/%q, want one incident.opened.v1 attempt", publisher.count, publisher.eventType)
	}
	firstType := publisher.eventType
	firstFacts := publisher.payload
	store.list = []Incident{{ID: first.ID, Fingerprint: first.Fingerprint, Severity: first.Severity, Status: StatusOpen, EventCount: first.EventCount, SourceCheckIDs: []string{result.CheckID}}}
	publisher.err = nil

	second, _, err := service.UpsertFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("second upsert error = %v", err)
	}
	if second == nil || second.EventCount != 1 {
		t.Fatalf("repeated open SQLite-faithful event count = %+v, want 1", second)
	}
	if publisher.count != 2 || publisher.eventType != firstType {
		t.Fatalf("publication attempts/type = %d/%q, want 2/%q", publisher.count, publisher.eventType, firstType)
	}
	if !reflect.DeepEqual(firstFacts, publisher.payload) {
		t.Fatalf("retry facts changed:\nfirst: %+v\nretry: %+v", firstFacts, publisher.payload)
	}
}

func TestUpsertFromCheckResultPublishesResolvedRecurrenceAsNewLifecycle(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	result := checks.Result{CheckID: "host-runtime-integrity", Status: checks.StatusCritical, Message: "runtime failed"}
	firstPublisher := &recordingPublisher{}
	service.SetEventPublisher(firstPublisher)

	first, _, err := service.UpsertFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("initial upsert error = %v", err)
	}
	if first == nil || first.EventCount != 1 || firstPublisher.count != 1 {
		t.Fatalf("initial lifecycle = incident %+v, publish count %d; want generation 1 and one opening event", first, firstPublisher.count)
	}
	store.list = []Incident{{ID: first.ID, Fingerprint: first.Fingerprint, Type: first.Type, Severity: first.Severity, Status: StatusResolved, EventCount: first.EventCount, SourceCheckIDs: []string{result.CheckID}}}
	publisher := &recordingPublisher{}
	service.SetEventPublisher(publisher)

	reopened, _, err := service.UpsertFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("resolved recurrence upsert error = %v", err)
	}
	if reopened == nil || reopened.Status != StatusOpen || reopened.EventCount != 2 {
		t.Fatalf("reopened incident = %+v, want open generation 2", reopened)
	}
	if publisher.count != 1 || publisher.eventType != "incident.opened.v1" {
		t.Fatalf("recurrence publications = %d/%q, want one incident.opened.v1", publisher.count, publisher.eventType)
	}
	if got := publisher.payload["event_count"]; got != 2 {
		t.Fatalf("event generation = %#v, want 2 to distinguish this reopen from generation 1", got)
	}

	store.list = []Incident{{ID: reopened.ID, Fingerprint: reopened.Fingerprint, Type: reopened.Type, Severity: reopened.Severity, Status: StatusOpen, EventCount: reopened.EventCount, SourceCheckIDs: []string{result.CheckID}}}
	if _, _, err := service.UpsertFromCheckResult(context.Background(), result); err != nil {
		t.Fatalf("later same-generation upsert error = %v", err)
	}
	if publisher.count != 1 {
		t.Fatalf("same-generation open update published %d times, want no duplicate opening event", publisher.count)
	}
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

func (m *memoryStore) ListPendingOpeningPublications(_ context.Context, limit int) ([]OpeningPublication, error) {
	if limit > len(m.pending) {
		limit = len(m.pending)
	}
	return append([]OpeningPublication(nil), m.pending[:limit]...), nil
}

func (m *memoryStore) AcknowledgeOpeningPublication(_ context.Context, incidentID string, generation int) error {
	for i, event := range m.pending {
		if event.IncidentID == incidentID && event.Generation == generation {
			m.pending = append(m.pending[:i], m.pending[i+1:]...)
			return nil
		}
	}
	return nil
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

func TestCriticalIncidentPublishesRemediationDecisionOnlyForFirstLifecycle(t *testing.T) {
	store := &memoryStore{}
	publisher := &decisionRecordingPublisher{}
	service := NewService(store)
	service.SetEventPublisher(publisher)
	result := checks.Result{
		CheckID: "host-runtime-integrity", Status: checks.StatusCritical, Message: "runtime failed",
		Details: map[string]any{"evidence": []map[string]any{{"kind": "runtime_not_callable", "service": "runtime"}}},
	}

	first, _, err := service.UpsertFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("first critical upsert error = %v", err)
	}
	if first.EventCount != 1 || publisher.count != 1 {
		t.Fatalf("first lifecycle event count/prompt count = %d/%d, want 1/1", first.EventCount, publisher.count)
	}
	store.list = []Incident{{ID: first.ID, Fingerprint: first.Fingerprint, Type: first.Type, Severity: first.Severity, Status: StatusResolved, EventCount: first.EventCount, SourceCheckIDs: []string{result.CheckID}}}

	second, _, err := service.UpsertFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("recurrent critical upsert error = %v", err)
	}
	if second.EventCount != 2 || publisher.count != 1 {
		t.Fatalf("recurrence event count/prompt count = %d/%d, want 2/1", second.EventCount, publisher.count)
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

// decisionRecordingPublisher captures the full DomainEvent so the test can
// assert the event type and the correlation keys the resolution must hand back.
type decisionRecordingPublisher struct {
	event eventbus.DomainEvent
	count int
}

func (p *decisionRecordingPublisher) Publish(context.Context, string, map[string]any) error {
	return nil
}

func (p *decisionRecordingPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.event = event
	p.count++
	return nil
}

// A critical incident with an applicable candidate raises the remediation
// approval on the operator-decision substrate: the composed event type and the
// four correlation keys the webhook re-verifies against.
func TestCriticalIncidentRaisesRemediationDecisionOnTheSubstrate(t *testing.T) {
	store := &memoryStore{}
	publisher := &decisionRecordingPublisher{}
	service := NewService(store)
	service.SetEventPublisher(publisher)

	incident := &Incident{
		ID:          "inc-42",
		Fingerprint: "fp-42",
		Title:       "NVIDIA module missing",
		Severity:    SeverityCritical,
		Status:      StatusOpen,
	}
	candidate := RemediationCandidate{ID: "candidate-x", Title: "Install module", RiskLevel: "moderate"}
	service.publishApprovalRequest(context.Background(), incident, candidate)

	if publisher.count != 1 {
		t.Fatalf("expected one decision event, got %d", publisher.count)
	}
	if publisher.event.EventType != "vrooli-autoheal.remediation.decision_requested.v1" {
		t.Fatalf("event type = %q", publisher.event.EventType)
	}
	if publisher.event.Source != "vrooli-autoheal" {
		t.Fatalf("event source = %q", publisher.event.Source)
	}
	correlation, _ := publisher.event.Payload["correlation"].(map[string]any)
	if correlation == nil {
		t.Fatalf("payload carries no correlation: %#v", publisher.event.Payload)
	}
	for key, want := range map[string]string{
		"incident_id":          "inc-42",
		"incident_fingerprint": "fp-42",
		"candidate_id":         "candidate-x",
		"candidate_risk_level": "moderate",
	} {
		if got, _ := correlation[key].(string); got != want {
			t.Fatalf("correlation[%q] = %q, want %q", key, got, want)
		}
	}
}
