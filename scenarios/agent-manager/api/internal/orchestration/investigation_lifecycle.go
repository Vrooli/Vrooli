package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/investigation"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
)

// ReconcileTypedInvestigation is the exported completion seam used by the
// handler when a workflow settles before its dispatch acknowledgement is
// persisted. Normal completions arrive through onWorkflowExecutionSettled;
// this method closes the synchronous fast-workflow race.
func (o *Orchestrator) ReconcileTypedInvestigation(ctx context.Context, execution *domain.WorkflowExecution) {
	o.reconcileTypedInvestigation(ctx, execution)
}

// RetryPendingInvestigationLearning replays only the retained, immutable
// result payload. It never reruns diagnosis and is safe to call after a
// memory timeout or lost acknowledgement.
func (o *Orchestrator) RetryPendingInvestigationLearning(ctx context.Context, id string) (*investigation.Lifecycle, error) {
	if o == nil || o.typedInvestigations == nil {
		return nil, fmt.Errorf("investigation lifecycle is unavailable")
	}
	item, err := o.typedInvestigations.Get(ctx, id)
	if err != nil {
		return item, err
	}
	if item.Result == nil {
		return item, fmt.Errorf("%w: investigation has no completed result", investigation.ErrInvalidTransition)
	}
	if item.Result.Learning.CaptureState != "pending" || o.learningRecorder == nil {
		return item, nil
	}
	if item.Result.Learning.AttemptID == "" {
		item.Result.Learning.AttemptID = item.ID + "/attempt-1"
	}
	if err := o.learningRecorder.RecordInvestigation(ctx, item); err != nil {
		return item, err
	}
	updater, ok := o.typedInvestigations.(investigation.LearningUpdater)
	if !ok {
		return item, fmt.Errorf("investigation repository does not support learning-state updates")
	}
	item.Result.Learning.CaptureState = "recorded"
	return updater.UpdateLearning(ctx, id, item.Result.Learning)
}

// reconcileTypedInvestigation is the restart-safe bridge from the workflow
// engine's terminal record to the caller-neutral investigation result. It
// deliberately adapts the existing structured investigation projection; the
// typed lifecycle remains the durable contract exposed to callers.
func (o *Orchestrator) reconcileTypedInvestigation(ctx context.Context, execution *domain.WorkflowExecution) {
	if o == nil || o.typedInvestigations == nil || execution == nil || execution.WorkflowKey != investigation.WorkflowKey {
		return
	}
	items, err := o.typedInvestigations.List(ctx, "", 200)
	if err != nil {
		return
	}
	for _, item := range items {
		if item == nil || item.WorkflowRef != execution.ID.String() || investigationTerminalStatus(item.OperationStatus) {
			continue
		}
		if execution.Status != domain.WorkflowExecutionSucceeded {
			_, _ = o.typedInvestigations.UpdateStatus(ctx, item.ID, investigation.OperationFailed, "")
			return
		}
		if item.OperationStatus == investigation.OperationQueued {
			if _, err := o.typedInvestigations.UpdateStatus(ctx, item.ID, investigation.OperationCollecting, execution.ID.String()); err != nil {
				return
			}
		}
		if item.OperationStatus == investigation.OperationCollecting {
			if _, err := o.typedInvestigations.UpdateStatus(ctx, item.ID, investigation.OperationDiagnosing, execution.ID.String()); err != nil {
				return
			}
		}
		result := typedInvestigationResult(item, execution, o.now(), o.authoritativeInvestigationCoverage(ctx, item.Request))
		// The attempt identity is part of the durable result, not only an
		// internal recorder fallback. Callers must be able to reattach to the
		// exact learning attempt after a successful capture or a retry.
		if result.Learning.AttemptID == "" {
			result.Learning.AttemptID = item.ID + "/attempt-1"
		}
		if o.learningRecorder != nil {
			candidate := *item
			candidate.Result = &result
			if err := o.learningRecorder.RecordInvestigation(ctx, &candidate); err == nil {
				result.Learning.CaptureState = "recorded"
			}
		}
		if err := result.Validate(item.Request); err != nil {
			return
		}
		_, _ = o.typedInvestigations.Complete(ctx, item.ID, result)
		return
	}
}

