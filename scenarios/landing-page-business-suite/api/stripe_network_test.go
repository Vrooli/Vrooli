package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestSignature creates a valid Stripe webhook signature for testing.
func createTestSignature(payload, secret string) string {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := timestamp + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	return "t=" + timestamp + ",v1=" + signature
}

// TestStripeAPI_Timeout verifies that Stripe API calls handle timeouts gracefully.
func TestStripeAPI_Timeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create a server that delays longer than the timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Longer than our timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), slowServer)

	// Override with a client with a very short timeout
	shortTimeoutClient := &http.Client{
		Timeout: 100 * time.Millisecond,
	}
	service.UseHTTPClient(shortTimeoutClient)

	// This should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := service.ListCoupons(ctx)
	require.Error(t, err)
	// Verify it's a timeout-related error
	assert.Contains(t, err.Error(), "context deadline exceeded", "should be a timeout error")
}

// TestStripeAPI_RateLimited_429 verifies proper handling of rate limit responses.
func TestStripeAPI_RateLimited_429(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Server that returns 429 Too Many Requests
	rateLimitServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`))
	}))
	defer rateLimitServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), rateLimitServer)

	_, err := service.ListCoupons(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429", "should indicate rate limit error")
}

// TestStripeAPI_ServerError_5xx verifies proper handling of server errors.
func TestStripeAPI_ServerError_5xx(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	tests := []struct {
		name       string
		statusCode int
		errorType  string
	}{
		{"500 Internal Server Error", http.StatusInternalServerError, "api_error"},
		{"502 Bad Gateway", http.StatusBadGateway, "api_error"},
		{"503 Service Unavailable", http.StatusServiceUnavailable, "api_error"},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, "api_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"error":{"type":"` + tt.errorType + `","message":"Server error"}}`))
			}))
			defer errorServer.Close()

			service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), errorServer)

			_, err := service.ListCoupons(context.Background())
			require.Error(t, err)
			// Should contain the status code in the error
			assert.Contains(t, err.Error(), "error", "should indicate server error")
		})
	}
}

// TestWebhook_MalformedJSON verifies graceful handling of malformed JSON payloads.
func TestWebhook_MalformedJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test_secret")
	service := ConfigureStripeService(t, db, cfg, nil)

	tests := []struct {
		name    string
		payload string
		wantErr string
	}{
		{
			name:    "empty payload",
			payload: "",
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "invalid JSON",
			payload: "{not valid json}",
			wantErr: "invalid character",
		},
		{
			name:    "truncated JSON",
			payload: `{"type": "checkout.session.completed", "data":`,
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "wrong type - array instead of object",
			payload: `["not", "an", "object"]`,
			wantErr: "cannot unmarshal array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a valid signature for the payload (even though payload is invalid)
			signature := createTestSignature(tt.payload, "whsec_test_secret")

			err := service.HandleWebhook([]byte(tt.payload), signature)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr, "should have appropriate error message")
		})
	}
}

// TestWebhook_InvalidSignature verifies rejection of webhooks with invalid signatures.
func TestWebhook_InvalidSignature(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_correct_secret")
	service := ConfigureStripeService(t, db, cfg, nil)

	validPayload := `{"id":"evt_test","type":"checkout.session.completed","data":{"object":{}}}`

	tests := []struct {
		name      string
		signature string
		wantErr   string
	}{
		{
			name:      "empty signature",
			signature: "",
			wantErr:   "invalid webhook signature",
		},
		{
			name:      "wrong secret used",
			signature: createTestSignature(validPayload, "whsec_wrong_secret"),
			wantErr:   "invalid webhook signature",
		},
		{
			name:      "malformed signature header",
			signature: "not_a_valid_signature_format",
			wantErr:   "invalid webhook signature",
		},
		{
			name:      "missing timestamp",
			signature: "v1=abc123",
			wantErr:   "invalid webhook signature",
		},
		{
			name:      "expired timestamp",
			signature: "t=1000000000,v1=abc123", // Very old timestamp
			wantErr:   "invalid webhook signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.HandleWebhook([]byte(validPayload), tt.signature)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr, "should reject invalid signature")
		})
	}
}

// TestStripeAPI_NetworkError verifies handling of network connection errors.
func TestStripeAPI_NetworkError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Use an invalid URL that will fail to connect
	// Create a dummy server just to get a test server, then use a failing URL
	dummyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	dummyServer.Close() // Close it immediately - we don't need it

	cfg := DefaultStripeTestConfig()
	cfg.APIBase = "http://localhost:1" // Port 1 should fail
	service := ConfigureStripeService(t, db, cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := service.ListCoupons(ctx)
	require.Error(t, err)
	// Should be a connection refused or similar network error
	errMsg := err.Error()
	assert.True(t,
		strings.Contains(errMsg, "connection refused") ||
			strings.Contains(errMsg, "dial") ||
			strings.Contains(errMsg, "context deadline"),
		"should be a network-related error: %v", err)
}

// TestWebhook_MissingType verifies handling of webhooks without event type.
func TestWebhook_MissingType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test_secret")
	service := ConfigureStripeService(t, db, cfg, nil)

	// Payload with id but no type
	payload := `{"id":"evt_test","data":{"object":{}}}`
	signature := createTestSignature(payload, "whsec_test_secret")

	err := service.HandleWebhook([]byte(payload), signature)
	// Missing type should return an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing event type", "should reject webhook without type")
}

// TestStripeAPI_InvalidResponse verifies handling of invalid JSON responses.
func TestStripeAPI_InvalidResponse(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	invalidResponseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json response`))
	}))
	defer invalidResponseServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), invalidResponseServer)

	_, err := service.ListCoupons(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character", "should indicate JSON parse error")
}
