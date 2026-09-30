package providerprobe

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
)

// doerFor serves one canned provider response and returns the client and URL
// that reach it.
func doerFor(t *testing.T, status int, body string) (HTTPDoer, string, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	return server.Client(), server.URL, server.Close
}

type failingDoer struct{}

func (failingDoer) Do(*http.Request) (*http.Response, error) { return nil, errors.New("network down") }

func TestVerifySendGridPassesOnlyWithMailSendScope(t *testing.T) {
	doer, endpoint, done := doerFor(t, http.StatusOK, `{"scopes":["mail.send","alerts.read"]}`)
	defer done()

	result := VerifySendGrid(context.Background(), doer, "SG.key", endpoint)

	if result.Status != StatusPass {
		t.Errorf("status = %s (%s), want pass", result.Status, result.Detail)
	}
}

// A key that authenticates but cannot send mail rejects every sign-in email,
// so it is a failure rather than a pass.
func TestVerifySendGridFailsWithoutMailSendScope(t *testing.T) {
	doer, endpoint, done := doerFor(t, http.StatusOK, `{"scopes":["alerts.read"]}`)
	defer done()

	result := VerifySendGrid(context.Background(), doer, "SG.key", endpoint)

	if result.Status != StatusFail {
		t.Errorf("status = %s, want fail", result.Status)
	}
}

func TestVerifySendGridFailsOnRejectedKey(t *testing.T) {
	doer, endpoint, done := doerFor(t, http.StatusUnauthorized, `{"errors":[{"message":"unauthorized"}]}`)
	defer done()

	result := VerifySendGrid(context.Background(), doer, "SG.revoked", endpoint)

	if result.Status != StatusFail || !result.Actionable() {
		t.Errorf("result = %+v, want an actionable failure", result)
	}
}

// An unreachable provider is not evidence of a bad credential.
func TestVerifySendGridUnreachableIsUnknownNotFail(t *testing.T) {
	result := VerifySendGrid(context.Background(), failingDoer{}, "SG.key", "http://sendgrid.invalid/v3/scopes")

	if result.Status != StatusUnknown || result.Actionable() {
		t.Errorf("result = %+v, want a non-actionable unknown", result)
	}
}

func TestVerifySendGridWithoutAKeyIsNotConfigured(t *testing.T) {
	if result := VerifySendGrid(context.Background(), nil, "  ", ""); result.Status != StatusNotConfigured {
		t.Errorf("status = %s, want not_configured", result.Status)
	}
}

// stubSMTP records the ceremony so the test can prove TLS came before auth
// and that no message was ever sent.
type stubSMTP struct {
	order      []string
	tlsErr     error
	authErr    error
	serverName string
}

func (s *stubSMTP) StartTLS(config *tls.Config) error {
	s.order = append(s.order, "starttls")
	s.serverName = config.ServerName
	return s.tlsErr
}

func (s *stubSMTP) Auth(smtp.Auth) error {
	s.order = append(s.order, "auth")
	return s.authErr
}
func (s *stubSMTP) Quit() error  { s.order = append(s.order, "quit"); return nil }
func (s *stubSMTP) Close() error { s.order = append(s.order, "close"); return nil }

func smtpTarget() SMTPTarget {
	return SMTPTarget{Host: "smtp.example.com", Port: 587, Username: "mailer", Password: "secret"}
}

func TestVerifySMTPAuthenticatesOverTLSAndSendsNothing(t *testing.T) {
	session := &stubSMTP{}

	result := VerifySMTP(context.Background(), func(context.Context, string) (SMTPSession, error) {
		return session, nil
	}, smtpTarget())

	if result.Status != StatusPass {
		t.Fatalf("status = %s (%s), want pass", result.Status, result.Detail)
	}
	if len(session.order) < 2 || session.order[0] != "starttls" || session.order[1] != "auth" {
		t.Errorf("ceremony = %v, want TLS before auth", session.order)
	}
	for _, step := range session.order {
		if step == "mail" || step == "data" || step == "rcpt" {
			t.Fatalf("a verification probe must not send a message: %v", session.order)
		}
	}
	if session.serverName != "smtp.example.com" {
		t.Errorf("server name = %q, want the relay host so TLS is actually verified", session.serverName)
	}
}

