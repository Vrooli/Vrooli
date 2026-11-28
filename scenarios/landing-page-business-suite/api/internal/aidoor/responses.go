package aidoor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// responses uses the same chat dispatch, key checks, reservation and settlement
// as the public chat route. It only translates the native Responses protocol.
// Event shapes: https://developers.openai.com/api/reference/resources/responses/streaming-events
func (d *Door) responses(w http.ResponseWriter, r *http.Request, key Key) {
	var input struct {
		Model           string           `json:"model"`
		Input           json.RawMessage  `json:"input"`
		Instructions    string           `json:"instructions"`
		Tools           []map[string]any `json:"tools"`
		Stream          bool             `json:"stream"`
		MaxOutputTokens int              `json:"max_output_tokens"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<20)).Decode(&input) != nil {
		fail(w, 400, "invalid Responses request")
		return
	}
	messages, err := responseInputMessages(input.Input)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	if input.Instructions != "" {
		messages = append([]map[string]any{{"role": "system", "content": input.Instructions}}, messages...)
	}
	var tools []map[string]any
	custom := map[string]bool{}
	namespaces := map[string]string{}
	var flattened []map[string]any
	for _, tool := range input.Tools {
		if tool["type"] != "namespace" {
			flattened = append(flattened, tool)
			continue
		}
		namespace, _ := tool["name"].(string)
		children, _ := tool["tools"].([]any)
		for _, child := range children {
			entry, ok := child.(map[string]any)
			if !ok {
				fail(w, 400, "invalid namespaced tool")
				return
			}
			name, _ := entry["name"].(string)
			qualified := namespace + "__" + name
			entry["name"] = qualified
			namespaces[qualified] = namespace
			flattened = append(flattened, entry)
		}
	}
	for _, tool := range flattened {
		kind, _ := tool["type"].(string)
		switch kind {
		case "function":
			fn := map[string]any{}
			for _, field := range []string{"name", "description", "parameters", "strict"} {
				if value, ok := tool[field]; ok {
					fn[field] = value
				}
			}
			tools = append(tools, map[string]any{"type": "function", "function": fn})
		case "custom":
			name, _ := tool["name"].(string)
			custom[name] = true
			tools = append(tools, map[string]any{"type": "function", "function": map[string]any{"name": name, "description": tool["description"], "parameters": map[string]any{"type": "object", "properties": map[string]any{"input": map[string]string{"type": "string"}}, "required": []string{"input"}}}})
		default:
			fail(w, 400, "unsupported Responses tool type: "+kind)
			return
		}
	}
	chat := map[string]any{"model": input.Model, "messages": messages, "stream": true}
	if len(tools) > 0 {
		chat["tools"] = tools
	}
	if input.MaxOutputTokens > 0 {
		chat["max_tokens"] = input.MaxOutputTokens
	}
	body, _ := json.Marshal(chat)
	request := r.Clone(r.Context())
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.ContentLength = int64(len(body))
	adapter := &responsesWriter{ResponseWriter: w, stream: input.Stream, model: input.Model, id: "resp_" + randomID(), calls: map[int]*responseCall{}, custom: custom, namespaces: namespaces}
	d.chat(adapter, request, key)
	adapter.finish()
}

func responseInputMessages(raw json.RawMessage) ([]map[string]any, error) {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return []map[string]any{{"role": "user", "content": text}}, nil
	}
	var items []map[string]any
	if json.Unmarshal(raw, &items) != nil {
		return nil, errors.New("invalid Responses input")
	}
	var messages []map[string]any
	for _, item := range items {
		kind, _ := item["type"].(string)
		switch kind {
		case "", "message":
			role, _ := item["role"].(string)
			if role == "developer" {
				role = "system"
			}
			content := item["content"]
			if parts, ok := content.([]any); ok {
				var translated []any
				for _, rawPart := range parts {
					part, ok := rawPart.(map[string]any)
					if !ok {
						return nil, errors.New("invalid message content")
					}
					switch part["type"] {
					case "input_text", "output_text", "text":
						translated = append(translated, map[string]any{"type": "text", "text": part["text"]})
					case "input_image":
						translated = append(translated, map[string]any{"type": "image_url", "image_url": map[string]any{"url": part["image_url"]}})
					default:
						return nil, errors.New("unsupported Responses content type")
					}
				}
				content = translated
			}
			messages = append(messages, map[string]any{"role": role, "content": content})
		case "function_call":
			messages = append(messages, map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"id": item["call_id"], "type": "function", "function": map[string]any{"name": responseToolName(item), "arguments": item["arguments"]}}}})
		case "custom_tool_call":
			arguments, _ := json.Marshal(map[string]any{"input": item["input"]})
			messages = append(messages, map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"id": item["call_id"], "type": "function", "function": map[string]any{"name": responseToolName(item), "arguments": string(arguments)}}}})
		case "function_call_output", "custom_tool_call_output":
			output := item["output"]
			if _, ok := output.(string); !ok {
				encoded, _ := json.Marshal(output)
				output = string(encoded)
			}
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": item["call_id"], "content": output})
		case "reasoning": // Opaque provider reasoning is not transferable to another router.
		default:
			return nil, errors.New("unsupported Responses input item")
		}
	}
	return messages, nil
}

func responseToolName(item map[string]any) string {
	name, _ := item["name"].(string)
	namespace, _ := item["namespace"].(string)
	if namespace != "" {
		return namespace + "__" + name
	}
	return name
}

type responseCall struct{ ID, Name, Arguments string }
type responsesWriter struct {
	http.ResponseWriter
	stream, started, done, failed bool
	model, id, text               string
	sequence                      int
	pending                       []byte
	calls                         map[int]*responseCall
	custom                        map[string]bool
	namespaces                    map[string]string
	usage                         map[string]any
}

// Preserve ResponseController access to the real connection, including the
// streaming path's removal of the server write deadline.
func (w *responsesWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *responsesWriter) WriteHeader(status int) {
	if status >= 400 {
		w.failed = true
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *responsesWriter) Flush() {
	if w.stream {
		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}
	}
}
func (w *responsesWriter) event(kind string, fields map[string]any) {
	if !w.stream {
		return
	}
	w.sequence++
	fields["type"] = kind
	fields["sequence_number"] = w.sequence
	encoded, _ := json.Marshal(fields)
	fmt.Fprintf(w.ResponseWriter, "event: %s\ndata: %s\n\n", kind, encoded)
	w.Flush()
}
func (w *responsesWriter) response(status string, items []any) map[string]any {
	return map[string]any{"id": w.id, "object": "response", "created_at": time.Now().Unix(), "model": w.model, "status": status, "output": items, "usage": w.usage, "error": nil}
}
func (w *responsesWriter) Write(p []byte) (int, error) {
	if w.failed {
		return w.ResponseWriter.Write(p)
	}
	if !w.started {
		w.started = true
		if w.stream {
			w.Header().Set("Content-Type", "text/event-stream")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		w.event("response.created", map[string]any{"response": w.response("in_progress", []any{})})
	}
	w.pending = append(w.pending, p...)
	for {
		line, rest, found := bytes.Cut(w.pending, []byte("\n"))
		if !found {
			break
		}
		w.pending = rest
		w.line(strings.TrimSpace(string(line)))
	}
	return len(p), nil
}
func (w *responsesWriter) line(line string) {
	if !strings.HasPrefix(line, "data:") {
		return
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "[DONE]" {
		w.done = true
		return
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int                              `json:"index"`
					ID       string                           `json:"id"`
					Function struct{ Name, Arguments string } `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
		Usage    struct{ Prompt, Completion, Total int } `json:"-"`
		RawUsage map[string]any                          `json:"usage"`
		Error    json.RawMessage                         `json:"error"`
	}
	if json.Unmarshal([]byte(data), &chunk) != nil || len(chunk.Error) > 0 {
		w.failed = true
		w.event("error", map[string]any{"code": "upstream_error", "message": "provider stream failed"})
		return
	}
	if chunk.RawUsage != nil {
		w.usage = map[string]any{"input_tokens": chunk.RawUsage["prompt_tokens"], "output_tokens": chunk.RawUsage["completion_tokens"], "total_tokens": chunk.RawUsage["total_tokens"]}
	}
	for _, choice := range chunk.Choices {
		if choice.Delta.Content != "" {
			if w.text == "" {
				w.event("response.output_item.added", map[string]any{"output_index": 0, "item": map[string]any{"id": w.id + "_message", "type": "message", "role": "assistant", "status": "in_progress", "content": []any{}}})
			}
			w.text += choice.Delta.Content
			w.event("response.output_text.delta", map[string]any{"output_index": 0, "content_index": 0, "item_id": w.id + "_message", "delta": choice.Delta.Content})
		}
		for _, call := range choice.Delta.ToolCalls {
			entry := w.calls[call.Index]
			if entry == nil {
				entry = &responseCall{}
				w.calls[call.Index] = entry
			}
			entry.ID += call.ID
			entry.Name += call.Function.Name
			entry.Arguments += call.Function.Arguments
		}
	}
}
func (w *responsesWriter) finish() {
	if w.failed || !w.started {
		return
	}
	if !w.done {
		w.event("error", map[string]any{"code": "incomplete_stream", "message": "provider stream ended before completion"})
		return
	}
	items := []any{}
	if w.text != "" {
		item := map[string]any{"id": w.id + "_message", "type": "message", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": w.text, "annotations": []any{}}}}
		items = append(items, item)
		w.event("response.output_text.done", map[string]any{"output_index": 0, "content_index": 0, "item_id": w.id + "_message", "text": w.text})
		w.event("response.output_item.done", map[string]any{"output_index": 0, "item": item})
	}
	indexes := make([]int, 0, len(w.calls))
	for index := range w.calls {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		call := w.calls[index]
		item := map[string]any{"id": w.id + "_" + call.ID, "type": "function_call", "call_id": call.ID, "name": call.Name, "arguments": call.Arguments, "status": "completed"}
		if w.custom[call.Name] {
			var args struct {
				Input string `json:"input"`
			}
			if json.Unmarshal([]byte(call.Arguments), &args) != nil {
				w.event("error", map[string]any{"code": "invalid_tool_arguments", "message": "provider returned invalid custom tool input"})
				return
			}
			item["type"] = "custom_tool_call"
			item["input"] = args.Input
			delete(item, "arguments")
		}
		if namespace := w.namespaces[call.Name]; namespace != "" {
			item["namespace"] = namespace
			item["name"] = strings.TrimPrefix(call.Name, namespace+"__")
		}
		outputIndex := len(items)
		items = append(items, item)
		w.event("response.output_item.added", map[string]any{"output_index": outputIndex, "item": item})
		w.event("response.output_item.done", map[string]any{"output_index": outputIndex, "item": item})
	}
	response := w.response("completed", items)
	if w.stream {
		w.event("response.completed", map[string]any{"response": response})
	} else {
		json.NewEncoder(w.ResponseWriter).Encode(response)
	}
}
