package routing_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/routing"
)

// slowEmbedRunner answers `gateway embed` the way resource-ollama does,
// including the timing it measures inside the command, after a fixed delay
// that stands in for the command's real cost. Everything else goes to inner.
type slowEmbedRunner struct {
	inner providers.CommandRunner
	delay time.Duration
	fail  atomic.Bool
}

const embedOutput = `{"embedding":[0.6,0.8],"input_limit_tokens":2048,"truncated":false,"admission_wait_ns":2000000,"provider_duration_ns":91000000}`

func (r *slowEmbedRunner) Run(ctx context.Context, command providers.Command) (providers.Result, error) {
	if !strings.Contains(command.String(), "gateway embed") {
		return r.inner.Run(ctx, command)
	}
	_, _ = r.inner.Run(ctx, command) // record it
	time.Sleep(r.delay)
	if r.fail.Load() {
		return providers.Result{}, &providers.CommandError{Code: "exit_error", Command: command.String(), ExitCode: 1, Stderr: "refused"}
	}
	return providers.Result{Stdout: embedOutput}, nil
}

func embeddingRoute() *sharedv1.GatewayRequest {
	return &sharedv1.GatewayRequest{
		Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, Role: "embedding.default",
		Profile: sharedv1.Profile_PROFILE_LOCAL_FIRST, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		PrivacyProvenance: "routing_test.synthetic_embedding", Scenario: "fixture-scenario", TimeoutMs: 15000,
	}
}

// Provider work and the time spent around it are recorded apart, so a slow
// gateway or resource command can no longer hide inside "provider" time.
func TestRoutedEmbeddingRecordsProviderTimeApartFromOverhead(t *testing.T) {
	runner := &slowEmbedRunner{inner: roleRunner(), delay: 120 * time.Millisecond}
	repo := routing.NewSQLRepository(newSchemaDB(t))
	svc := newService(testAdapters(runner), repo)

	resp, err := svc.Execute(context.Background(), embeddingRoute(), "search_document: hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())

	stored, err := repo.Get(context.Background(), resp.GetEvidence().GetEventId())
	require.NoError(t, err)
	require.Equal(t, int64(91_000_000), stored.GetProviderRuntimeDurationNs(), "the resource's own provider measurement")
	require.Equal(t, int64(2_000_000), stored.GetProviderAdmissionWaitNs())
	require.GreaterOrEqual(t, stored.GetProviderExecutionDurationNs(), int64(120*time.Millisecond), "the whole resource command")
	require.GreaterOrEqual(t, stored.GetTotalDurationNs(), stored.GetProviderExecutionDurationNs())
	require.Equal(t, stored.GetTotalDurationNs()-91_000_000-2_000_000, stored.GetOverheadDurationNs(),
		"overhead is everything outside provider work and admission, including the command's own start-up")
	require.GreaterOrEqual(t, stored.GetOverheadDurationNs(), int64(25*time.Millisecond))
}

// A resource that reports no timing has its whole command counted as provider
// time; the overhead is then only the gateway's own share.
func TestOverheadFallsBackToCommandTimeWhenTheResourceReportsNone(t *testing.T) {
	runner := roleRunner()
	repo := routing.NewSQLRepository(newSchemaDB(t))
	svc := newService(testAdapters(runner), repo)
	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.NoError(t, err)
	ev := resp.GetEvidence()
	require.Equal(t, "succeeded", ev.GetStatus())
	require.Zero(t, ev.GetProviderRuntimeDurationNs())
	require.Equal(t, max(ev.GetTotalDurationNs()-ev.GetProviderExecutionDurationNs(), 0), ev.GetOverheadDurationNs())
}

func countCommands(commands []providers.Command, fragment string) int {
	n := 0
	for _, command := range commands {
		if strings.Contains(command.String(), fragment) {
			n++
		}
	}
	return n
}

// Planning a route used to read every resource's policy and inventory, twice
// for local ones, plus a role resolution per request: about two seconds of
// resource processes before a 91 ms embedding. Behind the fact cache a steady
// stream of routes reads each fact once, and every call still reaches the
// resource, which re-verifies what it is told to expect.
func TestConsecutiveRoutesReadResourceFactsOnceAndRereadAfterAFailure(t *testing.T) {
	inner := roleRunner()
	runner := &slowEmbedRunner{inner: inner}
	cache := providers.NewFactCache(runner, providers.DefaultFactLifetime)
	svc := newService(testAdapters(cache), routing.NewSQLRepository(newSchemaDB(t)))

	for range 3 {
		resp, err := svc.Execute(context.Background(), embeddingRoute(), "search_document: hello")
		require.NoError(t, err)
		require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	}
	require.Equal(t, 1, countCommands(inner.Commands, "resource-ollama policy roles"))
	require.Equal(t, 1, countCommands(inner.Commands, "resource-openrouter policy roles"))
	require.Equal(t, 1, countCommands(inner.Commands, "resource-ollama models inventory"))
	require.Equal(t, 1, countCommands(inner.Commands, "resource-ollama policy resolve --role embedding.default"))
	require.Equal(t, 3, countCommands(inner.Commands, "gateway embed"), "every call reaches the resource")

	runner.fail.Store(true)
	resp, err := svc.Execute(context.Background(), embeddingRoute(), "search_document: refused")
	require.NoError(t, err)
	require.Equal(t, "failed", resp.GetEvidence().GetStatus())
	runner.fail.Store(false)
	resp, err = svc.Execute(context.Background(), embeddingRoute(), "search_document: after")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, 2, countCommands(inner.Commands, "resource-ollama policy roles"), "a failed dispatch drops the resource's facts")
	require.Equal(t, 1, countCommands(inner.Commands, "resource-openrouter policy roles"), "other resources keep theirs")
}

// Each local candidate's policy is read once per plan, not once to classify it
// and again to plan its rights.
func TestPlanningReadsEachResourcePolicyOnce(t *testing.T) {
	runner := roleRunner()
	svc := newService(testAdapters(runner), nil)
	_, err := svc.Preview(context.Background(), embeddingRoute())
	require.NoError(t, err)
	require.Equal(t, 1, countCommands(runner.Commands, "resource-ollama policy roles"))
	require.Equal(t, 1, countCommands(runner.Commands, "resource-ollama models inventory"))
}

// slowFactRunner makes every policy read take a fixed time, like a resource
// process does on a cold cache.
type slowFactRunner struct {
	inner providers.CommandRunner
	delay time.Duration
}

func (r slowFactRunner) Run(ctx context.Context, command providers.Command) (providers.Result, error) {
	if strings.Contains(command.String(), " policy roles ") {
		time.Sleep(r.delay)
	}
	return r.inner.Run(ctx, command)
}

// On a cold cache every provider's facts are resource processes. Reading them
// one provider after another made the first plan take the sum of all of them;
// they are read at once, so it takes about the slowest one.
func TestColdPlanReadsProvidersInParallel(t *testing.T) {
	const delay = 300 * time.Millisecond
	svc := newService(testAdapters(slowFactRunner{inner: roleRunner(), delay: delay}), nil)
	started := time.Now()
	resp, err := svc.Preview(context.Background(), embeddingRoute())
	require.NoError(t, err)
	require.Equal(t, "ollama", resp.GetSelectedProvider())
	require.Less(t, time.Since(started), 2*delay-50*time.Millisecond, "two providers' reads overlap")
}
