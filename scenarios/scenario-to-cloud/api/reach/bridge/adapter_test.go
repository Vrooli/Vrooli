package bridge

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	"google.golang.org/protobuf/types/known/timestamppb"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

type fakeClient struct {
	dispatches []nodereach.DispatchRequest
	calls      []nodereach.CallRequest
	waits      []string
	dispatch   nodereach.DispatchResponse
	dispatchEr error
	run        *runsv1.Run
	done       bool
	call       nodereach.CallResponse
	callErr    error
	nodes      []*registryv1.Node
}

type fakeArtifactClient struct {
	requests []ArtifactRequest
	result   ArtifactResult
}

type fakeBindingReader struct {
	snapshot BindingSnapshot
	err      error
	calls    []string
}

type bindingBridgeAPI struct {
	url    string
	client connect.HTTPClient
}

func (a bindingBridgeAPI) ResolveURL(context.Context) (string, error) { return a.url, nil }
func (a bindingBridgeAPI) ConnectTransport(context.Context, string) connect.HTTPClient {
	return a.client
}

type machineBindingHandler struct {
	machinesconnect.UnimplementedMachineServiceHandler
	requestID string
}

func (h *machineBindingHandler) GetMachine(_ context.Context, req *connect.Request[machinesv1.GetMachineRequest]) (*connect.Response[machinesv1.GetMachineResponse], error) {
	h.requestID = req.Msg.GetId()
	return connect.NewResponse(&machinesv1.GetMachineResponse{
		Machine:     &machinesv1.Machine{Id: "machine-9", Lifecycle: "active", EnrollmentGeneration: 9, NodeLineage: []*machinesv1.NodeLineage{{NodeId: "node-9", Current: true, EnrollmentGeneration: 9}}},
		CurrentNode: &machinesv1.CurrentNodeProjection{NodeId: "node-9", EnrollmentGeneration: 9},
	}), nil
}

func TestConnectBindingReaderUsesCurrentMachineLineage(t *testing.T) {
	handler := &machineBindingHandler{}
	path, service := machinesconnect.NewMachineServiceHandler(handler)
	mux := http.NewServeMux()
	mux.Handle(path, service)
	server := httptest.NewServer(mux)
	defer server.Close()
	reader := NewBindingReader(bindingBridgeAPI{url: server.URL, client: server.Client()})
	snapshot, err := reader.ReadBinding(context.Background(), "machine-9")
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Active || snapshot.CurrentNodeID != "node-9" || snapshot.EnrollmentGeneration != 9 || handler.requestID != "machine-9" {
		t.Fatalf("binding snapshot=%+v request machine=%q", snapshot, handler.requestID)
	}
}

func (f *fakeBindingReader) ReadBinding(_ context.Context, machineID string) (BindingSnapshot, error) {
	f.calls = append(f.calls, machineID)
	return f.snapshot, f.err
}

func testAdapter(c Client, artifacts ArtifactClient) *Adapter {
	return &Adapter{Client: c, Artifacts: artifacts, Bindings: &fakeBindingReader{snapshot: BindingSnapshot{Active: true, CurrentNodeID: "n-1", EnrollmentGeneration: 2}}}
}
func testAdapterFor(c Client) *Adapter { return testAdapter(c, nil) }

func (f *fakeArtifactClient) Distribute(_ context.Context, req ArtifactRequest) (ArtifactResult, error) {
	f.requests = append(f.requests, req)
	return f.result, nil
}

func (f *fakeArtifactClient) Get(_ context.Context, _ string) (ArtifactResult, error) {
	return f.result, nil
}

func (f *fakeClient) Dispatch(_ context.Context, req nodereach.DispatchRequest) (nodereach.DispatchResponse, error) {
	f.dispatches = append(f.dispatches, req)
	return f.dispatch, f.dispatchEr
}

func (f *fakeClient) Wait(_ context.Context, runID string, _ time.Duration) (*runsv1.Run, bool, error) {
	f.waits = append(f.waits, runID)
	return f.run, f.done, nil
}

func (f *fakeClient) Call(_ context.Context, req nodereach.CallRequest) (nodereach.CallResponse, error) {
	f.calls = append(f.calls, req)
	return f.call, f.callErr
}

func (f *fakeClient) List(context.Context, time.Duration) ([]*registryv1.Node, error) {
	return f.nodes, nil
}

func target() identity.TargetRef {
	return identity.TargetRef{MachineID: "m-1", NodeID: "n-1", EnrollmentGeneration: 2, Transport: identity.TransportBridge}
}

