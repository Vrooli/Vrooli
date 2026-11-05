package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/commerce"
)

// newReceiptValidators builds the platform receipt registry at the trusted
// composition root. The registry is always non-empty, but each validator is
// fail-closed until its server-only platform configuration is present.
// Receipt payloads never provide certificates, OAuth credentials, or identity
// bindings.
func newReceiptValidators(auth *administration.UserAuthService, plans *commerce.PlanService) commerce.ReceiptValidators {
	return commerce.ReceiptValidators{
		"apple": commerce.AppleSignedTransactionValidator{
			RootCertificate:  loadAppleRootCertificate(),
			ExpectedBundleID: strings.TrimSpace(resolveConfig("APPLE_BUNDLE_ID")),
			ProductPlans:     loadAppleProductPlans(plans),
			ResolveIdentity:  resolveStoreIdentity(auth),
		},
		"google": commerce.GooglePlayDeveloperValidator{
			PackageName:     strings.TrimSpace(resolveConfig("GOOGLE_PLAY_PACKAGE_NAME")),
			ProductID:       strings.TrimSpace(resolveConfig("GOOGLE_PLAY_PRODUCT_ID")),
			OAuthToken:      resolveGoogleOAuthToken,
			ResolveIdentity: resolveStoreIdentity(auth),
		},
	}
}

func loadAppleRootCertificate() *x509.Certificate {
	raw := strings.TrimSpace(resolveConfig("APPLE_ROOT_CERT_PEM"))
	if raw == "" {
		return nil
	}
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	return certificate
}

func loadAppleProductPlans(plans *commerce.PlanService) map[string]string {
	result := make(map[string]string)
	if raw := strings.TrimSpace(resolveConfig("APPLE_PRODUCT_PLANS")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &result); err == nil {
			return result
		}
	}
	// A plan catalog may carry the external store product in metadata. Read it
	// only at startup; the receipt validator remains independent of pricing
	// persistence after construction.
	if plans == nil {
		return result
	}
	overview, err := plans.GetPricingOverview()
	if err != nil || overview == nil {
		return result
	}
	plansToInspect := append(append([]*commerce.PlanOption{}, overview.Monthly...), overview.Yearly...)
	for _, plan := range plansToInspect {
		if plan == nil || plan.Metadata == nil {
			continue
		}
		value, ok := plan.Metadata["apple_product_id"]
		if !ok || value == nil || strings.TrimSpace(value.GetStringValue()) == "" {
			continue
		}
		result[value.GetStringValue()] = plan.PlanTier
	}
	return result
}

func resolveGoogleOAuthToken(ctx context.Context) (string, error) {
	token, err := resolveAuthorityCredential("GOOGLE_PLAY_OAUTH_TOKEN")
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return "", fmt.Errorf("google play OAuth token is not configured: %w", err)
		}
		return "", err
	}
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("google play OAuth token is empty")
	}
	return token, nil
}

func resolveStoreIdentity(auth *administration.UserAuthService) func(context.Context, string) (string, error) {
	return func(ctx context.Context, token string) (string, error) {
		if auth == nil || strings.TrimSpace(token) == "" {
			return "", commerce.ErrReceiptBound
		}
		user, err := auth.GetUserByID(ctx, strings.TrimSpace(token))
		if err != nil || user == nil || strings.TrimSpace(user.Email) == "" {
			return "", commerce.ErrReceiptBound
		}
		return user.Email, nil
	}
}
