// Package openai implements the OpenAI-compatible HTTP surface shared by
// hosted AI API resources. Provider-specific request extensions stay in the
// resource CLI; this package owns the common wire shape and failure mapping.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/ai-provider-go/providererror"
)

// Config describes one OpenAI-compatible provider endpoint.
type Config struct {
	ProviderName string
	BaseURL      string
	APIKey       string
	HTTPClient   HTTPClient
	UserAgent    string
}

// HTTPClient is the narrow HTTP seam used by tests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Client is an OpenAI-compatible provider client.
type Client struct {
	providerName string
	baseURL      string
	apiKey       string
	httpClient   HTTPClient
	userAgent    string
}

// New returns a client. A nil HTTP client uses http.DefaultClient.
func New(config Config) *Client {
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{
		providerName: strings.TrimSpace(config.ProviderName),
		baseURL:      strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"),
		apiKey:       strings.TrimSpace(config.APIKey),
		httpClient:   client,
		userAgent:    strings.TrimSpace(config.UserAgent),
	}
}

// ProviderName returns the configured provider name.
func (c *Client) ProviderName() string { return c.providerName }

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// Message is one OpenAI chat message. Content is string or []ContentPart.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ContentPart is one multimodal content part.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

// ChatRequest is the common OpenAI chat-completions request.
type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	// ExtraBody carries a provider's documented OpenAI-compatible extensions,
	// such as OrcaRouter's model fallback list. It is merged at the top level.
	ExtraBody map[string]any `json:"-"`
}

// Usage is provider-reported usage. Nil means the provider did not report it.
type Usage struct {
	PromptTokens     *int64
	CompletionTokens *int64
	Cost             *float64
}

// ChatResult is the decoded chat response plus the original body.
type ChatResult struct {
	Raw   []byte
	Text  string
	Usage Usage
}

// ListModels fetches the provider's /models payload.
func (c *Client) ListModels(ctx context.Context) ([]byte, error) {
	return c.do(ctx, http.MethodGet, "/models", nil)
}

// Chat performs a chat completion.
func (c *Client) Chat(ctx context.Context, request ChatRequest) (ChatResult, error) {
	body, err := marshalRequest(request)
	if err != nil {
		return ChatResult{}, err
	}
	raw, err := c.do(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return ChatResult{}, err
	}
	text, usage, err := DecodeChat(raw)
	if err != nil {
		return ChatResult{}, err
	}
	return ChatResult{Raw: raw, Text: text, Usage: usage}, nil
}

// GenerateImage performs an OpenAI-compatible image generation request.
func (c *Client) GenerateImage(ctx context.Context, request ImageRequest) ([]byte, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode image request: %w", err)
	}
	return c.do(ctx, http.MethodPost, "/images/generations", body)
}

func marshalRequest(request ChatRequest) ([]byte, error) {
	if len(request.ExtraBody) == 0 {
		encoded, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("encode chat request: %w", err)
		}
		return encoded, nil
	}
	base := map[string]any{}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode chat request: %w", err)
	}
	if err := json.Unmarshal(encoded, &base); err != nil {
		return nil, fmt.Errorf("normalize chat request: %w", err)
	}
	for key, value := range request.ExtraBody {
		base[key] = value
	}
	merged, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("encode extended chat request: %w", err)
	}
	return merged, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if c.baseURL == "" {
		return nil, &providererror.Error{Code: providererror.CodeUnconfigured, Message: "provider base URL is required"}
	}
	url := c.baseURL + path
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &providererror.Error{Code: providererror.CodeUnreachable, Err: err}
	}
	defer response.Body.Close()
	raw, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, &providererror.Error{Code: providererror.CodeStreamFailed, HTTPStatus: response.StatusCode, Err: readErr}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, providererror.NewHTTP(response.StatusCode, response.Header.Get("Retry-After"), errorDetail(raw))
	}
	return raw, nil
}

func errorDetail(raw []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		if message := strings.TrimSpace(envelope.Error.Message); message != "" {
			return message
		}
		if message := strings.TrimSpace(envelope.Message); message != "" {
			return message
		}
	}
	return strings.TrimSpace(string(raw))
}

// DecodeChat extracts the first choice's text and usage from an OpenAI chat
// response.
func DecodeChat(raw []byte) (string, Usage, error) {
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     *int64   `json:"prompt_tokens"`
			CompletionTokens *int64   `json:"completion_tokens"`
			Cost             *float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", Usage{}, fmt.Errorf("decode chat response: %w", err)
	}
	if len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" {
		return "", Usage{}, fmt.Errorf("provider returned no completion")
	}
	usage := Usage{}
	if envelope.Usage != nil {
		usage.PromptTokens = envelope.Usage.PromptTokens
		usage.CompletionTokens = envelope.Usage.CompletionTokens
		usage.Cost = envelope.Usage.Cost
	}
	return envelope.Choices[0].Message.Content, usage, nil
}

// MultimodalEnvelope is the standard-input shape shared by provider CLIs.
type MultimodalEnvelope struct {
	Prompt string        `json:"prompt"`
	Images []InlineImage `json:"images"`
}

type InlineImage struct {
	MediaType string `json:"media_type"`
	DataB64   string `json:"data_b64"`
}

// MessagesFromEnvelope builds one user message, using content parts when
// inline images are present.
func MessagesFromEnvelope(prompt string, images []InlineImage) []Message {
	if len(images) == 0 {
		return []Message{{Role: "user", Content: prompt}}
	}
	parts := make([]ContentPart, 0, len(images)+1)
	parts = append(parts, ContentPart{Type: "text", Text: prompt})
	for _, image := range images {
		mediaType := strings.TrimSpace(image.MediaType)
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		parts = append(parts, ContentPart{
			Type: "image_url",
			ImageURL: &ImageURL{
				URL: "data:" + mediaType + ";base64," + image.DataB64,
			},
		})
	}
	return []Message{{Role: "user", Content: parts}}
}

// DefaultTimeout is the default per-request timeout used by resource callers.
const DefaultTimeout = 120 * time.Second
