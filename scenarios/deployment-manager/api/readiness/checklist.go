// Package readiness owns the policy and decision state used to prepare a
// release. Evidence producers retain ownership of the measurements themselves.
package readiness

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const ChecklistVersion = 2

//go:embed readiness-policy.json
var builtInPolicyJSON []byte

//go:embed readiness-policy.schema.json
var builtInPolicySchemaJSON []byte

type CleanRequirement string

const (
	Required    CleanRequirement = "required"
	Advisory    CleanRequirement = "advisory"
	Uncheckable CleanRequirement = "human_review"
)

type GlobalImpact string

const (
	FoundationBlocker GlobalImpact = "foundation_blocker"
	SafetyBlocker     GlobalImpact = "safety_blocker"
	CapabilityGap     GlobalImpact = "capability_gap"
	HardeningGap      GlobalImpact = "hardening_gap"
	AdvisoryImpact    GlobalImpact = "advisory"
	UnknownImpact     GlobalImpact = "unknown"
)

type FreshnessPolicy struct {
	Basis         string `json:"basis"`
	MaxAgeSeconds int64  `json:"max_age_seconds,omitempty"`
}

type ProducerRoute struct {
	Binding string `json:"binding"`
}
type HumanReviewRoute struct {
	Kind string `json:"kind"`
}

type GherkinAcceptance struct {
	Given string `json:"given"`
	When  string `json:"when"`
	Then  string `json:"then"`
}

func (a GherkinAcceptance) Sentence() string {
	return fmt.Sprintf("Given %s, when %s, then %s", a.Given, a.When, a.Then)
}

type Remediation struct {
	Skill string `json:"skill"`
	Topic string `json:"topic"`
}

type WaiverPolicy struct {
	Eligible      bool  `json:"eligible"`
	MaxAgeSeconds int64 `json:"max_age_seconds,omitempty"`
}

type Item struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Category           string            `json:"category"`
	Owner              string            `json:"owner"`
	Applicability      string            `json:"applicability"`
	CleanRequirement   CleanRequirement  `json:"requirement"`
	GlobalImpact       GlobalImpact      `json:"global_impact"`
	Freshness          FreshnessPolicy   `json:"freshness"`
	Producer           *ProducerRoute    `json:"producer,omitempty"`
	HumanReview        *HumanReviewRoute `json:"human_review,omitempty"`
	Acceptance         GherkinAcceptance `json:"acceptance"`
	Remediation        Remediation       `json:"remediation"`
	Waiver             WaiverPolicy      `json:"waiver"`
	AcceptanceCriteria string            `json:"-"`
}

type Checklist struct {
	Version int    `json:"version"`
	Items   []Item `json:"items"`
}

// PolicyProducer records the deployment-manager-owned policy declaration as
// evidence. The policy version is part of the candidate identity, so this
// producer can prove the update-policy criterion without treating producer
// registration or a stored observation as proof.
type PolicyProducer struct {
	Policy Checklist
	Now    func() time.Time
}

// UnavailableProducer is an explicit fail-closed adapter for a declared
// producer binding whose owner has not supplied a live execution seam yet.
// It is deliberately an error-producing adapter: Prepare converts the error
// into unavailable evidence, and therefore cannot mistake registration or a
// stale stored observation for current owner execution.
type UnavailableProducer struct {
	Binding string
	Reason  string
}

func (p UnavailableProducer) Collect(_ context.Context, _ ReviewIdentity, criterion Item) (EvidenceItem, error) {
	binding := strings.TrimSpace(p.Binding)
	if binding == "" || criterion.ProducerBinding() != binding {
		return EvidenceItem{}, fmt.Errorf("unavailable producer cannot collect criterion %q", criterion.ID)
	}
	reason := strings.TrimSpace(p.Reason)
	if reason == "" {
		reason = "owner execution is not configured"
	}
	return EvidenceItem{}, fmt.Errorf("producer %s unavailable: %s", binding, reason)
}

