package relay_test

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/relay"
)

type fakeNodes struct{ node dispatch.TargetNode }

func (f fakeNodes) GetTarget(context.Context, string) (dispatch.TargetNode, error) {
	return f.node, nil
}

type fakePresence struct{ online, dispatchable bool }

func (f fakePresence) IsOnline(string) bool     { return f.online }
func (f fakePresence) Dispatchable(string) bool { return f.dispatchable }

type fakeAudit struct {
	mu      sync.Mutex
	records []audit.Record
}

func (f *fakeAudit) Append(_ context.Context, record audit.Record) (audit.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records = append(f.records, record)
	return record, nil
}

type fakePusher struct {
	push   func(relay.Request)
	cancel func(string)
}

func (f fakePusher) Push(_ context.Context, _ string, request relay.Request) (int, error) {
	if f.push != nil {
		f.push(request)
	}
	return 1, nil
}

func (f fakePusher) Cancel(_ context.Context, _ string, correlationID, _ string) (int, error) {
	if f.cancel != nil {
		f.cancel(correlationID)
	}
	return 1, nil
}

func TestAdmissionParityWithTypedDispatch(t *testing.T) {
	manifest := []string{"scenario test"}
	node := dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"scenario test"}}
	cases := []struct {
		name    string
		request relay.Request
		node    dispatch.TargetNode
	}{
		{name: "accepted", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test"}, node: node},
		{name: "unsafe token", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test", Args: []string{"x;bad"}}, node: node},
		{name: "out of scope", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test"}, node: dispatch.TargetNode{ID: "n1", Kind: "agent"}},
		{name: "unknown command", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario status"}, node: node},
		{name: "revoked", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test"}, node: dispatch.TargetNode{ID: "n1", Kind: "agent", Revoked: true, Scopes: node.Scopes}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			typedErr := dispatch.Admit(dispatch.Job{NodeID: tc.request.NodeID, Scenario: tc.request.Scenario, Verb: tc.request.Command, Args: tc.request.Args}, tc.node, manifest)
			relayErr := relay.Admit(tc.request, tc.node, manifest)
			if typedErr == nil || relayErr == nil {
				require.Equal(t, typedErr == nil, relayErr == nil)
				return
			}
			require.Equal(t, reflect.TypeOf(typedErr), reflect.TypeOf(relayErr))
			require.Equal(t, typedErr.Error(), relayErr.Error())
		})
	}
}

func TestCallStreamsAndAuditsNodeCommandOutcome(t *testing.T) {
	broker := relay.NewBroker()
	auditSink := &fakeAudit{}
	pusher := fakePusher{push: func(request relay.Request) {
		go func() {
			_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindAccepted})
			_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindData, Data: []byte("ok\n")})
			_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindCompleted, ExitCode: 0})
		}()
	}}
	svc := relay.NewService(fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"scenario test"}}}, fakePresence{online: true, dispatchable: true}, auditSink, pusher, broker, relay.WithManifest([]string{"scenario test"}))

	response, err := svc.Call(context.Background(), relay.Request{Actor: "owner", NodeID: "n1", Scenario: "demo", Command: "scenario test"})
	require.NoError(t, err)
	require.Equal(t, relay.KindCompleted, response.Kind)
	require.Equal(t, []byte("ok\n"), response.Data)
	require.Len(t, auditSink.records, 2)
	require.Equal(t, "n1", auditSink.records[0].NodeID)
	require.Equal(t, "scenario test", auditSink.records[0].Verb)
	require.Equal(t, audit.OutcomeCompleted, auditSink.records[len(auditSink.records)-1].Outcome)
}

func TestCallNamesAndCancelsResponseLimit(t *testing.T) {
	broker := relay.NewBroker()
	var cancelled string
	pusher := fakePusher{
		push: func(request relay.Request) {
			go func() {
				_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindData, Data: []byte("too-large")})
			}()
		},
		cancel: func(correlationID string) { cancelled = correlationID },
	}
	svc := relay.NewService(fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"scenario test"}}}, fakePresence{online: true, dispatchable: true}, nil, pusher, broker, relay.WithManifest([]string{"scenario test"}))

	response, err := svc.Call(context.Background(), relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test", MaxResponseBytes: 3})
	var limit relay.ErrResponseLimit
	require.ErrorAs(t, err, &limit)
	require.Equal(t, relay.KindFailed, response.Kind)
	require.Equal(t, relay.ResponseLimitReason, response.Reason[:len(relay.ResponseLimitReason)])
	require.NotEmpty(t, cancelled)
}
