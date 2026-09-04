package readiness

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type EvidenceProducer interface {
	Collect(context.Context, ReviewIdentity, Item) (EvidenceItem, error)
}

// ObservationProducer is the typed boundary between readiness orchestration
// and evidence owners. Owners report observations through the API; prepare
// reads only the observation for the exact identity, criterion, and declared
// policy binding.
type ObservationProducer struct {
	Repository ReviewRepository
	Binding    string
}

func (p ObservationProducer) Collect(ctx context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if p.Repository == nil || p.Binding == "" || criterion.ProducerBinding() != p.Binding {
		return EvidenceItem{}, errors.New("evidence observation producer is not configured for the policy binding")
	}
	item, err := p.Repository.FindObservation(ctx, identity, criterion.ID, p.Binding)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("read %s observation: %w", p.Binding, err)
	}
	return *item, nil
}

func ObservationProducers(policy Checklist, repository ReviewRepository) map[string]EvidenceProducer {
	result := make(map[string]EvidenceProducer)
	for _, criterion := range policy.Items {
		binding := criterion.ProducerBinding()
		if binding != "" {
			result[binding] = ObservationProducer{Repository: repository, Binding: binding}
		}
	}
	return result
}

type Predecessor struct {
	ReleaseID      string `json:"release_id"`
	Commit         string `json:"commit"`
	ArtifactDigest string `json:"artifact_digest"`
	PolicyVersion  int    `json:"policy_version"`
}

type PredecessorResolver interface {
	LatestDeployed(context.Context, ReviewIdentity) (*Predecessor, error)
}

type PrepareRequest struct {
	Identity         ReviewIdentity    `json:"identity"`
	ProvidedEvidence []EvidenceItem    `json:"evidence,omitempty"`
	Facts            map[string]string `json:"facts,omitempty"`
	Deliverable      string            `json:"deliverable,omitempty"`
	Trigger          string            `json:"trigger,omitempty"`
}

type PrepareDecision struct {
	Review  Review   `json:"review"`
	Verdict Verdict  `json:"verdict"`
	GoalRef string   `json:"goal_ref,omitempty"`
	Deduped bool     `json:"deduped"`
	Next    []string `json:"next_actions,omitempty"`
}

type Preparer struct {
	Policy      Checklist
	Repository  ReviewRepository
	Goals       goalOpener
	Predecessor PredecessorResolver
	Producers   map[string]EvidenceProducer
	Now         func() time.Time
}

