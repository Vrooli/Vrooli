package analysisoperations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations/analysis_operations_v1connect"
)

func TestHTTPWaitOutlivesWriteTimeoutAndReplaysReceipt(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	m := startManager(t, testRepository(t), testRunner{names: []string{"alpha"}, analyze: func(ctx context.Context, _ string) (ScenarioOutcome, error) {
		close(entered)
		select {
		case <-release:
			return ScenarioOutcome{}, nil
		case <-ctx.Done():
			return ScenarioOutcome{}, ctx.Err()
		}
	}})
	path, handler := NewHTTPHandler(m)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewUnstartedServer(ClearWaitDeadline(mux))
	server.Config.WriteTimeout = 30 * time.Millisecond
	server.Start()
	defer server.Close()
	client := rpc.NewAnalysisOperationsServiceClient(server.Client(), server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started, err := client.StartOperation(ctx, connect.NewRequest(&pb.StartOperationRequest{ClientOperationId: "http-replay"}))
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	timer := time.AfterFunc(150*time.Millisecond, func() { close(release) })
	defer timer.Stop()
	finished, err := client.WaitOperation(ctx, connect.NewRequest(&pb.WaitOperationRequest{OperationId: started.Msg.GetOperation().GetId()}))
	if err != nil {
		t.Fatalf("wait response lost to ordinary write deadline: %v", err)
	}
	if finished.Msg.GetOperation().GetState() != pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_SUCCEEDED {
		t.Fatalf("terminal: %+v", finished.Msg)
	}
	replayed, err := client.StartOperation(ctx, connect.NewRequest(&pb.StartOperationRequest{ClientOperationId: "http-replay"}))
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Msg.GetReused() || replayed.Msg.GetOperation().GetId() != started.Msg.GetOperation().GetId() {
		t.Fatalf("replay: %+v", replayed.Msg)
	}
	_, err = client.GetOperation(ctx, connect.NewRequest(&pb.GetOperationRequest{OperationId: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing receipt: %v", err)
	}
}

func TestTestModeAdmissionCannotEscapeToPrimaryAnalyzer(t *testing.T) {
	m := startManager(t, testRepository(t), testRunner{names: []string{"alpha"}, analyze: func(context.Context, string) (ScenarioOutcome, error) {
		t.Error("test mode escaped to analyzer")
		return ScenarioOutcome{}, nil
	}})
	h := &ConnectHandler{manager: m}
	_, err := h.StartOperation(database.WithTestMode(context.Background()), connect.NewRequest(&pb.StartOperationRequest{ClientOperationId: "test-mode"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("test-mode admission: %v", err)
	}
}
