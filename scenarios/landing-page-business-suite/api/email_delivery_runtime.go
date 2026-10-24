package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	maildns "github.com/vrooli/vrooli/packages/maildns-go"
	"landing-page-business-suite-api/internal/emaildelivery"
	"landing-page-business-suite-api/internal/logx"
)

// emailDeliveryAdapter is the composition boundary between the provider
// registry and the existing wire transports. It classifies ambiguous network
// failures as unknown so the worker never submits the same message to a
// different provider after an uncertain handoff.
type emailDeliveryAdapter struct {
	service *EmailService
}

func (a emailDeliveryAdapter) Send(ctx context.Context, provider emaildelivery.Provider, message emaildelivery.Message) emaildelivery.Outcome {
	if a.service == nil {
		return emaildelivery.Outcome{Kind: emaildelivery.OutcomePermanent, Detail: "email service unavailable"}
	}
	switch provider.Transport {
	case "mailgun_http":
		delivery, err := a.service.sendViaMailgun(ctx, message.Recipient, message.Sender, message.Subject, message.TextBody, message.HTMLBody)
		if err == nil {
			return emaildelivery.Outcome{Kind: emaildelivery.OutcomeAccepted, ProviderMessageID: delivery.ProviderMessageID}
		}
		return classifyEmailTransportError(err)
	case "sendgrid_http":
		delivery, err := a.service.sendViaSendGridCategory(ctx, message.Recipient, message.Subject, message.TextBody, message.HTMLBody, message.IdempotencyKey, string(message.Purpose))
		if err == nil {
			return emaildelivery.Outcome{Kind: emaildelivery.OutcomeAccepted, ProviderMessageID: delivery.ProviderMessageID}
		}
		return classifyEmailTransportError(err)
	case "mailgun_smtp", "smtp":
		branding := a.service.currentBranding()
		config, err := a.service.extractSMTPConfigWithError(branding)
		if err != nil {
			return emaildelivery.Outcome{Kind: emaildelivery.OutcomePermanent, Detail: "smtp credentials: " + err.Error()}
		}
		if !config.IsConfigured() {
			return emaildelivery.Outcome{Kind: emaildelivery.OutcomePermanent, Detail: "smtp sender configuration is incomplete"}
		}
		if err := a.service.sendSMTPMultipartWithIdempotency(config, message.Recipient, message.Subject, message.TextBody, message.HTMLBody, message.IdempotencyKey); err != nil {
			return classifyEmailTransportError(err)
		}
		return emaildelivery.Outcome{Kind: emaildelivery.OutcomeAccepted}
	case "development":
		if a.service.developmentRecorder == nil {
			return emaildelivery.Outcome{Kind: emaildelivery.OutcomePermanent, Detail: "development email recorder unavailable"}
		}
		if err := a.service.developmentRecorder(message.Recipient, message.Subject, message.TextBody, message.HTMLBody); err != nil {
			return emaildelivery.Outcome{Kind: emaildelivery.OutcomeTemporary, Detail: err.Error()}
		}
		return emaildelivery.Outcome{Kind: emaildelivery.OutcomeAccepted}
	default:
		return emaildelivery.Outcome{Kind: emaildelivery.OutcomePermanent, Detail: "unsupported email transport " + provider.Transport}
	}
}

func classifyEmailTransportError(err error) emaildelivery.Outcome {
	detail := strings.TrimSpace(err.Error())
	if detail == "" {
		detail = "email transport failed"
	}
	lower := strings.ToLower(detail)
	if strings.Contains(lower, "timeout") || strings.Contains(lower, "context deadline") || strings.Contains(lower, "connection reset") || strings.Contains(lower, "context canceled") || strings.Contains(lower, "http 5") {
		return emaildelivery.Outcome{Kind: emaildelivery.OutcomeUnknown, Detail: detail}
	}
	if strings.Contains(lower, "429") || strings.Contains(lower, "rate limit") || strings.Contains(lower, "temporarily") {
		return emaildelivery.Outcome{Kind: emaildelivery.OutcomeTemporary, Detail: detail}
	}
	return emaildelivery.Outcome{Kind: emaildelivery.OutcomePermanent, Detail: detail}
}

type emailCredentialChecker struct{ service *EmailService }

type emailQuotaChecker struct {
	db interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
}

func (c emailQuotaChecker) Available(ctx context.Context, provider emaildelivery.Provider) (bool, string) {
	if c.db == nil {
		return true, "quota state unavailable; no persisted window was found"
	}
	for period, ceiling := range provider.PublishedLimits {
		if ceiling <= 0 {
			continue
		}
		var used, persistedCeiling int
		err := c.db.QueryRowContext(ctx, `SELECT used_count, ceiling FROM email_quota_windows WHERE provider_id=$1 AND window_key=$2 AND window_ends_at > NOW()`, provider.ID, period).Scan(&used, &persistedCeiling)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return false, "quota state could not be read"
		}
		if persistedCeiling > 0 && used >= persistedCeiling {
			return false, fmt.Sprintf("%s quota exhausted (%d/%d)", period, used, persistedCeiling)
		}
	}
	return true, "quota available"
}

