package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const RightsEvaluatorVersion = "model-rights/v1"

const (
	RuntimeProviderOllama            = "ollama"
	RuntimeProviderOllaya            = "ollaya"
	RuntimeDigestOllamaModelContents = "ollama_model_contents"
	RuntimeDigestOllayaManifest      = "ollaya_registry_manifest"
	RuntimeSourceOllamaTags          = "ollama /api/tags"
	RuntimeSourceOllayaTags          = "ollaya /api/tags"
)

type RightsStatus string

const (
	RightsAllowed RightsStatus = "allowed"
	RightsDenied  RightsStatus = "denied"
	RightsUnknown RightsStatus = "unknown"
)

type RightsUse string

const (
	RightsUseLocalInference      RightsUse = "local_inference"
	RightsUseHostedInference     RightsUse = "hosted_inference"
	RightsUseRedistributeWeights RightsUse = "redistribute_weights"
	RightsUseRedistributeDerived RightsUse = "redistribute_derivative"
	RightsUseDistributeRuntime   RightsUse = "distribute_runtime"
)

type RightsClassification string

const (
	RightsUnrestricted RightsClassification = "unrestricted"
	RightsConditional  RightsClassification = "conditional"
	RightsProhibited   RightsClassification = "prohibited"
	RightsUnverified   RightsClassification = "unknown"
)

type RightsEvidence struct {
	URL        string `json:"url"`
	Revision   string `json:"revision"`
	SHA256     string `json:"sha256"`
	ObservedAt string `json:"observed_at"`
}

type RightsCondition struct {
	ID             string  `json:"id"`
	Kind           string  `json:"kind"`
	Comparator     string  `json:"comparator"`
	Threshold      float64 `json:"threshold"`
	Unit           string  `json:"unit"`
	EntityScope    string  `json:"entity_scope"`
	Interpretation string  `json:"interpretation"`
}

type ArtifactComponent struct {
	Kind     string `json:"kind"`
	Identity string `json:"identity"`
	Revision string `json:"revision"`
	SHA256   string `json:"sha256,omitempty"`
}

// ArtifactRights are publisher/source facts attached to one concrete model
// artifact. They do not contain operator business context or a cached verdict.
type ArtifactRights struct {
	ArtifactID     string               `json:"artifact_id"`
	Revision       string               `json:"revision"`
	SHA256         string               `json:"sha256"`
	LicenseID      string               `json:"license_id"`
	Classification RightsClassification `json:"classification"`
	Evidence       RightsEvidence       `json:"evidence"`
	Conditions     []RightsCondition    `json:"conditions,omitempty"`
	Obligations    []string             `json:"obligations,omitempty"`
	Lineage        []ArtifactComponent  `json:"lineage,omitempty"`
}

// RuntimeArtifactIdentity is an owner-supplied mapping from the publisher's
// artifact identity to a concrete provider digest domain. The resource adapter
// verifies the observed runtime digest before allowing inference; callers may
// not infer equivalence from the publisher artifact SHA alone.
type RuntimeArtifactIdentity struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	ArtifactID       string `json:"artifact_id"`
	ArtifactRevision string `json:"artifact_revision"`
	DigestKind       string `json:"digest_kind"`
	Digest           string `json:"digest"`
	Source           string `json:"source"`
	ObservedAt       string `json:"observed_at"`
}

// RightsContext is supplied by a trusted owner, never by an inference request.
// RevenueStatus deliberately represents the operator's stated category without
// fabricating a numeric amount or accounting period.
type RightsContext struct {
	ID            string `json:"id"`
	Version       string `json:"version"`
	EntityScope   string `json:"entity_scope"`
	RevenueStatus string `json:"revenue_status"`
	AsOf          string `json:"as_of"`
	Source        string `json:"source"`
}

