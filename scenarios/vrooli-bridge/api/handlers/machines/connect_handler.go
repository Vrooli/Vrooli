package machines

import (
	"context"
	"errors"
	"log"

	"vrooli-bridge/internal/auth"
	internalmachines "vrooli-bridge/internal/machines"
	internalonboard "vrooli-bridge/internal/onboard"

	"connectrpc.com/connect"

	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
)

const dryRunHeader = "X-Dry-Run"

// service is deliberately the small owner-facing Machine surface. Lifecycle
// cleanup, trust, and policy storage remain internal collaborators rather than
// being exposed accidentally by the HTTP handler.
type service interface {
	Create(context.Context, internalmachines.CreateInput) (internalmachines.Machine, error)
	Get(context.Context, string) (internalmachines.Machine, error)
	List(context.Context) ([]internalmachines.Machine, error)
	Archive(context.Context, string, int64) (internalmachines.Machine, error)
	Remove(context.Context, string, int64) (internalmachines.Machine, error)
	Merge(context.Context, internalmachines.MergeInput) (internalmachines.Machine, error)
	GetTrust(context.Context, string) (internalmachines.TrustRecord, error)
	ReviewHostKey(context.Context, string, string) (internalmachines.TrustRecord, error)
	CreateCleanupTombstone(context.Context, internalmachines.CleanupTombstone) (internalmachines.CleanupTombstone, error)
	ListCleanupTombstones(context.Context, string) ([]internalmachines.CleanupTombstone, error)
	UpdateCleanupTombstone(context.Context, string, internalmachines.CleanupStatus, string) (internalmachines.CleanupTombstone, error)
	ApplyPolicy(context.Context, internalmachines.PolicyChangeInput) (internalmachines.Machine, internalmachines.PolicySnapshot, error)
}

type Deps struct {
	Service         service
	Attempts        attemptReader
	Projection      projectionReader
	Audit           auditAppender
	HostKeyResetter HostKeyResetter
	NodeRevoker     NodeRevoker
	Repairer        Repairer
	Logger          *log.Logger
}

type auditAppender interface {
	AppendAudit(context.Context, internalmachines.AuditEvent) error
}

type auditReader interface {
	ListAudit(context.Context, string) ([]internalmachines.AuditEvent, error)
}

type attemptReader interface {
	ListAttemptsForMachine(context.Context, string) ([]internalonboard.EnrollmentAttempt, error)
}

type projectionReader interface {
	Compose(context.Context, internalmachines.Machine) (internalmachines.Projection, error)
}

// HostKeyResetter only clears the reviewed Machine host from Bridge-owned
// known_hosts so the normal strict first-touch path can pin it again. It is not
// a remote execution capability.
type HostKeyResetter interface {
	ForgetHostKey(host string, port int) error
}

// NodeRevoker composes the three local revocation effects in order: Registry
// durable identity, pairing credential, then live channel. SSH cleanup remains
// a separately recorded remote action.
type NodeRevoker interface {
	RevokeMachineNode(ctx context.Context, nodeID string) error
}

// Repairer starts the normal Machine enrollment flow with the existing
// Machine identity. The handler owns the API seam; the onboarding domain owns
// SSH, key reuse, and reconnect semantics.
type Repairer interface {
	Repair(context.Context, internalmachines.Machine) (opID, attemptID string, err error)
}

type connectHandler struct{ deps Deps }

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) CreateMachine(ctx context.Context, req *connect.Request[machinesv1.CreateMachineRequest]) (*connect.Response[machinesv1.CreateMachineResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	locators := make([]internalmachines.Locator, 0, len(req.Msg.Locators))
	for _, locator := range req.Msg.Locators {
		if locator == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("locators cannot contain null"))
		}
		locators = append(locators, internalmachines.Locator{Kind: locator.Kind, Value: locator.Value, Ordinal: int(locator.Ordinal)})
	}
	if req.Header().Get(dryRunHeader) == "true" {
		// Dry-runs validate the request path without persisting durable intent.
		// The synthetic ID is only carried to the dry-run onboarding validation;
		// no subsequent operation can observe or resolve it.
		return connect.NewResponse(&machinesv1.CreateMachineResponse{Machine: &machinesv1.Machine{Id: "dry-run-machine", Locators: req.Msg.Locators}}), nil
	}
	machine, err := h.deps.Service.Create(ctx, internalmachines.CreateInput{
		Locators:              locators,
		DesiredProfileID:      req.Msg.DesiredProfileId,
		DesiredProfileVersion: req.Msg.DesiredProfileVersion,
	})
	if err != nil {
		return nil, h.error("CreateMachine", err)
	}
	return connect.NewResponse(&machinesv1.CreateMachineResponse{Machine: domainToProto(machine)}), nil
}

