package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/ai-provider-go/providererror"
)

const (
	maxSSELineBytes  = 1024 * 1024
	maxSSEEventBytes = 2 * 1024 * 1024
)

// StreamChunk is one decoded OpenAI-compatible streaming event. Raw preserves
// the provider event for callers that need fields not represented here.
type StreamChunk struct {
	Raw          []byte
	Text         string
	Usage        Usage
	FinishReason string
	Done         bool
}

// ChatStream reads one event at a time. Calling Next is the backpressure
// boundary: the response is not read ahead while the caller processes a chunk.
// Always call Close when the caller stops before Done or EOF.
type ChatStream struct {
	response  *http.Response
	scanner   *bufio.Scanner
	data      []string
	dataBytes int
	closed    bool
	done      bool
}

// StreamChat opens an OpenAI-compatible SSE chat completion. The returned
// stream is consumed synchronously with Next, so a slow caller naturally
// backpressures the HTTP body. ctx cancellation interrupts both the request
// and subsequent reads.
func (c *Client) StreamChat(ctx context.Context, request ChatRequest) (*ChatStream, error) {
	if c.baseURL == "" {
		return nil, &providererror.Error{Code: providererror.CodeUnconfigured, Message: "provider base URL is required"}
	}
	body, err := marshalRequest(request)
	if err != nil {
		return nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("normalize stream request: %w", err)
	}
	payload["stream"] = json.RawMessage("true")
	body, err = json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode stream request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
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
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		raw, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil {
			return nil, &providererror.Error{Code: providererror.CodeStreamFailed, HTTPStatus: response.StatusCode, Err: readErr}
		}
		return nil, providererror.NewHTTP(response.StatusCode, response.Header.Get("Retry-After"), errorDetail(raw))
	}
	if response.Body == nil {
		return nil, &providererror.Error{Code: providererror.CodeStreamFailed, HTTPStatus: response.StatusCode, Message: "provider returned an empty stream body"}
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), maxSSELineBytes)
	return &ChatStream{response: response, scanner: scanner}, nil
}

// Next returns the next data event, the terminal [DONE] event, or io.EOF.
// Malformed events and interrupted reads are returned as typed stream errors.
func (s *ChatStream) Next() (StreamChunk, error) {
	if s == nil || s.closed || s.done {
		return StreamChunk{}, io.EOF
	}
	for s.scanner.Scan() {
		line := s.scanner.Text()
		if line == "" {
			chunk, ok, err := s.dispatch()
			if err != nil {
				_ = s.Close()
				s.done = true
				return StreamChunk{}, err
			}
			if ok {
				return chunk, nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") { // SSE comment/keepalive.
			continue
		}
		field, value, found := strings.Cut(line, ":")
		if !found || field != "data" {
			continue
		}
		value = strings.TrimPrefix(value, " ")
		additionalBytes := len(value)
		if len(s.data) > 0 {
			additionalBytes++ // strings.Join inserts one newline between data lines.
		}
		if additionalBytes > maxSSEEventBytes-s.dataBytes {
			_ = s.Close()
			s.done = true
			return StreamChunk{}, &providererror.Error{
				Code:       providererror.CodeStreamFailed,
				HTTPStatus: s.response.StatusCode,
				Message:    fmt.Sprintf("SSE event exceeds %d bytes", maxSSEEventBytes),
			}
		}
		s.dataBytes += additionalBytes
		s.data = append(s.data, value)
	}
	if err := s.scanner.Err(); err != nil {
		_ = s.Close()
		s.done = true
		return StreamChunk{}, &providererror.Error{Code: providererror.CodeStreamFailed, HTTPStatus: s.response.StatusCode, Err: err}
	}
	_ = s.Close()
	return StreamChunk{}, io.EOF
}

// Close releases the HTTP response. It is safe to call more than once.
func (s *ChatStream) Close() error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	if s.response == nil || s.response.Body == nil {
		return nil
	}
	return s.response.Body.Close()
}

func (s *ChatStream) dispatch() (StreamChunk, bool, error) {
	if len(s.data) == 0 {
		return StreamChunk{}, false, nil
	}
	raw := []byte(strings.Join(s.data, "\n"))
	s.data = nil
	s.dataBytes = 0
	if bytes.Equal(bytes.TrimSpace(raw), []byte("[DONE]")) {
		s.done = true
		_ = s.Close()
		return StreamChunk{Done: true, Raw: raw}, true, nil
	}
	var envelope struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     *int64   `json:"prompt_tokens"`
			CompletionTokens *int64   `json:"completion_tokens"`
			Cost             *float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return StreamChunk{}, false, &providererror.Error{Code: providererror.CodeStreamFailed, HTTPStatus: s.response.StatusCode, Err: fmt.Errorf("decode stream event: %w", err)}
	}
	chunk := StreamChunk{Raw: raw}
	if len(envelope.Choices) > 0 {
		chunk.Text = envelope.Choices[0].Delta.Content
		if envelope.Choices[0].FinishReason != nil {
			chunk.FinishReason = *envelope.Choices[0].FinishReason
		}
	}
	if envelope.Usage != nil {
		chunk.Usage = Usage{PromptTokens: envelope.Usage.PromptTokens, CompletionTokens: envelope.Usage.CompletionTokens, Cost: envelope.Usage.Cost}
	}
	return chunk, true, nil
}
