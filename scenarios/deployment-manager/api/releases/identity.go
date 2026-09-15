package releases

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/targetmodel"
)

// TargetIdentity is the complete delivery target identity. Platform alone is
// not sufficient: two artifacts for the same platform can differ by OS,
// architecture, installer format, or update protocol.
type TargetIdentity struct {
	ID           string `json:"id"`
	Platform     string `json:"platform"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Format       string `json:"format"`
}

func (t TargetIdentity) canonical() (TargetIdentity, error) {
	t.ID = strings.TrimSpace(t.ID)
	t.Platform = strings.TrimSpace(t.Platform)
	t.OS = strings.TrimSpace(t.OS)
	t.Architecture = strings.TrimSpace(t.Architecture)
	t.Format = strings.TrimSpace(t.Format)
	if t.ID == "" || t.Platform == "" || t.OS == "" || t.Architecture == "" || t.Format == "" {
		return TargetIdentity{}, fmt.Errorf("target identity requires id, platform, os, architecture, and format")
	}
	return t, nil
}

func (t TargetIdentity) key() string {
	return strings.Join([]string{t.ID, t.Platform, t.OS, t.Architecture, t.Format}, "\x00")
}

// TargetIdentityFromTarget preserves the shared observable target dimensions
// while adding the artifact format, which belongs to the release artifact and
// is intentionally absent from targetmodel.Target.
func TargetIdentityFromTarget(target targetmodel.Target, format string) (TargetIdentity, error) {
	return (TargetIdentity{ID: target.ID, Platform: target.Platform, OS: target.OS, Architecture: target.Architecture, Format: format}).canonical()
}

// ExecutionTarget projects a release identity into the shared target model for
// routing and comparison. Availability and health remain owner observations.
func (t TargetIdentity) ExecutionTarget() targetmodel.Target {
	return targetmodel.Target{ID: t.ID, Platform: t.Platform, OS: t.OS, Architecture: t.Architecture}
}

// CandidateArtifact identifies final bytes after packaging and signing.
type CandidateArtifact struct {
	Target          TargetIdentity `json:"target"`
	ImmutableRef    string         `json:"immutable_ref"`
	Digest          string         `json:"digest"`
	SizeBytes       int64          `json:"size_bytes"`
	SignatureDigest string         `json:"signature_digest"`
	SignerRef       string         `json:"signer_ref"`
}

// CapabilityDeclaration names the non-secret authorities responsible for the
// candidate's operational lifecycle. It is part of candidate identity when
// supplied, while older candidates may remain readable and fail the
// operations-ownership readiness criterion until they are re-registered with
// the declaration.
type CapabilityDeclaration struct {
	SupportOwner          string `json:"support_owner"`
	IncidentOwner         string `json:"incident_owner"`
	CustomerContact       string `json:"customer_contact"`
	ReleaseAuthority      string `json:"release_authority"`
	RollbackAuthority     string `json:"rollback_authority"`
	DegradedModeAuthority string `json:"degraded_mode_authority"`
}

func (d CapabilityDeclaration) Canonical() (CapabilityDeclaration, error) {
	d.SupportOwner = strings.TrimSpace(d.SupportOwner)
	d.IncidentOwner = strings.TrimSpace(d.IncidentOwner)
	d.CustomerContact = strings.TrimSpace(d.CustomerContact)
	d.ReleaseAuthority = strings.TrimSpace(d.ReleaseAuthority)
	d.RollbackAuthority = strings.TrimSpace(d.RollbackAuthority)
	d.DegradedModeAuthority = strings.TrimSpace(d.DegradedModeAuthority)
	if d.SupportOwner == "" || d.IncidentOwner == "" || d.CustomerContact == "" || d.ReleaseAuthority == "" || d.RollbackAuthority == "" || d.DegradedModeAuthority == "" {
		return CapabilityDeclaration{}, fmt.Errorf("capability declaration requires support, incident, customer contact, release, rollback, and degraded-mode authorities")
	}
	return d, nil
}

func (a CandidateArtifact) canonical() (CandidateArtifact, error) {
	target, err := a.Target.canonical()
	if err != nil {
		return CandidateArtifact{}, err
	}
	a.Target = target
	a.ImmutableRef = strings.TrimSpace(a.ImmutableRef)
	a.Digest = strings.TrimSpace(a.Digest)
	a.SignatureDigest = strings.TrimSpace(a.SignatureDigest)
	a.SignerRef = strings.TrimSpace(a.SignerRef)
	if a.ImmutableRef == "" || a.Digest == "" || a.SizeBytes < 0 || a.SignatureDigest == "" || a.SignerRef == "" {
		return CandidateArtifact{}, fmt.Errorf("candidate artifact requires immutable ref, digest, non-negative size, signature digest, and signer ref")
	}
	return a, nil
}

// Candidate is the immutable source/build-input/signed-artifact identity used
// by review, qualification, and promotion. The identity is computed from the
// canonical final bytes and every input that can change those bytes.
type Candidate struct {
	SourceRevision        string                 `json:"source_revision"`
	ProfileRevision       string                 `json:"profile_revision"`
	BuildInputs           map[string]string      `json:"build_inputs"`
	DependencyLockDigest  string                 `json:"dependency_lock_digest"`
	PolicyDigest          string                 `json:"policy_digest"`
	Artifacts             []CandidateArtifact    `json:"artifacts"`
	CapabilityDeclaration *CapabilityDeclaration `json:"capability_declaration,omitempty"`
}

func (c Candidate) Canonical() (Candidate, error) {
	c.SourceRevision = strings.TrimSpace(c.SourceRevision)
	c.ProfileRevision = strings.TrimSpace(c.ProfileRevision)
	c.DependencyLockDigest = strings.TrimSpace(c.DependencyLockDigest)
	c.PolicyDigest = strings.TrimSpace(c.PolicyDigest)
	if c.SourceRevision == "" || c.ProfileRevision == "" || c.DependencyLockDigest == "" || c.PolicyDigest == "" {
		return Candidate{}, fmt.Errorf("candidate requires source, profile, dependency-lock, and policy revisions")
	}
	if len(c.Artifacts) == 0 {
		return Candidate{}, fmt.Errorf("candidate requires at least one final artifact")
	}
	inputs := make(map[string]string, len(c.BuildInputs))
	for key, value := range c.BuildInputs {
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "" || value == "" {
			return Candidate{}, fmt.Errorf("candidate build inputs require non-empty keys and values")
		}
		inputs[key] = value
	}
	c.BuildInputs = inputs
	if c.CapabilityDeclaration != nil {
		declaration, err := c.CapabilityDeclaration.Canonical()
		if err != nil {
			return Candidate{}, err
		}
		c.CapabilityDeclaration = &declaration
	}
	artifacts := make([]CandidateArtifact, len(c.Artifacts))
	seen := make(map[string]struct{}, len(c.Artifacts))
	for index, artifact := range c.Artifacts {
		canonical, err := artifact.canonical()
		if err != nil {
			return Candidate{}, fmt.Errorf("artifact %d: %w", index, err)
		}
		if _, exists := seen[canonical.Target.key()]; exists {
			return Candidate{}, fmt.Errorf("candidate contains duplicate target %q", canonical.Target.ID)
		}
		seen[canonical.Target.key()] = struct{}{}
		artifacts[index] = canonical
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Target.key() < artifacts[j].Target.key() })
	c.Artifacts = artifacts
	return c, nil
}

func (c Candidate) CanonicalJSON() ([]byte, error) {
	canonical, err := c.Canonical()
	if err != nil {
		return nil, err
	}
	return json.Marshal(canonical)
}

func (c Candidate) Identity() (string, error) {
	payload, err := c.CanonicalJSON()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "candidate-" + hex.EncodeToString(sum[:]), nil
}

// ArtifactManifestDigest identifies the complete ordered set of finalized
// artifacts in the candidate. It is separate from Candidate.Identity so the
// release request can bind the bytes being published without treating the
// request's digest as decorative metadata.
func (c Candidate) ArtifactManifestDigest() (string, error) {
	canonical, err := c.Canonical()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(canonical.Artifacts)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "artifact-manifest-" + hex.EncodeToString(sum[:]), nil
}

// DestinationRevision identifies all non-secret coordinates needed to publish
// to one immutable destination revision.
type DestinationRevision struct {
	Kind                    string `json:"kind"`
	DestinationID           string `json:"destination_id"`
	ConfigurationDigest     string `json:"configuration_digest"`
	Channel                 string `json:"channel"`
	ExpectedChannelRevision string `json:"expected_channel_revision"`
}

func (d DestinationRevision) Canonical() (DestinationRevision, error) {
	d.Kind = strings.TrimSpace(d.Kind)
	d.DestinationID = strings.TrimSpace(d.DestinationID)
	d.ConfigurationDigest = strings.TrimSpace(d.ConfigurationDigest)
	d.Channel = strings.TrimSpace(d.Channel)
	d.ExpectedChannelRevision = strings.TrimSpace(d.ExpectedChannelRevision)
	if d.Kind == "" || d.DestinationID == "" || d.ConfigurationDigest == "" || d.Channel == "" {
		return DestinationRevision{}, fmt.Errorf("destination revision requires kind, destination, configuration digest, and channel")
	}
	return d, nil
}

func (d DestinationRevision) Identity() (string, error) {
	canonical, err := d.Canonical()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "destination-" + hex.EncodeToString(sum[:]), nil
}

// ReviewBinding pins the exact identities and policy that an approval covers.
type ReviewBinding struct {
	CandidateID           string   `json:"candidate_id"`
	DestinationRevisionID string   `json:"destination_revision_id"`
	Targets               []string `json:"targets"`
	Channel               string   `json:"channel"`
	EvidenceSetDigest     string   `json:"evidence_set_digest"`
	PolicyDigest          string   `json:"policy_digest"`
	AuthorizationEpoch    uint64   `json:"authorization_epoch"`
}

func (b ReviewBinding) Canonical() (ReviewBinding, error) {
	b.CandidateID = strings.TrimSpace(b.CandidateID)
	b.DestinationRevisionID = strings.TrimSpace(b.DestinationRevisionID)
	b.Channel = strings.TrimSpace(b.Channel)
	b.EvidenceSetDigest = strings.TrimSpace(b.EvidenceSetDigest)
	b.PolicyDigest = strings.TrimSpace(b.PolicyDigest)
	if b.CandidateID == "" || b.DestinationRevisionID == "" || b.Channel == "" || b.EvidenceSetDigest == "" || b.PolicyDigest == "" || b.AuthorizationEpoch == 0 {
		return ReviewBinding{}, fmt.Errorf("review binding requires candidate, destination, channel, evidence, policy, and authorization epoch")
	}
	seen := map[string]struct{}{}
	for i, target := range b.Targets {
		target = strings.TrimSpace(target)
		if target == "" {
			return ReviewBinding{}, fmt.Errorf("review binding contains an empty target")
		}
		if _, exists := seen[target]; exists {
			return ReviewBinding{}, fmt.Errorf("review binding contains duplicate target %q", target)
		}
		seen[target] = struct{}{}
		b.Targets[i] = target
	}
	if len(b.Targets) == 0 {
		return ReviewBinding{}, fmt.Errorf("review binding requires at least one target")
	}
	sort.Strings(b.Targets)
	return b, nil
}

func (b ReviewBinding) Identity() (string, error) {
	canonical, err := b.Canonical()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "review-" + hex.EncodeToString(sum[:]), nil
}

type ReceiptOutcome string

const (
	ReceiptPrepared    ReceiptOutcome = "prepared"
	ReceiptStaged      ReceiptOutcome = "staged"
	ReceiptPublished   ReceiptOutcome = "published"
	ReceiptVerified    ReceiptOutcome = "verified"
	ReceiptFailed      ReceiptOutcome = "failed"
	ReceiptAmbiguous   ReceiptOutcome = "ambiguous"
	ReceiptUnavailable ReceiptOutcome = "unavailable"
)

// PublicationReceipt is producer-attributed evidence of an external effect.
// A status string alone cannot establish publication; the exact identity and
// producer receipt are required by TrustedPublication.
type PublicationReceipt struct {
	CandidateID           string         `json:"candidate_id"`
	DestinationRevisionID string         `json:"destination_revision_id"`
	TargetID              string         `json:"target_id"`
	ArtifactDigest        string         `json:"artifact_digest"`
	DestinationObject     string         `json:"destination_object"`
	Producer              string         `json:"producer"`
	ExternalReceipt       string         `json:"external_receipt"`
	Outcome               ReceiptOutcome `json:"outcome"`
	ObservedAt            time.Time      `json:"observed_at"`
	// ReviewKey is the readiness review the owner re-checked before the
	// publication effect (exact binding, cloud publications).
	ReviewKey string `json:"review_key,omitempty"`
	// PredecessorArtifactDigest is the release the target reported as
	// previous when the new release was activated, resolved by the owner from
	// its published history, never supplied by the caller.
	PredecessorArtifactDigest string `json:"predecessor_artifact_digest,omitempty"`
	// EvidenceRef points at the owner's per-cell evidence for the published
	// artifact (drill-down from the dossier; bytes stay with the owner).
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

func (r PublicationReceipt) TrustedPublication() bool {
	return r.CandidateID != "" && r.DestinationRevisionID != "" && r.TargetID != "" && r.ArtifactDigest != "" && r.DestinationObject != "" && r.Producer != "" && r.ExternalReceipt != "" && (r.Outcome == ReceiptPublished || r.Outcome == ReceiptVerified) && !r.ObservedAt.IsZero()
}

// ClientUpdateReceipt proves that an installed predecessor consumed and
// relaunched the exact successor artifact.
type ClientUpdateReceipt struct {
	CandidateID     string         `json:"candidate_id"`
	PredecessorRef  string         `json:"predecessor_ref"`
	SuccessorDigest string         `json:"successor_digest"`
	TargetID        string         `json:"target_id"`
	VerifiedVersion string         `json:"verified_version"`
	Outcome         ReceiptOutcome `json:"outcome"`
	Producer        string         `json:"producer"`
	ExternalReceipt string         `json:"external_receipt"`
	ObservedAt      time.Time      `json:"observed_at"`
}

func (r ClientUpdateReceipt) TrustedUpdate() bool {
	return r.CandidateID != "" && r.PredecessorRef != "" && r.SuccessorDigest != "" && r.TargetID != "" && r.VerifiedVersion != "" && r.Producer != "" && r.ExternalReceipt != "" && r.Outcome == ReceiptVerified && !r.ObservedAt.IsZero()
}
