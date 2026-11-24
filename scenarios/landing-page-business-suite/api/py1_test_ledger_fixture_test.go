package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

func TestPY1LedgerFixtureEnvironmentDropsCredentialsAndCallerIdentity(t *testing.T) {
	want := []string{"PATH=/bin", "HOME=/tmp", "VROOLI_ROOT=/vrooli", "VROOLI_SOURCE_ROOT=/vrooli"}
	entries := append(append([]string(nil), want...),
		"STRIPE_SECRET_KEY=sk_test_fixture_only",
		"MONEY_LEDGER_SERVICE_TOKEN=fixture-only-token",
		"VROOLI_AUTH_SCENARIO_JWKS_URL=http://127.0.0.1:1234/jwks",
		"VROOLI_AGENT_RUN_ID=fixture-agent",
		"CODEX_THREAD_ID=fixture-thread",
	)
	got := py1LedgerFixtureEnvironment(entries)
	joined := strings.Join(got, "\n")
	for _, entry := range want {
		if !strings.Contains(joined, entry) {
			t.Errorf("safe fixture environment omitted %s", strings.SplitN(entry, "=", 2)[0])
		}
	}
	for _, forbidden := range []string{"STRIPE_SECRET_KEY=", "MONEY_LEDGER_SERVICE_TOKEN=", "VROOLI_AUTH_", "VROOLI_AGENT_", "CODEX_THREAD_ID="} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("fixture environment retained forbidden variable name %q", forbidden)
		}
	}
	if !strings.Contains(joined, "GOWORK=off") {
		t.Fatal("fixture environment must select the scenario's own Go module")
	}
}

func TestPY1LedgerFixtureLifecycleVerifiesIdentityAndRejectsUnsignedWrites(t *testing.T) {
	lease, err := startLocalPY1LedgerFixture(context.Background())
	if err != nil {
		t.Fatalf("start local isolated fixture: %v", err)
	}
	t.Cleanup(lease.Close)
	fixture, err := commerce.NewMoneyLedgerTestFixtureWriter(lease.config.URL, lease.config.FixtureID)
	if err != nil {
		t.Fatalf("verify isolated fixture identity: %v", err)
	}
	target, err := fixture.PY1Target()
	if err != nil || target.AdapterID != lease.config.AdapterID || target.BookID != lease.config.BookID || target.RevenueAccountID != lease.config.RevenueAccountID {
		t.Fatalf("fixture scope = %+v, err=%v", target, err)
	}
	authenticatorURL, err := discovery.ResolveScenarioURLDefault(context.Background(), "scenario-authenticator")
	if err != nil {
		t.Fatalf("discover Scenario Authenticator: %v", err)
	}
	keys, err := http.Get(strings.TrimRight(authenticatorURL, "/") + "/.well-known/jwks.json")
	if err != nil {
		t.Fatalf("read Scenario Authenticator JWKS: %v", err)
	}
	io.Copy(io.Discard, keys.Body)
	keys.Body.Close()
	if keys.StatusCode != http.StatusOK {
		t.Fatalf("Scenario Authenticator JWKS status = %d", keys.StatusCode)
	}
	client := ingest_v1connect.NewIngestServiceClient(lease.client, lease.config.URL)
	request := connect.NewRequest(&ingest_v1.IngestEventRequest{Event: &sharedpb.MoneyEvent{
		ExternalId: "lpbs-py1-checkout:cs_test_unsigned_fixture_probe", AdapterId: target.AdapterID,
		BookId: target.BookID, AccountId: target.RevenueAccountID, AmountMinor: 1, Currency: "USD",
	}})
	jwtHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"py1-nonexistent"}`))
	jwtPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"scenario-authenticator","aud":"scenario-authenticator:money-ledger","sub":"landing-page-business-suite-money-ledger-writer","actor_kind":"service","scope":["money-ledger:write"],"exp":1893456000}`))
	request.Header().Set("Authorization", "Bearer "+jwtHeader+"."+jwtPayload+".AA")
	if _, err := client.IngestEvent(context.Background(), request); err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("fixture accepted an unsigned write or failed before auth: %v", err)
	}
	lease.Close()
	if lease.cmd.ProcessState == nil || !lease.cmd.ProcessState.Exited() {
		t.Fatal("fixture process remained running after cleanup")
	}
}
