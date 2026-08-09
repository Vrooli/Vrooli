package tokenaccounting

// View identifies the question answered by a token attribution aggregate.
type View string

const (
	Footprint View = "footprint"
	Residency View = "residency"
	Incurred  View = "incurred"
)

// Basis identifies how a token number was obtained.
type Basis string

const (
	BasisMeasured  Basis = "measured"
	BasisEstimated Basis = "estimated"
	BasisUnknown   Basis = "unknown"
)

// Bucket identifies a run-level attribution bucket.
type Bucket string

const (
	BucketPreambleInjected    Bucket = "preamble_injected"
	BucketPreambleFixed       Bucket = "preamble_fixed"
	BucketToolResultResidency Bucket = "tool_result_residency"
	BucketAssistantOutput     Bucket = "assistant_output"
	BucketCompaction          Bucket = "compaction"
	BucketUnattributed        Bucket = "unattributed"
)

// AllBuckets returns the complete, stable bucket vocabulary.
func AllBuckets() []Bucket {
	return []Bucket{
		BucketPreambleInjected,
		BucketPreambleFixed,
		BucketToolResultResidency,
		BucketAssistantOutput,
		BucketCompaction,
		BucketUnattributed,
	}
}

// TokenAccounting is the run-level conservation ledger. Fields are factors
// or independently attributable totals; no field stores a derived product.
type TokenAccounting struct {
	PreambleInjectedTokens    int64
	PreambleFixedTokens       int64
	ToolResultResidencyTokens int64
	AssistantOutputTokens     int64
	CompactionTokens          int64
	UnattributedTokens        int64
	UnattributedReason        string
}

// Tokens returns the sum of every bucket, including the explicit residual.
func (a TokenAccounting) Tokens() int64 {
	return a.PreambleInjectedTokens + a.PreambleFixedTokens +
		a.ToolResultResidencyTokens + a.AssistantOutputTokens +
		a.CompactionTokens + a.UnattributedTokens
}

// Conserves reports whether the ledger reconciles exactly to the run total.
func (a TokenAccounting) Conserves(runTotal int64) bool {
	return a.Tokens() == runTotal
}
