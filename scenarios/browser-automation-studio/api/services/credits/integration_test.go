package credits

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// BAS→LPBS Integration Tests
//
// These tests use httptest.NewServer to mock the LPBS server and verify
// the full HTTP flow including auth headers, JSON marshaling, and retry logic.
// ============================================================================

// mockLPBSOptions configures the mock LPBS server behavior.
type mockLPBSOptions struct {
	requireAuth   bool
	expectedToken string
	failCount     int32         // Fail first N requests (for retry testing), use int32 for atomic ops
	responseDelay time.Duration // Delay before responding
	statusCode    int           // Response status code (default: 200)
}

// lpbsRequest captures a request to the mock LPBS server.
type lpbsRequest struct {
	Method      string
	Path        string
	AuthHeader  string
	ContentType string
	Body        LPBSUsageReport
}

// mockLPBSServer wraps the captured requests with thread-safe access.
type mockLPBSServer struct {
	Server      *httptest.Server
	mu          sync.Mutex
	requests    []lpbsRequest
	failedCount int32
}

// GetRequests returns a copy of all captured requests (thread-safe).
func (m *mockLPBSServer) GetRequests() []lpbsRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]lpbsRequest, len(m.requests))
	copy(result, m.requests)
	return result
}

// Len returns the number of captured requests (thread-safe).
func (m *mockLPBSServer) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests)
}

// Close shuts down the mock server.
func (m *mockLPBSServer) Close() {
	m.Server.Close()
}

// URL returns the mock server URL.
func (m *mockLPBSServer) URL() string {
	return m.Server.URL
}

// mockLPBSHTTPServer creates a mock LPBS server for testing.
// Returns a mockLPBSServer with thread-safe access to captured requests.
func mockLPBSHTTPServer(t *testing.T, opts mockLPBSOptions) *mockLPBSServer {
	t.Helper()

	mock := &mockLPBSServer{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request details
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var report LPBSUsageReport
		_ = json.Unmarshal(body, &report)

		req := lpbsRequest{
			Method:      r.Method,
			Path:        r.URL.Path,
			AuthHeader:  r.Header.Get("Authorization"),
			ContentType: r.Header.Get("Content-Type"),
			Body:        report,
		}

		mock.mu.Lock()
		mock.requests = append(mock.requests, req)
		mock.mu.Unlock()

		// Check auth if required
		if opts.requireAuth {
			expectedAuth := "Bearer " + opts.expectedToken
			if r.Header.Get("Authorization") != expectedAuth {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		// Simulate failures for retry testing
		currentFailCount := atomic.AddInt32(&mock.failedCount, 1) - 1
		if currentFailCount < opts.failCount {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Apply response delay
		if opts.responseDelay > 0 {
			time.Sleep(opts.responseDelay)
		}

		// Return success (or custom status code)
		statusCode := opts.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		w.WriteHeader(statusCode)
		w.Write([]byte(`{"success":true}`))
	}))

	mock.Server = server
	return mock
}

// createTestServiceWithLPBSURL creates a credit service configured to use a mock LPBS server.
func createTestServiceWithLPBSURL(t *testing.T, lpbsURL, lpbsSecret string) (*Service, *sql.DB) {
	t.Helper()

	db := createTestDB(t)
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(ServiceOptions{
		DB:           db,
		Logger:       log,
		Dialect:      "sqlite",
		LPBSURL:      lpbsURL,
		LPBSSecret:   lpbsSecret,
		AppBundleKey: "browser-automation-studio",
	})

	return svc, db
}

func TestIntegration_LPBSUsageReport_FullHTTPFlow(t *testing.T) {
	// Create mock LPBS server
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{
		requireAuth:   true,
		expectedToken: "test-lpbs-secret",
	})
	defer mock.Close()

	// Create service pointing to mock server
	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-lpbs-secret")
	defer db.Close()

	ctx := context.Background()

	// Perform a charge operation
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity:    "test@example.com",
		Operation:       OpAIWorkflowGenerate,
		IsBYOK:          false,
		ActualCostCents: 0.5,
		Metadata: ChargeMetadata{
			Model:        "gpt-4",
			PromptTokens: 1500,
		},
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Wait for async LPBS report to complete
	time.Sleep(200 * time.Millisecond)

	// Verify LPBS received the request
	requests := mock.GetRequests()
	if len(requests) == 0 {
		t.Fatal("Expected LPBS to receive a request, got none")
	}

	req := requests[0]

	// Verify request details
	if req.Method != http.MethodPost {
		t.Errorf("Expected POST method, got %s", req.Method)
	}
	if req.Path != "/api/v1/usage/report" {
		t.Errorf("Expected path '/api/v1/usage/report', got %s", req.Path)
	}
	if req.ContentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", req.ContentType)
	}

	// Verify payload
	if req.Body.UserIdentity != "test@example.com" {
		t.Errorf("Expected UserIdentity 'test@example.com', got '%s'", req.Body.UserIdentity)
	}
	if req.Body.LimitKey != "ai_credits" {
		t.Errorf("Expected LimitKey 'ai_credits', got '%s'", req.Body.LimitKey)
	}
	if req.Body.AppBundleKey != "browser-automation-studio" {
		t.Errorf("Expected AppBundleKey 'browser-automation-studio', got '%s'", req.Body.AppBundleKey)
	}
}

