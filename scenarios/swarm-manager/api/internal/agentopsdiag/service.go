// Package agentopsdiag is the Connect operator surface over the declarative
// agent-operations runtime (internal/opsrunner): resolve a binding, dry-run-
// validate an invocation, inspect a durable workflow instance and an immutable
// execution provenance, browse the operation catalog and per-target mode
// compatibility, read and administer the layered binding overrides, and observe
// migration state. Every RPC delegates the decision to the runtime (opsrunner +
// agentops) and maps the typed result onto the wire contract — it never
// re-implements precedence, compatibility, or digest logic, so the CLI/UI
// clients stay thin. Every RPC is read-only except PutBindingOverride /
// DeleteBindingOverride.
package agentopsdiag

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// Service implements the AgentOperationsService handler.
type Service struct {
	catalog    *opscatalog.Catalog
	resolver   *opsrunner.BindingResolver
	repo       *opsrunner.WorkflowRepo
	executions *opsrunner.ExecutionStore
	// overrides is the read side of the override store (contributing-layer
	// listings); writer is the write side. Both optional: nil disables the
	// override RPCs fail-closed.
	overrides opsrunner.OverrideStore
	writer    *opsrunner.OverrideWriter
	// modeDefs + checker back the mode-compatibility RPCs and the override
	// write validation. checker is the SAME LivePreparer-backed ModeChecker the
	// live invocation path resolves with; nil means mode checks are unavailable
	// (reads stay tolerant, writes refuse fail-closed).
	modeDefs map[string]operatingmode.Definition
	checker  agentops.ModeChecker
	// migrationStatusPath locates the Phase-8 migration-status document; empty
	// reports not-started.
	migrationStatusPath string
	// reconcileLoc + reconcileGrace back RunReconciliation (the operator-invoked
	// orphan-snapshot sweep, same function as the startup sweep). nil disables
	// the RPC fail-closed.
	reconcileLoc   opsrunner.DomainLocator
	reconcileGrace time.Duration
}

// NewService builds the service over the core runtime collaborators. Optional
// collaborators (mode registry, override administration, migration status) are
// attached via the With* builders.
func NewService(catalog *opscatalog.Catalog, resolver *opsrunner.BindingResolver, repo *opsrunner.WorkflowRepo, executions *opsrunner.ExecutionStore) *Service {
	return &Service{catalog: catalog, resolver: resolver, repo: repo, executions: executions}
}

// WithResolver replaces the binding resolver (used when the resolver must be
// constructed after the service, e.g. to share the service's mode checker).
func (s *Service) WithResolver(resolver *opsrunner.BindingResolver) *Service {
	s.resolver = resolver
	return s
}

// WithModes attaches the authored mode registry and the ModeChecker computed
// over it (production passes the same LivePreparer the live runner uses).
func (s *Service) WithModes(defs map[string]operatingmode.Definition, checker agentops.ModeChecker) *Service {
	s.modeDefs = defs
	s.checker = checker
	return s
}

// WithOverrideAdmin attaches the override store's read side (for
// contributing-layer listings) and write side (for Put/Delete).
func (s *Service) WithOverrideAdmin(store opsrunner.OverrideStore, writer *opsrunner.OverrideWriter) *Service {
	s.overrides = store
	s.writer = writer
	return s
}

// WithMigrationStatusPath sets where GetMigrationStatus reads the Phase-8
// migration-status document from.
func (s *Service) WithMigrationStatusPath(path string) *Service {
	s.migrationStatusPath = path
	return s
}

// WithReconciler attaches the domain locator and in-flight grace period backing
// RunReconciliation. Production passes the SAME locator (with scan roots) and
// grace the startup sweep uses, so an operator-invoked sweep behaves
// identically.
func (s *Service) WithReconciler(loc opsrunner.DomainLocator, grace time.Duration) *Service {
	s.reconcileLoc = loc
	s.reconcileGrace = grace
	return s
}

// RunReconciliation runs the orphan-snapshot sweep (idempotent, race-safe; the
// same opsrunner.ReconcileOrphanSnapshots the API runs at startup) and returns
// its report. Fails closed when no locator was attached.
func (s *Service) RunReconciliation(_ context.Context, _ *connect.Request[apipb.AgentOpsRunReconciliationRequest]) (*connect.Response[apipb.AgentOpsRunReconciliationResponse], error) {
	if s.reconcileLoc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("reconciliation is unavailable: no domain locator attached"))
	}
	report, err := opsrunner.ReconcileOrphanSnapshots(s.reconcileLoc, time.Now(), s.reconcileGrace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apipb.AgentOpsRunReconciliationResponse{
		DirsScanned:      int32(report.DirsScanned),
		SnapshotsSeen:    int32(report.SnapshotsSeen),
		Reaped:           append([]string(nil), report.Reaped...),
		SkippedTooRecent: int32(report.SkippedTooRecent),
	}), nil
}

// ModeChecker exposes the attached checker (nil when modes are unavailable) so
// the caller can share it with the binding resolver it constructs.
func (s *Service) ModeChecker() agentops.ModeChecker { return s.checker }

