package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireUserAuth_ValidBearerToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
