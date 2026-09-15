package deployments

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

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
	httpClient    *http.Client
	baseURL       string
	serviceSecret string
	log           func(string, map[string]interface{})
}

// LPBSClientConfig options for constructing the LPBS client. If BaseURL is
// empty, LPBS_BASE_URL env var is used. If ServiceSecret is empty,
// LPBS_SERVICE_SECRET env var is used.
type LPBSClientConfig struct {
	BaseURL       string
	ServiceSecret string
	Log           func(string, map[string]interface{})
}

// NewHTTPLPBSReleaseClient creates a new HTTP LPBS release client.
// Logs a warning if LPBS_SERVICE_SECRET is missing so misconfig is visible.
func NewHTTPLPBSReleaseClient(cfg LPBSClientConfig) (*HTTPLPBSReleaseClient, error) {
	base := cfg.BaseURL
	if strings.TrimSpace(base) == "" {
		base = strings.TrimSpace(os.Getenv("LPBS_BASE_URL"))
	}
	if base == "" {
		return nil, fmt.Errorf("LPBS base URL not configured (set LPBS_BASE_URL or pass BaseURL)")
	}
	secret := cfg.ServiceSecret
	if strings.TrimSpace(secret) == "" {
		secret = strings.TrimSpace(os.Getenv("LPBS_SERVICE_SECRET"))
	}
	log := cfg.Log
	if log == nil {
		log = func(string, map[string]interface{}) {}
	}
	if secret == "" {
		log("warn", map[string]interface{}{
			"msg": "LPBS_SERVICE_SECRET is not set; deploy-readiness checks will fail until the secret is configured",
		})
	}
	return &HTTPLPBSReleaseClient{
		httpClient:    &http.Client{Timeout: 60 * time.Second},
		baseURL:       strings.TrimRight(base, "/"),
		serviceSecret: secret,
		log:           log,
	}, nil
}

// CheckDeployReadiness calls POST /api/v1/deploy-readiness with service auth.
func (c *HTTPLPBSReleaseClient) CheckDeployReadiness(ctx context.Context, req *LPBSReadinessRequest) (*LPBSReadinessResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/deploy-readiness", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceSecret != "" {
		// LPBS requireAdminOrService accepts a Bearer service token; same
		// secret pattern S2D uses for inter-scenario calls.
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceSecret)
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

	endpoint := fmt.Sprintf("%s/api/v1/updates/%s/verify?%s",
		c.baseURL, url.PathEscape(req.AppKey), q.Encode())
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
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/download-channels/halt", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build channel halt request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceSecret != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceSecret)
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
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/download-channels/recover", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build channel recovery request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceSecret != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceSecret)
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
