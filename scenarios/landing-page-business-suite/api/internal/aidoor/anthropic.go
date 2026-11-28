package aidoor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// anthropicMessages adapts Claude Code's native Messages wire format to the
// OpenAI-compatible request accepted by AI Gateway. Credits, policy checks,
// reservations and settlement remain in the shared chat path.
func (d *Door) anthropicMessages(w http.ResponseWriter, r *http.Request, key Key) {
	var input map[string]any
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input) != nil {
		fail(w, http.StatusBadRequest, "invalid Messages request")
		return
	}
	model, _ := input["model"].(string)
	if model == "" || (key.Model != "" && key.Model != model) {
		fail(w, http.StatusForbidden, "model is outside key scope")
		return
	}
	openAI := anthropicRequest(input)
	body, err := json.Marshal(openAI)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid Messages request")
		return
	}
	request := r.Clone(r.Context())
	request.Body = ioNopCloser{Reader: bytes.NewReader(body)}
	request.ContentLength = int64(len(body))
	d.chat(&anthropicStream{ResponseWriter: w, model: model, tools: make(map[int]int)}, request, key)
}

type ioNopCloser struct{ *bytes.Reader }

func (r ioNopCloser) Close() error { return nil }

func anthropicRequest(input map[string]any) map[string]any {
	request := map[string]any{"model": input["model"], "stream": true}
	for _, name := range []string{"max_tokens", "temperature", "top_p", "stop_sequences"} {
		if value, ok := input[name]; ok {
			mapped := name
			if name == "stop_sequences" {
				mapped = "stop"
			}
			request[mapped] = value
		}
	}
	if text := anthropicText(input["system"]); text != "" {
		request["messages"] = []any{map[string]any{"role": "system", "content": text}}
	}
	messages, _ := input["messages"].([]any)
	converted := request["messages"].([]any)
	for _, item := range messages {
		message, _ := item.(map[string]any)
		role, _ := message["role"].(string)
		blocks, isBlocks := message["content"].([]any)
		if !isBlocks {
			converted = append(converted, map[string]any{"role": role, "content": message["content"]})
			continue
		}
		var textParts []string
		var toolCalls []any
		for _, value := range blocks {
			block, _ := value.(map[string]any)
			switch block["type"] {
			case "text":
				if text, ok := block["text"].(string); ok {
					textParts = append(textParts, text)
				}
			case "tool_use":
				args, _ := json.Marshal(block["input"])
				toolCalls = append(toolCalls, map[string]any{"id": block["id"], "type": "function", "function": map[string]any{"name": block["name"], "arguments": string(args)}})
			case "tool_result":
				converted = append(converted, map[string]any{"role": "tool", "tool_call_id": block["tool_use_id"], "content": anthropicText(block["content"])})
			}
		}
		out := map[string]any{"role": role, "content": strings.Join(textParts, "\n")}
		if len(toolCalls) > 0 {
			out["tool_calls"] = toolCalls
		}
		if len(textParts) > 0 || len(toolCalls) > 0 {
			converted = append(converted, out)
		}
	}
	request["messages"] = converted
	if tools, ok := input["tools"].([]any); ok && len(tools) > 0 {
		convertedTools := make([]any, 0, len(tools))
		for _, item := range tools {
			tool, _ := item.(map[string]any)
			function := map[string]any{"name": tool["name"], "parameters": tool["input_schema"]}
			if description, ok := tool["description"]; ok {
				function["description"] = description
			}
			convertedTools = append(convertedTools, map[string]any{"type": "function", "function": function})
		}
		request["tools"] = convertedTools
	}
	return request
}

func anthropicText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	blocks, _ := value.([]any)
	var parts []string
	for _, item := range blocks {
		block, _ := item.(map[string]any)
		if block["type"] == "text" {
			if text, ok := block["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

type anthropicStream struct {
	http.ResponseWriter
	model       string
	status      int
	started     bool
	textStarted bool
	textClosed  bool
	textIndex   int
	closed      bool
	tools       map[int]int
	nextBlock   int
}

func (w *anthropicStream) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *anthropicStream) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *anthropicStream) Write(data []byte) (int, error) {
	if w.status >= http.StatusBadRequest {
		return w.ResponseWriter.Write(data)
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "data:") {
		return len(data), nil
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "[DONE]" {
		w.finish("end_turn")
		return len(data), nil
	}
	var chunk struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal([]byte(payload), &chunk) != nil {
		return len(data), nil
	}
	w.start(chunk.ID, chunk.Model)
	for _, choice := range chunk.Choices {
		if choice.Delta.Content != "" {
			if !w.textStarted {
				w.textStarted = true
				w.textIndex = w.nextBlock
				w.emit("content_block_start", map[string]any{"index": w.textIndex, "content_block": map[string]any{"type": "text", "text": ""}})
				w.nextBlock++
			}
			w.emit("content_block_delta", map[string]any{"index": w.textIndex, "delta": map[string]any{"type": "text_delta", "text": choice.Delta.Content}})
		}
		for _, call := range choice.Delta.ToolCalls {
			blockIndex, ok := w.tools[call.Index]
			if !ok {
				blockIndex = w.nextBlock
				w.nextBlock++
				w.tools[call.Index] = blockIndex
				w.emit("content_block_start", map[string]any{"index": blockIndex, "content_block": map[string]any{"type": "tool_use", "id": call.ID, "name": call.Function.Name, "input": map[string]any{}}})
			}
			if call.Function.Arguments != "" {
				w.emit("content_block_delta", map[string]any{"index": blockIndex, "delta": map[string]any{"type": "input_json_delta", "partial_json": call.Function.Arguments}})
			}
		}
		if choice.FinishReason != "" {
			stop := "end_turn"
			if choice.FinishReason == "tool_calls" {
				stop = "tool_use"
			} else if choice.FinishReason == "length" {
				stop = "max_tokens"
			}
			w.finish(stop)
		}
	}
	return len(data), nil
}

func (w *anthropicStream) start(id, model string) {
	if w.started {
		return
	}
	w.started = true
	if id == "" {
		id = "msg_vrooli"
	} else if !strings.HasPrefix(id, "msg_") {
		id = "msg_" + id
	}
	if model == "" {
		model = w.model
	}
	w.emit("message_start", map[string]any{"message": map[string]any{"id": id, "type": "message", "role": "assistant", "model": model, "content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": map[string]any{"input_tokens": 0, "output_tokens": 0}}})
}

func (w *anthropicStream) finish(reason string) {
	if w.closed {
		return
	}
	if w.textStarted && !w.textClosed {
		w.emit("content_block_stop", map[string]any{"index": w.textIndex})
		w.textClosed = true
	}
	for _, index := range w.tools {
		w.emit("content_block_stop", map[string]any{"index": index})
	}
	w.emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": reason, "stop_sequence": nil}, "usage": map[string]any{"output_tokens": 0}})
	w.emit("message_stop", map[string]any{"type": "message_stop"})
	w.closed = true
}

func (w *anthropicStream) emit(event string, payload any) {
	if object, ok := payload.(map[string]any); ok {
		object["type"] = event
	}
	data, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(w.ResponseWriter, "event: %s\ndata: %s\n\n", event, data)
}