// ObservabilityReadiness is the owner observation needed by the target
// observability criterion. The resolver owns the transport and health
// interpretation; this package only binds the result to the review identity.
type ObservabilityReadiness struct {
	Healthy    bool
	Target     string
	Reference  string
	ObservedAt time.Time
	Detail     string
}

type ObservabilityResolver interface {
	CheckObservability(context.Context, ReviewIdentity) (ObservabilityReadiness, error)
}

type ObservabilityProducer struct {
	Resolver ObservabilityResolver
}

func (p ObservabilityProducer) Collect(ctx context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if criterion.ID != "observability-reachable" || criterion.ProducerBinding() != "deployment-manager.observability.readiness" {
		return EvidenceItem{}, fmt.Errorf("observability producer cannot collect criterion %q", criterion.ID)
	}
	if p.Resolver == nil {
		return EvidenceItem{}, fmt.Errorf("observability resolver is not configured")
	}
	observation, err := p.Resolver.CheckObservability(ctx, identity)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("check target observability: %w", err)
	}
	if strings.TrimSpace(observation.Target) == "" || strings.TrimSpace(observation.Reference) == "" || observation.ObservedAt.IsZero() {
		return EvidenceItem{}, fmt.Errorf("target observability owner returned incomplete evidence")
	}
	observedAt := observation.ObservedAt.UTC()
	status := SignalFailed
	if observation.Healthy {
		status = SignalPassed
	}
	detail := strings.TrimSpace(observation.Detail)
	if detail == "" {
		if observation.Healthy {
			detail = "owner health observation proves target observability is reachable"
		} else {
			detail = "owner health observation did not prove target observability"
		}
	}
	return EvidenceItem{
		CriterionID: criterion.ID, Status: status, Applicability: "applicable",
		Producer: "deployment-manager", ProducerVersion: "cloud-observability-v1",
		CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
		Target: observation.Target, Environment: identity.Channel,
		PolicyVersion: identity.PolicyVersion, ObservedAt: observedAt,
		Reference: observation.Reference, Detail: detail,
	}, nil
}

func (p PolicyProducer) Collect(_ context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	policy := p.Policy
	if len(policy.Items) == 0 {
		policy = DefaultChecklist()
	}
	if err := policy.Validate(); err != nil {
		return EvidenceItem{}, fmt.Errorf("validate readiness policy: %w", err)
	}
	if criterion.ID != "update-policy-set" || criterion.ProducerBinding() != "deployment-manager.policy.update" {
		return EvidenceItem{}, fmt.Errorf("policy producer cannot collect criterion %q", criterion.ID)
	}
	observedAt := time.Now().UTC()
	if p.Now != nil {
		observedAt = p.Now().UTC()
	}
	digest, err := PolicyDigest(policy)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("digest readiness policy: %w", err)
	}
	status := SignalPassed
	detail := fmt.Sprintf("active readiness policy version %d is bound to the candidate", policy.Version)
	if identity.PolicyVersion != policy.Version {
		status = SignalFailed
		detail = fmt.Sprintf("candidate policy version %d does not match active version %d", identity.PolicyVersion, policy.Version)
	}
	return EvidenceItem{
		CriterionID: criterion.ID, Status: status, Applicability: "applicable",
		Producer: "deployment-manager", ProducerVersion: fmt.Sprintf("policy-v%d", policy.Version),
		CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
		Target: strings.Join(identity.Targets, ","), Environment: identity.Channel,
		PolicyVersion: identity.PolicyVersion, ObservedAt: observedAt,
		Reference: "readiness-policy:" + digest, Detail: detail,
	}, nil
}

// CandidateProvenance is the immutable candidate projection needed by the
// artifact provenance criterion. The resolver owns the candidate store; the
// readiness package only evaluates the returned facts.
type CandidateProvenance struct {
	ID                     string
	SourceRevision         string
	ArtifactManifestDigest string
	DependencyLockDigest   string
	PolicyDigest           string
	BuildInputsPresent     bool
	ArtifactCount          int
	SignedArtifactCount    int
	ArtifactTargetIDs      []string
	ArtifactPlatforms      []string
	ArtifactRefsComplete   bool
	CapabilityDeclaration  OperationsOwnership
}

