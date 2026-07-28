package inference

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
)

func TestEmbedderUsesDistinctGatewayTasks(t *testing.T) {
	fake := &fakeInference{embedOut: []float64{0.1}}
	embedder := Embedder{Client: fake}
	if _, err := embedder.EmbedQuery(context.Background(), "query"); err != nil {
		t.Fatal(err)
	}
	if _, err := embedder.EmbedDocument(context.Background(), "document"); err != nil {
		t.Fatal(err)
	}
	if len(fake.embedTasks) != 2 || fake.embedTasks[0] != EmbeddingQuery || fake.embedTasks[1] != EmbeddingDocument {
		t.Fatalf("tasks = %v", fake.embedTasks)
	}
}

type fakeInference struct {
	embedOut   []float64
	embedTasks []EmbeddingTask
}

func (f *fakeInference) Embed(_ context.Context, _ string, task EmbeddingTask) ([]float64, error) {
	f.embedTasks = append(f.embedTasks, task)
	return f.embedOut, nil
}
func (f *fakeInference) Classify(context.Context, string) (string, error)  { return "", nil }
func (f *fakeInference) Summarize(context.Context, string) (string, error) { return "", nil }

func TestGatewayClientEmbedPrefixesClusteringText(t *testing.T) {
	client := NewGatewayClient(fakeRouting{response: `{"embedding":[0.1,0.2]}`})
	got, err := client.Embed(context.Background(), "memory", EmbeddingClustering)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("embedding = %v", got)
	}
}

type fakeRouting struct{ response string }

func (f fakeRouting) ExecuteRoute(_ context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	if req.Msg.GetRequest().GetRole() != EmbeddingRole || req.Msg.GetInputText() != clusteringPrefix+"memory" {
		panic("wrong gateway request")
	}
	return connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true, OutputText: f.response}), nil
}

func (fakeRouting) PreviewRoute(context.Context, *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error) {
	return nil, nil
}

func (fakeRouting) ListRouteEvidence(context.Context, *connect.Request[routingv1.ListRouteEvidenceRequest]) (*connect.Response[routingv1.ListRouteEvidenceResponse], error) {
	return nil, nil
}

func (fakeRouting) GetRouteEvidence(context.Context, *connect.Request[routingv1.GetRouteEvidenceRequest]) (*connect.Response[routingv1.GetRouteEvidenceResponse], error) {
	return nil, nil
}

func (fakeRouting) ListProviderHealth(context.Context, *connect.Request[routingv1.ListProviderHealthRequest]) (*connect.Response[routingv1.ListProviderHealthResponse], error) {
	return nil, nil
}
