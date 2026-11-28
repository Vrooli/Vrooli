package bridgepermissions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeNodes struct {
	node                             *registryv1.Node
	listCalls, getCalls, updateCalls int
	lastUpdate                       *registryv1.UpdateNodeRequest
	lastAuthorization                string
	receipts                         map[string]*registryv1.NodeUpdateReceipt
	requests                         map[string]*registryv1.UpdateNodeRequest
	updateErr                        error
}

func (f *fakeNodes) ListNodes(_ context.Context, r *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error) {
	f.listCalls++
	f.lastAuthorization = r.Header().Get("Authorization")
	return connect.NewResponse(&registryv1.ListNodesResponse{Nodes: []*registryv1.Node{f.node}}), nil
}
func (f *fakeNodes) GetNode(_ context.Context, r *connect.Request[registryv1.GetNodeRequest]) (*connect.Response[registryv1.GetNodeResponse], error) {
	f.getCalls++
	f.lastAuthorization = r.Header().Get("Authorization")
	if f.node == nil || r.Msg.Id != f.node.Id {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("missing local fake node"))
	}
	return connect.NewResponse(&registryv1.GetNodeResponse{Node: f.node}), nil
}
func (f *fakeNodes) UpdateNode(_ context.Context, r *connect.Request[registryv1.UpdateNodeRequest]) (*connect.Response[registryv1.UpdateNodeResponse], error) {
	f.updateCalls++
	f.lastAuthorization = r.Header().Get("Authorization")
	f.lastUpdate = r.Msg
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	actor := strings.TrimPrefix(f.lastAuthorization, "LocalSession ")
	if prior := f.receipts[r.Msg.RequestId]; prior != nil {
		if f.receipts[r.Msg.RequestId].ActorId == actor && proto.Equal(f.requests[r.Msg.RequestId], r.Msg) {
			return connect.NewResponse(&registryv1.UpdateNodeResponse{Node: f.node, Receipt: prior}), nil
		}
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("local fake request identity conflict"))
	}
	if f.node.OwnerRevision != r.Msg.GetExpectedRevision() {
		return nil, connect.NewError(connect.CodeAborted, errors.New("local fake stale revision"))
	}
	f.node.Scopes = append([]string(nil), r.Msg.Scopes...)
	f.node.OwnerRevision++
	receipt := &registryv1.NodeUpdateReceipt{RequestId: r.Msg.RequestId, ActorId: actor, TargetId: f.node.Id, Subject: BridgeUpdateSubject, ExpectedRevision: r.Msg.GetExpectedRevision(), NewRevision: f.node.OwnerRevision, PayloadDigest: bridgePayloadDigest(r.Msg), CreatedAt: timestamppb.New(time.Date(2026, 11, 20, 6, 0, 0, 0, time.UTC))}
	f.receipts[r.Msg.RequestId] = receipt
	f.requests[r.Msg.RequestId] = proto.Clone(r.Msg).(*registryv1.UpdateNodeRequest)
	return connect.NewResponse(&registryv1.UpdateNodeResponse{Node: f.node, Receipt: receipt}), nil
}

type fakeMachines struct {
	machine             *machinesv1.GetMachineResponse
	listCalls, getCalls int
	lastAuthorization   string
}

func (f *fakeMachines) ListMachines(_ context.Context, r *connect.Request[machinesv1.ListMachinesRequest]) (*connect.Response[machinesv1.ListMachinesResponse], error) {
	f.listCalls++
	f.lastAuthorization = r.Header().Get("Authorization")
	return connect.NewResponse(&machinesv1.ListMachinesResponse{Machines: []*machinesv1.Machine{f.machine.GetMachine()}}), nil
}
func (f *fakeMachines) GetMachine(_ context.Context, r *connect.Request[machinesv1.GetMachineRequest]) (*connect.Response[machinesv1.GetMachineResponse], error) {
	f.getCalls++
	f.lastAuthorization = r.Header().Get("Authorization")
	if f.machine == nil || f.machine.GetMachine().GetId() != r.Msg.Id {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("missing local fake machine"))
	}
	return connect.NewResponse(f.machine), nil
}