// OperationsOwnership is the candidate-bound projection needed by the
// operations-ownership criterion. The candidate repository owns the
// declaration; readiness only checks that every authority is named.
type OperationsOwnership struct {
	SupportOwner          string
	IncidentOwner         string
	CustomerContact       string
	ReleaseAuthority      string
	RollbackAuthority     string
	DegradedModeAuthority string
}

type OperationsOwnershipProducer struct {
	Resolver CandidateResolver
	Now      func() time.Time
}

func (p OperationsOwnershipProducer) Collect(ctx context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if criterion.ID != "operations-ownership-set" || criterion.ProducerBinding() != "deployment-manager.operations.readiness" {
		return EvidenceItem{}, fmt.Errorf("operations ownership producer cannot collect criterion %q", criterion.ID)
	}
	if p.Resolver == nil {
		return EvidenceItem{}, fmt.Errorf("candidate resolver is not configured")
	}
	if strings.TrimSpace(identity.CandidateID) == "" {
		return EvidenceItem{}, fmt.Errorf("candidate identity is required for operations ownership")
	}
	candidate, err := p.Resolver.ResolveCandidate(ctx, identity.CandidateID)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("resolve candidate %q: %w", identity.CandidateID, err)
	}
	ownership := candidate.CapabilityDeclaration
	missing := make([]string, 0, 6)
	if candidate.ID != identity.CandidateID {
		missing = append(missing, "candidate id")
	}
	for name, value := range map[string]string{
		"support owner": ownership.SupportOwner, "incident owner": ownership.IncidentOwner,
		"customer contact": ownership.CustomerContact, "release authority": ownership.ReleaseAuthority,
		"rollback authority": ownership.RollbackAuthority, "degraded-mode authority": ownership.DegradedModeAuthority,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	observedAt := time.Now().UTC()
	if p.Now != nil {
		observedAt = p.Now().UTC()
	}
	status := SignalPassed
	detail := "candidate names support, incident, customer contact, release, rollback, and degraded-mode authorities"
	if len(missing) > 0 {
		status = SignalFailed
		detail = "candidate operational ownership is incomplete: missing " + strings.Join(missing, ", ")
	}
	return EvidenceItem{
		CriterionID: criterion.ID, Status: status, Applicability: "applicable",
		Producer: "deployment-manager", ProducerVersion: "operations-ownership-v1",
		CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
		Target: strings.Join(identity.Targets, ","), Environment: identity.Channel,
		PolicyVersion: identity.PolicyVersion, ObservedAt: observedAt,
		Reference: "candidate-operations:" + candidate.ID, Detail: detail,
	}, nil
}

type PlatformAssetsProducer struct {
	Resolver CandidateResolver
	Now      func() time.Time
}

