package deployments

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"deployment-manager/shared"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// LPBSScenarioSlug is the discovery name for the landing-page-business-suite
// destination that owns the download catalog, bucket, and channels.
const LPBSScenarioSlug = "landing-page-business-suite"

// lpbsServiceSecretField names the credential authority field LPBS uses to
// validate a server-to-server bearer token.
const lpbsServiceSecretField = "service-secret"

// lpbsServiceSecretIdentity is the credential authority identity that owns the
// shared service secret.
const lpbsServiceSecretIdentity credentialauthority.Identity = "vrooli/landing-page-business-suite"

// defaultLPBSScenarioURL resolves the local LPBS API base through scenario
// discovery so deployment-manager can reach the destination without a
// hand-set LPBS_BASE_URL.
func defaultLPBSScenarioURL(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, LPBSScenarioSlug)
}

// defaultLPBSServiceSecret resolves the shared LPBS service credential from the
// credential authority, the same value LPBS compares against the bearer token.
func defaultLPBSServiceSecret() (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	before := authority.Status(lpbsServiceSecretIdentity, lpbsServiceSecretField)
	if string(before.ProviderState) != "available" || !before.Configured || strings.TrimSpace(before.Version) == "" {
		return "", fmt.Errorf("legacy LPBS credential is unavailable or unversioned")
	}
	value, err := authority.Resolve(lpbsServiceSecretIdentity, lpbsServiceSecretField)
	if err != nil {
		return "", fmt.Errorf("resolve legacy LPBS credential")
	}
	after := authority.Status(lpbsServiceSecretIdentity, lpbsServiceSecretField)
	if string(after.ProviderState) != "available" || !after.Configured || after.Version != before.Version {
		return "", fmt.Errorf("legacy LPBS credential changed during resolution")
	}
	return value, nil
}

const deploymentManagerLPBSPrincipal = "deployment-manager-lpbs-release-manager"

// deploymentManagerLPBSTokens reads the issued credential each time it must
// exchange for a short-lived LPBS release token. The authority value is never
// retained by the HTTP client.
func deploymentManagerLPBSTokens() *servicetoken.Tokens {
	role := lpbsidentity.DeploymentManagerRelease
	return servicetoken.New(servicetoken.Config{
		Name:   "deployment-manager " + role.Label,
		Expect: servicetoken.ExpectIdentity(role.Principal("deployment-manager"), role.Audience, role.Scope),
		Credential: func(context.Context) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			identity, err := credentialauthority.ParseIdentity("scenario-authenticator/service-principal/" + deploymentManagerLPBSPrincipal + "/" + lpbsidentity.Resource)
			if err != nil {
				return "", err
			}
			credential, err := authority.Require(identity, "credential")
			if err != nil {
				return "", err
			}
			return credential, nil
		},
		Describe: func(kind, cause error) error {
			if cause != nil {
				return fmt.Errorf("%w: %w", kind, cause)
			}
			return kind
		},
	}, servicetoken.WithExchanger(servicetoken.AuthenticatorExchanger(&http.Client{Timeout: 5 * time.Second}, func(ctx context.Context) (string, error) {
		if configured := strings.TrimSpace(os.Getenv("SCENARIO_AUTHENTICATOR_API_URL")); configured != "" {
			return strings.TrimRight(configured, "/"), nil
		}
		return discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	})))
}

// LPBSReleaseClient talks to landing-page-business-suite for release-time
// control-plane checks: upload readiness and post-publish verification.
// The same interface is wrapped by tests to inject fakes.
type LPBSReleaseClient interface {
	// CheckDeployReadiness calls POST /api/v1/deploy-readiness for the app.
	// A 200 response indicates LPBS is ready to accept a new release.
	CheckDeployReadiness(ctx context.Context, req *LPBSReadinessRequest) (*LPBSReadinessResult, error)

	// Verify calls GET /api/v1/updates/{app_key}/verify and reports whether
	// the expected version is live on the channel.
	Verify(ctx context.Context, req *LPBSVerifyRequest) (*LPBSVerifyResult, error)
}

type LPBSRecoveryRequest struct {
	AppKey                string
	VariantKey            string
	ExpectedRevision      int64
	ExpectedPredecessor   int64
	Action                string
	DataCompatibility     string
	ArtifactIDs           map[string]int64
	CandidateID           string
	DestinationRevisionID string
	Halted                bool
	Reason                string
	Confirmation          string
	DryRun                bool
}

