package infra

import (
	"net"
	"net/http"
	"time"
)

// NetworkDialer abstracts network connections for testing.
type NetworkDialer interface {
	// Dial connects to the address on the named network.
	Dial(network, address string) (net.Conn, error)
	// DialTimeout connects to the address with a timeout.
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
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

func (RealNetworkDialer) Dial(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func (RealNetworkDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
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
		return http.DefaultClient.Do(req)
	}
	return c.Client.Do(req)
}

// Ensure RealHTTPClient implements HTTPClient.
var _ HTTPClient = (*RealHTTPClient)(nil)
