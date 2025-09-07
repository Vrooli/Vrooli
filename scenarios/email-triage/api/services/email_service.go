package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"gopkg.in/gomail.v2"
	
	"email-triage/models"
)

// EmailService handles IMAP/SMTP operations with mail servers
type EmailService struct {
	mailServerURL string
}

// NewEmailService creates a new EmailService instance
func NewEmailService(mailServerURL string) *EmailService {
	return &EmailService{
		mailServerURL: mailServerURL,
	}
}

// TestConnection tests IMAP and SMTP connections for an email account
func (es *EmailService) TestConnection(account *models.EmailAccount) error {
	// Parse IMAP settings
	var imapConfig models.IMAPConfig
	if err := json.Unmarshal(account.IMAPSettings, &imapConfig); err != nil {
		return fmt.Errorf("invalid IMAP settings: %w", err)
	}
	
	// Parse SMTP settings
	var smtpConfig models.SMTPConfig
	if err := json.Unmarshal(account.SMTPSettings, &smtpConfig); err != nil {
		return fmt.Errorf("invalid SMTP settings: %w", err)
	}
	
	// Test IMAP connection
	if err := es.testIMAPConnection(imapConfig); err != nil {
		return fmt.Errorf("IMAP connection failed: %w", err)
	}
	
	// Test SMTP connection
	if err := es.testSMTPConnection(smtpConfig); err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	
	return nil
}

func (es *EmailService) testIMAPConnection(config models.IMAPConfig) error {
	// Connect to IMAP server
	address := fmt.Sprintf("%s:%d", config.Server, config.Port)
	
	var c *client.Client
	var err error
	
	if config.UseTLS {
		tlsConfig := &tls.Config{ServerName: config.Server}
		c, err = client.DialTLS(address, tlsConfig)
	} else {
		c, err = client.Dial(address)
	}
	
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer c.Close()
	
	// Login
	if err := c.Login(config.Username, config.Password); err != nil {
		return fmt.Errorf("IMAP login failed: %w", err)
	}
	
	// Try to select INBOX
	_, err = c.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}
	
	return nil
}

func (es *EmailService) testSMTPConnection(config models.SMTPConfig) error {
	d := gomail.NewDialer(config.Server, config.Port, config.Username, config.Password)
	
	if config.UseTLS {
		d.TLSConfig = &tls.Config{ServerName: config.Server}
	}
	
	// Test connection by opening and closing
	closer, err := d.Dial()
	if err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	closer.Close()
	
	return nil
}

// FetchNewEmails retrieves new emails from an account since last sync
func (es *EmailService) FetchNewEmails(account *models.EmailAccount, since time.Time) ([]*models.ProcessedEmail, error) {
	// Parse IMAP settings
	var imapConfig models.IMAPConfig
	if err := json.Unmarshal(account.IMAPSettings, &imapConfig); err != nil {
		return nil, fmt.Errorf("invalid IMAP settings: %w", err)
	}
	
	// Connect to IMAP server
	address := fmt.Sprintf("%s:%d", imapConfig.Server, imapConfig.Port)
	
	var c *client.Client
	var err error
	
	if imapConfig.UseTLS {
		tlsConfig := &tls.Config{ServerName: imapConfig.Server}
		c, err = client.DialTLS(address, tlsConfig)
	} else {
		c, err = client.Dial(address)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer c.Close()
	
	// Login
	if err := c.Login(imapConfig.Username, imapConfig.Password); err != nil {
		return nil, fmt.Errorf("IMAP login failed: %w", err)
	}
	
	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}
	
	// Search for messages since last sync
	criteria := imap.NewSearchCriteria()
	criteria.Since = since
	
	uids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("email search failed: %w", err)
	}
	
	if len(uids) == 0 {
		return []*models.ProcessedEmail{}, nil
	}
	
	// Fetch messages
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBodyStructure, imap.FetchInternalDate}, messages)
	}()
	
	var emails []*models.ProcessedEmail
	
	for msg := range messages {
		email, err := es.convertIMAPMessage(msg, account.ID)
		if err != nil {
			// Log error but continue processing other emails
			fmt.Printf("Error converting message %d: %v\n", msg.SeqNum, err)
			continue
		}
		emails = append(emails, email)
	}
	
	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch operation failed: %w", err)
	}
	
	return emails, nil
}

func (es *EmailService) convertIMAPMessage(msg *imap.Message, accountID string) (*models.ProcessedEmail, error) {
	if msg.Envelope == nil {
		return nil, fmt.Errorf("message envelope is nil")
	}
	
	email := &models.ProcessedEmail{
		ID:        generateUUID(),
		AccountID: accountID,
		MessageID: msg.Envelope.MessageId,
		Subject:   msg.Envelope.Subject,
	}
	
	// Extract sender
	if len(msg.Envelope.From) > 0 {
		email.SenderEmail = msg.Envelope.From[0].Address()
	}
	
	// Extract recipients
	var recipients []string
	for _, addr := range msg.Envelope.To {
		recipients = append(recipients, addr.Address())
	}
	for _, addr := range msg.Envelope.Cc {
		recipients = append(recipients, addr.Address())
	}
	email.RecipientEmails = recipients
	
	// Use internal date as processed time
	email.ProcessedAt = msg.InternalDate
	
	// TODO: Extract body content and create preview
	// This would require additional IMAP fetch operations
	email.BodyPreview = "Preview extraction not implemented yet"
	email.FullBody = ""
	
	return email, nil
}

// SendEmail sends an email using SMTP
func (es *EmailService) SendEmail(account *models.EmailAccount, to []string, subject, body string) error {
	// Parse SMTP settings
	var smtpConfig models.SMTPConfig
	if err := json.Unmarshal(account.SMTPSettings, &smtpConfig); err != nil {
		return fmt.Errorf("invalid SMTP settings: %w", err)
	}
	
	// Create message
	m := gomail.NewMessage()
	m.SetHeader("From", account.EmailAddress)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	
	// Setup dialer
	d := gomail.NewDialer(smtpConfig.Server, smtpConfig.Port, smtpConfig.Username, smtpConfig.Password)
	
	if smtpConfig.UseTLS {
		d.TLSConfig = &tls.Config{ServerName: smtpConfig.Server}
	}
	
	// Send email
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}

// ForwardEmail forwards an email to specified recipients
func (es *EmailService) ForwardEmail(account *models.EmailAccount, originalEmail *models.ProcessedEmail, forwardTo []string) error {
	subject := fmt.Sprintf("Fwd: %s", originalEmail.Subject)
	
	body := fmt.Sprintf(`---------- Forwarded message ----------
From: %s
Subject: %s
Date: %s

%s`, originalEmail.SenderEmail, originalEmail.Subject, originalEmail.ProcessedAt.Format(time.RFC3339), originalEmail.FullBody)
	
	return es.SendEmail(account, forwardTo, subject, body)
}

// Helper function to generate UUIDs (placeholder implementation)
func generateUUID() string {
	// This would typically use github.com/google/uuid
	return fmt.Sprintf("uuid-%d", time.Now().UnixNano())
}