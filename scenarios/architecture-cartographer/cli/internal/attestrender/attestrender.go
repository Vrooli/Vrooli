// Package attestrender renders the shared common.v1 attestation contract for
// human CLI output. Every cartographer CLI domain (domains/conflicts/zones/
// slice/archetype) carries the same honesty axes, so the basis/sufficiency
// tokens are formatted here once.
package attestrender

import commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

// Basis renders the attestation basis as a short lowercase token. Empty for an
// unspecified basis so callers can omit it.
func Basis(b commonv1.Basis) string {
	switch b {
	case commonv1.Basis_BASIS_DERIVED:
		return "derived"
	case commonv1.Basis_BASIS_VALIDATED:
		return "validated"
	case commonv1.Basis_BASIS_DECLARED_UNVERIFIED:
		return "declared-unverified"
	case commonv1.Basis_BASIS_CONTRADICTED:
		return "contradicted"
	case commonv1.Basis_BASIS_ABSENT:
		return "absent"
	default:
		return ""
	}
}

// Sufficiency renders the attestation sufficiency as a short token.
func Sufficiency(s commonv1.Sufficiency) string {
	switch s {
	case commonv1.Sufficiency_SUFFICIENCY_FULL:
		return "full"
	case commonv1.Sufficiency_SUFFICIENCY_PARTIAL:
		return "partial"
	case commonv1.Sufficiency_SUFFICIENCY_INSUFFICIENT:
		return "insufficient"
	default:
		return "unspecified"
	}
}
