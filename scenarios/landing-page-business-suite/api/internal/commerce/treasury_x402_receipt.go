package commerce

import (
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
	treasuryX402ReceiptPath      = "/api/v1/x402/receipts"
	TreasuryX402ReceiptPrincipal = "landing-page-business-suite-treasury-x402-receipt-reader"
	TreasuryX402ReceiptAudience  = "scenario-authenticator:treasury-x402-receipt"
	TreasuryX402ReceiptScope     = "treasury:x402-receipt-read"
)

var ErrTreasuryX402ReceiptNotFound = errors.New("Treasury has no accepted x402 receipt for this external id")

type TreasuryX402Receipt struct {
	ExternalID  string    `json:"external_id"`
	PostingID   string    `json:"posting_id"`
	ReceiptID   string    `json:"receipt_id"`
	AcceptedAt  time.Time `json:"accepted_at"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Basis       string    `json:"basis"`
}

type TreasuryX402ReceiptTokenSource interface {
	Token(context.Context) (string, error)
}

// NewTreasuryX402ReceiptTokens exchanges the operator-issued LPBS receipt-reader
// credential for the exact Treasury audience and scope declared in the
// scenario's service-principals.json. It never mints or writes credentials.
func NewTreasuryX402ReceiptTokens(credentials servicetoken.CredentialSource, authenticatorURL string, client *http.Client, options ...servicetoken.Option) *servicetoken.Tokens {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
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
		Name:       "LPBS Treasury x402 receipt reader",
		Credential: credentials,
		Expect:     servicetoken.ExpectIdentity(TreasuryX402ReceiptPrincipal, TreasuryX402ReceiptAudience, TreasuryX402ReceiptScope),
		IssueHint:  "the operator configures and issues the LPBS Treasury receipt-reader principal",
	}, base...)
}

// TreasuryX402ReceiptReader reads the one accepted seller-side Money Ledger
// posting Treasury owns for an inbound x402 admission. It is a read-only link:
// LPBS must not emit a second seller-side posting for the same sale.
type TreasuryX402ReceiptReader struct {
	baseURL string
	client  *http.Client
	tokens  TreasuryX402ReceiptTokenSource
}

func NewTreasuryX402ReceiptReader(baseURL string, tokens TreasuryX402ReceiptTokenSource, client *http.Client) (*TreasuryX402ReceiptReader, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("Treasury x402 receipt reader requires an origin URL")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return nil, errors.New("Treasury x402 receipt reader requires HTTPS or loopback HTTP")
	}
	if tokens == nil {
		return nil, errors.New("Treasury x402 receipt reader requires a service token source")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	} else {
		clone := *client
		client = &clone
	}
	// A service token is scoped for the Treasury receipt audience. Never follow
	// an origin-changing redirect with it.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &TreasuryX402ReceiptReader{baseURL: strings.TrimRight(parsed.String(), "/"), client: client, tokens: tokens}, nil
}

func (r *TreasuryX402ReceiptReader) ReadAccepted(ctx context.Context, externalID string) (TreasuryX402Receipt, error) {
	if r == nil || r.client == nil || r.tokens == nil {
		return TreasuryX402Receipt{}, errors.New("Treasury x402 receipt reader is unavailable")
	}
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return TreasuryX402Receipt{}, errors.New("Treasury x402 receipt lookup requires an external id")
	}
	token, err := r.tokens.Token(ctx)
	if err != nil {
		return TreasuryX402Receipt{}, fmt.Errorf("obtain Treasury receipt-read service token: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return TreasuryX402Receipt{}, errors.New("Treasury receipt-read service token is empty")
	}
	endpoint := r.baseURL + treasuryX402ReceiptPath + "?" + url.Values{"external_id": []string{externalID}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return TreasuryX402Receipt{}, fmt.Errorf("create Treasury receipt lookup: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	response, err := r.client.Do(request)
	if err != nil {
		return TreasuryX402Receipt{}, fmt.Errorf("read Treasury x402 receipt: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusNoContent {
		return TreasuryX402Receipt{}, ErrTreasuryX402ReceiptNotFound
	}
	if response.StatusCode != http.StatusOK {
		return TreasuryX402Receipt{}, fmt.Errorf("Treasury x402 receipt lookup returned HTTP %d", response.StatusCode)
	}
	var receipt TreasuryX402Receipt
	decoder := json.NewDecoder(io.LimitReader(response.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return TreasuryX402Receipt{}, fmt.Errorf("decode Treasury x402 receipt: %w", err)
	}
	if receipt.ExternalID != externalID || strings.TrimSpace(receipt.PostingID) == "" || strings.TrimSpace(receipt.ReceiptID) == "" || receipt.AcceptedAt.IsZero() || receipt.AmountMinor <= 0 || receipt.Currency == "" || receipt.Basis == "" {
		return TreasuryX402Receipt{}, errors.New("Treasury returned no complete accepted x402 Money Ledger receipt")
	}
	return receipt, nil
}
