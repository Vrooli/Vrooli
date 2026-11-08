package providers

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

const (
	DeviceKeyIdentity = "vrooli/ai-device"
	DeviceKeyField    = "key"
)

var ErrDeviceKeyGenerationChanged = errors.New("hosted device credential generation changed; recovery fails closed")

type (
	sourceContextKey struct{}
	gatewayHopKey    struct{}
)

// WithSource restricts a request to the user's chosen source. It cannot grant
// provider or billing authority; the selected source still verifies both.
func WithSource(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, sourceContextKey{}, source)
}

func Source(ctx context.Context) string {
	value, _ := ctx.Value(sourceContextKey{}).(string)
	return value
}

func WithGatewayHop(ctx context.Context, hop string) context.Context {
	return context.WithValue(ctx, gatewayHopKey{}, hop)
}

type HostedDoorClient struct {
	BaseURL          string
	HTTPClient       *http.Client
	ResolveDeviceKey func(context.Context) (string, error)
}

func (c *HostedDoorClient) request(ctx context.Context, method, path, key, idempotencyKey string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+"/api/v1/ai"+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vrooli-Gateway-Hop", "hosted-door")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	client := http.Client{Timeout: 15 * time.Minute}
	if c.HTTPClient != nil {
		client = *c.HTTPClient
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if path == "/receipt" && response.StatusCode == http.StatusNotFound {
			return response, nil
		}
		response.Body.Close()
		if response.StatusCode == 402 {
			return nil, ErrInsufficientCredits
		}
		return nil, fmt.Errorf("hosted door refused (HTTP %d)", response.StatusCode)
	}
	return response, nil
}

func deviceKeyGeneration(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:])
}

// DeviceKeyGeneration returns only a one-way fingerprint of the current
// Gateway-owned device credential. The credential itself is never persisted.
func (c *HostedDoorClient) DeviceKeyGeneration(ctx context.Context) (string, error) {
	resolve := c.ResolveDeviceKey
	if resolve == nil {
		resolve = func(ctx context.Context) (string, error) {
			return resolveProviderKey(ctx, DeviceKeyIdentity, DeviceKeyField)
		}
	}
	key, err := resolve(ctx)
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", errors.New("hosted device key is not configured")
	}
	return deviceKeyGeneration(key), nil
}

func recoveryState(status string) inferencev1.RecoveryReceiptState {
	switch status {
	case "pending":
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_PENDING
	case "settled":
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED
	case "released":
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_RELEASED
	default:
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN
	}
}

func safeAccountingStatus(status string) string {
	switch status {
	case "pending", "settled", "released", "unknown":
		return status
	default:
		return "unknown"
	}
}