// RegisterConnectService mounts the AgentOperationsService handler on the router.
func RegisterConnectService(router *mux.Router, svc *Service) {
	path, handler := apiconnect.NewAgentOperationsServiceHandler(svc)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

var _ apiconnect.AgentOperationsServiceHandler = (*Service)(nil)

func (s *Service) targetRef(sel *apipb.AgentOpsTargetSelector) (opsrunner.TargetRef, error) {
	if sel == nil {
		return opsrunner.TargetRef{}, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}
	kind, ok := targetKindFromProto(sel.GetKind())
	if !ok {
		return opsrunner.TargetRef{}, connect.NewError(connect.CodeInvalidArgument, errors.New("unknown or unspecified target kind"))
	}
	return opsrunner.TargetRef{Kind: kind, ID: sel.GetId()}, nil
}

// ResolveBinding resolves the winning binding for an operation against a target.
func (s *Service) ResolveBinding(ctx context.Context, req *connect.Request[apipb.AgentOpsResolveBindingRequest]) (*connect.Response[apipb.AgentOpsResolveBindingResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	op := agentops.OperationID(req.Msg.GetOperation())
	// An omitted operation version means "the latest authored contract", the
	// same convention the catalog uses everywhere else. Resolving with a
	// literally empty version would skip every system-default binding that pins
	// an exact operation_version (all of the shipped catalog does), reporting
	// no-binding for operations a live Invoke resolves fine.
	version := req.Msg.GetOperationVersion()
	if version == "" {
		if lc, ok := s.catalog.Contract(op, ""); ok {
			version = lc.Contract.Version
		}
	}
	res, err := s.resolver.Resolve(ctx, opsrunner.InvokeRequest{
		Target: target, Operation: op, OperationVersion: version,
	})
	if err != nil {
		return nil, bindingError(err)
	}
	return connect.NewResponse(&apipb.AgentOpsResolveBindingResponse{
		Resolved:       resolvedBindingToProto(res.Binding),
		PolicyId:       res.PolicyID,
		PolicyRevision: res.PolicyRevision,
	}), nil
}

// ValidateInvocation dry-runs an invocation without starting a run.
func (s *Service) ValidateInvocation(ctx context.Context, req *connect.Request[apipb.AgentOpsValidateInvocationRequest]) (*connect.Response[apipb.AgentOpsValidateInvocationResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	op := agentops.OperationID(req.Msg.GetOperation())
	out := &apipb.AgentOpsValidateInvocationResponse{}

	lc, ok := s.catalog.Contract(op, req.Msg.GetOperationVersion())
	out.OperationDeclared = ok
	if !ok {
		return connect.NewResponse(out), nil
	}
	compatErr := agentops.CheckOperationTargetCompatibility(op, lc.Contract.TargetRequirements.Capabilities, target.Kind)
	out.TargetCompatible = compatErr == nil
	if compatErr != nil {
		if provided, ok := agentops.ProvidedCapabilities(target.Kind); ok {
			for _, need := range lc.Contract.TargetRequirements.Capabilities {
				if !provided[need] {
					out.MissingCapabilities = append(out.MissingCapabilities, capabilityToProto(need))
				}
			}
		}
		return connect.NewResponse(out), nil
	}
	// Resolve at the declared contract's exact version (Contract() already
	// mapped an omitted request version onto the latest authored one), so a
	// version-pinned system default is in scope exactly as it is for a live
	// Invoke.
	res, err := s.resolver.Resolve(ctx, opsrunner.InvokeRequest{Target: target, Operation: op, OperationVersion: lc.Contract.Version})
	if err != nil {
		// A fail-closed binding error is a validation result, not a transport error.
		return connect.NewResponse(out), nil
	}
	out.BindingResolved = true
	out.Resolved = resolvedBindingToProto(res.Binding)
	return connect.NewResponse(out), nil
}

// InspectWorkflow returns the durable workflow instance for a target.
func (s *Service) InspectWorkflow(_ context.Context, req *connect.Request[apipb.AgentOpsInspectWorkflowRequest]) (*connect.Response[apipb.AgentOpsInspectWorkflowResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	w, found, err := s.repo.Load(target.Kind, target.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &apipb.AgentOpsInspectWorkflowResponse{Found: found}
	if found {
		resp.Workflow = workflowToProto(w)
	}
	return connect.NewResponse(resp), nil
}

// InspectExecution returns the pinned provenance and reproducibility of an
// execution.
func (s *Service) InspectExecution(_ context.Context, req *connect.Request[apipb.AgentOpsInspectExecutionRequest]) (*connect.Response[apipb.AgentOpsInspectExecutionResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	snap, found, err := s.executions.Load(target.Kind, target.ID, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &apipb.AgentOpsInspectExecutionResponse{Found: found}
	if found {
		resp.Provenance = provenanceToProto(snap.Provenance)
		resp.Outcome = snap.Outcome
		resp.RecordedAt = snap.RecordedAt
		_, reproErr := s.executions.Reproduce(target.Kind, target.ID, req.Msg.GetExecutionId())
		resp.Reproducible = reproErr == nil
	}
	return connect.NewResponse(resp), nil
}

// bindingError maps the agentops fail-closed binding errors to Connect codes.
func bindingError(err error) *connect.Error {
	switch {
	case errors.Is(err, agentops.ErrNoBinding):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, agentops.ErrInvalidOverride), errors.Is(err, agentops.ErrIncompatibleMode):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, agentops.ErrDeletedRevision):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