// [REQ:STC-P0-024] Effectful verbs become durable Bridge runs with argv
// passed as separate values; the result carries the run id so an operation
// can reattach after a lost reply (REACH-05).
func TestEffectfulCommandDispatchesAndAttachesByRunID(t *testing.T) {
	c := &fakeClient{dispatch: nodereach.DispatchResponse{RunID: "run-9"}, run: &runsv1.Run{Id: "run-9", Status: runsv1.RunStatus_RUN_STATUS_PASSED, ExitCode: 0}, done: true}
	a := testAdapterFor(c)
	res, err := a.Exec(context.Background(), target(), reach.Command{Verb: "cloud-target release stage", Args: []string{"--deployment", "dep-1", "--fence", "4"}, Effectful: true, RequiredScope: "vrooli:write"})
	if err != nil {
		t.Fatal(err)
	}
	if res.RunID != "run-9" || res.Transport != identity.TransportBridge {
		t.Fatalf("result = %+v", res)
	}
	d := c.dispatches[0]
	if d.NodeID != "n-1" || d.Scenario != Scenario || d.Verb != "cloud-target release stage" || len(d.Args) != 4 || d.Args[1] != "dep-1" {
		t.Fatalf("dispatch = %+v", d)
	}
	// Reattach without a second dispatch.
	res2, err := a.Attach(context.Background(), target(), "run-9", time.Second)
	if err != nil || res2.RunID != "run-9" || len(c.dispatches) != 1 {
		t.Fatalf("attach res=%+v err=%v dispatches=%d", res2, err, len(c.dispatches))
	}
}

func TestReadCommandUsesRelayAndReturnsOutput(t *testing.T) {
	c := &fakeClient{call: nodereach.CallResponse{CorrelationID: "corr-1", Outcome: relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED, Data: []byte(`{"outcome":"succeeded"}`), ExitCode: 0}}
	a := testAdapterFor(c)
	res, err := a.Exec(context.Background(), target(), reach.Command{Verb: "cloud-target receipt get", Args: []string{"--deployment", "dep-1", "--operation", "op-1", "--step", "release.stage", "--json"}, RequiredScope: "vrooli:read"})
	if err != nil {
		t.Fatal(err)
	}
	if res.CorrelationID != "corr-1" || res.Stdout == "" || len(c.dispatches) != 0 {
		t.Fatalf("result = %+v dispatches=%d", res, len(c.dispatches))
	}
	call := c.calls[0]
	if call.Command != "cloud-target" || call.Args[0] != "receipt" || call.RequiredScope != "vrooli:read" || call.NodeID != "n-1" {
		t.Fatalf("call = %+v", call)
	}
}

