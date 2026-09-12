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
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	attachedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices"
	attachedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices/attached_devices_v1connect"
)

func NewBridgeAttachedReader(httpClient *http.Client, resolveURL func(context.Context, string) (string, error)) AttachedReader {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
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
		out = append(out, attachedFromProto(d))
	}
	return out, nil
}

func (r *lazyBridgeAttachedReader) Get(ctx context.Context, id string) (AttachedDevice, error) {
	base, err := r.resolveURL(ctx, "vrooli-bridge")
	if err != nil {
		return AttachedDevice{}, err
	}
	client := attachedconnect.NewAttachedDeviceServiceClient(r.httpClient, strings.TrimRight(base, "/"))
	resp, err := client.GetAttachedDevice(ctx, connect.NewRequest(&attachedv1.GetAttachedDeviceRequest{Id: id}))
	if err != nil {
		return AttachedDevice{}, fmt.Errorf("get bridge attached device: %w", err)
	}
	if resp == nil || resp.Msg.Device == nil {
		return AttachedDevice{}, fmt.Errorf("bridge returned no attached device")
	}
	return attachedFromProto(resp.Msg.Device), nil
}

func attachedFromProto(d *attachedv1.AttachedDevice) AttachedDevice {
	if d == nil {
		return AttachedDevice{}
	}
	return AttachedDevice{ID: d.Id, Name: d.Name, HostNodeID: d.HostNodeId, Kind: d.Kind, Transport: d.Transport, Serial: d.Serial, OSVersion: d.OsVersion, TrustState: d.TrustState, Reachability: d.Reachability, HealthReason: d.HealthReason}
}

// ownerSessionHTTPClient supplies Bridge's owner credential without putting a
// long-lived secret in device-control configuration. Enrolled local hosts use
// the short-lived LocalSession scheme; explicitly configured owner-token files
// remain available for remote/deployed control planes.
type ownerSessionHTTPClient struct {
	base connect.HTTPClient

	mu     sync.Mutex
	scheme string
	token  string
	// resolveCredential is injectable so transport behavior can be tested
	// without depending on the operator's on-disk enrollment.
	resolveCredential func(context.Context) (string, string, error)
}

func (c *ownerSessionHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c == nil || c.base == nil {
		return nil, fmt.Errorf("bridge owner-session transport is not configured")
	}
	scheme, token, err := c.ownerCredential(req.Context())
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", scheme+" "+token)
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
	c.scheme, c.token = "", ""
	c.mu.Unlock()
	refreshedScheme, refreshed, refreshErr := c.ownerCredential(req.Context())
	if refreshErr != nil {
		return nil, refreshErr
	}
	retry, cloneErr := replayRequest(req)
	if cloneErr != nil {
		return nil, cloneErr
	}
	retry.Header.Set("Authorization", refreshedScheme+" "+refreshed)
	return c.base.Do(retry)
}

func (c *ownerSessionHTTPClient) ownerCredential(ctx context.Context) (string, string, error) {
	c.mu.Lock()
	if strings.TrimSpace(c.scheme) != "" && strings.TrimSpace(c.token) != "" {
		scheme, token := c.scheme, c.token
		c.mu.Unlock()
		return scheme, token, nil
	}
	c.mu.Unlock()

	resolver := c.resolveCredential
	if resolver == nil {
		resolver = resolveBridgeOwnerCredential
	}
	scheme, token, err := resolver(ctx)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(scheme) == "" || strings.TrimSpace(token) == "" {
		return "", "", fmt.Errorf("Bridge owner session is unavailable")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Another request may have resolved a credential while this request was
	// doing I/O. Reuse the first successful result rather than replacing it.
	if strings.TrimSpace(c.scheme) != "" && strings.TrimSpace(c.token) != "" {
		return c.scheme, c.token, nil
	}
	c.scheme, c.token = scheme, token
	return scheme, token, nil
}

func resolveBridgeOwnerCredential(ctx context.Context) (string, string, error) {
	// Prefer the enrolled local operator session. It is short-lived and carries
	// Bridge's local authorization scheme without putting a private key or
	// long-lived bearer token in the scenario process.
	if store, err := sharedsession.DefaultFileStore(); err == nil {
		if resolution, resolveErr := (sharedsession.LocalResolver{Store: store}).Resolve(); resolveErr == nil && strings.TrimSpace(resolution.Token) != "" {
			return sharedsession.LocalSessionScheme, resolution.Token, nil
		}
	}
	// Explicit token files are the deployment fallback for a control plane that
	// cannot use a local enrollment. They remain bearer credentials and must be
	// issued for Bridge's configured audience.
	if token, err := readOwnerTokenFile(); err != nil {
		return "", "", err
	} else if token != "" {
		return "Bearer", token, nil
	}
	// Preserve the peer-credential fallback for hosts that have not enrolled an
	// operator session. Bridge deployments must configure the matching audience
	// for this normal bearer-token path.
	token, err := exchangeLocalOwnerToken(ctx)
	if err != nil {
		return "", "", fmt.Errorf("obtain Bridge owner session: %w", err)
	}
	return "Bearer", token, nil
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
	client := accountsconnect.NewAccountsServiceClient(&http.Client{Transport: transport, Timeout: 30 * time.Second}, "http://local-authenticator")
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
