package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const RightsEvaluatorVersion = "model-rights/v2"

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
	// RightsBehavioralRestrictions permits only uses that remain subject to
	// publisher-defined behavioral terms. Those terms are surfaced as
	// obligations; redistribution stays unknown until its owner attests to the
	// downstream terms and notices.
	RightsBehavioralRestrictions RightsClassification = "behavioral_restrictions"
	RightsProhibited             RightsClassification = "prohibited"
	RightsUnverified             RightsClassification = "unknown"
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
	Period         string  `json:"period,omitempty"`
}

// ConditionalGrant states which uses a source license permits when every
// listed condition is satisfied. Keeping the uses and conditions together
// prevents a condition for one use from silently granting another.
type ConditionalGrant struct {
	ID          string            `json:"id"`
	Uses        []RightsUse       `json:"uses"`
	Conditions  []RightsCondition `json:"conditions"`
	Obligations []string          `json:"obligations,omitempty"`
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
	ArtifactID        string               `json:"artifact_id"`
	Revision          string               `json:"revision"`
	SHA256            string               `json:"sha256"`
	LicenseID         string               `json:"license_id"`
	Classification    RightsClassification `json:"classification"`
	UnverifiedReason  string               `json:"unverified_reason,omitempty"`
	Evidence          RightsEvidence       `json:"evidence"`
	Conditions        []RightsCondition    `json:"conditions,omitempty"`
	ConditionalGrants []ConditionalGrant   `json:"conditional_grants,omitempty"`
	Obligations       []string             `json:"obligations,omitempty"`
	ReviewTriggers    []string             `json:"review_triggers,omitempty"`
	Lineage           []ArtifactComponent  `json:"lineage,omitempty"`
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
// RevenueStatus preserves the operator's stated category. AnnualRevenueUSD is
// optional and is set only when an exact, owner-sourced amount is available;
// it lets different license thresholds be evaluated against the same context.
type RightsContext struct {
	ID                         string            `json:"id"`
	Version                    string            `json:"version"`
	EntityScope                string            `json:"entity_scope"`
	RevenueStatus              string            `json:"revenue_status"`
	AnnualRevenueUSD           *float64          `json:"annual_revenue_usd,omitempty"`
	MonthlyActiveUsers         map[string]string `json:"monthly_active_users,omitempty"`
	MonthlyActiveUsersEvidence map[string]string `json:"monthly_active_users_evidence,omitempty"`
	AsOf                       string            `json:"as_of"`
	Source                     string            `json:"source"`
}

type RightsDecision struct {
	Status            RightsStatus        `json:"status"`
	Reason            string              `json:"reason"`
	ArtifactID        string              `json:"artifact_id,omitempty"`
	ArtifactSHA256    string              `json:"artifact_sha256,omitempty"`
	RightsFactsSHA256 string              `json:"rights_facts_sha256,omitempty"`
	LicenseID         string              `json:"license_id,omitempty"`
	UnverifiedReason  string              `json:"unverified_reason,omitempty"`
	Evidence          RightsEvidence      `json:"evidence,omitempty"`
	Conditions        []RightsCondition   `json:"conditions,omitempty"`
	ConditionalGrants []ConditionalGrant  `json:"conditional_grants,omitempty"`
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
	d.UnverifiedReason = strings.TrimSpace(artifact.UnverifiedReason)
	d.RightsFactsSHA256 = canonicalDigest(artifact)
	d.Conditions = append([]RightsCondition(nil), artifact.Conditions...)
	d.ConditionalGrants = cloneConditionalGrants(artifact.ConditionalGrants)
	if len(artifact.ReviewTriggers) > 0 {
		d.ReviewTriggers = append([]string(nil), artifact.ReviewTriggers...)
	}
	d.Lineage = append([]ArtifactComponent(nil), artifact.Lineage...)
	d.Obligations = append([]string(nil), artifact.Obligations...)
	if validRightsContext(context) {
		d.ContextSHA256 = canonicalDigest(context)
	}
	if !validRightsArtifact(artifact) || !validRightsContext(context) || !validRightsUse(use) {
		return bindDecision(d)
	}
	for _, condition := range artifact.Conditions {
		if !validRightsCondition(condition) {
			return bindDecision(d)
		}
	}
	if artifact.Classification == RightsProhibited {
		d.Status, d.Reason = RightsDenied, "license_prohibits_use"
		return bindDecision(d)
	}
	if artifact.Classification == RightsUnverified {
		d.Reason = "rights_evidence_unverified"
		return bindDecision(d)
	}
	if artifact.Classification == RightsUnrestricted {
		d.Status, d.Reason = RightsAllowed, "verified_unrestricted_grant"
		return bindDecision(d)
	}
	if artifact.Classification == RightsBehavioralRestrictions {
		if len(artifact.Obligations) == 0 {
			return bindDecision(d)
		}
		if use == RightsUseLocalInference {
			d.Status, d.Reason = RightsAllowed, "behavioral_use_restrictions_apply"
			return bindDecision(d)
		}
		if use == RightsUseRedistributeWeights || use == RightsUseRedistributeDerived || use == RightsUseDistributeRuntime {
			d.Reason = "distribution_obligations_require_owner_attestation"
		}
		return bindDecision(d)
	}
	if artifact.Classification != RightsConditional {
		return bindDecision(d)
	}
	if len(artifact.ConditionalGrants) > 0 {
		matchedUse := false
		unknownCondition := false
		for _, grant := range artifact.ConditionalGrants {
			if !containsRightsUse(grant.Uses, use) {
				continue
			}
			matchedUse = true
			grantStatus, reason := evaluateConditions(grant.Conditions, context)
			if grantStatus == RightsAllowed {
				d.Status, d.Reason = RightsAllowed, reason
				d.Obligations = appendUniqueStrings(d.Obligations, grant.Obligations...)
				return bindDecision(d)
			}
			if grantStatus == RightsUnknown {
				unknownCondition = true
			} else {
				d.Reason = reason
			}
		}
		if !matchedUse {
			d.Status, d.Reason = RightsDenied, "conditional_grant_does_not_cover_use"
		} else if unknownCondition {
			d.Status, d.Reason = RightsUnknown, "conditional_grant_condition_unknown"
		} else {
			d.Status = RightsDenied
		}
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
		if condition.Kind != "annual_revenue" || condition.Comparator != "lt" || condition.Unit != "USD" || condition.Threshold <= 0 || strings.TrimSpace(condition.EntityScope) == "" || condition.EntityScope != context.EntityScope {
			return bindDecision(d)
		}
		if context.AnnualRevenueUSD != nil {
			if *context.AnnualRevenueUSD < condition.Threshold {
				d.Status, d.Reason = RightsAllowed, "annual_revenue_condition_satisfied"
			} else {
				d.Status, d.Reason = RightsDenied, "revenue_condition_not_satisfied"
				return bindDecision(d)
			}
			continue
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

func evaluateConditions(conditions []RightsCondition, context RightsContext) (RightsStatus, string) {
	if len(conditions) == 0 {
		return RightsUnknown, "conditional_grant_has_no_conditions"
	}
	for _, condition := range conditions {
		if strings.TrimSpace(condition.ID) == "" || strings.TrimSpace(condition.Interpretation) == "" ||
			condition.Threshold <= 0 || strings.TrimSpace(condition.EntityScope) == "" || condition.EntityScope != context.EntityScope {
			return RightsUnknown, "conditional_grant_condition_unknown"
		}
		switch condition.Kind {
		case "annual_revenue":
			if condition.Comparator != "lt" || condition.Unit != "USD" || condition.Period != "" {
				return RightsUnknown, "conditional_grant_condition_unknown"
			}
			if context.AnnualRevenueUSD != nil {
				if *context.AnnualRevenueUSD < condition.Threshold {
					continue
				}
				return RightsDenied, "revenue_condition_not_satisfied"
			}
			switch context.RevenueStatus {
			case "pre_revenue", "below_threshold":
			case "at_or_above_threshold":
				return RightsDenied, "revenue_condition_not_satisfied"
			default:
				return RightsUnknown, "conditional_grant_condition_unknown"
			}
		case "monthly_active_users":
			if condition.Comparator != "lte" || condition.Unit != "users" || !validYearMonth(condition.Period) {
				return RightsUnknown, "conditional_grant_condition_unknown"
			}
			fact := context.MonthlyActiveUsers[condition.Period]
			switch fact {
			case "at_or_below_threshold", "below_threshold":
			case "above_threshold":
				return RightsDenied, "monthly_active_users_condition_not_satisfied"
			default:
				users, err := strconv.ParseUint(fact, 10, 64)
				if err != nil {
					return RightsUnknown, "conditional_grant_condition_unknown"
				}
				if float64(users) > condition.Threshold {
					return RightsDenied, "monthly_active_users_condition_not_satisfied"
				}
			}
		default:
			return RightsUnknown, "conditional_grant_condition_unknown"
		}
	}
	return RightsAllowed, "condition_satisfied_by_current_stated_context"
}

func validYearMonth(value string) bool {
	_, err := time.Parse("2006-01", strings.TrimSpace(value))
	return err == nil
}

func containsRightsUse(uses []RightsUse, target RightsUse) bool {
	for _, use := range uses {
		if use == target {
			return true
		}
	}
	return false
}

func appendUniqueStrings(target []string, values ...string) []string {
	seen := make(map[string]struct{}, len(target)+len(values))
	for _, value := range target {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			target = append(target, value)
			seen[value] = struct{}{}
		}
	}
	return target
}

func cloneConditionalGrants(grants []ConditionalGrant) []ConditionalGrant {
	if len(grants) == 0 {
		return nil
	}
	cloned := make([]ConditionalGrant, len(grants))
	for i, grant := range grants {
		cloned[i] = grant
		cloned[i].Uses = append([]RightsUse(nil), grant.Uses...)
		cloned[i].Conditions = append([]RightsCondition(nil), grant.Conditions...)
		cloned[i].Obligations = append([]string(nil), grant.Obligations...)
	}
	return cloned
}

func validRightsArtifact(a *ArtifactRights) bool {
	if a.Classification == RightsUnverified && strings.TrimSpace(a.UnverifiedReason) != "" {
		// Preserve a sourced unknown without inventing an artifact digest or
		// license revision. This shape can only return RightsUnknown below.
		if strings.TrimSpace(a.ArtifactID) == "" || len(a.Conditions) != 0 || len(a.ConditionalGrants) != 0 {
			return false
		}
		for _, trigger := range a.ReviewTriggers {
			if strings.TrimSpace(trigger) == "" {
				return false
			}
		}
		return true
	}
	if a.Classification == RightsBehavioralRestrictions && len(a.Obligations) == 0 {
		return false
	}
	if strings.TrimSpace(a.ArtifactID) == "" || strings.TrimSpace(a.Revision) == "" || strings.TrimSpace(a.LicenseID) == "" || !isSHA256(a.SHA256) {
		return false
	}
	if strings.TrimSpace(a.Evidence.URL) == "" || strings.TrimSpace(a.Evidence.Revision) == "" ||
		(!isSHA256(a.Evidence.SHA256) && !isImmutableRevision(a.Evidence.Revision)) {
		return false
	}
	if _, err := time.Parse("2006-01-02", a.Evidence.ObservedAt); err != nil {
		return false
	}
	for _, trigger := range a.ReviewTriggers {
		if strings.TrimSpace(trigger) == "" {
			return false
		}
	}
	for _, grant := range a.ConditionalGrants {
		if strings.TrimSpace(grant.ID) == "" || len(grant.Uses) == 0 || len(grant.Conditions) == 0 {
			return false
		}
		for _, use := range grant.Uses {
			if !validRightsUse(use) {
				return false
			}
		}
		for _, condition := range grant.Conditions {
			if !validRightsCondition(condition) {
				return false
			}
		}
	}
	return true
}

func validRightsCondition(condition RightsCondition) bool {
	if strings.TrimSpace(condition.ID) == "" || strings.TrimSpace(condition.Interpretation) == "" || condition.Threshold <= 0 || strings.TrimSpace(condition.EntityScope) == "" {
		return false
	}
	switch condition.Kind {
	case "annual_revenue":
		return condition.Comparator == "lt" && condition.Unit == "USD" && condition.Period == ""
	case "monthly_active_users":
		return condition.Comparator == "lte" && condition.Unit == "users" && validYearMonth(condition.Period)
	default:
		return false
	}
}

// isImmutableRevision accepts full Git object revisions and explicit SHA-256
// revision identifiers. A revision-pinned source URL remains exact evidence
// even when a separate content digest has not been captured.
func isImmutableRevision(value string) bool {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validRightsContext(c RightsContext) bool {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Version) == "" || strings.TrimSpace(c.RevenueStatus) == "" || strings.TrimSpace(c.Source) == "" {
		return false
	}
	if (c.RevenueStatus == "exact_value") != (c.AnnualRevenueUSD != nil) {
		return false
	}
	if c.AnnualRevenueUSD != nil && (*c.AnnualRevenueUSD < 0 || math.IsNaN(*c.AnnualRevenueUSD) || math.IsInf(*c.AnnualRevenueUSD, 0)) {
		return false
	}
	_, err := time.Parse("2006-01-02", c.AsOf)
	if err != nil {
		return false
	}
	for period, status := range c.MonthlyActiveUsers {
		if !validYearMonth(period) {
			return false
		}
		switch status {
		case "below_threshold", "at_or_below_threshold", "above_threshold":
		default:
			if _, err := strconv.ParseUint(status, 10, 64); err != nil || strings.TrimSpace(c.MonthlyActiveUsersEvidence[period]) == "" {
				return false
			}
		}
	}
	return true
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
