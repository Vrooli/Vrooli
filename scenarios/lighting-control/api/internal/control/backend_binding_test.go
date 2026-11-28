package control

import (
	"context"
	"errors"
	"testing"
	"time"
)

type blockingBackend struct {
	entered chan struct{}
	release chan struct{}
}

func (b blockingBackend) ProtocolVersion(context.Context) (uint32, error) { return 6, nil }
func (b blockingBackend) Discover(context.Context) ([]SDKController, error) {
	close(b.entered)
	<-b.release
	return nil, ErrNoDevice
}
func (b blockingBackend) SendZoneColors(context.Context, uint32, uint32, []RGB) (SDKCommandResult, error) {
	return SDKCommandResult{}, nil
}

func TestBackendBindingIsOptInLoopbackOnlyAndRequiresIncarnation(t *testing.T) {
	binding := NewBackendBinding()
	if _, err := binding.ProtocolVersion(context.Background()); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("unbound status probe error = %v", err)
	}
	if err := binding.Bind(context.Background(), "192.0.2.8:6742", "generation-1"); err == nil {
		t.Fatal("remote SDK endpoint was accepted")
	}
	if err := binding.Bind(context.Background(), "127.0.0.1:6742", ""); err == nil {
		t.Fatal("missing backend incarnation was accepted")
	}
	if binding.Incarnation() != "" {
		t.Fatalf("failed bind retained incarnation %q", binding.Incarnation())
	}
	if err := binding.Quiesce(context.Background()); err != nil {
		t.Fatalf("quiesce without binding: %v", err)
	}
}

func TestBackendBindingQuiesceRevokesNewWorkAndDrainsActiveOperation(t *testing.T) {
	binding := NewBackendBinding()
	fake := blockingBackend{entered: make(chan struct{}), release: make(chan struct{})}
	binding.backend = fake
	binding.incarnation = "resource-pid-42:birth-7"
	done := make(chan error, 1)
	go func() {
		_, err := binding.Discover(context.Background())
		done <- err
	}()
	<-fake.entered
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := binding.Quiesce(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("quiesce before active operation drains = %v", err)
	}
	if binding.Incarnation() != "" {
		t.Fatal("quiesce left the old backend incarnation callable")
	}
	if _, err := binding.ProtocolVersion(context.Background()); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("new operation after quiesce error = %v", err)
	}
	close(fake.release)
	if err := <-done; !errors.Is(err, ErrNoDevice) {
		t.Fatalf("active operation result = %v", err)
	}
	if err := binding.Quiesce(context.Background()); err != nil {
		t.Fatalf("quiesce after drain: %v", err)
	}
}
