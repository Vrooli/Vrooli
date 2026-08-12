package searchregister_test

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/retry"
	searchregister "github.com/vrooli/searchregister-go"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// fakeClient records every RegisterProvider call and returns a scripted outcome.
// failFirst makes the first N attempts fail (to exercise the retry path) before
// succeeding; alwaysErr makes every attempt fail (to exercise graceful degrade).
type fakeClient struct {
	mu           sync.Mutex
	calls        []*registryv1.ProviderDescriptor
	presented    []string // the control_token echoed on each request, in call order
	failFirst    int
	created      bool
	token        string
	alwaysErr    error
	failProvider string // when set, RegisterProvider errors for this provider_id only
}

func (f *fakeClient) RegisterProvider(
	_ context.Context,
	req *connect.Request[registryv1.RegisterProviderRequest],
) (*connect.Response[registryv1.RegisterProviderResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req.Msg.GetDescriptor_())
	f.presented = append(f.presented, req.Msg.GetControlToken())
	if f.alwaysErr != nil {
		return nil, f.alwaysErr
	}
	if f.failProvider != "" && req.Msg.GetDescriptor_().GetProviderId() == f.failProvider {
		return nil, errors.New("hub rejected " + f.failProvider)
	}
	if f.failFirst > 0 {
		f.failFirst--
		return nil, errors.New("hub not ready")
	}
	return connect.NewResponse(&registryv1.RegisterProviderResponse{
		Descriptor_:  req.Msg.GetDescriptor_(),
		Created:      f.created,
		ControlToken: f.token,
	}), nil
}

func (f *fakeClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// fakeEvalClient records every RegisterSuite call (the corpus mirror) and returns
// a scripted outcome; alwaysErr makes every attempt fail (graceful-degrade path).
type fakeEvalClient struct {
	mu        sync.Mutex
	suites    []*evalv1.EvalSuite
	alwaysErr error
}

func (f *fakeEvalClient) RegisterSuite(
	_ context.Context,
	req *connect.Request[evalv1.RegisterSuiteRequest],
) (*connect.Response[evalv1.RegisterSuiteResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.suites = append(f.suites, req.Msg.GetSuite())
	if f.alwaysErr != nil {
		return nil, f.alwaysErr
	}
	return connect.NewResponse(&evalv1.RegisterSuiteResponse{Suite: req.Msg.GetSuite(), Created: true}), nil
}

func (f *fakeEvalClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.suites)
}

func testLogger(t *testing.T) *log.Logger {
	t.Helper()
	return log.New(io.Discard, "", 0)
}

func writeSearchFile(t *testing.T, raw string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "search.json")
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))
	return path
}

func cfgWith(t *testing.T, raw string, client *fakeClient) searchregister.Config {
	t.Helper()
	// A success-by-default eval fake keeps the existing tests hermetic (corpus
	// mirroring no longer dials the network). Corpus-specific tests use cfgWithEval.
	return cfgWithEval(t, raw, client, &fakeEvalClient{})
}

func cfgWithEval(t *testing.T, raw string, client *fakeClient, evalClient *fakeEvalClient) searchregister.Config {
	t.Helper()
	return searchregister.Config{
		ScenarioID:     "cli-health",
		SearchFilePath: writeSearchFile(t, declareTestMinimum(raw)),
		Logger:         testLogger(t),
		ResolveBaseURL: func(context.Context) (string, error) { return "http://search-hub.test", nil },
		NewClient:      func(string) searchregister.RegistryClient { return client },
		NewEvalClient:  func(string) searchregister.EvalClient { return evalClient },
		Retry: retry.Config{
			MaxAttempts:    5,
			Sleeper:        func(time.Duration) {},
			Rand:           func() float64 { return 0 },
			JitterFraction: 0,
		},
	}
}

// The registration fixtures predate the production minimum contract. Declare a
// zero threshold in these transport-focused fixtures so they test retry and
// mirroring behavior; dedicated validation tests cover the strict nil case.
func declareTestMinimum(raw string) string {
	minimum := `"minimum":{"reviewed_positive":0,"negative":0},`
	for _, marker := range []string{`"tests": {`, `"tests":{`} {
		raw = strings.ReplaceAll(raw, marker, marker+minimum)
	}
	return raw
}