func typedInvestigationResult(item *investigation.Lifecycle, execution *domain.WorkflowExecution, now time.Time, authoritativeCoverage ...[]investigation.CoveragePlane) investigation.Result {
	evidence := investigation.EvidenceReference{
		Owner:         "agent-manager",
		Kind:          "workflow-execution",
		Ref:           execution.ID.String(),
		Revision:      execution.DefinitionDigest,
		SchemaVersion: "workflow-execution/v1",
		SubjectRunIDs: append([]string(nil), item.Request.Subject.RunIDs...),
	}
	cut := investigation.EvidenceCut{ID: "workflow/" + execution.ID.String(), SubjectRevision: item.Request.Subject.Revision, CapturedAt: now.UTC().Format(time.RFC3339Nano)}
	coverage := []investigation.CoveragePlane{{Plane: "workflow_result", State: investigation.CoverageComplete, ThroughRef: execution.ID.String(), EvidenceRefs: []string{execution.ID.String()}}}
	if len(authoritativeCoverage) > 0 && len(authoritativeCoverage[0]) > 0 {
		coverage = append([]investigation.CoveragePlane(nil), authoritativeCoverage[0]...)
		for i := range coverage {
			if coverage[i].ThroughRef == "" {
				coverage[i].ThroughRef = execution.ID.String()
			}
			if len(coverage[i].EvidenceRefs) == 0 {
				coverage[i].EvidenceRefs = []string{execution.ID.String()}
			}
		}
	} else {
		for _, required := range item.Request.EvidencePolicy.RequiredPlanes {
			if strings.TrimSpace(required) == "" || required == "workflow_result" {
				continue
			}
			coverage = append(coverage, investigation.CoveragePlane{
				Plane: required, State: investigation.CoveragePartial, ThroughRef: execution.ID.String(), EvidenceRefs: []string{execution.ID.String()},
				Reason: "the provider received a bounded projection for this plane; per-plane completeness was not persisted in the workflow result",
			})
		}
	}
	result := investigation.Result{
		SchemaVersion:   investigation.ResultSchemaVersion,
		InvestigationID: item.ID,
		OperationStatus: investigation.OperationCompleted,
		Diagnosis:       investigation.Diagnosis{Condition: investigation.ConditionComplete, Disposition: investigation.DispositionInconclusive, Summary: "The diagnosis workflow completed without an actionable typed recommendation.", RootCause: investigation.RootCauseUnknown, Confidence: "Low"},
		EvidenceCut:     cut,
		Coverage:        coverage,
		Applicability:   investigation.Applicability{State: investigation.ApplicabilityCurrent, CheckedRevision: item.Request.Subject.Revision, CheckedAt: cut.CapturedAt},
		EvidenceRefs:    append([]investigation.EvidenceReference{evidence}, item.Request.DomainEvidence...),
		Learning:        investigation.Learning{AttemptID: item.ID + "/attempt-1", CaptureState: "pending", AdviceVerdict: "unreviewed"},
	}

	var output investigationWorkflowOutput
	if json.Unmarshal(execution.Output, &output) != nil {
		return result
	}
	if strings.TrimSpace(output.Findings.Summary) != "" {
		result.Diagnosis.Summary = output.Findings.Summary
	}
	if output.Findings.Confidence != "" {
		result.Diagnosis.Confidence = output.Findings.Confidence
	}
	if output.Findings.Condition != "" {
		result.Diagnosis.Condition = output.Findings.Condition
	}
	if output.Findings.Disposition != "" {
		result.Diagnosis.Disposition = output.Findings.Disposition
	}
	if output.Findings.RootCause != "" {
		result.Diagnosis.RootCause = output.Findings.RootCause
	}
	result.Diagnosis.UnprovenPredicates = append([]string(nil), output.Findings.UnprovenPredicates...)
	if output.Findings.PrimaryCategory != "" {
		result.Findings = append(result.Findings, investigation.Finding{ID: "category-primary", Relation: investigation.FindingContextual, Summary: output.Findings.PrimaryCategory, EvidenceRefs: []investigation.EvidenceReference{evidence}})
	}
	for categoryIndex, category := range output.Findings.Categories {
		if strings.TrimSpace(category.Name) != "" {
			relation := investigation.FindingContextual
			if len(category.SubjectRunIDs) > 0 {
				relation = investigation.FindingImplicated
			}
			refs := acceptedInvestigationEvidence(category.EvidenceRefs, item.Request.DomainEvidence, category.SubjectRunIDs, evidence)
			result.Findings = append(result.Findings, investigation.Finding{ID: fmt.Sprintf("category-%d", categoryIndex+1), SubjectRunIDs: category.SubjectRunIDs, Relation: relation, Summary: category.Name, EvidenceRefs: refs})
		}
		for recommendationIndex, recommendation := range category.Recommendations {
			kind := allowedInvestigationKind(item.Request.RecommendationPolicy.AllowedKinds)
			if kind == "" || strings.TrimSpace(recommendation.Text) == "" {
				continue
			}
			refs := acceptedInvestigationEvidence(recommendation.EvidenceRefs, item.Request.DomainEvidence, recommendation.SubjectRunIDs, evidence)
			result.Recommendations = append(result.Recommendations, investigation.Recommendation{ID: fmt.Sprintf("recommendation-%d-%d", categoryIndex+1, recommendationIndex+1), Kind: kind, SubjectRunIDs: recommendation.SubjectRunIDs, Text: recommendation.Text, EvidenceRefs: refs})
		}
	}
	if len(result.Recommendations) > 0 {
		result.Diagnosis.Disposition = investigation.DispositionRecommendAction
	} else if len(output.Findings.Categories) > 0 || output.Findings.PrimaryCategory != "" {
		result.Diagnosis.Disposition = investigation.DispositionNoIntervention
	}
	normalizeTypedInvestigationDiagnosis(&result.Diagnosis, &result.Recommendations, item.Request, coverage)
	return result
}

