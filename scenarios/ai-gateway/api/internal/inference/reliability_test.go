package inference

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"

	"github.com/stretchr/testify/require"
)

// repairingRepository returns an invalid value once, then a valid one, and
// records every request it saw so the repair path can be inspected.
type repairingRepository struct {
	mu       sync.Mutex
	requests []ProviderRequest
	values   []string
}

func (r *repairingRepository) Run(_ context.Context, request ProviderRequest) (ProviderResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, request)
	value := r.values[len(r.values)-1]
	if len(r.requests) <= len(r.values) {
		value = r.values[len(r.requests)-1]
	}
	return ProviderResult{ValueJSON: value, Provider: "fake", Model: "test", InputTokens: 5, OutputTokens: 2}, nil
}

func (r *repairingRepository) snapshot() []ProviderRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ProviderRequest(nil), r.requests...)
}

func TestServiceReAsksOnceWhenLocalValidationRejectsTheFirstValue(t *testing.T) {
	repository := &repairingRepository{values: []string{`{"label":"wrong"}`, `{"label":"infra"}`}}
	service := NewService(repository)

	response := service.Run(t.Context(), ProviderRequest{
		Source:      "connection refused",
		SchemaJSON:  `{"type":"object","required":["label"],"properties":{"label":{"enum":["infra"]}}}`,
		Instruction: "Classify by root cause.",
		Role:        "classify.fast",
	})

	require.True(t, response.GetValidated())
	require.Equal(t, `{"label":"infra"}`, response.GetValueJson())

	requests := repository.snapshot()
	require.Len(t, requests, 2, "a rejected value should be re-asked exactly once")
	require.Equal(t, "Classify by root cause.", requests[0].Instruction)
	require.Contains(t, requests[1].Instruction, "Classify by root cause.",
		"the repair attempt must preserve the caller's original intent")
	require.Contains(t, requests[1].Instruction, "rejected by local schema validation",
		"the repair attempt must carry the validator's reason")
	require.Equal(t, requests[0].SchemaJSON, requests[1].SchemaJSON,
		"repair changes instruction only; the schema gate stays the single description of what is enforceable")

	// Usage is billed for the work performed, not only the successful attempt.
	require.EqualValues(t, 10, response.GetUsage().GetInputTokens())
	require.EqualValues(t, 4, response.GetUsage().GetOutputTokens())
}

func TestServiceStopsReAskingAfterTheAttemptBudget(t *testing.T) {
	repository := &repairingRepository{values: []string{`{"label":"wrong"}`}}
	service := NewService(repository)

	response := service.Run(t.Context(), ProviderRequest{
		Source:     "source",
		SchemaJSON: `{"type":"object","required":["label"],"properties":{"label":{"enum":["infra"]}}}`,
		Role:       "classify.fast",
	})

	require.False(t, response.GetValidated())
	require.Equal(t, "INFERENCE_ERROR_CODE_VALIDATION_FAILED", response.GetError().GetCode().String())
	require.Len(t, repository.snapshot(), MaxValidationAttempts,
		"an unbounded repair loop would make cost unpredictable for callers")
}

// A provider failure is terminal here: the repository already walks every
// declared candidate before reporting one, so re-asking would only duplicate
// that work.
func TestServiceDoesNotReAskAfterAProviderFailure(t *testing.T) {
	repository := &countingFailingRepository{}
	service := NewService(repository)

	response := service.Run(t.Context(), ProviderRequest{Source: "source", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"})

	require.False(t, response.GetValidated())
	require.Equal(t, "INFERENCE_ERROR_CODE_UNAVAILABLE", response.GetError().GetCode().String())
	require.Equal(t, 1, repository.calls())
}

type countingFailingRepository struct {
	mu    sync.Mutex
	count int
}

func (r *countingFailingRepository) Run(context.Context, ProviderRequest) (ProviderResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.count++
	return ProviderResult{}, ErrUnavailable
}

func (r *countingFailingRepository) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

func TestRoleTimeoutReachesTheProviderCommand(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy resolve --role classify.routing --json":                                                                 {Stdout: `{"role":"classify.routing","model":"qwen3.5:4b"}`},
		"resource-ollama gateway generate --role classify.routing --json --prompt-stdin --temperature 0 --format {\"type\":\"string\"}": {Stdout: `{"response":"\"infra\"","eval_count":2}`},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {TimeoutMS: 45000, Candidates: []Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "test"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner}})
	require.NoError(t, err)

	_, err = repository.Run(context.Background(), ProviderRequest{Source: "s", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"})
	require.NoError(t, err)

	var generate *providers.Command
	for index := range runner.Commands {
		if strings.Contains(strings.Join(runner.Commands[index].Args, " "), "gateway generate") {
			generate = &runner.Commands[index]
		}
	}
	require.NotNil(t, generate, "the generate command should have been issued")
	require.Equal(t, 45*time.Second, generate.Timeout,
		"generation must not inherit providers.DefaultCommandTimeout, which is sized for metadata commands")
	require.Greater(t, generate.Timeout, providers.DefaultCommandTimeout)
}

func TestRoleTimeoutFallsBackToTheDefaultWhenUndeclared(t *testing.T) {
	require.Equal(t, DefaultRoleTimeout, InferenceRole{}.Timeout())
	require.Equal(t, 2*time.Second, InferenceRole{TimeoutMS: 2000}.Timeout())
}

func TestCatalogRejectsATimeoutBeyondTheMaximum(t *testing.T) {
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {
			TimeoutMS:  int(MaxRoleTimeout/time.Millisecond) + 1,
			Candidates: []Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "test"}},
		},
	}}
	require.ErrorContains(t, catalog.Validate(), "exceeds")
}

// A malformed catalog must degrade typed inference alone. Routing, measures,
// and conformance have no dependency on it and must keep serving.
func TestUnavailableRepositoryReportsAStatedReason(t *testing.T) {
	service := NewService(UnavailableRepository{Reason: "catalog is unreadable"})
	response := service.Run(t.Context(), ProviderRequest{Source: "s", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"})

	require.False(t, response.GetValidated())
	require.Equal(t, "INFERENCE_ERROR_CODE_UNAVAILABLE", response.GetError().GetCode().String())
	require.Contains(t, response.GetError().GetMessage(), "catalog is unreadable")
}

// The batch endpoint exists so a program can classify at volume; running items
// one at a time would defeat its purpose.
func TestServiceBatchRunsItemsConcurrently(t *testing.T) {
	const items = MaxBatchConcurrency
	repository := &blockingRepository{gate: make(chan struct{}), arrived: make(chan struct{}, items)}
	service := NewService(repository)

	requests := make([]ProviderRequest, items)
	for index := range requests {
		requests[index] = ProviderRequest{Source: "s", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"}
	}

	done := make(chan *struct{}, 1)
	go func() {
		service.RunBatch(context.Background(), requests)
		done <- nil
	}()

	// Every worker must reach the provider before any is released. Sequential
	// execution cannot satisfy this and fails the test by timeout.
	for count := 0; count < items; count++ {
		select {
		case <-repository.arrived:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of %d items were in flight; batch is running sequentially", count, items)
		}
	}
	close(repository.gate)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("batch did not complete after workers were released")
	}
}

type blockingRepository struct {
	gate    chan struct{}
	arrived chan struct{}
}

func (r *blockingRepository) Run(_ context.Context, request ProviderRequest) (ProviderResult, error) {
	r.arrived <- struct{}{}
	<-r.gate
	return ProviderResult{ValueJSON: `"` + request.Source + `"`, Provider: "fake", Model: "test"}, nil
}
