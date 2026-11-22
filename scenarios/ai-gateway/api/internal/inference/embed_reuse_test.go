package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/rolecatalog"
)

// embeddingContract is the contract the fixture role declares.
var embeddingContract = &rolecatalog.RoleEmbedding{Normalization: rolecatalog.NormalizationL2, MaxBatchInputs: 64, MaxBatchBytes: 1 << 20}

// unitVector is the vector the fixture resource returns for a text: a unit
// vector whose direction depends on the text length, so a test can tell which
// text each vector belongs to.
func unitVector(text string) []float64 {
	angle := float64(len(text)) / 10
	return []float64{math.Cos(angle), math.Sin(angle)}
}

// embeddingRunner answers `gateway embed --input-json-stdin` like the resource
// does: one vector per input, as `embedding` for one input and `embeddings` for
// several. Every other command goes to the fixture runner.
type embeddingRunner struct {
	*providermocks.FakeRunner
	embedFailures atomic.Int32 // number of upcoming gateway embed calls to fail
	timeouts      atomic.Int32 // number of upcoming gateway embed calls to time out
	// failRole makes gateway embed calls for that resource role fail once
	// failAfter of them have succeeded.
	failRole  string
	failAfter int32
	roleCalls atomic.Int32
	// scale multiplies every returned vector; zero means 1.
	scale float64
	// dropVectors removes that many vectors from every answer.
	dropVectors int
}

func (e *embeddingRunner) Run(ctx context.Context, command providers.Command) (providers.Result, error) {
	if !strings.Contains(command.String(), "gateway embed") {
		return e.FakeRunner.Run(ctx, command)
	}
	_, _ = e.FakeRunner.Run(ctx, command) // record it
	if e.failRole != "" && strings.Contains(command.String(), "--role "+e.failRole+" ") && e.roleCalls.Add(1) > e.failAfter {
		return providers.Result{}, &providers.CommandError{Code: "exit_error", Command: command.String(), ExitCode: 1}
	}
	if e.timeouts.Add(-1) >= 0 {
		return providers.Result{}, &providers.CommandError{Code: "timeout", Command: command.String(), ExitCode: -1, Err: context.DeadlineExceeded}
	}
	e.timeouts.Store(0)
	if e.embedFailures.Add(-1) >= 0 {
		return providers.Result{}, &providers.CommandError{Code: "exit_error", Command: command.String(), ExitCode: 1}
	}
	e.embedFailures.Store(0)
	var body struct {
		Inputs []string `json:"inputs"`
	}
	if err := json.Unmarshal([]byte(command.Stdin), &body); err != nil || !strings.Contains(command.String(), "--input-json-stdin") {
		return providers.Result{}, fmt.Errorf("fixture resource expects a JSON batch on stdin, got %q", command.Stdin)
	}
	scale := e.scale
	if scale == 0 {
		scale = 1
	}
	vectors := make([][]float64, 0, len(body.Inputs))
	for _, text := range body.Inputs {
		vector := unitVector(text)
		vector[0], vector[1] = vector[0]*scale, vector[1]*scale
		vectors = append(vectors, vector)
	}
	vectors = vectors[:len(vectors)-e.dropVectors]
	var encoded []byte
	if len(vectors) == 1 {
		encoded, _ = json.Marshal(map[string]any{"embedding": vectors[0]})
	} else {
		encoded, _ = json.Marshal(map[string]any{"embeddings": vectors})
	}
	return providers.Result{Stdout: string(encoded)}, nil
}

func (e *embeddingRunner) count(fragment string) int {
	n := 0
	for _, command := range e.FakeRunner.Commands {
		if strings.Contains(command.String(), fragment) {
			n++
		}
	}
	return n
}

func embeddingRepository(t *testing.T, runner *embeddingRunner, contract *rolecatalog.RoleEmbedding, resourceRoles ...string) *ResourceRepository {
	t.Helper()
	return embeddingRepositoryOver(t, runner, contract, resourceRoles...)
}