// authoritativeInvestigationCoverage derives coverage from the same bounded
// run reports supplied to admission. Model output cannot upgrade an
// unavailable or degraded source plane, so the lifecycle result records the
// owner-side availability rather than a provider assertion.
func (o *Orchestrator) authoritativeInvestigationCoverage(ctx context.Context, request investigation.Request) []investigation.CoveragePlane {
	if len(request.Subject.RunIDs) == 0 {
		return []investigation.CoveragePlane{{Plane: "workflow_result", State: investigation.CoverageComplete}}
	}
	planes := append([]string{"workflow_result"}, request.EvidencePolicy.RequiredPlanes...)
	planes = append(planes, request.EvidencePolicy.OptionalPlanes...)
	seen := make(map[string]struct{}, len(planes))
	coverage := make([]investigation.CoveragePlane, 0, len(planes))
	for _, plane := range planes {
		plane = strings.TrimSpace(plane)
		if plane == "" {
			continue
		}
		if _, exists := seen[plane]; exists {
			continue
		}
		seen[plane] = struct{}{}
		if plane == "workflow_result" {
			coverage = append(coverage, investigation.CoveragePlane{Plane: plane, State: investigation.CoverageComplete})
			continue
		}
		state, reason := o.coverageForPlane(ctx, request.Subject.RunIDs, plane)
		coverage = append(coverage, investigation.CoveragePlane{Plane: plane, State: state, Reason: reason})
	}
	return coverage
}

