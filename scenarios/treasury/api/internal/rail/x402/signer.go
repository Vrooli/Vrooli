package x402

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RPCSigner uses the standard eth_signTypedData_v4 wallet method. The signer
// URL may be remote only over HTTPS; loopback HTTP is allowed for a local
// operator wallet. Treasury never receives or stores a private key.
type RPCSigner struct{ client *http.Client }

func NewRPCSigner(client *http.Client) (*RPCSigner, error) {
	if client == nil {
		return nil, fmt.Errorf("%w: HTTP client is required", ErrInvalid)
	}
	return &RPCSigner{client: noRedirectClient(client)}, nil
}

func (s *RPCSigner) SignTypedData(ctx context.Context, credential Credential, typedData map[string]any) (string, error) {
	endpoint, err := url.Parse(strings.TrimSpace(credential.SignerURL))
	if err != nil || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.Fragment != "" {
		return "", errors.New("signer_url must be an absolute URL without userinfo or fragment")
	}
	if endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && isLoopback(endpoint.Hostname())) {
		return "", errors.New("signer_url must use HTTPS except on loopback")
	}
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "eth_signTypedData_v4", "params": []any{credential.Account, typedData}})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if authorization := strings.TrimSpace(credential.SignerAuthorization); authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("wallet signer returned HTTP %d", response.StatusCode)
	}
	var envelope struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes))
	if err := decoder.Decode(&envelope); err != nil {
		return "", err
	}
	if envelope.Error != nil {
		return "", fmt.Errorf("wallet signer refused typed data (%d): %s", envelope.Error.Code, envelope.Error.Message)
	}
	if !validSignature(envelope.Result) {
		return "", errors.New("wallet signer returned malformed signature")
	}
	return envelope.Result, nil
}

var _ TypedDataSigner = (*RPCSigner)(nil)
