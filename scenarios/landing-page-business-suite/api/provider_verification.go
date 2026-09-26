package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/health"

	"landing-page-business-suite-api/internal/emaildelivery"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/logx"
	"landing-page-business-suite-api/internal/opsalert"
	"landing-page-business-suite-api/internal/providerprobe"
)

// providerVerificationInterval is how often stored provider credentials are
// re-verified. Credentials break between deploys — a key is revoked, a
// password rotated — so the only signal that survives is a periodic one.
const providerVerificationInterval = time.Hour

// providerVerificationStartupDelay lets the process finish starting before
// the first probe, so startup is never blocked on a third party.
const providerVerificationStartupDelay = 30 * time.Second

// providerAlertWindow buckets alerts so a persistently broken credential
// raises one alert per window instead of one per probe.
const providerAlertWindow = 6 * time.Hour

// providerVerificationCache holds the most recent verdicts. Health reads the
// cache and never probes inline: a health request must not depend on a third
// party's latency, and a probe must not run once per poller.
type providerVerificationCache struct {
	mu      sync.RWMutex
	results map[string]providerprobe.Result
}

func newProviderVerificationCache() *providerVerificationCache {
	return &providerVerificationCache{results: map[string]providerprobe.Result{}}
}

func (c *providerVerificationCache) store(results ...providerprobe.Result) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, result := range results {
		c.results[result.Provider] = result
	}
}

func (c *providerVerificationCache) get(provider string) (providerprobe.Result, bool) {
	if c == nil {
		return providerprobe.Result{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, ok := c.results[provider]
	return result, ok
}

func (c *providerVerificationCache) snapshot() []providerprobe.Result {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]providerprobe.Result, 0, len(c.results))
	for _, result := range c.results {
		out = append(out, result)
	}
	return out
}

// verifySignInProviders authenticates the sign-in mail credentials. It sends
// no mail: SendGrid is asked for the key's scopes and the relay is asked to
// accept a login.
func (s *Server) verifySignInProviders(ctx context.Context) []providerprobe.Result {
	if s.emailService == nil || s.routedDB == nil {
		return nil
	}
	target := providerprobe.SMTPTarget{}
	if branding := s.emailService.currentBranding(); branding != nil {
		if config := s.emailService.extractSMTPConfig(branding); config != nil {
			target = providerprobe.SMTPTarget{Host: config.Host, Port: config.Port, Username: config.Username, Password: config.Password}
		}
	}
	providers, err := emaildelivery.LoadProviders(ctx, s.routedDB)
	if err != nil {
		return nil
	}
	results := make([]providerprobe.Result, 0, len(providers))
	for _, provider := range providers {
		var result providerprobe.Result
		switch provider.Transport {
		case "mailgun_http":
			status, detail := s.emailService.verifyMailgunAPI(ctx)
			result = providerprobe.Result{Provider: provider.ID, Capability: "customer sign-in email", Detail: detail, CheckedAt: time.Now().UTC()}
			switch {
			case status >= 200 && status < 300:
				result.Status = providerprobe.StatusPass
			case status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusBadRequest:
				result.Status = providerprobe.StatusFail
			default:
				result.Status = providerprobe.StatusUnknown
			}
		case "sendgrid_http":
			result = providerprobe.VerifySendGrid(ctx, nil, resolveSecret("SENDGRID_API_KEY"), envx.Get("LPBS_SENDGRID_SCOPES_URL"))
		case "mailgun_smtp", "smtp":
			result = providerprobe.VerifySMTP(ctx, nil, target)
		default:
			continue
		}
		// The registry, not the transport probe, owns provider identity.
		result.Provider = provider.ID
		results = append(results, result)
	}
	return results
}

// verifyPaymentsProvider authenticates the active Stripe secret key and
// checks its mode against the declared one. It performs a read-only account
// lookup and creates nothing.
func (s *Server) verifyPaymentsProvider(ctx context.Context) providerprobe.Result {
	mode := strings.ToLower(strings.TrimSpace(envx.Get("STRIPE_MODE")))
	if s.paymentSettings == nil || mode == "" {
		return providerprobe.Result{Provider: providerprobe.ProviderStripe, Capability: "card payments", Status: providerprobe.StatusNotConfigured, Detail: "this deployment declares no Stripe mode", CheckedAt: time.Now().UTC()}
	}
	settings, err := s.paymentSettings.GetStripeSettings(ctx)
	if err != nil || settings == nil {
		return providerprobe.Result{Provider: providerprobe.ProviderStripe, Capability: "card payments", Status: providerprobe.StatusUnknown, Detail: "the active Stripe credentials could not be read", CheckedAt: time.Now().UTC()}
	}
	return providerprobe.VerifyStripe(ctx, nil, settings.GetSecretKey(), mode, envx.Get("LPBS_STRIPE_ACCOUNT_URL"))
}

// runProviderVerification probes every provider once, caches the verdicts and
// alerts on actionable failures.
func (s *Server) runProviderVerification(ctx context.Context, transport *opsalert.Transport) {
	results := append(s.verifySignInProviders(ctx), s.verifyPaymentsProvider(ctx))
	s.providerVerification.store(results...)
	for _, result := range results {
		if !result.Actionable() {
			continue
		}
		logx.Info("provider_credential_unusable", map[string]interface{}{
			"level":      "warn",
			"provider":   result.Provider,
			"capability": result.Capability,
			"detail":     result.Detail,
		})
		if err := s.dispatchProviderAlert(ctx, transport, result); err != nil {
			logx.Info("provider_credential_alert_failed", map[string]interface{}{"level": "warn", "provider": result.Provider, "error": err.Error()})
		}
	}
}