func (o *Orchestrator) coverageForPlane(ctx context.Context, runIDs []string, plane string) (string, string) {
	if o == nil {
		return investigation.CoverageUnavailable, "run-report service is unavailable"
	}
	complete := true
	partial := false
	for _, rawID := range runIDs {
		runID, err := uuid.Parse(rawID)
		if err != nil {
			return investigation.CoverageIncompatible, fmt.Sprintf("subject run %q is not a UUID", rawID)
		}
		report, err := o.BuildRunReport(ctx, runID)
		if err != nil || report == nil {
			if err != nil {
				return investigation.CoverageUnavailable, fmt.Sprintf("run report unavailable: %v", err)
			}
			return investigation.CoverageUnavailable, "run report unavailable"
		}
		availability, ok := runReportAvailability(report, plane)
		if !ok {
			return investigation.CoverageUnknown, "no bounded run-report mapping exists for this plane"
		}
		switch availability.State {
		case runreport.AvailabilityAvailable, runreport.AvailabilityComplete, runreport.AvailabilityEmpty:
			continue
		case runreport.AvailabilityDegraded, runreport.AvailabilityUnreliable:
			complete = false
			partial = true
		default:
			complete = false
			return investigation.CoverageUnavailable, availability.Reason
		}
	}
	if complete {
		return investigation.CoverageComplete, "bounded run-report projection is available through the captured cut"
	}
	if partial {
		return investigation.CoveragePartial, "bounded run-report projection is degraded for at least one subject run"
	}
	return investigation.CoverageUnknown, "bounded run-report projection has no compatible source state"
}

// runReportAvailability maps investigation evidence planes to the report
// field that actually describes that plane. Invocation facts are derived from
// the run's captured events (or the durable invocation projection); receipt
// projection is a separate, optional cross-scenario evidence plane.
func runReportAvailability(report *runreport.RunReport, plane string) (runreport.Availability, bool) {
	if report == nil {
		return runreport.Availability{}, false
	}
	switch plane {
	case "run_state":
		return runreport.Availability{State: runreport.AvailabilityComplete}, true
	case "events":
		return report.EventsAvailability, true
	case "invocations":
		return report.InvocationValidity.Availability, true
	case "receipts":
		return report.ReceiptsAvailability, true
	case "diff":
		return report.Diff.Available, true
	default:
		return runreport.Availability{}, false
	}
}

// normalizeTypedInvestigationDiagnosis is the final safety boundary between
// a model-shaped payload and the caller-neutral result contract. A model may
// describe a clean or failed run confidently while the provider only persisted
// a partial projection. In that case no_intervention would overstate what the
// owner knows, so retain the condition but downgrade the conclusion to an
// explicit inconclusive result. The same rule applies to an unexplained failed
// run: a terminal failure is evidence of a condition, not proof of its cause.
func normalizeTypedInvestigationDiagnosis(diagnosis *investigation.Diagnosis, recommendations *[]investigation.Recommendation, request investigation.Request, coverage []investigation.CoveragePlane) {
	if diagnosis == nil {
		return
	}
	confidence := strings.ToLower(strings.TrimSpace(diagnosis.Confidence))
	unknownFailedCause := diagnosis.Condition == investigation.ConditionFailed && diagnosis.RootCause == investigation.RootCauseUnknown && confidence != "high"
	if diagnosis.Disposition == investigation.DispositionRecommendAction && (confidence == "low" || unknownFailedCause || hasIncompleteRequiredCoverage(request, coverage)) {
		diagnosis.Disposition = investigation.DispositionInconclusive
		diagnosis.Confidence = "Low"
		diagnosis.UnprovenPredicates = appendUniquePredicate(diagnosis.UnprovenPredicates, "whether the available evidence supports a corrective recommendation")
		if recommendations != nil {
			*recommendations = nil
		}
		return
	}
	if diagnosis.Disposition != investigation.DispositionNoIntervention {
		return
	}
	if hasIncompleteRequiredCoverage(request, coverage) {
		diagnosis.Disposition = investigation.DispositionInconclusive
		diagnosis.Confidence = "Low"
		diagnosis.UnprovenPredicates = appendUniquePredicate(diagnosis.UnprovenPredicates, "whether every required evidence plane is complete at the captured cut")
		return
	}
	if diagnosis.Condition == investigation.ConditionFailed && diagnosis.RootCause == investigation.RootCauseUnknown {
		diagnosis.Disposition = investigation.DispositionInconclusive
		diagnosis.Confidence = "Low"
		diagnosis.UnprovenPredicates = appendUniquePredicate(diagnosis.UnprovenPredicates, "which causal predicate explains the terminal failure")
	}
}

