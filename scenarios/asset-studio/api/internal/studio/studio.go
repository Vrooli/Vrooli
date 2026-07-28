// Package studio contains the policy-rich core of Asset Studio's P0 spine.
// Transport and persistence adapters deliberately stay outside this package:
// the release rules must be identical for the UI, CLI, and Connect handlers.
package studio

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type IdentityKind string

const (
	Character IdentityKind = "character"
	Scene     IdentityKind = "scene"
	Product   IdentityKind = "product"
)

type ActorKind string

const (
	Operator ActorKind = "operator"
	Agent    ActorKind = "agent"
)

type Identity struct {
	ID, Name, CredentialClaims string
	Kind                       IdentityKind
	Version                    int
	Traits                     map[string]string
	ReferenceImages            []string
	ConditioningReferences     []ConditioningReference
	Referenced                 bool
	Revisions                  []Revision
}

type (
	ConditioningReference struct{ Kind, ID, Version string }
	Revision              struct {
		ActorID   string
		ActorKind ActorKind
		At        time.Time
	}
)

func (i Identity) Validate() error {
	if strings.TrimSpace(i.Name) == "" {
		return errors.New("identity name is required")
	}
	var required []string
	switch i.Kind {
	case Character:
		required = []string{"face", "build"}
	case Scene:
		required = []string{"environment", "lighting"}
	case Product:
		required = []string{"form", "finish"}
	default:
		return fmt.Errorf("unrecognized identity kind %q", i.Kind)
	}
	for _, field := range required {
		if strings.TrimSpace(i.Traits[field]) == "" {
			return fmt.Errorf("%s trait is required for %s", field, i.Kind)
		}
	}
	return nil
}

type Spec struct {
	ID, Template, CampaignRef string
	IdentityVersionIDs        []string
	Fields                    map[string]string
}

func (s Spec) Resolve(identities map[string]Identity) (string, error) {
	resolved := s.Template
	keys := make([]string, 0, len(s.Fields))
	for key := range s.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		resolved = strings.ReplaceAll(resolved, "{{"+key+"}}", s.Fields[key])
	}
	var missing []string
	for {
		start := strings.Index(resolved, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(resolved[start:], "}}")
		if end < 0 {
			break
		}
		missing = append(missing, resolved[start+2:start+end])
		resolved = resolved[:start] + resolved[start+end+2:]
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("unfilled required fields: %s", strings.Join(missing, ", "))
	}
	for _, id := range s.IdentityVersionIDs {
		if _, ok := identities[id]; !ok {
			return "", fmt.Errorf("identity version %q is not resolvable", id)
		}
	}
	return resolved, nil
}

type RenderStatus string

const (
	RenderQueued    RenderStatus = "queued"
	RenderRunning   RenderStatus = "running"
	RenderSucceeded RenderStatus = "succeeded"
	RenderFailed    RenderStatus = "failed"
	RenderCancelled RenderStatus = "cancelled"
)

func (s RenderStatus) terminal() bool {
	return s == RenderSucceeded || s == RenderFailed || s == RenderCancelled
}

type Provenance struct {
	SpecID                           string
	IdentityVersionIDs               []string
	Backend, Model, Seed, Parameters string
}
type Render struct {
	ID                        string
	Status                    RenderStatus
	EstimatedCost, ActualCost float64
	ActualCostRecorded        bool
	Provenance                *Provenance
	AssetIDs                  []string
}

func (r *Render) Transition(next RenderStatus) error {
	if r.Status.terminal() {
		return fmt.Errorf("terminal render %q cannot transition", r.ID)
	}
	allowed := (r.Status == RenderQueued && (next == RenderRunning || next == RenderCancelled)) || (r.Status == RenderRunning && (next == RenderSucceeded || next == RenderFailed || next == RenderCancelled))
	if !allowed {
		return fmt.Errorf("illegal render transition %s -> %s", r.Status, next)
	}
	if next == RenderSucceeded && r.Provenance == nil {
		return errors.New("succeeded render requires provenance")
	}
	if next.terminal() && !r.ActualCostRecorded {
		return errors.New("terminal render requires actual cost")
	}
	r.Status = next
	return nil
}

type AssetStatus string

const (
	Produced  AssetStatus = "produced"
	InReview  AssetStatus = "in_review"
	Discarded AssetStatus = "discarded"
	Released  AssetStatus = "released"
)

type Asset struct {
	ID, RenderID, BlobKey, AltText, Disclosure, CredentialClaims string
	Status                                                       AssetStatus
	AIgenerated                                                  bool
	IdentityVersionIDs                                           []string
}
type VerdictBasis string

const (
	ReferenceSheet       VerdictBasis = "reference-sheet"
	ReferenceImageSet    VerdictBasis = "reference-image-set"
	ConditioningArtifact VerdictBasis = "conditioning-artifact"
	ProseOnly            VerdictBasis = "prose-only"
)

