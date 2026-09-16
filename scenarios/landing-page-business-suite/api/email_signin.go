package main

import (
	"errors"
	"fmt"
	"html"
	"mime"
	"net/smtp"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
)

// UseBrandingSource lets sign-in mail use the site's name and its SMTP relay.
func (s *EmailService) UseBrandingSource(source func() *experimentation.SiteBranding) {
	s.brandingSource = source
}

// SendSignIn delivers the sign-in link and code. SendGrid is the primary
// provider; the site's SMTP relay is a fallback so one provider outage does
// not lock every customer out.
func (s *EmailService) SendSignIn(message administration.SignInMessage) error {
	branding := s.currentBranding()
	appName := strings.TrimSpace(message.AppName)
	if branding != nil && strings.TrimSpace(branding.SiteName) != "" && (appName == "" || appName == "App") {
		appName = strings.TrimSpace(branding.SiteName)
	}
	if appName == "" {
		appName = "your account"
	}
	message.AppName = appName
	subject := fmt.Sprintf("%s is your %s sign-in code", message.Code, appName)
	textBody := buildSignInText(message)
	htmlBody := buildSignInHTML(message)

	var failures []string
	if s.IsSendGridConfigured() {
		if err := s.sendViaSendGrid(message.To, subject, textBody, htmlBody); err == nil {
			return nil
		} else {
			failures = append(failures, "sendgrid: "+err.Error())
		}
	}
	if branding != nil {
		config := s.extractSMTPConfig(branding)
		if config.IsConfigured() {
			if err := s.sendSMTPMultipart(config, message.To, subject, textBody, htmlBody); err == nil {
				if len(failures) > 0 {
					logx.Info("sign_in_delivery_fell_back_to_smtp", map[string]interface{}{"level": "warn", "primary_error": strings.Join(failures, "; ")})
				}
				return nil
			} else {
				failures = append(failures, "smtp: "+err.Error())
			}
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	if s.allowUnconfiguredDelivery {
		logx.Info("magic_link_delivery_disabled", map[string]interface{}{
			"level":   "warn",
			"to":      message.To,
			"message": "sign-in delivery is disabled in the non-production composition; no provider is configured",
		})
		return nil
	}
	return errors.New("no sign-in email provider is configured (SendGrid or site SMTP)")
}

func (s *EmailService) currentBranding() *experimentation.SiteBranding {
	if s.brandingSource == nil {
		return nil
	}
	return s.brandingSource()
}

func (s *EmailService) sendSMTPMultipart(config *SMTPConfig, to, subject, textBody, htmlBody string) error {
	boundary := fmt.Sprintf("lpbs-%d", time.Now().UnixNano())
	var body strings.Builder
	fmt.Fprintf(&body, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n", config.From, to, mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&body, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
	fmt.Fprintf(&body, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, textBody)
	fmt.Fprintf(&body, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n--%s--\r\n", boundary, htmlBody, boundary)
	sender := s.smtpSender
	if sender == nil {
		sender = smtp.SendMail
	}
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
	return sender(fmt.Sprintf("%s:%d", config.Host, config.Port), auth, config.From, []string{to}, []byte(body.String()))
}

func signInExpiryPhrase(ttl time.Duration) string {
	minutes := int(ttl.Round(time.Minute) / time.Minute)
	if minutes <= 0 {
		minutes = 15
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func buildSignInText(message administration.SignInMessage) string {
	return fmt.Sprintf(`Sign in to %s

Your sign-in code: %s

Or open this link on the device where you want to sign in:
%s

The code and link expire in %s and work once.

If you didn't ask to sign in, you can ignore this email. Nobody can sign in without this code or link.
`, message.AppName, message.Code, message.Link, signInExpiryPhrase(message.ExpiresIn))
}

func buildSignInHTML(message administration.SignInMessage) string {
	app := html.EscapeString(message.AppName)
	link := html.EscapeString(message.Link)
	code := html.EscapeString(message.Code)
	spaced := code
	if len(code) == 6 {
		spaced = code[:3] + "&thinsp;" + code[3:]
	}
	expiry := signInExpiryPhrase(message.ExpiresIn)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>Sign in to %[1]s</title>
</head>
<body style="margin:0;padding:0;background:#eef2f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0b1728;">
<div style="display:none;max-height:0;overflow:hidden;">Your code is %[3]s. It expires in %[5]s.</div>
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#eef2f7;padding:40px 16px;">
<tr><td align="center">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:480px;background:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 1px 3px rgba(11,23,40,.08);">
<tr><td style="background:#0b1728;background-image:linear-gradient(160deg,#15243c,#0b1728);padding:28px 36px;">
<p style="margin:0;font-size:13px;letter-spacing:.14em;text-transform:uppercase;color:#22d3ee;font-weight:600;">%[1]s</p>
<p style="margin:8px 0 0;font-size:22px;line-height:1.3;color:#f1f5f9;font-weight:600;">Your sign-in code</p>
</td></tr>
<tr><td style="padding:32px 36px 8px;">
<p style="margin:0 0 20px;font-size:15px;line-height:1.6;color:#334155;">Enter this code in the window where you asked to sign in:</p>
<p style="margin:0 0 24px;padding:18px 0;border-radius:12px;background:#f1f5f9;border:1px solid #e2e8f0;text-align:center;font-family:'SFMono-Regular',Menlo,Consolas,monospace;font-size:34px;letter-spacing:.18em;font-weight:700;color:#0b1728;">%[4]s</p>
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#334155;">Or sign in on this device:</p>
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="border-radius:999px;background:#0891b2;">
<a href="%[2]s" style="display:inline-block;padding:13px 28px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:999px;">Sign in to %[1]s</a>
</td></tr></table>
<p style="margin:24px 0 0;font-size:13px;line-height:1.6;color:#64748b;">The code and link expire in %[5]s and work once.</p>
</td></tr>
<tr><td style="padding:24px 36px 32px;">
<hr style="border:none;border-top:1px solid #e2e8f0;margin:0 0 20px;">
<p style="margin:0;font-size:12px;line-height:1.6;color:#94a3b8;">Didn't ask to sign in? You can ignore this email. Nobody can sign in without this code or link.</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`, app, link, code, spaced, expiry)
}