// embeddingRepositoryOver builds the fixture repository on any runner, such as
// a FactCache in front of the fixture resource.
func embeddingRepositoryOver(t *testing.T, runner providers.CommandRunner, contract *rolecatalog.RoleEmbedding, resourceRoles ...string) *ResourceRepository {
	t.Helper()
	candidates := make([]rolecatalog.Candidate, 0, len(resourceRoles))
	for _, role := range resourceRoles {
		candidates = append(candidates, rolecatalog.Candidate{Provider: providers.ProviderOllama, ResourceRole: role, Reason: "fixture"})
	}
	catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
		"embedding.default": {Candidates: candidates, Embedding: contract},
	}}
	repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner}}, fixtureRightsContext)
	require.NoError(t, err)
	return repository
}

func newEmbeddingFixture(t *testing.T) (*ResourceRepository, *embeddingRunner) {
	t.Helper()
	runner := &embeddingRunner{FakeRunner: &providermocks.FakeRunner{}}
	configureQualifiedLocalRole(runner.FakeRunner, "embed.resource", "embed-model", "local")
	return embeddingRepository(t, runner, embeddingContract, "embed.resource"), runner
}

func embeddingRequest(texts ...string) EmbeddingRequest {
	return EmbeddingRequest{
		Role: "embedding.default", Texts: texts, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, PrivacyProvenance: "gateway_test.synthetic_embedding_fixture",
	}
}

// The cost of an embedding request was proportional to its texts because each
// text spawned a resource command that re-resolved and re-authorized the role.
// A request now pays for authorization once and for one resource call.
func TestEmbeddingBatchIsOneAuthorizationAndOneResourceCall(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	texts := []string{"a", "bb", "ccc", "dddd", "eeeee", "ffffff", "ggggggg"}

	result, err := repository.Embed(context.Background(), embeddingRequest(texts...))
	require.NoError(t, err)
	require.Len(t, result.Vectors, len(texts))
	for index, text := range texts {
		require.Equal(t, unitVector(text), result.Vectors[index], "vector %d belongs to text %q", index, text)
	}
	require.Equal(t, rolecatalog.NormalizationL2, result.Normalization)
	require.Equal(t, 2, runner.count("policy resolve"), "one resolve and one revalidation per request, not per text")
	require.Equal(t, 1, runner.count("gateway embed"), "all texts reach the resource in one call")
	require.Equal(t, 1, runner.count("--input-json-stdin"))
}

// One text and many texts take the same path, so they share one scale.
func TestSingleEmbeddingUsesTheBatchContract(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	result, err := repository.Embed(context.Background(), embeddingRequest("only"))
	require.NoError(t, err)
	require.Equal(t, [][]float64{unitVector("only")}, result.Vectors)
	require.Equal(t, 1, runner.count("--input-json-stdin"))
	require.Zero(t, runner.count("--input-stdin"))
}

func TestEmbeddingSplitsAtTheRoleBatchBoundsAndKeepsOrder(t *testing.T) {
	runner := &embeddingRunner{FakeRunner: &providermocks.FakeRunner{}}
	configureQualifiedLocalRole(runner.FakeRunner, "embed.resource", "embed-model", "local")
	repository := embeddingRepository(t, runner, &rolecatalog.RoleEmbedding{Normalization: rolecatalog.NormalizationL2, MaxBatchInputs: 3, MaxBatchBytes: 1 << 20}, "embed.resource")
	texts := make([]string, 8)
	for index := range texts {
		texts[index] = strings.Repeat("x", index+1)
	}

	result, err := repository.Embed(context.Background(), embeddingRequest(texts...))
	require.NoError(t, err)
	require.Equal(t, 3, runner.count("gateway embed"), "8 texts at 3 per call")
	require.Equal(t, 2, runner.count("policy resolve"), "authorization is still once per request")
	for index, text := range texts {
		require.Equal(t, unitVector(text), result.Vectors[index], "vector %d belongs to text %q", index, text)
	}
}

