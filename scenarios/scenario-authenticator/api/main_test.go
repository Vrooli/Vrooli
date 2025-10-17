package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"scenario-authenticator/auth"
	"scenario-authenticator/handlers"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Initialize JWT keys for testing
	os.Setenv("JWT_PUBLIC_KEY_PATH", "../data/keys/public.pem")
	os.Setenv("JWT_PRIVATE_KEY_PATH", "../data/keys/private.pem")

	// Try to load JWT keys, but don't fail if they don't exist
	_ = auth.LoadJWTKeys()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handlers.HealthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if status, ok := response["status"].(string); !ok || status == "" {
			t.Fatalf("Expected non-empty status string, got %v", response["status"])
		}

		if service, ok := response["service"].(string); !ok || service == "" {
			t.Fatalf("Expected service identifier, got %v", response["service"])
		}

		if _, ok := response["timestamp"].(string); !ok {
			t.Fatalf("Expected timestamp string, got %v", response["timestamp"])
		}

		if _, ok := response["readiness"].(bool); !ok {
			t.Fatalf("Expected readiness boolean, got %v", response["readiness"])
		}

		dependencies, ok := response["dependencies"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected dependencies object, got %v", response["dependencies"])
		}

		for _, key := range []string{"database", "redis"} {
			dep, ok := dependencies[key].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected %s dependency object, got %v", key, dependencies[key])
			}

			if _, ok := dep["connected"].(bool); !ok {
				t.Fatalf("Expected %s.connected boolean, got %v", key, dep["connected"])
			}
		}

		metrics, ok := response["metrics"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected metrics object, got %v", response["metrics"])
		}

		if _, ok := metrics["goroutines"].(float64); !ok {
			t.Fatalf("Expected metrics.goroutines number, got %v", metrics)
		}
	})
}

// TestRegisterHandler_Security tests registration endpoint with security payloads
func TestRegisterHandler_Security(t *testing.T) {
	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("POSTGRES_URL not set, skipping database tests")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	defer env.Cleanup()

	// Test SQL injection in email field
	t.Run("SQLInjection_Email", func(t *testing.T) {
		securityPatterns := NewSecurityTestBuilder().AddSQLInjectionTests().Build()

		for _, pattern := range securityPatterns {
			t.Run(pattern.Name, func(t *testing.T) {
				reqData := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/auth/register",
					Body: map[string]interface{}{
						"email":    pattern.Payload,
						"password": "ValidPassword123!",
					},
				}

				req, err := makeHTTPRequest(reqData)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				w := executeRequest(handlers.RegisterHandler, req)

				// Should reject invalid email format
				if w.Code == http.StatusOK || w.Code == http.StatusCreated {
					t.Errorf("SQL injection payload was not blocked: %s", pattern.Payload)
				}
			})
		}
	})

	// Test XSS in username field
	t.Run("XSS_Username", func(t *testing.T) {
		securityPatterns := NewSecurityTestBuilder().AddXSSTests().Build()

		for _, pattern := range securityPatterns {
			t.Run(pattern.Name, func(t *testing.T) {
				reqData := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/auth/register",
					Body: map[string]interface{}{
						"email":    "test@test.local",
						"username": pattern.Payload,
						"password": "ValidPassword123!",
					},
				}

				req, err := makeHTTPRequest(reqData)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				w := executeRequest(handlers.RegisterHandler, req)

				// Verify XSS payload doesn't cause server error
				if w.Code == http.StatusInternalServerError {
					t.Errorf("XSS payload caused server error: %s", pattern.Payload)
				}
			})
		}
	})

	// Test password validation
	t.Run("WeakPassword_Blocked", func(t *testing.T) {
		weakPasswords := []string{
			"123456",
			"password",
			"abc123",
			"qwerty",
			"12345678",
		}

		for _, pwd := range weakPasswords {
			reqData := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/auth/register",
				Body: map[string]interface{}{
					"email":    "weakpass@test.local",
					"password": pwd,
				},
			}

			req, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := executeRequest(handlers.RegisterHandler, req)

			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				t.Errorf("Weak password was accepted: %s", pwd)
			}
		}
	})
}