func hasIncompleteRequiredCoverage(request investigation.Request, coverage []investigation.CoveragePlane) bool {
	states := make(map[string]string, len(coverage))
	for _, plane := range coverage {
		states[plane.Plane] = plane.State
	}
	for _, required := range request.EvidencePolicy.RequiredPlanes {
		if required == "" || states[required] != investigation.CoverageComplete {
			return true
		}
	}
	return false
}

func appendUniquePredicate(predicates []string, predicate string) []string {
	for _, existing := range predicates {
		if existing == predicate {
			return predicates
		}
	}
	return append(predicates, predicate)
}

// acceptedInvestigationEvidence is intentionally allow-list based. The model
// may cite a reference the caller supplied, but it cannot mint a new owner,
// revision, or locator. Every projected finding still carries the durable
// workflow execution reference as a fallback.
func acceptedInvestigationEvidence(candidates, allowed []investigation.EvidenceReference, subjectRuns []string, fallback investigation.EvidenceReference) []investigation.EvidenceReference {
	refs := []investigation.EvidenceReference{fallback}
	for _, candidate := range candidates {
		for _, known := range allowed {
			if sameInvestigationEvidence(candidate, known) && evidenceSupportsSubject(known, subjectRuns) {
				refs = append(refs, known)
				break
			}
		}
	}
	return refs
}

func evidenceSupportsSubject(ref investigation.EvidenceReference, subjectRuns []string) bool {
	if len(subjectRuns) == 0 {
		return true
	}
	if len(ref.SubjectRunIDs) == 0 {
		return false
	}
	for _, subjectRun := range subjectRuns {
		for _, evidenceRun := range ref.SubjectRunIDs {
			if subjectRun == evidenceRun {
				return true
			}
		}
	}
	return false
}

func sameInvestigationEvidence(left, right investigation.EvidenceReference) bool {
	if left.Owner != right.Owner || left.Kind != right.Kind || left.Ref != right.Ref || left.Revision != right.Revision || left.SchemaVersion != right.SchemaVersion || len(left.SubjectRunIDs) != len(right.SubjectRunIDs) {
		return false
	}
	seen := make(map[string]struct{}, len(left.SubjectRunIDs))
	for _, id := range left.SubjectRunIDs {
		seen[id] = struct{}{}
	}
	for _, id := range right.SubjectRunIDs {
		if _, ok := seen[id]; !ok {
			return false
		}
	}
	return true
}

type investigationWorkflowOutput struct {
	Findings investigationWorkflowFindings `json:"findings"`
}

type investigationWorkflowFindings struct {
	Condition          string   `json:"condition"`
	Disposition        string   `json:"disposition"`
	RootCause          string   `json:"rootCause"`
	UnprovenPredicates []string `json:"unprovenPredicates"`
	Summary            string   `json:"summary"`
	PrimaryCategory    string   `json:"primaryCategory"`
	Confidence         string   `json:"confidence"`
	Categories         []struct {
		Name            string                            `json:"name"`
		SubjectRunIDs   []string                          `json:"subjectRunIds"`
		EvidenceRefs    []investigation.EvidenceReference `json:"evidenceRefs"`
		Recommendations []struct {
			Text          string                            `json:"text"`
			SubjectRunIDs []string                          `json:"subjectRunIds"`
			EvidenceRefs  []investigation.EvidenceReference `json:"evidenceRefs"`
		} `json:"recommendations"`
	} `json:"categories"`
}

func allowedInvestigationKind(allowed []string) string {
	for _, candidate := range []string{"observe", "recommend", "recommend_action", "investigate"} {
		for _, value := range allowed {
			if strings.TrimSpace(value) == candidate {
				return candidate
			}
		}
	}
	return ""
}

func investigationTerminalStatus(status string) bool {
	return status == investigation.OperationCompleted || status == investigation.OperationFailed || status == investigation.OperationCancelled
}