func TestEmbeddingBatchesRespectBothBounds(t *testing.T) {
	contract := rolecatalog.RoleEmbedding{Normalization: rolecatalog.NormalizationL2, MaxBatchInputs: 2, MaxBatchBytes: 40}
	require.Equal(t, [][]string{{"a", "b"}, {"c"}}, embeddingBatches([]string{"a", "b", "c"}, contract), "input count bound")

	// {"inputs":[]} is 13 bytes; each 10-character text adds 12 encoded + 1.
	ten := strings.Repeat("y", 10)
	require.Equal(t, [][]string{{ten, ten}, {ten}}, embeddingBatches([]string{ten, ten, ten}, rolecatalog.RoleEmbedding{MaxBatchInputs: 64, MaxBatchBytes: 13 + 2*13}), "encoded size bound")

	oversized := strings.Repeat("z", 100)
	require.Equal(t, [][]string{{"a"}, {oversized}, {"b"}}, embeddingBatches([]string{"a", oversized, "b"}, contract), "a text over the bound travels alone")
	require.Empty(t, embeddingBatches(nil, contract))
}

// newCachedEmbeddingFixture puts the gateway's fact cache in front of the
// fixture resource, as production does.
func newCachedEmbeddingFixture(t *testing.T) (*ResourceRepository, *embeddingRunner, func(time.Duration)) {
	t.Helper()
	runner := &embeddingRunner{FakeRunner: &providermocks.FakeRunner{}}
	configureQualifiedLocalRole(runner.FakeRunner, "embed.resource", "embed-model", "local")
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	advance := func(d time.Duration) { mu.Lock(); defer mu.Unlock(); now = now.Add(d) }
	cache := providers.NewFactCache(runner, providers.DefaultFactLifetime, providers.WithFactClock(clock))
	return embeddingRepositoryOver(t, cache, embeddingContract, "embed.resource"), runner, advance
}

// Resolution facts are read from the resource once per fact lifetime, not once
// per request; every call still reaches the resource, which re-verifies the
// evidence it is told to expect.
func TestEmbeddingReusesResourceFactsUntilTheyExpire(t *testing.T) {
	repository, runner, advance := newCachedEmbeddingFixture(t)

	_, err := repository.Embed(context.Background(), embeddingRequest("one"))
	require.NoError(t, err)
	require.Equal(t, 1, runner.count("policy resolve"), "the revalidation reads the same fact")

	advance(5 * time.Second)
	_, err = repository.Embed(context.Background(), embeddingRequest("two"))
	require.NoError(t, err)
	require.Equal(t, 1, runner.count("policy resolve"), "before its refresh age the resolution is reused")
	require.Equal(t, 2, runner.count("gateway embed"), "every call still reaches the resource")

	advance(providers.DefaultFactLifetime.MaxAge)
	_, err = repository.Embed(context.Background(), embeddingRequest("three"))
	require.NoError(t, err)
	require.Equal(t, 2, runner.count("policy resolve"), "past its maximum age the role is resolved again")
}

// The resource refuses a call whose expected evidence no longer matches the
// current policy, rights context or model. Reused facts must then be dropped
// and the authorization redone once from fresh ones, so a change is never
// ignored for the length of the fact lifetime.
func TestStaleEmbeddingFactsAreDroppedAndAuthorizationRedoneOnce(t *testing.T) {
	repository, runner, _ := newCachedEmbeddingFixture(t)
	_, err := repository.Embed(context.Background(), embeddingRequest("warm"))
	require.NoError(t, err)
	require.Equal(t, 1, runner.count("policy resolve"))

	runner.embedFailures.Store(1) // the resource refuses the stale evidence once
	result, err := repository.Embed(context.Background(), embeddingRequest("changed"))
	require.NoError(t, err)
	require.Len(t, result.Vectors, 1)
	require.Equal(t, 2, runner.count("policy resolve"), "the stale facts are replaced by fresh ones")
	require.Equal(t, 3, runner.count("gateway embed"))

	runner.embedFailures.Store(100) // the resource keeps refusing
	_, err = repository.Embed(context.Background(), embeddingRequest("still changed"))
	require.ErrorIs(t, err, ErrUnavailable)
	require.Equal(t, 3, runner.count("policy resolve"), "one retry with fresh facts, then the candidate fails")
	require.Equal(t, 5, runner.count("gateway embed"))
	_, err = repository.Embed(context.Background(), embeddingRequest("after failure"))
	require.Error(t, err)
	require.Equal(t, 4, runner.count("policy resolve"), "a failed candidate leaves no facts behind")
}

