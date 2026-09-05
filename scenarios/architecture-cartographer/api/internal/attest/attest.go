// Package attest is the single chokepoint that maps cartographer's internal
// evidence, provenance, and convergence verdicts onto the shared honesty
// contract common.v1.AttestedAnswer. Every capability (domains/zones/slice/
// archetype + drift findings) builds its attestation here so direct CLI/RPC use
// and the future federated search present identically.
//
// The basis vocabulary is uniform across capabilities (see the plan's Contract
// Decisions and meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md):
//
//	code-computed                 -> DERIVED
//	doc claim present == code      -> VALIDATED
//	doc claim present != code      -> CONTRADICTED (cite both sides)
//	doc-only / unverifiable        -> DECLARED_UNVERIFIED
//	neither source                 -> ABSENT (pointer-only)
//
// DERIVED / VALIDATED / CONTRADICTED MUST carry at least one citation; Validate
// enforces that locally, mirroring search-hub's ValidateAttestation so a
// non-conformant attestation never escapes a capability.
package attest

import (
	"fmt"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Citation kinds. Mirrors the contract's kind vocabulary.
const (
	KindCode     = "code"
	KindDoc      = "doc"
	KindContract = "contract"
	KindRuntime  = "runtime"
	KindExternal = "external"
)

// Builder assembles an AttestedAnswer fluently. The zero value is not usable;
// start with New.
type Builder struct {
	answer *commonv1.AttestedAnswer
}

// New starts a builder for a claim.
func New(claim string) *Builder {
	return &Builder{answer: &commonv1.AttestedAnswer{Claim: strings.TrimSpace(claim)}}
}

// Basis sets the epistemic-provenance axis.
func (b *Builder) Basis(basis commonv1.Basis) *Builder {
	b.answer.Basis = basis
	return b
}

// Sufficiency sets the coverage axis.
func (b *Builder) Sufficiency(s commonv1.Sufficiency) *Builder {
	b.answer.Sufficiency = s
	return b
}

// Cite appends a provenance pointer. Empty locators are ignored.
func (b *Builder) Cite(locator, kind, note string) *Builder {
	locator = strings.TrimSpace(locator)
	if locator == "" {
		return b
	}
	b.answer.Citations = append(b.answer.Citations, &commonv1.Citation{
		Locator: locator,
		Kind:    strings.TrimSpace(kind),
		Note:    strings.TrimSpace(note),
	})
	return b
}

// CiteCode is shorthand for a code citation (file:line / package path).
func (b *Builder) CiteCode(locator, note string) *Builder { return b.Cite(locator, KindCode, note) }

// CiteDoc is shorthand for a doc citation (DOMAINS.md / ARCHITECTURE.md).
func (b *Builder) CiteDoc(locator, note string) *Builder { return b.Cite(locator, KindDoc, note) }

// Gap appends a known gap (what the answer does NOT cover).
func (b *Builder) Gap(gap string) *Builder {
	if gap = strings.TrimSpace(gap); gap != "" {
		b.answer.Gaps = append(b.answer.Gaps, gap)
	}
	return b
}

// FollowUp appends a suggested next read / command.
func (b *Builder) FollowUp(s string) *Builder {
	if s = strings.TrimSpace(s); s != "" {
		b.answer.SuggestedFollowUps = append(b.answer.SuggestedFollowUps, s)
	}
	return b
}

// Build returns the assembled answer without validating it. Prefer
// BuildValidated at capability boundaries.
func (b *Builder) Build() *commonv1.AttestedAnswer {
	return b.answer
}

// BuildValidated returns the assembled answer and the result of Validate, so a
// caller can drop or surface a non-conformant attestation rather than emit it.
func (b *Builder) BuildValidated() (*commonv1.AttestedAnswer, error) {
	a := b.Build()
	return a, Validate(a)
}

// Validate is the attestation conformance lint: a claim is always required, and
// a basis that references code (DERIVED / VALIDATED / CONTRADICTED) MUST carry
// at least one citation. DECLARED_UNVERIFIED and ABSENT may be uncited.
func Validate(a *commonv1.AttestedAnswer) error {
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

// ConvergenceBasis maps the uniform basis rule for a doc-vs-code convergence:
// hasCode/hasDoc describe which sources exist, and agree is whether they match
// when both are present.
func ConvergenceBasis(hasCode, hasDoc, agree bool) commonv1.Basis {
	switch {
	case hasCode && hasDoc && agree:
		return commonv1.Basis_BASIS_VALIDATED
	case hasCode && hasDoc && !agree:
		return commonv1.Basis_BASIS_CONTRADICTED
	case hasCode:
		return commonv1.Basis_BASIS_DERIVED
	case hasDoc:
		return commonv1.Basis_BASIS_DECLARED_UNVERIFIED
	default:
		return commonv1.Basis_BASIS_ABSENT
	}
}
