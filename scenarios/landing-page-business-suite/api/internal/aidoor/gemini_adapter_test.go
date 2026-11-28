package aidoor

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiRouteAcceptsAntigravityCustomEndpointPath(t *testing.T) {
	path := "/api/v1/ai/v1beta/models/http:/127.0.0.1:34567/api/v1/ai/models/fixture-model:streamGenerateContent"
	if !isGeminiGenerateContentPath(path) {
		t.Fatalf("Gemini dispatcher rejected Antigravity custom endpoint path %q", path)
	}
}

func TestGeminiRequestPreservesConversationAndTools(t *testing.T) {
	input := map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": "follow policy"}}},
		"contents": []any{
			map[string]any{"role": "user", "parts": []any{map[string]any{"text": "inspect this"}}},
			map[string]any{"role": "model", "parts": []any{map[string]any{"functionCall": map[string]any{"name": "read_file", "args": map[string]any{"path": "fixture.txt"}}}}},
			map[string]any{"role": "user", "parts": []any{map[string]any{"functionResponse": map[string]any{"name": "read_file", "response": map[string]any{"text": "fixture contents"}}}}},
		},
		"tools":            []any{map[string]any{"functionDeclarations": []any{map[string]any{"name": "read_file", "description": "Read a fixture", "parameters": map[string]any{"type": "object"}}}}},
		"generationConfig": map[string]any{"maxOutputTokens": float64(90), "temperature": float64(0.2), "topP": float64(0.8), "stopSequences": []any{"END"}},
	}
	request := geminiRequest(input, "fixture-model")
	if request["model"] != "fixture-model" || request["stream"] != true || request["max_tokens"] != float64(90) || request["temperature"] != float64(0.2) || request["top_p"] != float64(0.8) {
		t.Fatalf("Gemini request lost model or generation settings: %#v", request)
	}
	messages, ok := request["messages"].([]any)
	if !ok || len(messages) != 4 {
		t.Fatalf("Gemini request conversation = %#v, want system, user, tool call and tool result", request["messages"])
	}
	if messages[0].(map[string]any)["role"] != "system" || messages[1].(map[string]any)["content"] != "inspect this" || messages[2].(map[string]any)["role"] != "assistant" || messages[3].(map[string]any)["role"] != "tool" {
		t.Fatalf("Gemini conversation roles/content were not preserved: %#v", messages)
	}
	tools, ok := request["tools"].([]any)
	if !ok || len(tools) != 1 || tools[0].(map[string]any)["function"].(map[string]any)["name"] != "read_file" {
		t.Fatalf("Gemini function declarations were not translated: %#v", request["tools"])
	}
}

func TestGeminiStreamTranslatesOpenAIChunks(t *testing.T) {
	response := httptest.NewRecorder()
	stream := &geminiStream{ResponseWriter: response}
	chunk := `data: {"choices":[{"index":0,"delta":{"content":"fixture reply","tool_calls":[{"index":0,"function":{"name":"read_file","arguments":"{\"path\":\"fixture.txt\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":4,"completion_tokens":3,"total_tokens":7}}

`
	if _, err := stream.Write([]byte(chunk)); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text         string         `json:"text"`
					FunctionCall map[string]any `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata map[string]int `json:"usageMetadata"`
	}
	line := strings.TrimPrefix(strings.TrimSpace(response.Body.String()), "data: ")
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("decode Gemini stream chunk %q: %v", response.Body.String(), err)
	}
	if len(got.Candidates) != 1 || got.Candidates[0].FinishReason != "STOP" || got.Candidates[0].Content.Parts[0].Text != "fixture reply" || got.Candidates[0].Content.Parts[1].FunctionCall["name"] != "read_file" || got.UsageMetadata["totalTokenCount"] != 7 {
		t.Fatalf("Gemini response did not preserve content, tool call, finish reason and usage: %+v", got)
	}
}