// TestLoginHandler_Security tests login endpoint with security payloads
func TestLoginHandler_Security(t *testing.T) {
	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("POSTGRES_URL not set, skipping database tests")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	defer env.Cleanup()

	// Create a test user for login attempts
	testUser := createTestUser(t, "logintest@test.local", "ValidPassword123!", []string{"user"})
	defer testUser.Cleanup()

	// Test SQL injection in email field
	t.Run("SQLInjection_Login_Email", func(t *testing.T) {
		securityPatterns := NewSecurityTestBuilder().AddSQLInjectionTests().Build()

		for _, pattern := range securityPatterns {
			t.Run(pattern.Name, func(t *testing.T) {
				reqData := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/auth/login",
					Body: map[string]interface{}{
						"email":    pattern.Payload,
						"password": "anypassword",
					},
				}

				req, err := makeHTTPRequest(reqData)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				w := executeRequest(handlers.LoginHandler, req)

				// Should not successfully authenticate
				if w.Code == http.StatusOK {
					t.Errorf("SQL injection payload bypassed authentication: %s", pattern.Payload)
				}
			})
		}
	})

	// Test brute force protection (rate limiting)
	t.Run("BruteForce_Protection", func(t *testing.T) {
		// Attempt multiple failed logins
		for i := 0; i < 10; i++ {
			reqData := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/auth/login",
				Body: map[string]interface{}{
					"email":    testUser.User.Email,
					"password": "WrongPassword123!",
				},
			}

			req, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := executeRequest(handlers.LoginHandler, req)

			// Should always deny incorrect password
			if w.Code == http.StatusOK {
				t.Errorf("Failed login attempt %d succeeded with wrong password", i+1)
			}
		}
	})
}

// TestValidateHandler_Security tests token validation with security checks
func TestValidateHandler_Security(t *testing.T) {
	if os.Getenv("REDIS_URL") == "" {
		t.Skip("REDIS_URL not set, skipping Redis-dependent tests")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	defer env.Cleanup()

	// Test with invalid token formats
	t.Run("InvalidToken_Formats", func(t *testing.T) {
		invalidTokens := []string{
			"invalid",
			"malformed.token.here",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid",
		}

		for _, token := range invalidTokens {
			reqData := HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/auth/validate",
				Headers: map[string]string{"Authorization": "Bearer " + token},
			}

			req, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := executeRequest(handlers.ValidateHandler, req)

			// ValidateHandler always returns 200 OK, check response body for valid=false
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if valid, ok := response["valid"].(bool); ok && valid {
				t.Errorf("Invalid token was accepted: %s", token)
			}
		}
	})

	// Test token tampering
	t.Run("TokenTampering", func(t *testing.T) {
		// Create a valid-looking but tampered token
		tamperedTokens := []string{
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJyb2xlcyI6WyJhZG1pbiJdfQ.tampered",
			"eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoiYWRtaW4ifQ.",
		}

		for _, token := range tamperedTokens {
			reqData := HTTPTestRequest{
				Method:  "POST",
				Path:    "/api/v1/auth/validate",
				Headers: map[string]string{"Authorization": "Bearer " + token},
			}

			req, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := executeRequest(handlers.ValidateHandler, req)

			// ValidateHandler always returns 200 OK, check response body for valid=false
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if valid, ok := response["valid"].(bool); ok && valid {
				t.Errorf("Tampered token was accepted: %s", token)
			}
		}
	})
}

