package incidents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// RemediationDecisionSource and RemediationDecisionKind compose the decision
// event type this scenario raises:
// vrooli-autoheal.remediation.decision_requested.v1. The resolution webhook and
// its DomainEffect gate on the same pair, so producer and consumer never drift.
const (
	RemediationDecisionSource = "vrooli-autoheal"
	RemediationDecisionKind   = "remediation"
)

// RemediationDecisionEventType is the full producer event type the approval
// request is published under.
var RemediationDecisionEventType = operatordecision.DecisionRequestedEventType(RemediationDecisionSource, RemediationDecisionKind)

// RemediationDecisionSnapshotDigest binds the incident facts, selected candidate,
// and that candidate's generated artifacts to one operator decision.
func RemediationDecisionSnapshotDigest(incident Incident, candidate RemediationCandidate) (string, error) {
	artifacts := make([]RemediationArtifact, 0)
	for _, artifact := range incident.RemediationArtifacts {
		if artifact.RemediationID == candidate.ID {
			artifacts = append(artifacts, artifact)
		}
	}
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].ID != artifacts[j].ID {
			return artifacts[i].ID < artifacts[j].ID
		}
		return artifacts[i].Path < artifacts[j].Path
	})
	snapshot := struct {
		IncidentID  string                `json:"incident_id"`
		Fingerprint string                `json:"fingerprint"`
		Title       string                `json:"title"`
		Severity    Severity              `json:"severity"`
		Candidate   RemediationCandidate  `json:"candidate"`
		Artifacts   []RemediationArtifact `json:"artifacts"`
	}{incident.ID, incident.Fingerprint, incident.Title, incident.Severity, candidate, artifacts}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("marshal remediation decision snapshot: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// RemediationDecisionStorageSnapshot returns the exact JSON column values used
// to revalidate candidate/artifact storage at the authorization write boundary.
func RemediationDecisionStorageSnapshot(incident Incident) (string, string, error) {
	candidates, err := json.Marshal(incident.RemediationCandidates)
	if err != nil {
		return "", "", fmt.Errorf("marshal remediation candidates: %w", err)
	}
	artifacts, err := json.Marshal(incident.RemediationArtifacts)
	if err != nil {
		return "", "", fmt.Errorf("marshal remediation artifacts: %w", err)
	}
	return string(candidates), string(artifacts), nil
}

// BuildRemediationDecisionEvent constructs one immutable request revision. The
// caller persists the returned event before publishing it.
func RemediationDecisionBindingMatches(incident Incident, eventID string, generation int, digest, candidateID string) bool {
	if incident.Status != StatusOpen && incident.Status != StatusAcknowledged {
		return false
	}
	if incident.RemediationDecisionExpiresAt == nil || !time.Now().UTC().Before(*incident.RemediationDecisionExpiresAt) {
		return false
	}
	if eventID == "" || digest == "" || candidateID == "" || generation < 1 {
		return false
	}
	if incident.EventCount != generation || incident.RemediationDecisionGeneration != generation || incident.RemediationDecisionEventID != eventID || incident.RemediationDecisionCandidateID != candidateID || incident.RemediationCandidateDigest != digest {
		return false
	}
	candidate, ok := findRemediationCandidate(incident.RemediationCandidates, candidateID)
	if !ok || candidate.Applicability != "applicable" {
		return false
	}
	currentDigest, err := RemediationDecisionSnapshotDigest(incident, candidate)
	return err == nil && currentDigest == digest
}

func findRemediationCandidate(candidates []RemediationCandidate, id string) (RemediationCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return RemediationCandidate{}, false
}

func BuildRemediationDecisionEvent(incident Incident, candidate RemediationCandidate, revision int, digest string, occurred, deadline time.Time) eventbus.DomainEvent {
	facts := operatordecision.Facts{
		Subject:              fmt.Sprintf("Approve remediation for incident %s", incident.Title),
		Question:             fmt.Sprintf("Autoheal detected %q and proposes remediation %q (risk: %s). Approve applying it?", incident.Title, candidate.Title, remediationRiskLabel(candidate.RiskLevel)),
		Options:              []operatordecision.Option{{Key: "approve", Label: "Approve remediation"}, {Key: "reject", Label: "Do not remediate"}},
		Recommended:          "reject",
		RecommendationReason: "A generated host script can affect the system; review its current candidate and artifact before approving.",
		Reversible:           false,
		Urgency:              "high",
		Note:                 candidate.DecisionPrompt,
		Deadline:             deadline.UTC(),
		Correlation: map[string]string{
			"incident_id":          incident.ID,
			"incident_fingerprint": incident.Fingerprint,
			"candidate_id":         candidate.ID,
			"candidate_risk_level": candidate.RiskLevel,
			"generation":           fmt.Sprint(incident.RemediationDecisionGeneration),
			"revision":             fmt.Sprint(revision),
			"candidate_digest":     digest,
		},
	}
	return operatordecision.BuildEvent(RemediationDecisionSource, RemediationDecisionKind, occurred.UTC(), facts)
}

type Store interface {
	UpsertIncident(ctx context.Context, input UpsertInput) (*Incident, error)
	ListIncidents(ctx context.Context, filters ListFilters) (*ListResponse, error)
	GetIncident(ctx context.Context, id string) (*Incident, error)
	ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]Observation, error)
	UpdateIncidentStatus(ctx context.Context, incidentID string, status Status, note string) (*Incident, error)
	ListPendingOpeningPublications(ctx context.Context, limit int) ([]OpeningPublication, error)
	AcknowledgeOpeningPublication(ctx context.Context, incidentID string, generation int) error
	EnsureRemediationDecisionPublication(ctx context.Context, incidentID string, observedAt time.Time) (*RemediationDecisionPublication, error)
	ListPendingRemediationDecisionPublications(ctx context.Context, limit int) ([]RemediationDecisionPublication, error)
	AcknowledgeRemediationDecisionPublication(ctx context.Context, eventID string) error
}

