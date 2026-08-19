// Package x402 implements the x402 v2 outbound rail and inbound payment gate.
package x402

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"treasury/internal/rail"
)

const (
	paymentRequiredHeader  = "Payment-Required"
	paymentSignatureHeader = "Payment-Signature"
	paymentResponseHeader  = "Payment-Response"
	maxResponseBytes       = 1 << 20
)

var ErrInvalid = errors.New("invalid x402 payment")

// Credential is resolved from Credential Authority only for the duration of a
// settlement. It binds the payment target to an operator-controlled signer and
// explicit network/asset allowlists; no field is persisted by Treasury.
type Credential struct {
	EndpointURL         string                 `json:"endpoint_url"`
	SignerURL           string                 `json:"signer_url"`
	SignerAuthorization string                 `json:"signer_authorization,omitempty"`
	Account             string                 `json:"account"`
	Networks            []string               `json:"networks"`
	Assets              map[string]AssetPolicy `json:"assets"`
}

type AssetPolicy struct {
	Decimals int    `json:"decimals"`
	Currency string `json:"currency"`
}

type paymentRequired struct {
	X402Version int               `json:"x402Version"`
	Resource    *resourceInfo     `json:"resource,omitempty"`
	Accepts     []json.RawMessage `json:"accepts"`
	Extensions  json.RawMessage   `json:"extensions,omitempty"`
}

type resourceInfo struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type requirement struct {
	Scheme            string          `json:"scheme"`
	Network           string          `json:"network"`
	Amount            string          `json:"amount"`
	PayTo             string          `json:"payTo"`
	MaxTimeoutSeconds uint64          `json:"maxTimeoutSeconds"`
	Asset             string          `json:"asset"`
	Extra             json.RawMessage `json:"extra"`
}

type transferMethod struct {
	AssetTransferMethod string `json:"assetTransferMethod"`
	Name                string `json:"name"`
	Version             string `json:"version"`
}

type settleResponse struct {
	Success     bool   `json:"success"`
	ErrorReason string `json:"errorReason,omitempty"`
	Payer       string `json:"payer,omitempty"`
	Transaction string `json:"transaction,omitempty"`
	Network     string `json:"network"`
}

// TypedDataSigner delegates signing to an operator-owned wallet. Implementations
// must not log the typed data authorization header or returned signature.
type TypedDataSigner interface {
	SignTypedData(context.Context, Credential, map[string]any) (string, error)
}

type Adapter struct {
	client *http.Client
	signer TypedDataSigner
	now    func() time.Time
}

func New(client *http.Client, signer TypedDataSigner) (*Adapter, error) {
	if client == nil || signer == nil {
		return nil, fmt.Errorf("%w: HTTP client and typed-data signer are required", ErrInvalid)
	}
	return &Adapter{client: noRedirectClient(client), signer: signer, now: time.Now}, nil
}

func (a *Adapter) Name() string { return "x402" }

