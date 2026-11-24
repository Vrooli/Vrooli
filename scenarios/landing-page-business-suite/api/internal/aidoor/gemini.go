package aidoor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func isGeminiGenerateContentPath(path string) bool {
	if !(strings.HasPrefix(path, "/api/v1/ai/models/") || strings.HasPrefix(path, "/api/v1/ai/v1beta/models/")) {
		return false
	}
	return strings.HasSuffix(path, ":streamGenerateContent") || strings.HasSuffix(path, ":generateContent")
}

// geminiGenerateContent adapts Antigravity's Gemini generate-content protocol
// to the shared OpenAI-compatible chat route, preserving the door's existing
// credit authorization and settlement path.
func (d *Door) geminiGenerateContent(w http.ResponseWriter, r *http.Request, key Key) {
	model := geminiModelFromPath(r.URL.Path)
	if model == "" || (key.Model != "" && key.Model != model) {
		fail(w, http.StatusForbidden, "model is outside key scope")
		return
	}
	var input map[string]any
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input) != nil {
		fail(w, http.StatusBadRequest, "invalid Gemini request")
		return
	}
	body, err := json.Marshal(geminiRequest(input, model))
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid Gemini request")
		return
	}
	request := r.Clone(r.Context())
	request.Body = ioNopCloser{Reader: bytes.NewReader(body)}
	request.ContentLength = int64(len(body))
	d.chat(&geminiStream{ResponseWriter: w}, request, key)
}

// geminiModelFromPath accepts the Door's concise Gemini route and the path
// Antigravity's documented custom endpoint emits. Antigravity prefixes its
// configured base with /v1beta and carries the configured gemini-api URI into
// the model segment; unwrap that URI to the final model ID before scope checks.
func geminiModelFromPath(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/ai/")
	path = strings.TrimPrefix(path, "v1beta/")
	path = strings.TrimPrefix(path, "models/")
	path = strings.TrimSuffix(strings.TrimSuffix(path, ":streamGenerateContent"), ":generateContent")
	if index := strings.LastIndex(path, "/models/"); index >= 0 {
		path = path[index+len("/models/"):]
	}
	return strings.TrimSpace(path)
}

func geminiRequest(input map[string]any, model string) map[string]any {
	request := map[string]any{"model": model, "stream": true}
	messages := make([]any, 0)
	if text := geminiText(input["systemInstruction"]); text != "" {
		messages = append(messages, map[string]any{"role": "system", "content": text})
	}
	contents, _ := input["contents"].([]any)
	for _, value := range contents {
		content, _ := value.(map[string]any)
		role := "user"
		if content["role"] == "model" {
			role = "assistant"
		}
		parts, _ := content["parts"].([]any)
		var textParts []string
		var toolCalls []any
		for _, value := range parts {
			part, _ := value.(map[string]any)
			if text, ok := part["text"].(string); ok {
				textParts = append(textParts, text)
			}
			if functionCall, ok := part["functionCall"].(map[string]any); ok {
				args, _ := json.Marshal(functionCall["args"])
				toolCalls = append(toolCalls, map[string]any{"id": "call_" + fmt.Sprint(functionCall["name"]), "type": "function", "function": map[string]any{"name": functionCall["name"], "arguments": string(args)}})
			}
			if functionResponse, ok := part["functionResponse"].(map[string]any); ok {
				messages = append(messages, map[string]any{"role": "tool", "tool_call_id": "call_" + fmt.Sprint(functionResponse["name"]), "content": fmt.Sprint(functionResponse["response"])})
			}
		}
		message := map[string]any{"role": role, "content": strings.Join(textParts, "\n")}
		if len(toolCalls) > 0 {
			message["tool_calls"] = toolCalls
		}
		if len(textParts) > 0 || len(toolCalls) > 0 {
			messages = append(messages, message)
		}
	}
	request["messages"] = messages
	if tools, ok := input["tools"].([]any); ok && len(tools) > 0 {
		var converted []any
		for _, value := range tools {
			tool, _ := value.(map[string]any)
			declarations, _ := tool["functionDeclarations"].([]any)
			for _, value := range declarations {
				declaration, _ := value.(map[string]any)
				function := map[string]any{"name": declaration["name"], "parameters": declaration["parameters"]}
				if description, ok := declaration["description"]; ok {
					function["description"] = description
				}
				converted = append(converted, map[string]any{"type": "function", "function": function})
			}
		}
		if len(converted) > 0 {
			request["tools"] = converted
		}
	}
	if generation, ok := input["generationConfig"].(map[string]any); ok {
		for from, to := range map[string]string{"maxOutputTokens": "max_tokens", "temperature": "temperature", "topP": "top_p", "stopSequences": "stop"} {
			if value, ok := generation[from]; ok {
				request[to] = value
			}
		}
	}
	return request
}

func geminiText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	block, _ := value.(map[string]any)
	parts, _ := block["parts"].([]any)
	var texts []string
	for _, value := range parts {
		part, _ := value.(map[string]any)
		if text, ok := part["text"].(string); ok {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n")
}

type geminiStream struct {
	http.ResponseWriter
	status int
}

func (w *geminiStream) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *geminiStream) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
func (w *geminiStream) Write(data []byte) (int, error) {
	if w.status >= http.StatusBadRequest {
		return w.ResponseWriter.Write(data)
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "data:") {
		return len(data), nil
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "[DONE]" {
		return len(data), nil
	}
	var chunk struct {
		Choices []struct {
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
			Delta        struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int `json:"index"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
		Usage json.RawMessage `json:"usage"`
	}
	if json.Unmarshal([]byte(payload), &chunk) != nil {
		return len(data), nil
	}
	for _, choice := range chunk.Choices {
		parts := []any{}
		if choice.Delta.Content != "" {
			parts = append(parts, map[string]any{"text": choice.Delta.Content})
		}
		for _, call := range choice.Delta.ToolCalls {
			var args any
			if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil {
				args = map[string]any{}
			}
			parts = append(parts, map[string]any{"functionCall": map[string]any{"name": call.Function.Name, "args": args}})
		}
		candidate := map[string]any{"index": choice.Index, "content": map[string]any{"role": "model", "parts": parts}}
		if choice.FinishReason != "" {
			reason := "STOP"
			if choice.FinishReason == "length" {
				reason = "MAX_TOKENS"
			} else if choice.FinishReason == "tool_calls" {
				reason = "STOP"
			}
			candidate["finishReason"] = reason
		}
		response := map[string]any{"candidates": []any{candidate}}
		if len(chunk.Usage) > 0 && string(chunk.Usage) != "null" {
			var usage struct {
				Prompt     int `json:"prompt_tokens"`
				Completion int `json:"completion_tokens"`
				Total      int `json:"total_tokens"`
			}
			if json.Unmarshal(chunk.Usage, &usage) == nil {
				response["usageMetadata"] = map[string]any{"promptTokenCount": usage.Prompt, "candidatesTokenCount": usage.Completion, "totalTokenCount": usage.Total}
			}
		}
		encoded, _ := json.Marshal(response)
		if _, err := fmt.Fprintf(w.ResponseWriter, "data: %s\n\n", encoded); err != nil {
			return 0, err
		}
	}
	return len(data), nil
}