type emailCircuitChecker struct {
	db interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
}

func (c emailCircuitChecker) Available(ctx context.Context, provider emaildelivery.Provider) (bool, string) {
	if c.db == nil {
		return true, "circuit state unavailable; treating provider as closed"
	}
	var openUntil sql.NullTime
	err := c.db.QueryRowContext(ctx, `SELECT open_until FROM email_provider_circuits WHERE provider_id=$1`, provider.ID).Scan(&openUntil)
	if err == sql.ErrNoRows || !openUntil.Valid || !openUntil.Time.After(time.Now().UTC()) {
		return true, "provider circuit is closed"
	}
	return false, "provider circuit is open until " + openUntil.Time.UTC().Format(time.RFC3339)
}

func (c emailCredentialChecker) Verify(ctx context.Context, provider emaildelivery.Provider) (bool, string) {
	if c.service == nil {
		return false, "email service unavailable"
	}
	switch provider.Transport {
	case "mailgun_http":
		status, detail := c.service.verifyMailgunAPI(ctx)
		return status >= 200 && status < 300, detail
	case "sendgrid_http":
		if c.service.IsSendGridConfigured() {
			return true, "SendGrid credential and sender are configured"
		}
		return false, "SendGrid credential or sender is not configured"
	case "mailgun_smtp", "smtp":
		config, err := c.service.extractSMTPConfigWithError(c.service.currentBranding())
		if err != nil {
			return false, err.Error()
		}
		if config.IsConfigured() {
			return true, "SMTP credential and sender are configured"
		}
		return false, "SMTP credential or sender is not configured"
	default:
		return false, "unsupported email transport"
	}
}

type emailDNSChecker struct {
	service *maildns.Service
	domain  func() string
}

func (c emailDNSChecker) Verify(ctx context.Context, provider emaildelivery.Provider) (bool, string) {
	if c.service == nil || c.domain == nil {
		return false, "mail DNS verifier unavailable"
	}
	requirements := providerMailDNSRequirements(provider)
	if len(requirements) == 0 {
		return false, "provider has no DNS requirements"
	}
	report := c.service.Verify(ctx, c.domain(), requirements)
	for _, verdict := range report.Providers {
		if verdict.Provider == provider.ID || verdict.Provider == provider.Transport {
			return verdict.Status == maildns.Pass, verdict.Detail
		}
	}
	return false, "provider DNS authorization verdict missing"
}

func providerMailDNSRequirements(provider emaildelivery.Provider) []maildns.ProviderRequirement {
	spf, _ := provider.DNSRequirements["spf_mechanism"].(string)
	requirement := maildns.ProviderRequirement{Name: provider.ID, SPFMechanism: spf}
	if dkim, ok := provider.DNSRequirements["dkim"].([]any); ok {
		for _, raw := range dkim {
			entry, _ := raw.(map[string]any)
			selector, _ := entry["selector"].(string)
			typeName, _ := entry["type"].(string)
			expected, _ := entry["expected"].(string)
			requirement.DKIM = append(requirement.DKIM, maildns.DKIMRequirement{Selector: selector, Type: typeName, Expected: expected})
		}
	}
	return []maildns.ProviderRequirement{requirement}
}

func startEmailDeliveryWorker(db interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, service *EmailService, dnsService *maildns.Service, domain func() string) func() {
	ctx, cancel := context.WithCancel(context.Background())
	worker := emaildelivery.Worker{
		DB:     db,
		Router: emaildelivery.Router{Credentials: emailCredentialChecker{service: service}, DNS: emailDNSChecker{service: dnsService, domain: domain}, Quota: emailQuotaChecker{db: db}, Circuit: emailCircuitChecker{db: db}},
		Adapters: map[string]emaildelivery.Adapter{
			emaildelivery.ProviderMailgun:  emailDeliveryAdapter{service: service},
			emaildelivery.ProviderSendGrid: emailDeliveryAdapter{service: service},
		},
		AdapterFor: func(provider emaildelivery.Provider) emaildelivery.Adapter {
			return emailDeliveryAdapter{service: service}
		},
		Now: time.Now,
	}
	go func() {
		if err := worker.Run(ctx, 2*time.Second); err != nil && ctx.Err() == nil {
			logx.Error("email_delivery_worker_stopped", map[string]interface{}{"error": err.Error()})
		}
	}()
	return cancel
}
