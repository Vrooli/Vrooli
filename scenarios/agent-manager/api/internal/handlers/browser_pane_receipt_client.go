package handlers

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

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	browserPaneIssuerPrincipal = "agent-manager-browser-pane-issuer"
	browserPaneIssuerScope     = "browser-automation-studio:browser-pane.issue"
	browserPaneIssuerIdentity  = "scenario-authenticator/service-principal/agent-manager-browser-pane-issuer/browser-automation-studio"
)

// DiscoveredBrowserPaneReceiptIssuer uses a dedicated service identity for
// live-pane delegation. It intentionally does not reuse the evidence-proof
// principal; the credential must be separately declared and operator-issued.
type DiscoveredBrowserPaneReceiptIssuer struct {
	Tokens     *servicetoken.Tokens
	Client     *http.Client
	ResolveURL func(context.Context) (string, error)
}

func NewDiscoveredBrowserPaneReceiptIssuer() *DiscoveredBrowserPaneReceiptIssuer {
	transport := &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns: 8, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second}
	tokens := servicetoken.New(servicetoken.Config{
		Name: "agent-manager browser pane issuer",
		Credential: func(context.Context) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			identity, err := credentialauthority.ParseIdentity(browserPaneIssuerIdentity)
			if err != nil {
				return "", err
			}
			return authority.Require(identity, "credential")
		},
		Expect:        servicetoken.ExpectIdentity(browserPaneIssuerPrincipal, "scenario-authenticator:browser-automation-studio", browserPaneIssuerScope),
		RefreshMargin: time.Minute,
	})
	return &DiscoveredBrowserPaneReceiptIssuer{Tokens: tokens, Client: &http.Client{Timeout: 8 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
		}}
}

func (c *DiscoveredBrowserPaneReceiptIssuer) IssueBrowserPaneReceipt(ctx context.Context, runID uuid.UUID, grant, profileID, sessionID, role string) (BrowserPaneReceipt, error) {
	return c.issueBrowserPaneReceipt(ctx, runID, grant, profileID, sessionID, role, time.Time{})
}

func (c *DiscoveredBrowserPaneReceiptIssuer) IssueBrowserPaneReceiptUntil(ctx context.Context, runID uuid.UUID, grant, profileID, sessionID, role string, notAfter time.Time) (BrowserPaneReceipt, error) {
	return c.issueBrowserPaneReceipt(ctx, runID, grant, profileID, sessionID, role, notAfter)
}

func (c *DiscoveredBrowserPaneReceiptIssuer) issueBrowserPaneReceipt(ctx context.Context, runID uuid.UUID, grant, profileID, sessionID, role string, notAfter time.Time) (BrowserPaneReceipt, error) {
	if c == nil || c.Tokens == nil || c.Client == nil || c.ResolveURL == nil {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt issuer unavailable")
	}
	base, err := c.ResolveURL(ctx)
	if err != nil {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt endpoint unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt endpoint invalid")
	}
	token, err := c.Tokens.Token(ctx)
	if err != nil {
		return BrowserPaneReceipt{}, errors.New("BAS pane issuer service identity unavailable")
	}
	requestBody := map[string]any{"run_id": runID.String(), "grant_receipt": grant, "profile_id": profileID, "session_id": sessionID, "role": role}
	if !notAfter.IsZero() { requestBody["not_after"] = notAfter.UTC() }
	body, err := json.Marshal(requestBody)
	if err != nil {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt request invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(parsed.String(), "/")+"/api/v1/browser/internal/pane-stream-receipts/issue", bytes.NewReader(body))
	if err != nil {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt request unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.Client.Do(request)
	if err != nil {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return BrowserPaneReceipt{}, fmt.Errorf("BAS pane receipt refused request (HTTP %d)", response.StatusCode)
	}
	encoded, err := io.ReadAll(io.LimitReader(response.Body, (8<<10)+1))
	if err != nil || len(encoded) > 8<<10 {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt response exceeded limit")
	}
	var result BrowserPaneReceipt
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt response invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt response has trailing data")
	}
	if result.SessionID != sessionID || result.Role != role || strings.TrimSpace(result.Receipt) == "" || (!notAfter.IsZero() && (result.ExpiresAt.IsZero() || result.ExpiresAt.After(notAfter))) {
		return BrowserPaneReceipt{}, errors.New("BAS pane receipt response binding mismatch")
	}
	return result, nil
}

func (c *DiscoveredBrowserPaneReceiptIssuer) ListBrowserPaneSessions(ctx context.Context, runID uuid.UUID, grant, profileID string) ([]BrowserPaneSession, error) {
	if c == nil || c.Tokens == nil || c.Client == nil || c.ResolveURL == nil {
		return nil, errors.New("BAS pane session issuer unavailable")
	}
	base, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, errors.New("BAS pane session endpoint unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("BAS pane session endpoint invalid")
	}
	token, err := c.Tokens.Token(ctx)
	if err != nil {
		return nil, errors.New("BAS pane issuer service identity unavailable")
	}
	body, err := json.Marshal(map[string]string{"run_id": runID.String(), "grant_receipt": grant, "profile_id": profileID})
	if err != nil {
		return nil, errors.New("BAS pane session request invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(parsed.String(), "/")+"/api/v1/browser/internal/pane-stream-receipts/list", bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("BAS pane session request unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.Client.Do(request)
	if err != nil {
		return nil, errors.New("BAS pane session request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("BAS pane session list refused (HTTP %d)", response.StatusCode)
	}
	encoded, err := io.ReadAll(io.LimitReader(response.Body, (16<<10)+1))
	if err != nil || len(encoded) > 16<<10 {
		return nil, errors.New("BAS pane session list exceeded limit")
	}
	var result struct {
		Sessions []BrowserPaneSession `json:"sessions"`
	}
	dec := json.NewDecoder(bytes.NewReader(encoded))
	dec.DisallowUnknownFields()
	if dec.Decode(&result) != nil {
		return nil, errors.New("BAS pane session list invalid")
	}
	var trailing any
	if dec.Decode(&trailing) != io.EOF {
		return nil, errors.New("BAS pane session list has trailing data")
	}
	if len(result.Sessions) > 64 {
		return nil, errors.New("BAS pane session list exceeded count")
	}
	for _, session := range result.Sessions {
		if strings.TrimSpace(session.SessionID) == "" || strings.TrimSpace(session.Mode) == "" {
			return nil, errors.New("BAS pane session binding invalid")
		}
	}
	return result.Sessions, nil
}
