package control

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	attachedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices"
	attachedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices/attached_devices_v1connect"
)

func NewBridgeAttachedReader(httpClient *http.Client, resolveURL func(context.Context, string) (string, error)) AttachedReader {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if resolveURL == nil {
		resolveURL = discovery.ResolveScenarioURLDefault
	}
	return &lazyBridgeAttachedReader{httpClient: &ownerSessionHTTPClient{base: httpClient}, resolveURL: resolveURL}
}

type lazyBridgeAttachedReader struct {
	httpClient connect.HTTPClient
	resolveURL func(context.Context, string) (string, error)
}

func (r *lazyBridgeAttachedReader) List(ctx context.Context) ([]AttachedDevice, error) {
	base, err := r.resolveURL(ctx, "vrooli-bridge")
	if err != nil {
		return nil, err
	}
	client := attachedconnect.NewAttachedDeviceServiceClient(r.httpClient, strings.TrimRight(base, "/"))
	resp, err := client.ListAttachedDevices(ctx, connect.NewRequest(&attachedv1.ListAttachedDevicesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list bridge attached devices: %w", err)
	}
	out := make([]AttachedDevice, 0, len(resp.Msg.Devices))
	for _, d := range resp.Msg.Devices {
		out = append(out, AttachedDevice{ID: d.Id, Name: d.Name, HostNodeID: d.HostNodeId, Kind: d.Kind, Transport: d.Transport, Serial: d.Serial, OSVersion: d.OsVersion, TrustState: d.TrustState, Reachability: d.Reachability, HealthReason: d.HealthReason})
	}
	return out, nil
}

// ownerSessionHTTPClient supplies Bridge's owner bearer credential without
// putting a long-lived secret in device-control configuration. Local hosts
// use the authenticator's peer-credential exchange; explicitly configured
// owner-token files remain available for remote/deployed control planes.
type ownerSessionHTTPClient struct {
	base connect.HTTPClient

	mu    sync.Mutex
	token string
}

func (c *ownerSessionHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c == nil || c.base == nil {
		return nil, fmt.Errorf("bridge owner-session transport is not configured")
	}
	token, err := c.ownerToken(req.Context())
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.base.Do(req)
	if err != nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
	if req.GetBody == nil {
		return resp, err
	}
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
	refreshed, refreshErr := c.ownerToken(req.Context())
	if refreshErr != nil {
		return nil, refreshErr
	}
	retry, cloneErr := replayRequest(req)
	if cloneErr != nil {
		return nil, cloneErr
	}
	retry.Header.Set("Authorization", "Bearer "+refreshed)
	return c.base.Do(retry)
}

func (c *ownerSessionHTTPClient) ownerToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if strings.TrimSpace(c.token) != "" {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.TrimSpace(c.token) != "" {
		return c.token, nil
	}
	if token, err := readOwnerTokenFile(); err != nil {
		return "", err
	} else if token != "" {
		c.token = token
		return token, nil
	}
	token, err := exchangeLocalOwnerToken(ctx)
	if err != nil {
		return "", fmt.Errorf("obtain Bridge owner session: %w", err)
	}
	c.token = token
	return token, nil
}

func readOwnerTokenFile() (string, error) {
	path := strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_TOKEN_FILE"))
	if path == "" {
		path = strings.TrimSpace(os.Getenv("VROOLI_AUTH_TOKEN_FILE"))
	}
	if path == "" {
		return "", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat owner token file: %w", err)
	}
	if info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("owner token file must be owner-only: %s", filepath.Base(path))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read owner token file: %w", err)
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", fmt.Errorf("owner token file is empty")
	}
	return token, nil
}

func exchangeLocalOwnerToken(ctx context.Context) (string, error) {
	socketPath := strings.TrimSpace(os.Getenv("VROOLI_AUTH_SOCKET"))
	if socketPath == "" {
		socketPath = filepath.Join(os.TempDir(), "vrooli-scenario-authenticator-scenario-authenticator.sock")
	}
	machineID, err := os.Hostname()
	if err != nil || strings.TrimSpace(machineID) == "" {
		return "", fmt.Errorf("resolve machine id: %w", err)
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	defer transport.CloseIdleConnections()
	client := accountsconnect.NewAccountsServiceClient(&http.Client{Transport: transport}, "http://local-authenticator")
	resp, err := client.ExchangeMachinePrincipal(ctx, connect.NewRequest(&accountsv1.ExchangeMachinePrincipalRequest{MachineId: machineID}))
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Tokens == nil || strings.TrimSpace(resp.Msg.Tokens.AccessToken) == "" {
		return "", fmt.Errorf("local exchange returned no access token")
	}
	return resp.Msg.Tokens.AccessToken, nil
}

func replayRequest(req *http.Request) (*http.Request, error) {
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	retry := req.Clone(req.Context())
	retry.Body = body
	return retry, nil
}
