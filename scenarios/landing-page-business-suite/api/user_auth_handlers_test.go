package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMagicLinkRequestHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	rateLimiter := NewRateLimiter(5, 15*time.Minute)

	testEmail := "test-magic-link-handler@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	handler := handleMagicLinkRequest(authService, rateLimiter)

	// Test valid request
	reqBody := MagicLinkRequest{Email: testEmail}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp MagicLinkResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Message == "" {
		t.Error("Expected non-empty message")
	}
}

func TestMagicLinkRequestHandler_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	handler := handleMagicLinkRequest(authService, nil)

	// Test invalid email
	reqBody := MagicLinkRequest{Email: "invalid-email"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid email, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestMagicLinkRequestHandler_RateLimiting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	// Create rate limiter with very low limit for testing
	rateLimiter := NewRateLimiter(2, 15*time.Minute)

	testEmail := "test-rate-limit@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	handler := handleMagicLinkRequest(authService, rateLimiter)

	// Make requests up to limit
	for i := 0; i < 2; i++ {
		reqBody := MagicLinkRequest{Email: testEmail}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	// Next request should be rate limited
	reqBody := MagicLinkRequest{Email: testEmail}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d for rate limited request, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestTokenRefreshHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-refresh-handler@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user and session
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	tokenPair, err := authService.createSession(ctx, user, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	handler := handleTokenRefresh(authService)

	// Test valid refresh
	reqBody := TokenRefreshRequest{RefreshToken: tokenPair.RefreshToken}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Error("Expected access_token in response")
	}

	if resp["refresh_token"] == nil || resp["refresh_token"] == "" {
		t.Error("Expected refresh_token in response")
	}

	// New refresh token should be different from old one
	if resp["refresh_token"] == tokenPair.RefreshToken {
		t.Error("Expected new refresh token to be different from old one")
	}
}

func TestTokenRefreshHandler_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	handler := handleTokenRefresh(authService)

	// Test invalid refresh token
	reqBody := TokenRefreshRequest{RefreshToken: "invalid-token"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for invalid token, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMeHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-auth-me@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Create session and get access token
	tokenPair, err := authService.createSession(ctx, user, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	handler := handleAuthMe(authService)

	// Create request with proper context (simulating middleware)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	// Validate token and add claims to context (simulating middleware)
	claims, err := authService.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	ctx = context.WithValue(req.Context(), userClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	userResp, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected user object in response")
	}

	if userResp["email"] != testEmail {
		t.Errorf("Expected email %s, got %v", testEmail, userResp["email"])
	}

	if userResp["id"] != user.ID {
		t.Errorf("Expected user ID %s, got %v", user.ID, userResp["id"])
	}
}

func TestAuthMeHandler_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	handler := handleAuthMe(authService)

	// Request without authentication context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthorized request, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestProtectedEndpointsRejectUnauthenticated(t *testing.T) {
	// This test verifies that protected endpoints return 401 without auth
	// Note: This is a basic test - full integration tests would use a test server

	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testCases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"AuthMe", handleAuthMe(authService)},
		{"Logout", handleUserLogout(authService)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			tc.handler(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s: Expected status %d, got %d", tc.name, http.StatusUnauthorized, w.Code)
			}
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-logout@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user and session
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	tokenPair, err := authService.createSession(ctx, user, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	// Validate token and get claims
	claims, err := authService.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	handler := handleUserLogout(authService)

	// Create request with context
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	ctx = context.WithValue(req.Context(), userClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify refresh token no longer works
	_, err = authService.RefreshTokens(ctx, tokenPair.RefreshToken)
	if err == nil {
		t.Error("Expected error when using refresh token after logout")
	}
	if err != ErrSessionRevoked {
		t.Errorf("Expected ErrSessionRevoked, got %v", err)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Second)

	key := "test-key"

	// Should allow up to rate limit
	for i := 0; i < 3; i++ {
		if !limiter.Allow(key) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should reject over limit
	if limiter.Allow(key) {
		t.Error("Request 4 should be rejected")
	}

	// Check remaining
	remaining := limiter.Remaining(key)
	if remaining != 0 {
		t.Errorf("Expected 0 remaining, got %d", remaining)
	}

	// Wait for window to pass
	time.Sleep(1100 * time.Millisecond)

	// Should allow again
	if !limiter.Allow(key) {
		t.Error("Request should be allowed after window passed")
	}
}

func TestMagicLinkVerifyHandler_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()

	// Use direct struct initialization to ensure consistent TTL settings
	authService := &UserAuthService{
		db:           db,
		emailService: emailService,
		jwtSecret:    []byte("test-secret-key"),
		jwtIssuer:    "test",
		accessTTL:    15 * time.Minute,
		refreshTTL:   7 * 24 * time.Hour,
		magicLinkTTL: 15 * time.Minute,
		baseURL:      "http://localhost:3000/auth/verify",
		appName:      "Test App",
	}

	testEmail := "test-verify-handler-success@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Capture the generated token
	var capturedToken string
	authService.UseTokenCallback(func(email, token, magicLink string) {
		capturedToken = token
	})

	// Request magic link
	err := authService.RequestMagicLink(ctx, testEmail, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("RequestMagicLink failed: %v", err)
	}

	handler := handleMagicLinkVerify(authService)

	// Make verification request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+capturedToken, nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Error("Expected access_token in response")
	}

	if resp["refresh_token"] == nil || resp["refresh_token"] == "" {
		t.Error("Expected refresh_token in response")
	}

	if resp["token_type"] != "Bearer" {
		t.Errorf("Expected token_type 'Bearer', got %v", resp["token_type"])
	}

	userResp, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected user object in response")
	}

	if userResp["email"] != testEmail {
		t.Errorf("Expected email %s, got %v", testEmail, userResp["email"])
	}

	if userResp["email_verified"] != true {
		t.Error("Expected email_verified to be true")
	}
}

func TestMagicLinkVerifyHandler_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()

	// Create service with very short magic link TTL
	authService := &UserAuthService{
		db:           db,
		emailService: emailService,
		jwtSecret:    []byte("test-secret-key"),
		jwtIssuer:    "test",
		accessTTL:    15 * time.Minute,
		refreshTTL:   7 * 24 * time.Hour,
		magicLinkTTL: 1 * time.Millisecond, // Very short for testing
		baseURL:      "http://localhost:3000/auth/verify",
		appName:      "Test App",
	}

	testEmail := "test-verify-handler-expired@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Capture the generated token
	var capturedToken string
	authService.UseTokenCallback(func(email, token, magicLink string) {
		capturedToken = token
	})

	// Request magic link
	err := authService.RequestMagicLink(ctx, testEmail, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("RequestMagicLink failed: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	handler := handleMagicLinkVerify(authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+capturedToken, nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for expired token, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestMagicLinkVerifyHandler_UsedToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()

	// Use direct struct initialization to ensure consistent TTL settings
	authService := &UserAuthService{
		db:           db,
		emailService: emailService,
		jwtSecret:    []byte("test-secret-key"),
		jwtIssuer:    "test",
		accessTTL:    15 * time.Minute,
		refreshTTL:   7 * 24 * time.Hour,
		magicLinkTTL: 15 * time.Minute,
		baseURL:      "http://localhost:3000/auth/verify",
		appName:      "Test App",
	}

	testEmail := "test-verify-handler-used@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Capture the generated token
	var capturedToken string
	authService.UseTokenCallback(func(email, token, magicLink string) {
		capturedToken = token
	})

	// Request magic link
	err := authService.RequestMagicLink(ctx, testEmail, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("RequestMagicLink failed: %v", err)
	}

	handler := handleMagicLinkVerify(authService)

	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+capturedToken, nil)
	w1 := httptest.NewRecorder()
	handler(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("First request should succeed, got status %d", w1.Code)
	}

	// Second request should fail
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+capturedToken, nil)
	w2 := httptest.NewRecorder()
	handler(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for used token, got %d", http.StatusUnauthorized, w2.Code)
	}
}

func TestMagicLinkVerifyHandler_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	handler := handleMagicLinkVerify(authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token=invalid-token-12345", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for invalid token, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestMagicLinkVerifyHandler_MissingToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	handler := handleMagicLinkVerify(authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing token, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	// Configure trusted proxies for tests that use XFF/X-Real-IP
	// The 10.0.0.0/8 range is used as the trusted proxy network
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	testCases := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			xff:        "192.168.1.1",
			remoteAddr: "10.0.0.1:12345", // From trusted proxy
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			xff:        "192.168.1.1, 10.0.0.2, 10.0.0.3",
			remoteAddr: "10.0.0.1:12345", // From trusted proxy
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			xri:        "192.168.1.1",
			remoteAddr: "10.0.0.1:12345", // From trusted proxy
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:12345",
			expected:   "::1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetTrustedProxies()
			os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" {
				req.Header.Set("X-Real-IP", tc.xri)
			}
			req.RemoteAddr = tc.remoteAddr

			ip := getClientIP(req)
			if ip != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, ip)
			}
		})
	}
}
