package providers

import (
	"fmt"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// basisByName maps the wire string a provider emits for the attestation `basis`
// onto the AttestedAnswer.Basis enum. Mirrors the canonical contract in
// meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md.
var basisByName = map[string]commonv1.Basis{
	"derived":             commonv1.Basis_BASIS_DERIVED,
	"validated":           commonv1.Basis_BASIS_VALIDATED,
	"declared_unverified": commonv1.Basis_BASIS_DECLARED_UNVERIFIED,
	"contradicted":        commonv1.Basis_BASIS_CONTRADICTED,
	"absent":              commonv1.Basis_BASIS_ABSENT,
}

var sufficiencyByName = map[string]commonv1.Sufficiency{
	"full":         commonv1.Sufficiency_SUFFICIENCY_FULL,
	"partial":      commonv1.Sufficiency_SUFFICIENCY_PARTIAL,
	"insufficient": commonv1.Sufficiency_SUFFICIENCY_INSUFFICIENT,
}

// decodeAttestation decodes the per-item attestation object at `path` into a
// SearchHit.AttestedAnswer, following the fixed contract keys an architectural
// provider emits (claim, citations, basis, sufficiency, gaps,
// suggested_follow_ups). It returns nil when `path` is empty (every ordinary
// provider), resolves to no object, or the decoded answer is non-conformant
// (ValidateAttestation fails) — a malformed/untrustworthy attestation is dropped
// rather than carried, so consumers never see a DERIVED claim with no provenance.
// This is the generic carrier: zero provider-specific code, the descriptor's
// attestation_field is the only switch.
func decodeAttestation(item map[string]any, path string) *commonv1.AttestedAnswer {
	if path == "" {
		return nil
	}
	obj, ok := lookupPath(item, path).(map[string]any)
	if !ok || len(obj) == 0 {
		return nil
	}
	att := &commonv1.AttestedAnswer{
		Claim:       coerceString(obj["claim"]),
		Basis:       basisByName[strings.ToLower(coerceString(obj["basis"]))],
		Sufficiency: sufficiencyByName[strings.ToLower(coerceString(obj["sufficiency"]))],
	}
	if cites, ok := obj["citations"].([]any); ok {
		for _, c := range cites {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			locator := coerceString(cm["locator"])
			if locator == "" {
				continue
			}
			att.Citations = append(att.Citations, &commonv1.Citation{
				Locator: locator,
				Kind:    coerceString(cm["kind"]),
				Note:    coerceString(cm["note"]),
			})
		}
	}
	att.Gaps = coerceStringSlice(obj["gaps"])
	att.SuggestedFollowUps = coerceStringSlice(obj["suggested_follow_ups"])

	if err := ValidateAttestation(att); err != nil {
		// Non-conformant: drop it. Trust must be earned by provenance.
		return nil
	}
	return att
}

// ValidateAttestation is the attestation conformance lint: an attested answer
// must be internally honest. The keystone rule is that a DERIVED claim (asserted
// to be computed directly from code) MUST carry at least one citation — an
// uncited "I derived this from the code" is exactly the overclaim the contract
// exists to prevent. VALIDATED and CONTRADICTED likewise reference code and so
// also require a citation. DECLARED_UNVERIFIED and ABSENT may be uncited (a
// pointer-only or unverifiable answer). A claim is always required.
//
// Exposed so a provider-registration conformance test can gate architectural
// providers, and so the adapter can drop a non-conformant runtime attestation.
func ValidateAttestation(a *commonv1.AttestedAnswer) error {
	if a == nil {
		return fmt.Errorf("attestation: nil")
	}
	if strings.TrimSpace(a.GetClaim()) == "" {
		return fmt.Errorf("attestation: empty claim")
	}
	switch a.GetBasis() {
	case commonv1.Basis_BASIS_DERIVED, commonv1.Basis_BASIS_VALIDATED, commonv1.Basis_BASIS_CONTRADICTED:
		if len(a.GetCitations()) == 0 {
			return fmt.Errorf("attestation: basis %s requires at least one citation", a.GetBasis())
		}
	}
	return nil
}

// coerceStringSlice coerces a decoded JSON array into a non-empty []string,
// dropping empty entries. Returns nil for a non-array or empty result.
func coerceStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, e := range arr {
		if s := coerceString(e); s != "" {
			out = append(out, s)
		}
	}
	return out
}
