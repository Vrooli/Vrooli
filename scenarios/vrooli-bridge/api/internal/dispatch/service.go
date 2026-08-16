package dispatch

import (
	"context"
	"errors"
	"strings"
	"time"
)

func isDeviceScoped(verb string) bool {
	return strings.HasPrefix(strings.TrimSpace(verb), "device-control ")
}

// DefaultTimeoutSeconds is applied when a dispatch passes timeout_seconds <= 0,
// so a job always carries a finite wall-clock budget the node aborts on.
const DefaultTimeoutSeconds int64 = 1800 // 30 minutes

// Service is the application-layer surface the dispatch handler depends on. It
// orchestrates the full safety sequence: resolve the node, enforce the
// allowlist + scopes, audit, create the durable run, and push the typed job.
type Service interface {
	// Dispatch validates and (unless a dry-run) dispatches the job. It returns
	// a Decision on success, or a typed sentinel naming the rejection reason. A
	// rejection is audited before the error is returned.
	Dispatch(ctx context.Context, in DispatchInput) (Decision, error)
}

type service struct {
	nodes          NodeReader
	presence       Presence
	runs           RunController
	audit          AuditSink
	pusher         JobPusher
	manifest       []string
	outputs        map[string][]ArtifactOutput
	defaultTimeout int64
	leases         DeviceLeaseStore
}

// Option customises the service (manifest override, default timeout).
type Option func(*service)

// WithManifest overrides the recognised verb-namespace allowlist.
func WithManifest(manifest []string) Option {
	return func(s *service) { s.manifest = manifest }
}

func WithManifestOutputs(outputs map[string][]ArtifactOutput) Option {
	return func(s *service) { s.outputs = outputs }
}

// WithDefaultTimeout overrides the timeout applied when a job passes <= 0.
func WithDefaultTimeout(seconds int64) Option {
	return func(s *service) {
		if seconds > 0 {
			s.defaultTimeout = seconds
		}
	}
}

// WithDeviceLeaseStore supplies the bridge-owned short-lived lease record.
func WithDeviceLeaseStore(store DeviceLeaseStore) Option {
	return func(s *service) {
		if store != nil {
			s.leases = store
		}
	}
}