func (p PlatformAssetsProducer) Collect(ctx context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if criterion.ID != "platform-assets-set" || criterion.ProducerBinding() != "deployment-manager.assets.readiness" {
		return EvidenceItem{}, fmt.Errorf("platform assets producer cannot collect criterion %q", criterion.ID)
	}
	if p.Resolver == nil {
		return EvidenceItem{}, fmt.Errorf("candidate resolver is not configured")
	}
	if strings.TrimSpace(identity.CandidateID) == "" {
		return EvidenceItem{}, fmt.Errorf("candidate identity is required for platform assets")
	}
	candidate, err := p.Resolver.ResolveCandidate(ctx, identity.CandidateID)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("resolve candidate %q: %w", identity.CandidateID, err)
	}
	ids := make(map[string]struct{}, len(candidate.ArtifactTargetIDs))
	platforms := make(map[string]struct{}, len(candidate.ArtifactPlatforms))
	for _, value := range candidate.ArtifactTargetIDs {
		ids[strings.TrimSpace(value)] = struct{}{}
	}
	for _, value := range candidate.ArtifactPlatforms {
		platforms[strings.TrimSpace(value)] = struct{}{}
	}
	missing := make([]string, 0)
	if candidate.ID != identity.CandidateID {
		missing = append(missing, "candidate id")
	}
	for _, target := range identity.Targets {
		if _, ok := ids[target]; ok {
			continue
		}
		if _, ok := platforms[target]; ok {
			continue
		}
		missing = append(missing, target)
	}
	observedAt := time.Now().UTC()
	if p.Now != nil {
		observedAt = p.Now().UTC()
	}
	status := SignalPassed
	detail := fmt.Sprintf("candidate assets cover %d declared target(s)", len(identity.Targets))
	if !candidate.ArtifactRefsComplete || len(missing) > 0 {
		status = SignalFailed
		parts := make([]string, 0, 2)
		if len(missing) > 0 {
			parts = append(parts, "missing="+strings.Join(missing, ","))
		}
		if !candidate.ArtifactRefsComplete {
			parts = append(parts, "incomplete artifact identity")
		}
		detail = "candidate platform assets are incomplete: " + strings.Join(parts, "; ")
	}
	return EvidenceItem{
		CriterionID: criterion.ID, Status: status, Applicability: "applicable",
		Producer: "deployment-manager", ProducerVersion: "platform-assets-v1",
		CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
		Target: strings.Join(identity.Targets, ","), Environment: identity.Channel,
		PolicyVersion: identity.PolicyVersion, ObservedAt: observedAt,
		Reference: "candidate-assets:" + candidate.ID + ":" + identity.ArtifactDigest, Detail: detail,
	}, nil
}

type RecoveryReadiness struct {
	ExecutorConfigured      bool
	AuthorizationConfigured bool
	TargetIdentityBound     bool
}

type RecoveryReadinessResolver interface {
	CheckRecovery(context.Context, ReviewIdentity) (RecoveryReadiness, error)
}

type RecoveryProducer struct {
	Resolver RecoveryReadinessResolver
	Now      func() time.Time
}

func (p RecoveryProducer) Collect(ctx context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if criterion.ID != "recovery-proven" || criterion.ProducerBinding() != "deployment-manager.recovery.preflight" {
		return EvidenceItem{}, fmt.Errorf("recovery producer cannot collect criterion %q", criterion.ID)
	}
	if p.Resolver == nil {
		return EvidenceItem{}, fmt.Errorf("recovery readiness resolver is not configured")
	}
	readiness, err := p.Resolver.CheckRecovery(ctx, identity)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("check recovery readiness: %w", err)
	}
	observedAt := time.Now().UTC()
	if p.Now != nil {
		observedAt = p.Now().UTC()
	}
	missing := make([]string, 0, 3)
	if !readiness.ExecutorConfigured {
		missing = append(missing, "recovery executor")
	}
	if !readiness.AuthorizationConfigured {
		missing = append(missing, "destructive authorization")
	}
	if !readiness.TargetIdentityBound {
		missing = append(missing, "candidate, destination, and authorization epoch")
	}
	status := SignalPassed
	detail := "recovery executor, destructive authorization, and target identity are configured"
	if len(missing) > 0 {
		status = SignalFailed
		detail = "recovery preflight is incomplete: " + strings.Join(missing, ", ")
	}
	return EvidenceItem{
		CriterionID: criterion.ID, Status: status, Applicability: "applicable",
		Producer: "deployment-manager", ProducerVersion: "recovery-preflight-v1",
		CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
		Target: strings.Join(identity.Targets, ","), Environment: identity.Channel,
		PolicyVersion: identity.PolicyVersion, ObservedAt: observedAt,
		Reference: "recovery-preflight:" + identity.ProfileID + ":" + identity.CandidateCommit + ":" + identity.DestinationRevisionID, Detail: detail,
	}, nil
}

type CandidateResolver interface {
	ResolveCandidate(context.Context, string) (CandidateProvenance, error)
}

type CandidateProvenanceProducer struct {
	Resolver             CandidateResolver
	ExpectedPolicyDigest string
	Now                  func() time.Time
}

