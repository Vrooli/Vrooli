// Package bridgepermissions adapts Bridge's authoritative node registry to the
// onboarding permission-provider contract. It contains no local permission
// mirror: reads and writes are resolved against the current Bridge machine and
// node state for every operation.
package bridgepermissions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID          = "bridge.node-scopes"
	Owner               = "vrooli-bridge"
	BridgeUpdateSubject = "registry.node.owner-fields"
)

type TokenProvider func(context.Context) (string, error)

// NodeRegistryClient and MachineClient are intentionally narrow generated
// client subsets, making the real Bridge boundary straightforward to fake in
// local tests without introducing a second implementation.
type NodeRegistryClient interface {
	ListNodes(context.Context, *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error)
	GetNode(context.Context, *connect.Request[registryv1.GetNodeRequest]) (*connect.Response[registryv1.GetNodeResponse], error)
	UpdateNode(context.Context, *connect.Request[registryv1.UpdateNodeRequest]) (*connect.Response[registryv1.UpdateNodeResponse], error)
}

type MachineClient interface {
	ListMachines(context.Context, *connect.Request[machinesv1.ListMachinesRequest]) (*connect.Response[machinesv1.ListMachinesResponse], error)
	GetMachine(context.Context, *connect.Request[machinesv1.GetMachineRequest]) (*connect.Response[machinesv1.GetMachineResponse], error)
}

type MutationAuthorizer func(context.Context) error

type Provider struct {
	nodes             NodeRegistryClient
	machines          MachineClient
	token             TokenProvider
	authorizeMutation MutationAuthorizer
	now               func() time.Time
}

type Option func(*Provider)

func WithMutationAuthorizer(authorize MutationAuthorizer) Option {
	return func(p *Provider) { p.authorizeMutation = authorize }
}
func WithClock(now func() time.Time) Option { return func(p *Provider) { p.now = now } }

func NewProvider(nodes NodeRegistryClient, machines MachineClient, token TokenProvider, opts ...Option) *Provider {
	p := &Provider{nodes: nodes, machines: machines, token: token, authorizeMutation: authz.RequireMutation, now: func() time.Time { return time.Now().UTC() }}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

var _ permissions.Provider = (*Provider)(nil)

func (*Provider) ProviderID() string { return ProviderID }
func (*Provider) Owner() string      { return Owner }

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return nil, err
	}
	ctx, err = p.bridgeContext(ctx)
	if err != nil {
		return nil, err
	}
	nodeReq := connect.NewRequest(&registryv1.ListNodesRequest{})
	withAuthorization(nodeReq, ctx)
	nodeResp, err := p.nodes.ListNodes(ctx, nodeReq)
	if err != nil {
		return nil, bridgeError("list nodes", err)
	}
	machineReq := connect.NewRequest(&machinesv1.ListMachinesRequest{})
	withAuthorization(machineReq, ctx)
	machineResp, err := p.machines.ListMachines(ctx, machineReq)
	if err != nil {
		return nil, bridgeError("list machines", err)
	}
	if nodeResp == nil || nodeResp.Msg == nil || machineResp == nil || machineResp.Msg == nil {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("Bridge returned an incomplete permission inventory"))
	}
	nodes := make(map[string]*registryv1.Node, len(nodeResp.Msg.GetNodes()))
	for _, node := range nodeResp.Msg.GetNodes() {
		if node != nil && node.GetId() != "" {
			nodes[node.GetId()] = node
		}
	}
	descriptors := make([]permissions.Descriptor, 0)
	for _, machine := range machineResp.Msg.GetMachines() {
		if machine == nil || machine.GetLifecycle() != "active" || machine.GetEnrollmentGeneration() <= 0 {
			continue
		}
		current, err := p.currentMachine(ctx, machine.GetId())
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		if current == nil || current.GetCurrentNode() == nil {
			continue
		}
		node := nodes[current.GetCurrentNode().GetNodeId()]
		if !currentMachineMatches(current, machine.GetId(), node) {
			continue
		}
		target := permissions.TargetBinding{MachineID: machine.GetId(), NodeID: node.GetId(), EnrollmentGeneration: uint64(current.GetMachine().GetEnrollmentGeneration())}
		descriptors = append(descriptors, permissions.Descriptor{Version: permissions.ContractVersion, ID: permissionID(target), Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "vrooli-bridge:write", SubjectKind: "node-scopes", SubjectID: node.GetId(), TargetBound: true, Target: target, Source: "Bridge NodeRegistryService + MachineService", SchemaVersion: "registry/v1", CanRead: true, CanWrite: true, Sensitivity: "operator"})
	}
	return descriptors, nil
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	if !d.CanRead {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("Bridge node-scope permission is not readable"))
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: decisionRef(actor.Subject, req.Target)}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !d.CanRead {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("Bridge node-scope permission is not readable"))
	}
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, errors.New("Bridge node-scope read permit mismatch")
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.ReadResult{}, err
	}
	ctx, err = p.bridgeContext(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	node, err := p.currentTarget(ctx, req.Target)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	value, err := json.Marshal(scopeSet{Scopes: node.GetScopes()})
	if err != nil {
		return permissions.ReadResult{}, err
	}
	digest := sha256.Sum256(value)
	return permissions.ReadResult{PermissionID: d.ID, Owner: d.Owner, Target: req.Target, Revision: strconv.FormatInt(node.GetOwnerRevision(), 10), Value: value, ValueDigest: hex.EncodeToString(digest[:]), ObservedAt: p.now().UTC()}, nil
}

