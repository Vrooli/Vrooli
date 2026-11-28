package credentialleak

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const lpbsIdentity = "vrooli/landing-page-business-suite"

var scopePattern = regexp.MustCompile(`scope=([^;\s]+)`)

type serviceManifest struct {
	Credentials struct {
		Descriptors []struct {
			Identity     string `json:"logical_id"`
			Field        string `json:"field"`
			Provisioning string `json:"provisioning"`
		} `json:"descriptors"`
		Consumers []struct {
			Identity string `json:"logical_id"`
			Field    string `json:"field"`
			Reason   string `json:"reason"`
		} `json:"consumers"`
	} `json:"credentials"`
}

// ManifestFamilies derives leak-scanner families from the current LPBS
// credential declarations and their explicit consumer scopes. It excludes
// known public configuration fields, while returning all other declared
// LPBS descriptors and selected external-identity consumers.
func ManifestFamilies(serviceJSON []byte) ([]Family, error) {
	var manifest serviceManifest
	if err := json.Unmarshal(serviceJSON, &manifest); err != nil {
		return nil, fmt.Errorf("decode LPBS credential metadata: %w", err)
	}
	consumers := make(map[string]struct {
		identity string
		scope    string
	})
	for _, consumer := range manifest.Credentials.Consumers {
		match := scopePattern.FindStringSubmatch(consumer.Reason)
		if len(match) == 2 {
			consumers[consumer.Identity+"\x00"+consumer.Field] = struct {
				identity string
				scope    string
			}{consumer.Identity, match[1]}
		}
	}
	public := map[string]bool{
		"public-base-url": true, "stripe-test-publishable-key": true,
		"stripe-live-publishable-key": true, "sendgrid-webhook-public-key": true,
		"apple-root-cert-pem": true, "apple-bundle-id": true,
		"apple-product-plans": true, "google-play-package-name": true,
		"google-play-product-id": true,
	}
	families := make([]Family, 0, len(manifest.Credentials.Descriptors)+8)
	seen := make(map[string]bool)
	for _, descriptor := range manifest.Credentials.Descriptors {
		if descriptor.Identity != lpbsIdentity || public[descriptor.Field] {
			continue
		}
		consumer, ok := consumers[descriptor.Identity+"\x00"+descriptor.Field]
		scope := "unscoped"
		if ok {
			scope = consumer.scope
		}
		families = append(families, Family{Name: descriptor.Field, Identity: descriptor.Identity, Field: descriptor.Field, Scope: scope})
		seen[descriptor.Identity+"\x00"+descriptor.Field] = true
	}
	for _, consumer := range manifest.Credentials.Consumers {
		key := consumer.Identity + "\x00" + consumer.Field
		if consumer.Identity == lpbsIdentity || seen[key] {
			continue
		}
		scope := "external-owner:scope-not-exposed"
		if declared, ok := consumers[key]; ok {
			scope = declared.scope
		}
		families = append(families, Family{Name: "external-" + normalizedFamilyName(consumer.Identity+"-"+consumer.Field), Identity: consumer.Identity, Field: consumer.Field, Scope: scope})
		seen[key] = true
	}
	// LPBS-issued service principals have no local descriptor, so pin their
	// exact consumer identities and scopes here. Provider keys owned by the
	// customer/provider resource are deliberately outside this LPBS inventory.
	for _, family := range []Family{
		{Name: "lpbs-money-ledger-principal", Identity: "scenario-authenticator/service-principal/landing-page-business-suite-money-ledger-writer/money-ledger", Field: "credential", Scope: "money-ledger:write"},
		{Name: "lpbs-money-ledger-rehearsal-principal", Identity: "scenario-authenticator/service-principal/landing-page-business-suite/money-ledger", Field: "credential", Scope: "money-ledger:write"},
		{Name: "lpbs-vrooli-bridge-principal", Identity: "scenario-authenticator/service-principal/landing-page-business-suite/landing-page-business-suite-bridge-dispatch/vrooli-bridge", Field: "credential", Scope: "vrooli-bridge:dispatch"},
		{Name: "lpbs-treasury-order-principal", Identity: "scenario-authenticator/service-principal/landing-page-business-suite/treasury-x402-order", Field: "credential", Scope: "treasury:x402-authorize"},
		{Name: "lpbs-treasury-receipt-principal", Identity: "scenario-authenticator/service-principal/landing-page-business-suite/treasury-x402-receipt", Field: "credential", Scope: "treasury:x402-receipt-read"},
		{Name: "lpbs-browser-evidence-principal", Identity: "scenario-authenticator/service-principal/landing-page-business-suite/landing-page-business-suite-browser-evidence-attester/agent-manager", Field: "credential", Scope: "agent-manager:browser-evidence:attest"},
	} {
		if !seen[family.Identity+"\x00"+family.Field] {
			families = append(families, family)
		}
	}
	return families, nil
}

// ReadManifestFamilies reads only the public declaration file; it never
// consults the credential authority or any runtime value source.
func ReadManifestFamilies(path string) ([]Family, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read LPBS credential declarations: %w", err)
	}
	return ManifestFamilies(data)
}

func ValidateFamilies(families []Family) error {
	seen := map[string]bool{}
	for _, family := range families {
		if family.Name == "" || family.Identity == "" || family.Field == "" || family.Scope == "" {
			return fmt.Errorf("credential family is missing identity, field, or scope metadata")
		}
		key := family.Identity + "\x00" + family.Field
		if seen[key] {
			return fmt.Errorf("duplicate credential family metadata for %s", family.Name)
		}
		seen[key] = true
	}
	if len(families) < 15 {
		return fmt.Errorf("credential family inventory unexpectedly small: %d", len(families))
	}
	return nil
}

func normalizedFamilyName(value string) string {
	var result strings.Builder
	for _, r := range strings.ToUpper(value) {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			result.WriteRune(r)
		} else if result.Len() > 0 && !strings.HasSuffix(result.String(), "_") {
			result.WriteByte('_')
		}
	}
	return strings.Trim(result.String(), "_")
}
