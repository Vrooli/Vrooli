package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"web-console/internal/sessionstore"
)

const paidCodingRole = "code.default"

type paidCodingCompletionRequest struct {
	Model      string            `json:"model"`
	Messages   []json.RawMessage `json:"messages"`
	Tools      []json.RawMessage `json:"tools,omitempty"`
	ToolChoice json.RawMessage   `json:"tool_choice,omitempty"`
	Stream     bool              `json:"stream,omitempty"`
	MaxTokens  int               `json:"max_tokens,omitempty"`
}

type paidCodingUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (s *Server) paidCodingCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	if s == nil || !s.paidCodingProtocolQualified {
		http.Error(w, "Vrooli-funded coding is not qualified on this installation", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	binding, ok := activePaidCodingSessions.Lookup(token)
	if !ok {
		http.Error(w, "paid coding session unavailable", http.StatusUnauthorized)
		return
	}
	if s == nil || s.sessionStore == nil || s.sessions == nil {
		http.Error(w, "paid coding session unavailable", http.StatusServiceUnavailable)
		return
	}
	metadata, err := s.sessionStore.Get(r.Context(), binding.SessionID)
	if err != nil || metadata.BusinessAccountID != binding.BusinessAccountID || metadata.AgentType != sessionstore.AgentOpenCode || metadata.Status != sessionstore.StatusLive {
		activePaidCodingSessions.RemoveSession(binding.SessionID)
		http.Error(w, "paid coding session unavailable", http.StatusUnauthorized)
		return
	}
	if _, live := s.sessions.Get(binding.SessionID); !live {
		activePaidCodingSessions.RemoveSession(binding.SessionID)
		http.Error(w, "paid coding session unavailable", http.StatusUnauthorized)
		return
	}
	var req paidCodingCompletionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<20))
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid chat completion request", http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 || len(req.Messages) > 100 || req.MaxTokens < 0 || req.MaxTokens > 16384 || len(req.Tools) > 64 {
		http.Error(w, "invalid chat completion request", http.StatusBadRequest)
		return
	}
	for _, message := range req.Messages {
		if len(message) == 0 || !json.Valid(message) {
			http.Error(w, "invalid chat completion message", http.StatusBadRequest)
			return
		}
	}
	for _, tool := range req.Tools {
		if len(tool) == 0 || !json.Valid(tool) {
			http.Error(w, "invalid tool declaration", http.StatusBadRequest)
			return
		}
	}
	if len(req.ToolChoice) > 0 && !json.Valid(req.ToolChoice) {
		http.Error(w, "invalid tool choice", http.StatusBadRequest)
		return
	}
	if s.subscriptionResolver == nil {
		http.Error(w, "Vrooli subscription session unavailable", http.StatusServiceUnavailable)
		return
	}
	access, err := s.subscriptionResolver.Resolve(r.Context())
	if err != nil || strings.TrimSpace(access.AccessToken) == "" {
		http.Error(w, "Vrooli subscription session unavailable", http.StatusUnauthorized)
		return
	}
	baseURL := strings.TrimRight(strings.TrimSpace(s.subscriptionResolver.LPBSBaseURL), "/")
	if baseURL == "" {
		http.Error(w, "paid coding authority unavailable", http.StatusServiceUnavailable)
		return
	}
	upstreamPath := "/api/v1/ai/inference"
	if req.Stream {
		upstreamPath = "/api/v1/ai/inference-stream"
	}
	body, err := json.Marshal(map[string]any{
		"role": paidCodingRole, "business_account_id": binding.BusinessAccountID,
		"messages": req.Messages, "tools": req.Tools, "tool_choice": req.ToolChoice,
		"max_tokens": req.MaxTokens,
		"metadata":   map[string]string{"app_bundle_key": "web-console", "operation": "interactive_coding"},
	})
	if err != nil {
		http.Error(w, "could not prepare paid coding request", http.StatusInternalServerError)
		return
	}
	upstreamRequest, err := http.NewRequestWithContext(r.Context(), http.MethodPost, baseURL+upstreamPath, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "could not create paid coding request", http.StatusInternalServerError)
		return
	}
	upstreamRequest.Header.Set("Authorization", "Bearer "+access.AccessToken)
	upstreamRequest.Header.Set("Content-Type", "application/json")
	client := s.subscriptionResolver.HTTPClient
	if client == nil || client.Timeout > 0 && client.Timeout < 10*time.Minute {
		client = &http.Client{Transport: http.DefaultTransport, Timeout: 10 * time.Minute}
	}
	response, err := client.Do(upstreamRequest)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		http.Error(w, "paid coding provider request failed", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		http.Error(w, "paid coding provider refused the request", response.StatusCode)
		return
	}
	if req.Stream {
		s.forwardPaidCodingStream(r.Context(), w, response.Body, req.Model)
		return
	}
	s.writePaidCodingCompletion(w, response.Body, req.Model)
}

