package main

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// cleanupUserTestData removes test user data created during tests.
func cleanupUserTestData(t *testing.T, db *sql.DB, email string) {
	t.Helper()

	// Delete user and cascade to related records
	_, _ = db.Exec("DELETE FROM users WHERE email = $1", email)
}

func TestGetOrCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-create-user@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Test creating a new user
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	if user == nil {
		t.Fatal("GetOrCreateUser returned nil user")
	}

	if user.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, user.Email)
	}

	if user.EmailVerified {
		t.Error("Expected email_verified to be false for new user")
	}

	// Test getting existing user (idempotent)
	user2, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser (second call) failed: %v", err)
	}

	if user2.ID != user.ID {
		t.Errorf("Expected same user ID %s, got %s", user.ID, user2.ID)
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-get-by-email@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Test getting non-existent user
	user, err := authService.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetUserByEmail failed for non-existent user: %v", err)
	}
	if user != nil {
		t.Error("Expected nil user for non-existent email")
	}

	// Create user
	_, err = authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Test getting existing user
	user, err = authService.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetUserByEmail failed for existing user: %v", err)
	}
	if user == nil {
		t.Fatal("Expected user, got nil")
	}
	if user.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, user.Email)
	}
}

func TestJWTCreationAndValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-jwt@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Generate access token
	token, expiresAt, err := authService.generateAccessToken(user.ID, user.Email, "test-session-id")
	if err != nil {
		t.Fatalf("generateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("Expected expires_at to be in the future")
	}

	// Validate token
	claims, err := authService.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, claims.UserID)
	}

	if claims.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, claims.Email)
	}

	if claims.SessionID != "test-session-id" {
		t.Errorf("Expected session ID %s, got %s", "test-session-id", claims.SessionID)
	}
}

func TestJWTExpiry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()

	// Create service with very short TTL for testing
	authService := &UserAuthService{
		db:           db,
		emailService: emailService,
		jwtSecret:    []byte("test-secret-key"),
		jwtIssuer:    "test",
		accessTTL:    1 * time.Millisecond, // Very short for testing
		refreshTTL:   7 * 24 * time.Hour,
		magicLinkTTL: 15 * time.Minute,
	}

	testEmail := "test-jwt-expiry@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Generate token
	token, _, err := authService.generateAccessToken(user.ID, user.Email, "test-session")
	if err != nil {
		t.Fatalf("generateAccessToken failed: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Token should be expired
	_, err = authService.ValidateAccessToken(token)
	if err == nil {
		t.Error("Expected error for expired token")
	}
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestInvalidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	// Test various invalid tokens
	invalidTokens := []string{
		"",
		"invalid",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.invalid",
		"not.a.jwt",
	}

	for _, token := range invalidTokens {
		_, err := authService.ValidateAccessToken(token)
		if err == nil {
			t.Errorf("Expected error for invalid token %q", token)
		}
	}
}

func TestLinkStripeCustomer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-stripe-link@example.com"
	testCustomerID := "cus_test123"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Link stripe customer (should create user)
	err := authService.LinkStripeCustomer(ctx, testEmail, testCustomerID)
	if err != nil {
		t.Fatalf("LinkStripeCustomer failed: %v", err)
	}

	// Verify user was created with stripe customer ID
	user, err := authService.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if user == nil {
		t.Fatal("Expected user to be created")
	}

	if user.StripeCustomerID == nil || *user.StripeCustomerID != testCustomerID {
		t.Errorf("Expected stripe customer ID %s, got %v", testCustomerID, user.StripeCustomerID)
	}

	// Test updating existing user's stripe customer ID
	newCustomerID := "cus_test456"
	err = authService.LinkStripeCustomer(ctx, testEmail, newCustomerID)
	if err != nil {
		t.Fatalf("LinkStripeCustomer (update) failed: %v", err)
	}

	user, err = authService.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if user.StripeCustomerID == nil || *user.StripeCustomerID != newCustomerID {
		t.Errorf("Expected updated stripe customer ID %s, got %v", newCustomerID, user.StripeCustomerID)
	}
}

func TestHashToken(t *testing.T) {
	// Test that hashing is deterministic
	token := "test-token-12345"

	hash1 := hashToken(token)
	hash2 := hashToken(token)

	if hash1 != hash2 {
		t.Errorf("hashToken not deterministic: %s != %s", hash1, hash2)
	}

	// Test that different tokens produce different hashes
	hash3 := hashToken("different-token")
	if hash1 == hash3 {
		t.Error("Different tokens should produce different hashes")
	}

	// Test that hash is hex-encoded SHA-256 (64 chars)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}

func TestVerifyMagicLink_ValidToken(t *testing.T) {
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

	testEmail := "test-verify-magic-link@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Capture the generated token using the callback
	var capturedToken string
	authService.UseTokenCallback(func(email, token, magicLink string) {
		capturedToken = token
	})

	// Request magic link
	err := authService.RequestMagicLink(ctx, testEmail, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("RequestMagicLink failed: %v", err)
	}

	if capturedToken == "" {
		t.Fatal("Token callback was not called")
	}

	// Verify the magic link
	tokenPair, user, err := authService.VerifyMagicLink(ctx, capturedToken, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("VerifyMagicLink failed: %v", err)
	}

	if tokenPair == nil {
		t.Fatal("Expected token pair, got nil")
	}

	if tokenPair.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}

	if tokenPair.TokenType != "Bearer" {
		t.Errorf("Expected token type 'Bearer', got %s", tokenPair.TokenType)
	}

	if user == nil {
		t.Fatal("Expected user, got nil")
	}

	if user.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, user.Email)
	}

	if !user.EmailVerified {
		t.Error("Expected email to be verified after magic link verification")
	}
}

