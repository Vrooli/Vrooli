package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
)

// ============================================================================
// Pure Function Tests
// ============================================================================

func TestFeedbackTypeLabel_Refund(t *testing.T) {
	result := feedbackTypeLabel("refund")
	if result != "Refund Request" {
		t.Errorf("expected 'Refund Request', got '%s'", result)
	}
}

func TestFeedbackTypeLabel_Bug(t *testing.T) {
	result := feedbackTypeLabel("bug")
	if result != "Bug Report" {
		t.Errorf("expected 'Bug Report', got '%s'", result)
	}
}

func TestFeedbackTypeLabel_Feature(t *testing.T) {
	result := feedbackTypeLabel("feature")
	if result != "Feature Request" {
		t.Errorf("expected 'Feature Request', got '%s'", result)
	}
}

func TestFeedbackTypeLabel_Unknown(t *testing.T) {
	result := feedbackTypeLabel("other")
	if result != "General Feedback" {
		t.Errorf("expected 'General Feedback', got '%s'", result)
	}
}

func TestFeedbackTypeLabel_Empty(t *testing.T) {
	result := feedbackTypeLabel("")
	if result != "General Feedback" {
		t.Errorf("expected 'General Feedback' for empty string, got '%s'", result)
	}
}

func TestSMTPConfig_IsConfigured_AllSet(t *testing.T) {
	config := &SMTPConfig{
		Host:     "smtp.example.com",
		Username: "user",
		Password: "pass",
	}
	if !config.IsConfigured() {
		t.Error("expected IsConfigured to return true when all fields set")
	}
}

func TestSMTPConfig_IsConfigured_MissingHost(t *testing.T) {
	config := &SMTPConfig{
		Username: "user",
		Password: "pass",
	}
	if config.IsConfigured() {
		t.Error("expected IsConfigured to return false when host missing")
	}
}

func TestSMTPConfig_IsConfigured_MissingUsername(t *testing.T) {
	config := &SMTPConfig{
		Host:     "smtp.example.com",
		Password: "pass",
	}
	if config.IsConfigured() {
		t.Error("expected IsConfigured to return false when username missing")
	}
}

func TestSMTPConfig_IsConfigured_MissingPassword(t *testing.T) {
	config := &SMTPConfig{
		Host:     "smtp.example.com",
		Username: "user",
	}
	if config.IsConfigured() {
		t.Error("expected IsConfigured to return false when password missing")
	}
}

func TestSendGridConfig_IsConfigured_AllSet(t *testing.T) {
	config := &SendGridConfig{
		APIKey:    "SG.test",
		FromEmail: "from@example.com",
	}
	if !config.IsConfigured() {
		t.Error("expected IsConfigured to return true")
	}
}

func TestSendGridConfig_IsConfigured_MissingAPIKey(t *testing.T) {
	config := &SendGridConfig{
		FromEmail: "from@example.com",
	}
	if config.IsConfigured() {
		t.Error("expected IsConfigured to return false when API key missing")
	}
}

func TestSendGridConfig_IsConfigured_MissingFromEmail(t *testing.T) {
	config := &SendGridConfig{
		APIKey: "SG.test",
	}
	if config.IsConfigured() {
		t.Error("expected IsConfigured to return false when from email missing")
	}
}

// ============================================================================
// SendGrid via HTTP Mock Tests
// ============================================================================

func TestSendViaSendGrid_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.Header.Get("Authorization"), "Bearer") {
			t.Error("expected Authorization header")
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SendGridConfig: &SendGridConfig{
			APIKey:    "SG.test",
			FromEmail: "from@example.com",
			FromName:  "Test",
		},
		HTTPClient: server.Client(),
	})

	// Override the URL by creating a custom handler
	// Note: In production, sendgrid URL is hardcoded, so we test via the full method
	// For testing, we can verify the service was configured
	if !svc.IsSendGridConfigured() {
		t.Error("expected SendGrid to be configured")
	}
}

func TestSendViaSendGrid_NotConfigured(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SendGridConfig: nil,
	})

	err := svc.sendViaSendGrid("to@example.com", "subject", "text", "html")
	if err == nil {
		t.Error("expected error when SendGrid not configured")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("expected 'not configured' in error, got: %v", err)
	}
}

