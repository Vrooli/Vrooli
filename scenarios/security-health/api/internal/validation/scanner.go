package validation

import (
	"context"
	"time"
)

// Scanner is one pluggable security tool. The Service resolves each scanner's
// binary, asks whether it Applies to the detected substrate, and — when both
// hold — calls Scan. A scanner that Applies but whose Binary is absent is
// recorded as a skipped scanner (degraded, not failed); a scanner that does
// not Apply is silently inert.
//
// Adding a tool is one file implementing this interface plus one line in
// DefaultScanners.
type Scanner interface {
	// Name is the stable scanner identifier ("gitleaks", "gosec", …). It is
	// the value stamped on every Finding.Scanner this scanner emits and the
	// token reported in SkippedScanners.
	Name() string
	// Binary is the executable looked up on PATH. When absent and the scanner
	// Applies, the scan is skipped rather than failed.
	Binary() string
	// Applies reports whether this scanner has anything to do given the
	// detected substrate (e.g. gosec applies iff Substrate.Go).
	Applies(s Substrate) bool
	// Scan runs the tool against scenarioDir and returns normalized findings.
	// It must parse the tool's native output regardless of process exit code
	// (scanners exit non-zero when they find issues). A returned error means
	// the scan itself failed to produce parseable output — the Service turns
	// that into an INFO observation, never a hard failure, so one flaky
	// scanner can't gate a scenario spuriously.
	Scan(ctx context.Context, scenarioDir string, sub Substrate) ([]Finding, error)
}

// DOC: docs/internal/PERFORMANCE.md#incremental-validation-model
// IncrementalScanner is the opt-in correctness contract for reusable scanner
// evidence. Plan must fingerprint a conservative superset of every input that
// can affect Scan's normalized findings. Scanners which do not implement this
// interface still run through shared admission, but are never cached.
type IncrementalScanner interface {
	Scanner
	EvidencePlan(ctx context.Context, scenarioDir string, sub Substrate, now time.Time) (ScannerEvidencePlan, error)
}

// ScannerEvidencePlan describes the cost and freshness of one scanner result.
// Fingerprint includes source/manifests, tool identity, normalization policy,
// and a freshness epoch for scanners backed by mutable advisory databases.
type ScannerEvidencePlan struct {
	Fingerprint string
	Weight      int64
	FreshFor    time.Duration
}
