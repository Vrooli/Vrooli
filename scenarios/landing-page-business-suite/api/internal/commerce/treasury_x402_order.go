package commerce

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
	"time"

	"github.com/vrooli/api-core/servicetoken"
)

const (
	treasuryX402VerifyPath     = "/api/v1/x402/authorizations/verify"
	TreasuryX402OrderPrincipal = "landing-page-business-suite-treasury-x402-order"
	TreasuryX402OrderAudience  = "scenario-authenticator:treasury-x402-order"
	TreasuryX402AuthorizeScope = "treasury:x402-authorize"
	TreasuryX402SettleScope    = "treasury:x402-settle"
	TreasuryX402CancelScope    = "treasury:x402-cancel"
)

var ErrTreasuryX402SettlementPending = errors.New("Treasury x402 settlement is not yet conclusive")

type TreasuryX402VerifyRequest struct {
	PriceID          string `json:"price_id"`
	ResourceID       string `json:"resource_id"`
	OrderID          string `json:"order_id"`
	JobID            string `json:"job_id"`
	PaymentSignature string `json:"payment_signature"`
}

type TreasuryX402Authorization struct {
	AdmissionID string `json:"admission_id"`
	Status      string `json:"status"`
	Mode        string `json:"mode"`
	Payer       string `json:"payer"`
	Network     string `json:"network"`
	PriceID     string `json:"price_id"`
	ResourceID  string `json:"resource_id"`
	OrderID     string `json:"order_id"`
	JobID       string `json:"job_id"`
	Detail      string `json:"detail,omitempty"`
}

type TreasuryX402SettleRequest struct {
	AdmissionID string `json:"-"`
	PriceID     string `json:"-"`
	Payer       string `json:"-"`
	Network     string `json:"-"`
	ResourceID  string `json:"resource_id"`
	OrderID     string `json:"order_id"`
	JobID       string `json:"job_id"`
}

