package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

func TestRequireAIAccessFailsClosedWithoutLease(t *testing.T) {
	m := NewEntitlementMiddleware(
		entitlement.NewService(config.EntitlementConfig{}, logrus.New()),
		logrus.New(), config.EntitlementConfig{}, nil,
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai", nil)
	rr := httptest.NewRecorder()

	m.RequireAIAccess(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler must not run without a lease")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireAIAccessUsesLeaseFeatures(t *testing.T) {
	m := NewEntitlementMiddleware(
		entitlement.NewService(config.EntitlementConfig{}, logrus.New()),
		logrus.New(), config.EntitlementConfig{}, nil,
	)

	tests := []struct {
		name     string
		features []string
		wantCode int
	}{
		{name: "feature granted", features: []string{entitlement.FeatureAI}, wantCode: http.StatusNoContent},
		{name: "feature denied despite tier", features: []string{"recording"}, wantCode: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/ai", nil)
			rr := httptest.NewRecorder()
			ctx := entitlement.WithEntitlement(req.Context(), &entitlement.Entitlement{
				UserIdentity: "user@example.com",
				Tier:         entitlement.TierFree,
				Features:     tt.features,
			})
			req = req.WithContext(ctx)

			m.RequireAIAccess(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}