func (h *connectHandler) GetMachine(ctx context.Context, req *connect.Request[machinesv1.GetMachineRequest]) (*connect.Response[machinesv1.GetMachineResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	machine, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.error("GetMachine", err)
	}
	response := &machinesv1.GetMachineResponse{Machine: domainToProto(machine), EnrollmentAttempts: make([]*machinesv1.EnrollmentAttempt, 0), AuditEvents: make([]*machinesv1.MachineAuditEvent, 0), CleanupTombstones: make([]*machinesv1.MachineCleanup, 0)}
	cleanups, err := h.deps.Service.ListCleanupTombstones(ctx, machine.ID)
	if err != nil {
		return nil, h.error("GetMachine cleanup tombstones", err)
	}
	for _, cleanup := range cleanups {
		response.CleanupTombstones = append(response.CleanupTombstones, cleanupToProto(cleanup))
	}
	if reader, ok := h.deps.Audit.(auditReader); ok {
		events, auditErr := reader.ListAudit(ctx, machine.ID)
		if auditErr != nil {
			return nil, h.error("GetMachine audit", auditErr)
		}
		for _, event := range events {
			response.AuditEvents = append(response.AuditEvents, auditToProto(event))
		}
	}
	if h.deps.Attempts == nil {
		return h.composeProjection(ctx, machine, response)
	}
	attempts, err := h.deps.Attempts.ListAttemptsForMachine(ctx, machine.ID)
	if err != nil {
		return nil, h.error("GetMachine enrollment attempts", err)
	}
	for _, attempt := range attempts {
		response.EnrollmentAttempts = append(response.EnrollmentAttempts, attemptToProto(attempt))
	}
	return h.composeProjection(ctx, machine, response)
}

