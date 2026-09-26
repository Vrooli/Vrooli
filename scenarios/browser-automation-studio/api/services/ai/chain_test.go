package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type chainSubscriptionResolver struct {
	token string
	calls int
}

func (r *chainSubscriptionResolver) ResolveAt(_ context.Context, baseURL string) (credentialclient.ConsumerAccess, error) {
	r.calls++
	if !strings.HasPrefix(baseURL, "http://") {
		return credentialclient.ConsumerAccess{}, context.Canceled
	}
	return credentialclient.ConsumerAccess{AccessToken: r.token, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func TestAIProviderChainResolvesSharedSubscriptionTokenForVrooliProvider(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		if r.URL.Path == "/api/v1/ai/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":"ok","model":"policy","prompt_tokens":1,"completion_tokens":1}`))
	}))
	defer server.Close()
	resolver := &chainSubscriptionResolver{token: "consumer-access"}
	chain := NewAIProviderChain(AIProviderChainOptions{
		Logger:       logrus.New(),
		EnableVrooli: true, VrooliAPIURL: server.URL, DefaultModel: "policy",
		SubscriptionResolver: resolver,
	})
	result, err := chain.Execute(context.Background(), ProviderRequest{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != ProviderTypeVrooli || gotAuthorization != "Bearer consumer-access" || resolver.calls != 1 {
		t.Fatalf("result = %#v authorization = %q resolver calls = %d", result, gotAuthorization, resolver.calls)
	}
}
