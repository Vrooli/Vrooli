package agentaccess

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

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

type browserCredentialsKey struct{}

// WithBrowserAgentCredentials places only opaque credentials in request
// context. Agent Manager verifies them before returning a run principal.
func WithBrowserAgentCredentials(ctx context.Context, identityToken, grantReceipt string) context.Context {
	return context.WithValue(ctx, browserCredentialsKey{}, browserCredentials{identityToken: identityToken, grantReceipt: grantReceipt})
}

type browserCredentials struct{ identityToken, grantReceipt string }

func credentialsFromContext(ctx context.Context) (browserCredentials, error) {
	credentials, ok := ctx.Value(browserCredentialsKey{}).(browserCredentials)
	if !ok || strings.TrimSpace(credentials.identityToken) == "" || strings.TrimSpace(credentials.grantReceipt) == "" {
		return browserCredentials{}, errors.New("Agent Manager identity token and browser grant receipt are required")
	}
	return credentials, nil
}

// AgentManagerRunContext resolves a caller from the signed token on every
// request. It does not parse token claims or accept identities from JSON.
type AgentManagerRunContext struct {
	BaseURL string
	Client  *http.Client
}

// AgentManagerProfileResolver reloads the persisted run capability for each
// call. The session facade retains the returned Policy for stable per-session
// action/time counters and rejects profile drift before applying it.
type AgentManagerProfileResolver struct {
	RunContext            *AgentManagerRunContext
	ApprovalVerifier      sessionpermission.ReceiptVerifier
	ConsentVerifier       sessionpermission.ConsentVerifier
	CredentialUseVerifier sessionpermission.CredentialUseVerifier
	CreditBudgetVerifier  sessionpermission.CreditBudgetVerifier
	ByteBudgetVerifier    sessionpermission.ByteBudgetVerifier
}