func (p CandidateProvenanceProducer) Collect(ctx context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if criterion.ID != "artifact-provenance-complete" || criterion.ProducerBinding() != "deployment-manager.artifacts.provenance" {
		return EvidenceItem{}, fmt.Errorf("candidate provenance producer cannot collect criterion %q", criterion.ID)
	}
	if p.Resolver == nil {
		return EvidenceItem{}, fmt.Errorf("candidate provenance resolver is not configured")
	}
	if strings.TrimSpace(identity.CandidateID) == "" {
		return EvidenceItem{}, fmt.Errorf("candidate identity is required for artifact provenance")
	}
	candidate, err := p.Resolver.ResolveCandidate(ctx, identity.CandidateID)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("resolve candidate %q: %w", identity.CandidateID, err)
	}
	observedAt := time.Now().UTC()
	if p.Now != nil {
		observedAt = p.Now().UTC()
	}
	checks := []string{}
	if candidate.ID != identity.CandidateID {
		checks = append(checks, "candidate id")
	}
	if candidate.SourceRevision != identity.CandidateCommit {
		checks = append(checks, "source revision")
	}
	if candidate.ArtifactManifestDigest != identity.ArtifactDigest {
		checks = append(checks, "artifact manifest digest")
	}
	if candidate.DependencyLockDigest == "" {
		checks = append(checks, "dependency lock digest")
	}
	if candidate.PolicyDigest == "" {
		checks = append(checks, "policy digest")
	} else if strings.TrimSpace(p.ExpectedPolicyDigest) != "" && candidate.PolicyDigest != strings.TrimSpace(p.ExpectedPolicyDigest) {
		checks = append(checks, "active policy digest")
	}
	if !candidate.BuildInputsPresent {
		checks = append(checks, "build inputs")
	}
	if candidate.ArtifactCount == 0 || candidate.SignedArtifactCount != candidate.ArtifactCount {
		checks = append(checks, "signed artifact coverage")
	}
	status := SignalPassed
	detail := fmt.Sprintf("candidate %s matches source, artifact, build-input, dependency-lock, and signature identity", candidate.ID)
	if len(checks) > 0 {
		status = SignalFailed
		detail = "candidate provenance mismatch: " + strings.Join(checks, ", ")
	}
	return EvidenceItem{
		CriterionID: criterion.ID, Status: status, Applicability: "applicable",
		Producer: "deployment-manager", ProducerVersion: "candidate-provenance-v1",
		CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
		Target: strings.Join(identity.Targets, ","), Environment: identity.Channel,
		PolicyVersion: identity.PolicyVersion, ObservedAt: observedAt,
		Reference: "candidate:" + candidate.ID + ":" + candidate.ArtifactManifestDigest, Detail: detail,
	}, nil
}

type RampEvidence struct {
	Target      string
	Platform    string
	OS          string
	Ramp        string
	Disposition string
	RunID       string
	Reference   string
}

type RampEvidenceResolver interface {
	ListRampEvidence(context.Context, string, string) ([]RampEvidence, error)
}

type RampEvidenceProducer struct {
	Resolver RampEvidenceResolver
	Now      func() time.Time
}