type TreasuryX402Settlement struct {
	AdmissionID string `json:"admission_id"`
	PriceID     string `json:"price_id"`
	Status      string `json:"status"`
	Payer       string `json:"payer"`
	Mode        string `json:"mode"`
	Transaction string `json:"transaction_id"`
	Network     string `json:"network"`
	JobID       string `json:"job_id"`
	ResourceID  string `json:"resource_id"`
	OrderID     string `json:"order_id"`
	PostingID   string `json:"posting_id,omitempty"`
	ReceiptID   string `json:"receipt_id,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

type TreasuryX402CancelRequest struct {
	AdmissionID string `json:"-"`
	PriceID     string `json:"-"`
	Payer       string `json:"-"`
	Network     string `json:"-"`
	JobID       string `json:"job_id"`
	ResourceID  string `json:"resource_id"`
	OrderID     string `json:"order_id"`
}

type TreasuryX402Cancellation struct {
	AdmissionID string `json:"admission_id"`
	PriceID     string `json:"price_id"`
	Status      string `json:"status"`
	Payer       string `json:"payer"`
	Network     string `json:"network"`
	Mode        string `json:"mode"`
	JobID       string `json:"job_id"`
	ResourceID  string `json:"resource_id"`
	OrderID     string `json:"order_id"`
	Detail      string `json:"detail,omitempty"`
}

type TreasuryX402OrderClient interface {
	VerifyForResource(context.Context, TreasuryX402VerifyRequest) (TreasuryX402Authorization, error)
	SettleAuthorization(context.Context, TreasuryX402SettleRequest) (TreasuryX402Settlement, error)
	CancelAuthorization(context.Context, TreasuryX402CancelRequest) (TreasuryX402Cancellation, error)
}

// TreasuryX402ServiceOrderProvider adapts Treasury's deferred x402 flow to the
// LPBS job lifecycle: verify/authorize during Begin, then settle only after
// LPBS's delivery verifier passes. A failed verification never calls settle.
type TreasuryX402ServiceOrderProvider struct {
	client TreasuryX402OrderClient
}

func NewTreasuryX402ServiceOrderProvider(client TreasuryX402OrderClient) (*TreasuryX402ServiceOrderProvider, error) {
	if client == nil {
		return nil, errors.New("Treasury x402 order client is required")
	}
	return &TreasuryX402ServiceOrderProvider{client: client}, nil
}

func (p *TreasuryX402ServiceOrderProvider) Authorize(ctx context.Context, request ServiceOrderAuthorizationRequest) (ServiceOrderAuthorizationReceipt, error) {
	if request.Door != ServiceOrderDoorAgentRun {
		return ServiceOrderAuthorizationReceipt{}, errors.New("Treasury x402 provider only accepts the Agent Manager x402 door")
	}
	result, err := p.client.VerifyForResource(ctx, TreasuryX402VerifyRequest{
		PriceID: request.PriceID, ResourceID: request.ResourceID, OrderID: request.OrderID,
		JobID: request.JobID, PaymentSignature: request.PaymentSignature,
	})
	if err != nil {
		return ServiceOrderAuthorizationReceipt{}, err
	}
	if result.Status != "authorized" || result.Mode != "local-fork" || result.AdmissionID == "" || result.Payer == "" || result.Network == "" || result.PriceID != request.PriceID || result.ResourceID != request.ResourceID || result.OrderID != request.OrderID || result.JobID != request.JobID {
		return ServiceOrderAuthorizationReceipt{}, errors.New("Treasury did not confirm the matching local-fork authorization")
	}
	return ServiceOrderAuthorizationReceipt{
		EventID:         "treasury-x402-authorization:" + result.AdmissionID,
		AuthorizationID: result.AdmissionID, Payer: result.Payer, Network: result.Network, Sandbox: true, Approved: true,
	}, nil
}

func (p *TreasuryX402ServiceOrderProvider) Capture(ctx context.Context, request ServiceOrderCaptureRequest) (ServiceOrderCaptureReceipt, error) {
	if request.Door != ServiceOrderDoorAgentRun {
		return ServiceOrderCaptureReceipt{}, errors.New("Treasury x402 provider only accepts the Agent Manager x402 door")
	}
	result, err := p.client.SettleAuthorization(ctx, TreasuryX402SettleRequest{
		AdmissionID: request.AuthorizationID, PriceID: request.PriceID, Payer: request.Payer, Network: request.Network, ResourceID: serviceOrderResourceIDFromRequest(request),
		OrderID: request.OrderID, JobID: request.JobID,
	})
	if err != nil {
		return ServiceOrderCaptureReceipt{}, err
	}
	if result.Status != "settled" || result.Mode != "local-fork" || result.AdmissionID != request.AuthorizationID || result.PriceID != request.PriceID || result.Payer != request.Payer || result.Network != request.Network || result.JobID != request.JobID || result.ResourceID != serviceOrderResourceIDFromRequest(request) || result.OrderID != request.OrderID || result.Transaction == "" {
		return ServiceOrderCaptureReceipt{}, errors.New("Treasury settlement did not confirm the matching local-fork authorization and price")
	}
	return ServiceOrderCaptureReceipt{
		EventID:   "treasury-x402-settlement:" + result.Transaction,
		CaptureID: result.Transaction, TreasuryExternalID: "treasury-x402-inflow:" + result.Transaction,
		Sandbox: true, Captured: true,
	}, nil
}

func (p *TreasuryX402ServiceOrderProvider) Void(ctx context.Context, request ServiceOrderVoidRequest) (ServiceOrderVoidReceipt, error) {
	if request.Door != ServiceOrderDoorAgentRun {
		return ServiceOrderVoidReceipt{}, errors.New("Treasury x402 provider only accepts the Agent Manager x402 door")
	}
	cancelled, err := p.client.CancelAuthorization(ctx, TreasuryX402CancelRequest{
		AdmissionID: request.AuthorizationID, PriceID: request.PriceID, Payer: request.Payer, Network: request.Network, JobID: request.JobID,
		ResourceID: request.ResourceID, OrderID: request.OrderID,
	})
	if err != nil {
		return ServiceOrderVoidReceipt{}, err
	}
	if cancelled.AdmissionID != request.AuthorizationID || cancelled.Status != "cancelled" || cancelled.Mode != "local-fork" || cancelled.PriceID != request.PriceID || cancelled.Payer != request.Payer || cancelled.Network != request.Network || cancelled.JobID != request.JobID || cancelled.ResourceID != request.ResourceID || cancelled.OrderID != request.OrderID {
		return ServiceOrderVoidReceipt{}, errors.New("Treasury did not confirm cancellation of the matching local-fork authorization")
	}
	return ServiceOrderVoidReceipt{
		EventID: "treasury-x402-cancelled-authorization:" + request.AuthorizationID,
		VoidID:  "cancelled:" + request.AuthorizationID, Sandbox: true, Voided: true,
	}, nil
}

func serviceOrderResourceIDFromRequest(request ServiceOrderCaptureRequest) string {
	// Resource identity is derived from the same immutable offer snapshot that
	// created the order; it is not caller-controlled on the capture retry.
	return request.ResourceID
}

// NewTreasuryX402OrderTokens exchanges one separately scoped order principal.
// The receipt-reader principal is intentionally independent and read-only.
func NewTreasuryX402OrderTokens(credentials servicetoken.CredentialSource, authenticatorURL string, client *http.Client, options ...servicetoken.Option) *servicetoken.Tokens {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	} else {
		clone := *client
		clone.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		client = &clone
	}
	exchange := servicetoken.AuthenticatorExchanger(client, func(context.Context) (string, error) {
		if strings.TrimSpace(authenticatorURL) == "" {
			return "", errors.New("scenario-authenticator URL is unavailable")
		}
		return strings.TrimRight(strings.TrimSpace(authenticatorURL), "/"), nil
	})
	base := []servicetoken.Option{servicetoken.WithExchanger(exchange)}
	base = append(base, options...)
	return servicetoken.New(servicetoken.Config{
		Name:       "LPBS Treasury x402 order client",
		Credential: credentials,
		Expect: servicetoken.ExpectIdentity(TreasuryX402OrderPrincipal, TreasuryX402OrderAudience,
			TreasuryX402AuthorizeScope, TreasuryX402SettleScope, TreasuryX402CancelScope),
		IssueHint: "the operator configures and issues the LPBS Treasury x402 order principal",
	}, base...)
}

type TreasuryX402HTTPClient struct {
	baseURL string
	client  *http.Client
	tokens  TreasuryX402ReceiptTokenSource
}

func NewTreasuryX402HTTPClient(baseURL string, tokens TreasuryX402ReceiptTokenSource, client *http.Client) (*TreasuryX402HTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("Treasury x402 order client requires an origin URL")
	}
	if parsed.Scheme != "http" || !isLoopbackHost(parsed.Hostname()) {
		return nil, errors.New("test-mode Treasury x402 order client requires loopback HTTP")
	}
	if tokens == nil {
		return nil, errors.New("Treasury x402 order client requires a service token source")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	} else {
		clone := *client
		client = &clone
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &TreasuryX402HTTPClient{baseURL: strings.TrimRight(parsed.String(), "/"), client: client, tokens: tokens}, nil
}

func (c *TreasuryX402HTTPClient) VerifyForResource(ctx context.Context, input TreasuryX402VerifyRequest) (TreasuryX402Authorization, error) {
	if strings.TrimSpace(input.PriceID) == "" || strings.TrimSpace(input.ResourceID) == "" || strings.TrimSpace(input.OrderID) == "" || strings.TrimSpace(input.JobID) == "" || strings.TrimSpace(input.PaymentSignature) == "" {
		return TreasuryX402Authorization{}, errors.New("Treasury x402 verification requires offer price, resource, order, job and payment signature")
	}
	var response TreasuryX402Authorization
	if err := c.post(ctx, treasuryX402VerifyPath, input, &response); err != nil {
		return TreasuryX402Authorization{}, fmt.Errorf("Treasury x402 verify-for-resource: %w", err)
	}
	if response.Status != "authorized" || response.Mode != "local-fork" || response.AdmissionID == "" || response.Payer == "" || response.Network == "" || response.PriceID != input.PriceID || response.ResourceID != input.ResourceID || response.OrderID != input.OrderID || response.JobID != input.JobID {
		return TreasuryX402Authorization{}, errors.New("Treasury returned an incomplete or differently bound x402 authorization")
	}
	return response, nil
}

func (c *TreasuryX402HTTPClient) SettleAuthorization(ctx context.Context, input TreasuryX402SettleRequest) (TreasuryX402Settlement, error) {
	if strings.TrimSpace(input.AdmissionID) == "" || strings.TrimSpace(input.PriceID) == "" || strings.TrimSpace(input.Payer) == "" || strings.TrimSpace(input.Network) == "" || strings.TrimSpace(input.ResourceID) == "" || strings.TrimSpace(input.OrderID) == "" || strings.TrimSpace(input.JobID) == "" {
		return TreasuryX402Settlement{}, errors.New("Treasury x402 settlement requires admission, price, payer, network, resource, order and job identities")
	}
	path := "/api/v1/x402/authorizations/" + url.PathEscape(input.AdmissionID) + "/settle"
	var response TreasuryX402Settlement
	if err := c.post(ctx, path, input, &response); err != nil {
		return TreasuryX402Settlement{}, fmt.Errorf("Treasury x402 settle-authorization: %w", err)
	}
	if response.AdmissionID != input.AdmissionID || response.PriceID != input.PriceID || response.Payer != input.Payer || response.Network != input.Network || response.JobID != input.JobID || response.ResourceID != input.ResourceID || response.OrderID != input.OrderID {
		return TreasuryX402Settlement{}, errors.New("Treasury settlement response is bound to another admission, job, resource or order")
	}
	if response.Mode != "local-fork" {
		return TreasuryX402Settlement{}, errors.New("Treasury did not confirm local-fork settlement mode")
	}
	switch response.Status {
	case "settled":
		if response.Transaction == "" || response.Network == "" {
			return TreasuryX402Settlement{}, errors.New("Treasury returned an incomplete settled x402 receipt")
		}
		return response, nil
	case "unknown", "in_progress", "calling", "settling", "recovering":
		return response, ErrTreasuryX402SettlementPending
	default:
		return response, fmt.Errorf("Treasury x402 settlement ended in non-success status %q", response.Status)
	}
}

func (c *TreasuryX402HTTPClient) CancelAuthorization(ctx context.Context, input TreasuryX402CancelRequest) (TreasuryX402Cancellation, error) {
	if strings.TrimSpace(input.AdmissionID) == "" || strings.TrimSpace(input.PriceID) == "" || strings.TrimSpace(input.Payer) == "" || strings.TrimSpace(input.Network) == "" || strings.TrimSpace(input.ResourceID) == "" || strings.TrimSpace(input.OrderID) == "" || strings.TrimSpace(input.JobID) == "" {
		return TreasuryX402Cancellation{}, errors.New("Treasury x402 cancellation requires admission, price, payer, network, resource, order and job identities")
	}
	path := "/api/v1/x402/authorizations/" + url.PathEscape(input.AdmissionID) + "/cancel"
	var response TreasuryX402Cancellation
	if err := c.post(ctx, path, input, &response); err != nil {
		return TreasuryX402Cancellation{}, fmt.Errorf("Treasury x402 cancel-authorization: %w", err)
	}
	if response.AdmissionID != input.AdmissionID || response.PriceID != input.PriceID || response.Payer != input.Payer || response.Network != input.Network || response.JobID != input.JobID || response.ResourceID != input.ResourceID || response.OrderID != input.OrderID {
		return TreasuryX402Cancellation{}, errors.New("Treasury cancellation response is bound to another admission, job, resource or order")
	}
	if response.Status != "cancelled" || response.Mode != "local-fork" {
		return TreasuryX402Cancellation{}, errors.New("Treasury did not confirm cancellation of a local-fork authorization")
	}
	return response, nil
}

func (c *TreasuryX402HTTPClient) post(ctx context.Context, path string, input any, output any) error {
	if c == nil || c.client == nil || c.tokens == nil {
		return errors.New("Treasury x402 order client is unavailable")
	}
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return fmt.Errorf("obtain Treasury x402 order service token: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("Treasury x402 order service token is empty")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Treasury x402 order endpoint returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("decode Treasury x402 order response: %w", err)
	}
	return nil
}
