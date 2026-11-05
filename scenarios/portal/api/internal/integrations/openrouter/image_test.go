package openrouter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	return b.Bytes()
}

func TestImageRequestSendsTextThenExactImageAndStreamsResponse(t *testing.T) {
	pixels := testPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		var req struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Messages, 2)
		require.JSONEq(t, `"System prompt"`, string(req.Messages[0].Content))
		var parts []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			Image struct {
				URL string `json:"url"`
			} `json:"image_url"`
		}
		require.NoError(t, json.Unmarshal(req.Messages[1].Content, &parts))
		require.Len(t, parts, 2)
		require.Equal(t, "text", parts[0].Type)
		require.Equal(t, "Describe this crop", parts[0].Text)
		require.Equal(t, "image_url", parts[1].Type)
		require.Equal(t, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(pixels), parts[1].Image.URL)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"A crop\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()
	client, err := NewClient(Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)
	var text string
	require.NoError(t, client.StreamCompletion(context.Background(), CompletionRequest{Model: "vision-fixture", Messages: []Message{{Role: "system", Content: "System prompt"}, {Role: "user", Content: "Describe this crop", Images: [][]byte{pixels}}}}, func(event StreamEvent) error { text += event.Token; return nil }))
	require.Equal(t, "A crop", text)
}

func TestInvalidImagesNeverReachProvider(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) }))
	defer server.Close()
	client, err := NewClient(Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)
	pixels := testPNG(t)
	for _, messages := range [][]Message{
		{{Role: "user", Images: [][]byte{[]byte("private-invalid-image")}}},
		{{Role: "system", Images: [][]byte{pixels}}},
		{{Role: "user", Images: [][]byte{pixels, pixels, pixels}}, {Role: "user", Images: [][]byte{pixels, pixels}}},
	} {
		require.Error(t, client.StreamCompletion(context.Background(), CompletionRequest{Messages: messages}, func(StreamEvent) error { return nil }))
	}
	require.Zero(t, calls)
}

func TestPrivateImageRejectionIsNotRetriedAndDoesNotEchoProviderBody(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(429)
		_, _ = w.Write([]byte("private-image-content-echo"))
	}))
	defer server.Close()
	client, err := NewClient(Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)
	err = client.StreamCompletion(context.Background(), CompletionRequest{Messages: []Message{{Role: "user", Content: "Review", Images: [][]byte{testPNG(t)}}}}, func(StreamEvent) error { return nil })
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private-image-content-echo")
	require.Equal(t, 1, calls)
}
