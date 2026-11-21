package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	capacity "github.com/vrooli/vrooli/packages/capacity"
)

// Verdict is the admission answer for one batch.
type Verdict struct {
	// Kind is grant, degrade, queue or deny. It mirrors the capacity broker's
	// vocabulary so a caller can report the broker's reason verbatim.
	Kind string
	// Reason explains a non-grant verdict.
	Reason string
}

// Verdict kinds.
const (
	VerdictGrant   = "grant"
	VerdictDegrade = "degrade"
	VerdictQueue   = "queue"
	VerdictDeny    = "deny"
)

// Granted reports whether work may run. Degrade is granted: the broker admitted
// a smaller amount and the owner is responsible for fitting inside it.
func (v Verdict) Granted() bool { return v.Kind == VerdictGrant || v.Kind == VerdictDegrade }

// Lease is a capacity reservation that must be released when the work ends.
type Lease interface {
	Release(context.Context) error
}

// RenewableLease can extend its claims. Long-running batches should be renewed
// while they execute; a lost renewal is a lost admission.
type RenewableLease interface {
	Lease
	Heartbeat(context.Context) error
}

// AdmissionRequest is one batch admission request.
//
// The request carries the estimate, priority, and no-reclaim flag the broker
// needs, plus the identity that makes the resulting claim attributable. Identity
// is not optional: an anonymous claim cannot be reclaimed, reported, or tied to
// the job that holds it.
type AdmissionRequest struct {
	JobID         string
	Kind          string
	Owner         string
	ResourceClass ResourceClass
	Estimate      Estimate
	Priority      Priority
	NoReclaim     bool
}

// Validate rejects requests the broker would reject anyway, so a misconfigured
// job fails at the scheduler boundary with a clear error.
func (r AdmissionRequest) Validate() error {
	if strings.TrimSpace(r.JobID) == "" {
		return errors.New("schedule: admission request requires a job id")
	}
	if !r.Priority.Valid() {
		return fmt.Errorf("schedule: admission request has unknown priority %q", r.Priority)
	}
	if !r.ResourceClass.Valid() {
		return fmt.Errorf("schedule: admission request has unknown resource class %q", r.ResourceClass)
	}
	if r.ResourceClass == ResourceClassNone {
		return nil
	}
	if r.Estimate.RAMBytes <= 0 && r.Estimate.VRAMBytes <= 0 {
		return errors.New("schedule: admission request requires a measured RAM or VRAM estimate")
	}
	if r.ResourceClass == ResourceClassVRAM && r.Estimate.VRAMBytes <= 0 {
		return errors.New("schedule: VRAM admission requires a VRAM estimate")
	}
	return nil
}

// Admission is the only path from the scheduler to host capacity. It exists so
// the scheduler never re-implements admission policy: the production
// implementation wraps packages/capacity.Broker, and tests provide a fake.
type Admission interface {
	Acquire(ctx context.Context, req AdmissionRequest) (Lease, Verdict, error)
}

// AdmissionFunc adapts a function to Admission.
type AdmissionFunc func(ctx context.Context, req AdmissionRequest) (Lease, Verdict, error)

// Acquire implements Admission.
func (f AdmissionFunc) Acquire(ctx context.Context, req AdmissionRequest) (Lease, Verdict, error) {
	return f(ctx, req)
}

// ResourceBroker is the slice of packages/capacity.Broker the adapter uses.
// Keeping it an interface makes the adapter testable without a live host.
type ResourceBroker interface {
	AcquireResources(ctx context.Context, req capacity.Request) (capacity.RenewableLease, capacity.Verdict, error)
	AcquireGPU(ctx context.Context, ownerID, uuid string, vramBytes int64) (capacity.GPULease, capacity.Verdict, error)
}

// BrokerAdmissionOptions configures the production adapter.
type BrokerAdmissionOptions struct {
	// OwnerPrefix prefixes every claim owner. Use one per plane, for example
	// "schedule:agent-manager:embeddings", so claims are attributable.
	OwnerPrefix string
	// GPUUUID binds VRAM jobs to one physical device. The broker refuses to
	// guess an index; a scheduler with VRAM work must name the device.
	GPUUUID string
}

// BrokerAdmission adapts packages/capacity.Broker to Admission.
type BrokerAdmission struct {
	broker ResourceBroker
	opts   BrokerAdmissionOptions
}

// NewBrokerAdmission wraps a capacity broker. A nil broker is an error: silently
// granting capacity would defeat the point of the adapter.
func NewBrokerAdmission(broker ResourceBroker, opts BrokerAdmissionOptions) (*BrokerAdmission, error) {
	if broker == nil {
		return nil, errors.New("schedule: broker admission requires a capacity broker")
	}
	opts.OwnerPrefix = strings.Trim(strings.TrimSpace(opts.OwnerPrefix), ":")
	if opts.OwnerPrefix == "" {
		return nil, errors.New("schedule: broker admission requires an owner prefix")
	}
	return &BrokerAdmission{broker: broker, opts: opts}, nil
}

