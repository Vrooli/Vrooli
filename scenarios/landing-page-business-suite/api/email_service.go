package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/experimentation"
	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

// SMTPConfig holds SMTP configuration from database
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// IsConfigured returns true if SMTP settings are complete
func (c *SMTPConfig) IsConfigured() bool {
	return c.Host != "" && c.Username != "" && c.Password != ""
}

// SendGridConfig holds SendGrid configuration
type SendGridConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// IsConfigured returns true if SendGrid settings are complete
func (c *SendGridConfig) IsConfigured() bool {
	return c.APIKey != "" && c.FromEmail != ""
}

// SMTPSenderFunc abstracts SMTP sending for testability.
type SMTPSenderFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// EmailServiceOptions configures the EmailService for testing.
type EmailServiceOptions struct {
	SendGridConfig *SendGridConfig
	HTTPClient     *http.Client
	SMTPSender     SMTPSenderFunc
}

// EmailService handles sending emails using config from branding or SendGrid
type EmailService struct {
	sendGridConfig *SendGridConfig
	httpClient     *http.Client
	smtpSender     SMTPSenderFunc
}

// NewEmailService creates a new email service
func NewEmailService() *EmailService {
	// Load SendGrid config from secrets
	apiKey := resolveSecret("SENDGRID_API_KEY")
	fromEmail := resolveConfig("EMAIL_FROM_ADDRESS")
	fromName := resolveConfig("EMAIL_FROM_NAME")

	var sgConfig *SendGridConfig
	if apiKey != "" {
		if fromEmail == "" {
			fromEmail = "noreply@example.com"
		}
		if fromName == "" {
			fromName = "App"
		}
		sgConfig = &SendGridConfig{
			APIKey:    apiKey,
			FromEmail: fromEmail,
			FromName:  fromName,
		}
		logStructured("sendgrid_configured", map[string]interface{}{
			"level":      "info",
			"from_email": fromEmail,
			"from_name":  fromName,
		})
	} else {
		logStructured("sendgrid_not_configured", map[string]interface{}{
			"level":   "warn",
			"message": "SENDGRID_API_KEY not set; magic link emails will be logged only",
		})
	}

	return &EmailService{
		sendGridConfig: sgConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		smtpSender: smtp.SendMail,
	}
}

