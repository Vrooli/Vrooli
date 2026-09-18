package main

import (
	"testing"

	"landing-page-business-suite-api/internal/experimentation"
)

func stringPtr(value string) *string { return &value }

// The gap this closes: delivery health was attempt-driven, so a deployment
// with no usable provider read healthy whenever nobody was signing in. The
// configured state has to be observable on its own.
func TestSignInProvidersReportsConfiguredProviders(t *testing.T) {
	t.Run("no provider", func(t *testing.T) {
		service := NewEmailServiceWithOptions(EmailServiceOptions{})

		providers := service.SignInProviders()

		if providers.Any() || providers.SendGrid || providers.SMTP {
			t.Errorf("providers = %+v, want none", providers)
		}
	})

	t.Run("sendgrid only", func(t *testing.T) {
		service := NewEmailServiceWithOptions(EmailServiceOptions{
			SendGridConfig: &SendGridConfig{APIKey: "SG.key", FromEmail: "noreply@example.com"},
		})

		providers := service.SignInProviders()

		if !providers.SendGrid || providers.SMTP || !providers.Any() {
			t.Errorf("providers = %+v, want SendGrid only", providers)
		}
	})

	t.Run("smtp relay only", func(t *testing.T) {
		service := NewEmailServiceWithOptions(EmailServiceOptions{
			SMTPPasswordResolver: func() (string, error) { return "relay-secret", nil },
		})
		service.UseBrandingSource(func() *experimentation.SiteBranding {
			return &experimentation.SiteBranding{
				SMTPHost:     stringPtr("smtp.example.com"),
				SMTPUsername: stringPtr("mailer"),
			}
		})

		providers := service.SignInProviders()

		if providers.SendGrid || !providers.SMTP || !providers.Any() {
			t.Errorf("providers = %+v, want the SMTP relay only", providers)
		}
	})

	// A relay with no password cannot deliver, so it must not count as a
	// provider — this is the case that made "configured" meaningless.
	t.Run("smtp relay without a password", func(t *testing.T) {
		service := NewEmailServiceWithOptions(EmailServiceOptions{
			SMTPPasswordResolver: func() (string, error) { return "", nil },
		})
		service.UseBrandingSource(func() *experimentation.SiteBranding {
			return &experimentation.SiteBranding{
				SMTPHost:     stringPtr("smtp.example.com"),
				SMTPUsername: stringPtr("mailer"),
			}
		})

		if providers := service.SignInProviders(); providers.Any() {
			t.Errorf("providers = %+v, want none", providers)
		}
	})
}
