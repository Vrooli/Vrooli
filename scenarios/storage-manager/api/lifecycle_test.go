package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"storage-manager/internal/orchestrator"
	"storage-manager/internal/workgate"
)

func TestOwnerLifecycleTransportDrainsBackgroundAndResumes(t *testing.T) {
	gate := &workgate.Gate{}
	service := orchestrator.NewService(nil, orchestrator.NewMemoryStore(), nil)
	entered, release := make(chan struct{}), make(chan struct{})
	finished := make(chan struct{})
	go func() { defer close(finished); _ = gate.Run(func() error { close(entered); <-release; return nil }) }()
	<-entered
	p, err := lifecycle.NewProvider("storage-manager", "fixture", ownerLifecycleHooks(gate, service))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(p.Mount(http.NotFoundHandler()))
	defer server.Close()
	client, err := lifecycle.NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := client.Prepare(context.Background(), &commonv1.LifecyclePrepareRequest{OperationId: "test", Reason: "restart"})
	if err != nil {
		t.Fatal(err)
	}
	fence := prepared.GetStanding()
	if fence.GetRemaining() != 1 || !fence.GetQuiescencePending() {
		t.Fatalf("untracked background: %v", fence)
	}
	req := &commonv1.LifecycleDrainRequest{OperationId: fence.GetOperationId(), FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision(), QuiesceComponents: true}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	_, err = client.Drain(ctx, req)
	cancel()
	if err == nil {
		t.Fatal("drain succeeded while work was blocked")
	}
	close(release)
	<-finished
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	drained, err := client.Drain(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if !drained.GetStanding().GetDrained() {
		t.Fatalf("not drained: %v", drained)
	}
	ran := false
	_ = gate.Run(func() error { ran = true; return nil })
	if ran {
		t.Fatal("admitted during fence")
	}
	_, err = client.Resume(ctx, &commonv1.LifecycleResumeRequest{OperationId: fence.GetOperationId(), FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision()})
	if err != nil {
		t.Fatal(err)
	}
	_ = gate.Run(func() error { ran = true; return nil })
	if !ran {
		t.Fatal("background admission not resumed")
	}
}
