// Package investigation contains the caller-neutral contract for finite
// investigations. It deliberately has no orchestration or Plan Manager
// dependency: callers supply a subject and question, while Agent Manager owns
// validation, evidence identity, and the diagnostic result envelope.
package investigation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	RequestSchemaVersion = "investigation-request/v1"
	ResultSchemaVersion  = "investigation-result/v1"
	// WorkflowKey is the diagnosis-only workflow used by the typed lifecycle.
	// The legacy agent-manager/investigate workflow remains approval-gated and
	// is intentionally not reused for this read-only contract.
	WorkflowKey = "agent-manager/investigate-typed"

	OperationQueued     = "queued"
	OperationCollecting = "collecting"
	OperationDiagnosing = "diagnosing"
	OperationCompleted  = "completed"
	OperationFailed     = "failed"
	OperationCancelled  = "cancelled"

	ConditionProgressing = "progressing"
	ConditionWaiting     = "waiting"
	ConditionStalled     = "stalled"
	ConditionBlocked     = "blocked"
	ConditionFailed      = "failed"
	ConditionComplete    = "complete"
	ConditionUnknown     = "unknown"

	DispositionNoIntervention  = "no_intervention"
	DispositionRecommendAction = "recommend_action"
	DispositionInconclusive    = "inconclusive"

	RootCauseNotApplicable = "not_applicable"
	RootCauseUnknown       = "unknown"
	RootCauseMixed         = "mixed"

	ApplicabilityCurrent    = "current"
	ApplicabilityStale      = "stale"
	ApplicabilitySuperseded = "superseded"
	ApplicabilityUnknown    = "unknown"

	CoverageComplete     = "complete"
	CoveragePartial      = "partial"
	CoverageLagging      = "lagging"
	CoverageUnavailable  = "unavailable"
	CoverageIncompatible = "incompatible"
	CoverageRetentionGap = "retention_gap"
	CoverageUnknown      = "unknown"

	FindingImplicated = "implicated"
	FindingContextual = "contextual"
	FindingUnknown    = "unknown"
)

var (
	ErrInvalidRequest     = errors.New("invalid investigation request")
	ErrInvalidResult      = errors.New("invalid investigation result")
	ErrRequestKeyConflict = errors.New("investigation request key conflicts with an existing request")
)

// Subject is intentionally generic. Plan executions, run sets, and other
// owners use the same shape; owner-specific briefs stay outside this package.
type Subject struct {
	Owner    string   `json:"owner"`
	Kind     string   `json:"kind"`
	Ref      string   `json:"ref"`
	Revision string   `json:"revision"`
	RunIDs   []string `json:"runIds,omitempty"`
}

type EvidenceReference struct {
	Owner         string `json:"owner"`
	Kind          string `json:"kind"`
	Ref           string `json:"ref"`
	Revision      string `json:"revision"`
	SchemaVersion string `json:"schemaVersion"`
	// SubjectRunIDs is optional for evidence that covers the whole requested
	// subject. When present, it is the producer's explicit attribution of the
	// referenced artifact. Consumers must not use a bound reference for a
	// finding or recommendation whose subject does not overlap it.
	SubjectRunIDs []string `json:"subjectRunIds,omitempty"`
}

type MethodReference struct {
	SkillID  string `json:"skillId"`
	Revision string `json:"revision"`
}

type EvidencePolicy struct {
	Mode               string   `json:"mode"`
	RequiredPlanes     []string `json:"requiredPlanes,omitempty"`
	OptionalPlanes     []string `json:"optionalPlanes,omitempty"`
	MaxEvents          int      `json:"maxEvents"`
	MaxEvidenceBytes   int      `json:"maxEvidenceBytes"`
	MaxReconciliations int      `json:"maxReconciliations"`
}

type Budget struct {
	MaxDelegatedRuns  int   `json:"maxDelegatedRuns"`
	MaxTurns          int   `json:"maxTurns"`
	WallSeconds       int   `json:"wallSeconds"`
	MaxChargeMicroUSD int64 `json:"maxChargeMicroUsd"`
}

type RecommendationPolicy struct {
	AllowedKinds         []string `json:"allowedKinds,omitempty"`
	AllowSubjectMutation bool     `json:"allowSubjectMutation"`
}

