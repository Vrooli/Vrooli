package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/emaildelivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
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
	return c.Host != "" && c.Username != "" && c.Password != "" && c.From != ""
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
	// SendGridEndpoint is injectable so provider behavior can be tested against
	// a bounded local server without weakening the production endpoint.
	SendGridEndpoint string
	// MailgunEndpoint is injectable so provider behavior can be tested against
	// a bounded local server without weakening the production endpoint.
	MailgunEndpoint string
	HTTPClient      *http.Client
	SMTPSender      SMTPSenderFunc
	// SMTPPasswordResolver supplies the authority-owned SMTP secret. The
	// branding model deliberately cannot provide this value.
	SMTPPasswordResolver func() (string, error)
	MailgunKeyResolver   func() string
}

// EmailService handles sending emails using public branding settings plus
// authority-owned provider credentials.
type EmailService struct {
	sendGridConfig       *SendGridConfig
	sendGridEndpoint     string
	httpClient           *http.Client
	smtpSender           SMTPSenderFunc
	smtpPasswordResolver func() (string, error)
	brandingSource       func() *experimentation.SiteBranding
	developmentRecorder  func(to, subject, text, html string) error
	sendGridKeyResolver  func() string
	mailer               *emaildelivery.Mailer
	mailgunEndpoint      string
	mailgunKeyResolver   func() string
}

const defaultSendGridEndpoint = "https://api.sendgrid.com/v3/mail/send"
const defaultMailgunEndpoint = "https://api.mailgun.net"

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