type RightsDecision struct {
	Status            RightsStatus        `json:"status"`
	Reason            string              `json:"reason"`
	ArtifactID        string              `json:"artifact_id,omitempty"`
	ArtifactSHA256    string              `json:"artifact_sha256,omitempty"`
	RightsFactsSHA256 string              `json:"rights_facts_sha256,omitempty"`
	LicenseID         string              `json:"license_id,omitempty"`
	Evidence          RightsEvidence      `json:"evidence,omitempty"`
	Conditions        []RightsCondition   `json:"conditions,omitempty"`
	Lineage           []ArtifactComponent `json:"lineage,omitempty"`
	Use               RightsUse           `json:"use"`
	ContextID         string              `json:"context_id,omitempty"`
	ContextVersion    string              `json:"context_version,omitempty"`
	ContextSHA256     string              `json:"context_sha256,omitempty"`
	Obligations       []string            `json:"obligations,omitempty"`
	ReviewTriggers    []string            `json:"review_triggers"`
	EvaluatorVersion  string              `json:"evaluator_version"`
	EvaluatedAt       string              `json:"evaluated_at"`
	DecisionSHA256    string              `json:"decision_sha256"`
}

func EvaluateRights(artifact *ArtifactRights, use RightsUse, context RightsContext, now time.Time) RightsDecision {
	d := RightsDecision{
		Status: RightsUnknown, Reason: "rights_facts_incomplete", Use: use,
		ContextID: strings.TrimSpace(context.ID), ContextVersion: strings.TrimSpace(context.Version),
		ReviewTriggers:   []string{"artifact_changed", "license_source_changed", "organization_context_changed", "use_changed", "review_due"},
		EvaluatorVersion: RightsEvaluatorVersion, EvaluatedAt: now.UTC().Format(time.RFC3339Nano),
	}
	if artifact == nil {
		return bindDecision(d)
	}
	d.ArtifactID, d.ArtifactSHA256, d.LicenseID, d.Evidence = artifact.ArtifactID, artifact.SHA256, artifact.LicenseID, artifact.Evidence
	d.RightsFactsSHA256 = canonicalDigest(artifact)
	d.Conditions = append([]RightsCondition(nil), artifact.Conditions...)
	d.Lineage = append([]ArtifactComponent(nil), artifact.Lineage...)
	d.Obligations = append([]string(nil), artifact.Obligations...)
	if validRightsContext(context) {
		d.ContextSHA256 = canonicalDigest(context)
	}
	if !validRightsArtifact(artifact) || !validRightsContext(context) || !validRightsUse(use) {
		return bindDecision(d)
	}
	for _, condition := range artifact.Conditions {
		if strings.TrimSpace(condition.ID) == "" || strings.TrimSpace(condition.Interpretation) == "" {
			return bindDecision(d)
		}
	}
	if artifact.Classification == RightsProhibited {
		d.Status, d.Reason = RightsDenied, "license_prohibits_use"
		return bindDecision(d)
	}
	if artifact.Classification == RightsUnverified {
		return bindDecision(d)
	}
	if artifact.Classification == RightsUnrestricted {
		d.Status, d.Reason = RightsAllowed, "verified_unrestricted_grant"
		return bindDecision(d)
	}
	if artifact.Classification != RightsConditional {
		return bindDecision(d)
	}
	if use == RightsUseRedistributeWeights || use == RightsUseRedistributeDerived || use == RightsUseDistributeRuntime {
		d.Reason = "distribution_obligations_require_owner_attestation"
		return bindDecision(d)
	}
	if len(artifact.Conditions) == 0 {
		return bindDecision(d)
	}
	for _, condition := range artifact.Conditions {
		if condition.Kind != "annual_revenue" || condition.Comparator != "lt" || condition.Unit != "USD" || condition.Threshold <= 0 || condition.EntityScope != context.EntityScope {
			return bindDecision(d)
		}
		switch context.RevenueStatus {
		case "pre_revenue", "below_threshold":
			// The publisher text is internally ambiguous at the boundary; the
			// catalog's conservative interpretation rechecks at $10M.
			d.Status, d.Reason = RightsAllowed, "condition_satisfied_by_current_stated_context"
		case "at_or_above_threshold":
			d.Status, d.Reason = RightsDenied, "revenue_condition_not_satisfied"
		default:
			return bindDecision(d)
		}
	}
	return bindDecision(d)
}

func validRightsArtifact(a *ArtifactRights) bool {
	if strings.TrimSpace(a.ArtifactID) == "" || strings.TrimSpace(a.Revision) == "" || strings.TrimSpace(a.LicenseID) == "" || !isSHA256(a.SHA256) {
		return false
	}
	if strings.TrimSpace(a.Evidence.URL) == "" || strings.TrimSpace(a.Evidence.Revision) == "" || !isSHA256(a.Evidence.SHA256) {
		return false
	}
	if _, err := time.Parse("2006-01-02", a.Evidence.ObservedAt); err != nil {
		return false
	}
	return true
}