func (h *connectHandler) composeProjection(ctx context.Context, machine internalmachines.Machine, response *machinesv1.GetMachineResponse) (*connect.Response[machinesv1.GetMachineResponse], error) {
	projection := internalmachines.Projection{Machine: machine}
	if h.deps.Projection != nil {
		var err error
		projection, err = h.deps.Projection.Compose(ctx, machine)
		if err != nil {
			return nil, h.error("GetMachine projection", err)
		}
		response.CurrentNode = projectionToProto(projection)
	}
	trust, err := h.deps.Service.GetTrust(ctx, machine.ID)
	if err != nil {
		trust = internalmachines.TrustRecord{MachineID: machine.ID, HostKeyState: internalmachines.HostKeyUnverified}
	}
	policy, err := internalmachines.ResolveProfile(machine.ID, machine.DesiredProfileID, machine.DesiredProfileVersion, nil)
	if err != nil {
		policy = internalmachines.PolicySnapshot{}
	}
	response.Readiness = readinessToProto(internalmachines.EvaluateReadiness(machine, trust, policy, projection))
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListMachines(ctx context.Context, _ *connect.Request[machinesv1.ListMachinesRequest]) (*connect.Response[machinesv1.ListMachinesResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	machines, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, h.error("ListMachines", err)
	}
	response := &machinesv1.ListMachinesResponse{Machines: make([]*machinesv1.Machine, 0, len(machines))}
	for _, machine := range machines {
		response.Machines = append(response.Machines, domainToProto(machine))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ArchiveMachine(ctx context.Context, req *connect.Request[machinesv1.ArchiveMachineRequest]) (*connect.Response[machinesv1.ArchiveMachineResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	machine, err := h.deps.Service.Archive(ctx, req.Msg.Id, req.Msg.Version)
	if err != nil {
		return nil, h.error("ArchiveMachine", err)
	}
	h.record(ctx, machine.ID, "archive", "")
	return connect.NewResponse(&machinesv1.ArchiveMachineResponse{Machine: domainToProto(machine)}), nil
}

func (h *connectHandler) RemoveMachine(ctx context.Context, req *connect.Request[machinesv1.RemoveMachineRequest]) (*connect.Response[machinesv1.RemoveMachineResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	machine, err := h.deps.Service.Remove(ctx, req.Msg.Id, req.Msg.Version)
	if err != nil {
		return nil, h.error("RemoveMachine", err)
	}
	h.record(ctx, machine.ID, "remove", "history preserved")
	return connect.NewResponse(&machinesv1.RemoveMachineResponse{Machine: domainToProto(machine)}), nil
}

func (h *connectHandler) GetMachineTrust(ctx context.Context, req *connect.Request[machinesv1.GetMachineTrustRequest]) (*connect.Response[machinesv1.GetMachineTrustResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	trust, err := h.deps.Service.GetTrust(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, h.error("GetMachineTrust", err)
	}
	return connect.NewResponse(&machinesv1.GetMachineTrustResponse{Trust: trustToProto(trust)}), nil
}

func (h *connectHandler) ReviewMachineHostKey(ctx context.Context, req *connect.Request[machinesv1.ReviewMachineHostKeyRequest]) (*connect.Response[machinesv1.ReviewMachineHostKeyResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.HostKeyResetter == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("host-key review is unavailable"))
	}
	machine, err := h.deps.Service.Get(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, h.error("ReviewMachineHostKey", err)
	}
	host := ""
	for _, locator := range machine.Locators {
		if locator.Kind == "hostname" || locator.Kind == "ip" {
			host = locator.Value
			break
		}
	}
	if host == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Machine has no hostname or IP locator for trust review"))
	}
	if err := h.deps.HostKeyResetter.ForgetHostKey(host, 22); err != nil {
		h.deps.Logger.Printf("machines.ReviewMachineHostKey(%q): reset known host: %v", req.Msg.MachineId, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	trust, err := h.deps.Service.ReviewHostKey(ctx, req.Msg.MachineId, req.Msg.ReplacementHostKeyFingerprint)
	if err != nil {
		return nil, h.error("ReviewMachineHostKey", err)
	}
	h.record(ctx, req.Msg.MachineId, "review_host_key", "")
	return connect.NewResponse(&machinesv1.ReviewMachineHostKeyResponse{Trust: trustToProto(trust)}), nil
}

func (h *connectHandler) RequestMachineSSHCleanup(ctx context.Context, req *connect.Request[machinesv1.RequestMachineSSHCleanupRequest]) (*connect.Response[machinesv1.RequestMachineSSHCleanupResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	// Verify intent exists before a cleanup record can be created; this avoids
	// a tombstone becoming an unauditable free-form task queue.
	if _, err := h.deps.Service.Get(ctx, req.Msg.MachineId); err != nil {
		return nil, h.error("RequestMachineSSHCleanup", err)
	}
	cleanup, err := h.deps.Service.CreateCleanupTombstone(ctx, internalmachines.CleanupTombstone{MachineID: req.Msg.MachineId, Action: "remove_ssh_access"})
	if err != nil {
		return nil, h.error("RequestMachineSSHCleanup", err)
	}
	h.record(ctx, req.Msg.MachineId, "request_ssh_cleanup", cleanup.ID)
	return connect.NewResponse(&machinesv1.RequestMachineSSHCleanupResponse{Cleanup: cleanupToProto(cleanup)}), nil
}

func (h *connectHandler) UpdateMachineCleanup(ctx context.Context, req *connect.Request[machinesv1.UpdateMachineCleanupRequest]) (*connect.Response[machinesv1.UpdateMachineCleanupResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	cleanup, err := h.deps.Service.UpdateCleanupTombstone(ctx, req.Msg.Id, internalmachines.CleanupStatus(req.Msg.Status), req.Msg.Detail)
	if err != nil {
		return nil, h.error("UpdateMachineCleanup", err)
	}
	h.record(ctx, cleanup.MachineID, "update_ssh_cleanup", string(cleanup.Status))
	return connect.NewResponse(&machinesv1.UpdateMachineCleanupResponse{Cleanup: cleanupToProto(cleanup)}), nil
}

func (h *connectHandler) ApplyMachinePolicy(ctx context.Context, req *connect.Request[machinesv1.ApplyMachinePolicyRequest]) (*connect.Response[machinesv1.ApplyMachinePolicyResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	machine, snapshot, err := h.deps.Service.ApplyPolicy(ctx, internalmachines.PolicyChangeInput{MachineID: req.Msg.MachineId, ExpectedVersion: req.Msg.Version, ProfileID: req.Msg.ProfileId, ProfileVersion: req.Msg.ProfileVersion, Overrides: req.Msg.Overrides, Actor: actor, Reason: req.Msg.Reason, ConfirmRemoval: req.Msg.ConfirmRemoval})
	if err != nil {
		return nil, h.error("ApplyMachinePolicy", err)
	}
	h.record(ctx, machine.ID, "apply_policy", snapshot.ProfileID+"@"+snapshot.ProfileVersion)
	return connect.NewResponse(&machinesv1.ApplyMachinePolicyResponse{Machine: domainToProto(machine), Policy: policyToProto(snapshot)}), nil
}

func (h *connectHandler) RevokeMachineNode(ctx context.Context, req *connect.Request[machinesv1.RevokeMachineNodeRequest]) (*connect.Response[machinesv1.RevokeMachineNodeResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.NodeRevoker == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("node revocation is unavailable"))
	}
	machine, err := h.deps.Service.Get(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, h.error("RevokeMachineNode", err)
	}
	nodeID := ""
	for _, lineage := range machine.Lineage {
		if lineage.Current {
			nodeID = lineage.NodeID
			break
		}
	}
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Machine has no current paired Node"))
	}
	if err := h.deps.NodeRevoker.RevokeMachineNode(ctx, nodeID); err != nil {
		return nil, h.error("RevokeMachineNode", err)
	}
	h.record(ctx, machine.ID, "revoke_node", nodeID)
	return connect.NewResponse(&machinesv1.RevokeMachineNodeResponse{Machine: domainToProto(machine), RevokedNodeId: nodeID}), nil
}

func (h *connectHandler) RepairMachine(ctx context.Context, req *connect.Request[machinesv1.RepairMachineRequest]) (*connect.Response[machinesv1.RepairMachineResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.Repairer == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("machine repair is unavailable"))
	}
	machine, err := h.deps.Service.Get(ctx, req.Msg.GetMachineId())
	if err != nil {
		return nil, h.error("RepairMachine", err)
	}
	opID, attemptID, err := h.deps.Repairer.Repair(ctx, machine)
	if err != nil {
		return nil, h.error("RepairMachine", err)
	}
	h.record(ctx, machine.ID, "repair", opID)
	return connect.NewResponse(&machinesv1.RepairMachineResponse{Machine: domainToProto(machine), OnboardingOpId: opID, EnrollmentAttemptId: attemptID}), nil
}

func (h *connectHandler) MergeMachines(ctx context.Context, req *connect.Request[machinesv1.MergeMachinesRequest]) (*connect.Response[machinesv1.MergeMachinesResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	machine, err := h.deps.Service.Merge(ctx, internalmachines.MergeInput{FromMachineID: req.Msg.GetFromMachineId(), IntoMachineID: req.Msg.GetIntoMachineId(), Actor: actor})
	if err != nil {
		return nil, h.error("MergeMachines", err)
	}
	h.record(ctx, machine.ID, "merge", req.Msg.GetFromMachineId())
	return connect.NewResponse(&machinesv1.MergeMachinesResponse{Machine: domainToProto(machine), ArchivedMachineId: req.Msg.GetFromMachineId()}), nil
}

func (h *connectHandler) record(ctx context.Context, machineID, action, detail string) {
	if h.deps.Audit == nil {
		return
	}
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return
	}
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	if err := h.deps.Audit.AppendAudit(ctx, internalmachines.AuditEvent{MachineID: machineID, Action: action, Actor: actor, Detail: detail}); err != nil {
		h.deps.Logger.Printf("machines audit action=%q machine=%q: %v", action, machineID, err)
	}
}

func (h *connectHandler) error(operation string, err error) error {
	var invalid internalmachines.ErrInvalid
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound internalmachines.ErrNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var conflict internalmachines.ErrConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeAborted, conflict)
	}
	var ambiguous internalmachines.ErrAmbiguous
	if errors.As(err, &ambiguous) {
		return connect.NewError(connect.CodeFailedPrecondition, ambiguous)
	}
	h.deps.Logger.Printf("machines.%s: %v", operation, err)
	return connect.NewError(connect.CodeInternal, err)
}
