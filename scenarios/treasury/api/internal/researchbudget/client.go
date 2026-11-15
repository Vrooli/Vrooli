// Package researchbudget publishes Treasury-owned, run-bound budget and
// settlement evidence to Agent Manager. Treasury remains the authority for
// budget headroom and settled receipt provenance; this package only transports
// those immutable projections through a dedicated service identity.
package researchbudget

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

	"github.com/google/uuid"
	"treasury/internal/ledger"
)

// TokenSource must return an authenticator-issued Treasury service token scoped
// to Agent Manager. It must not mint a principal or forward a run/operator token.
type TokenSource interface {
	Token(context.Context) (string, error)
}

// Snapshot is the immutable budget authority projection sourced from
// Treasury's headroom and run binding.
type Snapshot struct {
	TreasuryBudgetID string    `json:"treasury_budget_id"`
	MandateID        string    `json:"mandate_id"`
	AgentSubject     string    `json:"agent_subject"`
	CapMinor         int64     `json:"cap_minor"`
	Currency         string    `json:"currency"`
	UnitCode         string    `json:"unit_code"`
	PolicyRevision   string    `json:"policy_revision"`
	CreatedAt        time.Time `json:"created_at"`
}

// Settlement is copied from a settled Treasury record and receipt. Receipt and
// settlement identities are both retained so Agent Manager never invents
// provenance from an agent supplied result.
type Settlement struct {
	ID                string    `json:"id"`
	SettlementID      string    `json:"settlement_id"`
	TreasuryReceiptID string    `json:"treasury_receipt_id"`
	IntegrationID     string    `json:"integration_id"`
	Category          string    `json:"category"`
	AmountMinor       int64     `json:"amount_minor"`
	Currency          string    `json:"currency"`
	UnitCode          string    `json:"unit_code"`
	ResultRef         string    `json:"result_ref"`
	BenefitState      string    `json:"benefit_state"`
	BenefitSummary    string    `json:"benefit_summary"`
	SettledAt         time.Time `json:"settled_at"`
}

// Client exposes only the two fixed Treasury evidence routes. Exact operation
// authority is evaluated by Agent Manager from its trusted request authority;
// this client cannot manufacture a grant or choose another write route.
type Client struct {
	baseURL string
	http    *http.Client
	tokens  TokenSource
}

func NewClient(baseURL string, client *http.Client, tokens TokenSource) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("Agent Manager HTTP(S) API URL is required")
	}
	if tokens == nil {
		return nil, errors.New("Agent Manager Treasury service token source is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	// Evidence must be sent only to the configured Agent Manager origin. A
	// redirect could otherwise forward Treasury's service credential to an
	// unrelated host or turn the fixed write path into a caller-controlled one.
	clientCopy := *client
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{baseURL: baseURL, http: &clientCopy, tokens: tokens}, nil
}

// NewAuthenticatedClient is a convenience constructor that binds Treasury's
// existing service credential reader to the distinct Agent Manager audience.
func NewAuthenticatedClient(baseURL string, client *http.Client, credentials interface {
	Resolve(context.Context, string, string) (string, error)
}, authenticatorURL string) (*Client, error) {
	if credentials == nil || strings.TrimSpace(authenticatorURL) == "" {
		return nil, errors.New("Treasury credential authority and scenario authenticator are required")
	}
	tokens, err := ledger.NewServiceTokenSourceFor(credentials, authenticatorURL, client, ledger.AgentManagerPrincipal, ledger.AgentManagerResource)
	if err != nil {
		return nil, err
	}
	return NewClient(baseURL, client, tokens)
}

func (c *Client) PutSnapshot(ctx context.Context, runID string, snapshot Snapshot) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("Agent Manager research budget client is unavailable")
	}
	parsedRunID, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return errors.New("valid Agent Manager run UUID is required")
	}
	if snapshot.TreasuryBudgetID == "" || snapshot.MandateID == "" || snapshot.AgentSubject == "" || snapshot.CapMinor < 0 || snapshot.Currency == "" || snapshot.UnitCode == "" || snapshot.PolicyRevision == "" || snapshot.CreatedAt.IsZero() {
		return errors.New("Treasury budget, mandate, agent, nonnegative cap, currency, unit, policy revision and creation time are required")
	}
	return c.post(ctx, "/api/v1/research-budgets/"+parsedRunID.String()+"/snapshot", snapshot)
}

func (c *Client) RecordSettlement(ctx context.Context, runID string, settlement Settlement) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("Agent Manager research budget client is unavailable")
	}
	parsedRunID, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return errors.New("valid Agent Manager run UUID is required")
	}
	if settlement.ID == "" || settlement.SettlementID == "" || settlement.TreasuryReceiptID == "" || settlement.IntegrationID == "" || settlement.Currency == "" || settlement.UnitCode == "" || settlement.AmountMinor < 0 || settlement.SettledAt.IsZero() {
		return errors.New("Treasury settlement and receipt identities, integration, nonnegative amount, currency, unit and settlement time are required")
	}
	return c.post(ctx, "/api/v1/research-budgets/"+parsedRunID.String()+"/settlements", settlement)
}

func (c *Client) post(ctx context.Context, path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Treasury research budget evidence: %w", err)
	}
	token, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return errors.New("Treasury Agent Manager service identity is unavailable")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build Agent Manager evidence request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("post Treasury research budget evidence: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Agent Manager research budget evidence returned %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	return nil
}