type LPBSRecoveryReceipt struct {
	AppKey                string    `json:"app_key"`
	VariantKey            string    `json:"variant_key"`
	Revision              int64     `json:"revision"`
	PredecessorRevision   int64     `json:"predecessor_revision"`
	Action                string    `json:"action"`
	Halted                bool      `json:"halted"`
	CandidateID           string    `json:"candidate_id,omitempty"`
	DestinationRevisionID string    `json:"destination_revision_id,omitempty"`
	Outcome               string    `json:"outcome"`
	Health                string    `json:"health"`
	ExternalReceipt       string    `json:"external_receipt"`
	ObservedAt            time.Time `json:"observed_at"`
	DryRun                bool      `json:"dry_run"`
}

type LPBSRecoveryClient interface {
	HaltChannel(context.Context, *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error)
}

type LPBSChannelRecoveryClient interface {
	RecoverChannel(context.Context, *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error)
}

// LPBSReadinessRequest is the body for the deploy-readiness check.
type LPBSReadinessRequest struct {
	AppKey        string `json:"app_key"`
	RemoteProfile string `json:"remote_profile,omitempty"`
	Channel       string `json:"channel,omitempty"`
	ProfileTag    string `json:"profile_tag,omitempty"`
}

// LPBSReadinessResult is the decoded deploy-readiness response.
type LPBSReadinessResult struct {
	Ready bool            `json:"ready"`
	Gates []ReadinessGate `json:"gates,omitempty"`
	Error string          `json:"error,omitempty"`
}

// ReadinessGate reports one gate's status within the readiness response.
type ReadinessGate struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

// LPBSVerifyRequest is the input for a post-release verify call.
type LPBSVerifyRequest struct {
	AppKey          string
	Channel         string
	Platform        string
	ExpectedVersion string
	ExpectedSHA512  string
	Deep            bool
}

// LPBSVerifyResult mirrors LPBS /verify response.
type LPBSVerifyResult struct {
	AppKey          string `json:"app_key"`
	Channel         string `json:"channel"`
	Platform        string `json:"platform"`
	ExpectedVersion string `json:"expected_version"`
	ObservedVersion string `json:"observed_version,omitempty"`
	ObservedSHA512  string `json:"observed_sha512,omitempty"`
	SHA512Match     bool   `json:"sha512_match"`
	Match           bool   `json:"match"`
	Error           string `json:"error,omitempty"`
}

