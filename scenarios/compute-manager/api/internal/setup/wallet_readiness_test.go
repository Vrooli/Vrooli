package setup

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/setup"
	_ "modernc.org/sqlite"
)

func TestAkashReadinessUsesCAStatusMetadataWithoutResolvingSecret(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	resolveCalls := 0
	statuses := map[string]bool{}
	svc := &service{repo: NewRepository(db), resolve: func(context.Context, string) (string, error) { resolveCalls++; return "secret-value", nil }, configured: func(_ context.Context, field string) (bool, error) { return statuses[field], nil }, now: func() time.Time { return now }}
	readinessResponse, err := svc.GetReadiness(context.Background(), connect.NewRequest(&setupv1.GetReadinessRequest{ProviderId: "akash"}))
	if err != nil {
		t.Fatal(err)
	}
	ready := readinessResponse.Msg.GetReadiness()
	if ready.GetCredentialStatus() != "not_configured" || ready.GetCurrent() != string(CredentialNeeded) || resolveCalls != 0 {
		t.Fatalf("descriptor was mistaken for a credential or secret was resolved: readiness=%+v resolver_calls=%d", ready, resolveCalls)
	}
	if _, err := svc.repo.StartStep(context.Background(), StepRecord{ProviderID: "akash", StepID: "wallet", AskID: "fixture-ask", EventTime: now, State: "pending", UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	ready, err = svc.readiness(context.Background(), *findProvider("akash"))
	if err != nil {
		t.Fatal(err)
	}
	if ready.GetCredentialStatus() != "awaiting_secure_entry" || resolveCalls != 0 {
		t.Fatalf("phone prompt state=%q resolver_calls=%d", ready.GetCredentialStatus(), resolveCalls)
	}
	statuses["akash-testnet-wallet"] = true
	statuses["akash-console-api-key"] = true
	ready, err = svc.readiness(context.Background(), *findProvider("akash"))
	if err != nil {
		t.Fatal(err)
	}
	if ready.GetCredentialStatus() != "ready_for_authorized_verification" || ready.GetPurchaseEnabled() || resolveCalls != 0 {
		t.Fatalf("CA presence was misreported as verified signer: readiness=%+v resolver_calls=%d", ready, resolveCalls)
	}
	if ready.GetMarketSourceCredentialStatus() != "credential_present_unverified" {
		t.Fatalf("market API-key metadata status=%q", ready.GetMarketSourceCredentialStatus())
	}
	svc.configured = func(context.Context, string) (bool, error) { return false, errors.New("metadata unavailable") }
	ready, err = svc.readiness(context.Background(), *findProvider("akash"))
	if err != nil {
		t.Fatal(err)
	}
	if ready.GetCredentialStatus() != "status_unavailable" || resolveCalls != 0 {
		t.Fatalf("unknown CA status was misreported: readiness=%+v resolver_calls=%d", ready, resolveCalls)
	}
	if ready.GetMarketSourceCredentialStatus() != "status_unavailable" {
		t.Fatalf("unavailable market key metadata status=%q", ready.GetMarketSourceCredentialStatus())
	}
}

func TestAkashMarketKeyPhoneStepReportsPendingWithoutSecretResolution(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	resolveCalls := 0
	svc := &service{repo: NewRepository(db), resolve: func(context.Context, string) (string, error) { resolveCalls++; return "never", nil }, configured: func(context.Context, string) (bool, error) { return false, nil }, now: func() time.Time { return now }}
	if _, err := svc.repo.StartStep(context.Background(), StepRecord{ProviderID: "akash", StepID: "console-api-key", AskID: "fixture-ask", EventTime: now, State: "pending", UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	ready, err := svc.readiness(context.Background(), *findProvider("akash"))
	if err != nil {
		t.Fatal(err)
	}
	if ready.GetMarketSourceCredentialStatus() != "awaiting_secure_entry" || resolveCalls != 0 || ready.GetPurchaseEnabled() {
		t.Fatalf("market key status=%q resolver_calls=%d purchase=%t", ready.GetMarketSourceCredentialStatus(), resolveCalls, ready.GetPurchaseEnabled())
	}
}

func TestAkashVerifyProviderRefusesBeforeCredentialResolution(t *testing.T) {
	resolveCalls := 0
	svc := &service{resolve: func(context.Context, string) (string, error) { resolveCalls++; return "never", nil }}
	_, err := svc.VerifyProvider(context.Background(), connect.NewRequest(&setupv1.VerifyProviderRequest{ProviderId: "akash"}))
	if err == nil {
		t.Fatal("Akash verification unexpectedly proceeded")
	}
	if resolveCalls != 0 {
		t.Fatalf("Akash verification resolved secret %d times", resolveCalls)
	}
}
