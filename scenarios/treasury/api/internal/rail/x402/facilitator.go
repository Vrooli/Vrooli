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

type HTTPFacilitator struct {
	client *http.Client
	base   *url.URL
}

func NewHTTPFacilitator(baseURL string, client *http.Client) (*HTTPFacilitator, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("%w: facilitator URL must be absolute and contain no credentials, query, or fragment", ErrInvalid)
	}
	if base.Scheme != "https" && !(base.Scheme == "http" && isLoopback(base.Hostname())) {
		return nil, fmt.Errorf("%w: facilitator URL must use HTTPS except on loopback", ErrInvalid)
	}
	if client == nil {
		return nil, fmt.Errorf("%w: HTTP client is required", ErrInvalid)
	}
	return &HTTPFacilitator{client: noRedirectClient(client), base: base}, nil
}

func (f *HTTPFacilitator) Verify(ctx context.Context, payload, requirements json.RawMessage) (VerifyResult, error) {
	response, err := f.call(ctx, "verify", payload, requirements)
	if err != nil {
		return VerifyResult{}, err
	}
	var value struct {
		Valid  bool   `json:"isValid"`
		Payer  string `json:"payer"`
		Reason string `json:"invalidReason"`
	}
	if err := json.Unmarshal(response, &value); err != nil {
		return VerifyResult{}, err
	}
	return VerifyResult{Valid: value.Valid, Payer: value.Payer, Reason: value.Reason}, nil
}

func (f *HTTPFacilitator) Settle(ctx context.Context, payload, requirements json.RawMessage) (SettleResult, error) {
	response, err := f.call(ctx, "settle", payload, requirements)
	if err != nil {
		return SettleResult{}, err
	}
	var value settleResponse
	if err := json.Unmarshal(response, &value); err != nil {
		return SettleResult{}, err
	}
	return SettleResult{Success: value.Success, Payer: value.Payer, Transaction: value.Transaction, Network: value.Network, Reason: value.ErrorReason}, nil
}

func (f *HTTPFacilitator) call(ctx context.Context, operation string, payload, requirements json.RawMessage) ([]byte, error) {
	var paymentPayload, paymentRequirements any
	if json.Unmarshal(payload, &paymentPayload) != nil || json.Unmarshal(requirements, &paymentRequirements) != nil {
		return nil, errors.New("invalid facilitator request JSON")
	}
	body, err := json.Marshal(map[string]any{"x402Version": 2, "paymentPayload": paymentPayload, "paymentRequirements": paymentRequirements})
	if err != nil {
		return nil, err
	}
	endpoint := *f.base
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + operation
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := f.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, errors.New("facilitator response exceeds limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("facilitator %s returned HTTP %d", operation, response.StatusCode)
	}
	return data, nil
}

var _ Facilitator = (*HTTPFacilitator)(nil)
