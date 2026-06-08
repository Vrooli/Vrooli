package searchregister_test

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/retry"
	searchregister "github.com/vrooli/searchregister-go"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// fakeClient records every RegisterProvider call and returns a scripted outcome.
// failFirst makes the first N attempts fail (to exercise the retry path) before
// succeeding; alwaysErr makes every attempt fail (to exercise graceful degrade).
type fakeClient struct {
	mu        sync.Mutex
	calls     []*registryv1.ProviderDescriptor
	failFirst int
	created   bool
	token     string
	alwaysErr error
}

func (f *fakeClient) RegisterProvider(
	_ context.Context,
	req *connect.Request[registryv1.RegisterProviderRequest],
) (*connect.Response[registryv1.RegisterProviderResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req.Msg.GetDescriptor_())
	if f.alwaysErr != nil {
		return nil, f.alwaysErr
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
	return searchregister.Config{
		ScenarioID:     "cli-health",
		SearchFilePath: writeSearchFile(t, raw),
		Logger:         testLogger(t),
		ResolveBaseURL: func(context.Context) (string, error) { return "http://search-hub.test", nil },
		NewClient:      func(string) searchregister.RegistryClient { return client },
		Retry: retry.Config{
			MaxAttempts:    5,
			Sleeper:        func(time.Duration) {},
			Rand:           func() float64 { return 0 },
			JitterFraction: 0,
		},
	}
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