func (a *Adapter) Settle(ctx context.Context, command rail.SettleCommand) (rail.Result, error) {
	if err := rail.ValidateSettle(command); err != nil {
		return rail.Result{}, err
	}
	credential, endpoint, err := parseCredential(command.Credential, command.Counterparty)
	if err != nil {
		return failed("credential_policy", err.Error()), nil
	}
	initial, err := a.request(ctx, endpoint, "", command.IdempotencyKey)
	if err != nil {
		return failed("x402_price_unavailable", "no payment was constructed: "+err.Error()), nil
	}
	defer initial.Body.Close()
	if initial.StatusCode != http.StatusPaymentRequired {
		return failed("x402_price_required", fmt.Sprintf("counterparty returned HTTP %d instead of 402", initial.StatusCode)), nil
	}
	required, err := decodePaymentRequired(initial)
	if err != nil {
		return failed("x402_price_invalid", err.Error()), nil
	}
	rawRequirement, accepted, decimals, err := selectRequirement(required, credential, endpoint, command.AmountMinor, command.Currency)
	if err != nil {
		return failed("mandate_per_call_cap", err.Error()), nil
	}
	payload, err := a.signPayload(ctx, credential, required, rawRequirement, accepted, command.IdempotencyKey)
	if err != nil {
		return failed("x402_signing_refused", "no payment was dispatched: "+err.Error()), nil
	}
	paid, err := a.request(ctx, endpoint, payload, command.IdempotencyKey)
	if err != nil {
		return unknown(payload, "paid request response unavailable: "+err.Error()), err
	}
	defer paid.Body.Close()
	if paid.StatusCode < 200 || paid.StatusCode >= 300 {
		if paid.StatusCode >= 500 {
			return unknown(payload, fmt.Sprintf("paid request returned HTTP %d", paid.StatusCode)), fmt.Errorf("paid request returned HTTP %d", paid.StatusCode)
		}
		return failed("x402_payment_rejected", fmt.Sprintf("paid request returned HTTP %d", paid.StatusCode)), nil
	}
	settled, err := decodeSettleResponse(paid.Header.Get(paymentResponseHeader))
	if err != nil {
		return unknown(payload, "paid response omitted a valid Payment-Response: "+err.Error()), err
	}
	if !settled.Success || settled.Transaction == "" || settled.Payer == "" || settled.Network != accepted.Network {
		return unknown(payload, "facilitator response did not contain a complete matching settlement"), fmt.Errorf("incomplete x402 settlement response")
	}
	return rail.Result{
		Outcome: rail.OutcomeSettled, ExternalID: settled.Transaction,
		ReceiptReference: paymentDigest(payload), Basis: "x402_facilitator_confirmation",
		OccurredAt: a.now().UTC(),
		Detail:     fmt.Sprintf("x402 v2 exact payment settled on %s (%d asset decimals)", settled.Network, decimals),
	}, nil
}

// QueryOutcome replays only the exact signed payload preserved by the first
// uncertain call. It never creates a new authorization or signs a second
// payment. A non-success remains unknown because an HTTP error cannot prove
// that the on-chain transfer did not occur.
func (a *Adapter) QueryOutcome(ctx context.Context, query rail.Query) (rail.Result, error) {
	credential, endpoint, err := parseCredential(query.Credential, query.Counterparty)
	if err != nil {
		return rail.Result{}, err
	}
	_ = credential
	payload := strings.TrimSpace(query.ReceiptReference)
	if payload == "" || paymentDigest(payload) != strings.TrimSpace(query.ExternalID) {
		return rail.Result{}, fmt.Errorf("%w: exact signed payload and matching digest are required", ErrInvalid)
	}
	response, err := a.request(ctx, endpoint, payload, query.IdempotencyKey)
	if err != nil {
		return rail.Result{Outcome: rail.OutcomeUnknown, ExternalID: query.ExternalID, ReceiptReference: payload, Basis: "x402_replay_inconclusive", Detail: err.Error()}, nil
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return rail.Result{Outcome: rail.OutcomeUnknown, ExternalID: query.ExternalID, ReceiptReference: payload, Basis: "x402_replay_inconclusive", Detail: fmt.Sprintf("paid replay returned HTTP %d", response.StatusCode)}, nil
	}
	settled, err := decodeSettleResponse(response.Header.Get(paymentResponseHeader))
	if err != nil || !settled.Success || settled.Transaction == "" {
		return rail.Result{Outcome: rail.OutcomeUnknown, ExternalID: query.ExternalID, ReceiptReference: payload, Basis: "x402_replay_inconclusive", Detail: "paid replay lacked a conclusive settlement receipt"}, nil
	}
	return rail.Result{Outcome: rail.OutcomeSettled, ExternalID: settled.Transaction, ReceiptReference: paymentDigest(payload), Basis: "x402_facilitator_replay_confirmation", OccurredAt: a.now().UTC(), Detail: "exact signed payment replay returned a settlement receipt"}, nil
}