// UnmarshalJSON accepts only the producer's canonical actual_* fields. The
// adapter translates those names into the deployment-manager domain's
// observed_* fields, while refusing a successful response that omits the
// producer's byte and version observations.
func (r *LPBSVerifyResult) UnmarshalJSON(data []byte) error {
	type wire struct {
		AppKey          string  `json:"app_key"`
		Channel         string  `json:"channel"`
		Platform        string  `json:"platform"`
		ExpectedVersion string  `json:"expected_version"`
		ActualVersion   *string `json:"actual_version"`
		ActualSHA512    *string `json:"actual_sha512"`
		SHA512Match     *bool   `json:"sha512_match"`
		Match           bool    `json:"match"`
		Error           string  `json:"error,omitempty"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.ActualVersion == nil || value.ActualSHA512 == nil || value.SHA512Match == nil {
		return fmt.Errorf("verify response is missing canonical actual_version, actual_sha512, or sha512_match")
	}
	r.AppKey, r.Channel, r.Platform = value.AppKey, value.Channel, value.Platform
	r.ExpectedVersion = value.ExpectedVersion
	r.ObservedVersion = *value.ActualVersion
	r.ObservedSHA512 = *value.ActualSHA512
	r.SHA512Match = *value.SHA512Match
	r.Match, r.Error = value.Match, value.Error
	return nil
}

// HTTPLPBSReleaseClient is the default LPBS release client over HTTP.
type HTTPLPBSReleaseClient struct {
	httpClient           *http.Client
	baseURL              string
	resolveBaseURL       func(context.Context) (string, error)
	serviceSecret        string
	resolveServiceSecret func() (string, error)
	tokens               *servicetoken.Tokens
	log                  func(string, map[string]interface{})
}

// LPBSClientConfig options for constructing the LPBS client. Base URL resolves
// from explicit config or scenario discovery. ResolveServiceSecret is an
// isolated test seam; production reads the authority only if the scoped role
// is confirmed unissued.
type LPBSClientConfig struct {
	BaseURL              string
	ServiceSecret        string
	Log                  func(string, map[string]interface{})
	ResolveBaseURL       func(ctx context.Context) (string, error)
	ResolveServiceSecret func() (string, error)
	Tokens               *servicetoken.Tokens
}

// NewHTTPLPBSReleaseClient creates a new HTTP LPBS release client. Explicit
// URLs stay pinned; discovered URLs are resolved for each operation. Legacy
// service credentials are resolved lazily only after the scoped role reports
// unissued.
func NewHTTPLPBSReleaseClient(cfg LPBSClientConfig) (*HTTPLPBSReleaseClient, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = strings.TrimSpace(os.Getenv("LPBS_BASE_URL"))
	}
	var resolveBaseURL func(context.Context) (string, error)
	if base == "" {
		resolveBaseURL = cfg.ResolveBaseURL
		if resolveBaseURL == nil {
			resolveBaseURL = defaultLPBSScenarioURL
		}
	} else {
		base = strings.TrimSpace(base)
	}
	log := cfg.Log
	if log == nil {
		log = func(string, map[string]interface{}) {}
	}
	resolveServiceSecret := cfg.ResolveServiceSecret
	if resolveServiceSecret == nil {
		resolveServiceSecret = defaultLPBSServiceSecret
	}
	secret := strings.TrimSpace(cfg.ServiceSecret)
	tokens := cfg.Tokens
	if tokens == nil {
		tokens = deploymentManagerLPBSTokens()
	}
	return &HTTPLPBSReleaseClient{
		httpClient:           &http.Client{Timeout: 60 * time.Second},
		baseURL:              strings.TrimRight(base, "/"),
		resolveBaseURL:       resolveBaseURL,
		serviceSecret:        secret,
		resolveServiceSecret: resolveServiceSecret,
		tokens:               tokens,
		log:                  log,
	}, nil
}

// authorize prefers the scoped, short-lived service principal. The legacy
// service-secret is accepted only while no credential has been issued for the
// scoped principal; exchange and identity failures must fail closed.
func (c *HTTPLPBSReleaseClient) authorize(ctx context.Context, req *http.Request) error {
	if c.tokens != nil {
		authorization, err := c.tokens.Authorization(ctx)
		if err == nil {
			req.Header.Set("Authorization", authorization)
			return nil
		}
		if !errors.Is(err, servicetoken.ErrNotIssued) || !errors.Is(err, credentialauthority.ErrUnconfigured) {
			return fmt.Errorf("LPBS release service identity: %w", err)
		}
	}
	legacy := strings.TrimSpace(c.serviceSecret)
	if legacy == "" && c.resolveServiceSecret != nil {
		resolved, err := c.resolveServiceSecret()
		if err != nil {
			return fmt.Errorf("LPBS legacy recovery credential: %w", err)
		}
		legacy = strings.TrimSpace(resolved)
	}
	if legacy == "" {
		return fmt.Errorf("LPBS scoped principal is unissued and no legacy recovery credential is available")
	}
	req.Header.Set("Authorization", "Bearer "+legacy)
	return nil
}

func (c *HTTPLPBSReleaseClient) endpoint(ctx context.Context, path string) (string, error) {
	base := strings.TrimSpace(c.baseURL)
	if c.resolveBaseURL != nil {
		resolved, err := c.resolveBaseURL(ctx)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", LPBSScenarioSlug, err)
		}
		base = strings.TrimSpace(resolved)
	}
	validated, err := shared.ValidateServiceURL(base)
	if err != nil {
		return "", fmt.Errorf("%s URL: %w", LPBSScenarioSlug, err)
	}
	return validated + path, nil
}

// lazyLPBSReleaseClient resolves the LPBS owner on first successful use. The
// destination may start after deployment-manager, so a discovery failure at
// process start must not permanently disable the governed release path.
type lazyLPBSReleaseClient struct {
	cfg   LPBSClientConfig
	mu    sync.Mutex
	inner *HTTPLPBSReleaseClient
}

// NewLazyLPBSReleaseClient creates a release client that resolves the
// destination lazily. The first request attempts discovery and credential
// resolution; a later request retries if the destination was not yet running.
func NewLazyLPBSReleaseClient(cfg LPBSClientConfig) *lazyLPBSReleaseClient {
	return &lazyLPBSReleaseClient{cfg: cfg}
}

func (c *lazyLPBSReleaseClient) resolve() (*HTTPLPBSReleaseClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inner != nil {
		return c.inner, nil
	}
	inner, err := NewHTTPLPBSReleaseClient(c.cfg)
	if err != nil {
		return nil, err
	}
	c.inner = inner
	return inner, nil
}

func (c *lazyLPBSReleaseClient) CheckDeployReadiness(ctx context.Context, req *LPBSReadinessRequest) (*LPBSReadinessResult, error) {
	inner, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return inner.CheckDeployReadiness(ctx, req)
}

func (c *lazyLPBSReleaseClient) Verify(ctx context.Context, req *LPBSVerifyRequest) (*LPBSVerifyResult, error) {
	inner, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return inner.Verify(ctx, req)
}

func (c *lazyLPBSReleaseClient) HaltChannel(ctx context.Context, req *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error) {
	inner, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return inner.HaltChannel(ctx, req)
}

func (c *lazyLPBSReleaseClient) RecoverChannel(ctx context.Context, req *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error) {
	inner, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return inner.RecoverChannel(ctx, req)
}

var (
	_ LPBSReleaseClient         = (*lazyLPBSReleaseClient)(nil)
	_ LPBSRecoveryClient        = (*lazyLPBSReleaseClient)(nil)
	_ LPBSChannelRecoveryClient = (*lazyLPBSReleaseClient)(nil)
)

// CheckDeployReadiness calls POST /api/v1/deploy-readiness with service auth.
func (c *HTTPLPBSReleaseClient) CheckDeployReadiness(ctx context.Context, req *LPBSReadinessRequest) (*LPBSReadinessResult, error) {
	endpoint, err := c.endpoint(ctx, "/api/v1/deploy-readiness")
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if err := c.authorize(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read readiness response: %w", err)
	}
	var wire struct {
		Ready *bool           `json:"ready"`
		Gates []ReadinessGate `json:"gates"`
		Error string          `json:"error,omitempty"`
	}
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(respBody, &wire); err != nil {
			return nil, fmt.Errorf("decode readiness response: %w", err)
		}
		if wire.Ready == nil {
			return nil, fmt.Errorf("decode readiness response: missing ready field")
		}
		result := &LPBSReadinessResult{Ready: *wire.Ready, Gates: wire.Gates, Error: wire.Error}
		if result.Ready {
			if len(result.Gates) == 0 {
				return nil, fmt.Errorf("decode readiness response: ready response has no gates")
			}
			for _, gate := range result.Gates {
				if strings.TrimSpace(gate.Name) == "" {
					return nil, fmt.Errorf("decode readiness response: gate name is required")
				}
				if !gate.Ready {
					return nil, fmt.Errorf("decode readiness response: ready response contains failed gate %q", gate.Name)
				}
			}
		}
		return result, nil
	}
	var result LPBSReadinessResult
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			result.Error = fmt.Sprintf("status %d: invalid response: %v", resp.StatusCode, err)
		}
	}
	if result.Error == "" {
		result.Error = fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody))
	}
	result.Ready = false
	return &result, nil
}

// Verify calls GET /api/v1/updates/{app_key}/verify with query params.
func (c *HTTPLPBSReleaseClient) Verify(ctx context.Context, req *LPBSVerifyRequest) (*LPBSVerifyResult, error) {
	if req.AppKey == "" {
		return nil, fmt.Errorf("app_key is required")
	}
	q := url.Values{}
	if req.Channel != "" {
		q.Set("channel", req.Channel)
	}
	if req.Platform != "" {
		q.Set("platform", req.Platform)
	}
	if req.ExpectedVersion != "" {
		q.Set("expected_version", req.ExpectedVersion)
	}
	if req.ExpectedSHA512 != "" {
		q.Set("expected_sha512", req.ExpectedSHA512)
	}
	if req.Deep {
		q.Set("deep", "true")
	}

	endpoint, err := c.endpoint(ctx, fmt.Sprintf("/api/v1/updates/%s/verify?%s", url.PathEscape(req.AppKey), q.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read verify response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return &LPBSVerifyResult{
			AppKey:          req.AppKey,
			Channel:         req.Channel,
			Platform:        req.Platform,
			ExpectedVersion: req.ExpectedVersion,
			Match:           false,
			Error:           fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}
	var result LPBSVerifyResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if result.AppKey != req.AppKey || result.Channel != req.Channel || result.Platform != req.Platform {
		return nil, fmt.Errorf("verify response identity does not match request")
	}
	if req.ExpectedVersion != "" && result.ExpectedVersion != req.ExpectedVersion {
		return nil, fmt.Errorf("verify response expected_version does not match request")
	}
	if req.ExpectedSHA512 != "" && (result.ObservedSHA512 == "" || result.ObservedSHA512 != req.ExpectedSHA512) {
		result.SHA512Match = digestEqual(result.ObservedSHA512, req.ExpectedSHA512)
		if !result.SHA512Match {
			result.Match = false
		}
	}
	return &result, nil
}

func (c *HTTPLPBSReleaseClient) HaltChannel(ctx context.Context, req *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error) {
	if req == nil || strings.TrimSpace(req.AppKey) == "" || req.ExpectedRevision < 0 {
		return nil, fmt.Errorf("app_key and non-negative expected revision are required")
	}
	if !req.DryRun && strings.TrimSpace(req.Confirmation) == "" {
		return nil, fmt.Errorf("confirmation is required for channel halt")
	}
	body, err := json.Marshal(map[string]interface{}{
		"app_key": req.AppKey, "variant_key": req.VariantKey, "expected_revision": req.ExpectedRevision,
		"halted": req.Halted, "reason": req.Reason, "dry_run": req.DryRun,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal channel halt: %w", err)
	}
	endpoint, err := c.endpoint(ctx, "/api/v1/admin/download-channels/halt")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build channel halt request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if err := c.authorize(ctx, httpReq); err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("channel halt request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read channel halt response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("channel halt failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var receipt LPBSRecoveryReceipt
	if err := json.Unmarshal(respBody, &receipt); err != nil {
		return nil, fmt.Errorf("decode channel halt receipt: %w", err)
	}
	if receipt.AppKey != req.AppKey || receipt.VariantKey != req.VariantKey || receipt.Revision != req.ExpectedRevision || receipt.Halted != req.Halted {
		return nil, fmt.Errorf("channel halt receipt does not match request")
	}
	if !req.DryRun && (receipt.Outcome != "halted" || receipt.Health != "stopped" || receipt.ExternalReceipt == "" || receipt.ObservedAt.IsZero()) {
		return nil, fmt.Errorf("channel halt receipt does not prove a stopped owner effect")
	}
	return &receipt, nil
}

func (c *HTTPLPBSReleaseClient) RecoverChannel(ctx context.Context, req *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error) {
	if req == nil || strings.TrimSpace(req.AppKey) == "" || req.ExpectedRevision < 0 || strings.TrimSpace(req.Action) == "" {
		return nil, fmt.Errorf("app_key, action, and non-negative expected revision are required")
	}
	if !req.DryRun && strings.TrimSpace(req.Confirmation) == "" {
		return nil, fmt.Errorf("confirmation is required for channel recovery")
	}
	body, err := json.Marshal(map[string]interface{}{
		"app_key": req.AppKey, "variant_key": req.VariantKey, "expected_revision": req.ExpectedRevision,
		"expected_predecessor_revision": req.ExpectedPredecessor, "action": req.Action,
		"data_compatibility": req.DataCompatibility, "artifact_ids": req.ArtifactIDs,
		"candidate_id": req.CandidateID, "destination_revision_id": req.DestinationRevisionID,
		"reason": req.Reason, "dry_run": req.DryRun,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal channel recovery: %w", err)
	}
	endpoint, err := c.endpoint(ctx, "/api/v1/admin/download-channels/recover")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build channel recovery request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if err := c.authorize(ctx, httpReq); err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("channel recovery request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read channel recovery response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("channel recovery failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var receipt LPBSRecoveryReceipt
	if err := json.Unmarshal(respBody, &receipt); err != nil {
		return nil, fmt.Errorf("decode channel recovery receipt: %w", err)
	}
	if receipt.AppKey != req.AppKey || receipt.VariantKey != req.VariantKey || receipt.Action != req.Action || receipt.Revision < req.ExpectedRevision || (req.CandidateID != "" && receipt.CandidateID != req.CandidateID) || (req.DestinationRevisionID != "" && receipt.DestinationRevisionID != req.DestinationRevisionID) {
		return nil, fmt.Errorf("channel recovery receipt does not match request")
	}
	if !req.DryRun && (receipt.ExternalReceipt == "" || receipt.ObservedAt.IsZero()) {
		return nil, fmt.Errorf("channel recovery receipt does not prove an owner effect")
	}
	return &receipt, nil
}

func lpbsVariantKey(channel string) string {
	if strings.TrimSpace(channel) == "" || strings.EqualFold(strings.TrimSpace(channel), "stable") {
		return "default"
	}
	return strings.TrimSpace(channel)
}

func parseChannelRevision(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}

// digestEqual accepts the hex form used by the desktop uploader and the
// base64 form emitted by the hosted deep verifier. Digest encoding is a wire
// representation detail; publication identity is the decoded SHA-512 value.
func digestEqual(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1 {
		return true
	}
	leftBytes, leftErr := decodeDigest(left)
	rightBytes, rightErr := decodeDigest(right)
	return leftErr == nil && rightErr == nil && subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}

func decodeDigest(value string) ([]byte, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "sha512:"))
	if decoded, err := hex.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(value)
}