func validRightsContext(c RightsContext) bool {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Version) == "" || strings.TrimSpace(c.EntityScope) == "" || strings.TrimSpace(c.RevenueStatus) == "" || strings.TrimSpace(c.Source) == "" {
		return false
	}
	_, err := time.Parse("2006-01-02", c.AsOf)
	return err == nil
}

func validRightsUse(use RightsUse) bool {
	switch use {
	case RightsUseLocalInference, RightsUseHostedInference, RightsUseRedistributeWeights, RightsUseRedistributeDerived, RightsUseDistributeRuntime:
		return true
	default:
		return false
	}
}

// RequireCurrentRightsDecision verifies the receipt and recomputes its result
// from the artifact, use, and trusted context the caller has now. A valid
// digest only proves that the receipt was not edited; it does not make the
// inputs it records current.
func RequireCurrentRightsDecision(decision RightsDecision, artifact *ArtifactRights, use RightsUse, currentContext RightsContext) error {
	if err := VerifyRightsDecision(decision); err != nil {
		return errors.New("local model rights decision is not verifiable")
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, decision.EvaluatedAt)
	if err != nil {
		return errors.New("local model rights decision has an invalid evaluation time")
	}
	current := EvaluateRights(artifact, use, currentContext, evaluatedAt)
	if current.DecisionSHA256 != decision.DecisionSHA256 {
		return errors.New("local model rights decision is stale or does not match the current artifact, use, or context")
	}
	switch decision.Status {
	case RightsAllowed:
		return nil
	case RightsDenied:
		return errors.New("local model use is denied by rights policy")
	default:
		return errors.New("local model rights are unknown")
	}
}

// RuntimeArtifactIdentityMatches requires an owner-supplied mapping from the
// legal artifact revision to the observed runtime digest. Each supported
// provider has one explicit digest domain and observed source; name equality
// alone never establishes identity.
func RuntimeArtifactIdentityMatches(binding *RuntimeArtifactIdentity, rights *ArtifactRights, provider, model, observedDigest string) bool {
	expectedDigestKind, expectedSource := runtimeDigestDomain(provider)
	if expectedDigestKind == "" || binding == nil || rights == nil || binding.Provider != provider || binding.Model != model ||
		binding.ArtifactID != rights.ArtifactID || binding.ArtifactRevision != rights.Revision ||
		binding.DigestKind != expectedDigestKind || binding.Source != expectedSource ||
		strings.TrimSpace(binding.ObservedAt) == "" || binding.Digest == "" || binding.Digest != observedDigest {
		return false
	}
	decoded, err := hex.DecodeString(binding.Digest)
	if err != nil || len(decoded) != 32 {
		return false
	}
	_, err = time.Parse("2006-01-02", binding.ObservedAt)
	return err == nil
}

func runtimeDigestDomain(provider string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case RuntimeProviderOllama:
		return RuntimeDigestOllamaModelContents, RuntimeSourceOllamaTags
	case RuntimeProviderOllaya:
		return RuntimeDigestOllayaManifest, RuntimeSourceOllayaTags
	default:
		return "", ""
	}
}

func isSHA256(value string) bool {
	b, err := hex.DecodeString(strings.TrimSpace(value))
	return err == nil && len(b) == sha256.Size
}

func canonicalDigest(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func bindDecision(d RightsDecision) RightsDecision {
	d.DecisionSHA256 = ""
	data, err := json.Marshal(d)
	if err != nil {
		d.Status, d.Reason = RightsUnknown, fmt.Sprintf("decision_binding_failed: %v", err)
		return d
	}
	sum := sha256.Sum256(data)
	d.DecisionSHA256 = hex.EncodeToString(sum[:])
	return d
}

func VerifyRightsDecision(decision RightsDecision) error {
	if decision.DecisionSHA256 == "" {
		return errors.New("rights decision digest is missing")
	}
	want := decision.DecisionSHA256
	got := bindDecision(decision).DecisionSHA256
	if got != want {
		return errors.New("rights decision facts changed")
	}
	return nil
}