func (a *Adapter) request(ctx context.Context, endpoint, payment, idempotencyKey string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Treasury-Idempotency-Key", idempotencyKey)
	if payment != "" {
		request.Header.Set(paymentSignatureHeader, payment)
	}
	return a.client.Do(request)
}

func parseCredential(raw, counterparty string) (Credential, string, error) {
	var credential Credential
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&credential); err != nil {
		return Credential{}, "", fmt.Errorf("credential value must be strict JSON: %w", err)
	}
	endpoint, err := url.Parse(strings.TrimSpace(credential.EndpointURL))
	if err != nil || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.Fragment != "" {
		return Credential{}, "", errors.New("endpoint_url must be an absolute URL without userinfo or fragment")
	}
	if endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && isLoopback(endpoint.Hostname())) {
		return Credential{}, "", errors.New("endpoint_url must use HTTPS except on loopback")
	}
	if !strings.EqualFold(endpoint.Hostname(), strings.TrimSpace(counterparty)) {
		return Credential{}, "", errors.New("endpoint_url host does not match the mandate counterparty")
	}
	if strings.TrimSpace(credential.SignerURL) == "" || !isEVMAddress(credential.Account) || len(credential.Networks) == 0 || len(credential.Assets) == 0 {
		return Credential{}, "", errors.New("signer_url, account, networks, and assets are required")
	}
	return credential, endpoint.String(), nil
}

func isLoopback(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func decodePaymentRequired(response *http.Response) (paymentRequired, error) {
	header := strings.TrimSpace(response.Header.Get(paymentRequiredHeader))
	if header == "" {
		return paymentRequired{}, errors.New("Payment-Required header is missing")
	}
	decoded, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return paymentRequired{}, fmt.Errorf("decode Payment-Required: %w", err)
	}
	var required paymentRequired
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(decoded), maxResponseBytes))
	if err := decoder.Decode(&required); err != nil {
		return paymentRequired{}, fmt.Errorf("parse Payment-Required: %w", err)
	}
	if required.X402Version != 2 || len(required.Accepts) == 0 {
		return paymentRequired{}, errors.New("x402 v2 with at least one accepted requirement is required")
	}
	return required, nil
}

func selectRequirement(required paymentRequired, credential Credential, endpoint string, capMinor int64, currency string) (json.RawMessage, requirement, int, error) {
	if required.Resource == nil || required.Resource.URL != endpoint {
		return nil, requirement{}, 0, errors.New("priced resource URL does not exactly match endpoint_url")
	}
	allowedNetworks := stringSet(credential.Networks)
	for _, raw := range required.Accepts {
		var candidate requirement
		if err := json.Unmarshal(raw, &candidate); err != nil || candidate.Scheme != "exact" || !strings.HasPrefix(candidate.Network, "eip155:") || !allowedNetworks[strings.ToLower(candidate.Network)] {
			continue
		}
		policy, ok := credential.Assets[strings.ToLower(candidate.Asset)]
		decimals := policy.Decimals
		if !ok || decimals < 2 || decimals > 30 || !strings.EqualFold(policy.Currency, currency) || !isEVMAddress(candidate.PayTo) || !isEVMAddress(candidate.Asset) || candidate.MaxTimeoutSeconds == 0 {
			continue
		}
		var method transferMethod
		if json.Unmarshal(candidate.Extra, &method) != nil || method.AssetTransferMethod != "eip3009" || method.Name == "" || method.Version == "" {
			continue
		}
		amount, ok := new(big.Int).SetString(candidate.Amount, 10)
		if !ok || amount.Sign() <= 0 {
			continue
		}
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals-2)), nil)
		cap := new(big.Int).Mul(big.NewInt(capMinor), multiplier)
		if amount.Cmp(cap) != 0 {
			return nil, requirement{}, 0, fmt.Errorf("quoted asset amount %s does not equal the authorized per-call amount %d minor units", amount, capMinor)
		}
		return raw, candidate, decimals, nil
	}
	return nil, requirement{}, 0, errors.New("no accepted requirement matched the governed network, asset, scheme, and transfer method")
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[strings.ToLower(strings.TrimSpace(value))] = true
	}
	return set
}

