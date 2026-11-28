package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatRawPreservesProviderSpecificResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"message":{"content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,abc"}}]}}]}`))
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL + "/v1", APIKey: "key"})
	raw, err := client.ChatRaw(context.Background(), ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Choices) != 1 || len(result.Choices[0].Message.Content) != 1 || result.Choices[0].Message.Content[0].Type != "image_url" {
		t.Fatalf("raw response = %s, err=%v", raw, err)
	}
}
