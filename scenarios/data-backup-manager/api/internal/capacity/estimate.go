// Package capacity owns the one admission model for backup repositories.
//
// A repository's byte cost is not sourceBytes × retentionCopies. Kopia
// deduplicates and compresses, the first copy is the only one that pays the
// full logical cost, and every later snapshot is an increment. The previous
// model multiplied the full source estimate by the sum of every GFS tier,
// which treated a 24/7/4/6/2 policy as 46 full copies and blocked repositories
// that had ample headroom.
//
// This package separates two questions:
//
//   - AdmissionBytes: can this write start now without crossing the reserve?
//   - ForecastBytes: will the configured retention policy fit over time?
//
// Admission is a hard gate; the forecast is a warning under the default
// advisory policy and a hard gate only under strict policy.
package capacity

import (
	"fmt"
	"math"
)

// Policy controls how the long-run retention forecast affects admission.
type Policy string

const (
	// PolicyAdvisory admits the immediate write and warns when the retention
	// forecast exceeds free space. This is the default for scheduled backups:
	// Kopia's dedup growth is best measured empirically, and retention/pruning
	// owns the long-run bound.
	PolicyAdvisory Policy = "advisory"
	// PolicyStrict requires the full retention forecast to fit before admitting
	// any write. Use for migration proofs and single-copy destinations where
	// running out of headroom is not recoverable.
	PolicyStrict Policy = "strict"
)

// DefaultReserveBytes is the free-space floor kept untouched by a backup
// write. It is a safety margin for repository metadata, temporary files, and
// the host's own operation, not part of the repository's capacity.
const DefaultReserveBytes int64 = 20 << 30 // 20 GiB

// Confidence describes how much of the estimate is measured rather than
// assumed. Callers must not present an estimated forecast as measured.
type Confidence string

const (
	Measured  Confidence = "measured"
	Estimated Confidence = "estimated"
	Unknown   Confidence = "unknown"
)

// Input is the evidence available before a write.
type Input struct {
	// LogicalSourceBytes is the complete selected source estimate. Zero means
	// the source could not be enumerated; admission stays unknown.
	LogicalSourceBytes int64
	// RetainedSnapshots is the maximum number of snapshots the retention policy
	// can keep for a source. It is a snapshot count, not a byte multiplier.
	RetainedSnapshots int
	// FirstCopy is true when the selected target has no successful snapshot on
	// this destination. The first copy pays the full logical cost.
	FirstCopy bool
	// MeasuredIncrementalBytes is the observed physical repository growth of a
	// representative previous run. Zero means no usable history.
	MeasuredIncrementalBytes int64
	// MeasuredDedupRatio is logical ÷ physical. Values below 1 are treated as
	// 1 (no dedup) so an unmeasured ratio never lowers the first-copy estimate.
	MeasuredDedupRatio float64
	// ReserveBytes is kept free after the write. Zero selects
	// DefaultReserveBytes; a negative value explicitly disables the reserve for
	// legacy callers that never had one.
	ReserveBytes int64
	// Policy controls whether the retention forecast is advisory or strict.
	// Empty defaults to PolicyAdvisory.
	Policy Policy
}

// Estimate is the admission and forecast result. All byte fields are absolute
// repository bytes, not multiples of the source size.
type Estimate struct {
	LogicalSourceBytes int64
	FirstCopyBytes     int64
	IncrementalBytes   int64
	RetainedSnapshots  int
	AdmissionBytes     int64
	ForecastBytes      int64
	ReserveBytes       int64
	Confidence         Confidence
	Policy             Policy
	// UnboundedRetention means the policy has no snapshot bound. Admission can
	// still prove the immediate write, but no honest long-run forecast exists.
	UnboundedRetention bool
	Basis              string
}

// Decision is the admission result for a measured free-space value.
type Decision struct {
	Allowed bool
	Warning bool
	Reason  string
}

