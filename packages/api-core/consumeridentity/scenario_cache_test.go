package consumeridentity

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScenarioJWKSFetcherUsesRequestContextAndBlocksRedirects(t *testing.T) {
	redirectTargetHits := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectTargetHits++
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			t.Errorf("request path=%q", r.URL.Path)
		}
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()

	ctxKey := struct{}{}
	ctx := context.WithValue(context.Background(), ctxKey, "request-context")
	resolveCalled := false
	fetch := scenarioJWKSFetcher(func(got context.Context, slug string) (string, error) {
		resolveCalled = true
		if got.Value(ctxKey) != "request-context" {
			t.Error("scenario resolution did not receive the request context")
		}
		if slug != "landing-page-business-suite" {
			t.Errorf("scenario slug=%q", slug)
		}
		return server.URL, nil
	}, &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, "landing-page-business-suite")
	_, err := fetch(httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))
	if err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatalf("redirect fetch err=%v; want bounded non-200 failure", err)
	}
	if !resolveCalled {
		t.Fatal("scenario resolver was not called")
	}
	if redirectTargetHits != 0 {
		t.Fatalf("redirect target received %d requests", redirectTargetHits)
	}
}

func TestScenarioJWKSFetcherPropagatesRequestCancellationToResolution(t *testing.T) {
	wantErr := errors.New("request canceled")
	fetch := scenarioJWKSFetcher(func(ctx context.Context, _ string) (string, error) {
		if ctx.Err() == nil {
			return "", errors.New("request context was not canceled")
		}
		return "", wantErr
	}, http.DefaultClient, "trusted-scenario")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := fetch(httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))
	if !errors.Is(err, wantErr) {
		t.Fatalf("fetch err=%v; want resolver error", err)
	}
}
