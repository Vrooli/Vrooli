package findings

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/vrooli/maturity-go/dimensions"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// Finding is one open problem the controller can act on, already resolved to a
// canonical dimension.
type Finding struct {
	ID        string                         `json:"id"`
	Dimension dimensions.Dimension           `json:"dimension"`
	Severity  architecturev1.FindingSeverity `json:"severity"`
	Location  string                         `json:"location,omitempty"`
	Code      string                         `json:"code,omitempty"`
	Message   string                         `json:"message,omitempty"`
	Phase     string                         `json:"phase,omitempty"`
	// Synthetic marks a finding manufactured from a failing-but-findingless
	// phase rather than a structured test-genie finding.
	Synthetic bool `json:"synthetic,omitempty"`
}

// SeverityWeight is the per-finding contribution to a dimension's weighted
// score. Severities escalate by doubling so a single blocker outranks a cluster
// of low-severity nits. UNSPECIFIED still counts (weight 1) so it is never
// silently dropped.
func SeverityWeight(s architecturev1.FindingSeverity) float64 {
	switch s {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return 8
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return 4
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return 2
	default: // INFO and UNSPECIFIED
		return 1
	}
}

// FindingsState is the controller's diagnose-stage state: the open findings
// bucketed by dimension with weighted scores, plus a fingerprint of the open
// set (computed now for the decision trace and to seed v1's cycle detection;
// v0 does no cycle-halting).
type FindingsState struct {
	Findings       []Finding                        `json:"findings"`
	DimensionScore map[dimensions.Dimension]float64 `json:"dimensionScore"`
	DimensionCount map[dimensions.Dimension]int     `json:"dimensionCount"`
	TotalScore     float64                          `json:"totalScore"`
	Fingerprint    string                           `json:"fingerprint"`
}

// BuildState aggregates findings into a FindingsState.
func BuildState(found []Finding) FindingsState {
	st := FindingsState{
		Findings:       found,
		DimensionScore: make(map[dimensions.Dimension]float64),
		DimensionCount: make(map[dimensions.Dimension]int),
	}
	ids := make([]string, 0, len(found))
	for _, f := range found {
		w := SeverityWeight(f.Severity)
		st.DimensionScore[f.Dimension] += w
		st.DimensionCount[f.Dimension]++
		st.TotalScore += w
		ids = append(ids, f.ID)
	}
	st.Fingerprint = fingerprint(ids)
	return st
}

// HeaviestDimensions returns dimensions ordered by weighted score descending,
// with a deterministic alphabetical tiebreak. Only dimensions with a positive
// score are returned.
func (s FindingsState) HeaviestDimensions() []dimensions.Dimension {
	dims := make([]dimensions.Dimension, 0, len(s.DimensionScore))
	for d, score := range s.DimensionScore {
		if score > 0 {
			dims = append(dims, d)
		}
	}
	sort.Slice(dims, func(i, j int) bool {
		si, sj := s.DimensionScore[dims[i]], s.DimensionScore[dims[j]]
		if si != sj {
			return si > sj
		}
		return dims[i] < dims[j]
	})
	return dims
}

// fingerprint hashes the sorted finding-id set so an identical open set yields
// an identical fingerprint regardless of audit ordering.
func fingerprint(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	h := sha256.New()
	for _, id := range sorted {
		h.Write([]byte(id))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
