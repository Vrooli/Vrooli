package health

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"resource-openrouter/cli/internal/auth"
	resourceenv "resource-openrouter/cli/internal/env"
)

type captureHTTPClient struct {
	body []byte
}

func (c *captureHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var err error
	c.body, err = io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"ok"}}]}`)),
		Header:     make(http.Header),
	}, nil
}

func TestGenerateEncodesImageContentPartsAtProviderBoundary(t *testing.T) {
	client := &captureHTTPClient{}
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	data := base64.StdEncoding.EncodeToString([]byte("image"))
	if _, err := Generate(context.Background(), client, runtime, auth.Credentials{APIKey: "test-key"}, "vision-model", "describe", 0, 32, nil, []ImageInput{{MediaType: "image/png", DataB64: data}}); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Messages []struct {
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(bytes.NewReader(client.body)).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Messages) != 1 || len(payload.Messages[0].Content) != 2 {
		t.Fatalf("unexpected message payload: %s", client.body)
	}
	if payload.Messages[0].Content[1].Type != "image_url" || !strings.HasPrefix(payload.Messages[0].Content[1].ImageURL.URL, "data:image/png;base64,") {
		t.Fatalf("image content part = %+v", payload.Messages[0].Content[1])
	}
}
