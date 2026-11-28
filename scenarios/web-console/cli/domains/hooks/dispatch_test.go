package hooks

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func withStdin(t *testing.T, body string) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString(body); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = previous; reader.Close() })
}

func TestPromptDispatchOutsideAWebConsoleSessionIsANoOp(t *testing.T) {
	t.Setenv("WC_WEB_CONSOLE_SESSION_ID", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a session Web Console did not launch must not post its prompt")
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	withStdin(t, `{"prompt":"hello","session_id":"terminal"}`)
	if err := dispatch([]string{"--event", promptEvent, "--url", server.URL, "--token", "t"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
}

func TestPromptDispatchInsideAWebConsoleSessionPosts(t *testing.T) {
	t.Setenv("WC_WEB_CONSOLE_SESSION_ID", "wc-1")
	posted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posted = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	withStdin(t, `{"prompt":"hello"}`)
	if err := dispatch([]string{"--event", promptEvent, "--url", server.URL, "--token", "t"}); err != nil || !posted {
		t.Fatalf("dispatch err=%v posted=%t", err, posted)
	}
}
