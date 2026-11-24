package setup

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/setup"
	_ "modernc.org/sqlite"
)

func TestCatalogueHasSecretFreeReadOnlyProviderEntries(t *testing.T) {
	entries := Catalogue()
	if len(entries) < 2 {
		t.Fatalf("catalogue entries = %d, want Hetzner and DigitalOcean", len(entries))
	}
	for _, entry := range entries {
		if entry.CredentialField == "" || entry.CredentialLogicalID == "" || (entry.ID != "akash" && len(entry.Probes) == 0) {
			t.Fatalf("incomplete provider entry: %#v", entry)
		}
		for _, probe := range entry.Probes {
			if probe.Method != "GET" || !probe.ReadOnly {
				t.Fatalf("unsafe probe in %s: %#v", entry.ID, probe)
			}
		}
	}
}

func TestAkashCatalogueDeclaresWalletWithoutChainProbeOrEffect(t *testing.T) {
	item := findProvider("akash")
	if item == nil || item.CredentialLogicalID != "vrooli/compute-manager" || item.CredentialField != "akash-testnet-wallet" || item.BillingFacts["network"] != "sandbox-2" || item.BillingFacts["effect_mode"] != "stub" || len(item.Probes) != 0 || len(item.ConsoleSteps) != 1 || item.ConsoleSteps[0].ID != "wallet" || item.ConsoleSteps[0].URL != "/embedded/vrooli-onboarding/?target=local&step=credentials" {
		t.Fatalf("unsafe or incomplete Akash phone setup descriptor: %+v", item)
	}
}

func TestDigitalOceanCatalogueDeclaresExactLifecycleAndBillingScopesAndSourcedFacts(t *testing.T) {
	do := findProvider("digitalocean")
	if do == nil {
		t.Fatal("DigitalOcean missing from the shared provider catalogue")
	}
	wantScopes := []string{"droplet:create", "droplet:delete", "droplet:read", "regions:read", "sizes:read", "actions:read", "image:read", "snapshot:read", "vpc:read", "tag:create", "tag:read", "billing:read"}
	if len(do.CredentialScopes) != len(wantScopes) {
		t.Fatalf("DigitalOcean scopes = %#v, want exact set %#v", do.CredentialScopes, wantScopes)
	}
	gotScopes := map[string]bool{}
	for _, scope := range do.CredentialScopes {
		gotScopes[scope] = true
	}
	for _, scope := range wantScopes {
		if !gotScopes[scope] {
			t.Errorf("DigitalOcean token scope %q missing", scope)
		}
	}
	if do.CredentialField != "digitalocean-api-token" || len(do.ConsoleSteps) != 3 || len(do.Offers) == 0 {
		t.Fatalf("DigitalOcean setup descriptor incomplete: %#v", do)
	}
	wantProbes := map[string]string{"droplets": "/v2/droplets", "sizes": "/v2/sizes", "regions": "/v2/regions", "invoices": "/v2/customers/my/invoices?per_page=1"}
	if len(do.Probes) != len(wantProbes) {
		t.Fatalf("DigitalOcean probe set = %#v", do.Probes)
	}
	for _, probe := range do.Probes {
		if probe.Method != http.MethodGet || !probe.ReadOnly || wantProbes[probe.ID] != probe.Path {
			t.Fatalf("DigitalOcean setup probe is not read-only: %#v", probe)
		}
	}
	if do.BillingFacts["minimum"] != "60 seconds or US$0.01, whichever is higher" || do.BillingFacts["bundled_cpu_monthly_cap"] != "672 hours (28 days)" || do.BillingFacts["price_observation_lifetime"] != "5 minutes; expired observations are refused" {
		t.Fatalf("DigitalOcean sourced billing/freshness facts incomplete: %#v", do.BillingFacts)
	}
}

func TestDigitalOceanVerificationWithMissingCredentialDoesNoProviderRequest(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-do-no-token?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	providerRequests := 0
	svc := &service{
		repo: NewRepository(db),
		resolve: func(_ context.Context, field string) (string, error) {
			if field != "digitalocean-api-token" {
				t.Fatalf("resolved credential field = %q", field)
			}
			return "", errors.New("fixture credential missing")
		},
		client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			providerRequests++
			return nil, errors.New("unexpected provider request")
		})},
		now: func() time.Time { return time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC) },
	}
	response, err := svc.VerifyProvider(context.Background(), connect.NewRequest(&setupv1.VerifyProviderRequest{ProviderId: "digitalocean", Region: "nyc1", Size: "s-1vcpu-1gb"}))
	if err != nil {
		t.Fatal(err)
	}
	if providerRequests != 0 {
		t.Fatalf("missing credential caused %d provider requests", providerRequests)
	}
	if response.Msg.GetReadiness().GetPurchaseEnabled() || response.Msg.GetReadiness().GetCredentialStatus() != "missing" {
		t.Fatalf("missing credential returned unsafe readiness: %#v", response.Msg.GetReadiness())
	}
}

func TestReadinessStopsAtFirstUnknown(t *testing.T) {
	verified := func(state State) Evidence { return Evidence{State: state, Verified: true} }
	evidence := map[State]Evidence{Authenticated: verified(Authenticated), AccountUsable: verified(AccountUsable), OfferAvailable: {State: OfferAvailable, Detail: "not available"}}
	got := Evaluate("hetzner", true, true, evidence)
	if got.Current != AccountUsable {
		t.Fatalf("state = %s, want account_usable", got.Current)
	}
	if got := Evaluate("hetzner", false, true, evidence); got.Current != NeedsConsoleSteps {
		t.Fatalf("without console steps state = %s", got.Current)
	}
	if got := Evaluate("hetzner", true, false, evidence); got.Current != CredentialNeeded {
		t.Fatalf("without credential state = %s", got.Current)
	}
}