// EstimateRequired applies the admission model. It is pure and safe to call
// from preflight, readiness, migration, and tests.
func EstimateRequired(in Input) Estimate {
	est := Estimate{
		LogicalSourceBytes: in.LogicalSourceBytes,
		RetainedSnapshots:  max(in.RetainedSnapshots, 1),
		ReserveBytes:       in.ReserveBytes,
		Policy:             in.Policy,
		UnboundedRetention: in.RetainedSnapshots <= 0,
	}
	switch {
	case est.ReserveBytes < 0:
		// A negative reserve is the explicit "no reserve" marker used by the
		// legacy public readiness surface, which did not carry one.
		est.ReserveBytes = 0
	case est.ReserveBytes == 0:
		est.ReserveBytes = DefaultReserveBytes
	}
	if est.Policy == "" {
		est.Policy = PolicyAdvisory
	}
	if in.LogicalSourceBytes <= 0 {
		est.Confidence = Unknown
		est.Basis = "source estimate unavailable; capacity admission is unknown"
		return est
	}

	dedup := in.MeasuredDedupRatio
	if dedup < 1 {
		// A ratio below 1 would claim that dedup costs more than the logical
		// data. Treat it as unmeasured and keep the conservative first copy.
		dedup = 1
	}

	first := int64(0)
	if in.FirstCopy {
		first = in.LogicalSourceBytes
		if dedup > 1 {
			first = int64(float64(in.LogicalSourceBytes) / dedup)
		}
	}

	incremental := in.MeasuredIncrementalBytes
	est.Confidence = Measured
	if incremental <= 0 {
		// Without a measured increment, assume 5% churn of the first-copy
		// footprint per retained snapshot, with a 64 MiB floor. This is a
		// forecast only; the advisory policy does not turn it into a hard gate.
		base := in.LogicalSourceBytes
		if dedup > 1 {
			base = int64(float64(base) / dedup)
		}
		incremental = base / 20
		if incremental < 64<<20 {
			incremental = 64 << 20
		}
		est.Confidence = Estimated
	}

	est.FirstCopyBytes = first
	est.IncrementalBytes = incremental
	if in.FirstCopy {
		est.AdmissionBytes = add(first, est.ReserveBytes)
		est.ForecastBytes = add(add(first, mul(incremental, int64(est.RetainedSnapshots-1))), est.ReserveBytes)
	} else {
		est.AdmissionBytes = add(incremental, est.ReserveBytes)
		est.ForecastBytes = add(mul(incremental, int64(est.RetainedSnapshots)), est.ReserveBytes)
	}
	retention := fmt.Sprintf("snapshots=%d", est.RetainedSnapshots)
	if est.UnboundedRetention {
		retention = "retention=unbounded"
	}
	est.Basis = fmt.Sprintf(
		"first_copy=%s incremental=%s %s reserve=%s policy=%s confidence=%s dedup=%.2fx",
		HumanBytes(est.FirstCopyBytes), HumanBytes(est.IncrementalBytes), retention,
		HumanBytes(est.ReserveBytes), est.Policy, est.Confidence, dedup,
	)
	return est
}

// Decide evaluates the estimate against free space. Admission is the hard
// gate. The retention forecast is a warning under advisory policy and a hard
// gate under strict policy.
func (e Estimate) Decide(freeBytes int64) Decision {
	if e.Confidence == Unknown || e.AdmissionBytes <= 0 {
		return Decision{Allowed: false, Reason: "capacity admission is unknown; the source estimate is unavailable"}
	}
	if freeBytes < 0 {
		freeBytes = 0
	}
	if freeBytes < e.AdmissionBytes {
		return Decision{
			Allowed: false,
			Reason: fmt.Sprintf("free %s is below the immediate write need %s",
				HumanBytes(freeBytes), HumanBytes(e.AdmissionBytes)),
		}
	}
	if e.UnboundedRetention {
		if e.Policy == PolicyStrict {
			return Decision{
				Allowed: false,
				Reason:  "retention is unbounded; strict policy cannot prove the long-run forecast",
			}
		}
		return Decision{
			Allowed: true,
			Warning: true,
			Reason:  "retention is unbounded; the immediate write fits, but no honest long-run capacity forecast exists",
		}
	}
	if freeBytes < e.ForecastBytes {
		if e.Policy == PolicyStrict {
			return Decision{
				Allowed: false,
				Reason: fmt.Sprintf("free %s is below the full retention forecast %s under strict policy",
					HumanBytes(freeBytes), HumanBytes(e.ForecastBytes)),
			}
		}
		return Decision{
			Allowed: true,
			Warning: true,
			Reason: fmt.Sprintf("free %s covers the immediate write %s but not the full retention forecast %s; retention and pruning own the long-run bound",
				HumanBytes(freeBytes), HumanBytes(e.AdmissionBytes), HumanBytes(e.ForecastBytes)),
		}
	}
	return Decision{Allowed: true}
}

// HumanBytes renders a byte count for operator-facing messages. It is here so
// capacity prose has one formatting rule instead of a different approximation
// in every caller.
func HumanBytes(bytes int64) string {
	if bytes < 0 {
		return "-" + HumanBytes(-bytes)
	}
	const (
		KiB = int64(1) << 10
		MiB = int64(1) << 20
		GiB = int64(1) << 30
		TiB = int64(1) << 40
	)
	switch {
	case bytes >= TiB:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/float64(TiB))
	case bytes >= GiB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/float64(GiB))
	case bytes >= MiB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/float64(MiB))
	case bytes >= KiB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func add(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	if b < 0 && a < math.MinInt64-b {
		return math.MinInt64
	}
	return a + b
}

func mul(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > 0 && b > 0 && a > math.MaxInt64/b {
		return math.MaxInt64
	}
	if a < 0 && b < 0 && a < math.MaxInt64/b {
		return math.MaxInt64
	}
	return a * b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
