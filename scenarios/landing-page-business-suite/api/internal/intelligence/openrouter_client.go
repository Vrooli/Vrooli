package intelligence

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// httpOpenRouterClient implements OpenRouterClient using HTTP.
type httpOpenRouterClient struct {
	apiKey     string
	baseURL    string
	referer    string
	title      string
	httpClient *http.Client
	log        func(event string, fields map[string]interface{})
}

// NewOpenRouterClient creates a new HTTP-based OpenRouter client.
func NewOpenRouterClient(opts OpenRouterClientOptions) OpenRouterClient {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := opts.HTTPClient
	if httpClient == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 120 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	logger := opts.Logger
	if logger == nil {
		logger = func(string, map[string]interface{}) {}
	}

	return &httpOpenRouterClient{
		apiKey:     opts.APIKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		log:        logger,
		referer:    defaultString(opts.Referer, "https://vrooli.com"),
		title:      defaultString(opts.Title, "Vrooli AI Metered Inference"),
	}
}

// Chat implements OpenRouterClient.
func (c *httpOpenRouterClient) Chat(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
	req.Stream = false
	req.Usage = &OpenRouterUsageRequest{Include: true}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.log("openrouter_error", map[string]interface{}{
			"level":  "error",
			"status": resp.StatusCode,
			"body":   string(bodyBytes),
		})
		return nil, fmt.Errorf("%w: status %d", ErrProvider, resp.StatusCode)
	}

	// Parse the OpenRouter response format
	var rawResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int      `json:"prompt_tokens"`
			CompletionTokens int      `json:"completion_tokens"`
			TotalTokens      int      `json:"total_tokens"`
			Cost             *float64 `json:"cost"`
			CostDetails      struct {
				UpstreamInferenceCost *float64 `json:"upstream_inference_cost"`
			} `json:"cost_details"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Extract content from first choice
	content := ""
	finishReason := ""
	if len(rawResp.Choices) > 0 {
		content = rawResp.Choices[0].Message.Content
		finishReason = rawResp.Choices[0].FinishReason
	}

	return &OpenRouterChatResponse{
		ID:           rawResp.ID,
		Model:        rawResp.Model,
		Content:      content,
		FinishReason: finishReason,
		Usage: OpenRouterUsage{
			PromptTokens:     rawResp.Usage.PromptTokens,
			CompletionTokens: rawResp.Usage.CompletionTokens,
			TotalTokens:      rawResp.Usage.TotalTokens,
			CostMicros:       reportedCostMicros(rawResp.Usage.Cost, rawResp.Usage.CostDetails.UpstreamInferenceCost),
			CostReported:     rawResp.Usage.Cost != nil || rawResp.Usage.CostDetails.UpstreamInferenceCost != nil,
		},
	}, nil
}

// ChatStream implements OpenRouterClient.
func (c *httpOpenRouterClient) ChatStream(ctx context.Context, req OpenRouterChatRequest, onChunk func(content string)) (*OpenRouterUsage, error) {
	req.Stream = true
	req.Usage = &OpenRouterUsageRequest{Include: true}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, string(bodyBytes))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024) // Handle long lines

	var usage OpenRouterUsage
	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data line
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			break
		}

		// Parse the chunk
		var chunk struct {
			ID      string `json:"id"`
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int      `json:"prompt_tokens"`
				CompletionTokens int      `json:"completion_tokens"`
				TotalTokens      int      `json:"total_tokens"`
				Cost             *float64 `json:"cost"`
				CostDetails      struct {
					UpstreamInferenceCost *float64 `json:"upstream_inference_cost"`
				} `json:"cost_details"`
			} `json:"usage,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			c.log("stream_chunk_parse_error", map[string]interface{}{
				"level": "warn",
				"error": err.Error(),
				"data":  data,
			})
			continue
		}

		// Extract content delta
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullContent.WriteString(content)
			if onChunk != nil {
				onChunk(content)
			}
		}

		// Capture usage if present (usually in the last chunk)
		if chunk.Usage != nil {
			usage = OpenRouterUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
				CostMicros:       reportedCostMicros(chunk.Usage.Cost, chunk.Usage.CostDetails.UpstreamInferenceCost),
				CostReported:     chunk.Usage.Cost != nil || chunk.Usage.CostDetails.UpstreamInferenceCost != nil,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		c.log("stream_scanner_error", map[string]interface{}{
			"level": "error",
			"error": err.Error(),
		})
	}

	// Estimate tokens if not provided in stream
	if usage.PromptTokens == 0 {
		// Can't reliably estimate prompt tokens from stream
		usage.PromptTokens = 0
	}
	if usage.CompletionTokens == 0 {
		// Rough estimate: 4 chars per token
		usage.CompletionTokens = len(fullContent.String()) / 4
	}
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

	return &usage, nil
}