func TestIntegration_LPBSUsageReport_AuthTokenValidation(t *testing.T) {
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{
		requireAuth:   true,
		expectedToken: "correct-token",
	})
	defer mock.Close()

	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "correct-token")
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Wait for async LPBS report
	time.Sleep(200 * time.Millisecond)

	requests := mock.GetRequests()
	if len(requests) == 0 {
		t.Fatal("Expected LPBS to receive a request")
	}

	// Verify Authorization header format
	expectedAuth := "Bearer correct-token"
	if requests[0].AuthHeader != expectedAuth {
		t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, requests[0].AuthHeader)
	}
}

func TestIntegration_LPBSUsageReport_RequestBodyFormat(t *testing.T) {
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{})
	defer mock.Close()

	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-secret")
	defer db.Close()

	ctx := context.Background()

	actualCostCents := 1.5 // $0.015 = 1.5 cents

	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity:    "user@example.com",
		Operation:       OpAIVisionNavigate,
		IsBYOK:          false,
		ActualCostCents: actualCostCents,
		Metadata: ChargeMetadata{
			Model:        "gpt-4-vision",
			PromptTokens: 2000,
		},
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	requests := mock.GetRequests()
	if len(requests) == 0 {
		t.Fatal("Expected LPBS to receive a request")
	}

	body := requests[0].Body

	// Verify JSON structure
	if body.UserIdentity != "user@example.com" {
		t.Errorf("Expected user_identity 'user@example.com', got '%s'", body.UserIdentity)
	}
	if body.LimitKey != "ai_credits" {
		t.Errorf("Expected limit_key 'ai_credits', got '%s'", body.LimitKey)
	}
	if body.AppBundleKey != "browser-automation-studio" {
		t.Errorf("Expected app_bundle_key 'browser-automation-studio', got '%s'", body.AppBundleKey)
	}

	// UsageAmount should be actualCostCents * 1,000,000
	expectedUsage := int64(actualCostCents * 1_000_000) // 1.5 * 1M = 1,500,000
	if body.UsageAmount != expectedUsage {
		t.Errorf("Expected usage_amount %d, got %d", expectedUsage, body.UsageAmount)
	}

	// Verify metadata
	if body.Metadata.Operation != string(OpAIVisionNavigate) {
		t.Errorf("Expected operation '%s', got '%s'", OpAIVisionNavigate, body.Metadata.Operation)
	}
	if body.Metadata.Model != "gpt-4-vision" {
		t.Errorf("Expected model 'gpt-4-vision', got '%s'", body.Metadata.Model)
	}
	if body.Metadata.PromptTokens != 2000 {
		t.Errorf("Expected prompt_tokens 2000, got %d", body.Metadata.PromptTokens)
	}
}

func TestIntegration_LPBSUsageReport_ServerUnavailable(t *testing.T) {
	// Create server that always fails
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{
		failCount: 100, // Always fail
	})
	defer mock.Close()

	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-secret")
	defer db.Close()

	ctx := context.Background()

	// Charge should succeed even when LPBS is unavailable
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() should not return error when LPBS fails: %v", err)
	}

	if !result.WasCharged {
		t.Error("Expected WasCharged=true even when LPBS fails")
	}

	// Verify local operation was logged
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query operation_log: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 local operation log entry, got %d", count)
	}
}

func TestIntegration_LPBSUsageReport_RetryWithBackoff(t *testing.T) {
	// Server fails first 2 times, then succeeds
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{
		failCount: 2,
	})
	defer mock.Close()

	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-secret")
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Wait for retries (500ms + 1s + some buffer = 2s)
	time.Sleep(2500 * time.Millisecond)

	// Should have 3 attempts (2 failures + 1 success)
	if mock.Len() < 3 {
		t.Errorf("Expected at least 3 retry attempts, got %d", mock.Len())
	}
}

func TestIntegration_LPBSUsageReport_AuthFailure(t *testing.T) {
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{
		requireAuth:   true,
		expectedToken: "correct-token",
	})
	defer mock.Close()

	// Use wrong token
	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "wrong-token")
	defer db.Close()

	ctx := context.Background()

	// Charge should still succeed locally
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() should not error locally: %v", err)
	}

	if !result.WasCharged {
		t.Error("Expected WasCharged=true despite LPBS auth failure")
	}

	// Wait for LPBS request
	time.Sleep(200 * time.Millisecond)

	// Verify request was made (and presumably rejected by LPBS)
	if mock.Len() == 0 {
		t.Error("Expected at least one request to LPBS")
	}
}

