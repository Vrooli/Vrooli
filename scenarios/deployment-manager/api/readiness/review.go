package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReviewStatus string

const (
	ReviewCollecting  ReviewStatus = "collecting"
	ReviewBlocked     ReviewStatus = "blocked"
	ReviewAgentReview ReviewStatus = "agent_review"
	ReviewApproved    ReviewStatus = "approved"
	ReviewPromoted    ReviewStatus = "promoted"
	ReviewSuperseded  ReviewStatus = "superseded"
)

type ComparisonMode string

const (
	ComparisonFirstRelease ComparisonMode = "first_release"
	ComparisonComparable   ComparisonMode = "comparable"
	ComparisonUnavailable  ComparisonMode = "comparison_unavailable"
)

type ReviewIdentity struct {
	Scenario        string   `json:"scenario"`
	ProfileID       string   `json:"profile_id"`
	CandidateCommit string   `json:"candidate_commit"`
	ArtifactDigest  string   `json:"artifact_digest"`
	Targets         []string `json:"targets"`
	Channel         string   `json:"channel"`
	PolicyVersion   int      `json:"policy_version"`
}

func (i ReviewIdentity) Canonical() (ReviewIdentity, error) {
	i.Scenario = strings.TrimSpace(i.Scenario)
	i.ProfileID = strings.TrimSpace(i.ProfileID)
	i.CandidateCommit = strings.TrimSpace(i.CandidateCommit)
	i.ArtifactDigest = strings.TrimSpace(i.ArtifactDigest)
	i.Channel = strings.TrimSpace(i.Channel)
	if i.Scenario == "" || i.ProfileID == "" || i.CandidateCommit == "" || i.ArtifactDigest == "" || i.Channel == "" || i.PolicyVersion <= 0 {
		return ReviewIdentity{}, fmt.Errorf("review identity requires scenario, profile, candidate commit, artifact digest, channel, and policy version")
	}
	targets := make([]string, 0, len(i.Targets))
	seen := map[string]struct{}{}
	for _, target := range i.Targets {
		target = strings.TrimSpace(target)
		if target == "" {
			return ReviewIdentity{}, fmt.Errorf("review identity contains an empty target")
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return ReviewIdentity{}, fmt.Errorf("review identity requires at least one target")
	}
	sort.Strings(targets)
	i.Targets = targets
	return i, nil
}

func (i ReviewIdentity) Key() (string, error) {
	canonical, err := i.Canonical()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "rr-" + hex.EncodeToString(sum[:]), nil
}

type Review struct {
	Key                       string         `json:"key"`
	Identity                  ReviewIdentity `json:"identity"`
	Status                    ReviewStatus   `json:"status"`
	ComparisonMode            ComparisonMode `json:"comparison_mode"`
	PredecessorReleaseID      string         `json:"predecessor_release_id,omitempty"`
	PredecessorCommit         string         `json:"predecessor_commit,omitempty"`
	PredecessorArtifactDigest string         `json:"predecessor_artifact_digest,omitempty"`
	GoalRef                   string         `json:"goal_ref,omitempty"`
	GoalClosedAt              *time.Time     `json:"goal_closed_at,omitempty"`
	ApprovedAt                *time.Time     `json:"approved_at,omitempty"`
	ApprovedBy                string         `json:"approved_by,omitempty"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
}

type EvidenceItem struct {
	ReviewKey           string       `json:"review_key"`
	CriterionID         string       `json:"criterion_id"`
	Status              SignalStatus `json:"status"`
	Applicability       string       `json:"applicability"`
	ApplicabilityReason string       `json:"applicability_reason,omitempty"`
	Producer            string       `json:"producer"`
	ProducerVersion     string       `json:"producer_version,omitempty"`
	CandidateCommit     string       `json:"candidate_commit"`
	ArtifactDigest      string       `json:"artifact_digest"`
	Target              string       `json:"target"`
	Environment         string       `json:"environment"`
	PolicyVersion       int          `json:"policy_version"`
	ObservedAt          time.Time    `json:"observed_at"`
	Reference           string       `json:"reference"`
	Detail              string       `json:"detail,omitempty"`
}

type ReviewFinding struct {
	ReviewKey   string       `json:"review_key"`
	CriterionID string       `json:"criterion_id"`
	Severity    string       `json:"severity"`
	Status      SignalStatus `json:"status"`
	Message     string       `json:"message"`
}

type ReviewWaiver struct {
	ReviewKey   string    `json:"review_key"`
	CriterionID string    `json:"criterion_id"`
	Actor       string    `json:"actor"`
	Reason      string    `json:"reason"`
	ExpiresAt   time.Time `json:"expires_at"`
	Trigger     string    `json:"trigger,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type EvidenceObservation struct {
	Identity        ReviewIdentity
	CriterionID     string
	ProducerBinding string
	Evidence        EvidenceItem
}

type HumanCheck struct {
	ReviewKey         string
	CriterionID       string
	Verdict           string
	Actor             string
	EvidenceReference string
	ReviewedAt        time.Time
}

type ReviewRepository interface {
	CreateOrGet(context.Context, *Review) (bool, error)
	Get(context.Context, string) (*Review, error)
	ListReviews(context.Context, ReviewStatus, int) ([]Review, error)
	ListEvaluation(context.Context, string) ([]EvidenceItem, []ReviewFinding, error)
	ListActiveWaivers(context.Context, string, time.Time) ([]ReviewWaiver, error)
	ListWaivers(context.Context, string, int) ([]ReviewWaiver, error)
	ReplaceEvaluation(context.Context, string, []EvidenceItem, []ReviewFinding, ReviewStatus) error
	SetGoal(context.Context, string, string) error
	RecordGoalClosure(context.Context, string, time.Time) error
	Approve(context.Context, string, ReviewIdentity, string, time.Time) error
	SaveWaiver(context.Context, ReviewWaiver) error
	SaveObservation(context.Context, EvidenceObservation) error
	FindObservation(context.Context, ReviewIdentity, string, string) (*EvidenceItem, error)
	MarkPromoted(context.Context, string, time.Time) error
	SaveHumanCheck(context.Context, HumanCheck) error
	ListHumanChecks(context.Context, string) ([]HumanCheck, error)
}