func fixture() (*fakeNodes, *fakeMachines, *Provider, context.Context) {
	node := &registryv1.Node{Id: "node-a", Name: "Local fixture", Scopes: []string{"vrooli-onboarding:read"}, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE, EnrollmentGeneration: 4, OwnerRevision: 7, Kind: registryv1.NodeKind_NODE_KIND_AGENT}
	machine := &machinesv1.Machine{Id: "machine-a", Lifecycle: "active", EnrollmentGeneration: 4, NodeLineage: []*machinesv1.NodeLineage{{NodeId: "node-a", Current: true, EnrollmentGeneration: 4}}}
	current := &machinesv1.CurrentNodeProjection{NodeId: "node-a", EnrollmentGeneration: 4}
	nodes := &fakeNodes{node: node, receipts: map[string]*registryv1.NodeUpdateReceipt{}, requests: map[string]*registryv1.UpdateNodeRequest{}}
	machines := &fakeMachines{machine: &machinesv1.GetMachineResponse{Machine: machine, CurrentNode: current}}
	p := NewProvider(nodes, machines, func(ctx context.Context) (string, error) {
		principal, ok := identity.PrincipalFromContext(ctx)
		if !ok {
			return "", errors.New("no principal")
		}
		return "LocalSession " + principal.Subject, nil
	}, WithMutationAuthorizer(func(context.Context) error { return nil }), WithClock(func() time.Time { return time.Date(2026, 11, 20, 6, 0, 0, 0, time.UTC) }))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true})
	return nodes, machines, p, ctx
}

func TestListAndReadAreBridgeAuthoritativeAndGenerationBound(t *testing.T) {
	nodes, machines, p, ctx := fixture()
	descriptors, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v", descriptors)
	}
	d := descriptors[0]
	if d.Target != (permissions.TargetBinding{MachineID: "machine-a", NodeID: "node-a", EnrollmentGeneration: 4}) || d.PrincipalID != "operator-a" {
		t.Fatalf("descriptor=%+v", d)
	}
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	read, err := r.Read(ctx, d.ID, permissions.ReadRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target})
	if err != nil {
		t.Fatal(err)
	}
	if read.Revision != "7" || !strings.Contains(string(read.Value), "vrooli-onboarding:read") {
		t.Fatalf("read=%+v", read)
	}
	if nodes.listCalls == 0 || nodes.getCalls == 0 || machines.listCalls == 0 || machines.getCalls == 0 || nodes.lastAuthorization != "LocalSession operator-a" || machines.lastAuthorization != "LocalSession operator-a" {
		t.Fatalf("Bridge local-fake calls/auth not propagated: nodes=%+v machines=%+v", nodes, machines)
	}

	machines.machine.CurrentNode.EnrollmentGeneration = 3
	_, err = r.Read(ctx, d.ID, permissions.ReadRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target})
	if err == nil {
		t.Fatal("stale generation must remove the descriptor and refuse the read")
	}
}

func TestReadAndWriteValidatePermitBeforeBridgeCalls(t *testing.T) {
	nodes, _, p, ctx := fixture()
	d := descriptorFixture()
	readReq := permissions.ReadRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target}
	if _, err := p.Read(ctx, d, permissions.Permit{}, readReq); err == nil {
		t.Fatal("missing read permit should refuse")
	}
	writeReq := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target, ExpectedRevision: "7", IdempotencyKey: "request-1", Value: []byte(`{"scopes":["vrooli-onboarding:read"]}`)}
	if _, err := p.Write(ctx, d, permissions.Permit{}, writeReq); err == nil {
		t.Fatal("missing write permit should refuse")
	}
	if nodes.listCalls+nodes.getCalls+nodes.updateCalls != 0 {
		t.Fatalf("invalid permits reached Bridge: %+v", nodes)
	}
}

