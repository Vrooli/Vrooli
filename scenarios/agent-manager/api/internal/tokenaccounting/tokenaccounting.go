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

// SegmentShare apportions one per-call token quantity across the segments a
// compound invocation was split into.
//
// A tool call carries exactly one input payload and one result payload, but
// the read model emits one fact per shell segment so that each command in a
// compound line stays independently rankable. Copying the whole call quantity
// onto every segment would make any SUM over facts overcount the run in
// proportion to how compound its commands were, which silently inflates the
// commands that happen to be written as pipelines.
//
// The split is even because the retained evidence cannot say which segment
// produced the payload: a pipeline's bytes are shaped by every stage, and the
// segment texts are not retained. An even split is a declared approximation,
// not a measurement — consumers distinguish it by SegmentCount > 1 on the
// fact. The remainder is assigned to the leading segments so the shares sum
// back to the original total exactly for any segment count.
//
// Totals are non-negative by construction (payload lengths and provider
// counts). A negative total has no apportionment and yields zero rather than
// distributing a nonsense remainder.
func SegmentShare(total int64, segmentCount, segmentIndex int) int64 {
	if segmentCount <= 1 {
		return total
	}
	if segmentIndex < 0 || segmentIndex >= segmentCount || total <= 0 {
		return 0
	}
	count := int64(segmentCount)
	share := total / count
	if int64(segmentIndex) < total%count {
		share++
	}
	return share
}
