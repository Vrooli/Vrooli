package runhistory

import (
	"bytes"
	"encoding/json"

	"unit-health/internal/evidence"
)

// ComparisonIdentity reuses the evidence owner's canonical input identity.
// Selection is the actual command/test selection, not merely a display label.
// Empty TestID denotes command-level evidence. Optional seed and retry ordinal
// distinguish an unsupplied value from an explicitly supplied zero/empty value.
type ComparisonIdentity struct {
	Evidence     evidence.Key `json:"evidence"`
	Selection    string       `json:"selection"`
	TestID       string       `json:"test_id,omitempty"`
	Seed         *string      `json:"seed,omitempty"`
	RetryOrdinal *int         `json:"retry_ordinal,omitempty"`
}

// Known rejects incomplete, corrupt, or future identities. An outcome can still
// be retained when its identity is unknown, but it cannot establish variation.
func (i *ComparisonIdentity) Known() bool {
	if i == nil || i.Selection == "" || (i.RetryOrdinal != nil && *i.RetryOrdinal < 0) {
		return false
	}
	var input evidence.KeyInput
	if json.Unmarshal(i.Evidence.Canonical, &input) != nil || input.SchemaVersion != "unit-health.evidence.v1" {
		return false
	}
	if input.SourceDigest == "" || input.ConfigDigest == "" || input.DependencyLockDigest == "" ||
		input.ToolchainIdentity == "" || input.AdapterID == "" || input.AdapterVersion == "" ||
		input.OS == "" || input.Architecture == "" || input.CoverageMode == "" || input.ArtifactSchema == "" {
		return false
	}
	key, err := evidence.NewKey(input)
	return err == nil && key.Digest == i.Evidence.Digest && bytes.Equal(key.Canonical, i.Evidence.Canonical)
}

// Comparable is deliberately stricter than equal command names. Different
// source/configuration/toolchain, selection, seeds, or retry positions never
// share a cohort. Two unknown identities are not comparable to one another.
func (i *ComparisonIdentity) Comparable(other *ComparisonIdentity) bool {
	return i.Known() && other.Known() && i.Evidence.Digest == other.Evidence.Digest &&
		bytes.Equal(i.Evidence.Canonical, other.Evidence.Canonical) && i.Selection == other.Selection &&
		i.TestID == other.TestID && sameOptional(i.Seed, other.Seed) && sameOptional(i.RetryOrdinal, other.RetryOrdinal)
}

// CohortDigest labels the full comparison scope, not merely the source key.
// The evidence owner remains the only canonicalization/digest implementation.
func (i *ComparisonIdentity) CohortDigest() string {
	if !i.Known() {
		return ""
	}
	var input evidence.KeyInput
	if json.Unmarshal(i.Evidence.Canonical, &input) != nil {
		return ""
	}
	selection, err := json.Marshal(struct {
		Selection    string
		TestID       string
		Seed         *string
		RetryOrdinal *int
	}{i.Selection, i.TestID, i.Seed, i.RetryOrdinal})
	if err != nil {
		return ""
	}
	if input.Environment == nil {
		input.Environment = map[string]string{}
	}
	input.Environment["unit_health_history_comparison_v1"] = string(selection)
	key, err := evidence.NewKey(input)
	if err != nil {
		return ""
	}
	return key.Digest
}

func sameOptional[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