func (p *Provider) Authorize(ctx context.Context, d permissions.Descriptor, req permissions.WriteRequest) (permissions.Decision, error) {
	if !d.CanWrite {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("Bridge node-scope permission is not writable"))
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	if strings.TrimSpace(req.ExpectedRevision) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return permissions.Decision{}, connect.NewError(connect.CodeInvalidArgument, errors.New("expected revision and idempotency key are required"))
	}
	if p.authorizeMutation == nil {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("Bridge scope mutation authorizer is unavailable"))
	}
	if err = p.authorizeMutation(ctx); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: decisionRef(actor.Subject, req.Target) + ":mutation"}, nil
}

func (p *Provider) Write(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.WriteRequest) (permissions.Receipt, error) {
	if !d.CanWrite {
		return permissions.Receipt{}, connect.NewError(connect.CodePermissionDenied, errors.New("Bridge node-scope permission is not writable"))
	}
	if !permit.ValidForWrite(d.ID, req) {
		return permissions.Receipt{}, errors.New("Bridge node-scope write permit mismatch")
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Receipt{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Receipt{}, err
	}
	expected, err := strconv.ParseInt(req.ExpectedRevision, 10, 64)
	if err != nil || expected <= 0 {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, errors.New("expected revision must be a positive Bridge owner revision"))
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, errors.New("request id is required"))
	}
	var wanted scopeSet
	if err = json.Unmarshal(req.Value, &wanted); err != nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("decode Bridge node scope set: %w", err))
	}
	wanted.Scopes, err = normalizeScopes(wanted.Scopes)
	if err != nil {
		return permissions.Receipt{}, err
	}
	ctx, err = p.bridgeContext(ctx)
	if err != nil {
		return permissions.Receipt{}, err
	}
	current, err := p.currentTarget(ctx, req.Target)
	if err != nil {
		return permissions.Receipt{}, err
	}
	if kindName(current.GetKind()) == "" {
		return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge node kind is unavailable for safe owner update"))
	}
	reqCAS := &registryv1.UpdateNodeRequest{Id: current.GetId(), Name: current.GetName(), Endpoint: current.GetEndpoint(), Capabilities: append([]string(nil), current.GetCapabilities()...), Scopes: wanted.Scopes, Revision: current.GetRevision(), Kind: current.GetKind(), DeploymentTier: current.GetDeploymentTier(), ExpectedRevision: &expected, RequestId: req.IdempotencyKey}
	updateReq := connect.NewRequest(reqCAS)
	withAuthorization(updateReq, ctx)
	response, err := p.nodes.UpdateNode(ctx, updateReq)
	if err != nil {
		return permissions.Receipt{}, bridgeError("update node scopes", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetReceipt() == nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeUnavailable, errors.New("Bridge scope update returned no durable receipt"))
	}
	wire := response.Msg.GetReceipt()
	if wire.GetRequestId() != req.IdempotencyKey || wire.GetActorId() != actor.Subject || wire.GetTargetId() != req.Target.NodeID || wire.GetSubject() != BridgeUpdateSubject || wire.GetExpectedRevision() != expected || wire.GetNewRevision() <= expected || wire.GetPayloadDigest() != bridgePayloadDigest(reqCAS) || wire.GetCreatedAt() == nil || !wire.GetCreatedAt().IsValid() {
		return permissions.Receipt{}, connect.NewError(connect.CodeDataLoss, errors.New("Bridge returned an unbound node update receipt"))
	}
	if response.Msg.GetNode() == nil || response.Msg.GetNode().GetId() != req.Target.NodeID || response.Msg.GetNode().GetOwnerRevision() < wire.GetNewRevision() {
		return permissions.Receipt{}, connect.NewError(connect.CodeDataLoss, errors.New("Bridge receipt does not match the returned node state"))
	}
	digest := sha256.Sum256(req.Value)
	return permissions.Receipt{ID: "bridge-node-update:" + wire.GetRequestId(), PermissionID: d.ID, Owner: d.Owner, ActorID: wire.GetActorId(), Target: req.Target, ExpectedRevision: strconv.FormatInt(wire.GetExpectedRevision(), 10), ResultRevision: strconv.FormatInt(wire.GetNewRevision(), 10), IdempotencyKey: wire.GetRequestId(), RequestDigest: hex.EncodeToString(digest[:]), DecisionRef: permit.DecisionRef(), Outcome: "updated", At: wire.GetCreatedAt().AsTime().UTC()}, nil
}