func (a *AgentManagerProfileResolver) Resolve(ctx context.Context, profileRef string) (sessionpermission.Profile, sessionpermission.Authorizer, error) {
	if a == nil || a.RunContext == nil {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager profile resolver unavailable")
	}
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return sessionpermission.Profile{}, nil, err
	}
	endpoint, client, err := a.RunContext.endpoint("/api/v1/browser/resolve-profile")
	if err != nil {
		return sessionpermission.Profile{}, nil, err
	}
	payload, err := json.Marshal(map[string]string{"profile_ref": profileRef})
	if err != nil {
		return sessionpermission.Profile{}, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return sessionpermission.Profile{}, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Identity-Token", credentials.identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", credentials.grantReceipt)
	resp, err := client.Do(req)
	if err != nil {
		return sessionpermission.Profile{}, nil, fmt.Errorf("resolve Agent Manager browser profile: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return sessionpermission.Profile{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return sessionpermission.Profile{}, nil, fmt.Errorf("Agent Manager profile resolver returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var response struct {
		Status            string `json:"status"`
		RunID             string `json:"run_id"`
		GrantReceipt      string `json:"grant_receipt"`
		Mode              string `json:"mode"`
		PermissionProfile struct {
			ProfileID                    string                                    `json:"profileId"`
			ProfileMode                  string                                    `json:"profileMode"`
			Level                        string                                    `json:"level"`
			ExtensionInstall             *bool                                     `json:"extensionInstall,omitempty"`
			ExtensionLoad                *bool                                     `json:"extensionLoad,omitempty"`
			View                         bool                                      `json:"view"`
			Control                      bool                                      `json:"control"`
			AllowedOrigins               []string                                  `json:"allowedOrigins"`
			CredentialRefs               []string                                  `json:"credentialRefs"`
			ExtensionIDs                 []string                                  `json:"extensionIds"`
			AllowHistory                 bool                                      `json:"allowHistory"`
			AudioMode                    string                                    `json:"audioMode"`
			PerformanceTrace             bool                                      `json:"performanceTrace"`
			AppMonitorMetrics            bool                                      `json:"appMonitorMetrics"`
			AppMonitorTrace              bool                                      `json:"appMonitorTrace"`
			AppMonitorTraceMaxDurationMS int64                                     `json:"appMonitorTraceMaxDurationMs"`
			AppMonitorTraceMaxBytes      int64                                     `json:"appMonitorTraceMaxBytes"`
			RequireApprovalFor           []string                                  `json:"requireApprovalFor"`
			IdentityProfile              *sessionpermission.BrowserIdentityProfile `json:"identity_profile,omitempty"`
		} `json:"permission_profile"`
		SessionPermissionMapping struct {
			View                 bool                `json:"view"`
			Control              bool                `json:"control"`
			ProfileMode          string              `json:"profile_mode"`
			ExtensionAccess      bool                `json:"extension_access"`
			ExtensionPermissions map[string][]string `json:"extension_permissions"`
		} `json:"session_permission_mapping"`
		Budget struct {
			SessionMinutes  int64 `json:"sessionMinutes"`
			Actions         int64 `json:"actions"`
			Bytes           int64 `json:"bytes"`
			InferenceTokens int64 `json:"inferenceTokens"`
		} `json:"budget"`
		Unsupported []string `json:"unsupported_sessionpermission_fields"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return sessionpermission.Profile{}, nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager profile response must contain one JSON object")
	}
	if response.Status != "active" || response.GrantReceipt != credentials.grantReceipt || response.PermissionProfile.ProfileID != profileRef {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager profile is inactive or not bound to the selected grant")
	}
	wantMode, supportedMode := agentManagerProfileMode(response.Mode)
	if !supportedMode || response.PermissionProfile.ProfileMode == "" || response.PermissionProfile.ProfileMode != wantMode || response.SessionPermissionMapping.ProfileMode != response.PermissionProfile.ProfileMode {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager browser profile is missing a matching persisted profile mode")
	}
	if response.PermissionProfile.PerformanceTrace && (response.Mode != "managed" || response.PermissionProfile.ProfileMode != "fresh") {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager performance trace grant is unsupported for this browser mode")
	}
	if (response.PermissionProfile.AppMonitorMetrics || response.PermissionProfile.AppMonitorTrace) && (response.Mode != "managed" || response.PermissionProfile.ProfileMode != "fresh") {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager App Monitor capture grant is unsupported for this browser mode")
	}
	if response.PermissionProfile.AppMonitorTrace && (response.PermissionProfile.AppMonitorTraceMaxDurationMS <= 0 || response.PermissionProfile.AppMonitorTraceMaxDurationMS > 60_000 || response.PermissionProfile.AppMonitorTraceMaxBytes <= 0 || response.PermissionProfile.AppMonitorTraceMaxBytes > 16<<20) {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager App Monitor trace grant has missing or oversized capture ceilings")
	}
	if !response.PermissionProfile.AppMonitorTrace && (response.PermissionProfile.AppMonitorTraceMaxDurationMS != 0 || response.PermissionProfile.AppMonitorTraceMaxBytes != 0) {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager App Monitor trace ceilings require the explicit trace grant")
	}
	profile := sessionpermission.Profile{ID: profileRef, ProfileID: response.PermissionProfile.ProfileID, ProfileMode: response.PermissionProfile.ProfileMode, CredentialIDs: append([]string(nil), response.PermissionProfile.CredentialRefs...), HistoryRead: response.PermissionProfile.AllowHistory, AudioMode: sessionpermission.AudioMode(response.PermissionProfile.AudioMode), PerformanceTrace: response.PermissionProfile.PerformanceTrace, AppMonitorMetrics: response.PermissionProfile.AppMonitorMetrics, AppMonitorTrace: response.PermissionProfile.AppMonitorTrace, AppMonitorTraceMaxDurationMS: response.PermissionProfile.AppMonitorTraceMaxDurationMS, AppMonitorTraceMaxBytes: response.PermissionProfile.AppMonitorTraceMaxBytes, MaxActions: response.Budget.Actions, MaxBytes: response.Budget.Bytes}
	if response.PermissionProfile.IdentityProfile != nil {
		identity := *response.PermissionProfile.IdentityProfile
		profile.IdentityProfile = &identity
	}
	unsupportedFields := make(map[string]bool, len(response.Unsupported))
	for _, field := range response.Unsupported {
		unsupportedFields[field] = true
	}
	if unsupportedFields["ProfileMode"] {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager does not persist a usable browser profile mode")
	}
	if !unsupportedFields["View"] {
		profile.View = response.SessionPermissionMapping.View
	}
	if !unsupportedFields["Control"] {
		profile.Control = response.SessionPermissionMapping.Control
	}
	if !unsupportedFields["ExtensionAccess"] {
		profile.ExtensionAccess = response.SessionPermissionMapping.ExtensionAccess
		profile.ExtensionPermissions = make(map[string][]string, len(response.SessionPermissionMapping.ExtensionPermissions))
		for id, permissions := range response.SessionPermissionMapping.ExtensionPermissions {
			id = strings.TrimSpace(id)
			if id == "" {
				return sessionpermission.Profile{}, nil, errors.New("Agent Manager extension allowlist contains an empty ID")
			}
			profile.ExtensionPermissions[id] = append([]string(nil), permissions...)
		}
		if profile.ExtensionAccess != (len(response.PermissionProfile.ExtensionIDs) > 0) || len(profile.ExtensionPermissions) != len(response.PermissionProfile.ExtensionIDs) {
			return sessionpermission.Profile{}, nil, errors.New("Agent Manager extension access mapping does not match its exact extension ID allowlist")
		}
		for _, id := range response.PermissionProfile.ExtensionIDs {
			if _, ok := profile.ExtensionPermissions[id]; !ok {
				return sessionpermission.Profile{}, nil, errors.New("Agent Manager extension access mapping is missing an allowlisted extension ID")
			}
		}
	}
	if response.PermissionProfile.ExtensionInstall != nil {
		profile.ExtensionInstall = *response.PermissionProfile.ExtensionInstall
	}
	if response.PermissionProfile.ExtensionLoad != nil {
		profile.ExtensionLoad = *response.PermissionProfile.ExtensionLoad
	}
	if unsupportedFields["ExtensionInstall"] && profile.ExtensionInstall {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager marked extension installation unsupported while granting it")
	}
	if unsupportedFields["ExtensionLoad"] && profile.ExtensionLoad {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager marked extension loading unsupported while granting it")
	}
	if (profile.ExtensionInstall || profile.ExtensionLoad) && (response.Mode != "managed" || unsupportedFields["ExtensionInstall"] || unsupportedFields["ExtensionLoad"] || unsupportedFields["ExtensionAccess"] || !profile.ExtensionAccess || len(profile.ExtensionPermissions) == 0) {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager extension install/load requires a managed profile and its exact extension ID allowlist")
	}
	switch response.PermissionProfile.Level {
	case "observe":
		profile.Effect = sessionpermission.EffectObserve
	case "act":
		profile.Effect = sessionpermission.EffectAct
	case "unrestricted":
		profile.Effect = sessionpermission.EffectUnrestricted
	default:
		return sessionpermission.Profile{}, nil, fmt.Errorf("unknown Agent Manager browser permission level %q", response.PermissionProfile.Level)
	}
	if response.Budget.SessionMinutes <= 0 || response.Budget.Actions <= 0 {
		return sessionpermission.Profile{}, nil, errors.New("Agent Manager browser session-minute and action budgets must be positive")
	}
	profile.MaxDuration = time.Duration(response.Budget.SessionMinutes) * time.Minute
	for _, origin := range response.PermissionProfile.AllowedOrigins {
		pattern, err := originPattern(origin)
		if err != nil {
			return sessionpermission.Profile{}, nil, err
		}
		profile.URLAllow = append(profile.URLAllow, pattern)
	}
	if len(response.PermissionProfile.RequireApprovalFor) > 0 {
		profile.Approval = sessionpermission.ApprovalEveryAction
	}
	if profile.Approval == "" {
		profile.Approval = sessionpermission.ApprovalNone
	}
	policy, err := sessionpermission.NewWithVerifiers(profile, a.ApprovalVerifier, a.ConsentVerifier)
	if err != nil {
		return sessionpermission.Profile{}, nil, err
	}
	policy.SetCredentialUseVerifier(a.CredentialUseVerifier)
	policy.SetCreditBudgetVerifier(a.CreditBudgetVerifier)
	policy.SetByteBudgetVerifier(a.ByteBudgetVerifier)
	return profile, policy, nil
}

func agentManagerProfileMode(mode string) (string, bool) {
	switch strings.TrimSpace(mode) {
	case "managed":
		return "fresh", true
	case "installed-binary":
		return "isolated", true
	case "user-attached":
		return "user_attached", true
	case "remote":
		return "remote", true
	default:
		return "", false
	}
}

func originPattern(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", fmt.Errorf("invalid Agent Manager allowed origin %q", raw)
	}
	return u.Scheme + "://" + u.Host + "/*", nil
}

func (a *AgentManagerRunContext) ResolveBrowserOpen(ctx context.Context) (TrustedBrowserOpenContext, error) {
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return TrustedBrowserOpenContext{}, err
	}
	trusted, err := a.resolveContext(ctx, credentials)
	if err != nil {
		return TrustedBrowserOpenContext{}, err
	}
	return trusted, nil
}

func (a *AgentManagerRunContext) resolveContext(ctx context.Context, credentials browserCredentials) (TrustedBrowserOpenContext, error) {
	endpoint, client, err := a.endpoint("/api/v1/browser/resolve-context")
	if err != nil {
		return TrustedBrowserOpenContext{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader("{}"))
	if err != nil {
		return TrustedBrowserOpenContext{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Identity-Token", credentials.identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", credentials.grantReceipt)
	resp, err := client.Do(req)
	if err != nil {
		return TrustedBrowserOpenContext{}, fmt.Errorf("resolve Agent Manager browser run context: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return TrustedBrowserOpenContext{}, fmt.Errorf("Agent Manager browser context resolver returned HTTP %d", resp.StatusCode)
	}
	var response struct {
		CallerID     string `json:"caller_id"`
		RunID        string `json:"run_id"`
		GrantReceipt string `json:"grant_receipt"`
		Capability   struct {
			Permissions struct {
				ProfileID string `json:"profileId"`
			} `json:"permissions"`
		} `json:"capability"`
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := decoder.Decode(&response); err != nil {
		return TrustedBrowserOpenContext{}, fmt.Errorf("decode Agent Manager browser context: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return TrustedBrowserOpenContext{}, errors.New("Agent Manager browser context must contain one JSON response")
	}
	if response.CallerID == "" || strings.TrimSpace(response.RunID) == "" || response.GrantReceipt == "" || response.GrantReceipt != credentials.grantReceipt || strings.TrimSpace(response.Capability.Permissions.ProfileID) == "" {
		return TrustedBrowserOpenContext{}, errors.New("Agent Manager browser context does not match the active run grant and persisted profile")
	}
	return TrustedBrowserOpenContext{CallerID: response.CallerID, GrantReceipt: response.GrantReceipt, IdentityToken: credentials.identityToken, PermissionRef: response.Capability.Permissions.ProfileID, RunID: response.RunID}, nil
}

func (a *AgentManagerRunContext) BrowserCallerID(ctx context.Context) (string, error) {
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return "", err
	}
	return a.resolveCaller(ctx, credentials)
}

func (a *AgentManagerRunContext) resolveCaller(ctx context.Context, credentials browserCredentials) (string, error) {
	endpoint, client, err := a.endpoint("/api/v1/browser/resolve-caller")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Agent-Identity-Token", credentials.identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", credentials.grantReceipt)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve Agent Manager browser caller: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Agent Manager caller resolution returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var result struct {
		CallerID     string `json:"caller_id"`
		GrantReceipt string `json:"grant_receipt"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if result.CallerID == "" || result.GrantReceipt != credentials.grantReceipt {
		return "", errors.New("Agent Manager caller response is not bound to the active grant")
	}
	return result.CallerID, nil
}

func (a *AgentManagerRunContext) endpoint(path string) (*url.URL, *http.Client, error) {
	if a == nil {
		return nil, nil, errors.New("Agent Manager run-context adapter is unavailable")
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(a.BaseURL), "/"))
	if err != nil || base == nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, nil, errors.New("Agent Manager URL must be an absolute http(s) origin")
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return base.ResolveReference(&url.URL{Path: path}), client, nil
}

// AgentManagerOpenAuthorizer delegates fresh-session admission to the existing
// Agent Manager run grant. It sends only the signed identity token and the
// opaque grant reference resolved from trusted context.
type AgentManagerOpenAuthorizer struct {
	BaseURL string
	Client  *http.Client
	Context TrustedOpenContextResolver
}

func (a *AgentManagerOpenAuthorizer) AuthorizeOpen(ctx context.Context, request sessionpermission.OpenRequest) error {
	if a == nil || a.Context == nil {
		return errors.New("Agent Manager trusted run context is required")
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(a.BaseURL), "/"))
	if err != nil || base == nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return errors.New("Agent Manager browser authorization URL must be an absolute http(s) origin")
	}
	trusted, err := a.Context.ResolveBrowserOpen(ctx)
	if err != nil {
		return fmt.Errorf("resolve trusted Agent Manager run: %w", err)
	}
	if trusted.IdentityToken == "" || trusted.CallerID == "" || trusted.GrantReceipt == "" {
		return errors.New("signed run token, caller, and grant receipt are required")
	}
	if request.CallerID != trusted.CallerID || request.GrantReceipt != trusted.GrantReceipt {
		return errors.New("browser open request is not bound to trusted run context")
	}
	mode, err := agentManagerMode(request.Backend)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]string{"mode": mode, "profile_id": request.ProfileID, "initial_url": request.InitialURL})
	if err != nil {
		return err
	}
	endpoint := base.ResolveReference(&url.URL{Path: "/api/v1/browser/authorize-open"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Identity-Token", trusted.IdentityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", trusted.GrantReceipt)
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Agent Manager browser open authorization: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Agent Manager denied browser open (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var result struct {
		Authorized   bool   `json:"authorized"`
		GrantReceipt string `json:"grant_receipt"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode Agent Manager open decision: %w", err)
	}
	if !result.Authorized || result.GrantReceipt != trusted.GrantReceipt {
		return errors.New("Agent Manager did not return an authorized matching grant decision")
	}
	return nil
}

func agentManagerMode(backend sessionpermission.Backend) (string, error) {
	switch backend {
	case sessionpermission.BackendManaged:
		return "managed", nil
	case sessionpermission.BackendInstalled:
		return "installed-binary", nil
	case sessionpermission.BackendUserAttached:
		return "user-attached", nil
	case sessionpermission.BackendRemote:
		return "remote", nil
	default:
		return "", fmt.Errorf("unsupported BAS browser backend %q", backend)
	}
}