func TestDeliverUsesBridgeArtifactsOwner(t *testing.T) {
	artifacts := &fakeArtifactClient{result: ArtifactResult{DistributionID: "dist-1", Delivered: true}}
	a := testAdapter(&fakeClient{}, artifacts)
	receipt, err := a.Deliver(context.Background(), target(), reach.Delivery{Files: []reach.ArtifactFile{{Role: "bundle", LocalPath: "/tmp/bundle.tar.gz", RemotePath: "/opt/app/bundle.tar.gz", SHA256: "abc"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts.requests) != 1 || artifacts.requests[0].NodeID != "n-1" || artifacts.requests[0].SourceRef != "/tmp/bundle.tar.gz" || artifacts.requests[0].DestinationPath != "/opt/app/bundle.tar.gz" {
		t.Fatalf("artifact requests = %+v", artifacts.requests)
	}
	if len(receipt.Files) != 1 || receipt.Files[0].SHA256 != "abc" || receipt.Transport != identity.TransportBridge {
		t.Fatalf("receipt = %+v", receipt)
	}
}

// Typed nodereach kinds map to reach kinds without reading message text;
// a revoked or unknown node never falls back to another transport.
func TestNodereachErrorsAreClassifiedByKind(t *testing.T) {
	cases := map[nodereach.ErrorKind]reach.Kind{
		nodereach.ErrNodeNotFound:      reach.KindEnrollmentRevoked,
		nodereach.ErrNodeUnavailable:   reach.KindTargetOffline,
		nodereach.ErrMissingScope:      reach.KindScopeMissing,
		nodereach.ErrHandshakeRejected: reach.KindProtocolUnsupported,
		nodereach.ErrBridgeUnavailable: reach.KindUnavailable,
		nodereach.ErrTransport:         reach.KindTransport,
	}
	for kind, want := range cases {
		c := &fakeClient{callErr: &nodereach.Error{Kind: kind, Node: "n-1", Err: errors.New("x")}}
		_, err := testAdapterFor(c).Exec(context.Background(), target(), reach.Command{Verb: "cloud-target receipt get", Args: []string{"--json"}})
		if !reach.IsKind(err, want) {
			t.Fatalf("%s: expected %s, got %v", kind, want, err)
		}
	}
	if _, err := testAdapterFor(&fakeClient{}).Exec(context.Background(), identity.TargetRef{Transport: identity.TransportBridge}, reach.Command{Verb: "cloud-target receipt get", Args: []string{"--json"}}); !reach.IsKind(err, reach.KindEnrollmentRevoked) {
		t.Fatalf("missing node id must be enrollment_revoked, got %v", err)
	}
}

func TestStaleOrUnverifiableMachineBindingRefusesBeforeEffects(t *testing.T) {
	tests := []struct {
		name    string
		binding BindingSnapshot
		reader  BindingReader
	}{
		{name: "superseded node", binding: BindingSnapshot{Active: true, CurrentNodeID: "replacement-node", EnrollmentGeneration: 2}},
		{name: "stale enrollment generation", binding: BindingSnapshot{Active: true, CurrentNodeID: "n-1", EnrollmentGeneration: 1}},
		{name: "inactive machine", binding: BindingSnapshot{Active: false, CurrentNodeID: "n-1", EnrollmentGeneration: 2}},
		{name: "no authoritative reader", reader: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeClient{dispatch: nodereach.DispatchResponse{RunID: "run-1"}}
			reader := test.reader
			if reader == nil && test.name != "no authoritative reader" {
				reader = &fakeBindingReader{snapshot: test.binding}
			}
			a := &Adapter{Client: client, Bindings: reader}
			_, err := a.Exec(context.Background(), target(), reach.Command{Verb: "cloud-target release activate", Effectful: true, RequiredScope: "vrooli:write"})
			if err == nil {
				t.Fatal("expected stale binding refusal")
			}
			if len(client.dispatches) != 0 {
				t.Fatalf("stale binding dispatched effects: %+v", client.dispatches)
			}
			if _, err := a.Attach(context.Background(), target(), "run-1", time.Second); err == nil || len(client.waits) != 0 {
				t.Fatalf("stale binding attached run: waits=%v err=%v", client.waits, err)
			}
		})
	}
}

// Negotiation reports offline, protocol-incompatible, revoked and missing
// scope as distinct states (REACH-04).
func TestNegotiateDistinguishesNodeStates(t *testing.T) {
	online := &registryv1.Node{Id: "n-1", Online: true, Dispatchable: true, ProtocolCompatible: true, Scopes: []string{"vrooli:read"}, Revision: "abc"}
	caps, err := testAdapterFor(&fakeClient{nodes: []*registryv1.Node{online}}).Negotiate(context.Background(), target())
	if err != nil || !caps.Online || caps.ProtocolVersion != "abc" {
		t.Fatalf("caps=%+v err=%v", caps, err)
	}
	if err := reach.RequireScope(caps.Scopes, "vrooli:write"); !reach.IsKind(err, reach.KindScopeMissing) {
		t.Fatalf("expected scope_missing, got %v", err)
	}
	offline := &registryv1.Node{Id: "n-1", Online: false}
	if _, err := testAdapterFor(&fakeClient{nodes: []*registryv1.Node{offline}}).Negotiate(context.Background(), target()); !reach.IsKind(err, reach.KindTargetOffline) {
		t.Fatalf("expected target_offline, got %v", err)
	}
	incompatible := &registryv1.Node{Id: "n-1", Online: true, ProtocolCompatible: false}
	if _, err := testAdapterFor(&fakeClient{nodes: []*registryv1.Node{incompatible}}).Negotiate(context.Background(), target()); !reach.IsKind(err, reach.KindProtocolUnsupported) {
		t.Fatalf("expected reach_protocol_unsupported, got %v", err)
	}
	revoked := &registryv1.Node{Id: "n-1", Online: true, RevokedAt: timestamppb.Now()}
	if _, err := testAdapterFor(&fakeClient{nodes: []*registryv1.Node{revoked}}).Negotiate(context.Background(), target()); !reach.IsKind(err, reach.KindEnrollmentRevoked) {
		t.Fatalf("expected enrollment_revoked, got %v", err)
	}
	if _, err := testAdapterFor(&fakeClient{}).Negotiate(context.Background(), target()); !reach.IsKind(err, reach.KindEnrollmentRevoked) {
		t.Fatalf("unknown node must be enrollment_revoked, got %v", err)
	}
}