func TestRegisterHappyPath(t *testing.T) {
	client := &fakeClient{created: true}
	results := searchregister.Register(context.Background(), cfgWith(t, cliHealthSearchJSON, client))

	require.Len(t, results, 1)
	require.NoError(t, results[0].Err)
	require.True(t, results[0].Created)
	require.Equal(t, "cli-health.commands", results[0].ProviderID)
	require.Equal(t, 1, client.callCount())
	require.Equal(t, "cli-health.commands", client.calls[0].GetProviderId())
}

func TestValidateRegistrationRequiresDeclaredMinimumForProduction(t *testing.T) {
	err := searchregister.ValidateRegistration(aisearch.ProviderConfig{ProviderID: "demo.docs"})
	if err == nil || !strings.Contains(err.Error(), "tests.minimum") {
		t.Fatalf("error = %v, want actionable tests.minimum rejection", err)
	}
}

func TestValidateRegistrationAllowsFixtureWithoutMinimum(t *testing.T) {
	if err := searchregister.ValidateRegistration(aisearch.ProviderConfig{ProviderID: "demo.fixture", Lifecycle: "fixture"}); err != nil {
		t.Fatalf("fixture registration: %v", err)
	}
}

func TestRegisterCapturesControlToken(t *testing.T) {
	client := &fakeClient{created: true, token: "tok-abc123"}
	cfg := cfgWith(t, cliHealthSearchJSON, client)
	var gotID, gotToken string
	cfg.OnControlToken = func(providerID, token string) { gotID, gotToken = providerID, token }

	results := searchregister.Register(context.Background(), cfg)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Err)
	require.Equal(t, "tok-abc123", results[0].ControlToken)
	require.Equal(t, "cli-health.commands", gotID)
	require.Equal(t, "tok-abc123", gotToken, "callback receives the minted token")
}

func TestRegisterTokenCallbackNotInvokedWhenEmpty(t *testing.T) {
	// A hub that predates token minting returns an empty token; the callback must
	// not fire (nothing to cache) and registration still succeeds.
	client := &fakeClient{created: true, token: ""}
	cfg := cfgWith(t, cliHealthSearchJSON, client)
	called := false
	cfg.OnControlToken = func(string, string) { called = true }

	results := searchregister.Register(context.Background(), cfg)
	require.NoError(t, results[0].Err)
	require.False(t, called, "no token => callback not invoked")
}

// TestRegisterEchoesCachedControlToken: a scenario that holds a token (from a
// prior registration in this process) presents it on re-registration as ownership
// proof — search-hub rejects an update whose token mismatches the stored one.
func TestRegisterEchoesCachedControlToken(t *testing.T) {
	client := &fakeClient{created: false, token: "tok-abc123"}
	cfg := cfgWith(t, cliHealthSearchJSON, client)
	cfg.ControlToken = func(string) string { return "tok-abc123" }

	results := searchregister.Register(context.Background(), cfg)

	require.NoError(t, results[0].Err)
	require.Equal(t, 1, client.callCount())
	require.Equal(t, "tok-abc123", client.presented[0], "cached token echoed as ownership proof")
}

// TestRegisterPresentsEmptyTokenByDefault: with no ControlToken seam wired (or an
// empty holder — first boot), the request carries an empty token, which search-hub
// treats as first-contact. Proves the echo is opt-in and the default is harmless.
func TestRegisterPresentsEmptyTokenByDefault(t *testing.T) {
	client := &fakeClient{created: true, token: "tok-xyz"}
	results := searchregister.Register(context.Background(), cfgWith(t, cliHealthSearchJSON, client))

	require.NoError(t, results[0].Err)
	require.Equal(t, 1, client.callCount())
	require.Equal(t, "", client.presented[0], "no holder => empty token presented")
}

func TestRegisterRetriesThenSucceeds(t *testing.T) {
	client := &fakeClient{failFirst: 2}
	results := searchregister.Register(context.Background(), cfgWith(t, cliHealthSearchJSON, client))

	require.Len(t, results, 1)
	require.NoError(t, results[0].Err, "should recover after transient failures")
	require.Equal(t, 3, client.callCount(), "2 failures + 1 success")
}