func (p *Preparer) Prepare(ctx context.Context, request PrepareRequest) (PrepareDecision, error) {
	if p == nil || p.Repository == nil {
		return PrepareDecision{}, errors.New("readiness preparer is not configured")
	}
	policy := p.Policy
	if len(policy.Items) == 0 {
		policy = DefaultChecklist()
	}
	if err := policy.Validate(); err != nil {
		return PrepareDecision{}, err
	}
	identity, err := request.Identity.Canonical()
	if err != nil {
		return PrepareDecision{}, err
	}
	if identity.PolicyVersion != policy.Version {
		return PrepareDecision{}, fmt.Errorf("candidate policy version %d does not match active version %d", identity.PolicyVersion, policy.Version)
	}
	now := time.Now().UTC()
	if p.Now != nil {
		now = p.Now().UTC()
	}
	review := Review{Identity: identity, Status: ReviewCollecting, ComparisonMode: ComparisonFirstRelease}
	if p.Predecessor != nil {
		predecessor, resolveErr := p.Predecessor.LatestDeployed(ctx, identity)
		switch {
		case resolveErr != nil:
			review.ComparisonMode = ComparisonUnavailable
		case predecessor == nil:
			review.ComparisonMode = ComparisonFirstRelease
		case predecessor.Commit == "" || predecessor.ArtifactDigest == "" || predecessor.PolicyVersion <= 0:
			review.ComparisonMode = ComparisonUnavailable
			review.PredecessorReleaseID, review.PredecessorCommit, review.PredecessorArtifactDigest = predecessor.ReleaseID, predecessor.Commit, predecessor.ArtifactDigest
		default:
			review.ComparisonMode = ComparisonComparable
			review.PredecessorReleaseID, review.PredecessorCommit, review.PredecessorArtifactDigest = predecessor.ReleaseID, predecessor.Commit, predecessor.ArtifactDigest
		}
	}
	deduped, err := p.Repository.CreateOrGet(ctx, &review)
	if err != nil {
		return PrepareDecision{}, err
	}
	provided := make(map[string]EvidenceItem, len(request.ProvidedEvidence))
	for _, evidence := range request.ProvidedEvidence {
		if _, exists := provided[evidence.CriterionID]; exists {
			return PrepareDecision{}, fmt.Errorf("duplicate provided evidence for %q", evidence.CriterionID)
		}
		provided[evidence.CriterionID] = evidence
	}
	activeWaivers, err := p.Repository.ListActiveWaivers(ctx, review.Key, now)
	if err != nil {
		return PrepareDecision{}, err
	}
	waivers := make(map[string]ReviewWaiver, len(activeWaivers))
	for _, waiver := range activeWaivers {
		waivers[waiver.CriterionID] = waiver
	}
	target := strings.Join(identity.Targets, ",")
	evidence := make([]EvidenceItem, 0, len(policy.Items))
	signals := make([]Signal, 0, len(policy.Items))
	for _, criterion := range policy.Items {
		applicability, reason := resolveApplicability(criterion.Applicability, request.Facts, review.ComparisonMode)
		if applicability == "not_applicable" {
			item := notApplicableEvidence(review.Key, identity, criterion, target, now, reason)
			evidence = append(evidence, item)
			signals = append(signals, Signal{ItemID: criterion.ID, Status: item.Status, Source: item.Producer, Commit: item.CandidateCommit, ObservedAt: item.ObservedAt, ArtifactDigest: item.ArtifactDigest, Target: item.Target, Environment: item.Environment, PolicyVersion: item.PolicyVersion, Reference: item.Reference, Applicability: item.Applicability, ApplicabilityReason: item.ApplicabilityReason})
			continue
		}
		item, ok := provided[criterion.ID]
		if !ok {
			if producer := p.Producers[criterion.ProducerBinding()]; producer != nil {
				item, err = producer.Collect(ctx, identity, criterion)
				if err != nil {
					item = unavailableEvidence(review.Key, identity, criterion, target, now, err.Error())
				}
			} else if criterion.HumanReview != nil {
				item = pendingHumanEvidence(review.Key, identity, criterion, target, now)
			} else {
				item = unavailableEvidence(review.Key, identity, criterion, target, now, "producer binding is unavailable")
			}
		}
		item.ReviewKey = review.Key
		if item.CriterionID != criterion.ID {
			return PrepareDecision{}, fmt.Errorf("producer returned criterion %q for %q", item.CriterionID, criterion.ID)
		}
		if item.CandidateCommit != identity.CandidateCommit || item.ArtifactDigest != identity.ArtifactDigest || item.PolicyVersion != identity.PolicyVersion {
			item.Status = SignalStale
			item.Detail = "evidence identity does not match the candidate review"
		}
		if item.Target == "" {
			item.Target = target
		}
		if item.Environment == "" {
			item.Environment = identity.Channel
		}
		if item.Applicability == "" {
			item.Applicability = "applicable"
		}
		if item.ObservedAt.IsZero() {
			item.ObservedAt = now
		}
		if item.Reference == "" {
			return PrepareDecision{}, fmt.Errorf("evidence for %q has no external reference", criterion.ID)
		}
		if applicability == "unknown" {
			item.Status = SignalUnknown
			item.Applicability = "unknown"
			item.Detail = reason
		}
		if waiver, ok := waivers[criterion.ID]; ok && criterion.Waiver.Eligible && item.Status != SignalPassed && item.Status != SignalNotApplicable {
			item.Status = SignalWaived
			item.Detail = fmt.Sprintf("waived by %s until %s: %s", waiver.Actor, waiver.ExpiresAt.UTC().Format(time.RFC3339), waiver.Reason)
		}
		evidence = append(evidence, item)
		signals = append(signals, Signal{ItemID: criterion.ID, Status: item.Status, Source: item.Producer, Commit: item.CandidateCommit, ObservedAt: item.ObservedAt, Detail: item.Detail, ProducerVersion: item.ProducerVersion, ArtifactDigest: item.ArtifactDigest, Target: item.Target, Environment: item.Environment, PolicyVersion: item.PolicyVersion, Reference: item.Reference, Applicability: item.Applicability, ApplicabilityReason: item.ApplicabilityReason})
	}
	verdict, err := Aggregate(identity.Scenario, identity.CandidateCommit, policy, signals, now)
	if err != nil {
		return PrepareDecision{}, err
	}
	findings := make([]ReviewFinding, 0, len(verdict.Findings))
	mechanicalBlocker := false
	byID := make(map[string]Item, len(policy.Items))
	for _, item := range policy.Items {
		byID[item.ID] = item
	}
	for _, finding := range verdict.Findings {
		findings = append(findings, ReviewFinding{ReviewKey: review.Key, CriterionID: finding.ItemID, Severity: finding.Severity, Status: finding.Status, Message: finding.Message})
		if finding.Severity == "error" && byID[finding.ItemID].CleanRequirement != Uncheckable {
			mechanicalBlocker = true
		}
	}
	if mechanicalBlocker {
		review.Status = ReviewBlocked
	} else {
		review.Status = ReviewAgentReview
	}
	if err := p.Repository.ReplaceEvaluation(ctx, review.Key, evidence, findings, review.Status); err != nil {
		return PrepareDecision{}, err
	}
	decision := PrepareDecision{Review: review, Verdict: verdict, Deduped: deduped}
	if len(verdict.Findings) > 0 {
		if p.Goals == nil {
			return PrepareDecision{}, errors.New("readiness goal service is unavailable")
		}
		spec, err := BuildGoalSpec(identity.Scenario, identity.CandidateCommit, request.Deliverable, request.Trigger, policy, verdict)
		if err != nil {
			return PrepareDecision{}, err
		}
		spec.Name = "readiness-" + review.Key
		goal, goalDeduped, err := p.Goals.Open(ctx, spec)
		if err != nil {
			return PrepareDecision{}, err
		}
		if err := p.Repository.SetGoal(ctx, review.Key, goal); err != nil {
			return PrepareDecision{}, err
		}
		decision.GoalRef, decision.Deduped = goal, deduped && goalDeduped
	}
	for _, finding := range verdict.Findings {
		decision.Next = append(decision.Next, finding.ItemID)
	}
	sort.Strings(decision.Next)
	stored, err := p.Repository.Get(ctx, review.Key)
	if err != nil {
		return PrepareDecision{}, err
	}
	decision.Review = *stored
	return decision, nil
}

