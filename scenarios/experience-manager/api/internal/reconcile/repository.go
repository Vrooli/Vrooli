package reconcile

import "context"

// EvidenceRepository persists per-claim reconciliation evidence.
type EvidenceRepository interface {
	SaveEvidence(ctx context.Context, evidence Evidence) error
	ListEvidence(ctx context.Context, filter EvidenceFilter) ([]Evidence, error)
}

// CaptureTimingRepository is implemented by evidence stores that retain
// per-target performance history alongside reconciliation evidence.
type CaptureTimingRepository interface {
	SaveCaptureTiming(ctx context.Context, timing CaptureTargetTiming) error
	ListCaptureTimings(ctx context.Context, filter CaptureTimingFilter) ([]CaptureTargetTiming, error)
}

// CaptureTargetTiming associates BAS timing diagnostics with a stable capture
// matrix target.
type CaptureTargetTiming struct {
	Scenario                  string
	DocumentKind              string
	PageID                    string
	ComponentID               string
	Route                     string
	StateID                   string
	ViewportID                string
	ViewportWidth             int
	ViewportHeight            int
	TotalMilliseconds         int64
	NavigationMilliseconds    int64
	ReadinessWaitMilliseconds int64
	Strategy                  string
	Outcome                   string
	CapturedAt                string
}

type CaptureTimingFilter struct {
	Scenario    string
	PageID      string
	ComponentID string
	Limit       int
}

// Evidence records the AX evidence used for one claim verdict.
type Evidence struct {
	ID             string
	Scenario       string
	DocumentKind   string
	PageID         string
	ComponentID    string
	ComponentTitle string
	ExampleName    string
	Route          string
	StateID        string
	ViewportID     string
	ViewportWidth  int
	ViewportHeight int
	ClaimID        string
	ClaimType      string
	Verdict        string
	CaptureRef     string
	AXNodeJSON     string
	Message        string
	CheckedAt      string
}

// EvidenceFilter narrows evidence reads.
type EvidenceFilter struct {
	Scenario    string
	PageID      string
	ComponentID string
	ClaimID     string
	Limit       int
}