func TestRegisterDegradesGracefully(t *testing.T) {
	client := &fakeClient{alwaysErr: errors.New("connection refused")}
	results := searchregister.Register(context.Background(), cfgWith(t, cliHealthSearchJSON, client))

	require.Len(t, results, 1)
	require.Error(t, results[0].Err, "exhausted retries surface as an error Result, not a panic")
	require.Equal(t, 5, client.callCount(), "all 5 attempts made")
}

func TestRegisterDegradesWhenHubUnresolvable(t *testing.T) {
	client := &fakeClient{}
	cfg := cfgWith(t, cliHealthSearchJSON, client)
	cfg.ResolveBaseURL = func(context.Context) (string, error) {
		return "", errors.New("scenario not running")
	}
	results := searchregister.Register(context.Background(), cfg)

	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	require.Zero(t, client.callCount(), "no RPC attempted when the hub URL cannot be resolved")
}

func TestRegisterMalformedFileIsNonFatal(t *testing.T) {
	client := &fakeClient{}
	cfg := cfgWith(t, `{ not valid json `, client)
	results := searchregister.Register(context.Background(), cfg)

	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	require.Zero(t, client.callCount())
}

// TestRegisterMirrorsCorpus is the corpusStoreMirrorsFile boot half: after the
// descriptor registers, the provider's tests block is mirrored to the eval store
// as "<provider_id>.primary", losslessly converted from the file.
func TestRegisterMirrorsCorpus(t *testing.T) {
	client := &fakeClient{created: true}
	evalClient := &fakeEvalClient{}
	results := searchregister.Register(context.Background(), cfgWithEval(t, cliHealthSearchJSON, client, evalClient))

	require.Len(t, results, 1)
	require.NoError(t, results[0].Err)
	require.True(t, results[0].CorpusRegistered)
	require.NoError(t, results[0].CorpusErr)
	require.Equal(t, 1, evalClient.callCount(), "exactly one suite mirrored")
	suite := evalClient.suites[0]
	require.Equal(t, "cli-health.commands.primary", suite.GetSuiteId())
	require.Equal(t, "cli-health.commands", suite.GetProviderId())
	require.Len(t, suite.GetCases(), 1)
	require.Equal(t, "x", suite.GetCases()[0].GetCaseId())
}

// TestRegisterCorpusDegradesGracefully: a failed eval mirror must NOT fail the
// provider's descriptor registration — it is logged on CorpusErr and the scenario
// keeps serving (it re-registers next boot).
func TestRegisterCorpusDegradesGracefully(t *testing.T) {
	client := &fakeClient{created: true}
	evalClient := &fakeEvalClient{alwaysErr: errors.New("eval store down")}
	results := searchregister.Register(context.Background(), cfgWithEval(t, cliHealthSearchJSON, client, evalClient))

	require.Len(t, results, 1)
	require.NoError(t, results[0].Err, "descriptor registration is independent of the corpus mirror")
	require.False(t, results[0].CorpusRegistered)
	require.Error(t, results[0].CorpusErr)
	require.Equal(t, 5, evalClient.callCount(), "all retry attempts made before degrading")
}

// TestRegisterSkipsEmptyCorpus: a provider with no cases never calls the eval
// store (nothing to mirror) and reports CorpusRegistered=false without an error.
func TestRegisterSkipsEmptyCorpus(t *testing.T) {
	const noCorpus = `{
  "version": "1.0.0",
  "providers": [{
    "provider_id": "demo.commands",
    "bucket": "BUCKET_DO",
    "type": "command",
    "scope": "SCOPE_PROJECT",
    "endpoint": {"http_json": {"path": "/x"}},
    "result_mapping": {"id_field": "name"},
    "tuning": {"engine": "dense"},
    "tests": {"cases": []}
  }]
}`
	client := &fakeClient{created: true}
	evalClient := &fakeEvalClient{}
	results := searchregister.Register(context.Background(), cfgWithEval(t, noCorpus, client, evalClient))

	require.Len(t, results, 1)
	require.NoError(t, results[0].Err)
	require.False(t, results[0].CorpusRegistered)
	require.NoError(t, results[0].CorpusErr)
	require.Zero(t, evalClient.callCount(), "no corpus => eval store untouched")
}

