package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DeviceKeyIdentity = "vrooli/ai-device"
	DeviceKeyField    = "key"
)

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

func (c *HostedDoorClient) request(ctx context.Context, method, path, key string, body any) (*http.Response, error) {
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
		response.Body.Close()
		if response.StatusCode == 402 {
			return nil, ErrInsufficientCredits
		}
		return nil, fmt.Errorf("hosted door refused (HTTP %d)", response.StatusCode)
	}
	return response, nil
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
	response, err := c.request(ctx, "GET", "/models", key, nil)
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
	response, err = c.request(ctx, "POST", "/chat/completions", key, body)
	if err != nil {
		return ExecutionResult{}, err
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
			return ExecutionResult{}, err
		}
		if len(chunk.Error) > 0 {
			return ExecutionResult{}, errors.New("hosted door stream failed")
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
		return ExecutionResult{}, err
	}
	if !done || content.Len() == 0 {
		return ExecutionResult{}, errors.New("hosted door stream ended without a settled completion")
	}
	result := map[string]any{"content": content.String()}
	if usageObserved {
		result["prompt_tokens"] = prompt
		result["completion_tokens"] = completion
	}
	data, err := json.Marshal(result)
	return ExecutionResult{OutputText: string(data)}, err
}