type scopeSet struct {
	Scopes []string `json:"scopes"`
}

func (p *Provider) currentTarget(ctx context.Context, target permissions.TargetBinding) (*registryv1.Node, error) {
	if target.MachineID == "" || target.NodeID == "" || target.EnrollmentGeneration == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge permission target binding is incomplete"))
	}
	machine, err := p.currentMachine(ctx, target.MachineID)
	if err != nil {
		return nil, err
	}
	nodeReq := connect.NewRequest(&registryv1.GetNodeRequest{Id: target.NodeID})
	withAuthorization(nodeReq, ctx)
	resp, err := p.nodes.GetNode(ctx, nodeReq)
	if err != nil {
		return nil, bridgeError("get target node", err)
	}
	if resp == nil || resp.Msg == nil {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("Bridge returned no node state"))
	}
	node := resp.Msg.GetNode()
	if node == nil || node.GetId() != target.NodeID || node.GetRevokedAt() != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge target node is missing or revoked"))
	}
	if !currentMachineMatches(machine, target.MachineID, node) || uint64(machine.Machine.GetEnrollmentGeneration()) != target.EnrollmentGeneration {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge machine, current node, or enrollment generation changed"))
	}
	if node.GetOwnerRevision() <= 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge node owner revision is unavailable"))
	}
	return node, nil
}

func (p *Provider) currentMachine(ctx context.Context, id string) (*machinesv1.GetMachineResponse, error) {
	if strings.TrimSpace(id) == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge machine id is required"))
	}
	req := connect.NewRequest(&machinesv1.GetMachineRequest{Id: id})
	withAuthorization(req, ctx)
	resp, err := p.machines.GetMachine(ctx, req)
	if err != nil {
		return nil, bridgeError("get machine", err)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMachine() == nil {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("Bridge returned no machine state"))
	}
	if resp.Msg.GetMachine().GetLifecycle() != "active" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge machine is not active"))
	}
	return resp.Msg, nil
}

func currentMachineMatches(detail *machinesv1.GetMachineResponse, machineID string, node *registryv1.Node) bool {
	if detail == nil || detail.GetMachine() == nil || detail.GetCurrentNode() == nil || node == nil {
		return false
	}
	machine, projection := detail.GetMachine(), detail.GetCurrentNode()
	if machine.GetId() != machineID || machine.GetLifecycle() != "active" || machine.GetEnrollmentGeneration() <= 0 || machine.GetEnrollmentGeneration() != node.GetEnrollmentGeneration() || projection.GetNodeId() != node.GetId() || projection.GetEnrollmentGeneration() != machine.GetEnrollmentGeneration() {
		return false
	}
	current := false
	for _, lineage := range machine.GetNodeLineage() {
		if lineage != nil && lineage.GetCurrent() {
			if current {
				return false
			}
			current = lineage.GetNodeId() == node.GetId() && lineage.GetEnrollmentGeneration() == machine.GetEnrollmentGeneration()
		}
	}
	return current && node.GetRevokedAt() == nil && node.GetStatus() != registryv1.NodeStatus_NODE_STATUS_REVOKED
}

