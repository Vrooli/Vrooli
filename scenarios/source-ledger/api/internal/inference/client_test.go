package inference

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"source-ledger/internal/policy"

	"github.com/stretchr/testify/require"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestEmbedderUsesDistinctGatewayTasks(t *testing.T) { // [REQ:VMEM-P1-005]
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

func TestGatewayClientEmbedReturnsVectorFromSingleCall(t *testing.T) {
	routing := &outcomeRouting{result: connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true, OutputText: `{"embedding":[0.3,0.4]}`})}
	client := NewGatewayClient(routing)
	got, err := client.Embed(context.Background(), "memory", EmbeddingClustering)
	require.NoError(t, err)
	require.Equal(t, []float64{0.3, 0.4}, got)
	require.Equal(t, 1, routing.calls)
}

func TestGatewayClientEmbedDoesNotRepeatUnknownOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		result     *connect.Response[routingv1.ExecuteRouteResponse]
		err        error
		wantError  string
		wantCause  error
		wantSyntax bool
	}{
		{name: "timeout", err: context.DeadlineExceeded, wantError: "context deadline exceeded", wantCause: context.DeadlineExceeded},
		{name: "cancellation", err: context.Canceled, wantError: "context canceled", wantCause: context.Canceled},
		{name: "nil response", wantError: "no route response"},
		{name: "nil message", result: &connect.Response[routingv1.ExecuteRouteResponse]{}, wantError: "no route response"},
		{name: "rejected response", result: connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: false}), wantError: "rejected embedding.default route"},
		{name: "blocked route", result: connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true, Evidence: &routingv1.RouteEvidence{Status: "blocked", RejectionReason: "provider_breaker_open"}}), wantError: "embedding.default route blocked: provider_breaker_open"},
		{name: "failed route", result: connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true, Evidence: &routingv1.RouteEvidence{Status: "failed", FailureReasons: []string{"provider.execution_failed"}}}), wantError: "route failed (provider.execution_failed)"},
		{name: "malformed JSON", result: connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true, OutputText: "{"}), wantError: "decode ai-gateway embedding response", wantSyntax: true},
		{name: "empty vector", result: connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true, OutputText: `{"embedding":[]}`}), wantError: "contained no vector"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routing := &outcomeRouting{result: tt.result, err: tt.err}
			got, err := NewGatewayClient(routing).Embed(context.Background(), "memory", EmbeddingClustering)
			require.Nil(t, got)
			require.ErrorContains(t, err, tt.wantError)
			if tt.wantCause != nil {
				require.ErrorIs(t, err, tt.wantCause)
			}
			if tt.wantSyntax {
				var syntaxErr *json.SyntaxError
				require.ErrorAs(t, err, &syntaxErr)
			}
			require.Equal(t, 1, routing.calls, "an unusable result does not prove the provider did no work")
		})
	}
}