// recordDevelopmentEmail is an honest local transport: it records the full
// message in a durable, operator-readable spool and only then reports success.
// It is installed only by the normal non-production composition; tests that
// construct EmailServiceWithOptions still see a missing-provider error.
func recordDevelopmentEmail(to, subject, text, html string) error {
	path := strings.TrimSpace(os.Getenv("LPBS_DEVELOPMENT_EMAIL_SPOOL"))
	if path == "" {
		path = filepath.Join("..", ".vrooli", "state", "email-development.ndjson")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	record, err := json.Marshal(map[string]string{"to": to, "subject": subject, "text": text, "html": html, "recorded_at": time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(record, '\n'))
	return err
}

// NewEmailService creates a new email service
func NewEmailService() *EmailService {
	// Load SendGrid config from secrets
	apiKey := resolveSecret("SENDGRID_API_KEY")
	fromEmail := resolveConfig("EMAIL_FROM_ADDRESS")
	fromName := resolveConfig("EMAIL_FROM_NAME")

	var sgConfig *SendGridConfig
	if apiKey != "" && strings.TrimSpace(fromEmail) != "" {
		if fromName == "" {
			fromName = "App"
		}
		sgConfig = &SendGridConfig{
			APIKey:    apiKey,
			FromEmail: fromEmail,
			FromName:  fromName,
		}
		logx.Info("sendgrid_configured", map[string]interface{}{
			"level":      "info",
			"from_email": fromEmail,
			"from_name":  fromName,
		})
	} else {
		logx.Info("sendgrid_not_configured", map[string]interface{}{
			"level":   "warn",
			"message": "SENDGRID_API_KEY not set; magic-link delivery is unavailable",
		})
	}

	service := &EmailService{
		sendGridConfig:   sgConfig,
		sendGridEndpoint: defaultSendGridEndpoint,
		mailgunEndpoint:  firstNonEmpty(resolveConfig("MAILGUN_API_BASE_URL"), defaultMailgunEndpoint),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		smtpSender:           secureSMTPSend,
		smtpPasswordResolver: func() (string, error) { return administration.ResolveAuthorityCredential("SMTP_PASSWORD") },
		sendGridKeyResolver:  func() string { return resolveSecret("SENDGRID_API_KEY") },
		mailgunKeyResolver:   func() string { return resolveSecret("MAILGUN_API_KEY") },
	}
	if !isProductionSecurityEnvironment() {
		service.developmentRecorder = recordDevelopmentEmail
	}
	return service
}

// NewEmailServiceWithOptions creates a new email service with custom options (for testing).
func NewEmailServiceWithOptions(opts EmailServiceOptions) *EmailService {
	sender := opts.SMTPSender
	if sender == nil {
		sender = secureSMTPSend
	}
	passwordResolver := opts.SMTPPasswordResolver
	if passwordResolver == nil {
		passwordResolver = func() (string, error) { return administration.ResolveAuthorityCredential("SMTP_PASSWORD") }
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &EmailService{
		sendGridConfig:       opts.SendGridConfig,
		sendGridEndpoint:     firstNonEmpty(opts.SendGridEndpoint, defaultSendGridEndpoint),
		mailgunEndpoint:      firstNonEmpty(opts.MailgunEndpoint, defaultMailgunEndpoint),
		httpClient:           httpClient,
		smtpSender:           sender,
		smtpPasswordResolver: passwordResolver,
		mailgunKeyResolver:   opts.MailgunKeyResolver,
	}
}

// IsSendGridConfigured returns true if SendGrid is properly configured
func (s *EmailService) IsSendGridConfigured() bool {
	s.refreshSendGridConfig()
	return s.sendGridConfig != nil && s.sendGridConfig.IsConfigured()
}

func (s *EmailService) refreshSendGridConfig() {
	if s.sendGridKeyResolver == nil {
		return
	}
	key := strings.TrimSpace(s.sendGridKeyResolver())
	from := strings.TrimSpace(resolveConfig("EMAIL_FROM_ADDRESS"))
	if key == "" || from == "" {
		s.sendGridConfig = nil
		return
	}
	s.sendGridConfig = &SendGridConfig{APIKey: key, FromEmail: from, FromName: firstNonEmpty(resolveConfig("EMAIL_FROM_NAME"), "App")}
}

// SendFeedbackNotification sends an email notification for new feedback
func (s *EmailService) SendFeedbackNotification(branding *experimentation.SiteBranding, feedback *domainmetrics.FeedbackRequest) error {
	config, configErr := s.extractSMTPConfigWithError(branding)
	if configErr != nil {
		if s.developmentRecorder == nil {
			return fmt.Errorf("resolve SMTP configuration: %w", configErr)
		}
		config = &SMTPConfig{}
	}

	if !config.IsConfigured() {
		if s.developmentRecorder != nil && branding != nil && branding.SupportEmail != nil && strings.TrimSpace(*branding.SupportEmail) != "" {
			subject := fmt.Sprintf("[Feedback] %s: %s", feedbackTypeLabel(feedback.Type), feedback.Subject)
			body := fmt.Sprintf("New feedback received\n\nFrom: %s\nSubject: %s\n\n%s", feedback.Email, feedback.Subject, feedback.Message)
			return s.developmentRecorder(*branding.SupportEmail, subject, body, "<pre>"+html.EscapeString(body)+"</pre>")
		}
		return fmt.Errorf("SMTP provider is not configured")
	}

	if branding.SupportEmail == nil || *branding.SupportEmail == "" {
		return fmt.Errorf("feedback support email is not configured")
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
	if s.mailer != nil {
		return s.mailer.Enqueue(context.Background(), emaildelivery.Message{
			DedupeKey: feedbackDedupeKey(feedback), IdempotencyKey: feedbackDedupeKey(feedback),
			Purpose: emaildelivery.PurposeContact, Recipient: to, Sender: config.From,
			Subject: subject, TextBody: body, HTMLBody: "<pre>" + html.EscapeString(body) + "</pre>",
			TemplateRef: "feedback.notification", Priority: 40,
		})
	}

	return s.Send(config, to, subject, body)
}

// UseMailer switches the service to the durable outbox boundary. It is used
// by production composition; test-only constructions retain the synchronous
// transport so their provider assertions remain local and deterministic.
func (s *EmailService) UseMailer(mailer *emaildelivery.Mailer) {
	s.mailer = mailer
}

func feedbackDedupeKey(feedback *domainmetrics.FeedbackRequest) string {
	return fmt.Sprintf("feedback:%d:%s", feedback.ID, strings.ToLower(strings.TrimSpace(feedback.Email)))
}

// extractSMTPConfig pulls SMTP settings from branding
func (s *EmailService) extractSMTPConfig(branding *experimentation.SiteBranding) *SMTPConfig {
	config, _ := s.extractSMTPConfigWithError(branding)
	return config
}

func (s *EmailService) extractSMTPConfigWithError(branding *experimentation.SiteBranding) (*SMTPConfig, error) {
	config := &SMTPConfig{
		Port: 587, // default
	}
	if branding == nil {
		return config, nil
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
	if s.smtpPasswordResolver != nil {
		password, err := s.smtpPasswordResolver()
		if err != nil {
			return config, err
		}
		config.Password = password
	}
	if branding.SMTPFrom != nil && *branding.SMTPFrom != "" {
		config.From = *branding.SMTPFrom
	}

	return config, nil
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
		sender = secureSMTPSend
	}

	err := sender(addr, auth, config.From, []string{to}, []byte(msg))
	if err != nil {
		logx.Error("email_send_failed", map[string]interface{}{
			"to":    to,
			"error": err.Error(),
		})
		return err
	}

	logx.Info("email_sent", map[string]interface{}{
		"to":      to,
		"subject": subject,
	})

	return nil
}

// secureSMTPSend bounds connection and protocol time, supports implicit TLS
// on port 465, and refuses to authenticate to a clear-text relay.
func secureSMTPSend(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse SMTP address: %w", err)
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var conn net.Conn
	if port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("dial SMTP server: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()
	if port != "465" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not offer STARTTLS; refusing clear-text authentication")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate SMTP: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("set SMTP recipient: %w", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("complete SMTP message: %w", err)
	}
	return client.Quit()
}

func newMessageID(from string) string {
	host := "localhost"
	if at := strings.LastIndex(from, "@"); at >= 0 && strings.TrimSpace(from[at+1:]) != "" {
		host = strings.TrimSpace(from[at+1:])
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), host)
	}
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(b), host)
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

// SendMagicLink sends a magic link email via SendGrid. A missing provider is
// never a successful delivery and the bearer URL is never logged.
func (s *EmailService) SendMagicLink(to, magicLink, appName string) error {
	if appName == "" {
		appName = "App"
	}

	subject := fmt.Sprintf("Sign in to %s", appName)
	htmlContent := buildMagicLinkHTML(magicLink, appName)
	textContent := buildMagicLinkText(magicLink, appName)
	if s.mailer != nil {
		digest := sha256.Sum256([]byte(magicLink))
		dedupe := fmt.Sprintf("magic-link:%s:%x", strings.ToLower(strings.TrimSpace(to)), digest[:])
		return s.mailer.Enqueue(context.Background(), emaildelivery.Message{
			DedupeKey: dedupe, IdempotencyKey: dedupe, Purpose: emaildelivery.PurposeMagicLink,
			Recipient: to, Sender: s.senderIdentity(), Subject: subject, TextBody: textContent,
			HTMLBody: htmlContent, TemplateRef: "auth.magic-link", Priority: 100,
		})
	}

	if !s.IsSendGridConfigured() {
		if s.developmentRecorder != nil {
			return s.developmentRecorder(to, subject, textContent, htmlContent)
		}
		return fmt.Errorf("magic-link email provider is not configured")
	}

	_, err := s.sendViaSendGridWithMetadata(context.Background(), to, subject, textContent, htmlContent, "")
	return err
}

// sendViaSendGrid sends an email via the SendGrid API
func (s *EmailService) sendViaSendGrid(to, subject, textContent, htmlContent string) error {
	_, err := s.sendViaSendGridWithMetadata(context.Background(), to, subject, textContent, htmlContent, "")
	return err
}

func (s *EmailService) sendViaSendGridWithMetadata(ctx context.Context, to, subject, textContent, htmlContent, requestID string) (*administration.SignInDelivery, error) {
	category := ""
	if requestID != "" {
		category = "lpbs-auth"
	}
	return s.sendViaSendGridCategory(ctx, to, subject, textContent, htmlContent, requestID, category)
}

// SendSecurityNotification sends a tracking-free administrator security
// notice using a distinct provider category.
func (s *EmailService) SendSecurityNotification(ctx context.Context, to, event, detail string) error {
	subject := "LPBS administrator security alert"
	textBody := fmt.Sprintf("Security event: %s\n\n%s", event, detail)
	htmlBody := fmt.Sprintf("<p>Security event: <strong>%s</strong></p><p>%s</p>", html.EscapeString(event), html.EscapeString(detail))
	if s.mailer != nil {
		dedupe := fmt.Sprintf("security:%s:%s:%x", strings.ToLower(strings.TrimSpace(to)), event, sha256.Sum256([]byte(detail)))
		return s.mailer.Enqueue(ctx, emaildelivery.Message{DedupeKey: dedupe, IdempotencyKey: dedupe, Purpose: emaildelivery.PurposeSecurity, Recipient: to, Sender: s.senderIdentity(), Subject: subject, TextBody: textBody, HTMLBody: htmlBody, TemplateRef: "security.alert", Priority: 80})
	}
	_, err := s.sendViaSendGridCategory(ctx, to, subject, textBody, htmlBody, "", "lpbs-admin-security")
	return err
}

func (s *EmailService) SendPasskeyNotification(ctx context.Context, to, event, detail string) error {
	return s.sendPasskeyNotification(ctx, to, event, detail)
}

func (s *EmailService) sendPasskeyNotification(ctx context.Context, to, event, detail string) error {
	subject := "A passkey was added to your account"
	if event == "passkey_removed" {
		subject = "A passkey was removed from your account"
	}
	textBody := fmt.Sprintf("%s\n\n%s\n\nReview your account security at /account/security.", subject, detail)
	htmlBody := fmt.Sprintf("<p>%s</p><p>%s</p><p>Review your account security at <a href=\"/account/security\">/account/security</a>.</p>", html.EscapeString(subject), html.EscapeString(detail))
	if s.mailer != nil {
		dedupe := fmt.Sprintf("passkey:%s:%s:%x", strings.ToLower(strings.TrimSpace(to)), event, sha256.Sum256([]byte(detail)))
		return s.mailer.Enqueue(ctx, emaildelivery.Message{DedupeKey: dedupe, IdempotencyKey: dedupe, Purpose: emaildelivery.PurposePasskey, Recipient: to, Sender: s.senderIdentity(), Subject: subject, TextBody: textBody, HTMLBody: htmlBody, TemplateRef: "passkey.notice", Priority: 60})
	}
	_, err := s.sendViaSendGridCategory(ctx, to, subject, textBody, htmlBody, "", "lpbs-auth")
	return err
}

func (s *EmailService) senderIdentity() string {
	if value := strings.TrimSpace(resolveConfig("EMAIL_FROM_ADDRESS")); value != "" {
		return value
	}
	if branding := s.currentBranding(); branding != nil && branding.SMTPFrom != nil {
		return strings.TrimSpace(*branding.SMTPFrom)
	}
	return ""
}

func (s *EmailService) sendViaSendGridCategory(ctx context.Context, to, subject, textContent, htmlContent, requestID, category string) (*administration.SignInDelivery, error) {
	s.refreshSendGridConfig()
	if s.sendGridConfig == nil || !s.sendGridConfig.IsConfigured() {
		if s.developmentRecorder != nil {
			if err := s.developmentRecorder(to, subject, textContent, htmlContent); err != nil {
				return nil, fmt.Errorf("record development email: %w", err)
			}
			return &administration.SignInDelivery{Provider: "development"}, nil
		}
		return nil, fmt.Errorf("SendGrid not configured")
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
	if category != "" {
		payload["categories"] = []string{category}
		payload["tracking_settings"] = map[string]any{
			"click_tracking":        map[string]any{"enable": false, "enable_text": false},
			"open_tracking":         map[string]any{"enable": false},
			"subscription_tracking": map[string]any{"enable": false},
			"ganalytics":            map[string]any{"enable": false},
		}
		if requestID != "" {
			payload["custom_args"] = map[string]string{"lpbs_sign_in_request_id": requestID}
		}
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal SendGrid payload: %w", err)
	}

	endpoint := s.sendGridEndpoint
	if endpoint == "" {
		endpoint = defaultSendGridEndpoint
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create SendGrid request: %w", err)
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 10*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	req = req.WithContext(ctx)

	req.Header.Set("Authorization", "Bearer "+s.sendGridConfig.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logx.Error("sendgrid_request_failed", map[string]interface{}{
			"error": err.Error(),
			"to":    to,
		})
		return nil, fmt.Errorf("SendGrid request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(bodyBytes))
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		logx.Error("sendgrid_api_error", map[string]interface{}{
			"status": resp.StatusCode,
			"body":   bodyStr,
			"to":     to,
		})
		return nil, fmt.Errorf("SendGrid API error: %d - %s", resp.StatusCode, bodyStr)
	}

	logx.Info("magic_link_sent", map[string]interface{}{
		"level": "info",
		"to":    to,
	})

	return &administration.SignInDelivery{Provider: emaildelivery.ProviderSendGrid, ProviderMessageID: strings.TrimSpace(resp.Header.Get("X-Message-Id"))}, nil
}

func (s *EmailService) mailgunAPIKey() string {
	if s == nil || s.mailgunKeyResolver == nil {
		return ""
	}
	key := strings.TrimSpace(s.mailgunKeyResolver())
	return strings.TrimPrefix(key, "api:")
}

func (s *EmailService) mailgunDomain(sender string) string {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(sender)), "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSuffix(parts[1], ".")
}

func (s *EmailService) mailgunBaseURL() string {
	if s == nil || strings.TrimSpace(s.mailgunEndpoint) == "" {
		return defaultMailgunEndpoint
	}
	return strings.TrimRight(strings.TrimSpace(s.mailgunEndpoint), "/")
}

// verifyMailgunAPI performs a read-only domain lookup. It returns the HTTP
// status (or zero when the provider could not be reached) and a redacted
// operator-facing detail.
func (s *EmailService) verifyMailgunAPI(ctx context.Context) (int, string) {
	key := s.mailgunAPIKey()
	domain := s.mailgunDomain(s.senderIdentity())
	if key == "" {
		return http.StatusUnauthorized, "no Mailgun API key is configured"
	}
	if domain == "" {
		return http.StatusBadRequest, "the sender address does not identify a Mailgun sending domain"
	}
	endpoint := s.mailgunBaseURL() + "/v3/" + url.PathEscape(domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, "Mailgun verification request could not be created"
	}
	req.SetBasicAuth("api", key)
	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "Mailgun could not be reached to verify the API key"
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return resp.StatusCode, fmt.Sprintf("Mailgun rejected the API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Sprintf("Mailgun answered HTTP %d when asked to verify the sending domain", resp.StatusCode)
	}
	return resp.StatusCode, "the Mailgun API key authenticates for the sending domain"
}

// sendViaMailgun sends through Mailgun's HTTPS API using the provider API key
// and the domain portion of the configured sender address. The API key never
// enters a URL, payload, log, or returned error.
func (s *EmailService) sendViaMailgun(ctx context.Context, to, sender, subject, textContent, htmlContent string) (*administration.SignInDelivery, error) {
	key := s.mailgunAPIKey()
	domain := s.mailgunDomain(sender)
	if key == "" {
		return nil, fmt.Errorf("Mailgun API key is not configured")
	}
	if domain == "" {
		return nil, fmt.Errorf("sender address does not identify a Mailgun sending domain")
	}
	form := url.Values{}
	form.Set("from", sender)
	form.Set("to", to)
	form.Set("subject", subject)
	form.Set("text", textContent)
	if strings.TrimSpace(htmlContent) != "" {
		form.Set("html", htmlContent)
	}
	endpoint := s.mailgunBaseURL() + "/v3/" + url.PathEscape(domain) + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create Mailgun request: %w", err)
	}
	req.SetBasicAuth("api", key)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Mailgun request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyText := strings.TrimSpace(string(body))
		if len(bodyText) > 500 {
			bodyText = bodyText[:500] + "..."
		}
		return nil, fmt.Errorf("Mailgun API error: %d - %s", resp.StatusCode, bodyText)
	}
	var response struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &response)
	return &administration.SignInDelivery{Provider: emaildelivery.ProviderMailgun, ProviderMessageID: strings.TrimSpace(response.ID)}, nil
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
