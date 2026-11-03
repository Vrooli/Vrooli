package baseusdc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RPC is the narrow JSON-RPC surface used by the receipt verifier. It is easy
// to replace with deterministic fixtures and deliberately exposes no signing
// methods.
type RPC interface {
	Call(context.Context, string, []any, any) error
}

type HTTPRPC struct {
	endpoint string
	client   *http.Client
}

const DefaultRPCURL = "https://mainnet.base.org"

// NewHTTPRPC creates a read-only JSON-RPC transport. The URL must come from
// operator configuration, never from a receipt or other request field. API
// credentials embedded by a provider are never included in errors.
func NewHTTPRPC(endpoint string, client *http.Client) (*HTTPRPC, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("Base RPC endpoint must be an absolute HTTP(S) URL without user information or fragment")
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
			return nil, errors.New("unencrypted Base RPC is limited to loopback endpoints")
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	} else {
		copy := *client
		client = &copy
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("Base RPC redirects are refused")
	}
	return &HTTPRPC{endpoint: u.String(), client: client}, nil
}

func (r *HTTPRPC) Call(ctx context.Context, method string, params []any, out any) error {
	if r == nil || r.client == nil || r.endpoint == "" {
		return errors.New("Base RPC is unavailable")
	}
	switch method {
	case "eth_chainId", "eth_getTransactionReceipt", "eth_getBlockByNumber":
	default:
		return errors.New("Base RPC method is outside the receipt-verification allowlist")
	}
	requestBody, err := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
	}{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return fmt.Errorf("encode Base RPC request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return errors.New("create Base RPC request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("Base RPC transport failed for %s", method)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Base RPC returned HTTP %d for %s", resp.StatusCode, method)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20+1))
	if err != nil {
		return fmt.Errorf("read Base RPC response for %s", method)
	}
	if len(body) > 2<<20 {
		return errors.New("Base RPC response exceeds 2 MiB limit")
	}
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode Base RPC response for %s", method)
	}
	if envelope.JSONRPC != "2.0" || envelope.ID != 1 {
		return fmt.Errorf("Base RPC response identity mismatch for %s", method)
	}
	if envelope.Error != nil {
		return fmt.Errorf("Base RPC method %s failed with code %d", method, envelope.Error.Code)
	}
	if len(envelope.Result) == 0 {
		return fmt.Errorf("Base RPC method %s returned no result", method)
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("decode Base RPC result for %s", method)
	}
	return nil
}
