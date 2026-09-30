package emaildelivery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Purpose string

const (
	ProviderMailgun  = "mailgun"
	ProviderSendGrid = "sendgrid"
	ProviderSMTP     = "smtp"
)

const (
	PurposeSignIn    Purpose = "signin"
	PurposeSecurity  Purpose = "security_alert"
	PurposePasskey   Purpose = "passkey_notice"
	PurposeContact   Purpose = "contact_form"
	PurposeMagicLink Purpose = "magic_link"
)

type Message struct {
	DedupeKey      string
	TemplateRef    string
	TemplateData   json.RawMessage
	IdempotencyKey string
	Purpose        Purpose
	Recipient      string
	Sender         string
	Subject        string
	TextBody       string
	HTMLBody       string
	Priority       int
	ExpiresAt      *time.Time
}

type Provider struct {
	ID                string
	Transport         string
	CredentialRef     string
	Settings          map[string]any
	DNSRequirements   map[string]any
	PublishedLimits   map[string]int
	CostRank          int
	SupportedPurposes map[Purpose]bool
	Recommended       bool
	Enabled           bool
}

type ProviderState struct {
	CredentialVerified bool
	DNSAuthorized      bool
	QuotaAvailable     bool
	CredentialDetail   string
	DNSDetail          string
	QuotaDetail        string
}

type Candidate struct {
	Provider   Provider
	SkipReason string
}
type RouteDecision struct {
	Chosen     *Provider
	Candidates []Candidate
}

type CredentialChecker interface {
	Verify(context.Context, Provider) (bool, string)
}
type DNSChecker interface {
	Verify(context.Context, Provider) (bool, string)
}
type QuotaChecker interface {
	Available(context.Context, Provider) (bool, string)
}
type CircuitChecker interface {
	Available(context.Context, Provider) (bool, string)
}

type Router struct {
	Credentials CredentialChecker
	DNS         DNSChecker
	Quota       QuotaChecker
	Circuit     CircuitChecker
}
type verifier interface {
	Verify(context.Context, Provider) (bool, string)
}

func (r Router) Select(ctx context.Context, purpose Purpose, providers []Provider) RouteDecision {
	sort.SliceStable(providers, func(i, j int) bool {
		if providers[i].CostRank != providers[j].CostRank {
			return providers[i].CostRank < providers[j].CostRank
		}
		return providers[i].ID < providers[j].ID
	})
	decision := RouteDecision{}
	for _, provider := range providers {
		candidate := Candidate{Provider: provider}
		switch {
		case !provider.Enabled:
			candidate.SkipReason = "provider is disabled"
		case !provider.SupportedPurposes[purpose]:
			candidate.SkipReason = "purpose is not supported"
		case r.Credentials != nil && !r.mustVerify(ctx, provider, r.Credentials):
			candidate.SkipReason = "credential is not verified"
		case r.DNS != nil && !r.mustVerify(ctx, provider, r.DNS):
			candidate.SkipReason = "sending domain is not DNS-authorized"
		case r.Quota != nil && !r.mustAvailable(ctx, provider):
			candidate.SkipReason = "quota window is exhausted"
		case r.Circuit != nil && !r.mustAvailable(ctx, provider, r.Circuit):
			candidate.SkipReason = "provider circuit breaker is open"
		default:
			if decision.Chosen == nil {
				chosen := provider
				decision.Chosen = &chosen
			} else {
				candidate.SkipReason = "a cheaper eligible provider was selected"
			}
		}
		decision.Candidates = append(decision.Candidates, candidate)
	}
	return decision
}

func (r Router) mustVerify(ctx context.Context, provider Provider, checker verifier) bool {
	ok, _ := checker.Verify(ctx, provider)
	return ok
}
func (r Router) mustAvailable(ctx context.Context, provider Provider, checker ...QuotaChecker) bool {
	q := r.Quota
	if len(checker) > 0 {
		q = checker[0]
	}
	ok, _ := q.Available(ctx, provider)
	return ok
}

type OutboxStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Database is the minimal transactional surface used by the enqueue boundary.
// Keeping it here lets callers participate in their own transaction through
// EnqueueTx while standalone notifications get an atomic local transaction.
type Database interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// Mailer is the sole non-transactional entry point for email requests. The
// caller-facing methods never select a provider or perform network I/O.
type Mailer struct {
	DB Database
}

func (m Mailer) Enqueue(ctx context.Context, message Message) error {
	if m.DB == nil {
		return fmt.Errorf("email outbox database is unavailable")
	}
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin email enqueue transaction: %w", err)
	}
	defer tx.Rollback()
	if err := EnqueueTx(ctx, tx, message); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit email enqueue transaction: %w", err)
	}
	return nil
}

func EnqueueTx(ctx context.Context, tx *sql.Tx, message Message) error {
	if strings.TrimSpace(message.DedupeKey) == "" || strings.TrimSpace(message.Recipient) == "" || strings.TrimSpace(message.Sender) == "" || strings.TrimSpace(message.Subject) == "" || strings.TrimSpace(message.TextBody) == "" || strings.TrimSpace(message.TemplateRef) == "" || message.Purpose == "" {
		return fmt.Errorf("dedupe key, purpose, recipient, sender, subject, text body, and template reference are required")
	}
	switch message.Purpose {
	case PurposeSignIn, PurposeSecurity, PurposePasskey, PurposeContact, PurposeMagicLink:
	default:
		return fmt.Errorf("unsupported email purpose %q", message.Purpose)
	}
	if message.Priority == 0 {
		message.Priority = 50
	}
	if len(message.TemplateData) == 0 {
		message.TemplateData = json.RawMessage(`{}`)
	}
	if message.IdempotencyKey == "" {
		message.IdempotencyKey = message.DedupeKey
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO email_outbox (dedupe_key, purpose, recipient, sender, subject, text_body, html_body, template_ref, template_data, idempotency_key, priority, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (dedupe_key) DO NOTHING`, message.DedupeKey, message.Purpose, message.Recipient, message.Sender, message.Subject, message.TextBody, message.HTMLBody, message.TemplateRef, message.TemplateData, message.IdempotencyKey, message.Priority, message.ExpiresAt)
	return err
}

func DecodeProvider(row func(...any) error) (Provider, error) {
	var p Provider
	var settings, dnsReq, limits, purposes []byte
	var purposeList []string
	if err := row(&p.ID, &p.Transport, &p.CredentialRef, &settings, &dnsReq, &limits, &purposes, &p.CostRank, &p.Recommended, &p.Enabled); err != nil {
		return p, err
	}
	if err := json.Unmarshal(settings, &p.Settings); err != nil {
		return p, err
	}
	if err := json.Unmarshal(dnsReq, &p.DNSRequirements); err != nil {
		return p, err
	}
	if err := json.Unmarshal(limits, &p.PublishedLimits); err != nil {
		return p, err
	}
	if err := json.Unmarshal(purposes, &purposeList); err != nil {
		return p, err
	}
	p.SupportedPurposes = map[Purpose]bool{}
	for _, purpose := range purposeList {
		p.SupportedPurposes[Purpose(purpose)] = true
	}
	return p, nil
}