// A call that ran out of time is not the resource refusing stale evidence;
// repeating it would only double the caller's wait.
func TestEmbeddingTimeoutIsNotRetriedWithFreshFacts(t *testing.T) {
	repository, runner, _ := newCachedEmbeddingFixture(t)
	_, err := repository.Embed(context.Background(), embeddingRequest("warm"))
	require.NoError(t, err)
	runner.timeouts.Store(1)
	_, err = repository.Embed(context.Background(), embeddingRequest("slow"))
	require.ErrorIs(t, err, ErrUnavailable)
	require.Equal(t, 2, runner.count("gateway embed"), "the timed-out call ran once")
}

// One response is one vector space. If a batch fails on the first candidate the
// whole request moves to the next candidate; it never mixes candidates.
func TestEmbeddingNeverMixesCandidatesWithinOneResponse(t *testing.T) {
	runner := &embeddingRunner{FakeRunner: &providermocks.FakeRunner{}}
	configureQualifiedLocalRole(runner.FakeRunner, "embed.first", "model-first", "local")
	configureQualifiedLocalRole(runner.FakeRunner, "embed.second", "model-second", "local")
	runner.failRole = "embed.first"
	runner.failAfter = 1 // the first candidate answers one batch and then fails
	repository := embeddingRepository(t, runner, &rolecatalog.RoleEmbedding{Normalization: rolecatalog.NormalizationL2, MaxBatchInputs: 2, MaxBatchBytes: 1 << 20}, "embed.first", "embed.second")

	result, err := repository.Embed(context.Background(), embeddingRequest("aa", "bbb", "cccc"))
	require.NoError(t, err)
	require.Equal(t, "model-second", result.Model)
	require.Equal(t, [][]float64{unitVector("aa"), unitVector("bbb"), unitVector("cccc")}, result.Vectors)
	require.Equal(t, 2, runner.count("gateway embed --role embed.second"), "the second candidate answers every batch")
}

func TestEmbeddingStillRefusesRemoteCandidatesWithAWarmAuthorizationCache(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	_, err := repository.Embed(context.Background(), embeddingRequest("warm the cache"))
	require.NoError(t, err)

	request := embeddingRequest("not local")
	request.Profile = sharedv1.Profile_PROFILE_LOCAL_FIRST
	_, err = repository.Embed(context.Background(), request)
	require.ErrorIs(t, err, ErrUnavailable, "typed embeddings need an explicit local-only profile whatever is cached")
	require.Equal(t, 1, runner.count("gateway embed"), "the refused request never reached the resource")
	require.True(t, errors.Is(err, ErrUnavailable))
}

// A role that declares l2 vectors is only as good as the check on them: a
// candidate whose vectors have another scale must not pass them on.
func TestEmbeddingRefusesVectorsThatBreakTheDeclaredNormalization(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	runner.scale = 15.9 // the raw scale of the legacy single-text endpoint
	_, err := repository.Embed(context.Background(), embeddingRequest("one", "two"))
	require.ErrorIs(t, err, ErrUnavailable)
	require.Contains(t, err.Error(), "embedding_not_normalized")
}

func TestEmbeddingRefusesAnAnswerWithTheWrongNumberOfVectors(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	runner.dropVectors = 1
	_, err := repository.Embed(context.Background(), embeddingRequest("one", "two", "three"))
	require.ErrorIs(t, err, ErrUnavailable)
	require.Contains(t, err.Error(), "returned 2 embedding vectors for 3 texts")
}

func TestEmbeddingRequiresARoleThatDeclaresItsContract(t *testing.T) {
	runner := &embeddingRunner{FakeRunner: &providermocks.FakeRunner{}}
	configureQualifiedLocalRole(runner.FakeRunner, "embed.resource", "embed-model", "local")
	repository := embeddingRepository(t, runner, nil, "embed.resource")
	_, err := repository.Embed(context.Background(), embeddingRequest("one"))
	require.ErrorIs(t, err, ErrUnavailable)
	require.Contains(t, err.Error(), "declares no embedding contract")
	require.Zero(t, runner.count("gateway embed"))
}