func (p RampEvidenceProducer) Collect(ctx context.Context, identity ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if criterion.ID != "ramp-evidence-complete" || criterion.ProducerBinding() != "deployment-manager.evidence.coverage" {
		return EvidenceItem{}, fmt.Errorf("ramp evidence producer cannot collect criterion %q", criterion.ID)
	}
	if p.Resolver == nil {
		return EvidenceItem{}, fmt.Errorf("ramp evidence resolver is not configured")
	}
	rows, err := p.Resolver.ListRampEvidence(ctx, identity.ProfileID, identity.CandidateCommit)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("list target evidence: %w", err)
	}
	usable := make([]RampEvidence, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.RunID) == "" {
			continue
		}
		usable = append(usable, row)
	}
	missing := make([]string, 0)
	failed := make([]string, 0)
	runs := make([]string, 0, len(identity.Targets))
	usedRows := make(map[int]struct{}, len(identity.Targets))
	for _, target := range identity.Targets {
		row, rowIndex, ok := matchingRampEvidence(usable, target, usedRows)
		if !ok {
			missing = append(missing, target)
			continue
		}
		usedRows[rowIndex] = struct{}{}
		if strings.ToLower(strings.TrimSpace(row.Disposition)) != "passed" {
			failed = append(failed, target)
			continue
		}
		runs = append(runs, row.RunID)
	}
	observedAt := time.Now().UTC()
	if p.Now != nil {
		observedAt = p.Now().UTC()
	}
	status := SignalPassed
	detail := fmt.Sprintf("current target evidence covers %d declared target(s)", len(identity.Targets))
	if len(missing) > 0 || len(failed) > 0 {
		status = SignalFailed
		parts := make([]string, 0, 2)
		if len(missing) > 0 {
			parts = append(parts, "missing="+strings.Join(missing, ","))
		}
		if len(failed) > 0 {
			parts = append(parts, "failed="+strings.Join(failed, ","))
		}
		detail = "target evidence coverage is incomplete: " + strings.Join(parts, "; ")
	}
	return EvidenceItem{
		CriterionID: criterion.ID, Status: status, Applicability: "applicable",
		Producer: "deployment-manager", ProducerVersion: "ramp-evidence-v1",
		CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
		Target: strings.Join(identity.Targets, ","), Environment: identity.Channel,
		PolicyVersion: identity.PolicyVersion, ObservedAt: observedAt,
		Reference: "evidence:" + identity.ProfileID + ":" + identity.CandidateCommit + ":" + strings.Join(runs, ","), Detail: detail,
	}, nil
}

func matchingRampEvidence(rows []RampEvidence, declared string, used map[int]struct{}) (RampEvidence, int, bool) {
	declared = strings.ToLower(strings.TrimSpace(declared))
	for index, row := range rows {
		if _, alreadyUsed := used[index]; alreadyUsed {
			continue
		}
		for _, candidate := range []string{row.Target, row.Platform, row.OS, row.Ramp} {
			candidate = strings.ToLower(strings.TrimSpace(candidate))
			if candidate != "" && (declared == candidate || strings.HasPrefix(declared, candidate+"-")) {
				return row, index, true
			}
		}
	}
	return RampEvidence{}, -1, false
}

var knownOwners = map[string]struct{}{
	"business-health": {}, "content-desk": {}, "deployment-manager": {},
	"git-control-tower": {}, "measures-health": {}, "offer-desk": {},
	"scenario-dependency-analyzer": {}, "scenario-to-desktop": {},
	"secrets-manager": {}, "security-health": {}, "storage-manager": {},
	"swarm-manager": {}, "test-genie": {},
}

func (i Item) Validate() error {
	if strings.TrimSpace(i.ID) == "" || strings.TrimSpace(i.Title) == "" {
		return fmt.Errorf("checklist item requires id and title")
	}
	if strings.TrimSpace(i.Category) == "" || strings.TrimSpace(i.Owner) == "" || strings.TrimSpace(i.Applicability) == "" {
		return fmt.Errorf("checklist item %q requires category, owner, and applicability", i.ID)
	}
	if _, ok := knownOwners[i.Owner]; !ok {
		return fmt.Errorf("checklist item %q has unknown owner %q", i.ID, i.Owner)
	}
	switch i.CleanRequirement {
	case Required, Advisory, Uncheckable:
	default:
		return fmt.Errorf("checklist item %q has invalid requirement %q", i.ID, i.CleanRequirement)
	}
	switch i.GlobalImpact {
	case FoundationBlocker, SafetyBlocker, CapabilityGap, HardeningGap, AdvisoryImpact, UnknownImpact:
	default:
		return fmt.Errorf("checklist item %q has invalid global_impact %q", i.ID, i.GlobalImpact)
	}
	if i.Freshness.Basis != "candidate_identity" && i.Freshness.Basis != "max_age" {
		return fmt.Errorf("checklist item %q has invalid freshness basis %q", i.ID, i.Freshness.Basis)
	}
	if i.Freshness.Basis == "max_age" && i.Freshness.MaxAgeSeconds <= 0 {
		return fmt.Errorf("checklist item %q max_age freshness requires max_age_seconds", i.ID)
	}
	if (i.Producer == nil) == (i.HumanReview == nil) {
		return fmt.Errorf("checklist item %q requires exactly one producer or human_review route", i.ID)
	}
	if i.Producer != nil && (!strings.Contains(i.Producer.Binding, ".") || strings.ContainsAny(i.Producer.Binding, " \t\n")) {
		return fmt.Errorf("checklist item %q has invalid producer binding %q", i.ID, i.Producer.Binding)
	}
	if i.HumanReview != nil && strings.TrimSpace(i.HumanReview.Kind) == "" {
		return fmt.Errorf("checklist item %q has empty human_review kind", i.ID)
	}
	structuredAcceptance := strings.TrimSpace(i.Acceptance.Given) != "" && strings.TrimSpace(i.Acceptance.When) != "" && strings.TrimSpace(i.Acceptance.Then) != ""
	if !structuredAcceptance && strings.TrimSpace(i.AcceptanceCriteria) == "" {
		return fmt.Errorf("checklist item %q requires Given/When/Then acceptance", i.ID)
	}
	if strings.TrimSpace(i.Remediation.Skill) == "" || strings.TrimSpace(i.Remediation.Topic) == "" {
		return fmt.Errorf("checklist item %q requires remediation skill and topic", i.ID)
	}
	return nil
}

