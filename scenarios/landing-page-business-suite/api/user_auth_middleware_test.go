package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRequireUserAuth_ValidBearerToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	testEmail := "test-middleware-bearer@example.com"
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

	// Track if handler was called and what claims it received
	var handlerCalled bool
	var receivedUserID string
	var receivedEmail string

	handler := server.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		receivedUserID = getUserID(r.Context())
		receivedEmail = getUserEmail(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !handlerCalled {
		t.Error("Handler should have been called")
	}

	if receivedUserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, receivedUserID)
	}

	if receivedEmail != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, receivedEmail)
	}
}

func TestRequireUserAuth_ValidCookie(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	testEmail := "test-middleware-cookie@example.com"
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

	// Track if handler was called
	var handlerCalled bool
	var receivedUserID string

	handler := server.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		receivedUserID = getUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: tokenPair.AccessToken,
	})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !handlerCalled {
		t.Error("Handler should have been called")
	}

	if receivedUserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, receivedUserID)
	}
}

func TestRequireUserAuth_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()

	// Create service with very short access TTL
	authService := &UserAuthService{
		db:           db,
		emailService: emailService,
		jwtSecret:    []byte("test-secret-key"),
		jwtIssuer:    "test",
		accessTTL:    1 * time.Millisecond, // Very short for testing
		refreshTTL:   7 * 24 * time.Hour,
		magicLinkTTL: 15 * time.Minute,
		baseURL:      "http://localhost:3000/auth/verify",
		appName:      "Test App",
	}
	server := setupMinimalAuthServer(t, authService)

	testEmail := "test-middleware-expired@example.com"
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

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	var handlerCalled bool
	handler := server.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for expired token, got %d", http.StatusUnauthorized, w.Code)
	}

	if handlerCalled {
		t.Error("Handler should not have been called for expired token")
	}
}

func TestRequireUserAuth_InvalidToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	var handlerCalled bool
	handler := server.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for invalid token, got %d", http.StatusUnauthorized, w.Code)
	}

	if handlerCalled {
		t.Error("Handler should not have been called for invalid token")
	}
}

func TestRequireUserAuth_MissingToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	var handlerCalled bool
	handler := server.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for missing token, got %d", http.StatusUnauthorized, w.Code)
	}

	if handlerCalled {
		t.Error("Handler should not have been called for missing token")
	}
}

func TestRequireUserAuth_BearerTakesPrecedence(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	testEmail1 := "test-middleware-bearer-prec@example.com"
	testEmail2 := "test-middleware-cookie-prec@example.com"
	defer cleanupUserTestData(t, db, testEmail1)
	defer cleanupUserTestData(t, db, testEmail2)

	ctx := context.Background()

	// Create two users with different sessions
	user1, err := authService.GetOrCreateUser(ctx, testEmail1)
	if err != nil {
		t.Fatalf("GetOrCreateUser 1 failed: %v", err)
	}
	tokenPair1, err := authService.createSession(ctx, user1, "127.0.0.1", "Agent-1")
	if err != nil {
		t.Fatalf("createSession 1 failed: %v", err)
	}

	user2, err := authService.GetOrCreateUser(ctx, testEmail2)
	if err != nil {
		t.Fatalf("GetOrCreateUser 2 failed: %v", err)
	}
	tokenPair2, err := authService.createSession(ctx, user2, "127.0.0.2", "Agent-2")
	if err != nil {
		t.Fatalf("createSession 2 failed: %v", err)
	}

	var receivedEmail string
	handler := server.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		receivedEmail = getUserEmail(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Send request with Bearer token (user1) and Cookie (user2)
	// Bearer should take precedence
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair1.AccessToken)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: tokenPair2.AccessToken,
	})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Should use Bearer token (user1)
	if receivedEmail != testEmail1 {
		t.Errorf("Expected Bearer token email %s to take precedence, got %s", testEmail1, receivedEmail)
	}
}

func TestOptionalUserAuth_WithValidToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	testEmail := "test-optional-valid@example.com"
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

	var handlerCalled bool
	var receivedUserID string

	handler := server.optionalUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		receivedUserID = getUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !handlerCalled {
		t.Error("Handler should have been called")
	}

	if receivedUserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, receivedUserID)
	}
}

func TestOptionalUserAuth_WithInvalidToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	var handlerCalled bool
	var receivedUserID string

	handler := server.optionalUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		receivedUserID = getUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler(w, req)

	// Should continue without error
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !handlerCalled {
		t.Error("Handler should have been called even with invalid token")
	}

	// User ID should be empty (no auth)
	if receivedUserID != "" {
		t.Errorf("Expected empty user ID for invalid token, got %s", receivedUserID)
	}
}

func TestOptionalUserAuth_NoToken(t *testing.T) {
	db := setupTestDB(t)

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)
	server := setupMinimalAuthServer(t, authService)

	var handlerCalled bool
	var receivedUserID string

	handler := server.optionalUserAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		receivedUserID = getUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Should continue without error
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !handlerCalled {
		t.Error("Handler should have been called without token")
	}

	// User ID should be empty (no auth)
	if receivedUserID != "" {
		t.Errorf("Expected empty user ID for no token, got %s", receivedUserID)
	}
}

