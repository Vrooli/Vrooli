package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// BrokerControlCredential identifies one desktop application's existing
// control-plane scope. The scope is carried by the authenticated credential,
// never by a shared-service request body.
type BrokerControlCredential struct {
	Endpoint string
	Scope    string
	Token    string
}

// BrokerControlGrantClient implements SharedBrokerClient against the Vrooli
// loopback broker protocol. It has use authority only: the protocol exposes no
// start, stop, configuration, or management operation to desktop bundles.
type BrokerControlGrantClient struct {
	endpoint string
	token    string
	client   *http.Client
}

func NewBrokerControlGrantClient(credential BrokerControlCredential) (*BrokerControlGrantClient, error) {
	endpoint, err := url.Parse(strings.TrimSpace(credential.Endpoint))
	if err != nil || endpoint.Scheme != "http" || !isLoopbackBrokerHost(endpoint.Hostname()) {
		return nil, fmt.Errorf("broker control endpoint must be an HTTP loopback address")
	}
	if strings.TrimSpace(credential.Scope) == "" || strings.TrimSpace(credential.Token) == "" {
		return nil, fmt.Errorf("broker control scope and token are required")
	}
	return &BrokerControlGrantClient{
		endpoint: strings.TrimRight(credential.Endpoint, "/"),
		token:    credential.Token,
		client:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *BrokerControlGrantClient) GrantSharedService(ctx context.Context, resource string, ttl time.Duration) (SharedBrokerGrant, error) {
	if c == nil || c.client == nil {
		return SharedBrokerGrant{}, fmt.Errorf("broker control client is required")
	}
	resource = strings.TrimSpace(resource)
	if resource == "" || ttl <= 0 {
		return SharedBrokerGrant{}, fmt.Errorf("resource and positive lease TTL are required")
	}
	var lease brokerLease
	if err := c.post(ctx, "/v1/acquire", struct {
		Resource   string `json:"resource"`
		TTLSeconds int64  `json:"ttl_seconds"`
	}{Resource: resource, TTLSeconds: int64(ttl / time.Second)}, &lease); err != nil {
		return SharedBrokerGrant{}, fmt.Errorf("acquire shared %s lease: %w", resource, err)
	}
	var instance brokerInstance
	if err := c.post(ctx, "/v1/authorize-use", struct {
		LeaseID  string `json:"lease_id"`
		Resource string `json:"resource"`
	}{LeaseID: lease.ID, Resource: resource}, &instance); err != nil {
		return SharedBrokerGrant{}, fmt.Errorf("authorize shared %s use: %w", resource, err)
	}
	if instance.Resource != resource || instance.Provider != "managed-shared" || !isLoopbackManagedServiceEndpoint(instance.Endpoint) {
		return SharedBrokerGrant{}, fmt.Errorf("broker returned an incompatible shared %s instance", resource)
	}
	var credential brokerScopedCredential
	if err := c.post(ctx, "/v1/credentials", struct {
		LeaseID  string `json:"lease_id"`
		Resource string `json:"resource"`
	}{LeaseID: lease.ID, Resource: resource}, &credential); err != nil {
		return SharedBrokerGrant{}, fmt.Errorf("issue scoped %s credential: %w", resource, err)
	}
	if credential.Resource != resource || credential.LeaseID != lease.ID || strings.TrimSpace(credential.Credential) == "" || credential.ExpiresAt.IsZero() {
		return SharedBrokerGrant{}, fmt.Errorf("broker returned an invalid scoped %s credential", resource)
	}
	return SharedBrokerGrant{Endpoint: instance.Endpoint, Credential: credential.Credential, ExpiresAt: credential.ExpiresAt}, nil
}

func (c *BrokerControlGrantClient) post(ctx context.Context, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("broker control request failed: %s", response.Status)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 32<<10)).Decode(output)
}

type brokerLease struct {
	ID string
}

type brokerInstance struct {
	Resource string
	Provider string
	Endpoint string
}

type brokerScopedCredential struct {
	LeaseID    string
	Resource   string
	ExpiresAt  time.Time
	Credential string
}

func isLoopbackBrokerHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isLoopbackManagedServiceEndpoint(raw string) bool {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && (endpoint.Scheme == "http" || endpoint.Scheme == "https") && isLoopbackBrokerHost(endpoint.Hostname()) && endpoint.RawQuery == "" && endpoint.Fragment == ""
}

// BrokerControlEndpoint builds the client-safe URL for a loopback listener.
// It is useful to the desktop host that creates the local broker service.
func BrokerControlEndpoint(listener net.Listener) (string, error) {
	if listener == nil {
		return "", fmt.Errorf("broker control listener is required")
	}
	tcp, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !tcp.IP.IsLoopback() {
		return "", fmt.Errorf("broker control listener must be loopback")
	}
	return "http://127.0.0.1:" + strconv.Itoa(tcp.Port), nil
}
