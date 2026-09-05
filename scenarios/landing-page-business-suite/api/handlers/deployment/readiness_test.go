package deployment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"landing-page-business-suite-api/internal/delivery"
)

func TestReadinessRejectsMalformedJSONBeforeDependencyAccess(t *testing.T) {
	w := httptest.NewRecorder()
	handler := Readiness(Dependencies{
		BundleKey:  func() string { t.Fatal("bundle key should not be read for malformed input"); return "" },
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
	})

	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/deploy-readiness", strings.NewReader("{")))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestConnectReadinessUsesSharedWorkflow(t *testing.T) {
	handler := NewConnectHandler(Dependencies{
		BundleKey: func() string { return "bundle" },
		Storage:   readinessStorage{},
	})

	response, err := handler.CheckReadiness(context.Background(), connect.NewRequest(&lpbsv1.CheckDeploymentReadinessRequest{}))
	if err != nil {
		t.Fatalf("CheckReadiness: %v", err)
	}
	if response.Msg.GetReady() {
		t.Fatal("expected missing storage to make readiness false")
	}
	if got := response.Msg.GetGates()[0].GetName(); got != "download_storage" {
		t.Fatalf("gate name=%q, want download_storage", got)
	}
}

type readinessStorage struct{}

func (readinessStorage) GetSettings(context.Context, string) (*delivery.StorageSettings, error) {
	return nil, nil
}