func TestWriteUsesCASAndMapsDurableSecretFreeReceipt(t *testing.T) {
	nodes, _, p, ctx := fixture()
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	d := descriptorFixture()
	value := []byte(`{"scopes":["vrooli-onboarding:read","vrooli-onboarding:write"]}`)
	req := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target, ExpectedRevision: "7", IdempotencyKey: "request-1", Value: value}
	receipt, err := r.Write(ctx, d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if nodes.updateCalls != 1 || nodes.lastUpdate.GetExpectedRevision() != 7 || nodes.lastUpdate.GetRequestId() != req.IdempotencyKey || nodes.lastAuthorization != "LocalSession operator-a" {
		t.Fatalf("CAS request=%+v calls=%d auth=%q", nodes.lastUpdate, nodes.updateCalls, nodes.lastAuthorization)
	}
	if receipt.ActorID != "operator-a" || receipt.Target != req.Target || receipt.ExpectedRevision != "7" || receipt.ResultRevision != "8" || receipt.IdempotencyKey != "request-1" || receipt.Outcome != "updated" {
		t.Fatalf("receipt=%+v", receipt)
	}
	if strings.Contains(fmt.Sprintf("%+v", receipt), "vrooli-onboarding:write") {
		t.Fatal("receipt unexpectedly included a scope value")
	}
	if err := jsonUnmarshalScopes(value, nodes.lastUpdate.GetScopes()); err != nil {
		t.Fatal(err)
	}
	replayed, replayErr := r.Write(ctx, d.ID, req)
	if replayErr != nil {
		t.Fatalf("same request should return receipt: %v", replayErr)
	}
	if replayed != receipt {
		t.Fatalf("replayed receipt=%+v, first receipt=%+v", replayed, receipt)
	}
	if nodes.updateCalls != 2 {
		t.Fatalf("exact retry not sent through Bridge: calls=%d", nodes.updateCalls)
	}
}

func TestWriteMissingPreconditionsAndStaleCASHaveNoUpdateEffect(t *testing.T) {
	nodes, _, p, ctx := fixture()
	d := descriptorFixture()
	missing := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target}
	if _, err := p.Authorize(ctx, d, missing); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing preconditions error=%v", err)
	}
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Write(ctx, d.ID, permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target, ExpectedRevision: "6", IdempotencyKey: "stale", Value: []byte(`{"scopes":[]}`)})
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale CAS error=%v", err)
	}
	if nodes.updateCalls != 1 {
		t.Fatalf("Bridge CAS must receive stale precondition and reject atomically: calls=%d", nodes.updateCalls)
	}
	if nodes.node.OwnerRevision != 7 || len(nodes.node.Scopes) != 1 || nodes.node.Scopes[0] != "vrooli-onboarding:read" {
		t.Fatalf("stale CAS changed state: node=%+v", nodes.node)
	}
	if _, err = r.Write(ctx, d.ID, permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target, ExpectedRevision: "not-a-number", IdempotencyKey: "bad", Value: []byte(`{"scopes":[]}`)}); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("malformed revision error=%v", err)
	}
}

func TestBridgeErrorClassesAndRequestIdentityConflictsArePreserved(t *testing.T) {
	for _, code := range []connect.Code{connect.CodePermissionDenied, connect.CodeUnauthenticated, connect.CodeNotFound, connect.CodeAborted, connect.CodeAlreadyExists} {
		err := bridgeError("test", connect.NewError(code, errors.New("local fake")))
		if connect.CodeOf(err) != code {
			t.Fatalf("code %s became %s", code, connect.CodeOf(err))
		}
	}
	if connect.CodeOf(bridgeError("test", errors.New("local fake unavailable"))) != connect.CodeUnavailable {
		t.Fatal("transport failure was not typed unavailable")
	}
	nodes, _, p, ctx := fixture()
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	d := descriptorFixture()
	req := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target, ExpectedRevision: "7", IdempotencyKey: "request-1", Value: []byte(`{"scopes":["vrooli-onboarding:write"]}`)}
	if _, err = r.Write(ctx, d.ID, req); err != nil {
		t.Fatal(err)
	}
	changed := req
	changed.Value = []byte(`{"scopes":["vrooli-onboarding:read"]}`)
	if _, err = r.Write(ctx, d.ID, changed); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed payload conflict error=%v", err)
	}
	if nodes.updateCalls != 2 {
		t.Fatalf("conflict should be checked by Bridge durable idempotency: calls=%d", nodes.updateCalls)
	}
}

