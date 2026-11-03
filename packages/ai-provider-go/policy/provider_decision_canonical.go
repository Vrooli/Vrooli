package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

const ProviderDecisionSchemaVersion uint32 = 1

const (
	ProviderDecisionScopeRule      = "provider_rule"
	ProviderDecisionScopeAgreement = "provider_agreement"
)

type canonicalProviderDecision struct {
	SchemaVersion uint32           `json:"schema_version"`
	Scope         string           `json:"scope"`
	Supplier      string           `json:"supplier"`
	Terms         *ProviderTerms   `json:"terms,omitempty"`
	Models        []canonicalModel `json:"models,omitempty"`
	Agreement     *Agreement       `json:"agreement,omitempty"`
}

type canonicalModel struct {
	Name string    `json:"name"`
	Rule ModelRule `json:"rule"`
}

// CanonicalProviderRuleBytes returns the versioned bytes whose digest is
// attached to a Notification Hub provider-rule Ask. Review status and its
// decision reference are excluded because the Ask itself establishes those
// facts; all supplier terms, model overrides, clauses, evidence and windows
// remain bound by the digest.
func CanonicalProviderRuleBytes(supplier string, terms ProviderTerms, models map[string]ModelRule) ([]byte, error) {
	supplier = strings.TrimSpace(supplier)
	if supplier == "" {
		return nil, errors.New("provider decision supplier is required")
	}
	normalizedTerms := normalizeCanonicalTerms(terms)
	modelRules := make(map[string]ModelRule, len(models))
	for name := range models {
		normalizedName := strings.TrimSpace(name)
		if normalizedName == "" {
			return nil, errors.New("provider decision model names must be non-empty")
		}
		if _, exists := modelRules[normalizedName]; exists {
			return nil, errors.New("provider decision model names collide after trimming")
		}
		modelRules[normalizedName] = models[name]
	}
	modelNames := make([]string, 0, len(modelRules))
	for name := range modelRules {
		modelNames = append(modelNames, name)
	}
	sort.Strings(modelNames)
	canonicalModels := make([]canonicalModel, 0, len(modelNames))
	for _, name := range modelNames {
		canonicalModels = append(canonicalModels, canonicalModel{Name: name, Rule: normalizeCanonicalModelRule(modelRules[name])})
	}
	return json.Marshal(canonicalProviderDecision{
		SchemaVersion: ProviderDecisionSchemaVersion,
		Scope:         ProviderDecisionScopeRule,
		Supplier:      supplier,
		Terms:         &normalizedTerms,
		Models:        canonicalModels,
	})
}

func ProviderRuleDigest(supplier string, terms ProviderTerms, models map[string]ModelRule) (string, error) {
	bytes, err := CanonicalProviderRuleBytes(supplier, terms, models)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), nil
}

// CanonicalProviderAgreementBytes returns the versioned bytes bound to a
// Notification Hub provider-agreement Ask. Approval references and the
// private verified bit are excluded; grant scope, trust facts, model scope,
// and effective windows are included.
func CanonicalProviderAgreementBytes(agreement Agreement) ([]byte, error) {
	agreement.Supplier = strings.TrimSpace(agreement.Supplier)
	if agreement.Supplier == "" {
		return nil, errors.New("provider decision supplier is required")
	}
	agreement.DecisionRef = ""
	agreement.Trust.DecisionRef = ""
	agreement.Trust.Facts = sortedTrimmed(agreement.Trust.Facts)
	agreement.UseClasses = sortedUseClasses(agreement.UseClasses)
	agreement.Models = sortedTrimmed(agreement.Models)
	if !agreement.EffectiveAt.IsZero() {
		agreement.EffectiveAt = agreement.EffectiveAt.UTC().Round(0)
	}
	if !agreement.ExpiresAt.IsZero() {
		agreement.ExpiresAt = agreement.ExpiresAt.UTC().Round(0)
	}
	return json.Marshal(canonicalProviderDecision{
		SchemaVersion: ProviderDecisionSchemaVersion,
		Scope:         ProviderDecisionScopeAgreement,
		Supplier:      agreement.Supplier,
		Agreement:     &agreement,
	})
}

func ProviderAgreementDigest(agreement Agreement) (string, error) {
	bytes, err := CanonicalProviderAgreementBytes(agreement)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeCanonicalTerms(terms ProviderTerms) ProviderTerms {
	terms.Review = Review{}
	terms.PlanRules = append([]PlanRule(nil), terms.PlanRules...)
	for i := range terms.PlanRules {
		terms.PlanRules[i].Plan = strings.TrimSpace(terms.PlanRules[i].Plan)
		terms.PlanRules[i].Conditions = sortedTrimmed(terms.PlanRules[i].Conditions)
		terms.PlanRules[i].UseClasses = cloneUsePermissions(terms.PlanRules[i].UseClasses)
	}
	sort.Slice(terms.PlanRules, func(i, j int) bool {
		if terms.PlanRules[i].Plan != terms.PlanRules[j].Plan {
			return terms.PlanRules[i].Plan < terms.PlanRules[j].Plan
		}
		left, _ := json.Marshal(terms.PlanRules[i])
		right, _ := json.Marshal(terms.PlanRules[j])
		return string(left) < string(right)
	})
	terms.Downstream.PinnedHosts = sortedTrimmed(terms.Downstream.PinnedHosts)
	return terms
}

func normalizeCanonicalModelRule(rule ModelRule) ModelRule {
	if rule.TermsOverride != nil {
		copyOverride := *rule.TermsOverride
		copyOverride.UseClasses = cloneUsePermissions(copyOverride.UseClasses)
		rule.TermsOverride = &copyOverride
	}
	return rule
}

func cloneUsePermissions(in map[UseClass]Permission) map[UseClass]Permission {
	if in == nil {
		return nil
	}
	out := make(map[UseClass]Permission, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sortedTrimmed(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		out = append(out, strings.TrimSpace(item))
	}
	sort.Strings(out)
	return out
}

func sortedUseClasses(in []UseClass) []UseClass {
	out := append([]UseClass(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
