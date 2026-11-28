package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

var ErrBrowserSessionReservationNotFound = errors.New("LPBS browser session reservation not found")

const (
	browserSessionSettlementPrincipal = "agent-manager-browser-session-settler"
	browserSessionSettlementResource  = "landing-page-business-suite"
	browserSessionSettlementAudience  = "scenario-authenticator:landing-page-business-suite"
	browserSessionSettlementScope     = "lpbs:browser-session.settle-and-receipt"
	browserSessionSettlementIdentity  = "scenario-authenticator/service-principal/agent-manager-browser-session-settler/landing-page-business-suite"
)

// DiscoveredBrowserSessionSettlementClient calls LPBS with its dedicated,
// scope-bound principal. It never forwards the incoming run identity token.
type DiscoveredBrowserSessionSettlementClient struct {
	Tokens     *servicetoken.Tokens
	Client     *http.Client
	ResolveURL func(context.Context) (string, error)
}

func NewDiscoveredBrowserSessionSettlementClient() *DiscoveredBrowserSessionSettlementClient {
	transport := &http.Transport{
		DialContext:  (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns: 8, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second,
	}
	tokens := servicetoken.New(servicetoken.Config{
		Name: "agent-manager browser session settlement",
		Credential: func(context.Context) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			identity, err := credentialauthority.ParseIdentity(browserSessionSettlementIdentity)
			if err != nil {
				return "", err
			}
			return authority.Require(identity, "credential")
		},
		Expect:        servicetoken.ExpectIdentity(browserSessionSettlementPrincipal, browserSessionSettlementAudience, browserSessionSettlementScope),
		RefreshMargin: time.Minute,
	})
	return &DiscoveredBrowserSessionSettlementClient{
		Tokens: tokens,
		Client: &http.Client{Timeout: 8 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "landing-page-business-suite")
		},
	}
}

func (c *DiscoveredBrowserSessionSettlementClient) SettleBrowserSession(ctx context.Context, runID uuid.UUID, sessionID, grantReceipt, proofReference string) (BrowserSessionReceipt, error) {
	if c == nil || c.Tokens == nil || c.Client == nil || c.ResolveURL == nil || runID == uuid.Nil || strings.TrimSpace(sessionID) == "" || strings.TrimSpace(grantReceipt) == "" || strings.TrimSpace(proofReference) == "" {
		return BrowserSessionReceipt{}, errors.New("LPBS browser settlement client is unavailable or request is incomplete")
	}
	base, err := c.ResolveURL(ctx)
	if err != nil {
		return BrowserSessionReceipt{}, errors.New("LPBS browser settlement endpoint is unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return BrowserSessionReceipt{}, errors.New("LPBS browser settlement endpoint is invalid")
	}
	token, err := c.Tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return BrowserSessionReceipt{}, errors.New("LPBS browser settlement service identity is unavailable")
	}
	body, err := json.Marshal(map[string]string{"run_id": runID.String(), "session_id": sessionID, "grant_receipt": grantReceipt, "proof_reference": proofReference})
	if err != nil {
		return BrowserSessionReceipt{}, errors.New("LPBS browser settlement request is invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(parsed.String(), "/")+"/api/v1/internal/browser-sessions/settle-and-receipt", bytes.NewReader(body))
	if err != nil {
		return BrowserSessionReceipt{}, errors.New("LPBS browser settlement request is unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.Client.Do(request)
	if err != nil {
		return BrowserSessionReceipt{}, errors.New("LPBS browser settlement request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode == http.StatusNotFound {
			return BrowserSessionReceipt{}, ErrBrowserSessionReservationNotFound
		}
		return BrowserSessionReceipt{}, fmt.Errorf("LPBS browser settlement refused request (HTTP %d)", response.StatusCode)
	}
	encoded, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(encoded) > 64<<10 {
		return BrowserSessionReceipt{}, errors.New("LPBS browser receipt response exceeded limit")
	}
	var receipt BrowserSessionReceipt
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return BrowserSessionReceipt{}, errors.New("LPBS browser receipt response is invalid")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return BrowserSessionReceipt{}, errors.New("LPBS browser receipt response is invalid")
	}
	if err := validateSettledBrowserSessionReceipt(receipt); err != nil {
		return BrowserSessionReceipt{}, err
	}
	return receipt, nil
}

func validateSettledBrowserSessionReceipt(receipt BrowserSessionReceipt) error {
	if receipt.Status != "settled" || strings.TrimSpace(receipt.ReservationID) == "" ||
		strings.TrimSpace(receipt.BusinessAccountID) == "" || strings.TrimSpace(receipt.Provider) == "" ||
		strings.TrimSpace(receipt.Model) == "" || receipt.SessionMinutes < 0 || receipt.Actions < 0 ||
		receipt.Bytes < 0 || receipt.CreditsCharged < 0 || receipt.CostMicros < 0 ||
		receipt.PromptTokens < 0 || receipt.CompletionTokens < 0 || len(receipt.InferenceOperations) == 0 {
		return errors.New("LPBS browser receipt response is incomplete or not settled")
	}
	var credits, cost, promptTokens, completionTokens int64
	seen := make(map[string]struct{}, len(receipt.InferenceOperations))
	for _, operation := range receipt.InferenceOperations {
		id := strings.TrimSpace(operation.RootOperationID)
		if id == "" || strings.TrimSpace(operation.Provider) == "" || strings.TrimSpace(operation.Model) == "" ||
			operation.CreditsCharged < 0 || operation.CostMicros < 0 || operation.PromptTokens < 0 || operation.CompletionTokens < 0 {
			return errors.New("LPBS browser receipt contains an invalid inference operation")
		}
		if _, duplicate := seen[id]; duplicate {
			return errors.New("LPBS browser receipt contains a duplicate inference operation")
		}
		seen[id] = struct{}{}
		var ok bool
		if credits, ok = addReceiptTotal(credits, operation.CreditsCharged); !ok {
			return errors.New("LPBS browser receipt inference totals overflow")
		}
		if cost, ok = addReceiptTotal(cost, operation.CostMicros); !ok {
			return errors.New("LPBS browser receipt inference totals overflow")
		}
		if promptTokens, ok = addReceiptTotal(promptTokens, operation.PromptTokens); !ok {
			return errors.New("LPBS browser receipt inference totals overflow")
		}
		if completionTokens, ok = addReceiptTotal(completionTokens, operation.CompletionTokens); !ok {
			return errors.New("LPBS browser receipt inference totals overflow")
		}
	}
	if credits != receipt.CreditsCharged || cost != receipt.CostMicros ||
		promptTokens != receipt.PromptTokens || completionTokens != receipt.CompletionTokens {
		return errors.New("LPBS browser receipt aggregate does not match its inference operations")
	}
	return nil
}

func addReceiptTotal(total, value int64) (int64, bool) {
	if value < 0 || total > math.MaxInt64-value {
		return 0, false
	}
	return total + value, true
}