func dollarsToMicros(cost, upstream float64) int64 {
	if cost == 0 {
		cost = upstream
	}
	if cost < 0 || math.IsNaN(cost) || math.IsInf(cost, 0) || cost > float64(math.MaxInt64)/1_000_000 {
		return -1
	}
	if cost == 0 {
		return 0
	}
	return int64(cost*1_000_000 + 0.5)
}

func reportedCostMicros(cost, upstream *float64) int64 {
	// OpenRouter may report a zero aggregate cost while supplying the actual
	// upstream inference cost separately. Preserve the historical preference
	// for that populated value, while still treating a lone explicit zero as
	// reported (the caller uses pointer presence for that distinction).
	if cost != nil && *cost == 0 && upstream != nil && *upstream > 0 {
		return dollarsToMicros(*upstream, 0)
	}
	if cost != nil {
		return dollarsToMicros(*cost, 0)
	}
	if upstream != nil {
		return dollarsToMicros(*upstream, 0)
	}
	return 0
}

// VerifyAPIKey implements OpenRouterClient.
func (c *httpOpenRouterClient) VerifyAPIKey(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/auth/key", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("key verification failed: status %d", resp.StatusCode)
	}

	return nil
}

func (c *httpOpenRouterClient) CurrentEndpoints(ctx context.Context, model string) ([]OpenRouterEndpointMetadata, error) {
	if !strings.Contains(model, "/") || strings.Contains(model, "..") {
		return nil, fmt.Errorf("invalid model for endpoint metadata")
	}
	url := c.baseURL + "/api/v1/models/" + strings.Trim(model, "/") + "/endpoints"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create endpoint metadata request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch current endpoint metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint metadata returned status %d", resp.StatusCode)
	}
	var raw struct {
		Data struct {
			Endpoints []struct {
				Provider string          `json:"provider_name"`
				Context  int             `json:"context_length"`
				Pricing  json.RawMessage `json:"pricing"`
			} `json:"endpoints"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode endpoint metadata: %w", err)
	}
	parsed := make([]OpenRouterEndpointMetadata, 0, len(raw.Data.Endpoints))
	for _, item := range raw.Data.Endpoints {
		var prices map[string]json.RawMessage
		if json.Unmarshal(item.Pricing, &prices) != nil || len(prices) < 2 {
			continue
		}
		known := map[string]bool{"prompt": true, "completion": true, "request": true, "image": true, "input_cache_read": true, "discount": true}
		unsupportedFee := false
		for key := range prices {
			if !known[key] {
				unsupportedFee = true
				break
			}
		}
		if unsupportedFee {
			continue
		}
		p, ok1 := parseEndpointPrice(prices["prompt"])
		cst, ok2 := parseEndpointPrice(prices["completion"])
		r, requestKnown, ok3 := optionalEndpointPrice(prices, "request")
		i, imageKnown, ok4 := optionalEndpointPrice(prices, "image")
		cacheRead, _, ok5 := optionalEndpointPrice(prices, "input_cache_read")
		_, _, discountOK := optionalEndpointPrice(prices, "discount")
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !discountOK || cacheRead > p {
			continue
		}
		parsed = append(parsed, OpenRouterEndpointMetadata{Provider: strings.TrimSpace(item.Provider), ContextTokens: item.Context, PromptPrice: p, CompletionPrice: cst, RequestPrice: r, ImagePrice: i, RequestPriceKnown: requestKnown, ImagePriceKnown: imageKnown})
	}
	return parsed, nil
}

func optionalEndpointPrice(prices map[string]json.RawMessage, key string) (float64, bool, bool) {
	raw, exists := prices[key]
	if !exists {
		return 0, false, true
	}
	price, ok := parseEndpointPrice(raw)
	return price, ok, ok
}

func parseEndpointPrice(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, n >= 0 && !math.IsNaN(n) && !math.IsInf(n, 0)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return n, err == nil && n >= 0 && !math.IsNaN(n) && !math.IsInf(n, 0)
}

func (c *httpOpenRouterClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", c.referer)
	req.Header.Set("X-Title", c.title)
}

func defaultString(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

// Compile-time interface checks
var _ OpenRouterClient = (*httpOpenRouterClient)(nil)
