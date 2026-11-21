// Package transport is the one place the CLI obtains its API transports: the
// cli-core REST client (base resolution, operator token, provenance headers)
// and the Connect HTTP client generated services are built over. Handlers
// never construct HTTP clients themselves.
package transport

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Transport carries the REST client, the Connect HTTP client and the root
// base URL generated Connect clients are constructed with.
type Transport struct {
	API     *cliutil.APIClient
	HTTP    connect.HTTPClient
	BaseURL string
	// Bounded returns a Connect HTTP client whose per-request deadline is
	// timeout. Long server-side blocks (operation wait) use it so the observer
	// never gives up before the owner answers.
	Bounded func(timeout time.Duration) connect.HTTPClient
}

// FromCore builds the transport over the scenario app: the same token source,
// base resolution and headers every cli-core command uses.
func FromCore(core *cliapp.ScenarioApp) Transport {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return Transport{
		API:     core.APIClient,
		HTTP:    httpClient,
		BaseURL: baseURL,
		Bounded: func(timeout time.Duration) connect.HTTPClient {
			client, _ := cliapp.NewConnectHTTPClientWithTimeout(core, timeout)
			return client
		},
	}
}

// ForBaseURL builds a transport against one server URL with no token. Tests
// use it to drive commands against fake servers.
func ForBaseURL(baseURL string) Transport {
	api := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: baseURL} },
		nil,
	)
	return Transport{
		API:     api,
		HTTP:    http.DefaultClient,
		BaseURL: baseURL,
		Bounded: func(timeout time.Duration) connect.HTTPClient {
			return &http.Client{Timeout: timeout}
		},
	}
}
