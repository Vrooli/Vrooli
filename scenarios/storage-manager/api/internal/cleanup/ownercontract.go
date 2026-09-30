package cleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"
)

// OwnerEstimateRequest is the wire request served by an owner scenario.
type OwnerEstimateRequest struct {
	MinAgeSeconds int64 `json:"min_age_seconds"`
	MaxBytes      int64 `json:"max_bytes,omitempty"`
	KeepCount     int   `json:"keep_count,omitempty"`
}

type OwnerEstimateResponse struct {
	ProviderID     string    `json:"provider_id"`
	EstimatedBytes int64     `json:"estimated_bytes"`
	ItemCount      int       `json:"item_count"`
	BlockedReason  string    `json:"blocked_reason,omitempty"`
	ObservedAt     time.Time `json:"observed_at"`
	MinAgeSeconds  int64     `json:"min_age_seconds,omitempty"`
	KeepCount      int       `json:"keep_count,omitempty"`
	MaxBytes       int64     `json:"max_bytes,omitempty"`
}

type OwnerPreviewRequest struct {
	Estimate OwnerEstimateResponse `json:"estimate"`
}

type OwnerPreviewItem struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	AgeSeconds int64  `json:"age_seconds"`
	Protected  bool   `json:"protected"`
}

type OwnerPreviewResponse struct {
	ProviderID    string             `json:"provider_id"`
	Items         []OwnerPreviewItem `json:"items"`
	BlockedReason string             `json:"blocked_reason,omitempty"`
	Warnings      []string           `json:"warnings,omitempty"`
	MinAgeSeconds int64              `json:"min_age_seconds,omitempty"`
	KeepCount     int                `json:"keep_count,omitempty"`
	MaxBytes      int64              `json:"max_bytes,omitempty"`
}

type OwnerApplyRequest struct {
	ProviderID     string               `json:"provider_id"`
	Preview        OwnerPreviewResponse `json:"preview"`
	IdempotencyKey string               `json:"idempotency_key"`
	ApprovalMode   ApprovalMode         `json:"approval_mode"`
}

type OwnerApplyResponse struct {
	ReclaimedBytes int64    `json:"reclaimed_bytes"`
	RemovedItemIDs []string `json:"removed_item_ids"`
	SkippedItemIDs []string `json:"skipped_item_ids"`
	Warnings       []string `json:"warnings,omitempty"`
	AlreadyDone    bool     `json:"already_done,omitempty"`
}

// HTTPScenarioProviderClient is the storage-manager transport for owner
// cleanup. Owner scenarios retain deletion authority; this client only sends
// the shared estimate/preview/apply contract.
type HTTPScenarioProviderClient struct {
	ResolveURL func(context.Context, string) (string, error)
	HTTPClient *http.Client
}

func (c *HTTPScenarioProviderClient) client(timeout time.Duration) *http.Client {
	base := http.DefaultClient
	if c != nil && c.HTTPClient != nil {
		base = c.HTTPClient
	}
	if base.Timeout != 0 || timeout == 0 {
		return base
	}
	client := *base
	client.Timeout = timeout
	return &client
}

func (c *HTTPScenarioProviderClient) endpoint(ctx context.Context, scenario, path string) (string, error) {
	if c == nil || c.ResolveURL == nil {
		return "", fmt.Errorf("owner scenario client unavailable")
	}
	base, err := c.ResolveURL(ctx, scenario)
	if err != nil {
		return "", fmt.Errorf("owner scenario unreachable: %w", err)
	}
	return strings.TrimRight(base, "/") + path, nil
}

func (c *HTTPScenarioProviderClient) request(ctx context.Context, method, url string, body any, out any, timeout time.Duration) (int, error) {
	return c.requestWithHeaders(ctx, method, url, body, out, timeout, nil)
}

