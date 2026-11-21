package monetization

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLPBSUsageTransportPreservesIdempotentUsageContract(t *testing.T) {
	var got lpbsUsageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/usage/report" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer short-lived-token" {
			t.Errorf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode usage payload: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	calledWith := ""
	transport := NewLPBSUsageTransport(server.URL+"/", func(_ context.Context, baseURL string) (string, error) {
		calledWith = baseURL
		return "short-lived-token", nil
	}, server.Client())
	usage := Usage{
		OperationID:  "operation-123",
		UserIdentity: "user@example.com",
		BundleKey:    "business_suite",
		AppKey:       "browser-automation-studio",
		MeterKey:     "ai_credits",
		Units:        750_000,
		OccurredAt:   time.Now().UTC(),
		Metadata: map[string]string{
			"operation":     "ai.generate",
			"model":         "model-1",
			"prompt_tokens": "37",
			"is_byok":       "false",
		},
	}
	if err := transport.Report(context.Background(), usage); err != nil {
		t.Fatal(err)
	}
	if calledWith != server.URL {
		t.Fatalf("resolver base URL = %q", calledWith)
	}
	if got.UserIdentity != usage.UserIdentity || got.LimitKey != usage.MeterKey || got.UsageAmount != usage.Units || got.Amount != usage.Units || got.AppBundleKey != usage.AppKey || got.OperationID != usage.OperationID {
		t.Fatalf("report contract = %+v", got)
	}
	if got.Metadata.Operation != "ai.generate" || got.Metadata.Model != "model-1" || got.Metadata.PromptTokens != 37 || got.Metadata.IsBYOK {
		t.Fatalf("report metadata = %+v", got.Metadata)
	}
}

func TestLPBSUsageTransportFailsClosedForInvalidOrUnavailableAuthority(t *testing.T) {
	valid := Usage{
		OperationID: "operation-1", UserIdentity: "user@example.com", BundleKey: "business_suite",
		AppKey: "web-console", MeterKey: "ai_credits", Units: 1, OccurredAt: time.Now().UTC(),
	}
	if err := (&LPBSUsageTransport{}).Report(context.Background(), valid); err == nil {
		t.Fatal("unconfigured transport accepted usage")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("private authority detail"))
	}))
	defer server.Close()
	transport := NewLPBSUsageTransport(server.URL, func(context.Context, string) (string, error) { return "token", nil }, server.Client())
	if err := transport.Report(context.Background(), Usage{}); !errors.Is(err, ErrInvalidUsage) {
		t.Fatalf("invalid usage error = %v", err)
	}
	err := transport.Report(context.Background(), valid)
	if err == nil || !strings.Contains(err.Error(), "status 403") || strings.Contains(err.Error(), "private authority detail") {
		t.Fatalf("authority failure leaked response or was accepted: %v", err)
	}
	transport.ResolveToken = func(context.Context, string) (string, error) { return "", errors.New("offline") }
	if err := transport.Report(context.Background(), valid); err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("resolver failure = %v", err)
	}
}