type EventPublisher interface {
	Publish(context.Context, string, map[string]any) error
}

// DecisionPublisher is the operatordecision.Publisher shape (PublishDomainEvent).
// Decision publications require this identity-preserving contract; the legacy
// facts publisher is reserved for mutable lifecycle notifications.
type publicationDrainDeferredKey struct{}

// DeferPublicationDrain marks a larger owner operation that will drain both
// publication queues once after all of its mutations are complete.
func DeferPublicationDrain(ctx context.Context) context.Context {
	return context.WithValue(ctx, publicationDrainDeferredKey{}, true)
}

func publicationDrainDeferred(ctx context.Context) bool {
	deferred, _ := ctx.Value(publicationDrainDeferredKey{}).(bool)
	return deferred
}

type DecisionPublisher interface {
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

type Service struct {
	store     Store
	now       func() time.Time
	publisher EventPublisher
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) SetEventPublisher(publisher EventPublisher) { s.publisher = publisher }

// UpdateIncidentStatus is the single lifecycle mutation boundary. Manual
// status changes must publish the same transition facts as check-driven
// recovery; otherwise an operator resolving an incident would leave
// downstream notification consumers with a stale open state.
func (s *Service) UpdateIncidentStatus(ctx context.Context, incidentID string, status Status, note string) (*Incident, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("incident service is unavailable")
	}
	current, err := s.store.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	updated, err := s.store.UpdateIncidentStatus(ctx, incidentID, status, note)
	if err != nil {
		return nil, err
	}
	if status == StatusResolved && (current == nil || current.Status != StatusResolved) {
		s.publish(ctx, "incident.resolved.v1", updated)
	}
	return updated, nil
}

func (s *Service) UpsertFromCheckResult(ctx context.Context, result checks.Result) (*Incident, bool, error) {
	incident, created, approval, err := s.upsertFromCheckResult(ctx, result)
	s.finishCheckResultUpsert(ctx, []pendingApprovalRequest{approval})
	return incident, created, err
}

type pendingApprovalRequest struct {
	incident *Incident
}

func (s *Service) finishCheckResultUpsert(ctx context.Context, approvals []pendingApprovalRequest) {
	for _, approval := range approvals {
		if approval.incident == nil {
			continue
		}
		if _, err := s.store.EnsureRemediationDecisionPublication(ctx, approval.incident.ID, approval.incident.LastSeenAt); err != nil {
			log.Printf("vrooli-autoheal remediation decision snapshot failed: incident=%s error=%v", approval.incident.ID, err)
		}
	}
	if !publicationDrainDeferred(ctx) {
		s.DrainPendingPublications(ctx)
	}
}

func (s *Service) upsertFromCheckResult(ctx context.Context, result checks.Result) (*Incident, bool, pendingApprovalRequest, error) {
	if result.Status == checks.StatusOK {
		return nil, false, pendingApprovalRequest{}, s.resolveRecoveredForCheck(ctx, result)
	}
	if result.Status == checks.StatusNotApplicable {
		return nil, false, pendingApprovalRequest{}, nil
	}
	rule, ok := classifyResult(result)
	if !ok {
		return nil, false, pendingApprovalRequest{}, nil
	}
	observedAt := result.Timestamp
	if observedAt.IsZero() {
		observedAt = s.now()
	}
	input := UpsertInput{
		Fingerprint:     rule.fingerprint,
		Type:            rule.incidentType,
		Severity:        rule.severity,
		Title:           rule.title,
		Summary:         result.Message,
		ObservedAt:      observedAt.UTC(),
		BootID:          stringDetail(result.Details, "bootId"),
		SourceCheckID:   result.CheckID,
		Evidence:        boundedEvidence(result.Details),
		Recommendations: stringSliceDetail(result.Details, "recommendations"),
	}
	previousSeverity, hadOpen := s.openIncidentSeverity(ctx, input.Fingerprint)
	enrichInputFromResult(&input, result)
	if err := s.supersedePriorIncidents(ctx, result.CheckID, input.Fingerprint, stringDetail(result.Details, "findingKey")); err != nil {
		return nil, false, pendingApprovalRequest{}, err
	}
	incident, err := s.store.UpsertIncident(ctx, input)
	if err != nil {
		return nil, true, pendingApprovalRequest{}, err
	}
	if incident != nil && hadOpen && previousSeverity != incident.Severity {
		s.publish(ctx, "incident.severity_changed.v1", incident)
	}
	return incident, true, pendingApprovalRequest{incident: incident}, nil
}