// dispatchProviderAlert records the failure and sends it to the operator
// webhook, reusing the auth alert log so a persistent failure alerts once per
// window rather than once per probe.
func (s *Server) dispatchProviderAlert(ctx context.Context, transport *opsalert.Transport, result providerprobe.Result) error {
	if s.db == nil {
		return nil
	}
	alertType := "provider_credential_unusable:" + result.Provider
	window := time.Now().UTC().Truncate(providerAlertWindow)
	body, err := json.Marshal(map[string]any{
		"type":     alertType,
		"scenario": "landing-page-business-suite",
		"details": map[string]any{
			"provider":   result.Provider,
			"capability": result.Capability,
			"detail":     result.Detail,
			"checked_at": result.CheckedAt,
		},
	})
	if err != nil {
		return err
	}
	inserted, err := s.db.ExecContext(ctx,
		`INSERT INTO auth_alert_log (alert_type, window_start, details, dispatch_status, created_at)
		 VALUES ($1,$3,$2,'pending',NOW()) ON CONFLICT (alert_type, window_start) DO NOTHING`,
		alertType, body, window)
	if err != nil {
		return err
	}
	if rows, _ := inserted.RowsAffected(); rows == 0 {
		// Already alerted in this window.
		return nil
	}
	url, enabled := s.operatorAlertWebhook(ctx)
	if !enabled || url == "" {
		_, _ = s.db.ExecContext(context.WithoutCancel(ctx),
			`UPDATE auth_alert_log SET dispatch_status='skipped', dispatch_error='no operator webhook is configured' WHERE alert_type=$1 AND window_start=$2`,
			alertType, window)
		return nil
	}
	if transport == nil {
		transport = opsalert.New()
	}
	if sendErr := transport.Send(ctx, url, body); sendErr != nil {
		_, _ = s.db.ExecContext(context.WithoutCancel(ctx),
			`UPDATE auth_alert_log SET dispatch_status='failed', dispatch_error=$1 WHERE alert_type=$2 AND window_start=$3`,
			sendErr.Error(), alertType, window)
		return sendErr
	}
	_, _ = s.db.ExecContext(context.WithoutCancel(ctx),
		`UPDATE auth_alert_log SET dispatch_status='sent', dispatch_error=NULL WHERE alert_type=$1 AND window_start=$2`,
		alertType, window)
	return nil
}

// operatorAlertWebhook reads the operator's alert destination.
func (s *Server) operatorAlertWebhook(ctx context.Context) (string, bool) {
	var url string
	var enabled bool
	if s.db == nil {
		return "", false
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(anomaly_webhook_url,''), COALESCE(anomaly_webhook_enabled,FALSE) FROM payment_settings WHERE id = 1`).
		Scan(&url, &enabled); err != nil {
		return "", false
	}
	return strings.TrimSpace(url), enabled
}

// startProviderVerification runs the periodic verification loop and returns a
// cancel function.
func (s *Server) startProviderVerification() func() {
	if s.providerVerification == nil {
		s.providerVerification = newProviderVerificationCache()
	}
	transport := opsalert.New()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		first := time.NewTimer(providerVerificationStartupDelay)
		defer first.Stop()
		select {
		case <-ctx.Done():
			return
		case <-first.C:
		}
		s.runProviderVerification(ctx, transport)
		ticker := time.NewTicker(providerVerificationInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runProviderVerification(ctx, transport)
			}
		}
	}()
	return cancel
}

// providerCredentialCheck publishes the cached verdict for one provider.
//
// Only an outright rejection degrades health. A credential that is not
// configured is a declaration, not a defect, and an unreachable provider is
// not evidence of a bad credential; both stay silent here so the signal that
// remains is always actionable.
func (s *Server) providerCredentialCheck(name, provider string) health.Checker {
	return health.Func(name, func(context.Context) error {
		result, ok := s.providerVerification.get(provider)
		if !ok || !result.Actionable() {
			return nil
		}
		return fmt.Errorf("%s: %s", result.Capability, result.Detail)
	})
}

// signInProviderCredentialCheck publishes the cached verdicts for the
// enabled provider registry rather than a fixed pair of provider names. A
// provider may be added or removed by configuration, so health must follow
// the same registry that the router and worker use.
func (s *Server) signInProviderCredentialCheck() health.Checker {
	return health.Func("sign_in_provider_credentials", func(ctx context.Context) error {
		if s == nil || s.routedDB == nil {
			return nil
		}
		providers, err := emaildelivery.LoadProviders(ctx, s.routedDB)
		if err != nil {
			return fmt.Errorf("read email provider registry: %w", err)
		}
		for _, provider := range providers {
			if !provider.Enabled || !provider.SupportedPurposes[emaildelivery.PurposeSignIn] {
				continue
			}
			result, ok := s.providerVerification.get(provider.ID)
			if ok && result.Actionable() {
				return fmt.Errorf("%s: %s", result.Capability, result.Detail)
			}
		}
		return nil
	})
}

// providerVerificationReport serves the cached verdicts to the admin portal.
func (s *Server) providerVerificationReport(w http.ResponseWriter, r *http.Request) {
	results := s.providerVerification.snapshot()
	if len(results) == 0 {
		writeJSON(w, map[string]any{"results": []providerprobe.Result{}, "message": "no provider verification has run yet"})
		return
	}
	writeJSON(w, map[string]any{"results": results})
}
