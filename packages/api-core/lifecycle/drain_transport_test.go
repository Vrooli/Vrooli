package lifecycle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// A maintenance wait follows its operation/request context. The ordinary API
// write timeout must not cut off the receipt after ingress is already closed.
func TestDrainSurvivesOrdinaryServerWriteDeadline(t *testing.T) {
	p, err := NewProvider("fixture", "instance", Hooks{Quiesce: func(ctx context.Context) error {
		timer := time.NewTimer(150 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, Resume: func(context.Context) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(p.Mount(http.NotFoundHandler()))
	server.Config.WriteTimeout = 50 * time.Millisecond
	server.Start()
	defer server.Close()
	client, err := NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	prepared, err := client.Prepare(ctx, &commonv1.LifecyclePrepareRequest{OperationId: "long-drain", Reason: "restart"})
	if err != nil {
		t.Fatal(err)
	}
	fence := prepared.GetStanding()
	drained, err := client.Drain(ctx, &commonv1.LifecycleDrainRequest{OperationId: fence.GetOperationId(), FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision(), QuiesceComponents: true})
	if err != nil {
		t.Fatalf("maintenance receipt lost at ordinary write deadline: %v", err)
	}
	if !drained.GetStanding().GetDrained() || !drained.GetStanding().GetComponentStopReady() {
		t.Fatalf("quiescence not proved: %v", drained)
	}
}

func TestDrainWriteDeadlineExceptionPreservesRequestCancellation(t *testing.T) {
	entered, stopped := make(chan struct{}), make(chan struct{})
	p, err := NewProvider("fixture", "instance", Hooks{Quiesce: func(ctx context.Context) error { close(entered); <-ctx.Done(); close(stopped); return ctx.Err() }, Resume: func(context.Context) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(p.Mount(http.NotFoundHandler()))
	defer server.Close()
	client, err := NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := client.Prepare(context.Background(), &commonv1.LifecyclePrepareRequest{OperationId: "cancel-drain", Reason: "restart"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fence := prepared.GetStanding()
	done := make(chan error, 1)
	go func() {
		_, err := client.Drain(ctx, &commonv1.LifecycleDrainRequest{OperationId: fence.GetOperationId(), FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision(), QuiesceComponents: true})
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("drain never entered quiescence")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled drain reported success")
		}
	case <-time.After(time.Second):
		t.Fatal("drain client ignored cancellation")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("quiescence ignored request cancellation")
	}
}
