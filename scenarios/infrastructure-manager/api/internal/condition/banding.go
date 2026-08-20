// Package condition contains the pure trust and banding rules. Source clients
// belong above this package; keeping the verdict functions free of I/O makes
// the conservative cases testable and prevents an unavailable sensor from
// becoming a plant failure by accident.
package condition

type TrustVerdict string

const (
	TrustValid        TrustVerdict = "VALID"
	TrustGhost        TrustVerdict = "GHOST"
	TrustSaturated    TrustVerdict = "SATURATED"
	TrustShelved      TrustVerdict = "SHELVED"
	TrustUnitMismatch TrustVerdict = "UNIT_MISMATCH"
	TrustUnavailable  TrustVerdict = "UNAVAILABLE"
	TrustUntrusted    TrustVerdict = "UNTRUSTED"
)

type TrustInput struct {
	Available    bool
	Ghost        bool
	Saturated    bool
	Shelved      bool
	UnitMatches  bool
	VerdictToken string
}

func EvaluateTrust(input TrustInput) TrustVerdict {
	if !input.Available {
		return TrustUnavailable
	}
	if input.Ghost {
		return TrustGhost
	}
	if input.Shelved {
		return TrustShelved
	}
	if input.Saturated {
		return TrustSaturated
	}
	if !input.UnitMatches {
		return TrustUnitMismatch
	}
	if input.VerdictToken != "" && input.VerdictToken != string(TrustValid) {
		return NormalizeTrust(input.VerdictToken)
	}
	return TrustValid
}

func NormalizeTrust(token string) TrustVerdict {
	switch token {
	case string(TrustValid):
		return TrustValid
	case string(TrustGhost):
		return TrustGhost
	case string(TrustSaturated):
		return TrustSaturated
	case string(TrustShelved):
		return TrustShelved
	case string(TrustUnitMismatch):
		return TrustUnitMismatch
	case string(TrustUnavailable):
		return TrustUnavailable
	default:
		return TrustUntrusted
	}
}

type BandVerdict string

const (
	BandInBand         BandVerdict = "IN_BAND"
	BandOutOfBand      BandVerdict = "OUT_OF_BAND"
	BandPendingSustain BandVerdict = "PENDING_SUSTAIN"
	BandNeedsBaseline  BandVerdict = "NEEDS_BASELINE"
	BandNotEvaluated   BandVerdict = "NOT_EVALUATED"
)

type Band struct {
	Min              *float64
	Max              *float64
	SustainSatisfied bool
	NeedsBaseline    bool
}

func EvaluateBand(value float64, trust TrustVerdict, band Band) BandVerdict {
	if trust != TrustValid {
		return BandNotEvaluated
	}
	if band.NeedsBaseline {
		return BandNeedsBaseline
	}
	inBand := (band.Min == nil || value >= *band.Min) && (band.Max == nil || value <= *band.Max)
	if inBand {
		return BandInBand
	}
	if !band.SustainSatisfied {
		return BandPendingSustain
	}
	return BandOutOfBand
}