// NewService constructs the production Service.
func NewService(nodes NodeReader, presence Presence, runsCtl RunController, sink AuditSink, pusher JobPusher, opts ...Option) Service {
	s := &service{
		nodes:          nodes,
		presence:       presence,
		runs:           runsCtl,
		audit:          sink,
		pusher:         pusher,
		manifest:       DefaultManifest,
		outputs:        defaultManifestOutputs,
		defaultTimeout: DefaultTimeoutSeconds,
		leases:         NewMemoryDeviceLeaseStore(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Dispatch(ctx context.Context, in DispatchInput) (Decision, error) {
	job := in.Job.trimmed()
	in.Job = job

	if isDeviceScoped(job.Verb) {
		if job.DeviceID == "" || job.LeaseToken == "" {
			s.auditReject(ctx, in, "device-scoped dispatch requires a device_id and held lease_token")
			return Decision{}, ErrDeviceLeaseRequired{DeviceID: job.DeviceID}
		}
		if !s.leases.Held(job.DeviceID, job.LeaseToken, time.Now().UTC()) {
			s.auditReject(ctx, in, "device-scoped dispatch requires a held lease")
			return Decision{}, ErrDeviceLeaseNotHeld{DeviceID: job.DeviceID}
		}
	}

	// 1. Resolve the target node.
	node, err := s.nodes.GetTarget(ctx, job.NodeID)
	if err != nil {
		var notFound ErrNodeNotFound
		if errors.As(err, &notFound) {
			return Decision{}, notFound
		}
		return Decision{}, err
	}

	// 2. The shared admission gate — used by both durable dispatch and the
	// short-lived channel relay so their decisions cannot drift.
	if err := Admit(job, node, s.manifest); err != nil {
		s.auditReject(ctx, in, err.Error())
		return Decision{}, err
	}

	// 3. The allowlist gate has completed. A rejection here is
	//    audited and surfaced before any run is created or anything is pushed.
	if err := Allow(job, node.Scopes, s.manifest); err != nil {
		s.auditReject(ctx, in, err.Error())
		return Decision{}, err
	}

	// 4. Dry-run: the job validated and would be dispatched, but we create no
	//    run, write no audit, and push nothing.
	if in.DryRun {
		return Decision{DryRun: true, Job: job}, nil
	}

	// 5. Protocol-compatibility gate (OT-P1-001): an online node whose agent
	//     protocol version is flagged (needs-update / incompatible) is excluded
	//     from work rather than mis-driven. Provisioning is exempt — bringing
	//     the node to a new revision is how the agent is updated.
	if s.presence.IsOnline(job.NodeID) && !s.presence.Dispatchable(job.NodeID) {
		s.auditReject(ctx, in, "node needs update (protocol incompatible)")
		return Decision{}, ErrNodeNeedsUpdate{ID: job.NodeID}
	}

	// 6. Create the durable run.
	timeout := job.TimeoutSeconds
	if timeout <= 0 {
		timeout = s.defaultTimeout
	}
	runID, err := s.runs.CreateRun(ctx, CreateRunInput{
		NodeID:         job.NodeID,
		Scenario:       job.Scenario,
		Verb:           job.Verb,
		Args:           job.Args,
		TimeoutSeconds: timeout,
	})
	if err != nil {
		return Decision{}, err
	}

	// 7. Audit the accepted dispatch. This is FAIL-CLOSED: remote code execution
	//    must be accountable, so if we cannot record the dispatch we abort the
	//    run and refuse rather than run un-audited.
	if err := s.audit.Record(ctx, Entry{
		Actor: in.Actor, NodeID: job.NodeID, Scenario: job.Scenario, Verb: job.Verb,
		Args: job.Args, Accepted: true, Detail: "dispatched", RunID: runID,
	}); err != nil {
		_ = s.runs.AbortRun(ctx, runID, "audit write failed")
		return Decision{}, err
	}

	// 8. Push the typed job to the node. If it does not land (the node dropped
	//    between the online check and the push), abort the run and fail; Phase 5
	//    adds a durable per-node queue that redelivers instead.
	pushed := PushedJob{
		RunID: runID, Scenario: job.Scenario, Verb: job.Verb, Args: job.Args, TimeoutSeconds: timeout,
		Outputs: append([]ArtifactOutput(nil), s.outputs[job.Verb]...),
	}
	queued := false
	var delivered int
	if aware, ok := s.pusher.(QueueAwarePusher); ok {
		delivered, queued, err = aware.PushJobOutcome(ctx, job.NodeID, pushed)
	} else {
		delivered, err = s.pusher.PushJob(ctx, job.NodeID, pushed)
	}
	if err != nil || delivered == 0 {
		_ = s.runs.AbortRun(ctx, runID, "job delivery failed")
		if err != nil {
			return Decision{}, err
		}
		return Decision{}, ErrDeliveryFailed{NodeID: job.NodeID}
	}

	return Decision{RunID: runID, Job: job, Queued: queued}, nil
}

// Admit applies the complete node/job authorization decision without causing
// side effects. Durable dispatch and channel relay both call this function so
// a command receives the same typed refusal before either path can reach a
// node. The manifest is explicit to keep tests and future manifest versions
// deterministic.
func Admit(job Job, node TargetNode, manifest []string) error {
	job = job.trimmed()
	if node.Revoked {
		return ErrNodeRevoked{ID: job.NodeID}
	}
	if node.Kind != "" && node.Kind != "agent" {
		return ErrUnsupportedNodeKind{ID: job.NodeID, Kind: node.Kind}
	}
	return Allow(job, node.Scopes, manifest)
}

// auditReject records a denied dispatch. It is best-effort: a rejection stands
// even if the audit write fails (the dispatch is denied regardless), but the
// record is the accountability default, so the call is always made.
func (s *service) auditReject(ctx context.Context, in DispatchInput, reason string) {
	_ = s.audit.Record(ctx, Entry{
		Actor: in.Actor, NodeID: in.Job.NodeID, Scenario: in.Job.Scenario, Verb: in.Job.Verb,
		Args: in.Job.Args, Accepted: false, Detail: reason,
	})
}
