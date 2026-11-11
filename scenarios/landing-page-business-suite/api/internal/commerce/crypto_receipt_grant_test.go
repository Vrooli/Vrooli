package commerce

import (
	"context"
	"database/sql"
	neturl "net/url"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/businessaccount"
)

type cryptoGrantPlanReader struct{ rate int64 }

func (r cryptoGrantPlanReader) GetBundleProduct() (*BundleProduct, error) {
	return &BundleProduct{CreditsPerUsd: r.rate}, nil
}

func TestAcceptedCryptoReceiptCreatesAndReplaysActualCreditLot(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("commerce"), postgres.WithUsername("commerce"), postgres.WithPassword("commerce"))
	if err != nil {
		t.Fatalf("start postgres testcontainer: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	primary, err := database.Open(ctx, database.Config{Driver: database.DriverPostgres, DSN: url})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = primary.Close() })
	deadline := time.Now().Add(10 * time.Second)
	for {
		if err := primary.PingContext(ctx); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("postgres testcontainer did not accept connections")
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, schema := range []string{businessaccount.Schema(), FinancialSchema(), OperationsSchema()} {
		if _, err := primary.ExecContext(ctx, schema); err != nil {
			t.Fatalf("apply owner schema: %v", err)
		}
	}
	admin, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE commerce_test`); err != nil {
		t.Fatal(err)
	}
	testURL, err := neturl.Parse(url)
	if err != nil {
		t.Fatal(err)
	}
	testURL.Path = "/commerce_test"
	routed := primary
	routed.SetTestPoolInitializer(func(ctx context.Context, testDB *sql.DB) error {
		for _, schema := range []string{businessaccount.Schema(), FinancialSchema(), OperationsSchema()} {
			if _, err := testDB.ExecContext(ctx, schema); err != nil {
				return err
			}
		}
		if _, err := testDB.ExecContext(ctx, `INSERT INTO business_accounts(id,display_name,billing_email) VALUES('acct-crypto','Crypto test','buyer@example.test')`); err != nil {
			return err
		}
		_, err := testDB.ExecContext(ctx, `INSERT INTO crypto_payer_customer_bindings(payer_identity,payer_address,business_account_id,customer_email) VALUES($1,$2,$3,$4)`, "customer:cf2-local", "0xabc", "acct-crypto", "buyer@example.test")
		return err
	})
	if err := routed.InstallTestPool(ctx, testURL.String(), "crypto-receipt-lease", 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = routed.ClearTestPool("crypto-receipt-lease") })
	testCtx := database.WithTestMode(ctx)
	grant := &CryptoReceiptGrantService{DB: routed, Plans: cryptoGrantPlanReader{rate: 1000}, Wallet: NewCreditWalletService(routed)}
	receipt := &ingestpb.AcceptedCryptoReceipt{
		Id: "ledger-receipt-cf2", PostingId: "posting-cf2", AcceptedBy: "treasury", AcceptedAt: timestamppb.Now(), Action: ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL,
		Source: &ingestpb.TreasuryReceiptLink{OperatorIdentity: "operator"}, AtomicDelta: "2500000", NoChargeback: true,
		Observation: &ingestpb.PreviewCryptoObservationRequest{Network: "eip155:8453", AssetId: "eip155:8453/erc20:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", AtomicAmount: "2500000", Decimals: 6, State: ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED},
		Valuation:   &ingestpb.CryptoValuation{AmountMinor: 250, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "test-explicit-valuation", ValuedAt: timestamppb.Now()},
		Attribution: &ingestpb.CryptoPayerAttribution{PayerAddress: "0xABC", PayerIdentity: "customer:cf2-local"},
	}
	lot, err := grant.GrantAcceptedReceipt(testCtx, receipt)
	if err != nil {
		t.Fatal(err)
	}
	if lot.ID == "" || lot.ReceiptID != receipt.Id || lot.Credits != 2500 {
		t.Fatalf("LPBS lot grant=%+v; want actual lot, receipt link and policy-derived credits", lot)
	}
	var network, asset, atomicAmount, payerAddress, payerIdentity, receiptID string
	var granted, paidMinor int64
	var noChargeback bool
	if err := routed.QueryRowContext(testCtx, `SELECT crypto_network,crypto_asset_id,crypto_atomic_amount::text,crypto_payer_address,crypto_payer_identity,money_ledger_receipt_id,crypto_no_chargeback,granted,paid_amount_minor FROM credit_funding_lots WHERE id=$1`, lot.ID).Scan(&network, &asset, &atomicAmount, &payerAddress, &payerIdentity, &receiptID, &noChargeback, &granted, &paidMinor); err != nil {
		t.Fatal(err)
	}
	if network != receipt.Observation.Network || asset != receipt.Observation.AssetId || atomicAmount != "2500000" || payerAddress != "0xabc" || payerIdentity != "customer:cf2-local" || receiptID != receipt.Id || !noChargeback || granted != 2500 || paidMinor != 250 {
		t.Fatalf("LPBS lot receipt attribution network=%s asset=%s atoms=%s payer=%s identity=%s receipt=%s no_chargeback=%t granted=%d paid_minor=%d", network, asset, atomicAmount, payerAddress, payerIdentity, receiptID, noChargeback, granted, paidMinor)
	}
	replay, err := grant.GrantAcceptedReceipt(testCtx, receipt)
	if err != nil || replay.ID != lot.ID {
		t.Fatalf("accepted receipt replay lot=%+v error=%v; want same lot %s", replay, err, lot.ID)
	}
	var lotCount int
	if err := routed.QueryRowContext(testCtx, `SELECT COUNT(*) FROM credit_funding_lots WHERE money_ledger_receipt_id=$1`, receipt.Id).Scan(&lotCount); err != nil || lotCount != 1 {
		t.Fatalf("accepted receipt lot count=%d error=%v; want exactly one", lotCount, err)
	}
	var primaryLotCount int
	if err := primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM credit_funding_lots WHERE money_ledger_receipt_id=$1`, receipt.Id).Scan(&primaryLotCount); err != nil || primaryLotCount != 0 {
		t.Fatalf("test-mode receipt leaked %d credit lots into primary database (error=%v)", primaryLotCount, err)
	}
}
