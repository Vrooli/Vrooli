package emaildelivery

import (
	"context"
	"database/sql"
	"encoding/json"
)

type ExecStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// SeedDefaults inserts the supported provider declarations once. Operators
// may then enable, disable, reorder, or extend the registry without a release.
func SeedDefaults(ctx context.Context, db ExecStore) error {
	defaults := []struct {
		id, transport, credential, dns, limits, purposes string
		rank                                             int
		recommended, enabled                             bool
	}{
		{ProviderMailgun, "mailgun_http", "vrooli/landing-page-business-suite:mailgun-api-key", `{"spf_mechanism":"include:mailgun.org","dkim":[{"selector":"k1","type":"TXT"}]}`, `{"day":100}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 10, true, true},
		{ProviderSendGrid, "sendgrid_http", "vrooli/landing-page-business-suite:sendgrid-api-key", `{"spf_mechanism":"include:sendgrid.net","dkim":[{"selector":"s1","type":"CNAME","expected":"sendgrid.net"},{"selector":"s2","type":"CNAME","expected":"sendgrid.net"}]}`, `{"day":100}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 20, false, false},
		// These catalog entries keep the surveyed alternatives visible to
		// operators without claiming that an account, credential, or
		// account-specific DKIM selector has been provisioned. They remain
		// disabled until their transport and DNS details are supplied in the
		// registry by the operator.
		{"resend", "https_api", "vrooli/landing-page-business-suite:resend-api-key", `{"spf_mechanism":"include:amazonses.com","dkim_dynamic":true}`, `{"day":100,"month":3000}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 30, true, false},
		{"brevo", "https_api", "vrooli/landing-page-business-suite:brevo-api-key", `{"spf_mechanism":"include:sendinblue.com","dkim_dynamic":true}`, `{"day":300,"month":9000}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 40, false, false},
		{"mailjet", "https_api", "vrooli/landing-page-business-suite:mailjet-api-key", `{"spf_mechanism":"include:spf.mailjet.com","dkim_dynamic":true}`, `{"day":200,"month":6000}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 50, false, false},
		{"mailtrap", "https_api", "vrooli/landing-page-business-suite:mailtrap-api-key", `{"spf_mechanism":"include:_spf.smtp.mailtrap.io","dkim_dynamic":true}`, `{"day":150,"month":4000}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 60, false, false},
		{"smtp2go", "smtp", "vrooli/landing-page-business-suite:smtp2go-credentials", `{"spf_mechanism":"include:spf.smtp2go.com","dkim_dynamic":true}`, `{"day":200,"month":1000}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 70, false, false},
		{"mailersend", "https_api", "vrooli/landing-page-business-suite:mailersend-api-key", `{"spf_mechanism":"include:mailersend.net","dkim_dynamic":true}`, `{"month":500}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 80, false, false},
		{"postmark", "https_api", "vrooli/landing-page-business-suite:postmark-server-token", `{"spf_mechanism":"include:spf.mtasv.net","dkim_dynamic":true}`, `{"month":100}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 90, false, false},
		{"ses", "https_api", "vrooli/landing-page-business-suite:ses-api-credentials", `{"spf_mechanism":"include:amazonses.com","dkim_dynamic":true}`, `{"month":0}`, `["signin","security_alert","passkey_notice","contact_form","magic_link"]`, 100, true, false},
	}
	for _, provider := range defaults {
		if _, err := db.ExecContext(ctx, `INSERT INTO email_providers (id, transport, credential_ref, dns_requirements, published_limits, supported_purposes, cost_rank, recommended, enabled) VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb,$7,$8,$9) ON CONFLICT (id) DO UPDATE SET transport=CASE WHEN email_providers.id='mailgun' AND email_providers.credential_ref IN ('vrooli/landing-page-business-suite:mailgun-smtp-password','vrooli/landing-page-business-suite:smtp-password') THEN EXCLUDED.transport ELSE email_providers.transport END, credential_ref=CASE WHEN email_providers.id='mailgun' AND email_providers.credential_ref IN ('vrooli/landing-page-business-suite:mailgun-smtp-password','vrooli/landing-page-business-suite:smtp-password') THEN EXCLUDED.credential_ref ELSE email_providers.credential_ref END, published_limits=CASE WHEN email_providers.published_limits='{}'::jsonb THEN EXCLUDED.published_limits ELSE email_providers.published_limits END`, provider.id, provider.transport, provider.credential, provider.dns, provider.limits, provider.purposes, provider.rank, provider.recommended, provider.enabled); err != nil {
			return err
		}
	}
	return nil
}

// LoadProviders is the only production read path for provider identity. A
// provider can be added, removed, or reordered by changing this registry row;
// transport code consumes the typed result and never embeds provider names in
// health or event logic.
func LoadProviders(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]Provider, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, transport, credential_ref, settings, dns_requirements, published_limits, supported_purposes, cost_rank, recommended, enabled FROM email_providers ORDER BY cost_rank, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var providers []Provider
	for rows.Next() {
		var p Provider
		var settings, dnsReq, limits, purposes []byte
		if err := rows.Scan(&p.ID, &p.Transport, &p.CredentialRef, &settings, &dnsReq, &limits, &purposes, &p.CostRank, &p.Recommended, &p.Enabled); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(settings, &p.Settings); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(dnsReq, &p.DNSRequirements); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(limits, &p.PublishedLimits); err != nil {
			return nil, err
		}
		var purposeList []string
		if err := json.Unmarshal(purposes, &purposeList); err != nil {
			return nil, err
		}
		p.SupportedPurposes = make(map[Purpose]bool, len(purposeList))
		for _, purpose := range purposeList {
			p.SupportedPurposes[Purpose(purpose)] = true
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}