func (p *Provider) bridgeContext(ctx context.Context) (context.Context, error) {
	if p.nodes == nil || p.machines == nil || p.token == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("Bridge permission provider is unavailable"))
	}
	token, err := p.token(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("obtain Bridge owner session: %w", err))
	}
	if strings.TrimSpace(token) == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("Bridge owner session is unavailable"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, authHeaderKey{}, token), nil
}

type authHeaderKey struct{}

func bridgeError(op string, err error) error {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return err
	}
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("Bridge %s unavailable: %w", op, err))
}
func isNotFound(err error) bool { return connect.CodeOf(err) == connect.CodeNotFound }
func verifiedActor(ctx context.Context) (identity.Principal, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || !p.IsHuman() {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified human operator required"))
	}
	return p, nil
}
func validateBinding(actor identity.Principal, d permissions.Descriptor, actorID, owner string, target permissions.TargetBinding) error {
	if d.Version != permissions.ContractVersion || d.ID != permissionID(target) || d.Owner != Owner || owner != Owner || actor.Subject != actorID || d.PrincipalID != actorID || d.PrincipalKind != string(actor.Kind) || d.Scope != "vrooli-bridge:write" || !d.TargetBound || d.Target != target || d.SubjectKind != "node-scopes" || d.SubjectID != target.NodeID || d.Source != "Bridge NodeRegistryService + MachineService" || d.SchemaVersion != "registry/v1" || d.Sensitivity != "operator" {
		return connect.NewError(connect.CodePermissionDenied, errors.New("Bridge node scope permission actor, owner, subject, or target mismatch"))
	}
	return nil
}
func permissionID(t permissions.TargetBinding) string {
	return "bridge/node-scopes/" + t.MachineID + "/" + t.NodeID
}
func decisionRef(actor string, t permissions.TargetBinding) string {
	return "bridge-owner:" + actor + ":" + t.MachineID + ":" + t.NodeID + ":" + strconv.FormatUint(t.EnrollmentGeneration, 10)
}
func normalizeScopes(scopes []string) ([]string, error) {
	out := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}
	for _, scope := range scopes {
		s := strings.TrimSpace(scope)
		if s == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scope values cannot be empty"))
		}
		if _, ok := seen[s]; ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duplicate scope value"))
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}

func bridgePayloadDigest(req *registryv1.UpdateNodeRequest) string {
	payload := struct {
		ID, Name, Endpoint             string
		Capabilities, Scopes           []string
		Revision, Kind, DeploymentTier string
	}{
		strings.TrimSpace(req.GetId()), strings.TrimSpace(req.GetName()), strings.TrimSpace(req.GetEndpoint()),
		trimAll(req.GetCapabilities()), trimAll(req.GetScopes()), strings.TrimSpace(req.GetRevision()),
		kindName(req.GetKind()), strings.TrimSpace(req.GetDeploymentTier()),
	}
	encoded, _ := json.Marshal(payload)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func trimAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, strings.TrimSpace(v))
	}
	return out
}

func kindName(kind registryv1.NodeKind) string {
	switch kind {
	case registryv1.NodeKind_NODE_KIND_AGENT:
		return "agent"
	case registryv1.NodeKind_NODE_KIND_SSH:
		return "ssh"
	case registryv1.NodeKind_NODE_KIND_ATTACHED:
		return "attached"
	case registryv1.NodeKind_NODE_KIND_CONTROL_PLANE:
		return "control_plane"
	default:
		return ""
	}
}

func withAuthorization[T any](request *connect.Request[T], ctx context.Context) {
	if token, ok := ctx.Value(authHeaderKey{}).(string); ok {
		request.Header().Set("Authorization", token)
	}
}
