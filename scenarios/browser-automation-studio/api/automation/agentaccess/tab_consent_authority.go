package agentaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/principalrole"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

const (
	BrowserTabConsentAudience = "scenario-authenticator:agent-manager"
	BrowserTabConsentScope    = "agent-manager:browser-tab-consent"
	BrowserTabConsentSuffix   = "-browser-tab-consent"
)

// BrowserTabConsentBrokerRole is BAS's narrow service identity for issuing
// and checking receipts after the selected-tab bridge has verified pairing.
var BrowserTabConsentBrokerRole = principalrole.NewRole(
	BrowserTabConsentAudience, BrowserTabConsentScope, "agent-manager", BrowserTabConsentSuffix, "browser tab consent broker",
)

var ErrTabConsentServiceUnavailable = errors.New("selected-tab consent service identity is unavailable")

// AgentManagerTabConsentAuthority delegates selected-tab receipts to the
// existing Agent Manager browser permission owner. BAS never stores an
// independent consent record; its own lease still limits the active socket.
type AgentManagerTabConsentAuthority struct {
	RunContext    *AgentManagerRunContext
	ServiceTokens BrowserEvidenceTokenSource
	Client        *http.Client
}

func (a *AgentManagerTabConsentAuthority) IssueBrowserTabConsent(ctx context.Context, consent TabConsentRequest) (string, error) {
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return "", err
	}
	if consent.GrantReceipt == "" || consent.GrantReceipt != credentials.grantReceipt {
		return "", errors.New("selected-tab consent request is not bound to the active browser grant")
	}
	payload := map[string]any{
		"session_id": consent.SessionID, "lease_id": consent.LeaseID, "owner_id": consent.OwnerID,
		"profile_id": consent.PermissionRef, "tab_id": consent.TabID, "url": consent.URL,
		"pairing_nonce": consent.PairingNonce, "expires_at": consent.ExpiresAt.UTC(),
	}
	data, err := a.call(ctx, credentials, "/api/v1/browser/tab-consent/issue", payload, http.StatusOK)
	if err != nil {
		return "", err
	}
	var result struct {
		Receipt string `json:"consent_receipt"`
	}
	if err := json.Unmarshal(data, &result); err != nil || strings.TrimSpace(result.Receipt) == "" {
		return "", errors.New("Agent Manager returned an empty selected-tab consent receipt")
	}
	return result.Receipt, nil
}

func (a *AgentManagerTabConsentAuthority) VerifyBrowserTabConsent(ctx context.Context, receipt string, action sessionpermission.ActionRequest) error {
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return err
	}
	payload := map[string]string{
		"consent_receipt": strings.TrimSpace(receipt),
		"session_id":      action.SessionID, "lease_id": action.LeaseID, "owner_id": action.OwnerID,
		"tab_id": action.TabID, "current_url": action.CurrentURL,
	}
	_, err = a.call(ctx, credentials, "/api/v1/browser/tab-consent/verify", payload, http.StatusNoContent)
	return err
}

func (a *AgentManagerTabConsentAuthority) RevokeBrowserTabConsent(ctx context.Context, receipt, runID, sessionID string) error {
	payload := map[string]string{"consent_receipt": strings.TrimSpace(receipt), "run_id": strings.TrimSpace(runID), "session_id": strings.TrimSpace(sessionID)}
	_, err := a.call(ctx, browserCredentials{}, "/api/v1/browser/tab-consent/revoke", payload, http.StatusNoContent)
	return err
}

func (a *AgentManagerTabConsentAuthority) call(ctx context.Context, credentials browserCredentials, path string, payload any, expectedStatus int) ([]byte, error) {
	if a == nil || a.RunContext == nil {
		return nil, errors.New("Agent Manager tab consent authority is unavailable")
	}
	endpoint, defaultClient, err := a.RunContext.endpoint(path)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Identity-Token", credentials.identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", credentials.grantReceipt)
	if a.ServiceTokens == nil {
		return nil, ErrTabConsentServiceUnavailable
	}
	serviceToken, err := a.ServiceTokens.Token(ctx)
	if err != nil || strings.TrimSpace(serviceToken) == "" {
		if err == nil {
			err = errors.New("service identity returned an empty token")
		}
		return nil, fmt.Errorf("resolve BAS selected-tab consent service identity: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+serviceToken)
	client := a.Client
	if client == nil {
		client = defaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Agent Manager selected-tab consent request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != expectedStatus {
		return nil, fmt.Errorf("Agent Manager selected-tab consent returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

var _ TabConsentIssuer = (*AgentManagerTabConsentAuthority)(nil)
var _ sessionpermission.ConsentVerifier = (*AgentManagerTabConsentAuthority)(nil)