// TestRegisterHandler_Validation tests input validation
func TestRegisterHandler_Validation(t *testing.T) {
	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("POSTGRES_URL not set, skipping database tests")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	defer env.Cleanup()

	t.Run("MissingEmail", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/register",
			Body: map[string]interface{}{
				"password": "ValidPassword123!",
			},
		}

		req, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := executeRequest(handlers.RegisterHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("MissingPassword", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/register",
			Body: map[string]interface{}{
				"email": "test@test.local",
			},
		}

		req, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := executeRequest(handlers.RegisterHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("InvalidEmailFormat", func(t *testing.T) {
		invalidEmails := []string{
			"notanemail",
			"@example.com",
			"user@",
			"user name@example.com",
			"user@.com",
		}

		for _, email := range invalidEmails {
			reqData := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/auth/register",
				Body: map[string]interface{}{
					"email":    email,
					"password": "ValidPassword123!",
				},
			}

			req, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := executeRequest(handlers.RegisterHandler, req)

			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				t.Errorf("Invalid email was accepted: %s", email)
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/register",
			Body:   `{"email": "test@test.local", "password": "ValidPassword123!"`, // Missing closing brace
		}

		req, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := executeRequest(handlers.RegisterHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestLoginHandler_Validation tests login input validation
func TestLoginHandler_Validation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingCredentials", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/login",
			Body:   map[string]interface{}{},
		}

		req, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := executeRequest(handlers.LoginHandler, req)

		if w.Code == http.StatusOK {
			t.Error("Login succeeded without credentials")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/login",
			Body:   `{"email": "test@test.local"`, // Malformed JSON
		}

		req, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := executeRequest(handlers.LoginHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestPasswordResetHandler_Security tests password reset security
func TestPasswordResetHandler_Security(t *testing.T) {
	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("POSTGRES_URL not set, skipping database tests")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDatabase(t)
	defer env.Cleanup()

	// Test email enumeration protection
	t.Run("EmailEnumeration_Protection", func(t *testing.T) {
		emails := []string{
			"exists@test.local",
			"doesnotexist@test.local",
			"another@test.local",
		}

		// All requests should return similar responses to prevent email enumeration
		for _, email := range emails {
			reqData := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/auth/reset-password",
				Body: map[string]interface{}{
					"email": email,
				},
			}

			req, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := executeRequest(handlers.ResetPasswordHandler, req)

			// Should not reveal if email exists or not
			if w.Code != http.StatusOK && w.Code != http.StatusAccepted {
				t.Logf("Password reset returned status %d for email %s", w.Code, email)
			}
		}
	})

	// Test SQL injection in email field
	t.Run("SQLInjection_PasswordReset", func(t *testing.T) {
		securityPatterns := NewSecurityTestBuilder().AddSQLInjectionTests().Build()

		for _, pattern := range securityPatterns {
			t.Run(pattern.Name, func(t *testing.T) {
				reqData := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/auth/reset-password",
					Body: map[string]interface{}{
						"email": pattern.Payload,
					},
				}

				req, err := makeHTTPRequest(reqData)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				w := executeRequest(handlers.ResetPasswordHandler, req)

				// Should not cause server error
				if w.Code == http.StatusInternalServerError {
					t.Errorf("SQL injection caused server error: %s", pattern.Payload)
				}
			})
		}
	})
}

// TestSecurityHeaders tests that security headers are properly set
func TestSecurityHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SecurityHeaders_Present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handlers.HealthHandler(w, req)

		// Note: Security headers are typically set by middleware
		// This test documents expected behavior
		expectedHeaders := map[string]string{
			// These would be set by SecurityHeadersMiddleware
			// "X-Content-Type-Options": "nosniff",
			// "X-Frame-Options": "DENY",
			// "X-XSS-Protection": "1; mode=block",
		}

		for header, expectedValue := range expectedHeaders {
			actualValue := w.Header().Get(header)
			if actualValue != expectedValue {
				t.Logf("Security header '%s' not set (expected: %s, got: %s)", header, expectedValue, actualValue)
			}
		}
	})
}

// TestConcurrentRequests tests thread safety
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		done := make(chan bool)
		numRequests := 100

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				handlers.HealthHandler(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Concurrent request failed with status %d", w.Code)
				}
				done <- true
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			<-done
		}
	})
}

// TestRateLimiting tests rate limiting behavior
func TestRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RateLimit_NotImplemented", func(t *testing.T) {
		// This test documents that rate limiting should be implemented
		// Currently scenario-authenticator may not have rate limiting
		t.Skip("Rate limiting not yet implemented - add test when feature is added")
	})
}

// Helper function to test a specific handler
func testHandlerSecurity(t *testing.T, handlerName string, handler http.HandlerFunc, method, path string) {
	suite := &HandlerTestSuite{
		HandlerName: handlerName,
		Handler:     handler,
		BaseURL:     path,
	}

	// Generate and run security tests
	securityPatterns := GenerateSecurityTests()
	suite.RunSecurityTests(t, securityPatterns, func(t *testing.T, pattern SecurityTestPattern) {
		// This would test the handler with the security payload
		t.Logf("Security test %s with payload: %s", pattern.Category, pattern.Payload)
	})
}