func TestGatewayClientEmbedRejectsMissingRoutingDependency(t *testing.T) {
	for name, client := range map[string]*GatewayClient{"nil receiver": nil, "nil routing": NewGatewayClient(nil)} {
		t.Run(name, func(t *testing.T) {
			got, err := client.Embed(context.Background(), "memory", EmbeddingClustering)
			require.Nil(t, got)
			require.ErrorContains(t, err, "routing client is not configured")
		})
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

func TestGatewayClientCarriesEntryKindIntoClassificationContext(t *testing.T) {
	memory := classificationMemory("memory", "work-record")
	client := NewGatewayClient(fakeRouting{response: `{"response":"episode"}`, wantRole: ClassificationRole, wantInput: classificationPrompt(memory), wantTimeout: GenerationTimeout, wantMaxOutputTokens: ClassificationMaxOutputTokens})
	got, err := client.ClassifyEntry(context.Background(), "memory", "work-record")
	require.NoError(t, err)
	require.Equal(t, "episode", got)
}

func TestGatewayClientBuildsClassificationVocabularyFromProvider(t *testing.T) {
	client := NewGatewayClient(fakeRouting{response: `{"response":"future-facet"}`, wantRole: ClassificationRole, wantInput: classificationPrompt("memory", VocabularyEntry{ID: "future-facet", Label: "Future facet", Guidance: "A future category."}), wantTimeout: GenerationTimeout, wantMaxOutputTokens: ClassificationMaxOutputTokens}, func(context.Context) ([]VocabularyEntry, error) {
		return []VocabularyEntry{{ID: "future-facet", Label: "Future facet", Guidance: "A future category."}}, nil
	})
	got, err := client.Classify(context.Background(), "memory")
	require.NoError(t, err)
	require.Equal(t, "future-facet", got)
}

func TestClassificationPromptCarriesPolicyOwnedGuidance(t *testing.T) { // [REQ:VMEM-P0-005] [REQ:VMEM-P1-005]
	got := classificationPrompt("memory", VocabularyEntry{ID: "gotcha", Label: "Gotcha", Guidance: "A recurring failure mode or trap.", Examples: []string{"A recurring provider timeout trap."}})
	require.Contains(t, got, "gotcha")
	require.Contains(t, got, "A recurring failure mode or trap")
	require.Contains(t, got, "A recurring provider timeout trap")
	require.Contains(t, got, "return only the exact ID")
}

func TestClassificationPromptBoundsFewShotExamples(t *testing.T) {
	example := strings.Repeat("x", classificationExampleRunes+100)
	got := classificationPrompt("memory", VocabularyEntry{ID: "episode", Label: "Episode", Examples: []string{example, "second", "third"}})
	require.Contains(t, got, "[... classification excerpt omitted ...]")
	require.NotContains(t, got, "second")
	require.NotContains(t, got, "third")
}

func TestGatewayClientUsesLongerTimeoutForCompactionSummary(t *testing.T) {
	client := NewGatewayClient(fakeRouting{response: `{"response":"summary"}`, wantRole: SummaryRole, wantInput: "cluster", wantTimeout: SummaryTimeout, wantMaxOutputTokens: policy.DefaultSummaryMaxOutputTokens})
	got, err := client.Summarize(context.Background(), "cluster")
	require.NoError(t, err)
	require.Equal(t, "summary", got)
}

func TestSummaryCeilingIsConfigurableInOnePlace(t *testing.T) {
	client := NewGatewayClient(fakeRouting{response: `{"response":"summary"}`, wantRole: SummaryRole, wantInput: "cluster", wantTimeout: SummaryTimeout, wantMaxOutputTokens: 900}).WithSummaryMaxOutputTokens(900)
	_, err := client.Summarize(context.Background(), "cluster")
	require.NoError(t, err)
	// A non-positive configured value keeps the default instead of disabling the bound.
	client = NewGatewayClient(fakeRouting{response: `{"response":"summary"}`, wantRole: SummaryRole, wantInput: "cluster", wantTimeout: SummaryTimeout, wantMaxOutputTokens: policy.DefaultSummaryMaxOutputTokens}).WithSummaryMaxOutputTokens(0)
	_, err = client.Summarize(context.Background(), "cluster")
	require.NoError(t, err)
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

type outcomeRouting struct {
	fakeRouting
	result *connect.Response[routingv1.ExecuteRouteResponse]
	err    error
	calls  int
}

func (r *outcomeRouting) ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	r.calls++
	return r.result, r.err
}

func (f fakeRouting) ExecuteRoute(_ context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	request := req.Msg.GetRequest()
	if request.GetRole() != f.wantRole || req.Msg.GetInputText() != f.wantInput ||
		request.GetTimeoutMs() != int32(f.wantTimeout/time.Millisecond) ||
		request.GetMaxOutputTokens() != int32(f.wantMaxOutputTokens) ||
		request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL ||
		request.GetScenario() != "source-ledger" ||
		request.GetPrivacyProvenance() != privacyProvenance {
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

// Background indexing steps aside for foreground inference and never makes
// foreground work wait on itself.
func TestEmbedWhenIdleYieldsToForegroundCalls(t *testing.T) {
	routing := &outcomeRouting{result: connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true, OutputText: `{"embedding":[0.5]}`})}
	client := NewGatewayClient(routing)

	_, err := client.Embed(context.Background(), "recall query", EmbeddingQuery)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err = client.EmbedWhenIdle(ctx, "research chunk")
	require.ErrorIs(t, err, context.DeadlineExceeded, "a background embed waits out the quiet gap")
	require.Equal(t, 1, routing.calls)

	client.lastDone.Store(time.Now().Add(-idleGap).UnixNano())
	for range 2 {
		vec, err := client.EmbedWhenIdle(context.Background(), "research chunk")
		require.NoError(t, err)
		require.Equal(t, []float64{0.5}, vec)
	}
	require.Equal(t, 3, routing.calls, "background calls do not count as foreground activity")
}

type fakeBatchEmbed struct {
	inferenceconnect.InferenceServiceClient
	texts    [][]string
	requests []*inferencev1.EmbedRequest
}

func (f *fakeBatchEmbed) Embed(_ context.Context, req *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	f.texts = append(f.texts, req.Msg.GetTexts())
	f.requests = append(f.requests, req.Msg)
	resp := &inferencev1.EmbedResponse{}
	for range req.Msg.GetTexts() {
		resp.Vectors = append(resp.Vectors, &inferencev1.EmbeddingVector{Values: []float64{1}})
	}
	return connect.NewResponse(resp), nil
}

type blockingBatchCall struct {
	task    EmbeddingTask
	release chan struct{}
}

type blockingBatchEmbed struct {
	inferenceconnect.InferenceServiceClient
	calls  chan *blockingBatchCall
	active atomic.Int32
	max    atomic.Int32
}

func newBlockingBatchEmbed() *blockingBatchEmbed {
	return &blockingBatchEmbed{calls: make(chan *blockingBatchCall, 16)}
}

func (f *blockingBatchEmbed) Embed(ctx context.Context, req *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	task := EmbeddingDocument
	if len(req.Msg.GetTexts()) > 0 && strings.HasPrefix(req.Msg.GetTexts()[0], queryPrefix) {
		task = EmbeddingQuery
	}
	active := f.active.Add(1)
	for prior := f.max.Load(); active > prior; prior = f.max.Load() {
		if f.max.CompareAndSwap(prior, active) {
			break
		}
	}
	defer f.active.Add(-1)
	call := &blockingBatchCall{task: task, release: make(chan struct{})}
	f.calls <- call
	select {
	case <-call.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	resp := &inferencev1.EmbedResponse{}
	for range req.Msg.GetTexts() {
		resp.Vectors = append(resp.Vectors, &inferencev1.EmbeddingVector{Values: []float64{1}})
	}
	return connect.NewResponse(resp), nil
}

func nextBatchCall(t *testing.T, calls <-chan *blockingBatchCall) *blockingBatchCall {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for embedding batch")
		return nil
	}
}

func requireNoBatchCall(t *testing.T, calls <-chan *blockingBatchCall, window time.Duration) {
	t.Helper()
	select {
	case call := <-calls:
		t.Fatalf("unexpected overlapping batch call: task=%s", call.task)
	case <-time.After(window):
	}
}

// The file corpus embeds many chunks per gateway call, with task prefixes,
// and a background batch still steps aside for foreground work.
func TestEmbedBatchSendsOneCallAndYieldsToForeground(t *testing.T) {
	batch := &fakeBatchEmbed{}
	client := NewGatewayClient(&outcomeRouting{}).WithBatchEmbedding(batch)

	vecs, err := client.EmbedBatch(context.Background(), EmbeddingQuery, []string{"q"}, false)
	require.NoError(t, err)
	require.Len(t, vecs, 1)
	require.Equal(t, []string{queryPrefix + "q"}, batch.texts[0])
	// The gateway refuses a typed embedding without these three fields.
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, batch.requests[0].GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, batch.requests[0].GetPrivacyClass())
	require.Equal(t, privacyProvenance, batch.requests[0].GetPrivacyProvenance())

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err = client.EmbedBatch(ctx, EmbeddingDocument, []string{"a", "b"}, true)
	require.ErrorIs(t, err, context.DeadlineExceeded, "the foreground query just finished")

	client.lastDone.Store(time.Now().Add(-idleGap).UnixNano())
	vecs, err = client.EmbedBatch(context.Background(), EmbeddingDocument, []string{"a", "b"}, true)
	require.NoError(t, err)
	require.Len(t, vecs, 2)
	require.Equal(t, []string{documentPrefix + "a", documentPrefix + "b"}, batch.texts[1])
}

func TestBackgroundBatchCallsAreSerialized(t *testing.T) {
	batch := newBlockingBatchEmbed()
	client := NewGatewayClient(&outcomeRouting{}).WithBatchEmbedding(batch)
	client.lastDone.Store(time.Now().Add(-idleGap).UnixNano())

	const workers = 4
	ready := make(chan struct{}, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ready <- struct{}{}
			<-start
			_, _ = client.EmbedBatch(context.Background(), EmbeddingDocument, []string{"doc"}, true)
		}()
	}
	for range workers {
		<-ready
	}
	close(start)

	for range workers {
		call := nextBatchCall(t, batch.calls)
		require.Equal(t, EmbeddingDocument, call.task)
		requireNoBatchCall(t, batch.calls, 50*time.Millisecond)
		close(call.release)
	}
	wg.Wait()
	require.EqualValues(t, 1, batch.max.Load(), "parallel workers may queue, but only one background request may reach AI Gateway")
}

