package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type artifactDoerFunc func(*http.Request) (*http.Response, error)

func (f artifactDoerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestProxyRunArtifactUsesInjectedHTTPDoer(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/artifact", nil)
	request.Header.Set("Range", "bytes=0-3")
	called := false
	doer := artifactDoerFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		if got := r.Header.Get("Range"); got != "bytes=0-3" {
			t.Fatalf("Range=%q", got)
		}
		return &http.Response{StatusCode: http.StatusPartialContent, Header: http.Header{"Content-Type": []string{"video/mp4"}}, Body: io.NopCloser(strings.NewReader("clip"))}, nil
	})
	if err := proxyRunArtifact(context.Background(), recorder, request, doer, "http://test-genie/artifact"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("injected HTTP doer was not called")
	}
	if recorder.Code != http.StatusPartialContent || recorder.Body.String() != "clip" {
		t.Fatalf("response status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

type captureLogger struct{ messages []string }

func (l *captureLogger) Printf(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}
func (l *captureLogger) Infof(format string, args ...any) { l.Printf(format, args...) }

func TestServerLogfUsesInjectedLogger(t *testing.T) {
	logger := &captureLogger{}
	s := &Server{Logger: logger}
	s.logf("operation %s failed", "capture")
	if len(logger.messages) != 1 || logger.messages[0] != "operation capture failed" {
		t.Fatalf("messages=%v", logger.messages)
	}
}

func TestLoginWithAuthenticatorUsesInjectedHTTPDoer(t *testing.T) {
	var called bool
	var doer artifactDoerFunc = func(r *http.Request) (*http.Response, error) {
		called = true
		if r.URL.Host != "auth.test" {
			t.Fatalf("request host=%q", r.URL.Host)
		}
		return nil, context.DeadlineExceeded
	}
	_, err := loginWithAuthenticator(context.Background(), doer, "https://auth.test/", "user@example.test", "secret")
	if err == nil || !called {
		t.Fatalf("called=%v err=%v", called, err)
	}
}
