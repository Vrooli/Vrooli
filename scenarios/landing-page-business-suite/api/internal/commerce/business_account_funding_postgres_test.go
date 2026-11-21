package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"landing-page-business-suite-api/internal/businessaccount"
	reservationstate "landing-page-business-suite-api/internal/commerce/reservation/flow/generated"
)

func TestFundingLotsPostgresSpendOrderAndIsolation(t *testing.T) {
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
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	deadline := time.Now().Add(10 * time.Second)
	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("postgres testcontainer did not accept connections")
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, schema := range []string{businessaccount.Schema(), FinancialSchema(), OperationsSchema()} {
		if _, err := db.ExecContext(ctx, schema); err != nil {
			t.Fatalf("apply production commerce schema: %v", err)
		}
	}
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets VALUES('acct','buyer@example.com',0,0,NOW())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets VALUES('acct-legacy','legacy@example.com',11,2,NOW())`); err != nil {
		t.Fatal(err)
	}
	t.Run("historical balances never become funding", func(t *testing.T) {
		for modeIndex, liveMode := range []bool{false, true} {
			service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 1000}, Dialect: "postgres", LiveMode: boolRef(liveMode), Now: func() time.Time { return now }})
			if _, err := service.ReserveCredits(ctx, "acct-legacy", "legacy@example.com", "pro", "ai_credits", 1); err == nil {
				t.Fatalf("historical balance funded a reservation in live mode %t", liveMode)
			}
			if err := service.RecordUsage(ctx, "acct-legacy", "legacy@example.com", "pro", "ai_credits", 1, fmt.Sprintf("unfunded-usage-%t", liveMode), "console", "model", 1, 1, 1, "provider"); err == nil {
				t.Fatalf("historical balance funded direct usage in live mode %t", liveMode)
			}
			reservationID := fmt.Sprintf("00000000-0000-4000-8000-%012d", modeIndex+200)
			if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_reservations(id,business_account_id,actor_identity,billing_period,limit_key,reserved_amount,status,expires_at,livemode) VALUES($1,'acct-legacy','legacy@example.com',$2,'ai_credits',1,'pending',$3,$4)`, reservationID, now.Format("2006-01"), now.Add(time.Hour), liveMode); err != nil {
				t.Fatal(err)
			}
			if err := service.FinalizeReservationWithCost(ctx, "acct-legacy", reservationID, 1, "console", "model", 1, 1, 1, "provider"); err == nil {
				t.Fatalf("historical balance funded settlement in live mode %t", liveMode)
			}
		}
		for _, table := range []string{"credit_funding_lots", "business_account_usage_records", "business_account_usage_events"} {
			var count int64
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE business_account_id='acct-legacy'`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("historical balance created %d rows in %s; want none", count, table)
			}
		}
		wallet := NewCreditWalletService(db)
		if balance, err := wallet.BalanceForBusinessAccount("acct-legacy", "legacy@example.com"); err != nil || balance != 0 {
			t.Fatalf("historical cached balance displayed=%d error=%v; want 0", balance, err)
		}
		for attempt := 0; attempt < 2; attempt++ {
			if err := wallet.AddCreditsForBusinessAccount("acct-legacy", "legacy@example.com", 5, "credit_topup", "stripe", "evt-new-attributed-grant", map[string]interface{}{"livemode": false}); err != nil {
				t.Fatal(err)
			}
		}
		if balance, err := wallet.BalanceForBusinessAccount("acct-legacy", "legacy@example.com"); err != nil || balance != 5 {
			t.Fatalf("new attributed balance after replay=%d error=%v; want 5", balance, err)
		}
		if balance, err := wallet.BalanceForBusinessAccount("acct-legacy", "other@example.com"); err != nil || balance != 0 {
			t.Fatalf("another identity saw balance=%d error=%v; want 0", balance, err)
		}
	})
	lots := []struct {
		id, ref       string
		remaining     int64
		expiry        *time.Time
		mode          bool
		state, status string
	}{
		{"00000000-0000-4000-8000-000000000001", "early", 15, pTime(now.Add(24 * time.Hour)), true, "settled", "open"},
		{"00000000-0000-4000-8000-000000000002", "later", 20, nil, true, "settled", "open"},
		{"00000000-0000-4000-8000-000000000003", "held", 100, nil, true, "pending", "open"},
		{"00000000-0000-4000-8000-000000000004", "test", 100, nil, false, "settled", "open"},
	}
	for _, lot := range lots {
		if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,expires_at,status,created_at) VALUES($1,'acct','paid','stripe',$2,$3,$4,$5,$5,$6,$7,NOW())`, lot.id, lot.ref, lot.mode, lot.state, lot.remaining, lot.expiry, lot.status); err != nil {
			t.Fatal(err)
		}
	}
	id := 0
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 1000}, Dialect: "postgres", LiveMode: boolRef(true), Now: func() time.Time { return now }, NewID: func() string { id++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", id+100) }})
	reservation, err := service.ReserveCredits(ctx, "acct", "buyer@example.com", "pro", "ai_credits", 30)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct", reservation, 30, "console", "model", 900, 1, 1, "provider"); err != nil {
		t.Fatal(err)
	}
	var early, later, allocs int64
	if err := db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE provider_ref='early'`).Scan(&early); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE provider_ref='later'`).Scan(&later); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_lot_allocations WHERE operation_id=$1`, reservation).Scan(&allocs); err != nil {
		t.Fatal(err)
	}
	if early != 0 || later != 5 || allocs != 2 {
		t.Fatalf("allocation spend order early=%d later=%d allocations=%d; want 0,5,2", early, later, allocs)
	}
	assertPostgresReservationOverrunCapped(t, ctx, db, service)
	grantWallet := NewCreditWalletService(db)
	if err := grantWallet.AddCreditsForBusinessAccount("acct-grants", "buyer@example.com", 40, "credit_topup", "stripe", "evt-test-grant", map[string]interface{}{"livemode": false}); err != nil {
		t.Fatalf("write test-mode account grant: %v", err)
	}
	if err := grantWallet.AddCreditsForBusinessAccount("acct-grants", "buyer@example.com", 40, "credit_topup", "stripe", "evt-test-grant", map[string]interface{}{"livemode": false}); err != nil {
		t.Fatalf("replay test-mode account grant: %v", err)
	}
	if err := grantWallet.AddCreditsForBusinessAccount("acct-grants", "buyer@example.com", 60, "credit_topup", "stripe", "evt-live-grant", map[string]interface{}{"livemode": true}); err != nil {
		t.Fatalf("write live-mode account grant: %v", err)
	}
	var testLots, liveLots int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots WHERE business_account_id='acct-grants' AND livemode=FALSE`).Scan(&testLots); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots WHERE business_account_id='acct-grants' AND livemode=TRUE`).Scan(&liveLots); err != nil {
		t.Fatal(err)
	}
	if testLots != 1 || liveLots != 1 {
		t.Fatalf("grant lots by mode test=%d live=%d; want 1 each", testLots, liveLots)
	}
	if balance, err := grantWallet.BalanceForBusinessAccount("acct-grants", "buyer@example.com"); err != nil || balance != 100 {
		t.Fatalf("attributed grant balance=%d error=%v; want 100", balance, err)
	}
	var grantIdentity string
	if err := db.QueryRow(`SELECT customer_email FROM business_account_credit_wallets WHERE business_account_id='acct-grants'`).Scan(&grantIdentity); err != nil || grantIdentity != "buyer@example.com" {
		t.Fatalf("grant identity=%q error=%v; want buyer@example.com", grantIdentity, err)
	}
	var linkedGrants int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_transactions t JOIN credit_funding_lots l ON l.id=t.lot_id WHERE t.business_account_id='acct-grants' AND t.amount_credits=l.granted AND t.source=l.provider AND t.external_event_id=l.external_event_id`).Scan(&linkedGrants); err != nil || linkedGrants != 2 {
		t.Fatalf("attributed grant links=%d error=%v; want 2", linkedGrants, err)
	}
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES('00000000-0000-4000-8000-000000000005','acct-test','test','stripe','test-only',FALSE,'settled',50,50,'open')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO business_account_credit_wallets VALUES('acct-test','test@example.com',50,0,NOW())`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReserveCredits(ctx, "acct-test", "test@example.com", "pro", "ai_credits", 1); err == nil {
		t.Fatal("live route must reject test-only credits")
	}
	if _, err := db.Exec(`INSERT INTO business_account_credit_wallets VALUES('acct-retry','retry@example.com',0,0,NOW())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES('00000000-0000-4000-8000-000000000006','acct-retry','paid','stripe','retry-live',TRUE,'settled',20,20,'open')`); err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{
		`CREATE SEQUENCE e2a1_retry_once_seq`,
		`CREATE FUNCTION e2a1_fail_first_reservation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF nextval('e2a1_retry_once_seq')=1 THEN RAISE EXCEPTION 'serialization conflict test' USING ERRCODE='40001'; END IF; RETURN NEW; END $$`,
		`CREATE TRIGGER e2a1_retry_once BEFORE INSERT ON business_account_credit_reservations FOR EACH ROW EXECUTE FUNCTION e2a1_fail_first_reservation()`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	retryService := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 1000}, Dialect: "postgres", LiveMode: boolRef(true), Now: func() time.Time { return now }, NewID: func() string { id++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", id+400) }})
	if _, err := retryService.ReserveCredits(ctx, "acct-retry", "retry@example.com", "pro", "ai_credits", 1); err != nil {
		t.Fatalf("retry serialization conflict: %v", err)
	}
	var retryAttempts, pendingReservations int64
	if err := db.QueryRow(`SELECT last_value FROM e2a1_retry_once_seq`).Scan(&retryAttempts); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE business_account_id='acct-retry' AND status='pending'`).Scan(&pendingReservations); err != nil {
		t.Fatal(err)
	}
	if retryAttempts != 2 || pendingReservations != 1 {
		t.Fatalf("serialization retry attempts=%d pending=%d; want 2 and 1", retryAttempts, pendingReservations)
	}
	if _, err := service.ReserveCredits(ctx, "acct", "buyer@example.com", "pro", "ai_credits", 1); err != nil {
		t.Fatalf("remaining settled live lot should fund follow-up: %v", err)
	}
	t.Run("Given a hold and an unsettled lot, when the unsettled lot freezes, then settlement uses only settled funding", func(t *testing.T) {
		const account = "acct-unsettled-freeze"
		const settledLot = "00000000-0000-4000-8000-000000000601"
		const pendingLot = "00000000-0000-4000-8000-000000000602"
		if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets VALUES($1,'buyer@example.com',0,0,NOW())`, account); err != nil {
			t.Fatal(err)
		}
		for _, lot := range []struct {
			id, settlement string
			expiresAt      time.Time
		}{
			{settledLot, "settled", now.Add(48 * time.Hour)},
			{pendingLot, "pending", now.Add(24 * time.Hour)},
		} {
			if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,expires_at,status,created_at) VALUES($1,$2,'paid','stripe',$5,TRUE,$3,10,10,$4,'open',NOW())`, lot.id, account, lot.settlement, lot.expiresAt, lot.id); err != nil {
				t.Fatal(err)
			}
		}
		reservation, err := service.ReserveCredits(ctx, account, "buyer@example.com", "pro", "ai_credits", 10)
		if err != nil {
			t.Fatalf("reserve against the settled lot: %v", err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyFundingReversal(ctx, tx, pendingLot, account, "radar.early_fraud_warning.created", map[string]interface{}{"id": "warning-pending"}); err != nil {
			_ = tx.Rollback()
			t.Fatalf("freeze unsettled lot through reversal owner: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := service.FinalizeReservationWithCost(ctx, account, reservation, 10, "console", "model", 100, 1, 1, "provider"); err != nil {
			t.Fatalf("settle the prior hold: %v", err)
		}
		var allocatedLot string
		if err := db.QueryRowContext(ctx, `SELECT lot_id FROM credit_lot_allocations WHERE operation_id=$1`, reservation).Scan(&allocatedLot); err != nil {
			t.Fatal(err)
		}
		if allocatedLot != settledLot {
			t.Fatalf("settlement used lot %s; want the only settled lot %s", allocatedLot, settledLot)
		}
		var pendingRemaining, settledRemaining, actual, charged, overrun, eventCredits, usageAmount int64
		for _, check := range []struct {
			lot  string
			into *int64
		}{
			{pendingLot, &pendingRemaining}, {settledLot, &settledRemaining},
		} {
			if err := db.QueryRowContext(ctx, `SELECT remaining FROM credit_funding_lots WHERE id=$1`, check.lot).Scan(check.into); err != nil {
				t.Fatal(err)
			}
		}
		if err := db.QueryRowContext(ctx, `SELECT actual_amount,charged_amount,overrun_amount FROM business_account_credit_reservations WHERE id=$1`, reservation).Scan(&actual, &charged, &overrun); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT credits FROM business_account_usage_events WHERE operation_id=$1`, reservation).Scan(&eventCredits); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT usage_amount FROM business_account_usage_records WHERE business_account_id=$1`, account).Scan(&usageAmount); err != nil {
			t.Fatal(err)
		}
		if pendingRemaining != 10 || settledRemaining != 0 || actual != 10 || charged != 10 || overrun != 0 || eventCredits != 10 || usageAmount != 10 {
			t.Fatalf("money facts pending=%d settled=%d actual=%d charged=%d overrun=%d event=%d usage=%d", pendingRemaining, settledRemaining, actual, charged, overrun, eventCredits, usageAmount)
		}
		if err := service.FinalizeReservationWithCost(ctx, account, reservation, 10, "console", "model", 100, 1, 1, "provider"); err != nil {
			t.Fatalf("exact settlement replay: %v", err)
		}
		if err := service.FinalizeReservationWithCost(ctx, account, reservation, 11, "console", "model", 100, 1, 1, "provider"); err == nil {
			t.Fatal("conflicting terminal replay should fail")
		}
	})
}

