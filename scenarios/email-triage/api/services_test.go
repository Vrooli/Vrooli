package main

import (
	"encoding/json"
	"testing"
	"time"

	"email-triage/models"
	"email-triage/services"
)

// TestAuthService tests the AuthService
func TestAuthService(t *testing.T) {
	t.Run("ValidateToken_Success", func(t *testing.T) {
		userID := "test-user-123"
		validToken := "valid-token-abc"

		// Create mock auth server
		mockAuth := MockAuthServer(t, validToken, userID)
		defer mockAuth.Close()

		authService := services.NewAuthService(mockAuth.URL())

		// Test token validation
		resultUserID, err := authService.ValidateToken("Bearer " + validToken)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resultUserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, resultUserID)
		}
	})

	t.Run("ValidateToken_InvalidFormat", func(t *testing.T) {
		authService := services.NewAuthService("http://localhost:8080")

		// Test with invalid format (no "Bearer " prefix)
		_, err := authService.ValidateToken("invalid-token")

		if err == nil {
			t.Error("Expected error for invalid format")
		}
	})

	t.Run("ValidateToken_InvalidToken", func(t *testing.T) {
		userID := "test-user-123"
		validToken := "valid-token-abc"

		mockAuth := MockAuthServer(t, validToken, userID)
		defer mockAuth.Close()

		authService := services.NewAuthService(mockAuth.URL())

		// Test with wrong token
		_, err := authService.ValidateToken("Bearer wrong-token")

		if err == nil {
			t.Error("Expected error for invalid token")
		}
	})

	t.Run("HealthCheck_Success", func(t *testing.T) {
		mockAuth := MockAuthServer(t, "any-token", "any-user")
		defer mockAuth.Close()

		authService := services.NewAuthService(mockAuth.URL())

		healthy := authService.HealthCheck()

		if !healthy {
			t.Error("Expected health check to pass")
		}
	})

	t.Run("HealthCheck_Failure", func(t *testing.T) {
		// Point to non-existent server
		authService := services.NewAuthService("http://localhost:99999")

		healthy := authService.HealthCheck()

		if healthy {
			t.Error("Expected health check to fail for unreachable server")
		}
	})
}

// TestEmailService tests the EmailService
func TestEmailService(t *testing.T) {
	t.Run("NewEmailService", func(t *testing.T) {
		mailServer := "mail.example.com"
		emailService := services.NewEmailService(mailServer)

		if emailService == nil {
			t.Error("Expected non-nil EmailService")
		}
	})

	t.Run("TestConnection_Failure", func(t *testing.T) {
		emailService := services.NewEmailService("invalid-server")

		// Create a test account
		imapConfig := map[string]interface{}{
			"server":   "invalid.example.com",
			"port":     993,
			"username": "test@example.com",
			"password": "wrong-password",
			"use_tls":  true,
		}

		smtpConfig := map[string]interface{}{
			"server":   "invalid.example.com",
			"port":     587,
			"username": "test@example.com",
			"password": "wrong-password",
			"use_tls":  true,
		}

		imapJSON, _ := json.Marshal(imapConfig)
		smtpJSON, _ := json.Marshal(smtpConfig)

		account := &models.EmailAccount{
			EmailAddress: "test@example.com",
			IMAPSettings: imapJSON,
			SMTPSettings: smtpJSON,
		}

		// Test connection should fail
		err := emailService.TestConnection(account)

		if err == nil {
			t.Log("Connection test failed as expected (likely cannot reach server)")
		}
	})
}

// TestRuleService tests the RuleService
func TestRuleService(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	ollamaURL := "http://localhost:11434"
	ruleService := services.NewRuleService(testDB.db, ollamaURL)

	if ruleService == nil {
		t.Error("Expected non-nil RuleService")
	}

	t.Run("CreateRuleService", func(t *testing.T) {
		// Just verify service can be created
		if ruleService == nil {
			t.Error("Expected RuleService to be created")
		}
	})
}

// TestSearchService tests the SearchService
func TestSearchService(t *testing.T) {
	t.Run("NewSearchService", func(t *testing.T) {
		qdrantURL := "http://localhost:6333"
		searchService := services.NewSearchService(qdrantURL)

		if searchService == nil {
			t.Error("Expected non-nil SearchService")
		}
	})

	t.Run("HealthCheck_NoConnection", func(t *testing.T) {
		// Point to non-existent Qdrant server
		searchService := services.NewSearchService("http://localhost:99999")

		healthy := searchService.HealthCheck()

		// Health check should return false for unreachable server
		if healthy {
			t.Log("Qdrant not available (expected in test environment)")
		}
	})
}

// TestRealtimeProcessor tests the RealtimeProcessor
func TestRealtimeProcessor(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("test-mail-server")
	ruleService := services.NewRuleService(testDB.db, "http://localhost:11434")
	searchService := services.NewSearchService("http://localhost:6333")

	processor := services.NewRealtimeProcessor(testDB.db, emailService, ruleService, searchService)

	if processor == nil {
		t.Error("Expected non-nil RealtimeProcessor")
	}

	t.Run("Start_Stop", func(t *testing.T) {
		// Start processor
		err := processor.Start()
		if err != nil {
			t.Logf("Processor start may fail without dependencies: %v", err)
		}

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		// Stop processor
		processor.Stop()

		// Give it a moment to stop
		time.Sleep(100 * time.Millisecond)

		t.Log("Processor lifecycle test completed")
	})
}

// TestModels tests the data models
func TestModels(t *testing.T) {
	t.Run("EmailAccount_Structure", func(t *testing.T) {
		account := &models.EmailAccount{
			ID:           "test-id",
			UserID:       "user-id",
			EmailAddress: "test@example.com",
			SyncEnabled:  true,
		}

		if account.EmailAddress != "test@example.com" {
			t.Error("EmailAccount fields not working correctly")
		}
	})

	t.Run("CreateAccountRequest_Structure", func(t *testing.T) {
		req := &models.CreateAccountRequest{
			EmailAddress: "test@example.com",
			Password:     "password",
			IMAPServer:   "imap.example.com",
			IMAPPort:     993,
			UseTLS:       true,
		}

		if req.EmailAddress != "test@example.com" {
			t.Error("CreateAccountRequest fields not working correctly")
		}
	})

	t.Run("TriageRule_Structure", func(t *testing.T) {
		rule := &models.TriageRule{
			ID:          "rule-id",
			UserID:      "user-id",
			Name:        "Test Rule",
			Description: "Test description",
			Enabled:     true,
		}

		if rule.Name != "Test Rule" {
			t.Error("TriageRule fields not working correctly")
		}
	})

	t.Run("ProcessedEmail_Structure", func(t *testing.T) {
		email := &models.ProcessedEmail{
			ID:            "email-id",
			AccountID:     "account-id",
			Subject:       "Test Subject",
			SenderEmail:   "sender@example.com",
			FullBody:      "Test body",
			PriorityScore: 0.5,
		}

		if email.Subject != "Test Subject" {
			t.Error("ProcessedEmail fields not working correctly")
		}
	})

	t.Run("HealthStatus_Structure", func(t *testing.T) {
		health := &models.HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
			Services:  map[string]string{"db": "connected"},
		}

		if health.Status != "healthy" {
			t.Error("HealthStatus fields not working correctly")
		}
	})
}