// NewEmailServiceWithOptions creates a new email service with custom options (for testing).
func NewEmailServiceWithOptions(opts EmailServiceOptions) *EmailService {
	sender := opts.SMTPSender
	if sender == nil {
		sender = smtp.SendMail
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &EmailService{
		sendGridConfig: opts.SendGridConfig,
		httpClient:     httpClient,
		smtpSender:     sender,
	}
}

// IsSendGridConfigured returns true if SendGrid is properly configured
func (s *EmailService) IsSendGridConfigured() bool {
	return s.sendGridConfig != nil && s.sendGridConfig.IsConfigured()
}

// SendFeedbackNotification sends an email notification for new feedback
func (s *EmailService) SendFeedbackNotification(branding *experimentation.SiteBranding, feedback *domainmetrics.FeedbackRequest) error {
	config := s.extractSMTPConfig(branding)

	if !config.IsConfigured() {
		logStructured("email_skipped", map[string]interface{}{
			"reason": "smtp not configured in branding settings",
		})
		return nil
	}

	if branding.SupportEmail == nil || *branding.SupportEmail == "" {
		return nil
	}

	to := *branding.SupportEmail
	subject := fmt.Sprintf("[Feedback] %s: %s", feedbackTypeLabel(feedback.Type), feedback.Subject)

	body := fmt.Sprintf(`New feedback received

Type: %s
From: %s
Subject: %s

Message:
%s
`, feedbackTypeLabel(feedback.Type), feedback.Email, feedback.Subject, feedback.Message)

	if feedback.OrderID != nil && *feedback.OrderID != "" {
		body += fmt.Sprintf("\nOrder/Subscription ID: %s\n", *feedback.OrderID)
	}

	body += fmt.Sprintf("\n---\nSubmitted at: %s\nFeedback ID: %d\n", feedback.CreatedAt.Format("2006-01-02 15:04:05 UTC"), feedback.ID)

	return s.Send(config, to, subject, body)
}

// extractSMTPConfig pulls SMTP settings from branding
func (s *EmailService) extractSMTPConfig(branding *experimentation.SiteBranding) *SMTPConfig {
	config := &SMTPConfig{
		Port: 587, // default
	}

	if branding.SMTPHost != nil {
		config.Host = *branding.SMTPHost
	}
	if branding.SMTPPort != nil {
		config.Port = *branding.SMTPPort
	}
	if branding.SMTPUsername != nil {
		config.Username = *branding.SMTPUsername
	}
	if branding.SMTPPassword != nil {
		config.Password = *branding.SMTPPassword
	}
	if branding.SMTPFrom != nil && *branding.SMTPFrom != "" {
		config.From = *branding.SMTPFrom
	} else {
		config.From = config.Username // default to username
	}

	return config
}

// Send sends an email using the provided config
func (s *EmailService) Send(config *SMTPConfig, to, subject, body string) error {
	if !config.IsConfigured() {
		return fmt.Errorf("email service not configured")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		config.From, to, subject, body)

	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	sender := s.smtpSender
	if sender == nil {
		sender = smtp.SendMail
	}

	err := sender(addr, auth, config.From, []string{to}, []byte(msg))
	if err != nil {
		logStructuredError("email_send_failed", map[string]interface{}{
			"to":    to,
			"error": err.Error(),
		})
		return err
	}

	logStructured("email_sent", map[string]interface{}{
		"to":      to,
		"subject": subject,
	})

	return nil
}

func feedbackTypeLabel(t string) string {
	switch t {
	case "refund":
		return "Refund Request"
	case "bug":
		return "Bug Report"
	case "feature":
		return "Feature Request"
	default:
		return "General Feedback"
	}
}

// SendMagicLink sends a magic link email via SendGrid.
// If SendGrid is not configured, it logs the link for development purposes.
func (s *EmailService) SendMagicLink(to, magicLink, appName string) error {
	if appName == "" {
		appName = "App"
	}

	subject := fmt.Sprintf("Sign in to %s", appName)
	htmlContent := buildMagicLinkHTML(magicLink, appName)
	textContent := buildMagicLinkText(magicLink, appName)

	// If SendGrid is not configured, log the link for development
	if !s.IsSendGridConfigured() {
		logStructured("magic_link_dev_mode", map[string]interface{}{
			"level":      "info",
			"to":         to,
			"magic_link": magicLink,
			"message":    "SendGrid not configured - magic link logged for development",
		})
		return nil
	}

	return s.sendViaSendGrid(to, subject, textContent, htmlContent)
}

// sendViaSendGrid sends an email via the SendGrid API
func (s *EmailService) sendViaSendGrid(to, subject, textContent, htmlContent string) error {
	if s.sendGridConfig == nil || !s.sendGridConfig.IsConfigured() {
		return fmt.Errorf("SendGrid not configured")
	}

	// Build SendGrid API request
	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": to},
				},
			},
		},
		"from": map[string]string{
			"email": s.sendGridConfig.FromEmail,
			"name":  s.sendGridConfig.FromName,
		},
		"subject": subject,
		"content": []map[string]string{
			{"type": "text/plain", "value": textContent},
			{"type": "text/html", "value": htmlContent},
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal SendGrid payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create SendGrid request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.sendGridConfig.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logStructuredError("sendgrid_request_failed", map[string]interface{}{
			"error": err.Error(),
			"to":    to,
		})
		return fmt.Errorf("SendGrid request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(bodyBytes))
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		logStructuredError("sendgrid_api_error", map[string]interface{}{
			"status": resp.StatusCode,
			"body":   bodyStr,
			"to":     to,
		})
		return fmt.Errorf("SendGrid API error: %d - %s", resp.StatusCode, bodyStr)
	}

	logStructured("magic_link_sent", map[string]interface{}{
		"level": "info",
		"to":    to,
	})

	return nil
}

// buildMagicLinkHTML creates an HTML email for magic link authentication
func buildMagicLinkHTML(magicLink, appName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sign in to %s</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; margin: 0; padding: 40px 20px; background-color: #f5f5f5;">
    <div style="max-width: 480px; margin: 0 auto; background: white; border-radius: 8px; padding: 40px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
        <h1 style="margin: 0 0 24px; font-size: 24px; font-weight: 600; color: #333;">Sign in to %s</h1>
        <p style="margin: 0 0 24px; color: #555; line-height: 1.5;">Click the button below to sign in to your account. This link will expire in 15 minutes.</p>
        <a href="%s" style="display: inline-block; padding: 14px 28px; background-color: #0066cc; color: white; text-decoration: none; border-radius: 6px; font-weight: 500; font-size: 16px;">Sign In</a>
        <p style="margin: 24px 0 0; color: #888; font-size: 14px; line-height: 1.5;">If you didn't request this email, you can safely ignore it.</p>
        <hr style="margin: 24px 0; border: none; border-top: 1px solid #eee;">
        <p style="margin: 0; color: #888; font-size: 12px;">If the button doesn't work, copy and paste this link into your browser:</p>
        <p style="margin: 8px 0 0; color: #0066cc; font-size: 12px; word-break: break-all;">%s</p>
    </div>
</body>
</html>`, appName, appName, magicLink, magicLink)
}

// buildMagicLinkText creates a plain text email for magic link authentication
func buildMagicLinkText(magicLink, appName string) string {
	return fmt.Sprintf(`Sign in to %s

Click the link below to sign in to your account. This link will expire in 15 minutes.

%s

If you didn't request this email, you can safely ignore it.
`, appName, magicLink)
}
