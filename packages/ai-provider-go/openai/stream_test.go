package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/ai-provider-go/providererror"
)

func TestStreamChatFramesEventsAndUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("Accept = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["stream"] != true {
			t.Errorf("stream = %#v", body["stream"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, ": keepalive\r\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\r\n\r\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"cost\":0.01}}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	stream, err := New(Config{BaseURL: server.URL + "/v1", APIKey: "test-key"}).StreamChat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	first, err := stream.Next()
	if err != nil || first.Text != "hello" {
		t.Fatalf("first = %+v, err=%v", first, err)
	}
	second, err := stream.Next()
	if err != nil || second.Text != " world" {
		t.Fatalf("second = %+v, err=%v", second, err)
	}
	usage, err := stream.Next()
	if err != nil || usage.Usage.PromptTokens == nil || *usage.Usage.PromptTokens != 3 || usage.Usage.Cost == nil || *usage.Usage.Cost != 0.01 {
		t.Fatalf("usage = %+v, err=%v", usage, err)
	}
	done, err := stream.Next()
	if err != nil || !done.Done {
		t.Fatalf("done = %+v, err=%v", done, err)
	}
	if _, err := stream.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("after done error = %v", err)
	}
}

func TestStreamChatPropagatesCancellationAndClosesBody(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := New(Config{BaseURL: server.URL}).StreamChat(ctx, ChatRequest{Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	cancel()
	_, err = stream.Next()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Next after cancellation error = %v", err)
	}
	typed, ok := providererror.As(err)
	if !ok || typed.Code != providererror.CodeStreamFailed {
		t.Fatalf("cancellation error = %#v", err)
	}
	_ = stream.Close()
}

func TestStreamChatMapsHTTPErrorBeforeReturningStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"message":"slow down"}}`)
	}))
	defer server.Close()
	_, err := New(Config{BaseURL: server.URL}).StreamChat(context.Background(), ChatRequest{Model: "m"})
	typed, ok := providererror.As(err)
	if !ok || typed.Code != providererror.CodeRateLimited || typed.RetryAfter != "7" {
		t.Fatalf("error = %#v", err)
	}
}

func TestStreamChatRejectsMalformedAndOversizedEvents(t *testing.T) {
	for name, payload := range map[string]string{
		"malformed": "data: {not json}\n\n",
		"oversized": "data: " + strings.Repeat("x", maxSSELineBytes+1) + "\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, payload)
			}))
			defer server.Close()
			stream, err := New(Config{BaseURL: server.URL}).StreamChat(context.Background(), ChatRequest{Model: "m"})
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()
			_, err = stream.Next()
			typed, ok := providererror.As(err)
			if !ok || typed.Code != providererror.CodeStreamFailed {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestStreamChatBoundsCumulativeSSEEventAndAcceptsMultilineData(t *testing.T) {
	t.Run("many individually valid lines exceed cumulative event limit", func(t *testing.T) {
		line := strings.Repeat("x", maxSSELineBytes/2)
		var payload strings.Builder
		for i := 0; i < 5; i++ {
			payload.WriteString("data: ")
			payload.WriteString(line)
			payload.WriteByte('\n')
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, payload.String())
		}))
		defer server.Close()

		stream, err := New(Config{BaseURL: server.URL}).StreamChat(context.Background(), ChatRequest{Model: "m"})
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		_, err = stream.Next()
		typed, ok := providererror.As(err)
		if !ok || typed.Code != providererror.CodeStreamFailed || !strings.Contains(typed.Message, "SSE event exceeds") {
			t.Fatalf("error = %#v", err)
		}
	})

	t.Run("valid multiline JSON event", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"choices\":[\ndata: {\"delta\":{\"content\":\"multiline\"}}]}\n\n")
		}))
		defer server.Close()

		stream, err := New(Config{BaseURL: server.URL}).StreamChat(context.Background(), ChatRequest{Model: "m"})
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		chunk, err := stream.Next()
		if err != nil || chunk.Text != "multiline" {
			t.Fatalf("chunk = %+v, err=%v", chunk, err)
		}
	})
}

func TestStreamNextReturnsEventsIncrementally(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"one\"}}]}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"two\"}}]}\n\n")
	}))
	defer server.Close()
	stream, err := New(Config{BaseURL: server.URL}).StreamChat(context.Background(), ChatRequest{Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	chunk, err := stream.Next()
	if err != nil || chunk.Text != "one" {
		t.Fatalf("chunk=%+v err=%v", chunk, err)
	}
	chunk, err = stream.Next()
	if err != nil || chunk.Text != "two" {
		t.Fatalf("second chunk=%+v err=%v", chunk, err)
	}
}