func (c Checklist) Validate() error {
	if c.Version != ChecklistVersion {
		return fmt.Errorf("unsupported checklist version %d", c.Version)
	}
	if len(c.Items) == 0 {
		return fmt.Errorf("readiness checklist is empty")
	}
	seen := make(map[string]struct{}, len(c.Items))
	for _, item := range c.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		if _, ok := seen[item.ID]; ok {
			return fmt.Errorf("duplicate checklist item %q", item.ID)
		}
		seen[item.ID] = struct{}{}
	}
	return nil
}

func decodePolicy(data []byte) (Checklist, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("readiness-policy.schema.json", bytes.NewReader(builtInPolicySchemaJSON)); err != nil {
		return Checklist{}, fmt.Errorf("load readiness policy schema: %w", err)
	}
	schema, err := compiler.Compile("readiness-policy.schema.json")
	if err != nil {
		return Checklist{}, fmt.Errorf("compile readiness policy schema: %w", err)
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return Checklist{}, fmt.Errorf("decode readiness policy document: %w", err)
	}
	if err := schema.Validate(document); err != nil {
		return Checklist{}, fmt.Errorf("validate readiness policy schema: %w", err)
	}
	var checklist Checklist
	if err := json.Unmarshal(data, &checklist); err != nil {
		return Checklist{}, fmt.Errorf("decode readiness policy: %w", err)
	}
	if err := checklist.Validate(); err != nil {
		return Checklist{}, err
	}
	return checklist, nil
}

func Load(path string) (Checklist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Checklist{}, fmt.Errorf("read readiness policy: %w", err)
	}
	return decodePolicy(data)
}

func mustBuiltInPolicy() Checklist {
	policy, err := decodePolicy(builtInPolicyJSON)
	if err != nil {
		panic(fmt.Sprintf("invalid built-in readiness policy: %v", err))
	}
	return policy
}

var builtInPolicy = mustBuiltInPolicy()

func DefaultChecklist() Checklist {
	policy := builtInPolicy
	policy.Items = append([]Item(nil), builtInPolicy.Items...)
	return policy
}

func CheckProjection(data []byte) error {
	candidate, err := decodePolicy(data)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(candidate, builtInPolicy) {
		return fmt.Errorf("readiness policy projection differs from built-in policy version %d", ChecklistVersion)
	}
	return nil
}

func BuiltInPolicyJSON() []byte { return append([]byte(nil), builtInPolicyJSON...) }

func PolicyDigest(policy Checklist) (string, error) {
	canonical := policy
	if err := canonical.Validate(); err != nil {
		return "", err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + fmt.Sprintf("%x", sum[:]), nil
}
