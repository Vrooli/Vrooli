package inference

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

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
	client := NewGatewayClient(fakeRouting{response: `{"embedding":[0.1,0.2]}`, wantRole: EmbeddingRole, wantInput: clusteringPrefix + "memory", wantTimeout: EmbeddingTimeout})
	got, err := client.Embed(context.Background(), "memory", EmbeddingClustering)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("embedding = %v", got)
	}
}

func TestGatewayClientUsesGenerationTimeoutAndDecodesClassification(t *testing.T) {
	client := NewGatewayClient(fakeRouting{response: `{"response":"taxonomy"}`, wantRole: ClassificationRole, wantInput: classificationPrompt("memory"), wantTimeout: GenerationTimeout, wantMaxOutputTokens: ClassificationMaxOutputTokens})
	got, err := client.Classify(context.Background(), "memory")
	if err != nil {
		t.Fatal(err)
	}
	if got != "taxonomy" {
		t.Fatalf("classification = %q", got)
	}
}

func TestGatewayClientUsesLongerTimeoutForCompactionSummary(t *testing.T) {
	client := NewGatewayClient(fakeRouting{response: `{"response":"summary"}`, wantRole: SummaryRole, wantInput: "cluster", wantTimeout: SummaryTimeout, wantMaxOutputTokens: SummaryMaxOutputTokens})
	got, err := client.Summarize(context.Background(), "cluster")
	require.NoError(t, err)
	require.Equal(t, "summary", got)
}

func TestGeneratedTextPreservesPlainTextForNonResourceFakes(t *testing.T) {
	got, err := generatedText("plain summary")
	if err != nil {
		t.Fatal(err)
	}
	if got != "plain summary" {
		t.Fatalf("generated text = %q", got)
	}
}

func TestClassificationExcerptPreservesHeadAndTailWithinBound(t *testing.T) {
	input := strings.Repeat("a", classificationInputRunes/2+1) + "TAIL" + strings.Repeat("z", classificationInputRunes/2+1)
	got := classificationExcerpt(input)
	require.Contains(t, got, "[... classification excerpt omitted ...]")
	require.True(t, strings.HasPrefix(got, "a"))
	require.True(t, strings.HasSuffix(got, "z"))
	require.NotContains(t, got, "TAIL")
}

func TestEmbeddingExcerptPreservesHeadAndTailWithinBound(t *testing.T) {
	input := strings.Repeat("a", embeddingInputRunes/2+1) + "TAIL" + strings.Repeat("z", embeddingInputRunes/2+1)
	got := embeddingExcerpt(input)
	require.Contains(t, got, "[... embedding excerpt omitted ...]")
	require.True(t, strings.HasPrefix(got, "a"))
	require.True(t, strings.HasSuffix(got, "z"))
	require.NotContains(t, got, "TAIL")
}

func TestEmbeddingInputKeepsProviderPayloadWithinSafeBound(t *testing.T) {
	input := strings.Repeat("a", embeddingInputRunes+1000)
	payload := embeddingInput(EmbeddingDocument, input)

	require.LessOrEqual(t, utf8.RuneCountInString(payload), embeddingInputRunes+utf8.RuneCountInString(documentPrefix)+utf8.RuneCountInString("\n[... embedding excerpt omitted ...]\n"))
}

type fakeRouting struct {
	response            string
	wantRole            string
	wantInput           string
	wantTimeout         time.Duration
	wantMaxOutputTokens int
}

func (f fakeRouting) ExecuteRoute(_ context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	if req.Msg.GetRequest().GetRole() != f.wantRole || req.Msg.GetInputText() != f.wantInput || req.Msg.GetRequest().GetTimeoutMs() != int32(f.wantTimeout/time.Millisecond) || req.Msg.GetRequest().GetMaxOutputTokens() != int32(f.wantMaxOutputTokens) {
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

func (fakeRouting) CancelMediaExecution(context.Context, *connect.Request[routingv1.CancelMediaExecutionRequest]) (*connect.Response[routingv1.CancelMediaExecutionResponse], error) {
	return nil, nil
}

func (fakeRouting) SubmitMedia(context.Context, *connect.Request[routingv1.SubmitMediaRequest]) (*connect.Response[routingv1.SubmitMediaResponse], error) {
	return nil, nil
}

func (fakeRouting) GetMediaExecution(context.Context, *connect.Request[routingv1.GetMediaExecutionRequest]) (*connect.Response[routingv1.GetMediaExecutionResponse], error) {
	return nil, nil
}

func (fakeRouting) WaitMediaExecution(context.Context, *connect.Request[routingv1.WaitMediaExecutionRequest]) (*connect.Response[routingv1.WaitMediaExecutionResponse], error) {
	return nil, nil
}

func (fakeRouting) RetryMediaExecution(context.Context, *connect.Request[routingv1.RetryMediaExecutionRequest]) (*connect.Response[routingv1.RetryMediaExecutionResponse], error) {
	return nil, nil
}