func assertPostgresReservationOverrunCapped(t *testing.T, ctx context.Context, db *sql.DB, service *BusinessAccountUsageService) {
	t.Helper()
	if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets VALUES('acct-overrun','buyer@example.com',5,0,NOW())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES('00000000-0000-4000-8000-000000000500','acct-overrun','paid','stripe','overrun-lot',TRUE,'settled',5,5,'open',NOW())`); err != nil {
		t.Fatal(err)
	}
	overrunReservation, err := service.ReserveCredits(ctx, "acct-overrun", "buyer@example.com", "pro", "ai_credits", 5)
	if err != nil {
		t.Fatalf("reserve postgres overrun case: %v", err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-overrun", overrunReservation, 7, "console", "model", 777, 2, 3, "provider"); err != nil {
		t.Fatalf("settle postgres overrun case: %v", err)
	}
	var overrunStatus string
	var observed, charged, overrun, eventCredits, eventCost int64
	if err := db.QueryRow(`SELECT status,actual_amount,charged_amount,overrun_amount FROM business_account_credit_reservations WHERE id=$1`, overrunReservation).Scan(&overrunStatus, &observed, &charged, &overrun); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT credits,cost_micros FROM business_account_usage_events WHERE operation_id=$1`, overrunReservation).Scan(&eventCredits, &eventCost); err != nil {
		t.Fatal(err)
	}
	if overrunStatus != string(reservationstate.ReservationOverrun) || observed != 7 || charged != 5 || overrun != 2 || eventCredits != 5 || eventCost != 777 {
		t.Fatalf("postgres overrun status=%s observed=%d charged=%d overrun=%d event credits=%d provider cost=%d", overrunStatus, observed, charged, overrun, eventCredits, eventCost)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-overrun", overrunReservation, 7, "console", "model", 777, 2, 3, "provider"); err != nil {
		t.Fatalf("replay postgres overrun case: %v", err)
	}
}

func pTime(value time.Time) *time.Time { return &value }
func boolRef(value bool) *bool         { return &value }
