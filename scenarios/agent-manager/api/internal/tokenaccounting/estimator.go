package tokenaccounting

// estimatedBytesPerToken is a dependency-free approximation for payloads
// whose provider tokenizer is unavailable. Four UTF-8 bytes per token tracks
// ordinary ASCII prose reasonably, but can overestimate token counts for
// multi-byte text and under/over-shoot structured command output depending on
// vocabulary. Phase 8 measures the error against provider usage.
const estimatedBytesPerToken = 4

// Estimate is a token count together with the evidence basis that produced it.
type Estimate struct {
	Tokens int64
	Basis  Basis
}

// EstimateText estimates tokens from a payload using the package's one
// documented heuristic. A non-empty payload always retains a positive count.
func EstimateText(payload string) Estimate {
	if payload == "" {
		return Estimate{Basis: BasisEstimated}
	}
	return Estimate{Tokens: int64((len(payload) + estimatedBytesPerToken - 1) / estimatedBytesPerToken), Basis: BasisEstimated}
}

// Measured wraps a provider-reported token count.
func Measured(tokens int64) Estimate {
	return Estimate{Tokens: tokens, Basis: BasisMeasured}
}

// Unknown represents a token quantity that cannot be recovered from the
// retained evidence.
func Unknown() Estimate {
	return Estimate{Basis: BasisUnknown}
}