func (c *HTTPScenarioProviderClient) requestWithHeaders(ctx context.Context, method, url string, body any, out any, timeout time.Duration, headers http.Header) (int, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = strings.NewReader(string(payload))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	resp, err := c.client(timeout).Do(req)
	if err != nil {
		return 0, fmt.Errorf("owner scenario unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("owner cleanup returned HTTP %d", resp.StatusCode)
	}
	if out != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func (c *HTTPScenarioProviderClient) Estimate(ctx context.Context, scenario string, policy ProviderPolicy) (Estimate, error) {
	url, err := c.endpoint(ctx, scenario, "/api/v1/cleanup/estimate")
	if err != nil {
		return Estimate{BlockedReason: blockedReason(err)}, nil
	}
	parsed, parseErr := urlpkg.Parse(url)
	if parseErr == nil {
		query := parsed.Query()
		query.Set("min_age_seconds", strconv.FormatInt(int64(policy.MinAge/time.Second), 10))
		if policy.MaxBytes > 0 {
			query.Set("max_bytes", strconv.FormatInt(policy.MaxBytes, 10))
		}
		parsed.RawQuery = query.Encode()
		url = parsed.String()
	}
	var response OwnerEstimateResponse
	status, err := c.requestWithHeaders(ctx, http.MethodGet, url, nil, &response, 30*time.Second, http.Header{"X-Vrooli-Recovery-Only": []string{"true"}})
	if err != nil {
		if status == http.StatusNotFound {
			return Estimate{BlockedReason: "owner scenario does not implement cleanup"}, nil
		}
		return Estimate{BlockedReason: blockedReason(err)}, nil
	}
	return Estimate{ProviderID: response.ProviderID, EstimatedBytes: response.EstimatedBytes, ItemCount: response.ItemCount, BlockedReason: response.BlockedReason, ObservedAt: response.ObservedAt, RequiresApproval: policy.ApprovalMode != ApprovalModeNone, MinAge: time.Duration(response.MinAgeSeconds) * time.Second, KeepCount: response.KeepCount, MaxBytes: response.MaxBytes}, nil
}

func (c *HTTPScenarioProviderClient) Preview(ctx context.Context, scenario string, estimate Estimate) (Preview, error) {
	url, err := c.endpoint(ctx, scenario, "/api/v1/cleanup/preview")
	if err != nil {
		return Preview{BlockedReason: blockedReason(err)}, nil
	}
	var response OwnerPreviewResponse
	_, err = c.requestWithHeaders(ctx, http.MethodPost, url, OwnerPreviewRequest{Estimate: OwnerEstimateResponse{ProviderID: estimate.ProviderID, EstimatedBytes: estimate.EstimatedBytes, ItemCount: estimate.ItemCount, BlockedReason: estimate.BlockedReason, ObservedAt: estimate.ObservedAt, MinAgeSeconds: int64(estimate.MinAge / time.Second), KeepCount: estimate.KeepCount, MaxBytes: estimate.MaxBytes}}, &response, 30*time.Second, http.Header{"X-Vrooli-Recovery-Only": []string{"true"}})
	if err != nil {
		return Preview{BlockedReason: blockedReason(err)}, nil
	}
	out := Preview{
		ProviderID: response.ProviderID, BlockedReason: response.BlockedReason, Warnings: response.Warnings,
		MinAge: time.Duration(response.MinAgeSeconds) * time.Second, KeepCount: response.KeepCount, MaxBytes: response.MaxBytes,
	}
	for _, item := range response.Items {
		if item.Protected {
			continue
		}
		out.Items = append(out.Items, PreviewItem{ID: item.ID, Path: item.Path, Bytes: item.Bytes, Action: "owner cleanup", SafetyTier: SafetyTierSafeWithOwner})
	}
	return out, nil
}

func (c *HTTPScenarioProviderClient) Apply(ctx context.Context, req ScenarioCleanupRequest) (ApplyResult, error) {
	url, err := c.endpoint(ctx, req.ScenarioID, "/api/v1/cleanup/apply")
	if err != nil {
		return ApplyResult{}, err
	}
	wirePreview := OwnerPreviewResponse{
		ProviderID: req.ProviderID, MinAgeSeconds: int64(req.Preview.MinAge / time.Second),
		KeepCount: req.Preview.KeepCount, MaxBytes: req.Preview.MaxBytes,
	}
	for _, item := range req.Preview.Items {
		wirePreview.Items = append(wirePreview.Items, OwnerPreviewItem{ID: item.ID, Path: item.Path, Bytes: item.Bytes})
	}
	var response OwnerApplyResponse
	_, err = c.requestWithHeaders(ctx, http.MethodPost, url, OwnerApplyRequest{ProviderID: req.ProviderID, Preview: wirePreview, IdempotencyKey: req.IdempotencyKey, ApprovalMode: req.ApprovalMode}, &response, 10*time.Minute, http.Header{"X-Vrooli-Recovery-Lock": []string{"held-by-storage-manager"}, "X-Vrooli-Recovery-Only": []string{"true"}})
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{ProviderID: req.ProviderID, Applied: response.ReclaimedBytes > 0 || len(response.RemovedItemIDs) > 0, AlreadyDone: response.AlreadyDone, AppliedItems: response.RemovedItemIDs, SkippedItems: response.SkippedItemIDs, ReclaimedBytes: response.ReclaimedBytes, Warnings: response.Warnings}, nil
}

func blockedReason(err error) string {
	message := err.Error()
	if strings.Contains(message, "client unavailable") {
		return "owner scenario client unavailable"
	}
	if strings.Contains(message, "does not implement cleanup") {
		return "owner scenario does not implement cleanup"
	}
	return "owner scenario unreachable"
}

// BlockedReasonForError maps transport failures to the operator-facing reason
// used by owner-delegated providers. It deliberately leaves domain/provider
// errors to their original callers.
func BlockedReasonForError(err error) string {
	if err == nil {
		return ""
	}
	return blockedReason(err)
}