func (c *HostedDoorClient) receipt(ctx context.Context, key, attemptKey string) (*inferencev1.RecoveryReceipt, bool, error) {
	response, err := c.request(ctx, http.MethodGet, "/receipt", key, attemptKey, nil)
	if err != nil {
		return nil, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	var safe struct {
		Status         string `json:"status"`
		ChargedCredits *int64 `json:"charged_credits"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&safe); err != nil {
		return nil, false, err
	}
	status := safeAccountingStatus(safe.Status)
	if safe.ChargedCredits != nil && *safe.ChargedCredits < 0 {
		safe.ChargedCredits = nil
	}
	digest := sha256.Sum256([]byte(attemptKey))
	return &inferencev1.RecoveryReceipt{
		AttemptReference: hex.EncodeToString(digest[:8]),
		State:            recoveryState(status), AccountingStatus: status,
		ChargedCredits: safe.ChargedCredits,
	}, true, nil
}

func uncertainResult(attemptKey string, receipt *inferencev1.RecoveryReceipt) ExecutionResult {
	if receipt == nil {
		digest := sha256.Sum256([]byte(attemptKey))
		receipt = &inferencev1.RecoveryReceipt{AttemptReference: hex.EncodeToString(digest[:8]), State: inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN}
	}
	return ExecutionResult{Recovery: receipt}
}

func (c *HostedDoorClient) Run(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if hop, _ := ctx.Value(gatewayHopKey{}).(string); hop != "" {
		return ExecutionResult{}, errors.New("gateway forwarding loop refused")
	}
	if len(req.Attachments) > 0 || req.SpendLimit != nil {
		return ExecutionResult{}, errors.New("hosted door request shape requires its feature contract")
	}
	resolve := c.ResolveDeviceKey
	if resolve == nil {
		resolve = func(ctx context.Context) (string, error) {
			return resolveProviderKey(ctx, DeviceKeyIdentity, DeviceKeyField)
		}
	}
	key, err := resolve(ctx)
	if err != nil {
		return ExecutionResult{}, err
	}
	if key == "" {
		return ExecutionResult{}, errors.New("hosted device key is not configured")
	}
	generation := deviceKeyGeneration(key)
	if req.ExpectedKeyGeneration != "" && req.ExpectedKeyGeneration != generation {
		return ExecutionResult{}, ErrDeviceKeyGenerationChanged
	}
	if req.IdempotencyKey == "" {
		return ExecutionResult{}, errors.New("hosted inference requires a durable idempotency key")
	}
	if req.RecoveryOnly {
		receipt, found, err := c.receipt(ctx, key, req.IdempotencyKey)
		if err != nil || !found {
			return uncertainResult(req.IdempotencyKey, receipt), nil
		}
		return uncertainResult(req.IdempotencyKey, receipt), nil
	}
	prior, found, err := c.receipt(ctx, key, req.IdempotencyKey)
	if err != nil {
		return ExecutionResult{}, err // No provider POST has occurred.
	}
	if found {
		return uncertainResult(req.IdempotencyKey, prior), nil
	}
	response, err := c.request(ctx, "GET", "/models", key, "", nil)
	if err != nil {
		return ExecutionResult{}, err
	}
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&models)
	response.Body.Close()
	if err != nil {
		return ExecutionResult{}, err
	}
	if len(models.Data) == 0 {
		return ExecutionResult{}, errors.New("no eligible hosted model")
	}
	body := map[string]any{"model": models.Data[0].ID, "messages": []map[string]string{{"role": "user", "content": req.InputText}}, "stream": true}
	if req.MaxOutputTokens > 0 {
		body["max_tokens"] = req.MaxOutputTokens
	}
	response, err = c.request(ctx, "POST", "/chat/completions", key, req.IdempotencyKey, body)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) {
			return ExecutionResult{}, err // LPBS refused admission before provider dispatch.
		}
		receipt, _, receiptErr := c.receipt(ctx, key, req.IdempotencyKey)
		if receiptErr != nil {
			receipt = nil
		}
		return uncertainResult(req.IdempotencyKey, receipt), nil
	}
	defer response.Body.Close()
	var content strings.Builder
	var prompt, completion int64
	usageObserved := false
	scanner := bufio.NewScanner(io.LimitReader(response.Body, 16<<20))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	done := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			done = true
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				Prompt     int64 `json:"prompt_tokens"`
				Completion int64 `json:"completion_tokens"`
			} `json:"usage"`
			Error json.RawMessage `json:"error"`
		}
		if err = json.Unmarshal([]byte(payload), &chunk); err != nil {
			receipt, _, receiptErr := c.receipt(ctx, key, req.IdempotencyKey)
			if receiptErr != nil {
				receipt = nil
			}
			return uncertainResult(req.IdempotencyKey, receipt), nil
		}
		if len(chunk.Error) > 0 {
			receipt, _, receiptErr := c.receipt(ctx, key, req.IdempotencyKey)
			if receiptErr != nil {
				receipt = nil
			}
			return uncertainResult(req.IdempotencyKey, receipt), nil
		}
		for _, choice := range chunk.Choices {
			content.WriteString(choice.Delta.Content)
		}
		if chunk.Usage != nil {
			usageObserved = true
			prompt, completion = chunk.Usage.Prompt, chunk.Usage.Completion
		}
	}
	if err = scanner.Err(); err != nil {
		receipt, _, receiptErr := c.receipt(ctx, key, req.IdempotencyKey)
		if receiptErr != nil {
			receipt = nil
		}
		return uncertainResult(req.IdempotencyKey, receipt), nil
	}
	if !done || content.Len() == 0 {
		receipt, _, receiptErr := c.receipt(ctx, key, req.IdempotencyKey)
		if receiptErr != nil {
			receipt = nil
		}
		return uncertainResult(req.IdempotencyKey, receipt), nil
	}
	result := map[string]any{"content": content.String()}
	if usageObserved {
		result["prompt_tokens"] = prompt
		result["completion_tokens"] = completion
	}
	data, err := json.Marshal(result)
	return ExecutionResult{OutputText: string(data)}, err
}
