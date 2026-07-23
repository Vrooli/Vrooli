package resources

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BrokerControlServer exposes the narrow broker authority surface over a
// loopback-only transport. Credentials identify an application scope; request
// bodies never get to choose that scope. This is intentionally separate from
// a resource endpoint, which is not proof of Vrooli ownership.
type BrokerControlServer struct {
	broker      *Broker
	credentials map[string]string // scope -> opaque per-scope credential
	issuers     map[string]CredentialIssuer
	issuerMu    sync.RWMutex
	server      *http.Server
	listener    net.Listener
}

type BrokerControlCredential struct {
	Endpoint string
	Scope    string
	Token    string
}

type acquireBrokerRequest struct {
	Resource   string `json:"resource"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

type authorizeUseBrokerRequest struct {
	LeaseID  string `json:"lease_id"`
	Resource string `json:"resource"`
}

type authorizeManagementBrokerRequest struct {
	InstanceID string `json:"instance_id"`
}

type issueCredentialBrokerRequest struct {
	LeaseID  string `json:"lease_id"`
	Resource string `json:"resource"`
}

// StartBrokerControlServer serves on a caller-provided loopback listener.
// Supplying the listener lets the embedding runtime retain ownership of port
// allocation and shutdown. Each credential must be unique and non-empty.
func StartBrokerControlServer(listener net.Listener, broker *Broker, credentials map[string]string) (*BrokerControlServer, error) {
	if listener == nil || broker == nil {
		return nil, fmt.Errorf("broker control listener and broker are required")
	}
	if !isLoopbackAddress(listener.Addr()) {
		return nil, fmt.Errorf("broker control transport must listen on loopback")
	}
	copyCredentials := make(map[string]string, len(credentials))
	seenTokens := map[string]bool{}
	for scope, token := range credentials {
		scope, token = strings.TrimSpace(scope), strings.TrimSpace(token)
		if scope == "" || token == "" || seenTokens[token] {
			return nil, fmt.Errorf("broker control credentials require unique non-empty scope tokens")
		}
		copyCredentials[scope] = token
		seenTokens[token] = true
	}
	if len(copyCredentials) == 0 {
		return nil, fmt.Errorf("broker control credentials are required")
	}
	control := &BrokerControlServer{broker: broker, credentials: copyCredentials, issuers: make(map[string]CredentialIssuer), listener: listener}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/acquire", control.acquire)
	mux.HandleFunc("/v1/authorize-use", control.authorizeUse)
	mux.HandleFunc("/v1/authorize-management", control.authorizeManagement)
	mux.HandleFunc("/v1/credentials", control.issueCredential)
	control.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = control.server.Serve(listener) }()
	return control, nil
}

// RegisterCredentialIssuer binds resource-native policy issuance to the
// broker's authenticated control surface. The issuer sees only an already
// verified instance and lease; it never receives a raw endpoint from a client.
func (s *BrokerControlServer) RegisterCredentialIssuer(resource string, issuer CredentialIssuer) error {
	resource = strings.TrimSpace(resource)
	if resource == "" || issuer == nil {
		return fmt.Errorf("resource and credential issuer are required")
	}
	s.issuerMu.Lock()
	defer s.issuerMu.Unlock()
	if _, exists := s.issuers[resource]; exists {
		return fmt.Errorf("credential issuer already registered for %s", resource)
	}
	s.issuers[resource] = issuer
	return nil
}

func (s *BrokerControlServer) Close(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *BrokerControlServer) scopeForRequest(w http.ResponseWriter, request *http.Request) (string, bool) {
	if request.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
	for scope, expected := range s.credentials {
		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1 {
			return scope, true
		}
	}
	http.Error(w, "broker credential is not authorized", http.StatusUnauthorized)
	return "", false
}

func (s *BrokerControlServer) acquire(w http.ResponseWriter, request *http.Request) {
	scope, ok := s.scopeForRequest(w, request)
	if !ok {
		return
	}
	var input acquireBrokerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, request.Body, 8<<10)).Decode(&input); err != nil {
		http.Error(w, "invalid acquire request", http.StatusBadRequest)
		return
	}
	lease, err := s.broker.Acquire(input.Resource, scope, time.Duration(input.TTLSeconds)*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeBrokerJSON(w, lease)
}

func (s *BrokerControlServer) authorizeUse(w http.ResponseWriter, request *http.Request) {
	scope, ok := s.scopeForRequest(w, request)
	if !ok {
		return
	}
	var input authorizeUseBrokerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, request.Body, 8<<10)).Decode(&input); err != nil {
		http.Error(w, "invalid use authorization request", http.StatusBadRequest)
		return
	}
	instance, err := s.broker.AuthorizeUse(input.LeaseID, input.Resource, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeBrokerJSON(w, instance)
}

func (s *BrokerControlServer) authorizeManagement(w http.ResponseWriter, request *http.Request) {
	scope, ok := s.scopeForRequest(w, request)
	if !ok {
		return
	}
	var input authorizeManagementBrokerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, request.Body, 8<<10)).Decode(&input); err != nil {
		http.Error(w, "invalid management authorization request", http.StatusBadRequest)
		return
	}
	instance, err := s.broker.AuthorizeManagement(input.InstanceID, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeBrokerJSON(w, instance)
}

func (s *BrokerControlServer) issueCredential(w http.ResponseWriter, request *http.Request) {
	scope, ok := s.scopeForRequest(w, request)
	if !ok {
		return
	}
	var input issueCredentialBrokerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, request.Body, 8<<10)).Decode(&input); err != nil {
		http.Error(w, "invalid credential request", http.StatusBadRequest)
		return
	}
	s.issuerMu.RLock()
	issuer := s.issuers[input.Resource]
	s.issuerMu.RUnlock()
	if issuer == nil {
		http.Error(w, "resource has no scoped credential issuer", http.StatusNotFound)
		return
	}
	credential, err := s.broker.IssueScopedCredential(input.LeaseID, input.Resource, scope, issuer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeBrokerJSON(w, credential)
}

func writeBrokerJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

// BrokerControlClient is deliberately scoped at construction. It cannot send
// a caller-selected application scope over the wire.
type BrokerControlClient struct {
	credential BrokerControlCredential
	httpClient *http.Client
}

func NewBrokerControlClient(credential BrokerControlCredential) (*BrokerControlClient, error) {
	if strings.TrimSpace(credential.Scope) == "" || strings.TrimSpace(credential.Token) == "" {
		return nil, fmt.Errorf("broker control scope and token are required")
	}
	endpoint, err := url.Parse(strings.TrimSpace(credential.Endpoint))
	if err != nil || endpoint.Scheme != "http" || !isLoopbackHost(endpoint.Hostname()) {
		return nil, fmt.Errorf("broker control endpoint must be an HTTP loopback address")
	}
	credential.Endpoint = strings.TrimRight(credential.Endpoint, "/")
	return &BrokerControlClient{credential: credential, httpClient: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *BrokerControlClient) Acquire(ctx context.Context, resource string, ttl time.Duration) (Lease, error) {
	var lease Lease
	if err := c.post(ctx, "/v1/acquire", acquireBrokerRequest{Resource: resource, TTLSeconds: int64(ttl / time.Second)}, &lease); err != nil {
		return Lease{}, err
	}
	return lease, nil
}

func (c *BrokerControlClient) AuthorizeUse(ctx context.Context, leaseID, resource string) (ManagedInstance, error) {
	var instance ManagedInstance
	if err := c.post(ctx, "/v1/authorize-use", authorizeUseBrokerRequest{LeaseID: leaseID, Resource: resource}, &instance); err != nil {
		return ManagedInstance{}, err
	}
	return instance, nil
}

func (c *BrokerControlClient) AuthorizeManagement(ctx context.Context, instanceID string) (ManagedInstance, error) {
	var instance ManagedInstance
	if err := c.post(ctx, "/v1/authorize-management", authorizeManagementBrokerRequest{InstanceID: instanceID}, &instance); err != nil {
		return ManagedInstance{}, err
	}
	return instance, nil
}

func (c *BrokerControlClient) IssueScopedCredential(ctx context.Context, leaseID, resource string) (ScopedCredential, error) {
	var credential ScopedCredential
	if err := c.post(ctx, "/v1/credentials", issueCredentialBrokerRequest{LeaseID: leaseID, Resource: resource}, &credential); err != nil {
		return ScopedCredential{}, err
	}
	return credential, nil
}

func (c *BrokerControlClient) post(ctx context.Context, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.credential.Endpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.credential.Token)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("broker control request failed: %s", response.Status)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 32<<10)).Decode(output)
}

func isLoopbackAddress(address net.Addr) bool {
	if tcp, ok := address.(*net.TCPAddr); ok {
		return tcp.IP.IsLoopback()
	}
	return false
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// BrokerControlEndpoint returns a client-safe loopback URL for a listener.
func BrokerControlEndpoint(listener net.Listener) (string, error) {
	if listener == nil || !isLoopbackAddress(listener.Addr()) {
		return "", fmt.Errorf("broker control listener must be loopback")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return "http://127.0.0.1:" + strconv.Itoa(port), nil
}
