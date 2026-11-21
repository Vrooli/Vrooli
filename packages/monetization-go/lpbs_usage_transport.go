package monetization

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const usageReportTimeout = 15 * time.Second

// LPBSAccessResolver obtains a short-lived consumer access token for the
// configured authority. Implementations must never persist or log the token.
type LPBSAccessResolver func(context.Context, string) (string, error)

// LPBSUsageTransport reports immutable outbox events to the LPBS usage API.
// Retry and durable delivery state remain owned by Outbox.
type LPBSUsageTransport struct {
	BaseURL      string
	ResolveToken LPBSAccessResolver
	Client       *http.Client
}

// NewLPBSUsageTransport creates the shared transport used by scenario outboxes.
func NewLPBSUsageTransport(baseURL string, resolve LPBSAccessResolver, client *http.Client) *LPBSUsageTransport {
	if client == nil {
		client = &http.Client{Timeout: usageReportTimeout}
	}
	return &LPBSUsageTransport{BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), ResolveToken: resolve, Client: client}
}

type lpbsUsageRequest struct {
	UserIdentity string            `json:"user_identity"`
	LimitKey     string            `json:"limit_key"`
	UsageAmount  int64             `json:"usage_amount"`
	Amount       int64             `json:"amount"`
	AppBundleKey string            `json:"app_bundle_key"`
	OperationID  string            `json:"operation_id"`
	Metadata     lpbsUsageMetadata `json:"metadata,omitempty"`
}

type lpbsUsageMetadata struct {
	Operation    string `json:"operation,omitempty"`
	Model        string `json:"model,omitempty"`
	PromptTokens int    `json:"prompt_tokens,omitempty"`
	IsBYOK       bool   `json:"is_byok,omitempty"`
}

// Report sends one usage event with the same idempotency key on every retry.
func (t *LPBSUsageTransport) Report(ctx context.Context, usage Usage) error {
	if t == nil || t.ResolveToken == nil || t.BaseURL == "" {
		return errors.New("LPBS usage transport is unavailable")
	}
	if err := validateUsage(usage); err != nil {
		return err
	}
	token, err := t.ResolveToken(ctx, t.BaseURL)
	if err != nil {
		return fmt.Errorf("resolve LPBS access: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("LPBS consumer access token is unavailable")
	}
	metadata := usage.Metadata
	promptTokens, err := parseUsageInt(metadata["prompt_tokens"])
	if err != nil {
		return fmt.Errorf("usage metadata prompt_tokens is invalid: %w", err)
	}
	byok, err := parseUsageBool(metadata["is_byok"])
	if err != nil {
		return fmt.Errorf("usage metadata is_byok is invalid: %w", err)
	}
	payload := lpbsUsageRequest{
		UserIdentity: usage.UserIdentity,
		LimitKey:     usage.MeterKey,
		UsageAmount:  usage.Units,
		Amount:       usage.Units,
		AppBundleKey: usage.AppKey,
		OperationID:  usage.OperationID,
		Metadata: lpbsUsageMetadata{
			Operation:    metadata["operation"],
			Model:        metadata["model"],
			PromptTokens: promptTokens,
			IsBYOK:       byok,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal LPBS usage report: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.BaseURL+"/api/v1/usage/report", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create LPBS usage request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := t.Client.Do(request)
	if err != nil {
		return fmt.Errorf("send LPBS usage report: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("LPBS usage report returned status %d", response.StatusCode)
	}
	return nil
}

func parseUsageInt(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.Atoi(strings.TrimSpace(value))
}

func parseUsageBool(value string) (bool, error) {
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	return strconv.ParseBool(strings.TrimSpace(value))
}