func TestForegroundBatchProceedsAlongsideAtMostOneBackgroundBatch(t *testing.T) {
	batch := newBlockingBatchEmbed()
	client := NewGatewayClient(&outcomeRouting{}).WithBatchEmbedding(batch)
	client.lastDone.Store(time.Now().Add(-idleGap).UnixNano())

	firstDone := make(chan error, 1)
	secondCtx, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	secondDone := make(chan error, 1)
	go func() {
		_, err := client.EmbedBatch(context.Background(), EmbeddingDocument, []string{"first"}, true)
		firstDone <- err
	}()
	first := nextBatchCall(t, batch.calls)
	go func() {
		_, err := client.EmbedBatch(secondCtx, EmbeddingDocument, []string{"second"}, true)
		secondDone <- err
	}()
	requireNoBatchCall(t, batch.calls, 50*time.Millisecond)

	foregroundDone := make(chan error, 1)
	go func() {
		_, err := client.EmbedBatch(context.Background(), EmbeddingQuery, []string{"query"}, false)
		foregroundDone <- err
	}()
	foreground := nextBatchCall(t, batch.calls)
	require.Equal(t, EmbeddingQuery, foreground.task)
	close(foreground.release)
	require.NoError(t, <-foregroundDone, "foreground work must not queue behind the active background batch")

	close(first.release)
	require.NoError(t, <-firstDone)
	requireNoBatchCall(t, batch.calls, 300*time.Millisecond)
	cancelSecond()
	require.ErrorIs(t, <-secondDone, context.Canceled)
	require.EqualValues(t, 2, batch.max.Load(), "one foreground and at most one background request may overlap")
}
