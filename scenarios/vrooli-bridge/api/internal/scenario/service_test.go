package scenario

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeNodes struct{ node TargetNode }

func (f fakeNodes) GetTarget(context.Context, string) (TargetNode, error) { return f.node, nil }

type fakePresence struct{ online, dispatchable bool }

func (f fakePresence) IsOnline(string) bool     { return f.online }
func (f fakePresence) Dispatchable(string) bool { return f.dispatchable }

type fakePusher struct {
	request   Request
	delivered int
}

func (f *fakePusher) Push(_ context.Context, _ string, request Request) (int, error) {
	f.request = request
	return f.delivered, nil
}

func TestServiceCallBindsResponseToNodeAndBoundsIt(t *testing.T) {
	pusher := &fakePusher{delivered: 1}
	broker := NewBroker()
	svc := NewService(fakeNodes{node: TargetNode{ID: "node-1", Scopes: []string{"onboarding:read"}}}, fakePresence{online: true, dispatchable: true}, pusher, broker, WithAdmission(func(request Request, node TargetNode) error {
		if request.Service != "demo.Service" || node.ID != "node-1" {
			t.Fatalf("admission received %#v / %#v", request, node)
		}
		return nil
	}), WithLimits(3, 1))

	resultCh := make(chan Response, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := svc.Call(context.Background(), Request{NodeID: "node-1", Scenario: "demo", Service: "demo.Service", Method: "Get", Body: []byte("req"), MaxResponseBytes: 3})
		resultCh <- result
		errCh <- err
	}()
	deadline := time.After(time.Second)
	for pusher.request.CorrelationID == "" {
		select {
		case <-deadline:
			t.Fatal("request was not pushed")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := broker.Deliver("node-1", Response{CorrelationID: pusher.request.CorrelationID, Body: []byte("12345")}); err != nil {
		t.Fatal(err)
	}
	result := <-resultCh
	if err := <-errCh; !errors.Is(err, nil) {
		t.Fatalf("Call error = %v", err)
	}
	if string(result.Body) != "123" || !result.Truncated {
		t.Fatalf("result = %#v, want bounded response", result)
	}
}

func TestBrokerRejectsResponseFromWrongNode(t *testing.T) {
	broker := NewBroker()
	responses, unregister, err := broker.Register("c1", "node-1")
	if err != nil {
		t.Fatal(err)
	}
	defer unregister()
	if err := broker.Deliver("node-2", Response{CorrelationID: "c1"}); err == nil {
		t.Fatal("wrong node response was accepted")
	}
	select {
	case <-responses:
		t.Fatal("wrong node response reached waiter")
	default:
	}
}