type Provenance struct {
	Kind                 string `json:"kind,omitempty"`
	TriggerOccurrenceRef string `json:"triggerOccurrenceRef,omitempty"`
}

type Request struct {
	SchemaVersion        string               `json:"schemaVersion"`
	RequestKey           string               `json:"requestKey"`
	CallerAuthority      string               `json:"callerAuthority,omitempty"`
	Subject              Subject              `json:"subject"`
	Question             string               `json:"question,omitempty"`
	MethodRef            *MethodReference     `json:"methodRef,omitempty"`
	DomainEvidence       []EvidenceReference  `json:"domainEvidence,omitempty"`
	EvidencePolicy       EvidencePolicy       `json:"evidencePolicy"`
	Budget               Budget               `json:"budget"`
	RecommendationPolicy RecommendationPolicy `json:"recommendationPolicy"`
	Provenance           Provenance           `json:"provenance,omitempty"`
}

type EvidenceCut struct {
	ID              string            `json:"id"`
	SubjectRevision string            `json:"subjectRevision"`
	RunEventHeads   map[string]string `json:"runEventHeads,omitempty"`
	CapturedAt      string            `json:"capturedAt"`
}

type CoveragePlane struct {
	Plane        string   `json:"plane"`
	State        string   `json:"state"`
	ThroughRef   string   `json:"throughRef,omitempty"`
	EvidenceRefs []string `json:"evidenceRefs,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

type Diagnosis struct {
	Condition          string   `json:"condition"`
	Disposition        string   `json:"disposition"`
	Summary            string   `json:"summary"`
	RootCause          string   `json:"rootCause"`
	Confidence         string   `json:"confidence"`
	UnprovenPredicates []string `json:"unprovenPredicates,omitempty"`
}

type Finding struct {
	ID            string              `json:"id"`
	SubjectRunIDs []string            `json:"subjectRunIds,omitempty"`
	Relation      string              `json:"relation"`
	Summary       string              `json:"summary"`
	EvidenceRefs  []EvidenceReference `json:"evidenceRefs"`
}

type Recommendation struct {
	ID            string              `json:"id"`
	Kind          string              `json:"kind"`
	SubjectRunIDs []string            `json:"subjectRunIds,omitempty"`
	Text          string              `json:"text"`
	EvidenceRefs  []EvidenceReference `json:"evidenceRefs"`
}

type Applicability struct {
	State           string `json:"state"`
	CheckedRevision string `json:"checkedRevision"`
	CheckedAt       string `json:"checkedAt"`
}

type Learning struct {
	AttemptID     string `json:"attemptId"`
	CaptureState  string `json:"captureState"`
	AdviceVerdict string `json:"adviceVerdict"`
}

type NextAction struct {
	Kind  string `json:"kind"`
	Owner string `json:"owner,omitempty"`
	Ref   string `json:"ref,omitempty"`
}

type Result struct {
	SchemaVersion   string              `json:"schemaVersion"`
	InvestigationID string              `json:"investigationId"`
	OperationStatus string              `json:"operationStatus"`
	Diagnosis       Diagnosis           `json:"diagnosis"`
	EvidenceCut     EvidenceCut         `json:"evidenceCut"`
	Coverage        []CoveragePlane     `json:"coverage"`
	Findings        []Finding           `json:"findings"`
	Recommendations []Recommendation    `json:"recommendations"`
	Applicability   Applicability       `json:"applicability"`
	EvidenceRefs    []EvidenceReference `json:"evidenceRefs"`
	Learning        Learning            `json:"learning"`
	NextAction      *NextAction         `json:"nextAction,omitempty"`
}

// DecodeRequest is the strict boundary used by API adapters. Unknown fields
// are rejected so a caller cannot silently believe it supplied a constraint
// that the owner ignored.
func DecodeRequest(raw []byte) (Request, error) {
	var request Request
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) Validate() error {
	if r.SchemaVersion != RequestSchemaVersion {
		return invalidRequest("schemaVersion", "must be %q", RequestSchemaVersion)
	}
	if emptyOrTooLong(r.RequestKey, 256) {
		return invalidRequest("requestKey", "is required and must be at most 256 characters")
	}
	if err := validateSubject(r.Subject); err != nil {
		return err
	}
	if strings.TrimSpace(r.Question) == "" && !validMethod(r.MethodRef) {
		return invalidRequest("question/methodRef", "at least one bounded investigative method is required")
	}
	if r.MethodRef != nil && !validMethod(r.MethodRef) {
		return invalidRequest("methodRef", "skillId and revision are required")
	}
	if r.CallerAuthority != "" && !isKnownAuthority(r.CallerAuthority) {
		return invalidRequest("callerAuthority", "is not an accepted investigation authority")
	}
	if len(r.DomainEvidence) > 50 {
		return invalidRequest("domainEvidence", "must contain at most 50 references")
	}
	for i, ref := range r.DomainEvidence {
		if err := validateEvidenceReference(ref, fmt.Sprintf("domainEvidence[%d]", i)); err != nil {
			return err
		}
		if err := validateEvidenceSubjectRuns(ref.SubjectRunIDs, r.Subject.RunIDs, fmt.Sprintf("domainEvidence[%d].subjectRunIds", i)); err != nil {
			return err
		}
	}
	if len(r.DomainEvidence) > 0 {
		raw, err := json.Marshal(r.DomainEvidence)
		if err != nil || len(raw) > 32*1024 {
			return invalidRequest("domainEvidence", "serialized references must be at most 32768 bytes")
		}
	}
	if r.EvidencePolicy.Mode != "bounded_current" && r.EvidencePolicy.Mode != "bounded_historical" {
		return invalidRequest("evidencePolicy.mode", "must be bounded_current or bounded_historical")
	}
	if r.EvidencePolicy.MaxEvents <= 0 || r.EvidencePolicy.MaxEvents > 10000 {
		return invalidRequest("evidencePolicy.maxEvents", "must be between 1 and 10000")
	}
	if r.EvidencePolicy.MaxEvidenceBytes <= 0 || r.EvidencePolicy.MaxEvidenceBytes > 16*1024*1024 {
		return invalidRequest("evidencePolicy.maxEvidenceBytes", "must be between 1 and 16777216")
	}
	if r.EvidencePolicy.MaxReconciliations < 0 || r.EvidencePolicy.MaxReconciliations > 1 {
		return invalidRequest("evidencePolicy.maxReconciliations", "must be between 0 and 1")
	}
	if r.Budget.MaxDelegatedRuns < 0 || r.Budget.MaxDelegatedRuns > 1 || r.Budget.MaxTurns <= 0 || r.Budget.WallSeconds <= 0 || r.Budget.MaxChargeMicroUSD < 0 {
		return invalidRequest("budget", "must be finite and allow at most one delegated investigator")
	}
	if r.RecommendationPolicy.AllowSubjectMutation {
		return invalidRequest("recommendationPolicy.allowSubjectMutation", "subject mutation is not part of investigation admission")
	}
	return nil
}

func (r Request) CanonicalDigest() (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	canonical := r
	canonical.RequestKey = strings.TrimSpace(canonical.RequestKey)
	canonical.Question = strings.TrimSpace(canonical.Question)
	canonical.CallerAuthority = strings.TrimSpace(canonical.CallerAuthority)
	canonical.Subject.Owner = strings.TrimSpace(canonical.Subject.Owner)
	canonical.Subject.Kind = strings.TrimSpace(canonical.Subject.Kind)
	canonical.Subject.Ref = strings.TrimSpace(canonical.Subject.Ref)
	canonical.Subject.Revision = strings.TrimSpace(canonical.Subject.Revision)
	canonical.Subject.RunIDs = sortedUnique(canonical.Subject.RunIDs)
	canonical.EvidencePolicy.RequiredPlanes = sortedUnique(canonical.EvidencePolicy.RequiredPlanes)
	canonical.EvidencePolicy.OptionalPlanes = sortedUnique(canonical.EvidencePolicy.OptionalPlanes)
	canonical.RecommendationPolicy.AllowedKinds = sortedUnique(canonical.RecommendationPolicy.AllowedKinds)
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("marshal canonical investigation request: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func (r Result) Validate(request Request) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("%w: request: %v", ErrInvalidResult, err)
	}
	if r.SchemaVersion != ResultSchemaVersion {
		return invalidResult("schemaVersion", "must be %q", ResultSchemaVersion)
	}
	if emptyOrTooLong(r.InvestigationID, 256) || !validOperationStatus(r.OperationStatus) {
		return invalidResult("identity/status", "investigationId is required and operationStatus must be a known finite state")
	}
	if strings.TrimSpace(r.Diagnosis.Summary) == "" || !validCondition(r.Diagnosis.Condition) || !validDisposition(r.Diagnosis.Disposition) || !validConfidence(r.Diagnosis.Confidence) {
		return invalidResult("diagnosis", "condition, disposition, summary, and confidence are required")
	}
	if r.Diagnosis.RootCause == "" {
		return invalidResult("diagnosis.rootCause", "must explicitly identify a cause, unknown, mixed, or not_applicable")
	}
	if strings.TrimSpace(r.EvidenceCut.ID) == "" || strings.TrimSpace(r.EvidenceCut.SubjectRevision) == "" || strings.TrimSpace(r.EvidenceCut.CapturedAt) == "" {
		return invalidResult("evidenceCut", "id, subjectRevision, and capturedAt are required")
	}
	if r.EvidenceCut.SubjectRevision != request.Subject.Revision {
		return invalidResult("evidenceCut.subjectRevision", "must match the requested subject revision")
	}
	if r.Diagnosis.Disposition == DispositionNoIntervention && len(r.Recommendations) != 0 {
		return invalidResult("recommendations", "no_intervention results must contain zero recommendations")
	}
	if len(r.Coverage) == 0 {
		return invalidResult("coverage", "at least one evidence plane must be declared")
	}
	coveredPlanes := make(map[string]CoveragePlane, len(r.Coverage))
	validRuns := make(map[string]struct{}, len(request.Subject.RunIDs))
	for _, runID := range request.Subject.RunIDs {
		validRuns[runID] = struct{}{}
	}
	knownEvidence := make(map[string]struct{}, len(r.EvidenceRefs))
	for i, ref := range r.EvidenceRefs {
		if err := validateEvidenceReference(ref, fmt.Sprintf("evidenceRefs[%d]", i)); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidResult, err)
		}
		if err := validateEvidenceSubjectRuns(ref.SubjectRunIDs, request.Subject.RunIDs, fmt.Sprintf("evidenceRefs[%d].subjectRunIds", i)); err != nil {
			return err
		}
		knownEvidence[evidenceKey(ref)] = struct{}{}
	}
	for i, plane := range r.Coverage {
		if strings.TrimSpace(plane.Plane) == "" || !validCoverageState(plane.State) {
			return invalidResult(fmt.Sprintf("coverage[%d]", i), "plane and a known state are required")
		}
		coveredPlanes[plane.Plane] = plane
		if (plane.State == CoverageUnavailable || plane.State == CoverageUnknown || plane.State == CoverageLagging || plane.State == CoverageIncompatible || plane.State == CoverageRetentionGap) && strings.TrimSpace(plane.Reason) == "" {
			return invalidResult(fmt.Sprintf("coverage[%d].reason", i), "unknown or incomplete evidence must explain why")
		}
	}
	for _, required := range request.EvidencePolicy.RequiredPlanes {
		plane, ok := coveredPlanes[required]
		if !ok {
			return invalidResult("coverage", "required evidence plane %q is missing", required)
		}
		if supportedHealthyDiagnosis(r.Diagnosis) && plane.State != CoverageComplete {
			return invalidResult("diagnosis", "supported no-intervention conclusions require complete coverage for required plane %q", required)
		}
	}
	for i, finding := range r.Findings {
		if err := validateFinding(finding, validRuns, knownEvidence, fmt.Sprintf("findings[%d]", i)); err != nil {
			return err
		}
	}
	for i, recommendation := range r.Recommendations {
		if strings.TrimSpace(recommendation.ID) == "" || strings.TrimSpace(recommendation.Kind) == "" || strings.TrimSpace(recommendation.Text) == "" {
			return invalidResult(fmt.Sprintf("recommendations[%d]", i), "id, kind, and text are required")
		}
		if !contains(request.RecommendationPolicy.AllowedKinds, recommendation.Kind) {
			return invalidResult(fmt.Sprintf("recommendations[%d].kind", i), "kind %q is not allowed by the caller policy", recommendation.Kind)
		}
		if err := validateSubjectRuns(recommendation.SubjectRunIDs, validRuns, fmt.Sprintf("recommendations[%d].subjectRunIds", i)); err != nil {
			return err
		}
		if err := validateEvidenceRefs(recommendation.EvidenceRefs, knownEvidence, validRuns, recommendation.SubjectRunIDs, fmt.Sprintf("recommendations[%d].evidenceRefs", i)); err != nil {
			return err
		}
	}
	if r.Diagnosis.RootCause == RootCauseMixed {
		refs := make(map[string]struct{})
		for _, finding := range r.Findings {
			for _, ref := range finding.EvidenceRefs {
				refs[evidenceKey(ref)] = struct{}{}
			}
		}
		if len(refs) < 2 {
			return invalidResult("diagnosis.rootCause", "mixed cause requires at least two distinct evidence chains")
		}
	}
	if !validApplicability(r.Applicability.State) || strings.TrimSpace(r.Applicability.CheckedRevision) == "" {
		return invalidResult("applicability", "state and checkedRevision are required")
	}
	return nil
}

func supportedHealthyDiagnosis(d Diagnosis) bool {
	confidence := strings.ToLower(strings.TrimSpace(d.Confidence))
	return d.Disposition == DispositionNoIntervention &&
		(d.Condition == ConditionProgressing || d.Condition == ConditionComplete) &&
		(confidence == "supported" || confidence == "high")
}

func validateFinding(f Finding, validRuns map[string]struct{}, knownEvidence map[string]struct{}, path string) error {
	if strings.TrimSpace(f.ID) == "" || strings.TrimSpace(f.Summary) == "" || (f.Relation != FindingImplicated && f.Relation != FindingContextual && f.Relation != FindingUnknown) {
		return invalidResult(path, "id, summary, and a known relation are required")
	}
	if err := validateSubjectRuns(f.SubjectRunIDs, validRuns, path+".subjectRunIds"); err != nil {
		return err
	}
	if f.Relation == FindingImplicated && len(f.SubjectRunIDs) == 0 {
		return invalidResult(path+".subjectRunIds", "an implicated finding must name at least one subject run")
	}
	return validateEvidenceRefs(f.EvidenceRefs, knownEvidence, validRuns, f.SubjectRunIDs, path+".evidenceRefs")
}

func validateEvidenceRefs(refs []EvidenceReference, known, validRuns map[string]struct{}, subjectRuns []string, path string) error {
	if len(refs) == 0 {
		return invalidResult(path, "at least one typed evidence reference is required")
	}
	for i, ref := range refs {
		if err := validateEvidenceReference(ref, fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
		if err := validateEvidenceSubjectRuns(ref.SubjectRunIDs, keysFromRunSet(validRuns), fmt.Sprintf("%s[%d].subjectRunIds", path, i)); err != nil {
			return err
		}
		if len(subjectRuns) > 0 && len(ref.SubjectRunIDs) == 0 {
			return invalidResult(path, "subject-scoped references must declare subject run ids")
		}
		if len(subjectRuns) > 0 && len(ref.SubjectRunIDs) > 0 && !evidenceOverlapsSubject(ref.SubjectRunIDs, subjectRuns) {
			return invalidResult(path, "reference is explicitly attributed to a different subject run")
		}
		if len(known) > 0 {
			if _, ok := known[evidenceKey(ref)]; !ok {
				return invalidResult(path, "reference is not present in result evidenceRefs")
			}
		}
	}
	return nil
}

func validateSubject(subject Subject) error {
	if emptyOrTooLong(subject.Owner, 128) || emptyOrTooLong(subject.Kind, 128) || emptyOrTooLong(subject.Ref, 512) || emptyOrTooLong(subject.Revision, 256) {
		return invalidRequest("subject", "owner, kind, ref, and revision are required and bounded")
	}
	seen := map[string]struct{}{}
	for _, runID := range subject.RunIDs {
		runID = strings.TrimSpace(runID)
		if runID == "" {
			return invalidRequest("subject.runIds", "run ids cannot be empty")
		}
		if _, ok := seen[runID]; ok {
			return invalidRequest("subject.runIds", "run ids must be unique")
		}
		seen[runID] = struct{}{}
	}
	return nil
}

func validateSubjectRuns(ids []string, valid map[string]struct{}, path string) error {
	for _, id := range ids {
		if _, ok := valid[id]; !ok {
			return invalidResult(path, "run id %q is outside the requested subject", id)
		}
	}
	return nil
}

func validateEvidenceReference(ref EvidenceReference, path string) error {
	if emptyOrTooLong(ref.Owner, 128) || emptyOrTooLong(ref.Kind, 128) || emptyOrTooLong(ref.Ref, 1024) || emptyOrTooLong(ref.Revision, 256) || emptyOrTooLong(ref.SchemaVersion, 256) {
		return invalidResult(path, "owner, kind, ref, revision, and schemaVersion are required and bounded")
	}
	return nil
}

func validateEvidenceSubjectRuns(ids, valid []string, path string) error {
	if len(ids) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	validSet := make(map[string]struct{}, len(valid))
	for _, id := range valid {
		validSet[id] = struct{}{}
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return invalidResult(path, "subject run ids cannot be empty")
		}
		if _, ok := seen[id]; ok {
			return invalidResult(path, "subject run ids must be unique")
		}
		seen[id] = struct{}{}
		if _, ok := validSet[id]; !ok {
			return invalidResult(path, "run id %q is outside the requested subject", id)
		}
	}
	return nil
}

func keysFromRunSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}

func evidenceOverlapsSubject(evidenceRuns, subjectRuns []string) bool {
	for _, evidenceRun := range evidenceRuns {
		for _, subjectRun := range subjectRuns {
			if evidenceRun == subjectRun {
				return true
			}
		}
	}
	return false
}

func validMethod(ref *MethodReference) bool {
	return ref != nil && strings.TrimSpace(ref.SkillID) != "" && strings.TrimSpace(ref.Revision) != ""
}
func isKnownAuthority(value string) bool {
	switch value {
	case "agent", "service", "plan-manager", "test", "system":
		return true
	default:
		return false
	}
}
func validOperationStatus(value string) bool {
	switch value {
	case OperationQueued, OperationCollecting, OperationDiagnosing, OperationCompleted, OperationFailed, OperationCancelled:
		return true
	default:
		return false
	}
}
func validCondition(value string) bool {
	switch value {
	case ConditionProgressing, ConditionWaiting, ConditionStalled, ConditionBlocked, ConditionFailed, ConditionComplete, ConditionUnknown:
		return true
	default:
		return false
	}
}
func validDisposition(value string) bool {
	switch value {
	case DispositionNoIntervention, DispositionRecommendAction, DispositionInconclusive:
		return true
	default:
		return false
	}
}
func validConfidence(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high", "medium", "low", "supported":
		return true
	default:
		return false
	}
}
func validCoverageState(value string) bool {
	switch value {
	case CoverageComplete, CoveragePartial, CoverageLagging, CoverageUnavailable, CoverageIncompatible, CoverageRetentionGap, CoverageUnknown:
		return true
	default:
		return false
	}
}
func validApplicability(value string) bool {
	switch value {
	case ApplicabilityCurrent, ApplicabilityStale, ApplicabilitySuperseded, ApplicabilityUnknown:
		return true
	default:
		return false
	}
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func emptyOrTooLong(value string, max int) bool {
	value = strings.TrimSpace(value)
	return value == "" || len(value) > max
}
func sortedUnique(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
func evidenceKey(ref EvidenceReference) string {
	return strings.Join([]string{ref.Owner, ref.Kind, ref.Ref, ref.Revision, ref.SchemaVersion, strings.Join(sortedUnique(ref.SubjectRunIDs), ",")}, "\x00")
}
func invalidRequest(path, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidRequest, path, fmt.Sprintf(format, args...))
}
func invalidResult(path, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidResult, path, fmt.Sprintf(format, args...))
}