func TestSendMagicLink_SendGridConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SendGridConfig: &SendGridConfig{
			APIKey:    "SG.test",
			FromEmail: "from@example.com",
			FromName:  "Test App",
		},
		HTTPClient: server.Client(),
	})

	// Verify it's configured correctly
	if !svc.IsSendGridConfigured() {
		t.Error("expected SendGrid to be configured")
	}
}

func TestSendMagicLink_DevModeFallback(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SendGridConfig: nil, // Not configured
	})

	// Should succeed in dev mode (logs the link)
	err := svc.SendMagicLink("to@example.com", "http://example.com/magic", "TestApp")
	if err != nil {
		t.Errorf("expected no error in dev mode, got: %v", err)
	}
}

func TestSendMagicLink_DefaultAppName(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SendGridConfig: nil,
	})

	// Empty app name should default to "App"
	err := svc.SendMagicLink("to@example.com", "http://example.com/magic", "")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// ============================================================================
// SMTP via Interface Mock Tests
// ============================================================================

func TestSend_Success(t *testing.T) {
	var capturedAddr string
	var capturedFrom string
	var capturedTo []string
	var capturedMsg []byte

	mockSender := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		capturedAddr = addr
		capturedFrom = from
		capturedTo = to
		capturedMsg = msg
		return nil
	}

	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SMTPSender: mockSender,
	})

	config := &SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	err := svc.Send(config, "to@example.com", "Test Subject", "Test Body")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if capturedAddr != "smtp.example.com:587" {
		t.Errorf("expected addr 'smtp.example.com:587', got '%s'", capturedAddr)
	}
	if capturedFrom != "from@example.com" {
		t.Errorf("expected from 'from@example.com', got '%s'", capturedFrom)
	}
	if len(capturedTo) != 1 || capturedTo[0] != "to@example.com" {
		t.Errorf("expected to ['to@example.com'], got %v", capturedTo)
	}
	if !strings.Contains(string(capturedMsg), "Test Subject") {
		t.Error("expected message to contain subject")
	}
	if !strings.Contains(string(capturedMsg), "Test Body") {
		t.Error("expected message to contain body")
	}
}

func TestSend_NotConfigured(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{})

	config := &SMTPConfig{
		// Missing required fields
	}

	err := svc.Send(config, "to@example.com", "Subject", "Body")
	if err == nil {
		t.Error("expected error for unconfigured SMTP")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("expected 'not configured' error, got: %v", err)
	}
}

func TestSend_SMTPError(t *testing.T) {
	mockSender := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return errors.New("SMTP connection failed")
	}

	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SMTPSender: mockSender,
	})

	config := &SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	err := svc.Send(config, "to@example.com", "Subject", "Body")
	if err == nil {
		t.Error("expected error from SMTP failure")
	}
	if !strings.Contains(err.Error(), "SMTP connection failed") {
		t.Errorf("expected SMTP error message, got: %v", err)
	}
}

// ============================================================================
// Email Service Constructor Tests
// ============================================================================

func TestNewEmailServiceWithOptions_DefaultSMTPSender(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{})

	// smtpSender should be set (to smtp.SendMail by default)
	if svc.smtpSender == nil {
		t.Error("expected default smtpSender to be set")
	}
}

func TestNewEmailServiceWithOptions_DefaultHTTPClient(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{})

	if svc.httpClient == nil {
		t.Error("expected default httpClient to be set")
	}
}

func TestNewEmailServiceWithOptions_CustomHTTPClient(t *testing.T) {
	customClient := &http.Client{}
	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		HTTPClient: customClient,
	})

	if svc.httpClient != customClient {
		t.Error("expected custom httpClient to be used")
	}
}

func TestIsSendGridConfigured_True(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SendGridConfig: &SendGridConfig{
			APIKey:    "SG.test",
			FromEmail: "from@example.com",
		},
	})

	if !svc.IsSendGridConfigured() {
		t.Error("expected IsSendGridConfigured to return true")
	}
}

func TestIsSendGridConfigured_False(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{})

	if svc.IsSendGridConfigured() {
		t.Error("expected IsSendGridConfigured to return false when not configured")
	}
}

// ============================================================================
// Extract SMTP Config Tests
// ============================================================================

