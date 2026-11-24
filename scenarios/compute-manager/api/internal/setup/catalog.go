// Package setup describes provider onboarding and evaluates evidence-backed
// readiness. Catalogue entries contain public facts only; credentials are
// referred to by descriptor and resolved only by the probe adapter.
package setup

import "time"

type State string

const (
	NeedsConsoleSteps State = "needs_console_steps"
	CredentialNeeded  State = "credential_needed"
	Authenticated     State = "authenticated"
	AccountUsable     State = "account_usable"
	OfferAvailable    State = "offer_available"
	PaymentValid      State = "payment_valid"
	PurchaseEnabled   State = "purchase_enabled"
)

type ConsoleStep struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Instructions string `json:"instructions"`
	URL          string `json:"url"`
	Decision     string `json:"decision"`
}
type Probe struct {
	ID         string `json:"id"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Permission string `json:"permission"`
	ReadOnly   bool   `json:"readOnly"`
}
type Offer struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	CPU     string   `json:"cpu"`
	Memory  string   `json:"memory"`
	Regions []string `json:"regions"`
}
type Provider struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	ConsoleURL          string            `json:"consoleUrl"`
	CredentialLogicalID string            `json:"credentialLogicalId"`
	CredentialField     string            `json:"credentialField"`
	CredentialLabel     string            `json:"credentialLabel"`
	CredentialScopes    []string          `json:"credentialScopes"`
	ConsoleSteps        []ConsoleStep     `json:"consoleSteps"`
	Probes              []Probe           `json:"probes"`
	BillingFacts        map[string]string `json:"billingFacts"`
	Offers              []Offer           `json:"offers"`
}

type Evidence struct {
	ID            string    `json:"id"`
	State         State     `json:"state"`
	Verified      bool      `json:"verified"`
	Detail        string    `json:"detail"`
	CheckedAt     time.Time `json:"checked_at,omitempty"`
	SourceURL     string    `json:"source_url,omitempty"`
	SourceVersion string    `json:"source_version,omitempty"`
}
type Readiness struct {
	ProviderID string     `json:"provider_id"`
	Current    State      `json:"current"`
	Evidence   []Evidence `json:"evidence"`
}

var catalogue = []Provider{
	{
		ID: "hetzner", Name: "Hetzner Cloud", ConsoleURL: "https://console.hetzner.cloud/",
		CredentialLogicalID: "vrooli/compute-manager", CredentialField: "hetzner-api-token", CredentialLabel: "Hetzner Cloud API token for authorized lifecycle operations",
		CredentialScopes: []string{"Hetzner Cloud Read & Write permission (the declared lifecycle credential); setup verification sends GET requests only"},
		ConsoleSteps: []ConsoleStep{
			{ID: "account", Title: "Confirm your existing account", Instructions: "Use the Hetzner Cloud console you already have. No account creation is required.", URL: "https://console.hetzner.cloud/", Decision: "Does your existing Cloud account open successfully?"},
			{ID: "payment", Title: "Confirm payment method", Instructions: "If your account does not already have a valid payment method, add or update one on Hetzner's website. Compute Manager does not enter payment details.", URL: "https://console.hetzner.cloud/", Decision: "Is a payment method present and valid in the Hetzner account?"},
			{ID: "token", Title: "Create the Compute Manager API token", Instructions: "Create the project API token required by the declared Compute Manager lifecycle credential. Enter it only through Vrooli Onboarding's secure credential step. Setup verification itself sends GET requests only; a later server action remains separately gated.", URL: "https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/", Decision: "Have you created the Hetzner API token for Compute Manager?"},
		},
		Probes:       []Probe{{ID: "servers", Method: "GET", Path: "/v1/servers", Permission: "read servers", ReadOnly: true}, {ID: "server_types", Method: "GET", Path: "/v1/server_types", Permission: "read server types", ReadOnly: true}, {ID: "locations", Method: "GET", Path: "/v1/locations", Permission: "read locations", ReadOnly: true}},
		BillingFacts: map[string]string{"rounding": "hour", "minimum": "one hour", "stopped": "continues billing", "inbound": "not billed", "source": "https://docs.hetzner.com/cloud/billing/faq/"},
		Offers:       []Offer{{ID: "cx23", Label: "CX23", CPU: "2 shared vCPU", Memory: "4 GB", Regions: []string{"nbg1", "fsn1"}}},
	},
	{
		ID: "digitalocean", Name: "DigitalOcean", ConsoleURL: "https://cloud.digitalocean.com/",
		CredentialLogicalID: "vrooli/compute-manager", CredentialField: "digitalocean-api-token", CredentialLabel: "DigitalOcean scoped API token for Droplet lifecycle and invoice reads",
		CredentialScopes: []string{
			"droplet:create", "droplet:delete", "droplet:read", "regions:read", "sizes:read",
			"actions:read", "image:read", "snapshot:read", "vpc:read", "tag:create", "tag:read", "billing:read",
		},
		ConsoleSteps: []ConsoleStep{
			{ID: "account", Title: "Confirm provider account", Instructions: "Sign in to the DigitalOcean account.", URL: "https://cloud.digitalocean.com/", Decision: "Can you open the DigitalOcean account?"},
			{ID: "payment", Title: "Confirm payment method", Instructions: "If the account has no valid payment method, add it on the provider website only.", URL: "https://cloud.digitalocean.com/account/billing", Decision: "Is a payment method present and valid?"},
			{ID: "token", Title: "Create the Compute Manager API token", Instructions: "Create a scoped API token for the declared Compute Manager lifecycle credential. Setup verification itself sends GET requests only; a later server action remains separately gated. Enter it only through Vrooli Onboarding's secure credential step.", URL: "https://cloud.digitalocean.com/account/api/tokens", Decision: "Have you created the DigitalOcean API token for Compute Manager?"},
		},
		Probes: []Probe{
			{ID: "droplets", Method: "GET", Path: "/v2/droplets", Permission: "droplet:read", ReadOnly: true},
			{ID: "sizes", Method: "GET", Path: "/v2/sizes", Permission: "sizes:read", ReadOnly: true},
			{ID: "regions", Method: "GET", Path: "/v2/regions", Permission: "regions:read", ReadOnly: true},
			{ID: "invoices", Method: "GET", Path: "/v2/customers/my/invoices?per_page=1", Permission: "billing:read", ReadOnly: true},
		},
		BillingFacts: map[string]string{
			"rounding":                   "per second",
			"minimum":                    "60 seconds or US$0.01, whichever is higher",
			"bundled_cpu_monthly_cap":    "672 hours (28 days)",
			"stopped":                    "powered-off Droplets continue billing; billing ends when destroyed",
			"inbound":                    "not billed",
			"source":                     "https://docs.digitalocean.com/products/droplets/details/pricing/",
			"price_source":               "https://docs.digitalocean.com/products/droplets/details/pricing/",
			"sizes_source":               "GET https://api.digitalocean.com/v2/sizes; refreshed at verification time",
			"regions_source":             "GET https://api.digitalocean.com/v2/regions; refreshed at verification time",
			"price_observation_lifetime": "5 minutes; expired observations are refused",
			"v5_ipv4":                    "separate price component is dynamic and unsupported; quote remains unknown",
			"payment_step":               "operator confirms payment only on DigitalOcean's website; Compute Manager does not read card data",
			"payment_source":             "https://cloud.digitalocean.com/account/billing",
			"credential_scope_source":    "https://docs.digitalocean.com/reference/api/scopes/",
		},
		Offers: []Offer{{ID: "s-1vcpu-1gb", Label: "Basic Droplet (availability verified at probe time)", CPU: "1 vCPU", Memory: "1 GB"}},
	},
	{
		ID: "akash", Name: "Akash Network", ConsoleURL: "https://console.akash.network/",
		CredentialLogicalID: "vrooli/compute-manager", CredentialField: "akash-testnet-wallet",
		CredentialLabel:  "Akash sandbox-2 wallet signing credential (enter only through secure onboarding)",
		CredentialScopes: []string{"sandbox-2 signing capability; no chain probe or transaction is enabled by this setup step"},
		ConsoleSteps:     []ConsoleStep{{ID: "wallet", Title: "Enter the existing sandbox wallet securely", Instructions: "Use the secure Vrooli Onboarding credential step for the declared Compute Manager sandbox-2 wallet. Do not send the secret in this setup flow. This step does not create/import a wallet, verify a signer, or enable Akash effects.", URL: "/embedded/vrooli-onboarding/?target=local&step=credentials", Decision: "Have you entered the existing sandbox-2 wallet credential through secure Vrooli Onboarding?"}},
		BillingFacts:     map[string]string{"network": "sandbox-2", "effect_mode": "stub", "credential_reference": "credential-authority://vrooli/compute-manager/akash-testnet-wallet", "credential_authority_signer_status": "unverified", "credential_descriptor_status": "declared", "price_status": "no_valid_price", "blocked_reason": "A credential reference is only an identifier. Credential presence is checked from authority metadata; signer usability, price, and network access are not verified in this setup slice.", "setup_actions_enabled": "false", "read_only_probe_enabled": "false"},
	},
}

func Catalogue() []Provider { return append([]Provider(nil), catalogue...) }

// Evaluate advances only through a contiguous prefix of verified evidence.
// Any missing or failed check stops readiness at the previous state.
func Evaluate(providerID string, consoleDone, credentialPresent bool, evidence map[State]Evidence) Readiness {
	r := Readiness{ProviderID: providerID, Current: NeedsConsoleSteps}
	steps := []State{Authenticated, AccountUsable, OfferAvailable, PaymentValid, PurchaseEnabled}
	if !consoleDone {
		return r
	}
	r.Current = CredentialNeeded
	if !credentialPresent {
		return r
	}
	for _, state := range steps {
		e, ok := evidence[state]
		if !ok || !e.Verified {
			break
		}
		e.State = state
		r.Evidence = append(r.Evidence, e)
		r.Current = state
	}
	return r
}