type Verdict struct {
	AssetID, IdentityVersionID, ActorID string
	ActorKind                           ActorKind
	Passed                              bool
	Basis                               VerdictBasis
	At                                  time.Time
}
type ReleaseCause string

const (
	CauseNotSelected      ReleaseCause = "candidate_not_selected"
	CauseDiscarded        ReleaseCause = "candidate_discarded"
	CauseAltText          ReleaseCause = "missing_alt_text"
	CauseDisclosure       ReleaseCause = "missing_disclosure"
	CauseCredentialClaims ReleaseCause = "credential_claims_present"
	CauseUnresolved       ReleaseCause = "unresolved_conformance"
	CauseFailed           ReleaseCause = "failed_conformance"
)

type ReleaseError struct {
	Cause  ReleaseCause
	Detail string
}

func (e *ReleaseError) Error() string { return string(e.Cause) + ": " + e.Detail }

type Studio struct {
	Identities   map[string]Identity
	Specs        map[string]Spec
	Renders      map[string]*Render
	Assets       map[string]*Asset
	Verdicts     []Verdict
	ImportHashes map[string]string
}

func New() *Studio {
	return &Studio{Identities: map[string]Identity{}, Specs: map[string]Spec{}, Renders: map[string]*Render{}, Assets: map[string]*Asset{}, Verdicts: nil, ImportHashes: map[string]string{}}
}

func (s *Studio) Author(identity Identity, actorID string, actor ActorKind, now time.Time) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	if identity.Version == 0 {
		identity.Version = 1
	}
	identity.Revisions = append(identity.Revisions, Revision{actorID, actor, now})
	s.Identities[identity.ID] = identity
	return nil
}

func (s *Studio) Revise(identity Identity, actorID string, actor ActorKind, now time.Time) (Identity, error) {
	old, ok := s.Identities[identity.ID]
	if !ok {
		return Identity{}, fmt.Errorf("identity %q not found", identity.ID)
	}
	if old.Referenced {
		identity.Version = old.Version + 1
	} else {
		identity.Version = old.Version
	}
	if err := s.Author(identity, actorID, actor, now); err != nil {
		return Identity{}, err
	}
	return identity, nil
}

func (s *Studio) Select(assetID string) error {
	asset, ok := s.Assets[assetID]
	if !ok {
		return fmt.Errorf("asset %q not found", assetID)
	}
	render := s.Renders[asset.RenderID]
	if render == nil {
		return fmt.Errorf("render %q not found", asset.RenderID)
	}
	for _, candidateID := range render.AssetIDs {
		candidate := s.Assets[candidateID]
		if candidateID == assetID {
			candidate.Status = InReview
		} else {
			candidate.Status = Discarded
		}
	}
	return nil
}

func (s *Studio) Judge(verdict Verdict) error {
	if verdict.ActorKind != Operator {
		return errors.New("conformance verdict requires operator actor")
	}
	switch verdict.Basis {
	case ReferenceSheet, ReferenceImageSet, ConditioningArtifact, ProseOnly:
	default:
		return errors.New("conformance verdict requires a recognised basis")
	}
	verdict.At = verdict.At.UTC()
	s.Verdicts = append(s.Verdicts, verdict)
	return nil
}

func (s *Studio) Release(assetID string) error {
	asset := s.Assets[assetID]
	if asset == nil {
		return fmt.Errorf("asset %q not found", assetID)
	}
	if asset.Status == Discarded {
		return &ReleaseError{CauseDiscarded, assetID}
	}
	if asset.Status != InReview {
		return &ReleaseError{CauseNotSelected, assetID}
	}
	if strings.TrimSpace(asset.AltText) == "" {
		return &ReleaseError{CauseAltText, assetID}
	}
	if strings.TrimSpace(asset.Disclosure) == "" {
		return &ReleaseError{CauseDisclosure, assetID}
	}
	if strings.TrimSpace(asset.CredentialClaims) != "" {
		return &ReleaseError{CauseCredentialClaims, assetID}
	}
	for _, identityID := range asset.IdentityVersionIDs {
		found := false
		for _, verdict := range s.Verdicts {
			if verdict.AssetID == assetID && verdict.IdentityVersionID == identityID {
				found = true
				if !verdict.Passed {
					return &ReleaseError{CauseFailed, identityID}
				}
				break
			}
		}
		if !found {
			return &ReleaseError{CauseUnresolved, identityID}
		}
	}
	asset.Status = Released
	for _, identityID := range asset.IdentityVersionIDs {
		identity := s.Identities[identityID]
		identity.Referenced = true
		s.Identities[identityID] = identity
	}
	return nil
}