func TestIntegration_LPBSUsageReport_BYOKOperation(t *testing.T) {
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{})
	defer mock.Close()

	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-secret")
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       true, // BYOK operation
		Metadata: ChargeMetadata{
			Model: "claude-3-opus",
		},
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	requests := mock.GetRequests()
	if len(requests) == 0 {
		t.Fatal("Expected LPBS to receive a request for BYOK operations")
	}

	body := requests[0].Body

	// BYOK operations should report 0 usage
	if body.UsageAmount != 0 {
		t.Errorf("Expected usage_amount=0 for BYOK, got %d", body.UsageAmount)
	}

	// Metadata should indicate BYOK
	if !body.Metadata.IsBYOK {
		t.Error("Expected is_byok=true in metadata")
	}
}

func TestIntegration_LPBSUsageReport_MultipleCharges(t *testing.T) {
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{})
	defer mock.Close()

	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-secret")
	defer db.Close()

	ctx := context.Background()

	// Perform multiple charges
	for i := 0; i < 3; i++ {
		_, err := svc.Charge(ctx, ChargeRequest{
			UserIdentity: "test@example.com",
			Operation:    OpAIWorkflowGenerate,
			IsBYOK:       false,
		})
		if err != nil {
			t.Fatalf("Charge() %d returned error: %v", i, err)
		}
	}

	// Wait for all async reports
	time.Sleep(500 * time.Millisecond)

	// Should have 3 reports to LPBS
	if mock.Len() != 3 {
		t.Errorf("Expected 3 LPBS reports, got %d", mock.Len())
	}
}

func TestIntegration_LPBSUsageReport_NoURLConfigured(t *testing.T) {
	// Create service without LPBS URL
	db := createTestDB(t)
	defer db.Close()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(ServiceOptions{
		DB:      db,
		Logger:  log,
		Dialect: "sqlite",
		// No LPBSURL configured
	})

	ctx := context.Background()

	// Should work without LPBS
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	if !result.WasCharged {
		t.Error("Expected WasCharged=true")
	}
}

func TestIntegration_LPBSUsageReport_DifferentOperationTypes(t *testing.T) {
	mock := mockLPBSHTTPServer(t, mockLPBSOptions{})
	defer mock.Close()

	svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-secret")
	defer db.Close()

	ctx := context.Background()

	operations := []OperationType{
		OpAIWorkflowGenerate,
		OpAIWorkflowModify,
		OpAIVisionNavigate,
		OpAIElementAnalyze,
	}

	for _, op := range operations {
		_, err := svc.Charge(ctx, ChargeRequest{
			UserIdentity: "test@example.com",
			Operation:    op,
			IsBYOK:       false,
		})
		if err != nil {
			t.Fatalf("Charge() for %s returned error: %v", op, err)
		}
	}

	time.Sleep(500 * time.Millisecond)

	requests := mock.GetRequests()
	if len(requests) != len(operations) {
		t.Errorf("Expected %d LPBS reports, got %d", len(operations), len(requests))
	}

	// Verify each operation type was reported
	reportedOps := make(map[string]bool)
	for _, req := range requests {
		reportedOps[req.Body.Metadata.Operation] = true
	}

	for _, op := range operations {
		if !reportedOps[string(op)] {
			t.Errorf("Operation %s was not reported to LPBS", op)
		}
	}
}

func TestIntegration_LPBSUsageReport_CostCalculation(t *testing.T) {
	testCases := []struct {
		name            string
		actualCostCents float64
		expectedUsage   int64
	}{
		{"1 cent", 1.0, 1_000_000},
		{"10 cents", 10.0, 10_000_000},
		{"0.1 cents", 0.1, 100_000},
		{"0.001 cents", 0.001, 1_000},
		{"25 cents", 25.0, 25_000_000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := mockLPBSHTTPServer(t, mockLPBSOptions{})
			defer mock.Close()

			svc, db := createTestServiceWithLPBSURL(t, mock.URL(), "test-secret")
			defer db.Close()

			ctx := context.Background()

			_, err := svc.Charge(ctx, ChargeRequest{
				UserIdentity:    "test@example.com",
				Operation:       OpAIWorkflowGenerate,
				IsBYOK:          false,
				ActualCostCents: tc.actualCostCents,
			})
			if err != nil {
				t.Fatalf("Charge() returned error: %v", err)
			}

			time.Sleep(200 * time.Millisecond)

			requests := mock.GetRequests()
			if len(requests) == 0 {
				t.Fatal("Expected LPBS to receive a request")
			}

			if requests[0].Body.UsageAmount != tc.expectedUsage {
				t.Errorf("Expected usage_amount %d, got %d", tc.expectedUsage, requests[0].Body.UsageAmount)
			}
		})
	}
}
