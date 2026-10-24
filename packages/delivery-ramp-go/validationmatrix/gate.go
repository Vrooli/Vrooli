package validationmatrix

import (
	"fmt"
	"strings"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

// EvaluateReleaseGate applies the fail-closed matrix rule. Non-applicable
// cells are not required coverage; every required applicable cell must be a
// pass with at least one linked, checksummed, redacted evidence item.
func EvaluateReleaseGate(matrix *domainv1.ValidationMatrix) *domainv1.ReleaseGate {
	gate := &domainv1.ReleaseGate{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED}
	if matrix != nil {
		gate.MatrixId = matrix.GetMatrixId()
	}
	if matrix == nil || strings.TrimSpace(matrix.GetMatrixId()) == "" {
		gate.Reason = stringPtr("validation matrix identity is missing")
		return gate
	}
	_, structured, err := expectedRequiredCells(matrix)
	if err != nil {
		gate.Reason = stringPtr(err.Error())
		return gate
	}
	if structured {
		return evaluateMatrixWideGate(matrix, gate)
	}
	observed := make(map[string]*domainv1.ValidationCell, len(matrix.GetCells()))
	for _, cell := range matrix.GetCells() {
		if cell != nil {
			key := cellKey(cell)
			if key != "" {
				if _, duplicate := observed[key]; duplicate {
					gate.FailedCellIds = append(gate.FailedCellIds, cell.GetCellId())
				} else {
					observed[key] = cell
				}
			}
		}
		if cell == nil || !cell.GetRequired() || !cell.GetApplicable() {
			continue
		}
		gate.RequiredCellCount++
		if cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS {
			gate.FailedCellIds = append(gate.FailedCellIds, cell.GetCellId())
			continue
		}
		if !completeEvidence(matrix, cell) {
			gate.FailedCellIds = append(gate.FailedCellIds, cell.GetCellId())
			continue
		}
		gate.PassingCellCount++
	}
	if gate.RequiredCellCount == 0 {
		gate.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN
		gate.Reason = stringPtr("no required applicable validation cells exist")
		return gate
	}
	if len(gate.FailedCellIds) > 0 || gate.PassingCellCount != gate.RequiredCellCount {
		gate.Reason = stringPtr(fmt.Sprintf("%d required validation cell(s) failed or lack complete evidence", len(gate.FailedCellIds)))
		return gate
	}
	gate.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS
	gate.Passed = true
	return gate
}

// evaluateMatrixWideGate treats a required journey/profile as the unit of
// release coverage. A target that cannot advertise the journey's required
// capabilities is not a failed required cell; another target may satisfy the
// chapter. The selected target is retained on the gate so reports cannot
// mistake emulator coverage for physical-device coverage.
func evaluateMatrixWideGate(matrix *domainv1.ValidationMatrix, gate *domainv1.ReleaseGate) *domainv1.ReleaseGate {
	observed := make(map[string]*domainv1.ValidationCell, len(matrix.GetCells()))
	for _, cell := range matrix.GetCells() {
		if cell == nil {
			continue
		}
		key := cellKey(cell)
		if key == "" {
			continue
		}
		if _, duplicate := observed[key]; duplicate {
			gate.FailedCellIds = append(gate.FailedCellIds, cell.GetCellId())
			continue
		}
		observed[key] = cell
	}

	targets := make(map[string]*domainv1.ValidationTargetDescriptor, len(matrix.GetTargets()))
	for _, target := range matrix.GetTargets() {
		if target != nil {
			targets[target.GetTargetId()] = target
		}
	}
	for _, journey := range matrix.GetJourneys() {
		if journey == nil || !journey.GetRequired() {
			continue
		}
		for _, profile := range matrix.GetEnvironmentProfiles() {
			chapterKey := chapterKey(journey.GetJourneyId(), profile)
			gate.RequiredCellCount++
			candidateIDs := make([]string, 0, len(targets))
			for _, target := range matrix.GetTargets() {
				// Keep the capability filter adjacent to the matrix snapshot so
				// every candidate is evaluated against the same immutable inputs.
				if target == nil || !target.GetAvailable() || !supportsCapabilityEnums(target.GetCapabilities(), journey.GetRequiredCapabilities()) {
					continue
				}
				candidateIDs = append(candidateIDs, target.GetTargetId())
			}
			if len(candidateIDs) == 0 {
				gate.MissingCellIds = append(gate.MissingCellIds, chapterKey)
				continue
			}

			passed := false
			for _, targetID := range candidateIDs {
				cell := observed[selectionKey(journey.GetJourneyId(), targetID, profile)]
				if cell == nil {
					continue
				}
				if !cell.GetRequired() || !cell.GetApplicable() || cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS || !completeEvidence(matrix, cell) {
					continue
				}
				passed = true
				gate.PassingCellCount++
				if gate.SatisfyingTargetIds == nil {
					gate.SatisfyingTargetIds = make(map[string]string)
				}
				gate.SatisfyingTargetIds[chapterKey] = targetID
				break
			}
			if passed {
				continue
			}
			for _, targetID := range candidateIDs {
				if cell := observed[selectionKey(journey.GetJourneyId(), targetID, profile)]; cell != nil {
					gate.FailedCellIds = append(gate.FailedCellIds, cell.GetCellId())
				}
			}
			if len(gate.FailedCellIds) == 0 || !containsCellForChapter(gate.FailedCellIds, observed, journey.GetJourneyId(), profile) {
				gate.MissingCellIds = append(gate.MissingCellIds, chapterKey)
			}
		}
	}
	if len(gate.MissingCellIds) > 0 {
		gate.Reason = stringPtr("required journey coverage is incomplete")
		return gate
	}
	if len(gate.FailedCellIds) > 0 || gate.PassingCellCount != gate.RequiredCellCount {
		gate.Reason = stringPtr(fmt.Sprintf("%d required journey/profile chapter(s) failed or lack complete evidence", gate.RequiredCellCount-gate.PassingCellCount))
		return gate
	}
	gate.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS
	gate.Passed = true
	return gate
}

func chapterKey(journeyID string, profile domainv1.ValidationEnvironmentProfile) string {
	return fmt.Sprintf("journey=%s profile=%s", journeyID, profile.String())
}

func supportsCapabilityEnums(observed, required []domainv1.ValidationTargetCapability) bool {
	for _, want := range required {
		found := false
		for _, got := range observed {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func containsCellForChapter(failed []string, observed map[string]*domainv1.ValidationCell, journeyID string, profile domainv1.ValidationEnvironmentProfile) bool {
	for _, cellID := range failed {
		for _, cell := range observed {
			if cell.GetCellId() == cellID && cell.GetJourneyId() == journeyID && cell.GetEnvironmentProfile() == profile {
				return true
			}
		}
	}
	return false
}

func expectedRequiredCells(matrix *domainv1.ValidationMatrix) (map[string]struct{}, bool, error) {
	if matrix == nil {
		return nil, false, nil
	}
	journeys, targets, profiles := matrix.GetJourneys(), matrix.GetTargets(), matrix.GetEnvironmentProfiles()
	if len(journeys) == 0 && len(targets) == 0 && len(profiles) == 0 {
		// Preserve the legacy evidence-only matrix contract used by older
		// persisted runs that predate catalog snapshots.
		return nil, false, nil
	}
	if len(journeys) == 0 || len(targets) == 0 || len(profiles) == 0 {
		return nil, true, fmt.Errorf("validation matrix selection snapshot is incomplete")
	}
	seenJourneys := make(map[string]struct{}, len(journeys))
	for _, journey := range journeys {
		if journey == nil || strings.TrimSpace(journey.GetJourneyId()) == "" {
			return nil, true, fmt.Errorf("validation matrix contains a journey without an identity")
		}
		if _, duplicate := seenJourneys[journey.GetJourneyId()]; duplicate {
			return nil, true, fmt.Errorf("validation matrix contains duplicate journey %q", journey.GetJourneyId())
		}
		seenJourneys[journey.GetJourneyId()] = struct{}{}
	}
	seenTargets := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if target == nil || strings.TrimSpace(target.GetTargetId()) == "" {
			return nil, true, fmt.Errorf("validation matrix contains a target without an identity")
		}
		if _, duplicate := seenTargets[target.GetTargetId()]; duplicate {
			return nil, true, fmt.Errorf("validation matrix contains duplicate target %q", target.GetTargetId())
		}
		seenTargets[target.GetTargetId()] = struct{}{}
	}
	expected := make(map[string]struct{}, len(journeys)*len(targets)*len(profiles))
	for _, journey := range journeys {
		if !journey.GetRequired() {
			continue
		}
		for _, target := range targets {
			for _, profile := range profiles {
				expected[selectionKey(journey.GetJourneyId(), target.GetTargetId(), profile)] = struct{}{}
			}
		}
	}
	return expected, true, nil
}

func cellKey(cell *domainv1.ValidationCell) string {
	if cell == nil || strings.TrimSpace(cell.GetJourneyId()) == "" || strings.TrimSpace(cell.GetTargetId()) == "" {
		return ""
	}
	return selectionKey(cell.GetJourneyId(), cell.GetTargetId(), cell.GetEnvironmentProfile())
}

func selectionKey(journeyID, targetID string, profile domainv1.ValidationEnvironmentProfile) string {
	return fmt.Sprintf("journey=%s target=%s profile=%s", journeyID, targetID, profile.String())
}

func completeEvidence(matrix *domainv1.ValidationMatrix, cell *domainv1.ValidationCell) bool {
	if len(cell.GetEvidence()) == 0 {
		return false
	}
	structured := len(matrix.GetJourneys()) > 0
	required := map[domainv1.LayeredEvidence_Kind]bool{}
	if structured {
		required = map[domainv1.LayeredEvidence_Kind]bool{
			domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME:   true,
			domainv1.LayeredEvidence_KIND_TARGET:            true,
			domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION: true,
		}
	}
	// Scenario journeys require a provider-owned semantic workflow artifact.
	// A platform-only journey may omit it, but still needs desktop, target, and
	// machine evidence. A missing catalog record is treated conservatively as a
	// scenario journey.
	needsWorkflow := structured
	for _, journey := range matrix.GetJourneys() {
		if journey.GetJourneyId() == cell.GetJourneyId() {
			needsWorkflow = !strings.EqualFold(strings.TrimSpace(journey.GetExecutionMode()), "platform")
			break
		}
	}
	if needsWorkflow {
		required[domainv1.LayeredEvidence_KIND_BAS_WORKFLOW] = true
	}
	for _, evidence := range cell.GetEvidence() {
		if evidence == nil || strings.TrimSpace(evidence.GetEvidenceId()) == "" || strings.TrimSpace(evidence.GetUri()) == "" || strings.TrimSpace(evidence.GetSha256()) == "" || !evidence.GetRedacted() {
			return false
		}
		if structured {
			delete(required, evidence.GetKind())
		}
	}
	return !structured || len(required) == 0
}

func stringPtr(value string) *string { return &value }