func TestVerifyMagicLink_ExpiredToken(t *testing.T) {
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

	testEmail := "test-verify-expired@example.com"
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

	// Verify should fail with expired error
	_, _, err = authService.VerifyMagicLink(ctx, capturedToken, "127.0.0.1", "Test-Agent")
	if err == nil {
		t.Fatal("Expected error for expired token")
	}
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestVerifyMagicLink_UsedToken(t *testing.T) {
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

	testEmail := "test-verify-used@example.com"
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

	// First verification should succeed
	_, _, err = authService.VerifyMagicLink(ctx, capturedToken, "127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("First VerifyMagicLink failed: %v", err)
	}

	// Second verification should fail with used error
	_, _, err = authService.VerifyMagicLink(ctx, capturedToken, "127.0.0.1", "Test-Agent")
	if err == nil {
		t.Fatal("Expected error for used token")
	}
	if err != ErrTokenUsed {
		t.Errorf("Expected ErrTokenUsed, got %v", err)
	}
}

func TestVerifyMagicLink_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	ctx := context.Background()

	// Test various invalid tokens
	invalidTokens := []string{
		"",
		"   ",
		"invalid-token-that-doesnt-exist",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	for _, token := range invalidTokens {
		_, _, err := authService.VerifyMagicLink(ctx, token, "127.0.0.1", "Test-Agent")
		if err == nil {
			t.Errorf("Expected error for invalid token %q", token)
		}
		if err != ErrTokenInvalid {
			t.Errorf("Expected ErrTokenInvalid for token %q, got %v", token, err)
		}
	}
}

func TestLogoutAllSessions_RevokesOtherSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-logout-all@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Create 3 sessions
	session1, err := authService.createSession(ctx, user, "127.0.0.1", "Agent-1")
	if err != nil {
		t.Fatalf("createSession 1 failed: %v", err)
	}

	session2, err := authService.createSession(ctx, user, "127.0.0.2", "Agent-2")
	if err != nil {
		t.Fatalf("createSession 2 failed: %v", err)
	}

	session3, err := authService.createSession(ctx, user, "127.0.0.3", "Agent-3")
	if err != nil {
		t.Fatalf("createSession 3 failed: %v", err)
	}

	// Get session ID for the "current" session (session1)
	claims1, err := authService.ValidateAccessToken(session1.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	// Logout all sessions except the current one
	err = authService.LogoutAllSessions(ctx, user.ID, claims1.SessionID)
	if err != nil {
		t.Fatalf("LogoutAllSessions failed: %v", err)
	}

	// Session 1 (current) should still work
	_, err = authService.RefreshTokens(ctx, session1.RefreshToken)
	if err != nil {
		t.Errorf("Session 1 should still be valid, got error: %v", err)
	}

	// Session 2 should be revoked
	_, err = authService.RefreshTokens(ctx, session2.RefreshToken)
	if err == nil {
		t.Error("Session 2 should be revoked")
	}
	if err != ErrSessionRevoked {
		t.Errorf("Expected ErrSessionRevoked for session 2, got %v", err)
	}

	// Session 3 should be revoked
	_, err = authService.RefreshTokens(ctx, session3.RefreshToken)
	if err == nil {
		t.Error("Session 3 should be revoked")
	}
	if err != ErrSessionRevoked {
		t.Errorf("Expected ErrSessionRevoked for session 3, got %v", err)
	}
}

func TestLogoutAllSessions_NoOtherSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	emailService := NewEmailService()
	authService := NewUserAuthService(db, emailService)

	testEmail := "test-logout-all-single@example.com"
	defer cleanupUserTestData(t, db, testEmail)

	ctx := context.Background()

	// Create user
	user, err := authService.GetOrCreateUser(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Create only 1 session
	session1, err := authService.createSession(ctx, user, "127.0.0.1", "Agent-1")
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	// Get session ID
	claims1, err := authService.ValidateAccessToken(session1.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	// Logout all sessions except the current one (should be a no-op)
	err = authService.LogoutAllSessions(ctx, user.ID, claims1.SessionID)
	if err != nil {
		t.Fatalf("LogoutAllSessions failed: %v", err)
	}

	// The single session should still work
	_, err = authService.RefreshTokens(ctx, session1.RefreshToken)
	if err != nil {
		t.Errorf("Single session should still be valid, got error: %v", err)
	}
}
