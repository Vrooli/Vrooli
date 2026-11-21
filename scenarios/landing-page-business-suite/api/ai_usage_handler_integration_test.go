package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aihandler "landing-page-business-suite-api/handlers/intelligence"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/commerce"
)

// TestAIUsageHandler_Success keeps the database-backed usage endpoint covered
// at the composition boundary. Endpoint behavior belongs to handlers/intelligence;
// this test owns the root-only database fixture and authentication context.
func TestAIUsageHandler_Success(t *testing.T) {
	db := setupTestDB(t)
	limitsService := NewLimitsService(db, "sqlite")
	usageService := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		DB:            db,
		LimitsService: limitsService,
		Dialect:       "sqlite",
	})

	handler := aihandler.New(aihandler.Dependencies{
		Usage: func(ctx context.Context, identity, tier string) (aihandler.UsageSummary, error) {
			summary, err := usageService.GetUsageSummary(ctx, identity, tier)
			if err != nil {
				return aihandler.UsageSummary{}, err
			}
			return aihandler.UsageSummary{BillingPeriod: summary.BillingPeriod, ResetDate: summary.ResetDate, Usage: summary.Usage, Limits: summary.Limits, Remaining: summary.Remaining}, nil
		},
		UserIdentity:   getUserEmail,
		WriteJSONError: writeJSONError,
		Log:            func(string, map[string]interface{}) {},
		LogError:       func(string, map[string]interface{}) {},
	})

	claims := &administration.UserClaims{UserID: "user-123", Email: "test@example.com"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/usage", nil).
		WithContext(context.WithValue(context.Background(), userClaimsKey, claims))
	response := httptest.NewRecorder()
	handler.Usage()(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, key := range []string{"user_identity", "tier", "display"} {
		if _, ok := payload[key]; !ok {
			t.Errorf("missing %q in response: %#v", key, payload)
		}
	}
}