// Acquire admits one batch. Deferred work always asks without reclaim, so a
// background job can never evict a resident interactive model.
func (a *BrokerAdmission) Acquire(ctx context.Context, req AdmissionRequest) (Lease, Verdict, error) {
	if err := req.Validate(); err != nil {
		return nil, Verdict{Kind: VerdictDeny, Reason: err.Error()}, err
	}
	if req.ResourceClass == ResourceClassNone {
		return releasedLease{}, Verdict{Kind: VerdictGrant}, nil
	}
	ownerID := a.ownerID(req)
	if req.ResourceClass == ResourceClassVRAM {
		if a.opts.GPUUUID == "" {
			return nil, Verdict{Kind: VerdictDeny, Reason: "scheduler has no bound GPU identity; VRAM work cannot be admitted"}, nil
		}
		lease, verdict, err := a.broker.AcquireGPU(ctx, ownerID, a.opts.GPUUUID, req.Estimate.VRAMBytes)
		if err != nil {
			return nil, mapCapacityVerdict(verdict), err
		}
		if lease == nil {
			return nil, mapCapacityVerdict(verdict), nil
		}
		return lease, mapCapacityVerdict(verdict), nil
	}
	lease, verdict, err := a.broker.AcquireResources(ctx, capacity.Request{
		OwnerID:   ownerID,
		RAMBytes:  req.Estimate.RAMBytes,
		CPUMilli:  0,
		Priority:  string(req.Priority),
		NoReclaim: true,
	})
	if err != nil {
		return nil, mapCapacityVerdict(verdict), err
	}
	if lease == nil {
		return nil, mapCapacityVerdict(verdict), nil
	}
	return lease, mapCapacityVerdict(verdict), nil
}

func (a *BrokerAdmission) ownerID(req AdmissionRequest) string {
	kind := strings.ReplaceAll(strings.TrimSpace(req.Kind), ":", "-")
	return a.opts.OwnerPrefix + ":" + kind + ":" + req.JobID
}

func mapCapacityVerdict(verdict capacity.Verdict) Verdict {
	out := Verdict{Kind: verdict.Kind, Reason: verdict.Reason}
	if out.Kind == "" {
		out.Kind = VerdictDeny
	}
	if out.Kind != VerdictGrant && out.Kind != VerdictDegrade && out.Kind != VerdictQueue && out.Kind != VerdictDeny {
		// An unknown verdict kind is a denial, not a grant. Failing closed is the
		// only safe interpretation of an admission answer we cannot read.
		out = Verdict{Kind: VerdictDeny, Reason: "unrecognised capacity verdict " + verdict.Kind + ": " + verdict.Reason}
	}
	return out
}

type releasedLease struct{}

func (releasedLease) Release(context.Context) error { return nil }

// HostState is a read-only host observation. It carries raw readings; the
// policy's IdleLoadBelow and IdleRAMFreeBytes decide what counts as idle, so
// idle semantics live in one place.
type HostState struct {
	Load1             float64   `json:"load1"`
	AvailableRAMBytes uint64    `json:"availableRamBytes"`
	ObservedAt        time.Time `json:"observedAt"`
}

// HostObserver supplies host observations to idle gating and status.
type HostObserver interface {
	Observe(ctx context.Context) (HostState, error)
}

// HostObserverFunc adapts a function to HostObserver.
type HostObserverFunc func(ctx context.Context) (HostState, error)

// Observe implements HostObserver.
func (f HostObserverFunc) Observe(ctx context.Context) (HostState, error) { return f(ctx) }

// HostObservationSource is the slice of packages/capacity.Broker that observes
// host state without creating a claim.
type HostObservationSource interface {
	ObserveHostState(ctx context.Context) (capacity.HostObservation, error)
}

// BrokerObserver adapts the capacity broker's observation to HostState.
type BrokerObserver struct {
	Source HostObservationSource
}

// Observe implements HostObserver.
func (o BrokerObserver) Observe(ctx context.Context) (HostState, error) {
	if o.Source == nil {
		return HostState{}, errors.New("schedule: host observation source is unavailable")
	}
	observation, err := o.Source.ObserveHostState(ctx)
	if err != nil {
		return HostState{}, err
	}
	return HostState{
		Load1:             observation.Load1,
		AvailableRAMBytes: observation.AvailableRAMBytes,
		ObservedAt:        time.Now().UTC(),
	}, nil
}