func (a *Adapter) signPayload(ctx context.Context, credential Credential, required paymentRequired, raw json.RawMessage, accepted requirement, idempotencyKey string) (string, error) {
	chainID, err := strconv.ParseUint(strings.TrimPrefix(accepted.Network, "eip155:"), 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid EIP-155 network: %w", err)
	}
	var method transferMethod
	if err := json.Unmarshal(accepted.Extra, &method); err != nil {
		return "", err
	}
	now := a.now().UTC()
	validAfter := now.Add(-10 * time.Minute).Unix()
	validBefore := now.Add(time.Duration(accepted.MaxTimeoutSeconds) * time.Second).Unix()
	nonceInput := append([]byte(credential.Account+"\x00"+idempotencyKey+"\x00"), raw...)
	nonce := sha256.Sum256(nonceInput)
	authorization := map[string]any{
		"from": credential.Account, "to": accepted.PayTo, "value": accepted.Amount,
		"validAfter": strconv.FormatInt(validAfter, 10), "validBefore": strconv.FormatInt(validBefore, 10),
		"nonce": "0x" + hex.EncodeToString(nonce[:]),
	}
	typedData := map[string]any{
		"types": map[string]any{
			"EIP712Domain":              []map[string]string{{"name": "name", "type": "string"}, {"name": "version", "type": "string"}, {"name": "chainId", "type": "uint256"}, {"name": "verifyingContract", "type": "address"}},
			"TransferWithAuthorization": []map[string]string{{"name": "from", "type": "address"}, {"name": "to", "type": "address"}, {"name": "value", "type": "uint256"}, {"name": "validAfter", "type": "uint256"}, {"name": "validBefore", "type": "uint256"}, {"name": "nonce", "type": "bytes32"}},
		},
		"primaryType": "TransferWithAuthorization",
		"domain":      map[string]any{"name": method.Name, "version": method.Version, "chainId": strconv.FormatUint(chainID, 10), "verifyingContract": accepted.Asset},
		"message":     authorization,
	}
	signature, err := a.signer.SignTypedData(ctx, credential, typedData)
	if err != nil {
		return "", err
	}
	if !validSignature(signature) {
		return "", errors.New("signer returned a malformed 65-byte EVM signature")
	}
	var acceptedJSON any
	if err := json.Unmarshal(raw, &acceptedJSON); err != nil {
		return "", err
	}
	var extensions any = map[string]any{}
	if len(required.Extensions) > 0 && string(required.Extensions) != "null" {
		if err := json.Unmarshal(required.Extensions, &extensions); err != nil {
			return "", fmt.Errorf("parse payment extensions: %w", err)
		}
	}
	payload := map[string]any{
		"x402Version": 2, "accepted": acceptedJSON, "resource": required.Resource,
		"payload":    map[string]any{"signature": signature, "authorization": authorization},
		"extensions": extensions,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}

func validSignature(value string) bool {
	value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 65
}

func isEVMAddress(value string) bool {
	value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 20
}

func noRedirectClient(client *http.Client) *http.Client {
	copy := *client
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &copy
}

func decodeSettleResponse(header string) (settleResponse, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header))
	if err != nil {
		return settleResponse{}, fmt.Errorf("decode Payment-Response: %w", err)
	}
	var response settleResponse
	if err := json.Unmarshal(decoded, &response); err != nil {
		return settleResponse{}, fmt.Errorf("parse Payment-Response: %w", err)
	}
	return response, nil
}

func paymentDigest(payload string) string {
	digest := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func failed(basis, detail string) rail.Result {
	return rail.Result{Outcome: rail.OutcomeFailed, Basis: basis, Detail: detail}
}

func unknown(payload, detail string) rail.Result {
	return rail.Result{Outcome: rail.OutcomeUnknown, ExternalID: paymentDigest(payload), ReceiptReference: payload, Basis: "x402_payment_dispatched", Detail: detail}
}

var _ rail.Adapter = (*Adapter)(nil)
