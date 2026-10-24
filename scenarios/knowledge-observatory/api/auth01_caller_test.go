package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/owneridentity"
	"knowledge-observatory/internal/services/deepsearch"
)

type auth01FixtureVerifier struct{}

func (auth01FixtureVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type auth01SearchSpy struct {
	fakeDeepSearchService
	calls int
	t     *testing.T
}

func (f *auth01SearchSpy) StartSearch(ctx context.Context, _ deepsearch.DeepSearchRequest) (*deepsearch.DeepSearchJob, error) {
	f.calls++
	if owneridentity.RequireCreateRunCaller(ctx, time.Now(), true) != nil {
		f.t.Error("verified proof dropped at service")
	}
	return &deepsearch.DeepSearchJob{JobID: "fixture"}, nil
}

func TestAuth01KnowledgeIngressPositiveAndZeroEffect(t *testing.T) {
	spy := &auth01SearchSpy{t: t}
	s := &Server{docDeepSearchService: spy, runCallerValidator: auth01FixtureVerifier{}}
	for _, headers := range []http.Header{{}, {"Authorization": []string{"Bearer fixture", ""}}, {"Authorization": []string{"Bearer fixture"}, "X-Agent-Identity-Token": []string{"fixture-run"}}} {
		r := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"fixture"}`))
		r.Header = headers
		w := httptest.NewRecorder()
		s.handleDocsSearchDeep(w, r)
		if w.Code != 401 || spy.calls != 0 {
			t.Fatal("invalid ingress effects")
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"fixture"}`))
	r.Header.Set("Authorization", "Bearer fixture-owner")
	w := httptest.NewRecorder()
	s.handleDocsSearchDeep(w, r)
	if w.Code != 200 || spy.calls != 1 {
		t.Fatalf("positive status%d calls%d", w.Code, spy.calls)
	}
}
