package testquality

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// Candidate is the bounded, metadata-only representation used by sampled review.
// Source bodies are intentionally absent so a cohort can be persisted safely.
type Candidate struct {
	TestID         string   `json:"testId"`
	File           string   `json:"file"`
	Workspace      string   `json:"workspace"`
	Framework      string   `json:"framework"`
	TestKind       string   `json:"testKind"`
	StaticStatus   Status   `json:"staticStatus"`
	RuntimeStatus  string   `json:"runtimeStatus,omitempty"`
	Severity       Severity `json:"severity"`
	SourceDigest   string   `json:"sourceDigest"`
	RequirementIDs []string `json:"requirementIds,omitempty"`
}

func (c Candidate) Identity() string {
	return fmt.Sprintf("%s:%s:%s:%s", c.Workspace, c.File, c.TestID, c.Framework)
}

type Exclusion struct {
	Identity string `json:"identity"`
	Reason   string `json:"reason"`
}

type Cohort struct {
	SchemaVersion  string      `json:"schemaVersion"`
	PolicyVersion  string      `json:"policyVersion"`
	Seed           string      `json:"seed"`
	SourceIdentity string      `json:"sourceIdentity"`
	Denominator    int         `json:"denominator"`
	Candidates     []Candidate `json:"candidates"`
	Selected       []Candidate `json:"selected"`
	Excluded       []Exclusion `json:"excluded"`
	Controls       int         `json:"controls"`
	Truncated      bool        `json:"truncated"`
	Unavailable    bool        `json:"unavailable"`
}

const SelectionPolicyVersion = "test-quality-stratified/v1"

// Select deterministically selects a bounded cohort. Findings and unknowns get
// priority, while clean controls are selected from the same framework/kind.
func Select(candidates []Candidate, sampleSize int, seed, sourceIdentity string, includeControls bool) (Cohort, error) {
	if sampleSize < 1 || sampleSize > 256 {
		return Cohort{}, fmt.Errorf("sample size must be in [1,256]")
	}
	if seed == "" || sourceIdentity == "" {
		return Cohort{}, fmt.Errorf("seed and source identity are required")
	}
	ordered := append([]Candidate(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Identity() < ordered[j].Identity() })
	cohort := Cohort{SchemaVersion: "test-quality-cohort/v1", PolicyVersion: SelectionPolicyVersion, Seed: seed, SourceIdentity: sourceIdentity, Denominator: len(ordered), Candidates: ordered, Selected: []Candidate{}, Excluded: []Exclusion{}}
	if len(ordered) == 0 {
		cohort.Unavailable = true
		return cohort, nil
	}

	priority := func(c Candidate) int {
		switch NormalizeStatus(c.StaticStatus) {
		case Violation:
			return 0
		case Unknown:
			return 1
		default:
			if c.RuntimeStatus == "unknown" {
				return 1
			}
			return 2
		}
	}
	score := func(c Candidate) string {
		h := sha256.Sum256([]byte(seed + "\x00" + c.Identity()))
		return hex.EncodeToString(h[:])
	}
	// Stable ordering within each stratum makes reselection reproducible.
	sort.SliceStable(ordered, func(i, j int) bool {
		pi, pj := priority(ordered[i]), priority(ordered[j])
		if pi != pj {
			return pi < pj
		}
		return score(ordered[i]) < score(ordered[j])
	})
	selected := map[string]bool{}
	add := func(c Candidate) {
		if len(cohort.Selected) < sampleSize && !selected[c.Identity()] {
			selected[c.Identity()] = true
			cohort.Selected = append(cohort.Selected, c)
		}
	}
	for _, c := range ordered {
		if priority(c) < 2 {
			add(c)
		}
	}
	if includeControls && len(cohort.Selected) < sampleSize {
		for _, c := range ordered {
			if priority(c) == 2 {
				add(c)
				cohort.Controls++
			}
		}
	}
	for _, c := range ordered {
		if len(cohort.Selected) >= sampleSize {
			break
		}
		add(c)
	}
	for _, c := range ordered {
		if !selected[c.Identity()] {
			cohort.Excluded = append(cohort.Excluded, Exclusion{Identity: c.Identity(), Reason: "sample-cap"})
		}
	}
	sort.Slice(cohort.Selected, func(i, j int) bool { return cohort.Selected[i].Identity() < cohort.Selected[j].Identity() })
	sort.Slice(cohort.Excluded, func(i, j int) bool { return cohort.Excluded[i].Identity < cohort.Excluded[j].Identity })
	cohort.Truncated = len(cohort.Excluded) > 0
	return cohort, nil
}
