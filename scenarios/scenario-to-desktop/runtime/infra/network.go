package infra

import (
	"context"
	"net"
	"net/http"
)

// NetworkDialer abstracts network connections for testing.
type NetworkDialer interface {
	// DialContext connects with the caller's cancellation and deadline.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	// Listen creates a listener on the given address.
	Listen(network, address string) (net.Listener, error)
}

// HTTPClient abstracts HTTP client operations for testing.
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// RealNetworkDialer implements NetworkDialer using the net package.
type RealNetworkDialer struct{}

func (RealNetworkDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, network, address) // #nosec G704 -- callers validate destinations; context bounds the connection
}

func (RealNetworkDialer) Listen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

// Ensure RealNetworkDialer implements NetworkDialer.
var _ NetworkDialer = RealNetworkDialer{}

// RealHTTPClient implements HTTPClient using http.Client.
type RealHTTPClient struct {
	Client *http.Client
}

func (c *RealHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.Client == nil {
		return http.DefaultClient.Do(req) // #nosec G704 -- caller constructs the request under runtime policy
	}
	return c.Client.Do(req) // #nosec G704 -- injected client executes a caller-validated request
}

// Ensure RealHTTPClient implements HTTPClient.
var _ HTTPClient = (*RealHTTPClient)(nil)
