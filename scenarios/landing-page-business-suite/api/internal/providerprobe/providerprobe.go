// Package providerprobe verifies that a stored provider credential actually
// authenticates against its provider.
//
// Presence is not usability. A SendGrid key can be present and revoked, an
// SMTP password present and rotated, a Stripe key present and belonging to
// the other mode. Every one of those reads "configured" to a presence check
// and fails the moment a customer needs it — which is how a live site can
// carry an SMTP password its provider rejects with 535 and report clean.
//
// Every probe here is read-only and side-effect free: no mail is sent, no
// payment object is created. A probe authenticates and hangs up.
package providerprobe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Status is a probe outcome.
type Status string

const (
	ProviderSendGrid = "sendgrid"
	ProviderSMTP     = "smtp"
	ProviderStripe   = "stripe"
)

const (
	// StatusPass means the credential authenticated.
	StatusPass Status = "pass"
	// StatusFail means the provider rejected the credential, or the
	// configuration cannot work. This is actionable and alertable.
	StatusFail Status = "fail"
	// StatusNotConfigured means there is nothing to verify. It is not a
	// defect: a deployment may legitimately not use a provider.
	StatusNotConfigured Status = "not_configured"
	// StatusUnknown means the probe could not reach a verdict — the provider
	// was unreachable, or the attempt timed out. It must never be treated as
	// a pass, and must never raise an alert on its own: an unreachable
	// provider is not evidence of a bad credential.
	StatusUnknown Status = "unknown"
)

