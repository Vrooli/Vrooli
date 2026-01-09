package main

import (
	"fmt"
	"net/smtp"
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

// EmailService handles sending emails using config from branding
type EmailService struct{}

// NewEmailService creates a new email service
func NewEmailService() *EmailService {
	return &EmailService{}
}

// SendFeedbackNotification sends an email notification for new feedback
func (s *EmailService) SendFeedbackNotification(branding *SiteBranding, feedback *FeedbackRequest) error {
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
func (s *EmailService) extractSMTPConfig(branding *SiteBranding) *SMTPConfig {
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

	err := smtp.SendMail(addr, auth, config.From, []string{to}, []byte(msg))
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