// TestRegisterSkipsCorpusWhenDescriptorFails: if the descriptor never registers,
// the corpus is not mirrored (the suite FKs the provider).
func TestRegisterSkipsCorpusWhenDescriptorFails(t *testing.T) {
	client := &fakeClient{alwaysErr: errors.New("connection refused")}
	evalClient := &fakeEvalClient{}
	results := searchregister.Register(context.Background(), cfgWithEval(t, cliHealthSearchJSON, client, evalClient))

	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	require.Zero(t, evalClient.callCount(), "no descriptor => no corpus mirror")
}

// twoProviderSearchJSON declares two providers with DISTINCT corpora so a test can
// prove the descriptors[i] ↔ file.Providers[i] alignment Register relies on (a
// swapped index would mirror provider A's corpus under provider B).
const twoProviderSearchJSON = `{
  "version": "1.0.0",
  "providers": [
    {
      "provider_id": "demo.alpha",
      "bucket": "BUCKET_DO",
      "type": "command",
      "scope": "SCOPE_PROJECT",
      "endpoint": {"http_json": {"path": "/alpha"}},
      "result_mapping": {"id_field": "name"},
      "tuning": {"engine": "dense"},
      "tests": {"cases": [{"id": "a1", "query": "alpha query"}]}
    },
    {
      "provider_id": "demo.beta",
      "bucket": "BUCKET_KNOW",
      "type": "doc",
      "scope": "SCOPE_PROJECT",
      "endpoint": {"http_json": {"path": "/beta"}},
      "result_mapping": {"id_field": "path"},
      "tuning": {"engine": "hybrid"},
      "tests": {"cases": [{"id": "b1", "query": "beta query"}, {"id": "b2", "query": "beta query two"}]}
    }
  ]
}`

// TestRegisterMultipleProvidersAligned: every provider registers in declared
// order, and each provider's corpus mirrors under ITS OWN suite id — the
// load-bearing descriptors[i] ↔ file.Providers[i] alignment.
func TestRegisterMultipleProvidersAligned(t *testing.T) {
	client := &fakeClient{created: true}
	evalClient := &fakeEvalClient{}
	results := searchregister.Register(context.Background(), cfgWithEval(t, twoProviderSearchJSON, client, evalClient))

	require.Len(t, results, 2)
	require.NoError(t, results[0].Err)
	require.NoError(t, results[1].Err)
	require.Equal(t, "demo.alpha", results[0].ProviderID)
	require.Equal(t, "demo.beta", results[1].ProviderID)
	require.Equal(t, []string{"demo.alpha", "demo.beta"}, []string{
		client.calls[0].GetProviderId(), client.calls[1].GetProviderId(),
	}, "descriptors registered in declared order")

	require.Equal(t, 2, evalClient.callCount(), "one suite mirrored per provider")
	byProvider := map[string]*evalv1.EvalSuite{
		evalClient.suites[0].GetProviderId(): evalClient.suites[0],
		evalClient.suites[1].GetProviderId(): evalClient.suites[1],
	}
	// Alignment proof: alpha's suite carries alpha's single case, beta's its two.
	require.Len(t, byProvider["demo.alpha"].GetCases(), 1)
	require.Equal(t, "a1", byProvider["demo.alpha"].GetCases()[0].GetCaseId())
	require.Len(t, byProvider["demo.beta"].GetCases(), 2)
	require.Equal(t, "b1", byProvider["demo.beta"].GetCases()[0].GetCaseId())
}

// TestRegisterMultipleProvidersPartialFailure: one provider's descriptor failing
// does not block the others, and the failed provider's corpus is NOT mirrored
// (the suite FKs the provider) while the healthy provider's still is.
func TestRegisterMultipleProvidersPartialFailure(t *testing.T) {
	client := &fakeClient{created: true, failProvider: "demo.alpha"}
	evalClient := &fakeEvalClient{}
	results := searchregister.Register(context.Background(), cfgWithEval(t, twoProviderSearchJSON, client, evalClient))

	require.Len(t, results, 2)
	require.Error(t, results[0].Err, "demo.alpha rejected")
	require.False(t, results[0].CorpusRegistered, "failed descriptor => no corpus mirror")
	require.NoError(t, results[1].Err, "demo.beta independent of alpha")
	require.True(t, results[1].CorpusRegistered)

	require.Equal(t, 1, evalClient.callCount(), "only the healthy provider's corpus mirrored")
	require.Equal(t, "demo.beta", evalClient.suites[0].GetProviderId())
}