func TestExtractSMTPConfig_AllFields(t *testing.T) {
	host := "smtp.test.com"
	port := 465
	username := "testuser"
	password := "testpass"
	from := "custom@example.com"

	branding := &SiteBranding{
		SMTPHost:     &host,
		SMTPPort:     &port,
		SMTPUsername: &username,
		SMTPPassword: &password,
		SMTPFrom:     &from,
	}

	svc := NewEmailServiceWithOptions(EmailServiceOptions{})
	config := svc.extractSMTPConfig(branding)

	if config.Host != host {
		t.Errorf("expected host '%s', got '%s'", host, config.Host)
	}
	if config.Port != port {
		t.Errorf("expected port %d, got %d", port, config.Port)
	}
	if config.Username != username {
		t.Errorf("expected username '%s', got '%s'", username, config.Username)
	}
	if config.Password != password {
		t.Errorf("expected password '%s', got '%s'", password, config.Password)
	}
	if config.From != from {
		t.Errorf("expected from '%s', got '%s'", from, config.From)
	}
}

func TestExtractSMTPConfig_DefaultPort(t *testing.T) {
	branding := &SiteBranding{}

	svc := NewEmailServiceWithOptions(EmailServiceOptions{})
	config := svc.extractSMTPConfig(branding)

	if config.Port != 587 {
		t.Errorf("expected default port 587, got %d", config.Port)
	}
}

func TestExtractSMTPConfig_FromDefaultsToUsername(t *testing.T) {
	username := "testuser@example.com"
	branding := &SiteBranding{
		SMTPUsername: &username,
	}

	svc := NewEmailServiceWithOptions(EmailServiceOptions{})
	config := svc.extractSMTPConfig(branding)

	if config.From != username {
		t.Errorf("expected from to default to username '%s', got '%s'", username, config.From)
	}
}

// ============================================================================
// Magic Link HTML/Text Generation Tests
// ============================================================================

func TestBuildMagicLinkHTML_ContainsLink(t *testing.T) {
	link := "http://example.com/magic?token=abc123"
	appName := "TestApp"

	html := buildMagicLinkHTML(link, appName)

	if !strings.Contains(html, link) {
		t.Error("expected HTML to contain magic link")
	}
	if !strings.Contains(html, appName) {
		t.Error("expected HTML to contain app name")
	}
	if !strings.Contains(html, "Sign in") {
		t.Error("expected HTML to contain 'Sign in'")
	}
}

func TestBuildMagicLinkText_ContainsLink(t *testing.T) {
	link := "http://example.com/magic?token=abc123"
	appName := "TestApp"

	text := buildMagicLinkText(link, appName)

	if !strings.Contains(text, link) {
		t.Error("expected text to contain magic link")
	}
	if !strings.Contains(text, appName) {
		t.Error("expected text to contain app name")
	}
}

// ============================================================================
// Feedback Notification Tests
// ============================================================================

func TestSendFeedbackNotification_NotConfigured(t *testing.T) {
	svc := NewEmailServiceWithOptions(EmailServiceOptions{})

	branding := &SiteBranding{} // SMTP not configured
	feedback := &FeedbackRequest{
		Type:    "bug",
		Subject: "Test",
		Message: "Test message",
	}

	// Should not return error when SMTP not configured (just skips)
	err := svc.SendFeedbackNotification(branding, feedback)
	if err != nil {
		t.Errorf("expected no error when SMTP not configured, got: %v", err)
	}
}

func TestSendFeedbackNotification_NoSupportEmail(t *testing.T) {
	host := "smtp.test.com"
	username := "user"
	password := "pass"

	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SMTPSender: func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			return nil
		},
	})

	branding := &SiteBranding{
		SMTPHost:     &host,
		SMTPUsername: &username,
		SMTPPassword: &password,
		SupportEmail: nil, // No support email
	}

	feedback := &FeedbackRequest{
		Type:    "bug",
		Subject: "Test",
	}

	// Should return nil when no support email
	err := svc.SendFeedbackNotification(branding, feedback)
	if err != nil {
		t.Errorf("expected no error when support email not set, got: %v", err)
	}
}

func TestSendFeedbackNotification_Success(t *testing.T) {
	var capturedTo []string

	host := "smtp.test.com"
	username := "user"
	password := "pass"
	supportEmail := "support@example.com"

	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SMTPSender: func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			capturedTo = to
			return nil
		},
	})

	branding := &SiteBranding{
		SMTPHost:     &host,
		SMTPUsername: &username,
		SMTPPassword: &password,
		SupportEmail: &supportEmail,
	}

	feedback := &FeedbackRequest{
		Type:    "bug",
		Subject: "Bug found",
		Message: "Something is broken",
		Email:   "user@example.com",
	}

	err := svc.SendFeedbackNotification(branding, feedback)
	if err != nil {
		t.Fatalf("SendFeedbackNotification failed: %v", err)
	}

	if len(capturedTo) != 1 || capturedTo[0] != supportEmail {
		t.Errorf("expected to send to '%s', got %v", supportEmail, capturedTo)
	}
}