// resetTrustedProxies resets the trusted proxies configuration for testing.
// This must be called before each test that modifies TRUSTED_PROXY_CIDRS.
func resetTrustedProxies() {
	trustedProxyCIDRs = nil
	trustedProxiesOnce = sync.Once{}
}

func TestGetClientIP_NoTrustedProxies(t *testing.T) {
	resetTrustedProxies()
	os.Unsetenv("TRUSTED_PROXY_CIDRS")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	req.Header.Set("X-Forwarded-For", "5.6.7.8")

	ip := getClientIP(req)

	// Without trusted proxies, should return the direct IP (RemoteAddr)
	if ip != "1.2.3.4" {
		t.Errorf("Expected direct IP 1.2.3.4, got %s", ip)
	}
}

func TestGetClientIP_TrustedProxyWithXFF(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.5:12345" // From trusted proxy
	req.Header.Set("X-Forwarded-For", "5.6.7.8, 10.0.0.1")

	ip := getClientIP(req)

	// With trusted proxy, should return the first IP from X-Forwarded-For
	if ip != "5.6.7.8" {
		t.Errorf("Expected client IP 5.6.7.8 from XFF, got %s", ip)
	}
}

func TestGetClientIP_UntrustedProxyXFFIgnored(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"       // NOT from trusted proxy
	req.Header.Set("X-Forwarded-For", "5.6.7.8") // Spoofed header - should be ignored

	ip := getClientIP(req)

	// Connection not from trusted proxy, XFF should be ignored
	if ip != "192.168.1.100" {
		t.Errorf("Expected direct IP 192.168.1.100 (XFF ignored), got %s", ip)
	}
}

func TestGetClientIP_TrustedProxyWithXRealIP(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "172.16.0.0/12")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "172.17.0.1:12345" // From trusted proxy
	req.Header.Set("X-Real-IP", "8.8.8.8")

	ip := getClientIP(req)

	// With trusted proxy and X-Real-IP, should return X-Real-IP
	if ip != "8.8.8.8" {
		t.Errorf("Expected client IP 8.8.8.8 from X-Real-IP, got %s", ip)
	}
}

func TestGetClientIP_InvalidXFFFormat(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.5:12345" // From trusted proxy
	req.Header.Set("X-Forwarded-For", "not-an-ip-address")

	ip := getClientIP(req)

	// Invalid IP in XFF, should fall back to RemoteAddr
	if ip != "10.0.0.5" {
		t.Errorf("Expected fallback to direct IP 10.0.0.5, got %s", ip)
	}
}

func TestGetClientIP_IPv6Address(t *testing.T) {
	resetTrustedProxies()
	os.Unsetenv("TRUSTED_PROXY_CIDRS")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[::1]:12345"

	ip := getClientIP(req)

	if ip != "::1" {
		t.Errorf("Expected IPv6 address ::1, got %s", ip)
	}
}

func TestGetClientIP_MultipleTrustedCIDRs(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	testCases := []struct {
		name       string
		remoteAddr string
		xff        string
		expected   string
	}{
		{"10.x from trusted", "10.1.2.3:12345", "1.2.3.4", "1.2.3.4"},
		{"172.x from trusted", "172.20.0.5:12345", "2.3.4.5", "2.3.4.5"},
		{"192.168.x from trusted", "192.168.1.1:12345", "3.4.5.6", "3.4.5.6"},
		{"public IP not trusted", "8.8.8.8:12345", "spoofed.ip", "8.8.8.8"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetTrustedProxies()
			os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Header.Set("X-Forwarded-For", tc.xff)

			ip := getClientIP(req)
			if ip != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, ip)
			}
		})
	}
}

func TestIsIPFromTrustedProxy(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,127.0.0.1/32")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	testCases := []struct {
		ip      string
		trusted bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"127.0.0.1", true},
		{"127.0.0.2", false}, // Not in /32
		{"192.168.1.1", false},
		{"8.8.8.8", false},
		{"invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			result := isIPFromTrustedProxy(tc.ip)
			if result != tc.trusted {
				t.Errorf("isIPFromTrustedProxy(%s) = %v, expected %v", tc.ip, result, tc.trusted)
			}
		})
	}
}

func TestValidateIPFormat(t *testing.T) {
	testCases := []struct {
		ip    string
		valid bool
	}{
		{"1.2.3.4", true},
		{"192.168.1.1", true},
		{"::1", true},
		{"2001:db8::1", true},
		{"not-an-ip", false},
		{"1.2.3.4.5", false},
		{"", false},
		{"1.2.3", false},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			result := validateIPFormat(tc.ip)
			if result != tc.valid {
				t.Errorf("validateIPFormat(%s) = %v, expected %v", tc.ip, result, tc.valid)
			}
		})
	}
}
