package vrooli

import (
	"context"
	"strings"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type ResourceStatus struct {
	Success   bool
	Name      string
	Installed bool
	Running   bool
	Healthy   *bool
	// Serving is true when the resource answers requests, whether or not it is
	// healthy. A resource running below its declared accelerator backend
	// reports healthy=false with serving=true; restarting it would loop against
	// something that is working.
	Serving *bool
	// ModeDrift is true when the resource is serving below its declared
	// accelerator backend.
	ModeDrift bool
	// NeedsReacquire is true when the staged artifact was chosen under host
	// facts the host no longer reports. The bytes are intact; a restart cannot
	// fix it and a re-acquire can.
	NeedsReacquire bool
	StatusCode     string
	// DeclaredMode is the accelerator backend the resource asked for; empty
	// when it declares none.
	DeclaredMode string
	// ObservedMode is the backend the host says it is on; empty means the
	// placement could not be read, which is never agreement.
	ObservedMode string
	// ModeReason is the host's evidence behind ObservedMode.
	ModeReason string
	// ProbeError is the producer's full diagnosis. For needs_reacquire it names
	// which host facts moved and the command that repairs it, so an incident
	// body carries the reason without an operator re-running anything.
	ProbeError    string
	Status        string
	CompanionDown bool
}

// SnapshotSource identifies the typed producer used for an observation.
type SnapshotSource string

const (
	SnapshotSourceFleet     SnapshotSource = "fleet"
	SnapshotSourceNamedDeep SnapshotSource = "named_deep"
	SnapshotSourceFallback  SnapshotSource = "fallback"
)

// ProbeLevel describes how much work produced an observation.
type ProbeLevel string

const (
	ProbeLevelFast         ProbeLevel = "fast"
	ProbeLevelDeep         ProbeLevel = "deep"
	ProbeLevelUndetermined ProbeLevel = "undetermined"
)

// ResourceStatusSnapshot is an immutable-by-convention copy of a typed fleet
// response. Callers receive cloned maps and tri-state values from Snapshot so
// one check cannot mutate another check's view.
type ResourceStatusSnapshot struct {
	ObservedAt    time.Time
	ExpiresAt     time.Time
	Complete      bool
	Source        SnapshotSource
	ProbeLevel    ProbeLevel
	Resources     map[string]ResourceStatus
	RefreshError  string
	NextRefreshAt time.Time
	StaleUntil    time.Time
}

// AgeAt returns the non-negative age of the observation at now.
func (s ResourceStatusSnapshot) AgeAt(now time.Time) time.Duration {
	if s.ObservedAt.IsZero() || now.Before(s.ObservedAt) {
		return 0
	}
	return now.Sub(s.ObservedAt)
}

// FreshAt reports whether the snapshot is current and not a fallback result.
func (s ResourceStatusSnapshot) FreshAt(now time.Time) bool {
	return s.Source != SnapshotSourceFallback && !s.ExpiresAt.IsZero() && now.Before(s.ExpiresAt)
}

// ResourceStatusSnapshotProvider is the Autoheal-facing typed observation
// seam. Implementations own caching and coalescing; checks only classify the
// returned evidence and request a named deep read for safety-sensitive cases.
type ResourceStatusSnapshotProvider interface {
	Snapshot(ctx context.Context, names []string) (ResourceStatusSnapshot, error)
	DeepStatus(ctx context.Context, name string) (ResourceStatus, error)
}

// ResourceStatusForceProvider is the safety-sensitive extension used after a
// resource mutation. A forced read bypasses the named-status cache while still
// sharing any already-running deep request for the same resource.
type ResourceStatusForceProvider interface {
	ResourceStatusSnapshotProvider
	ForceDeepStatus(ctx context.Context, name string) (ResourceStatus, error)
}

// SnapshotMetrics is a bounded operator-facing summary of the provider. It
// intentionally contains counts and timings, never command lines or payloads.
type SnapshotMetrics struct {
	SnapshotRequests    uint64            `json:"snapshotRequests"`
	FleetRefreshes      uint64            `json:"fleetRefreshes"`
	NamedDeepChecks     uint64            `json:"namedDeepChecks"`
	CoalescedCalls      uint64            `json:"coalescedCalls"`
	RefreshFailures     uint64            `json:"refreshFailures"`
	StaleReads          uint64            `json:"staleReads"`
	IncompleteResponses uint64            `json:"incompleteResponses"`
	UndeterminedResults uint64            `json:"undeterminedResults"`
	EscalationReasons   map[string]uint64 `json:"escalationReasons,omitempty"`
	RefreshLatency      time.Duration     `json:"refreshLatency"`
	DeepLatency         time.Duration     `json:"deepLatency"`
}

// SnapshotMetricsProvider is optional so small test fakes need only implement
// the two observation methods.
type SnapshotMetricsProvider interface {
	SnapshotMetrics() SnapshotMetrics
}

// SnapshotMetricsRecorder lets checks count domain outcomes without exposing
// command lines, payloads, or unbounded error labels.
type SnapshotMetricsRecorder interface {
	RecordUndetermined()
	RecordEscalation(reason string)
}

// SnapshotClockProvider exposes the provider clock to domain consumers so
// deterministic fixtures and production freshness checks use the same notion
// of now.
type SnapshotClockProvider interface {
	SnapshotNow() time.Time
}

// IsDegraded reports a resource that is serving but not meeting its contract.
// It is the state that must never be treated as an outage.
func (r ResourceStatus) IsDegraded() bool {
	if r.Healthy == nil || *r.Healthy {
		return false
	}
	return r.ModeDrift || (r.Serving != nil && *r.Serving)
}

func (r ResourceStatus) NormalizedStatus() string {
	return strings.ToLower(strings.TrimSpace(r.Status))
}

// HasCompanionDownSignal exposes the producer's machine-readable companion
// fault classification to checks and healers. Human status text is diagnostic;
// recovery decisions consume this typed flag.
func (r ResourceStatus) HasCompanionDownSignal() bool { return r.CompanionDown }

// ResourceStatus runs `vrooli resource status <name> --json` and returns the
// typed status mapped onto the autoheal domain shape.
func (c *Client) ResourceStatus(ctx context.Context, resourceName string) (ResourceStatus, []byte, error) {
	resp, err := c.cli.ResourceStatus(ctx, resourceName)
	if err != nil {
		return ResourceStatus{}, nil, err
	}
	status := resourceStatusFromRecord(resp.GetSuccess(), resp.GetResource())
	// The single-resource envelope repeats these fields for compatibility;
	// prefer that established top-level projection over a sparse nested record.
	status.Success = resp.GetSuccess()
	status.Name = resp.GetName()
	status.Installed = resp.GetInstalled()
	status.Running = resp.GetRunning()
	status.Status = resp.GetStatus()
	status.Healthy = valueToBoolPtr(resp.GetHealthy())
	status.Serving = valueToBoolPtr(resp.GetServing())
	status.CompanionDown = status.CompanionDown || companionDownSignal(status.Status)
	status.ModeDrift = status.ModeDrift || resp.GetModeDrift()
	return status, diagnosticJSON(resp), nil
}

// ResourceStatuses maps the typed fleet response into the Autoheal status
// vocabulary without routing through human-oriented CLI text.
func (c *Client) ResourceStatuses(ctx context.Context) ([]ResourceStatus, bool, int, []byte, error) {
	resp, err := c.cli.ResourceStatuses(ctx)
	if err != nil {
		return nil, false, 0, nil, err
	}
	statuses := make([]ResourceStatus, 0, len(resp.GetResources()))
	for _, item := range resp.GetResources() {
		statuses = append(statuses, resourceStatusFromRecord(resp.GetSuccess(), item))
	}
	return statuses, resp.GetSuccess(), len(resp.GetDiscoveryFailures()), diagnosticJSON(resp), nil
}

func resourceStatusFromRecord(success bool, item *cliv1.ResourceStatus) ResourceStatus {
	if item == nil {
		return ResourceStatus{Success: success}
	}
	resource := item.GetResource()
	status := ResourceStatus{
		Success:       success,
		Name:          resource.GetName(),
		Installed:     item.GetInstalled(),
		Running:       item.GetRunning(),
		Healthy:       valueToBoolPtr(item.GetHealthy()),
		Serving:       valueToBoolPtr(item.GetServing()),
		StatusCode:    item.GetStatusCode(),
		ProbeError:    item.GetProbeError(),
		DeclaredMode:  item.GetDeclaredMode(),
		ObservedMode:  item.GetObservedMode(),
		ModeDrift:     item.GetModeDrift(),
		ModeReason:    item.GetModeReason(),
		Status:        item.GetMessage(),
		CompanionDown: companionDownSignal(item.GetMessage()),
	}
	if status.Status == "" {
		status.Status = item.GetHealth()
	}
	if status.Status == "" {
		status.Status = status.StatusCode
	}
	status.NeedsReacquire = status.StatusCode == statusCodeNeedsReacquire
	return status
}

func companionDownSignal(status string) bool {
	return strings.Contains(strings.ToLower(status), "companion down")
}

// valueToBoolPtr maps a tri-state protobuf Value onto *bool: a bool value
// yields a pointer to it; null / unset / non-bool yields nil.
func valueToBoolPtr(v *structpb.Value) *bool {
	if v == nil {
		return nil
	}
	if b, ok := v.GetKind().(*structpb.Value_BoolValue); ok {
		val := b.BoolValue
		return &val
	}
	return nil
}

// statusCodeNeedsReacquire is the control plane's typed artifact-drift code.
const statusCodeNeedsReacquire = "needs_reacquire"