// Result is one provider's verdict.
type Result struct {
	// Provider is a stable name: "sendgrid", "smtp", "stripe".
	Provider string `json:"provider"`
	// Capability is the customer-facing capability this credential powers.
	Capability string `json:"capability,omitempty"`
	Status     Status `json:"status"`
	// Detail is operator-readable, and never contains the credential.
	Detail    string    `json:"detail,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// Actionable reports whether this result should degrade health and alert.
func (r Result) Actionable() bool { return r.Status == StatusFail }

const (
	defaultSendGridScopesURL = "https://api.sendgrid.com/v3/scopes"
	defaultStripeAccountURL  = "https://api.stripe.com/v1/account"
	defaultProbeTimeout      = 10 * time.Second
	// mailSendScope is the only SendGrid scope sign-in delivery needs.
	mailSendScope = "mail.send"
)

// HTTPDoer is the HTTP seam; tests replace it.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func now() time.Time { return time.Now().UTC() }

// VerifySendGrid authenticates the API key against SendGrid's scopes
// endpoint, which is read-only, and confirms the key carries mail.send. A key
// that authenticates but cannot send mail is a failure, not a pass: it will
// reject every sign-in email.
func VerifySendGrid(ctx context.Context, doer HTTPDoer, apiKey, endpoint string) Result {
	result := Result{Provider: ProviderSendGrid, Capability: "customer sign-in email", CheckedAt: now()}
	if strings.TrimSpace(apiKey) == "" {
		result.Status = StatusNotConfigured
		result.Detail = "no SendGrid API key is configured"
		return result
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = defaultSendGridScopesURL
	}
	body, status, err := getJSON(ctx, doer, endpoint, map[string]string{"Authorization": "Bearer " + apiKey})
	switch {
	case err != nil:
		result.Status = StatusUnknown
		result.Detail = "SendGrid could not be reached to verify the key"
		return result
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		result.Status = StatusFail
		result.Detail = fmt.Sprintf("SendGrid rejected the API key (HTTP %d)", status)
		return result
	case status < 200 || status >= 300:
		result.Status = StatusUnknown
		result.Detail = fmt.Sprintf("SendGrid answered HTTP %d when asked to verify the key", status)
		return result
	}
	if !strings.Contains(string(body), mailSendScope) {
		result.Status = StatusFail
		result.Detail = "the SendGrid key authenticates but does not carry the mail.send scope"
		return result
	}
	result.Status = StatusPass
	result.Detail = "the SendGrid key authenticates and can send mail"
	return result
}

// SMTPTarget is the relay configuration to verify.
type SMTPTarget struct {
	Host     string
	Port     int
	Username string
	Password string
}

// Configured reports whether there is anything to verify.
func (t SMTPTarget) Configured() bool {
	return strings.TrimSpace(t.Host) != "" && strings.TrimSpace(t.Username) != "" && strings.TrimSpace(t.Password) != ""
}

// SMTPSession is the subset of *smtp.Client this package uses.
type SMTPSession interface {
	StartTLS(*tls.Config) error
	Auth(smtp.Auth) error
	Quit() error
	Close() error
}

// SMTPDialer opens a session to addr. Tests replace it.
type SMTPDialer func(ctx context.Context, addr string) (SMTPSession, error)

// DialSMTP is the production dialer.
func DialSMTP(ctx context.Context, addr string) (SMTPSession, error) {
	dialer := &smtpDialer{timeout: defaultProbeTimeout}
	return dialer.dial(ctx, addr)
}

// VerifySMTP authenticates against the relay and hangs up. It sends no
// message: the login is the whole test, and 535 is exactly what a rotated or
// wrong password returns.
//
// TLS is required before authentication, matching the delivery path: a probe
// must not prove a credential works over a channel the sender would refuse.
func VerifySMTP(ctx context.Context, dial SMTPDialer, target SMTPTarget) Result {
	result := Result{Provider: ProviderSMTP, Capability: "customer sign-in email (legacy SMTP compatibility)", CheckedAt: now()}
	if !target.Configured() {
		result.Status = StatusNotConfigured
		result.Detail = "the site SMTP relay is not fully configured"
		return result
	}
	if dial == nil {
		dial = DialSMTP
	}
	port := target.Port
	if port == 0 {
		port = 587
	}
	session, err := dial(ctx, fmt.Sprintf("%s:%d", strings.TrimSpace(target.Host), port))
	if err != nil {
		result.Status = StatusUnknown
		result.Detail = "the SMTP relay could not be reached to verify the password"
		return result
	}
	defer func() { _ = session.Close() }()
	if err := session.StartTLS(&tls.Config{ServerName: strings.TrimSpace(target.Host), MinVersion: tls.VersionTLS12}); err != nil {
		result.Status = StatusFail
		result.Detail = "the SMTP relay would not start TLS, so the password cannot be sent safely: " + err.Error()
		return result
	}
	if err := session.Auth(smtp.PlainAuth("", target.Username, target.Password, strings.TrimSpace(target.Host))); err != nil {
		result.Status = StatusFail
		result.Detail = "the SMTP relay rejected the stored password: " + err.Error()
		return result
	}
	_ = session.Quit()
	result.Status = StatusPass
	result.Detail = "the SMTP relay accepted the stored password"
	return result
}

// VerifyStripe authenticates the secret key against Stripe's account
// endpoint, which is read-only, and reports whether the key's mode matches
// the deployment's declared mode.
//
// The mode check is the point: a test-mode key in a live deployment
// authenticates perfectly and quietly takes no money.
func VerifyStripe(ctx context.Context, doer HTTPDoer, secretKey, declaredMode, endpoint string) Result {
	result := Result{Provider: ProviderStripe, Capability: "card payments", CheckedAt: now()}
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		result.Status = StatusNotConfigured
		result.Detail = "no Stripe secret key is configured"
		return result
	}
	if keyMode := stripeKeyMode(secretKey); keyMode != "" && declaredMode != "" && keyMode != strings.ToLower(strings.TrimSpace(declaredMode)) {
		result.Status = StatusFail
		result.Detail = fmt.Sprintf("the configured Stripe key is a %s-mode key but this deployment declares %s mode", keyMode, strings.ToLower(strings.TrimSpace(declaredMode)))
		return result
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = defaultStripeAccountURL
	}
	_, status, err := getJSON(ctx, doer, endpoint, map[string]string{"Authorization": "Bearer " + secretKey})
	switch {
	case err != nil:
		result.Status = StatusUnknown
		result.Detail = "Stripe could not be reached to verify the key"
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		result.Status = StatusFail
		result.Detail = fmt.Sprintf("Stripe rejected the secret key (HTTP %d)", status)
	case status < 200 || status >= 300:
		result.Status = StatusUnknown
		result.Detail = fmt.Sprintf("Stripe answered HTTP %d when asked to verify the key", status)
	default:
		result.Status = StatusPass
		result.Detail = "the Stripe secret key authenticates in " + strings.ToLower(strings.TrimSpace(declaredMode)) + " mode"
	}
	return result
}

// stripeKeyMode reads the mode a Stripe key declares in its own prefix.
func stripeKeyMode(key string) string {
	switch {
	case strings.HasPrefix(key, "sk_live_"), strings.HasPrefix(key, "rk_live_"):
		return "live"
	case strings.HasPrefix(key, "sk_test_"), strings.HasPrefix(key, "rk_test_"):
		return "test"
	default:
		return ""
	}
}

func getJSON(ctx context.Context, doer HTTPDoer, url string, headers map[string]string) ([]byte, int, error) {
	if doer == nil {
		doer = &http.Client{Timeout: defaultProbeTimeout}
	}
	ctx, cancel := context.WithTimeout(ctx, defaultProbeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := doer.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}
	return body, resp.StatusCode, nil
}
