package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/administration"
)

func TestMagicLinkRequestHandler(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)
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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)
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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

	testEmail := "test-refresh-handler@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user and session
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	tokenPair, err := authService.CreateSession(ctx, user, "127.0.0.1", "Test-Agent")
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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

	testEmail := "test-auth-me@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Create session and get access token
	tokenPair, err := authService.CreateSession(ctx, user, "127.0.0.1", "Test-Agent")
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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

	testEmail := "test-logout@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user and session
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	tokenPair, err := authService.CreateSession(ctx, user, "127.0.0.1", "Test-Agent")
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
	if err != administration.ErrSessionRevoked {
		t.Errorf("Expected administration.ErrSessionRevoked, got %v", err)
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

	emailService := NewEmailService()

	// Use direct struct initialization to ensure consistent TTL settings
	authService := newUserAuthServiceForTest(db, emailService)

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

	emailService := NewEmailService()

	// Create service with very short magic link TTL
	authService := newUserAuthServiceForTestWithOptions(db, emailService, 15*time.Minute, 7*24*time.Hour, time.Millisecond)

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

	emailService := NewEmailService()

	// Use direct struct initialization to ensure consistent TTL settings
	authService := newUserAuthServiceForTest(db, emailService)

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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

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

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

	handler := handleMagicLinkVerify(authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing token, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- formatNullableTime Tests ---

func TestFormatNullableTime_Nil(t *testing.T) {
	result := formatNullableTime(nil)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}
}

func TestFormatNullableTime_ValidTime(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := formatNullableTime(&testTime)

	expected := "2024-01-15T10:30:00Z"
	if result != expected {
		t.Errorf("Expected '%s', got '%v'", expected, result)
	}
}

// --- isSecureContext Tests ---

func TestIsSecureContext_HTTPS(t *testing.T) {
	// Set env variable for HTTPS base URL
	os.Setenv("AUTH_MAGIC_LINK_BASE_URL", "https://example.com/auth/verify")
	defer os.Unsetenv("AUTH_MAGIC_LINK_BASE_URL")

	result := isSecureContext()
	if !result {
		t.Error("Expected isSecureContext to return true for HTTPS URL")
	}
}

func TestIsSecureContext_HTTP(t *testing.T) {
	// Set env variable for HTTP base URL
	os.Setenv("AUTH_MAGIC_LINK_BASE_URL", "http://localhost:3000/auth/verify")
	defer os.Unsetenv("AUTH_MAGIC_LINK_BASE_URL")

	result := isSecureContext()
	if result {
		t.Error("Expected isSecureContext to return false for HTTP URL")
	}
}

func TestIsSecureContext_Empty(t *testing.T) {
	// Clear env variable
	os.Unsetenv("AUTH_MAGIC_LINK_BASE_URL")

	result := isSecureContext()
	// Empty string doesn't start with "https://"
	if result {
		t.Error("Expected isSecureContext to return false for empty URL")
	}
}

// --- setAuthCookies Tests ---

func TestSetAuthCookies_Attributes(t *testing.T) {
	// Ensure we're in non-secure context for predictable testing
	os.Setenv("AUTH_MAGIC_LINK_BASE_URL", "http://localhost:3000")
	defer os.Unsetenv("AUTH_MAGIC_LINK_BASE_URL")

	w := httptest.NewRecorder()
	tokenPair := &administration.TokenPair{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	}

	setAuthCookies(w, tokenPair)

	cookies := w.Result().Cookies()
	if len(cookies) != 2 {
		t.Errorf("Expected 2 cookies, got %d", len(cookies))
	}

	// Find access token cookie
	var accessCookie, refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "access_token" {
			accessCookie = c
		}
		if c.Name == "refresh_token" {
			refreshCookie = c
		}
	}

	if accessCookie == nil {
		t.Fatal("access_token cookie not found")
	}
	if refreshCookie == nil {
		t.Fatal("refresh_token cookie not found")
	}

	// Verify access token attributes
	if !accessCookie.HttpOnly {
		t.Error("access_token should be HttpOnly")
	}
	if accessCookie.Path != "/" {
		t.Errorf("access_token path should be '/', got '%s'", accessCookie.Path)
	}
	if accessCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("access_token SameSite should be Lax, got %v", accessCookie.SameSite)
	}

	// Verify refresh token attributes
	if !refreshCookie.HttpOnly {
		t.Error("refresh_token should be HttpOnly")
	}
	if refreshCookie.Path != "/api/v1/auth" {
		t.Errorf("refresh_token path should be '/api/v1/auth', got '%s'", refreshCookie.Path)
	}
}

// --- clearAuthCookies Tests ---

func TestClearAuthCookies(t *testing.T) {
	os.Setenv("AUTH_MAGIC_LINK_BASE_URL", "http://localhost:3000")
	defer os.Unsetenv("AUTH_MAGIC_LINK_BASE_URL")

	w := httptest.NewRecorder()

	clearAuthCookies(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 2 {
		t.Errorf("Expected 2 cookies to be cleared, got %d", len(cookies))
	}

	for _, c := range cookies {
		if c.MaxAge != -1 {
			t.Errorf("Cookie %s should have MaxAge -1 (deleted), got %d", c.Name, c.MaxAge)
		}
		if c.Value != "" {
			t.Errorf("Cookie %s should have empty value, got '%s'", c.Name, c.Value)
		}
	}
}

// --- redirectWithTokens Tests ---

func TestRedirectWithTokens_ValidURL(t *testing.T) {
	tokenPair := &administration.TokenPair{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token=xxx", nil)

	redirectWithTokens(w, req, "https://example.com/dashboard", tokenPair)

	if w.Code != http.StatusFound {
		t.Errorf("Expected redirect status %d, got %d", http.StatusFound, w.Code)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header to be set")
	}

	// Verify fragment contains tokens
	if !strings.Contains(location, "#") {
		t.Error("Expected URL to contain fragment (#)")
	}
	if !strings.Contains(location, "access_token=test-access") {
		t.Error("Expected fragment to contain access_token")
	}
	if !strings.Contains(location, "refresh_token=test-refresh") {
		t.Error("Expected fragment to contain refresh_token")
	}
}

func TestRedirectWithTokens_InvalidURL(t *testing.T) {
	tokenPair := &administration.TokenPair{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)

	// Use an invalid URL that will fail to parse
	redirectWithTokens(w, req, "://invalid-url", tokenPair)

	// Should fall back to JSON response
	if w.Code == http.StatusFound {
		t.Error("Should not redirect for invalid URL")
	}

	// Should return JSON with tokens
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Should return JSON response: %v", err)
	}

	if resp["access_token"] != "test-access" {
		t.Error("Expected access_token in JSON fallback response")
	}
}

// --- Token Refresh Cookie Tests ---

func TestTokenRefresh_FromCookie(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

	testEmail := "test-refresh-cookie@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user and session
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	tokenPair, err := authService.CreateSession(ctx, user, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	handler := handleTokenRefresh(authService)

	// Send empty body but with cookie
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: tokenPair.RefreshToken,
	})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestTokenRefresh_EmptyBodyWithCookie(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

	testEmail := "test-refresh-empty-body@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user and session
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	tokenPair, err := authService.CreateSession(ctx, user, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	handler := handleTokenRefresh(authService)

	// Send completely empty body but with cookie
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: tokenPair.RefreshToken,
	})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestTokenRefresh_MissingToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := newUserAuthServiceForTest(db, emailService)

	handler := handleTokenRefresh(authService)

	// Send empty request - no body, no cookie
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
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
