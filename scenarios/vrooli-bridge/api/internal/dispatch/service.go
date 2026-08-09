package dispatch

import (
	"context"
	"errors"
)

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

	// 1. Resolve the target node.
	node, err := s.nodes.GetTarget(ctx, job.NodeID)
	if err != nil {
		var notFound ErrNodeNotFound
		if errors.As(err, &notFound) {
			return Decision{}, notFound
		}
		return Decision{}, err
	}

	// 2. A revoked node can run nothing.
	if node.Revoked {
		s.auditReject(ctx, in, "node revoked")
		return Decision{}, ErrNodeRevoked{ID: job.NodeID}
	}

	// 3. The allowlist gate — the heart of OT-P0-004. A rejection here is
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

	// 5. The node must currently hold a channel to receive the push.
	if !s.presence.IsOnline(job.NodeID) {
		s.auditReject(ctx, in, "node offline")
		return Decision{}, ErrNodeOffline{ID: job.NodeID}
	}

	// 5b. Protocol-compatibility gate (OT-P1-001): an online node whose agent
	//     protocol version is flagged (needs-update / incompatible) is excluded
	//     from work rather than mis-driven. Provisioning is exempt — bringing
	//     the node to a new revision is how the agent is updated.
	if !s.presence.Dispatchable(job.NodeID) {
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
	delivered, err := s.pusher.PushJob(ctx, job.NodeID, PushedJob{
		RunID: runID, Scenario: job.Scenario, Verb: job.Verb, Args: job.Args, TimeoutSeconds: timeout,
		Outputs: append([]ArtifactOutput(nil), s.outputs[job.Verb]...),
	})
	if err != nil || delivered == 0 {
		_ = s.runs.AbortRun(ctx, runID, "job delivery failed")
		if err != nil {
			return Decision{}, err
		}
		return Decision{}, ErrDeliveryFailed{NodeID: job.NodeID}
	}

	return Decision{RunID: runID, Job: job}, nil
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
