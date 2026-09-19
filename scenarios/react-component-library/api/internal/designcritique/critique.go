// Package designcritique owns attributed visual review facts. A visual rating
// never substitutes for Experience Manager's deterministic behavioral claims.
package designcritique

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const RubricVersion = "visual-design/1"
const PolicyVersion = "visual-floor/1"

var dimensions = []string{"task_clarity", "hierarchy", "navigation_continuity", "responsive_layout", "state_recovery", "accessibility", "terminology_i18n", "visual_consistency"}

func Dimensions() []string { return append([]string(nil), dimensions...) }
func Anchors() []string {
	return []string{"Unusable or absent", "Severe issue", "Material issue", "Acceptable with minor issues", "Strong"}
}

var hashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Target struct {
	Scenario   string `json:"scenario"`
	DesignID   string `json:"designId"`
	Revision   string `json:"revision"`
	RenderHash string `json:"renderHash"`
}
type Critic struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version string `json:"version"`
	Model   string `json:"model,omitempty"`
	Profile string `json:"profile,omitempty"`
}

// State is the critic's observed state label, not proof that a behavioral claim
// or fixture transition has passed. Region "$page" denotes a page-wide finding.
type Evidence struct {
	RenderHash string `json:"renderHash,omitempty"`
	CaptureID  string `json:"captureId"`
	Artifact   string `json:"artifact"`
	Region     string `json:"region"`
	State      string `json:"state"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}
type Rating struct {
	Dimension string     `json:"dimension"`
	Score     int        `json:"score"`
	Rationale string     `json:"rationale"`
	Evidence  []Evidence `json:"evidence"`
}
type Finding struct {
	Dimension  string   `json:"dimension"`
	Severity   string   `json:"severity"`
	Rationale  string   `json:"rationale"`
	Correction string   `json:"correction"`
	Evidence   Evidence `json:"evidence"`
}
type Review struct {
	Target        Target    `json:"target"`
	RubricVersion string    `json:"rubricVersion"`
	PolicyVersion string    `json:"policyVersion"`
	Critic        Critic    `json:"critic"`
	Ratings       []Rating  `json:"ratings"`
	Findings      []Finding `json:"findings"`
}
type Assessment struct {
	VisualFloorMet        bool     `json:"visualFloorMet"`
	MinimumScore          int      `json:"minimumScore"`
	BlockingDimensions    []string `json:"blockingDimensions"`
	BlockingFindings      []int    `json:"blockingFindings"`
	AcceptanceEstablished bool     `json:"acceptanceEstablished"`
}

// EvidenceVerifier must resolve available image evidence through its producer,
// and verify target identity, receipt membership, viewport and semantic region.
// A caller-provided URL or a model's assertion is not verification.
type EvidenceVerifier interface {
	Verify(context.Context, Target, Evidence) error
}

func Evaluate(ctx context.Context, review Review, verifier EvidenceVerifier) (Assessment, error) {
	if err := Validate(review); err != nil {
		return Assessment{}, err
	}
	if verifier == nil {
		return Assessment{}, fmt.Errorf("visual image evidence verifier is required")
	}
	seen := map[Evidence]bool{}
	verify := func(e Evidence) error {
		if seen[e] {
			return nil
		}
		if err := verifier.Verify(ctx, review.Target, e); err != nil {
			return fmt.Errorf("image evidence unavailable or mismatched: %w", err)
		}
		seen[e] = true
		return nil
	}
	for _, rating := range review.Ratings {
		for _, e := range rating.Evidence {
			if err := verify(e); err != nil {
				return Assessment{}, err
			}
		}
	}
	for _, finding := range review.Findings {
		if err := verify(finding.Evidence); err != nil {
			return Assessment{}, err
		}
	}
	return aggregate(review), nil
}
func aggregate(review Review) Assessment {
	result := Assessment{MinimumScore: 4, BlockingDimensions: []string{}, BlockingFindings: []int{}}
	scores := map[string]int{}
	for _, rating := range review.Ratings {
		scores[rating.Dimension] = rating.Score
		result.MinimumScore = min(result.MinimumScore, rating.Score)
	}
	// Rubric order, not input order, determines aggregation output.
	for _, dimension := range dimensions {
		if scores[dimension] < 3 {
			result.BlockingDimensions = append(result.BlockingDimensions, dimension)
		}
	}
	for i, finding := range review.Findings {
		if finding.Severity == "critical" || finding.Severity == "major" {
			result.BlockingFindings = append(result.BlockingFindings, i)
		}
	}
	result.VisualFloorMet = len(result.BlockingDimensions) == 0 && len(result.BlockingFindings) == 0
	// Calibration, independent review, rendered-region completeness and producer
	// behavioral claims are separate gates; this assessment cannot establish them.
	return result
}

func Validate(r Review) error {
	if !bounded(r.Target.Scenario, 128) || !bounded(r.Target.DesignID, 128) || !hashPattern.MatchString(r.Target.Revision) || !hashPattern.MatchString(r.Target.RenderHash) {
		return fmt.Errorf("exact design and render identity are required")
	}
	if r.RubricVersion != RubricVersion || r.PolicyVersion != PolicyVersion {
		return fmt.Errorf("unsupported rubric or visual policy version")
	}
	if !bounded(r.Critic.ID, 200) || !bounded(r.Critic.Version, 200) || (r.Critic.Kind != "human" && r.Critic.Kind != "model") {
		return fmt.Errorf("critic identity, kind and version are required")
	}
	if r.Critic.Kind == "model" && (!bounded(r.Critic.Model, 200) || !bounded(r.Critic.Profile, 200)) {
		return fmt.Errorf("model critic requires model and profile identity")
	}
	if len(r.Critic.Model) > 200 || len(r.Critic.Profile) > 200 {
		return fmt.Errorf("critic metadata exceeds limits")
	}
	if len(r.Ratings) != len(dimensions) || len(r.Findings) > 128 {
		return fmt.Errorf("all rubric dimensions and bounded findings are required")
	}
	allowed := map[string]bool{}
	seen := map[string]bool{}
	for _, d := range dimensions {
		allowed[d] = true
	}
	for _, rating := range r.Ratings {
		if !allowed[rating.Dimension] || seen[rating.Dimension] || rating.Score < 0 || rating.Score > 4 || !bounded(rating.Rationale, 4000) || len(rating.Evidence) < 1 || len(rating.Evidence) > 16 {
			return fmt.Errorf("invalid, duplicate or unsupported rating")
		}
		seen[rating.Dimension] = true
		for _, e := range rating.Evidence {
			if err := validateEvidence(e); err != nil {
				return err
			}
		}
	}
	for _, f := range r.Findings {
		if !allowed[f.Dimension] || (f.Severity != "critical" && f.Severity != "major" && f.Severity != "minor" && f.Severity != "informational") || !bounded(f.Rationale, 4000) || !bounded(f.Correction, 4000) {
			return fmt.Errorf("finding requires dimension, severity, rationale and correction")
		}
		if err := validateEvidence(f.Evidence); err != nil {
			return err
		}
	}
	return nil
}
func validateEvidence(e Evidence) error {
	if e.RenderHash != "" && !hashPattern.MatchString(e.RenderHash) {
		return fmt.Errorf("evidence render identity must be an exact hash")
	}
	if !bounded(e.CaptureID, 200) || !bounded(e.Artifact, 2048) || !bounded(e.Region, 128) || !bounded(e.State, 200) || e.Width < 100 || e.Width > 4000 || e.Height < 100 || e.Height > 4000 {
		return fmt.Errorf("evidence requires capture, artifact, region, observed state and viewport")
	}
	return nil
}
func bounded(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= limit
}