func (s *Service) openIncidentSeverity(ctx context.Context, fingerprint string) (Severity, bool) {
	if fingerprint == "" {
		return "", false
	}
	response, err := s.store.ListIncidents(ctx, ListFilters{Status: StatusOpen, Limit: 200})
	if err != nil || response == nil {
		return "", false
	}
	for _, incident := range response.Incidents {
		if incident.Fingerprint == fingerprint {
			return incident.Severity, true
		}
	}
	return "", false
}

// supersedePriorIncidents resolves a check's earlier open incidents when its
// evidence changed fingerprint: one check, one live incident. A result
// narrowed to one finding (findingKey set) is one of several live incidents
// the same check owns at once, so it supersedes nothing; each finding's
// fingerprint is stable and its incident lives and resolves on its own.
func (s *Service) supersedePriorIncidents(ctx context.Context, checkID, fingerprint, findingKey string) error {
	if checkID == "" || fingerprint == "" || findingKey != "" {
		return nil
	}
	resp, err := s.store.ListIncidents(ctx, ListFilters{Status: StatusOpen, Limit: 200})
	if err != nil || resp == nil {
		return err
	}
	for _, incident := range resp.Incidents {
		if incident.Fingerprint == fingerprint || !incidentFromCheck(incident, checkID) {
			continue
		}
		if _, err := s.UpdateIncidentStatus(ctx, incident.ID, StatusResolved,
			fmt.Sprintf("superseded by newer evidence from %s (%s)", checkID, fingerprint)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) UpsertFromCheckResults(ctx context.Context, results []checks.Result) (int, error) {
	count := 0
	var returnErr error
	var approvals []pendingApprovalRequest
	for _, result := range results {
		created, pending, err := s.upsertAllFromCheckResult(ctx, result)
		count += created
		approvals = append(approvals, pending...)
		if err != nil {
			returnErr = err
			break
		}
	}
	s.finishCheckResultUpsert(ctx, approvals)
	return count, returnErr
}

// UpsertAllFromCheckResult opens one incident per finding when a result
// carries a findings list (the emergency watchdog report does), and one
// incident otherwise. Each finding gets its own fingerprint dimension so a
// CPU-pressure finding and a fork-rate finding from the same tick are two
// incidents, not one record whose evidence flips between them.
func (s *Service) UpsertAllFromCheckResult(ctx context.Context, result checks.Result) (int, error) {
	count, approvals, err := s.upsertAllFromCheckResult(ctx, result)
	s.finishCheckResultUpsert(ctx, approvals)
	return count, err
}

func (s *Service) upsertAllFromCheckResult(ctx context.Context, result checks.Result) (int, []pendingApprovalRequest, error) {
	findings := findingList(result.Details)
	if len(findings) == 0 || result.Status == checks.StatusOK {
		_, created, approval, err := s.upsertFromCheckResult(ctx, result)
		if created {
			return 1, []pendingApprovalRequest{approval}, err
		}
		return 0, nil, err
	}
	count := 0
	var approvals []pendingApprovalRequest
	for _, finding := range findings {
		_, created, approval, err := s.upsertFromCheckResult(ctx, withFinding(result, finding))
		if created {
			count++
			if approval.incident != nil {
				approvals = append(approvals, approval)
			}
		}
		if err != nil {
			return count, approvals, err
		}
	}
	return count, approvals, nil
}

// findingList reads Details["findings"] in either shape a check may emit.
func findingList(details map[string]any) []map[string]any {
	if details == nil {
		return nil
	}
	switch raw := details["findings"].(type) {
	case []map[string]any:
		return raw
	case []any:
		out := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

// withFinding narrows a result to one finding: the finding's name becomes the
// findingKey fingerprint dimension, its reason the message, and its
// attribution (when it has one) the evidence the title is built from.
func withFinding(result checks.Result, finding map[string]any) checks.Result {
	narrowed := result
	narrowed.Details = make(map[string]any, len(result.Details)+3)
	for key, value := range result.Details {
		if key == "findings" || key == "attribution" {
			continue
		}
		narrowed.Details[key] = value
	}
	name := stringDetail(finding, "name")
	narrowed.Details["findingKey"] = name
	if reason := stringDetail(finding, "reason"); reason != "" {
		narrowed.Details["findingReason"] = reason
		narrowed.Message = name + ": " + reason
	}
	if attribution, ok := finding["attribution"].(map[string]any); ok {
		narrowed.Details["attribution"] = attribution
	}
	return narrowed
}

// OpenHealIncident records the durable operator-facing escalation once a
// recovery action has crossed the configured consecutive-failure threshold.
// The fingerprint is stable for a check/action pair, so later ticks coalesce
// into one incident while preserving the latest error in its evidence.
func (s *Service) OpenHealIncident(ctx context.Context, checkID, actionID, lastError string, consecutiveFailures int) error {
	observedAt := s.now().UTC()
	_, returnErr := s.store.UpsertIncident(ctx, UpsertInput{
		Fingerprint:   Fingerprint(string(TypeAutohealFailure), "heal", checkID, actionID),
		Type:          TypeAutohealFailure,
		Severity:      SeverityCritical,
		Title:         fmt.Sprintf("Auto-heal escalation: %s", checkID),
		Summary:       fmt.Sprintf("auto-heal action %s for check %s failed repeatedly: %s", actionID, checkID, lastError),
		ObservedAt:    observedAt,
		SourceCheckID: checkID,
		Evidence: map[string]any{
			"checkId":             checkID,
			"actionId":            actionID,
			"lastError":           lastError,
			"consecutiveFailures": consecutiveFailures,
		},
		Recommendations: []string{"Inspect the recorded auto-heal error and restore the check's dependency before retrying."},
	})
	if !publicationDrainDeferred(ctx) {
		s.DrainPendingPublications(ctx)
	}
	return returnErr
}

// ResolveHealIncident closes the matching open escalation after a later
// successful heal. It deliberately only resolves incidents carrying the same
// action evidence, leaving unrelated findings for the check untouched.
func (s *Service) ResolveHealIncident(ctx context.Context, checkID, actionID string) error {
	resp, err := s.store.ListIncidents(ctx, ListFilters{Status: StatusOpen, Type: TypeAutohealFailure, Limit: 200})
	if err != nil || resp == nil {
		return err
	}
	for _, incident := range resp.Incidents {
		if !incidentFromCheck(incident, checkID) || stringDetail(incident.Evidence, "actionId") != actionID {
			continue
		}
		if _, err := s.UpdateIncidentStatus(ctx, incident.ID, StatusResolved,
			fmt.Sprintf("auto-resolved: %s action %s succeeded", checkID, actionID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) resolveRecoveredForCheck(ctx context.Context, result checks.Result) error {
	if result.CheckID == "" {
		return nil
	}
	resp, err := s.store.ListIncidents(ctx, ListFilters{Status: StatusOpen, Limit: 200})
	if err != nil || resp == nil {
		return err
	}
	for _, incident := range resp.Incidents {
		if !incidentFromCheck(incident, result.CheckID) || !autoResolvableIncident(incident) {
			continue
		}
		updated, err := s.UpdateIncidentStatus(ctx, incident.ID, StatusResolved,
			fmt.Sprintf("auto-resolved: %s reported OK", result.CheckID))
		if err != nil {
			return err
		}
		s.publish(ctx, "incident.resolved.v1", updated)
	}
	return nil
}

const pendingOpeningPublicationBatch = 100

// DrainPendingPublications gives one operation a shared maximum of 100
// attempts across opening notices and remediation decisions.
func (s *Service) DrainPendingPublications(ctx context.Context) {
	attempted := s.publishPendingOpeningPublications(ctx, pendingOpeningPublicationBatch)
	s.publishPendingRemediationDecisionPublications(ctx, pendingOpeningPublicationBatch-attempted)
}

func (s *Service) publishPendingOpeningPublications(ctx context.Context, limit int) int {
	if s == nil || s.store == nil || s.publisher == nil || limit <= 0 {
		return 0
	}
	if readiness, ok := s.publisher.(interface{ Enabled() bool }); ok && !readiness.Enabled() {
		return 0
	}
	pending, err := s.store.ListPendingOpeningPublications(ctx, limit)
	if err != nil {
		log.Printf("vrooli-autoheal pending incident-open read failed: %v", err)
		return 0
	}
	attempted := 0
	for _, event := range pending {
		attempted++
		if err := s.publisher.Publish(ctx, event.EventType, event.Payload); err != nil {
			log.Printf("vrooli-autoheal incident event publish failed: type=%s incident=%s generation=%d error=%v", event.EventType, event.IncidentID, event.Generation, err)
			return attempted // Preserve ordering; the next scheduled tick retries this exact snapshot.
		}
		if err := s.store.AcknowledgeOpeningPublication(ctx, event.IncidentID, event.Generation); err != nil {
			log.Printf("vrooli-autoheal incident publication acknowledgement failed: incident=%s generation=%d error=%v", event.IncidentID, event.Generation, err)
			return attempted // A duplicate publish is safe: the event ID is stable for this snapshot.
		}
	}
	return attempted
}

func (s *Service) publishPendingRemediationDecisionPublications(ctx context.Context, limit int) {
	if s == nil || s.store == nil || s.publisher == nil || limit <= 0 {
		return
	}
	if readiness, ok := s.publisher.(interface{ Enabled() bool }); ok && !readiness.Enabled() {
		return
	}
	pending, err := s.store.ListPendingRemediationDecisionPublications(ctx, limit)
	if err != nil {
		log.Printf("vrooli-autoheal pending remediation decision read failed: %v", err)
		return
	}
	for _, publication := range pending {
		var event eventbus.DomainEvent
		if err := json.Unmarshal(publication.EventJSON, &event); err != nil {
			log.Printf("vrooli-autoheal remediation decision decode failed: event=%s error=%v", publication.EventID, err)
			return
		}
		if operatordecision.EventID(event) != publication.EventID {
			log.Printf("vrooli-autoheal remediation decision identity mismatch: event=%s", publication.EventID)
			return
		}
		decisionPublisher, ok := s.publisher.(DecisionPublisher)
		if !ok {
			log.Printf("vrooli-autoheal remediation decision publisher unavailable: incident=%s generation=%d revision=%d", publication.IncidentID, publication.Generation, publication.Revision)
			return
		}
		if publishErr := decisionPublisher.PublishDomainEvent(ctx, event); publishErr != nil {
			log.Printf("vrooli-autoheal remediation decision publish failed: incident=%s generation=%d revision=%d error=%v", publication.IncidentID, publication.Generation, publication.Revision, publishErr)
			return
		}
		if err := s.store.AcknowledgeRemediationDecisionPublication(ctx, publication.EventID); err != nil {
			log.Printf("vrooli-autoheal remediation decision acknowledgement failed: event=%s error=%v", publication.EventID, err)
			return
		}
	}
}

func (s *Service) publish(ctx context.Context, eventType string, incident *Incident) {
	if s == nil || s.publisher == nil || incident == nil {
		return
	}
	// This envelope carries facts only. Notification copy, recipients, and
	// sensitivity are owned by notification-hub.
	if err := s.publisher.Publish(ctx, eventType, incidentEventFacts(incident)); err != nil {
		log.Printf("vrooli-autoheal incident event publish failed: type=%s incident=%s error=%v", eventType, incident.ID, err)
	}
}

// remediationRiskLabel renders an empty/unknown risk level as "unknown" so the
// operator prompt never silently implies a low-risk action.
func remediationRiskLabel(level string) string {
	if strings.TrimSpace(level) == "" {
		return "unknown"
	}
	return level
}

// OpeningEventFacts freezes the event facts for one incident lifecycle. The
// generation discriminator prevents an identical recurrence from deduping
// against a prior resolved lifecycle.
func OpeningEventFacts(incident *Incident) map[string]any {
	facts := incidentEventFacts(incident)
	facts["event_count"] = incident.EventCount
	facts["opened_at"] = incident.LastSeenAt.UTC().Format(time.RFC3339Nano)
	return facts
}

func incidentEventFacts(incident *Incident) map[string]any {
	return map[string]any{
		"incident_id":          incident.ID,
		"incident_fingerprint": incident.Fingerprint,
		"incident_type":        string(incident.Type),
		// Domain aliases are not accepted by protobuf Struct conversion. Emit
		// their wire representation explicitly at this boundary.
		"severity":         string(incident.Severity),
		"status":           string(incident.Status),
		"source_check_id":  firstString(incident.SourceCheckIDs),
		"incident_title":   incident.Title,
		"incident_summary": incident.Summary,
	}
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

type incidentRule struct {
	incidentType Type
	severity     Severity
	title        string
	fingerprint  string
}

func classifyResult(result checks.Result) (incidentRule, bool) {
	if result.Status == checks.StatusOK {
		return incidentRule{}, false
	}
	severity := SeverityWarning
	if result.Status == checks.StatusCritical {
		severity = SeverityCritical
	}
	if strings.HasPrefix(result.CheckID, "host-") {
		return incidentRule{
			incidentType: TypeHostIntegrity,
			severity:     severity,
			title:        hostFindingTitle(result),
			fingerprint:  Fingerprint(string(TypeHostIntegrity), result.CheckID, stableEvidenceDimension(result)),
		}, true
	}
	if key := stringDetail(result.Details, "findingKey"); result.CheckID == "system-emergency-watchdog-report" && key != "" {
		return incidentRule{
			incidentType: TypeHostPressure,
			severity:     severity,
			title:        hostPressureTitle(key, result.Details),
			fingerprint:  Fingerprint(string(TypeHostPressure), result.CheckID, "findingKey="+key),
		}, true
	}
	switch result.CheckID {
	case "infra-rdp":
		// Only credential and session faults become incidents. A daemon that is
		// merely stopped is already covered by the restart action and does not
		// need an operator-facing record.
		state := stringDetail(result.Details, "credentialState")
		if state != "empty" && state != "unreadable" && boolDetail(result.Details, "sessionAvailable") {
			return incidentRule{}, false
		}
		return incidentRule{
			incidentType: TypeHostIntegrity,
			severity:     severity,
			title:        "Remote desktop is not serviceable",
			fingerprint: Fingerprint(string(TypeHostIntegrity), result.CheckID, state,
				stringDetail(result.Details, "credentialModel")),
		}, true
	case "system-boot-history":
		return incidentRule{
			incidentType: TypeUncleanBoot,
			severity:     severity,
			title:        "Unclean boot history detected",
			fingerprint:  Fingerprint(string(TypeUncleanBoot), result.CheckID, stringDetail(result.Details, "latestUncleanBootId")),
		}, true
	case "system-pstore-evidence":
		if boolDetail(result.Details, "coverageGap") {
			return incidentRule{
				incidentType: TypeAutohealFailure,
				severity:     SeverityWarning,
				title:        "Crash artifact coverage gap detected",
				fingerprint:  Fingerprint(string(TypeAutohealFailure), result.CheckID, stringDetail(result.Details, "coverageGapReason")),
			}, true
		}
		if intDetail(result.Details, "pmsgCount") > 0 && intDetail(result.Details, "dmesgCount") == 0 && intDetail(result.Details, "consoleCount") == 0 {
			return incidentRule{incidentType: TypeHostIntegrity, severity: SeverityWarning, title: "Userspace pstore artifacts detected", fingerprint: Fingerprint(string(TypeHostIntegrity), result.CheckID, "pmsg")}, true
		}
		return incidentRule{incidentType: TypeUncleanBoot, severity: severity, title: "Kernel crash artifacts detected", fingerprint: Fingerprint(string(TypeUncleanBoot), result.CheckID)}, true
	case "system-panic-evidence":
		if boolDetail(result.Details, "coverageGap") {
			return incidentRule{
				incidentType: TypeAutohealFailure,
				severity:     SeverityWarning,
				title:        "Kernel panic evidence coverage gap detected",
				fingerprint:  Fingerprint(string(TypeAutohealFailure), result.CheckID, stringDetail(result.Details, "coverageGapReason")),
			}, true
		}
		// Identify the panic by what faulted, not by when. A recurring driver
		// bug is one incident whose evidence keeps updating; a genuinely
		// different panic gets its own record. Fingerprinting on the timestamp
		// instead would turn one defect into an unbounded stream of incidents.
		return incidentRule{
			incidentType: TypeUncleanBoot,
			severity:     severity,
			title:        "Kernel panic captured by kdump",
			fingerprint: Fingerprint(string(TypeUncleanBoot), result.CheckID,
				stringDetail(result.Details, "latestReason"), stringDetail(result.Details, "latestComm")),
		}, true
	case "system-mce-recent":
		if boolDetail(result.Details, "coverageGap") {
			return incidentRule{
				incidentType: TypeAutohealFailure,
				severity:     SeverityWarning,
				title:        "MCE telemetry coverage gap detected",
				fingerprint:  Fingerprint(string(TypeAutohealFailure), result.CheckID, stringDetail(result.Details, "coverageGapReason")),
			}, true
		}
		return incidentRule{incidentType: TypeHostIntegrity, severity: severity, title: "Recent machine-check evidence detected", fingerprint: Fingerprint(string(TypeHostIntegrity), result.CheckID)}, true
	default:
		incidentType := TypeAutohealFailure
		title := "Autoheal check failed: " + result.CheckID
		if strings.HasPrefix(result.CheckID, "scenario-") {
			incidentType = TypeScenarioFailure
			title = "Scenario failure: " + strings.TrimPrefix(result.CheckID, "scenario-")
		} else if strings.HasPrefix(result.CheckID, "resource-") {
			incidentType = TypeResourceFailure
			title = "Resource failure: " + strings.TrimPrefix(result.CheckID, "resource-")
		}
		return incidentRule{
			incidentType: incidentType,
			severity:     severity,
			title:        title,
			fingerprint:  Fingerprint(string(incidentType), result.CheckID, stableEvidenceDimension(result)),
		}, true
	}
}

// hostPressureTitle names the storm's owner from the finding's attribution:
// "fork storm from claude pid 4242 in scope vrooli-agent-abc.scope". A finding
// without attribution says so rather than pretending to know.
func hostPressureTitle(findingKey string, details map[string]any) string {
	subject := map[string]string{"fork-rate": "fork storm", "cpu-pressure": "CPU saturation", "stranded-memory": "stranded memory"}[findingKey]
	if subject == "" {
		subject = "emergency watchdog finding " + findingKey
	}
	attribution, _ := details["attribution"].(map[string]any)
	top, _ := attribution["top_parent"].(map[string]any)
	name := stringDetail(top, "name")
	pid := numberDetail(top, "pid")
	if name == "" || pid == 0 {
		return subject + " (unattributed)"
	}
	title := fmt.Sprintf("%s from %s pid %d", subject, name, pid)
	if scope := stringDetail(top, "scope"); scope != "" {
		title += " in scope " + scope[strings.LastIndex(scope, "/")+1:]
	}
	return title
}

// numberDetail reads an integer that may have crossed JSON as a float.
func numberDetail(details map[string]any, key string) int64 {
	switch v := details[key].(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func hostFindingTitle(result checks.Result) string {
	if message := strings.TrimSpace(result.Message); message != "" {
		return message
	}
	return "Host finding: " + result.CheckID
}

func stableEvidenceDimension(result checks.Result) string {
	for _, key := range []string{
		"evidenceKind", "findingKey", "coverageGapReason", "latestUncleanBootId",
		"statusReason", "packageManager", "serviceStatus", "scenarioStatus", "statusText",
	} {
		if value := stringDetail(result.Details, key); value != "" {
			return key + "=" + value
		}
	}
	if kind := firstEvidenceKindName(result.Details); kind != "" {
		return "evidence=" + kind
	}
	return "message=" + strings.TrimSpace(result.Message)
}

func firstEvidenceKindName(details map[string]any) string {
	items := evidenceItemsForResult(checks.Result{Details: details}, SeverityWarning)
	if len(items) == 0 {
		return ""
	}
	return items[0].Kind
}

func incidentFromCheck(incident Incident, checkID string) bool {
	for _, existing := range incident.SourceCheckIDs {
		if existing == checkID {
			return true
		}
	}
	return false
}

func autoResolvableIncident(incident Incident) bool {
	if incident.Status != StatusOpen {
		return false
	}
	switch incident.Type {
	case TypeHostIntegrity, TypeAutohealFailure:
		return true
	default:
		return false
	}
}

func boundedEvidence(details map[string]any) map[string]any {
	if details == nil {
		return nil
	}
	out := map[string]any{}
	for key, value := range details {
		if key == "recommendations" {
			continue
		}
		out[key] = value
	}
	return out
}

func stringDetail(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	if value, ok := details[key].(string); ok {
		return value
	}
	return ""
}

func stringSliceDetail(details map[string]any, key string) []string {
	if details == nil {
		return nil
	}
	raw, ok := details[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func boolDetail(details map[string]any, key string) bool {
	if details == nil {
		return false
	}
	value, _ := details[key].(bool)
	return value
}

func intDetail(details map[string]any, key string) int {
	if details == nil {
		return 0
	}
	switch value := details[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func enrichInputFromResult(input *UpsertInput, result checks.Result) {
	if input == nil {
		return
	}
	input.Diagnosis = diagnosisForResult(result)
	input.Confidence = confidenceForResult(result)
	input.EvidenceItems = evidenceItemsForResult(result, input.Severity)
	input.CorroborationNeeded = stringSliceDetail(result.Details, "corroborationNeeded")
	input.SafeActions = stringSliceDetail(result.Details, "safeActions")
	input.OperatorActions = stringSliceDetail(result.Details, "operatorActions")
	input.RollbackOrFallback = stringSliceDetail(result.Details, "rollbackOrFallback")
	input.PostChecks = stringSliceDetail(result.Details, "postChecks")
	input.RemediationCandidates = remediationCandidatesForResult(result)
}

func diagnosisForResult(result checks.Result) string {
	switch result.CheckID {
	case "infra-rdp":
		if !boolDetail(result.Details, "sessionAvailable") {
			return "Remote desktop has no graphical session to share"
		}
		return "Remote desktop is running but cannot authenticate any client"
	case "host-runtime-integrity":
		return "Host runtime exists but cannot communicate with its backing driver or daemon"
	case "host-device-driver-binding":
		return "Important host device is present without an active kernel driver binding"
	case "host-kernel-module-drift":
		if hasEvidenceKind(result.Details, "missing_nvidia_module_package") {
			return "NVIDIA driver stack unavailable for the running kernel"
		}
		return "Running kernel and installed module/package inventory do not fully agree"
	case "host-kernel-error-signals":
		if hasEvidenceKind(result.Details, "data_fabric_sync_flood") {
			return "Kernel reported a data fabric sync flood reset event"
		}
		return "Kernel logged recent host or device error signals"
	case "system-boot-history":
		return "Recent boot history indicates unclean shutdown or reset"
	case "system-pstore-evidence":
		if boolDetail(result.Details, "coverageGap") {
			return "Persistent crash artifact coverage is incomplete because no readable pstore source is available"
		}
		if intDetail(result.Details, "pmsgCount") > 0 && intDetail(result.Details, "dmesgCount") == 0 && intDetail(result.Details, "consoleCount") == 0 {
			return "Userspace pstore messages were found without kernel crash dumps"
		}
		return "Persistent kernel crash artifacts were found"
	case "system-mce-recent":
		if boolDetail(result.Details, "coverageGap") {
			return "Machine-check telemetry coverage is incomplete"
		}
		return "Recent machine-check evidence was found"
	default:
		return ""
	}
}

func confidenceForResult(result checks.Result) string {
	if result.Status == checks.StatusCritical {
		return "high"
	}
	if result.Status == checks.StatusWarning {
		return "medium"
	}
	return ""
}

func evidenceItemsForResult(result checks.Result, severity Severity) []EvidenceItem {
	details := result.Details
	raw, ok := details["evidence"].([]map[string]any)
	if !ok {
		if values, ok := details["evidence"].([]any); ok {
			for _, value := range values {
				if item, ok := value.(map[string]any); ok {
					raw = append(raw, item)
				}
			}
		}
	}
	if len(raw) == 0 {
		return nil
	}
	items := make([]EvidenceItem, 0, len(raw))
	for i, value := range raw {
		kind := fmt.Sprintf("%v", value["kind"])
		if kind == "<nil>" || strings.TrimSpace(kind) == "" {
			kind = "check_evidence"
		}
		itemSeverity := severity
		if s, ok := value["severity"].(string); ok && ValidSeverity(s) && s != "" {
			itemSeverity = Severity(s)
		}
		items = append(items, EvidenceItem{
			ID:                    fmt.Sprintf("%s-%d", result.CheckID, i+1),
			Kind:                  kind,
			Severity:              itemSeverity,
			Summary:               evidenceSummary(kind, value),
			Source:                result.CheckID,
			BootID:                stringDetail(details, "bootId"),
			Data:                  value,
			PlatformApplicability: platformApplicability(details),
		})
	}
	return items
}

func evidenceSummary(kind string, data map[string]any) string {
	switch kind {
	case "missing_nvidia_module_package":
		return fmt.Sprintf("Expected NVIDIA module package is missing for running kernel: %v", data["expectedPackage"])
	case "nvidia_kernel_package_state":
		return "NVIDIA package state was collected for the running kernel"
	case "data_fabric_sync_flood":
		return "Kernel reported an uncorrected data fabric sync flood event"
	case "runtime_not_callable":
		return "Runtime command exists but is not callable"
	case "unbound_capability_device":
		return "Capability device has no active kernel driver binding"
	default:
		return strings.ReplaceAll(kind, "_", " ")
	}
}

func platformApplicability(details map[string]any) string {
	if details == nil {
		return ""
	}
	if unsupported, ok := details["unsupportedCapabilities"]; ok && fmt.Sprintf("%v", unsupported) != "[]" {
		return "unsupported"
	}
	return "applicable"
}

func hasEvidenceKind(details map[string]any, kind string) bool {
	for _, item := range evidenceItemsForResult(checks.Result{Details: details}, SeverityWarning) {
		if item.Kind == kind {
			return true
		}
	}
	return false
}

func remediationCandidatesForResult(result checks.Result) []RemediationCandidate {
	if evidence, ok := firstEvidenceKind(result.Details, "runtime_not_callable"); ok {
		service := strings.TrimSpace(fmt.Sprintf("%v", evidence["service"]))
		applicability := "applicable"
		if service == "" || service == "<nil>" {
			applicability = "needs_corroboration"
		}
		return []RemediationCandidate{{
			ID: "operator-runtime-restart", Title: "Review a platform-native runtime restart", Applicability: applicability,
			Platforms: []string{"linux", "macos", "windows"}, RequiresOperator: true, RequiresPrivilege: true,
			RiskLevel: "moderate", TemplateID: "operator-runtime-restart",
			PreflightChecks: []string{"confirm the affected service identity", "confirm the platform-native action", "obtain operator approval"},
			ArtifactPolicy:  "generate_only_under_user_state", PostChecks: []string{"re-run the originating check"},
			DecisionPrompt: "Should the operator review the generated platform-native restart instructions?",
		}}
	}
	if result.CheckID != "host-kernel-module-drift" {
		return nil
	}
	evidence, ok := firstEvidenceKind(result.Details, "missing_nvidia_module_package")
	if !ok {
		return nil
	}
	applicability := "applicable"
	if platformApplicability(result.Details) == "unsupported" {
		applicability = "unsupported"
	} else if strings.TrimSpace(fmt.Sprintf("%v", evidence["expectedPackage"])) == "" ||
		strings.TrimSpace(fmt.Sprintf("%v", evidence["runningKernel"])) == "" {
		applicability = "needs_corroboration"
	} else if !candidateAvailable(evidence["candidate"]) {
		applicability = "blocked"
	}
	return []RemediationCandidate{{
		ID:                "ubuntu-nvidia-kernel-module-mismatch",
		Title:             "Install matching NVIDIA kernel module package for the running kernel",
		Applicability:     applicability,
		Platforms:         []string{"linux", "ubuntu", "debian"},
		RequiresOperator:  true,
		RequiresPrivilege: true,
		RiskLevel:         "moderate",
		TemplateID:        "ubuntu-nvidia-kernel-module-mismatch",
		PreflightChecks: []string{
			"confirm running kernel and expected linux-modules-nvidia package",
			"confirm apt candidate exists",
			"run apt install simulation before mutation",
		},
		Simulation:     "apt-get -s install <expected-package>",
		ArtifactPolicy: "generate_only_under_user_state",
		RollbackOrFallback: []string{
			"boot a kernel with an already-installed matching NVIDIA module package if the install fails",
			"do not remove kernels or driver packages as part of this remediation",
		},
		PostChecks: []string{
			"nvidia-smi",
			"lsmod | grep '^nvidia'",
			"lspci -nnk | grep -A3 -i nvidia",
			"vrooli-autoheal incidents latest --json",
		},
		DecisionPrompt: "Should the operator run the generated NVIDIA kernel-module repair script?",
	}}
}

func firstEvidenceKind(details map[string]any, kind string) (map[string]any, bool) {
	if details == nil {
		return nil, false
	}
	switch values := details["evidence"].(type) {
	case []map[string]any:
		for _, item := range values {
			if fmt.Sprintf("%v", item["kind"]) == kind {
				return item, true
			}
		}
	case []any:
		for _, value := range values {
			item, ok := value.(map[string]any)
			if ok && fmt.Sprintf("%v", item["kind"]) == kind {
				return item, true
			}
		}
	}
	return nil, false
}

func candidateAvailable(candidate any) bool {
	if candidate == nil {
		return false
	}
	if values, ok := candidate.(map[string]any); ok {
		if available, ok := values["available"].(bool); ok {
			return available
		}
		if available, ok := values["Available"].(bool); ok {
			return available
		}
	}
	value := reflect.ValueOf(candidate)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Struct {
		field := value.FieldByName("Available")
		return field.IsValid() && field.Kind() == reflect.Bool && field.Bool()
	}
	return false
}
