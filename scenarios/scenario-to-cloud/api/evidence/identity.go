package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence"
)

// ReviewIdentity is the exact identity a governance decision binds to:
// source (candidate commit), artifact (release / candidate digest),
// configuration, target set, environment, channel and policy version. A
// publication may only proceed for the identity whose digest the approved
// review carries; any change produces a different digest.
type ReviewIdentity struct {
	ScenarioID      string `json:"scenario_id"`
	ProfileID       string `json:"profile_id"`
	CandidateCommit string `json:"candidate_commit"`
	// ArtifactDigest is Deployment Manager's candidate artifact digest when the
	// release is governed there; otherwise it equals ReleaseDigest.
	ArtifactDigest      string   `json:"artifact_digest"`
	ReleaseDigest       string   `json:"release_digest"`
	ConfigurationDigest string   `json:"configuration_digest"`
	TargetSet           []string `json:"target_set"`
	Environment         string   `json:"environment"`
	Channel             string   `json:"channel"`
	PolicyVersion       int      `json:"policy_version"`
	// Release-bound extras (all three together or none), mirrored from the
	// Deployment Manager readiness identity.
	CandidateID           string `json:"candidate_id,omitempty"`
	DestinationRevisionID string `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch    uint64 `json:"authorization_epoch,omitempty"`
}

// Canonical trims, sorts and validates the identity. It refuses an identity
// that cannot be bound exactly.
func (i ReviewIdentity) Canonical() (ReviewIdentity, error) {
	i.ScenarioID = strings.TrimSpace(i.ScenarioID)
	i.ProfileID = strings.TrimSpace(i.ProfileID)
	i.CandidateCommit = strings.TrimSpace(i.CandidateCommit)
	i.ArtifactDigest = strings.TrimSpace(i.ArtifactDigest)
	i.ReleaseDigest = strings.ToLower(strings.TrimSpace(i.ReleaseDigest))
	i.ConfigurationDigest = strings.ToLower(strings.TrimSpace(i.ConfigurationDigest))
	i.Environment = strings.TrimSpace(i.Environment)
	i.Channel = strings.TrimSpace(i.Channel)
	i.CandidateID = strings.TrimSpace(i.CandidateID)
	i.DestinationRevisionID = strings.TrimSpace(i.DestinationRevisionID)
	if i.ArtifactDigest == "" {
		i.ArtifactDigest = i.ReleaseDigest
	}
	if i.ScenarioID == "" || i.ProfileID == "" || i.CandidateCommit == "" || i.ReleaseDigest == "" || i.ConfigurationDigest == "" || i.Environment == "" || i.Channel == "" || i.PolicyVersion <= 0 {
		return ReviewIdentity{}, fmt.Errorf("review identity requires scenario, profile, candidate commit, release digest, configuration digest, environment, channel and policy version")
	}
	targets := make([]string, 0, len(i.TargetSet))
	seen := map[string]struct{}{}
	for _, target := range i.TargetSet {
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
	i.TargetSet = targets
	bound := i.CandidateID != ""
	if bound != (i.DestinationRevisionID != "") || bound != (i.AuthorizationEpoch != 0) {
		return ReviewIdentity{}, fmt.Errorf("release-bound identity requires candidate, destination revision and authorization epoch together")
	}
	return i, nil
}

// Digest is sha256 over the canonical JSON of the identity.
func (i ReviewIdentity) Digest() (string, error) {
	canonical, err := i.Canonical()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Proto projects the identity (with its digest) onto the wire.
func (i ReviewIdentity) Proto() *evidencev1.ReviewIdentity {
	digest, _ := i.Digest()
	return &evidencev1.ReviewIdentity{
		ScenarioId: i.ScenarioID, ProfileId: i.ProfileID, CandidateCommit: i.CandidateCommit,
		ArtifactDigest: i.ArtifactDigest, ReleaseDigest: i.ReleaseDigest, ConfigurationDigest: i.ConfigurationDigest,
		TargetSet: append([]string(nil), i.TargetSet...), Environment: i.Environment, Channel: i.Channel,
		PolicyVersion: int32(i.PolicyVersion), CandidateId: i.CandidateID, DestinationRevisionId: i.DestinationRevisionID,
		AuthorizationEpoch: i.AuthorizationEpoch, Digest: digest,
	}
}

// IdentityFromProto decodes the wire identity. The wire digest is ignored:
// the digest is always recomputed from the fields.
func IdentityFromProto(p *evidencev1.ReviewIdentity) ReviewIdentity {
	if p == nil {
		return ReviewIdentity{}
	}
	return ReviewIdentity{
		ScenarioID: p.GetScenarioId(), ProfileID: p.GetProfileId(), CandidateCommit: p.GetCandidateCommit(),
		ArtifactDigest: p.GetArtifactDigest(), ReleaseDigest: p.GetReleaseDigest(), ConfigurationDigest: p.GetConfigurationDigest(),
		TargetSet: append([]string(nil), p.GetTargetSet()...), Environment: p.GetEnvironment(), Channel: p.GetChannel(),
		PolicyVersion: int(p.GetPolicyVersion()), CandidateID: p.GetCandidateId(), DestinationRevisionID: p.GetDestinationRevisionId(),
		AuthorizationEpoch: p.GetAuthorizationEpoch(),
	}
}

// ReviewSnapshot is what the governance owner reports back about a review.
// Identity fields are the Deployment Manager readiness identity; the cloud
// compares them field by field with its own canonical identity.
type ReviewSnapshot struct {
	Key                   string   `json:"key"`
	Status                string   `json:"status"`
	Scenario              string   `json:"scenario"`
	ProfileID             string   `json:"profile_id"`
	CandidateCommit       string   `json:"candidate_commit"`
	ArtifactDigest        string   `json:"artifact_digest"`
	Targets               []string `json:"targets"`
	Channel               string   `json:"channel"`
	PolicyVersion         int      `json:"policy_version"`
	CandidateID           string   `json:"candidate_id,omitempty"`
	DestinationRevisionID string   `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch    uint64   `json:"authorization_epoch,omitempty"`
	PredecessorReleaseID  string   `json:"predecessor_release_id,omitempty"`
	PredecessorArtifact   string   `json:"predecessor_artifact_digest,omitempty"`
	ApprovedBy            string   `json:"approved_by,omitempty"`
}