func resolveApplicability(rule string, facts map[string]string, comparison ComparisonMode) (string, string) {
	switch rule {
	case "all", "declared_targets":
		return "applicable", ""
	case "deployed_predecessor":
		if comparison == ComparisonFirstRelease {
			return "not_applicable", "no deployed predecessor exists"
		}
		if comparison == ComparisonComparable {
			return "applicable", ""
		}
		return "unknown", "deployed predecessor evidence is unavailable"
	case "deployed_predecessor_with_schema_change":
		if comparison == ComparisonFirstRelease {
			return "not_applicable", "first release has no deployed schema to migrate"
		}
		if comparison == ComparisonUnavailable {
			return "unknown", "deployed predecessor evidence is unavailable"
		}
		if value, ok := facts["schema_changed"]; ok && value == "false" {
			return "not_applicable", "candidate and predecessor schema fingerprints are equal"
		}
		if facts["schema_changed"] == "true" {
			return "applicable", ""
		}
		return "unknown", "schema_changed fact was not provided"
	default:
		if value, ok := facts[rule]; ok && value == "false" {
			return "not_applicable", fmt.Sprintf("release fact %s=false", rule)
		}
		if facts[rule] == "true" {
			return "applicable", ""
		}
		return "unknown", fmt.Sprintf("release fact %s is unknown", rule)
	}
}

func (i Item) ProducerBinding() string {
	if i.Producer == nil {
		return ""
	}
	return i.Producer.Binding
}

func unavailableEvidence(key string, identity ReviewIdentity, criterion Item, target string, now time.Time, detail string) EvidenceItem {
	return EvidenceItem{ReviewKey: key, CriterionID: criterion.ID, Status: SignalUnavailable, Applicability: "unknown", Producer: criterion.Owner, CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest, Target: target, Environment: identity.Channel, PolicyVersion: identity.PolicyVersion, ObservedAt: now, Reference: "binding:" + criterion.ProducerBinding(), Detail: detail}
}

func pendingHumanEvidence(key string, identity ReviewIdentity, criterion Item, target string, now time.Time) EvidenceItem {
	return EvidenceItem{ReviewKey: key, CriterionID: criterion.ID, Status: SignalUnknown, Applicability: "applicable", Producer: "swarm-manager", CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest, Target: target, Environment: identity.Channel, PolicyVersion: identity.PolicyVersion, ObservedAt: now, Reference: "human-review:" + criterion.HumanReview.Kind, Detail: "independent human review is pending"}
}

func notApplicableEvidence(key string, identity ReviewIdentity, criterion Item, target string, now time.Time, reason string) EvidenceItem {
	return EvidenceItem{ReviewKey: key, CriterionID: criterion.ID, Status: SignalNotApplicable, Applicability: "not_applicable", ApplicabilityReason: reason, Producer: "deployment-manager", CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest, Target: target, Environment: identity.Channel, PolicyVersion: identity.PolicyVersion, ObservedAt: now, Reference: "applicability:" + criterion.Applicability, Detail: reason}
}
