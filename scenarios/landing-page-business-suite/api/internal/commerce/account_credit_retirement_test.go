package commerce

import (
	"context"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"

	_ "modernc.org/sqlite"
)

func TestAccountCreditGrantRejectsMissingAccount(t *testing.T) {
	wallet := NewCreditWalletService(nil)
	if err := wallet.AddCreditsForBusinessAccount("", "member@example.test", 10, "credit_topup", "stripe", "event", map[string]interface{}{"livemode": false}); err == nil {
		t.Fatal("a grant without a verified account must fail before any ledger mutation")
	}
}

func TestPersonalCreditReadersRequireAccount(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE credit_wallets (customer_email TEXT, balance_credits INTEGER, bonus_credits INTEGER, updated_at DATETIME); INSERT INTO credit_wallets VALUES ('member@example.test', 77, 0, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, testEntitlementPlanCatalog{}, Runtime{})
	if _, err := service.GetCreditsForBusinessAccountContext(context.Background(), "member@example.test", ""); err == nil {
		t.Fatal("an empty selected account must not fall back to the email wallet")
	}
}

func TestAccountCreditGrantRequiresCompleteFundingAttribution(t *testing.T) {
	for _, tc := range []struct {
		name, source, event string
		metadata            map[string]interface{}
	}{
		{"missing source", "", "event", map[string]interface{}{"livemode": false}},
		{"missing event", "stripe", "", map[string]interface{}{"livemode": false}},
		{"missing mode", "stripe", "event", nil},
		{"unknown mode", "stripe", "event", map[string]interface{}{"livemode": "unknown"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := NewCreditWalletService(nil).AddCreditsForBusinessAccount("acct", "member@example.test", 10, "credit_topup", tc.source, tc.event, tc.metadata); err == nil {
				t.Fatal("incomplete funding provenance must fail before persistence")
			}
		})
	}
}

func TestAccountFundingEventReplayPreservesOriginalGrant(t *testing.T) {
	for _, tc := range []struct {
		name, account string
		amount        int64
		mode          bool
		wantError     bool
	}{
		{"identical", "acct", 10, false, false},
		{"changed account", "other", 10, false, true},
		{"changed amount", "acct", 11, false, true},
		{"changed mode", "acct", 10, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO business_account_credit_transactions").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("SELECT business_account_id,customer_email,amount_credits,transaction_type,metadata FROM business_account_credit_transactions").WithArgs("stripe", "event").WillReturnRows(sqlmock.NewRows([]string{"account", "email", "amount", "kind", "metadata"}).AddRow("acct", "member@example.test", 10, "credit_topup", []byte(`{ "note": "grant", "livemode": false }`)))
			mock.ExpectRollback()
			err = NewCreditWalletService(db).AddCreditsForBusinessAccount(tc.account, "member@example.test", tc.amount, "credit_topup", "stripe", "event", map[string]interface{}{"livemode": tc.mode, "note": "grant"})
			if (err != nil) != tc.wantError {
				t.Fatalf("replay error=%v, wantError=%v", err, tc.wantError)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