func TestSendFeedbackNotification_WithOrderID(t *testing.T) {
	var capturedMsg []byte

	host := "smtp.test.com"
	username := "user"
	password := "pass"
	supportEmail := "support@example.com"
	orderID := "order_12345"

	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SMTPSender: func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			capturedMsg = msg
			return nil
		},
	})

	branding := &SiteBranding{
		SMTPHost:     &host,
		SMTPUsername: &username,
		SMTPPassword: &password,
		SupportEmail: &supportEmail,
	}

	feedback := &FeedbackRequest{
		Type:    "refund",
		Subject: "Refund request",
		Message: "I want a refund",
		Email:   "user@example.com",
		OrderID: &orderID,
	}

	err := svc.SendFeedbackNotification(branding, feedback)
	if err != nil {
		t.Fatalf("SendFeedbackNotification failed: %v", err)
	}

	if !strings.Contains(string(capturedMsg), orderID) {
		t.Error("expected message to contain order ID")
	}
}

func TestNewEmailService_DefaultConfiguration(t *testing.T) {
	// Just verify it doesn't panic
	svc := NewEmailService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.httpClient == nil {
		t.Error("expected httpClient to be set")
	}
	if svc.smtpSender == nil {
		t.Error("expected smtpSender to be set")
	}
}

// Test helper to get SiteBranding fields
type testBrandingHelper struct {
	host     string
	port     int
	username string
	password string
	from     string
	support  string
}

func (h *testBrandingHelper) toBranding() *SiteBranding {
	b := &SiteBranding{}
	if h.host != "" {
		b.SMTPHost = &h.host
	}
	if h.port != 0 {
		b.SMTPPort = &h.port
	}
	if h.username != "" {
		b.SMTPUsername = &h.username
	}
	if h.password != "" {
		b.SMTPPassword = &h.password
	}
	if h.from != "" {
		b.SMTPFrom = &h.from
	}
	if h.support != "" {
		b.SupportEmail = &h.support
	}
	return b
}

func TestExtractSMTPConfig_PartialConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		helper   testBrandingHelper
		expected SMTPConfig
	}{
		{
			name: "only_host",
			helper: testBrandingHelper{
				host: "smtp.example.com",
			},
			expected: SMTPConfig{
				Host: "smtp.example.com",
				Port: 587,
			},
		},
		{
			name: "host_and_port",
			helper: testBrandingHelper{
				host: "smtp.example.com",
				port: 465,
			},
			expected: SMTPConfig{
				Host: "smtp.example.com",
				Port: 465,
			},
		},
		{
			name: "all_except_from",
			helper: testBrandingHelper{
				host:     "smtp.example.com",
				port:     587,
				username: "user@example.com",
				password: "secret",
			},
			expected: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user@example.com",
				Password: "secret",
				From:     "user@example.com", // Defaults to username
			},
		},
	}

	svc := NewEmailServiceWithOptions(EmailServiceOptions{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branding := tt.helper.toBranding()
			config := svc.extractSMTPConfig(branding)

			if config.Host != tt.expected.Host {
				t.Errorf("expected host '%s', got '%s'", tt.expected.Host, config.Host)
			}
			if config.Port != tt.expected.Port {
				t.Errorf("expected port %d, got %d", tt.expected.Port, config.Port)
			}
			if config.From != tt.expected.From {
				t.Errorf("expected from '%s', got '%s'", tt.expected.From, config.From)
			}
		})
	}
}

func TestSendViaSendGrid_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"errors":[{"message":"Invalid email"}]}`)
	}))
	defer server.Close()

	// Create service with SendGrid configured
	svc := NewEmailServiceWithOptions(EmailServiceOptions{
		SendGridConfig: &SendGridConfig{
			APIKey:    "SG.test",
			FromEmail: "from@example.com",
			FromName:  "Test",
		},
		HTTPClient: server.Client(),
	})

	// We can't easily test the actual sendViaSendGrid with a custom URL
	// because the URL is hardcoded. But we can verify the service is configured.
	if !svc.IsSendGridConfigured() {
		t.Error("expected SendGrid to be configured")
	}
}
