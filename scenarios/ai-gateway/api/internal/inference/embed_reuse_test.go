package inference

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/rolecatalog"
)

// embeddingRunner answers `gateway embed` with a vector derived from the text
// on stdin, so a test can tell which text each vector belongs to, and
// delegates every other command to the fixture runner.
type embeddingRunner struct {
	*providermocks.FakeRunner
	embedFailures atomic.Int32 // number of upcoming gateway embed calls to fail
	embedCalls    atomic.Int32
	// failRole makes gateway embed calls for that resource role fail once
	// failAfter of them have succeeded.
	failRole  string
	failAfter int32
	roleCalls atomic.Int32
}

func (e *embeddingRunner) Run(ctx context.Context, command providers.Command) (providers.Result, error) {
	if !strings.Contains(command.String(), "gateway embed") {
		return e.FakeRunner.Run(ctx, command)
	}
	_, _ = e.FakeRunner.Run(ctx, command) // record it
	e.embedCalls.Add(1)
	if e.failRole != "" && strings.Contains(command.String(), "--role "+e.failRole+" ") && e.roleCalls.Add(1) > e.failAfter {
		return providers.Result{}, &providers.CommandError{Code: "exit_error", Command: command.String(), ExitCode: 1}
	}
	if e.embedFailures.Add(-1) >= 0 {
		return providers.Result{}, &providers.CommandError{Code: "exit_error", Command: command.String(), ExitCode: 1}
	}
	e.embedFailures.Store(0)
	return providers.Result{Stdout: fmt.Sprintf(`{"embedding":[%d,0]}`, len(command.Stdin))}, nil
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

func newEmbeddingFixture(t *testing.T) (*ResourceRepository, *embeddingRunner) {
	t.Helper()
	runner := &embeddingRunner{FakeRunner: &providermocks.FakeRunner{}}
	configureQualifiedLocalRole(runner.FakeRunner, "embed.resource", "embed-model", "local")
	catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
		"embedding.default": {Candidates: []rolecatalog.Candidate{{Provider: providers.ProviderOllama, ResourceRole: "embed.resource", Reason: "fixture"}}},
	}}
	repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner}}, fixtureRightsContext)
	require.NoError(t, err)
	return repository, runner
}

func embeddingRequest(texts ...string) EmbeddingRequest {
	return EmbeddingRequest{
		Role: "embedding.default", Texts: texts, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, PrivacyProvenance: "gateway_test.synthetic_embedding_fixture",
	}
}

// The cost of an embedding request was proportional to its texts because each
// text re-resolved and re-authorized the role. A request now pays for that once.
func TestEmbeddingAuthorizesOncePerRequestAndKeepsTextOrder(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	zero := time.Duration(0)
	repository.authorizationTTL = &zero // no reuse across requests in this test
	texts := []string{"a", "bb", "ccc", "dddd", "eeeee", "ffffff", "ggggggg"}

	result, err := repository.Embed(context.Background(), embeddingRequest(texts...))
	require.NoError(t, err)
	require.Len(t, result.Vectors, len(texts))
	for index, text := range texts {
		require.Equal(t, []float64{float64(len(text)), 0}, result.Vectors[index], "vector %d belongs to text %q", index, text)
	}
	require.Equal(t, 2, runner.count("policy resolve"), "one resolve and one revalidation per request, not per text")
	require.Equal(t, len(texts), runner.count("gateway embed"))
}

func TestEmbeddingReusesAuthorizationUntilItExpires(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	repository.clock = func() time.Time { return now }

	_, err := repository.Embed(context.Background(), embeddingRequest("one"))
	require.NoError(t, err)
	require.Equal(t, 2, runner.count("policy resolve"))

	now = now.Add(5 * time.Second)
	_, err = repository.Embed(context.Background(), embeddingRequest("two"))
	require.NoError(t, err)
	require.Equal(t, 2, runner.count("policy resolve"), "inside the TTL the resolution is reused")
	require.Equal(t, 2, runner.count("gateway embed"), "every call still reaches the resource, which re-verifies the evidence")

	now = now.Add(defaultEmbeddingAuthorizationTTL)
	_, err = repository.Embed(context.Background(), embeddingRequest("three"))
	require.NoError(t, err)
	require.Equal(t, 4, runner.count("policy resolve"), "after the TTL the role is resolved and authorized again")
}

// The resource refuses a call whose expected evidence no longer matches the
// current policy, rights context or model. A reused authorization must then be
// dropped and the full authorization run once, so a change is never ignored
// for the length of the TTL.
func TestStaleEmbeddingAuthorizationIsDroppedAndRedoneOnce(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	_, err := repository.Embed(context.Background(), embeddingRequest("warm"))
	require.NoError(t, err)
	require.Equal(t, 2, runner.count("policy resolve"))

	runner.embedFailures.Store(1) // the resource refuses the stale evidence once
	result, err := repository.Embed(context.Background(), embeddingRequest("changed"))
	require.NoError(t, err)
	require.Len(t, result.Vectors, 1)
	require.Equal(t, 4, runner.count("policy resolve"), "the stale entry is replaced by a fresh authorization")

	runner.embedFailures.Store(100) // the resource keeps refusing
	_, err = repository.Embed(context.Background(), embeddingRequest("still changed"))
	require.ErrorIs(t, err, ErrUnavailable)
	require.Equal(t, 6, runner.count("policy resolve"), "one retry with a fresh authorization, then the candidate fails")
	_, err = repository.Embed(context.Background(), embeddingRequest("after failure"))
	require.Error(t, err)
	require.Equal(t, 8, runner.count("policy resolve"), "a failed candidate leaves nothing cached")
}

// One response is one vector space. If a text fails on the first candidate the
// whole request moves to the next candidate; it never mixes candidates.
func TestEmbeddingNeverMixesCandidatesWithinOneResponse(t *testing.T) {
	runner := &embeddingRunner{FakeRunner: &providermocks.FakeRunner{}}
	configureQualifiedLocalRole(runner.FakeRunner, "embed.first", "model-first", "local")
	configureQualifiedLocalRole(runner.FakeRunner, "embed.second", "model-second", "local")
	runner.failRole = "embed.first"
	runner.failAfter = 1 // the first candidate answers one text and then fails
	catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
		"embedding.default": {Candidates: []rolecatalog.Candidate{
			{Provider: providers.ProviderOllama, ResourceRole: "embed.first", Reason: "fixture"},
			{Provider: providers.ProviderOllama, ResourceRole: "embed.second", Reason: "fixture"},
		}},
	}}
	repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner}}, fixtureRightsContext)
	require.NoError(t, err)

	result, err := repository.Embed(context.Background(), embeddingRequest("aa", "bbb", "cccc"))
	require.NoError(t, err)
	require.Equal(t, "model-second", result.Model)
	require.Equal(t, [][]float64{{2, 0}, {3, 0}, {4, 0}}, result.Vectors)
	for _, command := range runner.FakeRunner.Commands {
		if strings.Contains(command.String(), "gateway embed --role embed.second") {
			return
		}
	}
	t.Fatal("every text of the response must come from the second candidate")
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