// This is the live failure: the stored password is rejected with 535.
func TestVerifySMTPFailsOnRejectedPassword(t *testing.T) {
	session := &stubSMTP{authErr: errors.New("535 Authentication failed")}

	result := VerifySMTP(context.Background(), func(context.Context, string) (SMTPSession, error) {
		return session, nil
	}, smtpTarget())

	if result.Status != StatusFail {
		t.Fatalf("status = %s, want fail", result.Status)
	}
	if !strings.Contains(result.Detail, "535") {
		t.Errorf("detail = %q, want the relay's own rejection", result.Detail)
	}
}

func TestVerifySMTPFailsWhenTLSIsRefused(t *testing.T) {
	session := &stubSMTP{tlsErr: errors.New("no TLS")}

	result := VerifySMTP(context.Background(), func(context.Context, string) (SMTPSession, error) {
		return session, nil
	}, smtpTarget())

	if result.Status != StatusFail {
		t.Errorf("status = %s, want fail: a password must never be sent in the clear", result.Status)
	}
}

func TestVerifySMTPUnreachableIsUnknown(t *testing.T) {
	result := VerifySMTP(context.Background(), func(context.Context, string) (SMTPSession, error) {
		return nil, errors.New("connection refused")
	}, smtpTarget())

	if result.Status != StatusUnknown || result.Actionable() {
		t.Errorf("result = %+v, want a non-actionable unknown", result)
	}
}

func TestVerifySMTPPartialConfigurationIsNotConfigured(t *testing.T) {
	target := smtpTarget()
	target.Password = ""

	if result := VerifySMTP(context.Background(), nil, target); result.Status != StatusNotConfigured {
		t.Errorf("status = %s, want not_configured", result.Status)
	}
}

// The mode mismatch is caught before any network call: a test key in a live
// deployment authenticates perfectly and takes no money.
func TestVerifyStripeFailsOnModeMismatchWithoutCallingStripe(t *testing.T) {
	result := VerifyStripe(context.Background(), failingDoer{}, "sk_test_abc", "live", "http://stripe.invalid/v1/account")

	if result.Status != StatusFail {
		t.Fatalf("status = %s, want fail", result.Status)
	}
	if !strings.Contains(result.Detail, "test") || !strings.Contains(result.Detail, "live") {
		t.Errorf("detail = %q, want both modes named", result.Detail)
	}
}

func TestVerifyStripePassesWhenTheKeyAuthenticatesInTheDeclaredMode(t *testing.T) {
	doer, endpoint, done := doerFor(t, http.StatusOK, `{"id":"acct_123"}`)
	defer done()

	result := VerifyStripe(context.Background(), doer, "sk_test_abc", "test", endpoint)

	if result.Status != StatusPass {
		t.Errorf("status = %s (%s), want pass", result.Status, result.Detail)
	}
}

func TestVerifyStripeFailsOnRejectedKey(t *testing.T) {
	doer, endpoint, done := doerFor(t, http.StatusUnauthorized, `{"error":{"message":"Invalid API Key"}}`)
	defer done()

	result := VerifyStripe(context.Background(), doer, "sk_live_revoked", "live", endpoint)

	if result.Status != StatusFail {
		t.Errorf("status = %s, want fail", result.Status)
	}
}

func TestVerifyStripeUnreachableIsUnknown(t *testing.T) {
	result := VerifyStripe(context.Background(), failingDoer{}, "sk_live_abc", "live", "http://stripe.invalid/v1/account")

	if result.Status != StatusUnknown || result.Actionable() {
		t.Errorf("result = %+v, want a non-actionable unknown", result)
	}
}

func TestResultDetailNeverCarriesTheCredential(t *testing.T) {
	doer, endpoint, done := doerFor(t, http.StatusUnauthorized, `{"errors":[{"message":"unauthorized"}]}`)
	defer done()

	for _, result := range []Result{
		VerifySendGrid(context.Background(), doer, "SG.supersecret", endpoint),
		VerifyStripe(context.Background(), doer, "sk_live_supersecret", "live", endpoint),
	} {
		if strings.Contains(result.Detail, "supersecret") {
			t.Errorf("provider %s leaked its credential into %q", result.Provider, result.Detail)
		}
	}
}