func (s *Server) writePaidCodingCompletion(w http.ResponseWriter, body io.Reader, model string) {
	var upstream struct {
		ID               string            `json:"id"`
		Content          string            `json:"content"`
		ToolCalls        []json.RawMessage `json:"tool_calls"`
		PromptTokens     int               `json:"prompt_tokens"`
		CompletionTokens int               `json:"completion_tokens"`
		TotalTokens      int               `json:"total_tokens"`
		FinishReason     string            `json:"finish_reason"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 16<<20)).Decode(&upstream); err != nil {
		http.Error(w, "invalid paid coding response", http.StatusBadGateway)
		return
	}
	finishReason := upstream.FinishReason
	if finishReason == "" {
		finishReason = "stop"
		if len(upstream.ToolCalls) > 0 {
			finishReason = "tool_calls"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id": upstream.ID, "object": "chat.completion", "created": time.Now().Unix(), "model": publicPaidCodingModel(model),
		"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": nullableContent(upstream.Content), "tool_calls": upstream.ToolCalls}, "finish_reason": finishReason}},
		"usage":   paidCodingUsage{PromptTokens: upstream.PromptTokens, CompletionTokens: upstream.CompletionTokens, TotalTokens: upstream.TotalTokens},
	})
}

func (s *Server) forwardPaidCodingStream(ctx context.Context, w http.ResponseWriter, body io.Reader, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()
	model = publicPaidCodingModel(model)
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 2<<20)
	var measured paidCodingUsage
	toolCallsSeen := false
	completed := false
	writeChunk := func(delta map[string]any, finish string, usage *paidCodingUsage) {
		chunk := map[string]any{"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
			"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}
		if usage != nil {
			chunk["usage"] = usage
		}
		data, _ := json.Marshal(chunk)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var event struct {
			Type      string            `json:"type"`
			Content   string            `json:"content"`
			Error     string            `json:"error"`
			ToolCalls []json.RawMessage `json:"tool_calls"`
			Usage     *paidCodingUsage  `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			writePaidCodingStreamError(w, flusher, "paid coding stream returned an invalid event")
			return
		}
		switch event.Type {
		case "chunk":
			delta := map[string]any{}
			if event.Content != "" {
				delta["content"] = event.Content
			}
			if len(event.ToolCalls) > 0 {
				delta["tool_calls"] = event.ToolCalls
				toolCallsSeen = true
			}
			if len(delta) > 0 {
				writeChunk(delta, "", nil)
			}
		case "done":
			completed = true
			if event.Usage != nil {
				measured = *event.Usage
			}
			finish := "stop"
			if toolCallsSeen {
				finish = "tool_calls"
			}
			writeChunk(map[string]any{}, finish, &measured)
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		case "error":
			writePaidCodingStreamError(w, flusher, "paid coding stream failed")
			return
		}
	}
	if ctx.Err() != nil {
		return
	}
	if err := scanner.Err(); err != nil {
		writePaidCodingStreamError(w, flusher, "paid coding stream interrupted")
		return
	}
	if !completed {
		writePaidCodingStreamError(w, flusher, "paid coding stream ended before completion")
	}
}

func writePaidCodingStreamError(w http.ResponseWriter, flusher http.Flusher, message string) {
	payload, _ := json.Marshal(map[string]any{"error": map[string]string{"message": message, "type": "server_error"}})
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
}

func publicPaidCodingModel(_ string) string { return "vrooli/coding" }

func nullableContent(content string) any {
	if content == "" {
		return nil
	}
	return content
}