// Review status vocabulary as reported by Deployment Manager.
const (
	ReviewApproved   = "approved"
	ReviewPromoted   = "promoted"
	ReviewSuperseded = "superseded"
)

// MatchesIdentity reports whether the review the owner returned is bound to
// exactly this identity. A mismatch on any bound field is a refusal reason.
func (r ReviewSnapshot) MatchesIdentity(identity ReviewIdentity) error {
	canonical, err := identity.Canonical()
	if err != nil {
		return err
	}
	targets := append([]string(nil), r.Targets...)
	sort.Strings(targets)
	switch {
	case r.Scenario != canonical.ScenarioID:
		return fmt.Errorf("review scenario %q is not %q", r.Scenario, canonical.ScenarioID)
	case r.ProfileID != canonical.ProfileID:
		return fmt.Errorf("review profile %q is not %q", r.ProfileID, canonical.ProfileID)
	case r.CandidateCommit != canonical.CandidateCommit:
		return fmt.Errorf("review candidate commit %q is not %q", r.CandidateCommit, canonical.CandidateCommit)
	case r.ArtifactDigest != canonical.ArtifactDigest:
		return fmt.Errorf("review artifact digest %q is not %q", r.ArtifactDigest, canonical.ArtifactDigest)
	case strings.Join(targets, ",") != strings.Join(canonical.TargetSet, ","):
		return fmt.Errorf("review target set %v is not %v", targets, canonical.TargetSet)
	case r.Channel != canonical.Channel:
		return fmt.Errorf("review channel %q is not %q", r.Channel, canonical.Channel)
	case r.PolicyVersion != canonical.PolicyVersion:
		return fmt.Errorf("review policy version %d is not %d", r.PolicyVersion, canonical.PolicyVersion)
	case r.CandidateID != canonical.CandidateID || r.DestinationRevisionID != canonical.DestinationRevisionID || r.AuthorizationEpoch != canonical.AuthorizationEpoch:
		return fmt.Errorf("review release binding does not match the requested candidate, destination revision and authorization epoch")
	}
	return nil
}