func TestRequestReuseWithChangedActorOrExpectedRevisionConflicts(t *testing.T) {
	nodes, _, p, actorA := fixture()
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	d := descriptorFixture()
	base := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target, ExpectedRevision: "7", IdempotencyKey: "same-request", Value: []byte(`{"scopes":["vrooli-onboarding:write"]}`)}
	if _, err = r.Write(actorA, d.ID, base); err != nil {
		t.Fatal(err)
	}
	actorB := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-b", Verified: true})
	changedActor := base
	changedActor.ActorID = "operator-b"
	if _, err = r.Write(actorB, d.ID, changedActor); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("actor reuse conflict error=%v", err)
	}
	changedRevision := base
	changedRevision.ExpectedRevision = "6"
	if _, err = r.Write(actorA, d.ID, changedRevision); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("expected revision reuse conflict error=%v", err)
	}
	if nodes.updateCalls != 3 {
		t.Fatalf("Bridge must authoritatively resolve duplicate identities, calls=%d", nodes.updateCalls)
	}
}

func TestChangedTargetOrSubjectIsRefusedBeforeBridgeCall(t *testing.T) {
	nodes, _, p, ctx := fixture()
	d := descriptorFixture()
	req := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: d.Target, ExpectedRevision: "7", IdempotencyKey: "same-request", Value: []byte(`{"scopes":[]}`)}
	changedTarget := req
	changedTarget.Target.EnrollmentGeneration++
	if _, err := p.Authorize(ctx, d, changedTarget); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("target reuse error=%v", err)
	}
	changedSubject := d
	changedSubject.SubjectID = "other-node"
	if _, err := p.Authorize(ctx, changedSubject, req); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("subject reuse error=%v", err)
	}
	if nodes.listCalls+nodes.getCalls+nodes.updateCalls != 0 {
		t.Fatalf("mismatched target/subject reached Bridge: %+v", nodes)
	}
}

func TestMissingBridgeOwnerSessionFailsClosed(t *testing.T) {
	nodes, machines, _, ctx := fixture()
	p := NewProvider(nodes, machines, func(context.Context) (string, error) { return "", nil })
	_, err := p.List(ctx)
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("missing owner session error=%v", err)
	}
	if nodes.listCalls+machines.listCalls+nodes.getCalls+machines.getCalls != 0 {
		t.Fatal("Bridge call happened without an owner session")
	}
}

func descriptorFixture() permissions.Descriptor {
	return permissions.Descriptor{Version: permissions.ContractVersion, ID: permissionID(permissions.TargetBinding{MachineID: "machine-a", NodeID: "node-a", EnrollmentGeneration: 4}), Owner: Owner, PrincipalKind: string(identity.ActorHuman), PrincipalID: "operator-a", Scope: "vrooli-bridge:write", SubjectKind: "node-scopes", SubjectID: "node-a", TargetBound: true, Target: permissions.TargetBinding{MachineID: "machine-a", NodeID: "node-a", EnrollmentGeneration: 4}, Source: "Bridge NodeRegistryService + MachineService", SchemaVersion: "registry/v1", CanRead: true, CanWrite: true, Sensitivity: "operator"}
}

func jsonUnmarshalScopes(want []byte, got []string) error {
	var value scopeSet
	if err := json.Unmarshal(want, &value); err != nil {
		return err
	}
	if len(value.Scopes) != len(got) {
		return fmt.Errorf("scopes=%v want %v", got, value.Scopes)
	}
	for i := range got {
		if got[i] != value.Scopes[i] {
			return fmt.Errorf("scopes=%v want %v", got, value.Scopes)
		}
	}
	return nil
}
