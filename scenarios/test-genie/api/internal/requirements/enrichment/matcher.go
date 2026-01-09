package enrichment

import (
	"path/filepath"
	"strings"

	"test-genie/internal/requirements/types"
)

// Matcher matches validations to evidence records.
type Matcher struct {
	// Configuration options can be added here
}

// NewMatcher creates a new Matcher.
func NewMatcher() *Matcher {
	return &Matcher{}
}

// FindBestMatch finds the best matching evidence record for a validation.
func (m *Matcher) FindBestMatch(val *types.Validation, reqID string, evidence *types.EvidenceBundle) *types.EvidenceRecord {
	if val == nil || evidence == nil {
		return nil
	}

	var candidates []types.EvidenceRecord

	// Strategy 1: Match by validation ref
	if val.Ref != "" {
		candidates = append(candidates, m.matchByRef(val.Ref, evidence)...)
	}

	// Strategy 2: Match by workflow ID
	if val.WorkflowID != "" {
		candidates = append(candidates, m.matchByWorkflowID(val.WorkflowID, evidence)...)
	}

	// Strategy 3: Match by phase and requirement ID
	if val.Phase != "" {
		candidates = append(candidates, m.matchByPhase(val.Phase, reqID, evidence)...)
	}

	// Strategy 4: Match by requirement ID directly
	candidates = append(candidates, m.matchByRequirementID(reqID, evidence)...)

	// Strategy 5: Check manual validations
	if val.Type == types.ValTypeManual && evidence.ManualValidations != nil {
		if manual, ok := evidence.ManualValidations.Get(reqID); ok && manual.IsValid() {
			candidates = append(candidates, types.EvidenceRecord{
				RequirementID: reqID,
				Status:        manual.ToLiveStatus(),
				Phase:         "manual",
				Evidence:      manual.Notes,
				UpdatedAt:     manual.ValidatedAt,
				SourcePath:    manual.ArtifactPath,
			})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Select best candidate based on priority
	return m.selectBest(candidates)
}

// FindDirectEvidence finds evidence directly mapped to a requirement ID.
func (m *Matcher) FindDirectEvidence(reqID string, evidence *types.EvidenceBundle) []types.EvidenceRecord {
	if evidence == nil {
		return nil
	}

	var results []types.EvidenceRecord

	// Check phase results
	if records := evidence.PhaseResults.Get(reqID); len(records) > 0 {
		results = append(results, records...)
	}

	// Check vitest evidence
	if vitestResults, ok := evidence.VitestEvidence[reqID]; ok {
		for _, vr := range vitestResults {
			results = append(results, types.EvidenceRecord{
				RequirementID: reqID,
				Status:        vr.ToLiveStatus(),
				Phase:         "unit",
				SourcePath:    vr.FilePath,
			})
		}
	}

	// Check manual validations
	if evidence.ManualValidations != nil {
		if manual, ok := evidence.ManualValidations.Get(reqID); ok && manual.IsValid() {
			results = append(results, types.EvidenceRecord{
				RequirementID: reqID,
				Status:        manual.ToLiveStatus(),
				Phase:         "manual",
				Evidence:      manual.Notes,
				UpdatedAt:     manual.ValidatedAt,
			})
		}
	}

	return results
}

// matchByRef finds evidence matching a validation ref path.
func (m *Matcher) matchByRef(ref string, evidence *types.EvidenceBundle) []types.EvidenceRecord {
	var results []types.EvidenceRecord
	normalizedRef := normalizeRef(ref)

	// Search in phase results
	for _, records := range evidence.PhaseResults {
		for _, record := range records {
			if normalizeRef(record.ValidationRef) == normalizedRef ||
				normalizeRef(record.SourcePath) == normalizedRef ||
				pathContains(record.SourcePath, ref) {
				results = append(results, record)
			}
		}
	}

	// Search in vitest evidence
	for _, vitestResults := range evidence.VitestEvidence {
		for _, vr := range vitestResults {
			if normalizeRef(vr.FilePath) == normalizedRef ||
				pathContains(vr.FilePath, ref) {
				results = append(results, types.EvidenceRecord{
					RequirementID: vr.RequirementID,
					Status:        vr.ToLiveStatus(),
					Phase:         "unit",
					SourcePath:    vr.FilePath,
				})
			}
		}
	}

	return results
}

// matchByWorkflowID finds evidence matching a workflow ID.
func (m *Matcher) matchByWorkflowID(workflowID string, evidence *types.EvidenceBundle) []types.EvidenceRecord {
	var results []types.EvidenceRecord
	normalizedID := strings.ToLower(strings.TrimSpace(workflowID))

	// Search in phase results (playbooks phase typically)
	for _, records := range evidence.PhaseResults {
		for _, record := range records {
			if record.Phase == "playbooks" || record.Phase == "automation" {
				// Check if metadata contains workflow ID
				if meta, ok := record.Metadata["workflow_id"].(string); ok {
					if strings.ToLower(meta) == normalizedID {
						results = append(results, record)
					}
				}
				// Check if source path contains workflow ID
				if strings.Contains(strings.ToLower(record.SourcePath), normalizedID) {
					results = append(results, record)
				}
			}
		}
	}

	return results
}

// matchByPhase finds evidence matching a specific phase.
func (m *Matcher) matchByPhase(phase, reqID string, evidence *types.EvidenceBundle) []types.EvidenceRecord {
	var results []types.EvidenceRecord
	normalizedPhase := strings.ToLower(strings.TrimSpace(phase))

	// Direct phase match in evidence map
	for _, records := range evidence.PhaseResults {
		for _, record := range records {
			if strings.ToLower(record.Phase) == normalizedPhase {
				// If requirement ID matches or is unset, include it
				if record.RequirementID == reqID || record.RequirementID == "" {
					results = append(results, record)
				}
			}
		}
	}

	// Check phase-level results
	phaseKey := "__phase__" + normalizedPhase
	if records := evidence.PhaseResults.Get(phaseKey); len(records) > 0 {
		results = append(results, records...)
	}

	return results
}

// matchByRequirementID finds evidence directly mapped to a requirement ID.
func (m *Matcher) matchByRequirementID(reqID string, evidence *types.EvidenceBundle) []types.EvidenceRecord {
	var results []types.EvidenceRecord
	normalizedID := strings.ToUpper(strings.TrimSpace(reqID))

	// Search in phase results
	for _, records := range evidence.PhaseResults {
		for _, record := range records {
			if strings.ToUpper(record.RequirementID) == normalizedID {
				results = append(results, record)
			}
		}
	}

	// Search in vitest evidence
	for id, vitestResults := range evidence.VitestEvidence {
		if strings.ToUpper(id) == normalizedID {
			for _, vr := range vitestResults {
				results = append(results, types.EvidenceRecord{
					RequirementID: reqID,
					Status:        vr.ToLiveStatus(),
					Phase:         "unit",
					SourcePath:    vr.FilePath,
				})
			}
		}
	}

	return results
}

// selectBest selects the best candidate from a list of evidence records.
func (m *Matcher) selectBest(candidates []types.EvidenceRecord) *types.EvidenceRecord {
	if len(candidates) == 0 {
		return nil
	}

	// Priority order: failed > skipped > passed > not_run > unknown
	// Also prefer more recent evidence
	best := candidates[0]

	for _, c := range candidates[1:] {
		// If current has higher priority status, use it
		if statusPriority(c.Status) > statusPriority(best.Status) {
			best = c
			continue
		}

		// If same priority but more recent, use it
		if statusPriority(c.Status) == statusPriority(best.Status) &&
			c.UpdatedAt.After(best.UpdatedAt) {
			best = c
		}
	}

	return &best
}

// statusPriority returns priority for status (higher = more important).
func statusPriority(s types.LiveStatus) int {
	switch s {
	case types.LiveFailed:
		return 4
	case types.LiveSkipped:
		return 3
	case types.LivePassed:
		return 2
	case types.LiveNotRun:
		return 1
	default:
		return 0
	}
}

// normalizeRef normalizes a file reference for comparison.
func normalizeRef(ref string) string {
	// Convert backslashes to forward slashes
	ref = strings.ReplaceAll(ref, "\\", "/")
	// Remove leading ./
	ref = strings.TrimPrefix(ref, "./")
	// Lowercase for case-insensitive comparison
	return strings.ToLower(strings.TrimSpace(ref))
}

// pathContains checks if a path contains a reference pattern.
func pathContains(fullPath, pattern string) bool {
	normalizedPath := normalizeRef(fullPath)
	normalizedPattern := normalizeRef(pattern)

	// Direct contains check
	if strings.Contains(normalizedPath, normalizedPattern) {
		return true
	}

	// Check just the filename
	pathBase := filepath.Base(normalizedPath)
	patternBase := filepath.Base(normalizedPattern)
	if pathBase == patternBase {
		return true
	}

	return false
}
