package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"landing-page-business-suite-api/internal/businessaccount"
)

// lotModel is deliberately independent of the SQL predicate. The status and
// provider-state axes are the two checked formal flows' inputs; the remaining
// attributes are repository facts that must be composed before admission.
type lotModel struct {
	status, settlement, source string
	lotLive, spendLive         bool
	ready, unexpired           bool
	value                      int64
	accountMatches             bool
}

type composedScopeLimits struct{}

func (composedScopeLimits) GetLimit(_ context.Context, tier, key string, _ *string) (*TierLimit, error) {
	return &TierLimit{TierID: tier, LimitKey: key, LimitType: "cost_based", LimitValue: 7, CostMultiplier: 1000000}, nil
}

func (composedScopeLimits) GetTierLimits(_ context.Context, tier string) ([]TierLimit, error) {
	app := "app-a"
	return []TierLimit{
		{TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: 7, CostMultiplier: 1000000},
		{TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: 4, CostMultiplier: 1000000, AppBundleKey: &app},
	}, nil
}

func (m lotModel) spendable() bool {
	return m.status == "open" && m.settlement == "settled" && m.lotLive == m.spendLive &&
		m.ready && m.unexpired && m.value > 0 && m.accountMatches
}

func newComposedMoneyDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("commerce"), postgres.WithUsername("commerce"), postgres.WithPassword("commerce"))
	if err != nil {
		t.Fatalf("start composed money Postgres: %v", err)
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
			t.Fatal("composed money Postgres did not accept connections")
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, schema := range []string{businessaccount.Schema(), FinancialSchema(), OperationsSchema()} {
		if _, err := db.ExecContext(ctx, schema); err != nil {
			t.Fatalf("apply production commerce schema: %v", err)
		}
	}
	return db
}

func TestComposedMoneyModelPostgres(t *testing.T) {
	db := newComposedMoneyDB(t)
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	t.Run("SM1 eligibility product", func(t *testing.T) {
		ctx := context.Background()
		index := 0
		for _, status := range []string{"open", "frozen", "exhausted", "expired", "reversed"} {
			for _, settlement := range []string{"pending", "settled", "failed", "reversed"} {
				for _, source := range []string{"paid", "test"} {
					for _, lotLive := range []bool{false, true} {
						for _, spendLive := range []bool{false, true} {
							for _, timing := range []string{"ready", "future", "expired"} {
								for _, value := range []int64{0, 5} {
									for _, accountMatches := range []bool{false, true} {
										index++
										id := fmt.Sprintf("00000000-0000-4000-8001-%012d", index)
										account := fmt.Sprintf("matrix-%d", index)
										lotAccount := account
										if !accountMatches {
											lotAccount += "-other"
										}
										spendableAfter, expiresAt := now.Add(-time.Hour), now.Add(time.Hour)
										if timing == "future" {
											spendableAfter = now.Add(time.Hour)
										}
										if timing == "expired" {
											expiresAt = now
										}
										m := lotModel{status, settlement, source, lotLive, spendLive, timing != "future", timing != "expired", value, accountMatches}
										if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,spendable_after,expires_at,granted,remaining,status) VALUES($1,$2,$3,'fixture','fixture',$4,$5,$6,$7,5,$8,$9)`, id, lotAccount, source, lotLive, settlement, spendableAfter, expiresAt, value, status); err != nil {
											t.Fatalf("insert model case %d: %v", index, err)
										}
										tx, err := db.BeginTx(ctx, nil)
										if err != nil {
											t.Fatal(err)
										}
										balance, err := spendableFundingBalance(ctx, tx, "postgres", account, spendLive, now)
										if err != nil {
											_ = tx.Rollback()
											t.Fatalf("balance case %d: %v", index, err)
										}
										lots, err := loadSpendableLots(ctx, tx, "postgres", account, spendLive, now)
										_ = tx.Rollback()
										if err != nil {
											t.Fatalf("load case %d: %v", index, err)
										}
										want := int64(0)
										if m.spendable() {
											want = value
										}
										if balance != want || len(lots) != int(want/5) {
											t.Fatalf("case %d %+v: balance=%d lots=%d want=%d", index, m, balance, len(lots), want)
										}
										if want > 0 && (lots[0].id != id || lots[0].sourceKind != source) {
											t.Fatalf("case %d lost source provenance: got lot=%s source=%s, want lot=%s source=%s", index, lots[0].id, lots[0].sourceKind, id, source)
										}
									}
								}
							}
						}
					}
				}
			}
		}
		if index != 1920 {
			t.Fatalf("SM1 model cases=%d, want 1920", index)
		}
	})
	t.Run("SM2 settlement product and terminal replay", func(t *testing.T) {
		ctx := context.Background()
		index := 0
		for _, source := range []string{"paid", "test"} {
			for _, live := range []bool{false, true} {
				for _, app := range []string{"", "app-a"} {
					for _, actual := range []int64{0, 5, 6} {
						index++
						account := fmt.Sprintf("settlement-%d", index)
						lot := fmt.Sprintf("00000000-0000-4000-8002-%012d", index)
						operation := fmt.Sprintf("00000000-0000-4000-8003-%012d", index)
						if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets(business_account_id,customer_email) VALUES($1,'actor@example.com')`, account); err != nil {
							t.Fatal(err)
						}
						if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES($1,$2,$3,'fixture','fixture',$4,'settled',5,5,'open')`, lot, account, source, live); err != nil {
							t.Fatal(err)
						}
						service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "postgres", LiveMode: boolRef(live), Now: func() time.Time { return now }})
						id, created, err := service.ReserveScopedCreditsWithID(ctx, account, "actor@example.com", "pro", "ai_credits", app, 5, operation)
						if err != nil || !created || id != operation {
							t.Fatalf("reserve case %d: id=%s created=%t err=%v", index, id, created, err)
						}
						if _, replayed, err := service.ReserveScopedCreditsWithID(ctx, account, "actor@example.com", "pro", "ai_credits", app, 5, operation); err != nil || replayed {
							t.Fatalf("reserve exact replay case %d: created=%t err=%v", index, replayed, err)
						}
						secondOperation := fmt.Sprintf("00000000-0000-4000-8006-%012d", index)
						if _, _, err := service.ReserveScopedCreditsWithID(ctx, account, "actor@example.com", "pro", "ai_credits", app, 1, secondOperation); err == nil {
							t.Fatalf("case %d spent a pending hold", index)
						}
						for _, change := range []struct {
							account, actor, app string
							amount              int64
						}{
							{account + "-other", "actor@example.com", app, 5},
							{account, "other@example.com", app, 5},
							{account, "actor@example.com", app + "-other", 5},
							{account, "actor@example.com", app, 4},
						} {
							if _, _, err := service.ReserveScopedCreditsWithID(ctx, change.account, change.actor, "pro", "ai_credits", change.app, change.amount, operation); err == nil {
								t.Fatalf("case %d accepted conflicting reservation replay: %+v", index, change)
							}
						}
						if err := service.VerifyActor(ctx, account, "other@example.com", operation); err == nil {
							t.Fatalf("case %d accepted wrong actor", index)
						}
						if err := service.FinalizeReservationWithCost(ctx, account+"-other", operation, actual, app, "model", 101, 1, 2, "provider"); err == nil {
							t.Fatalf("case %d accepted wrong account settlement", index)
						}
						if err := service.FinalizeReservationWithCost(ctx, account, operation, actual, app, "model", 101, 1, 2, "provider"); err != nil {
							t.Fatalf("settle case %d: %v", index, err)
						}
						if err := service.FinalizeReservationWithCost(ctx, account, operation, actual, app, "model", 101, 1, 2, "provider"); err != nil {
							t.Fatalf("exact settlement replay case %d: %v", index, err)
						}
						if err := service.FinalizeReservationWithCost(ctx, account, operation, actual, app, "model", 102, 1, 2, "provider"); err == nil {
							t.Fatalf("case %d accepted conflicting terminal replay", index)
						}
						charged := actual
						if charged > 5 {
							charged = 5
						}
						overrun := actual - charged
						wantStatus := "finalized"
						if overrun > 0 {
							wantStatus = "overrun"
						}
						var status, gotActor, gotApp string
						var gotMode bool
						var measured, gotCharged, gotOverrun int64
						if err := db.QueryRowContext(ctx, `SELECT status,actor_identity,app_bundle_key,livemode,actual_amount,charged_amount,overrun_amount FROM business_account_credit_reservations WHERE id=$1`, operation).Scan(&status, &gotActor, &gotApp, &gotMode, &measured, &gotCharged, &gotOverrun); err != nil {
							t.Fatal(err)
						}
						if status != wantStatus || gotActor != "actor@example.com" || gotMode != live || measured != actual || gotCharged != charged || gotOverrun != overrun {
							t.Fatalf("case %d reservation: status=%s actor=%s actual=%d charge=%d overrun=%d", index, status, gotActor, measured, gotCharged, gotOverrun)
						}
						wantApp := app
						if wantApp == "" {
							wantApp = defaultUsageEventAppKey
						}
						if gotApp != wantApp {
							t.Fatalf("case %d app=%q want %q", index, gotApp, wantApp)
						}
						var allocated, allocationRows, usageAmount, eventCredits, eventRows, ledgerRows, remaining int64
						if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(credits),0),COUNT(*) FROM credit_lot_allocations WHERE operation_id=$1`, operation).Scan(&allocated, &allocationRows); err != nil {
							t.Fatal(err)
						}
						if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1`, account).Scan(&usageAmount); err != nil {
							t.Fatal(err)
						}
						if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(credits),0),COUNT(*) FROM business_account_usage_events WHERE operation_id=$1`, operation).Scan(&eventCredits, &eventRows); err != nil {
							t.Fatal(err)
						}
						if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_ledger_outbox WHERE lot_id=$1`, lot).Scan(&ledgerRows); err != nil {
							t.Fatal(err)
						}
						if err := db.QueryRowContext(ctx, `SELECT remaining FROM credit_funding_lots WHERE id=$1`, lot).Scan(&remaining); err != nil {
							t.Fatal(err)
						}
						wantAllocRows := int64(1)
						if charged == 0 {
							wantAllocRows = 0
						}
						if allocated != charged || allocationRows != wantAllocRows || usageAmount != actual || eventCredits != charged || eventRows != 1 || ledgerRows != 0 || remaining != 5-charged {
							t.Fatalf("case %d source=%s mode=%t app=%q measured=%d: allocations=%d/%d usage=%d event=%d/%d ledger=%d remaining=%d", index, source, live, app, actual, allocated, allocationRows, usageAmount, eventCredits, eventRows, ledgerRows, remaining)
						}
						if charged > 0 {
							var allocationSource, allocationAccount string
							var allocationMode bool
							if err := db.QueryRowContext(ctx, `SELECT l.source_kind,l.livemode,l.business_account_id FROM credit_lot_allocations a JOIN credit_funding_lots l ON l.id=a.lot_id WHERE a.operation_id=$1`, operation).Scan(&allocationSource, &allocationMode, &allocationAccount); err != nil {
								t.Fatal(err)
							}
							if allocationSource != source || allocationMode != live || allocationAccount != account {
								t.Fatalf("case %d allocated source=%s mode=%t account=%s", index, allocationSource, allocationMode, allocationAccount)
							}
						}
						_, created, err = service.ReserveScopedCreditsWithID(ctx, account, "actor@example.com", "pro", "ai_credits", app, 1, secondOperation)
						if (err == nil) != (charged == 0) || (err == nil && !created) {
							t.Fatalf("case %d available after settlement: charged=%d created=%t err=%v", index, charged, created, err)
						}
					}
				}
			}
		}
		if index != 24 {
			t.Fatalf("SM2 product cases=%d, want 24", index)
		}
	})
	t.Run("SM2 concurrent exact settlement", func(t *testing.T) {
		ctx := context.Background()
		const account = "settlement-race"
		const lot = "00000000-0000-4000-8002-000000000999"
		const operation = "00000000-0000-4000-8003-000000000999"
		if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets(business_account_id,customer_email) VALUES($1,'actor@example.com')`, account); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES($1,$2,'paid','fixture','fixture',TRUE,'settled',5,5,'open')`, lot, account); err != nil {
			t.Fatal(err)
		}
		service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "postgres", LiveMode: boolRef(true), Now: func() time.Time { return now }})
		if _, _, err := service.ReserveScopedCreditsWithID(ctx, account, "actor@example.com", "pro", "ai_credits", "app-a", 5, operation); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		errs := make(chan error, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- service.FinalizeReservationWithCost(ctx, account, operation, 6, "app-a", "model", 101, 1, 2, "provider")
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("concurrent exact settlement: %v", err)
			}
		}
		var allocations, events, usage, ledger int64
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM credit_lot_allocations WHERE operation_id=$1`, operation).Scan(&allocations); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM business_account_usage_events WHERE operation_id=$1`, operation).Scan(&events); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1`, account).Scan(&usage); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_ledger_outbox WHERE lot_id=$1`, lot).Scan(&ledger); err != nil {
			t.Fatal(err)
		}
		if allocations != 1 || events != 1 || usage != 6 || ledger != 0 {
			t.Fatalf("race allocations=%d events=%d usage=%d ledger=%d", allocations, events, usage, ledger)
		}
	})
	t.Run("SM2 account actor app and global scopes compose", func(t *testing.T) {
		ctx := context.Background()
		for i, account := range []string{"scope-a", "scope-b"} {
			lot := fmt.Sprintf("00000000-0000-4000-8009-%012d", i)
			if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets(business_account_id,customer_email) VALUES($1,'actor@example.com')`, account); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES($1,$2,'paid','fixture','fixture',TRUE,'settled',10,10,'open')`, lot, account); err != nil {
				t.Fatal(err)
			}
		}
		service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: composedScopeLimits{}, Dialect: "postgres", LiveMode: boolRef(true), Now: func() time.Time { return now }})
		reserve := func(account, actor, app string, amount int64, n int) error {
			operation := fmt.Sprintf("00000000-0000-4000-8010-%012d", n)
			_, _, err := service.ReserveScopedCreditsWithID(ctx, account, actor, "pro", "ai_credits", app, amount, operation)
			return err
		}
		if err := reserve("scope-a", "actor@example.com", "app-a", 4, 1); err != nil {
			t.Fatal(err)
		}
		if err := reserve("scope-a", "other@example.com", "app-a", 1, 2); err == nil {
			t.Fatal("app-a cap ignored across actors")
		}
		if err := reserve("scope-a", "other@example.com", "app-b", 3, 3); err != nil {
			t.Fatal(err)
		}
		if err := reserve("scope-a", "actor@example.com", "app-b", 1, 4); err == nil {
			t.Fatal("global cap ignored across apps and actors")
		}
		if err := reserve("scope-b", "actor@example.com", "app-b", 7, 5); err != nil {
			t.Fatalf("other account inherited cap: %v", err)
		}
		for _, check := range []struct {
			account, app string
			want         int64
		}{
			{"scope-a", "app-a", 4}, {"scope-a", "app-b", 3}, {"scope-b", "app-b", 7},
		} {
			var got int64
			if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND app_bundle_key=$2 AND status='pending'`, check.account, check.app).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != check.want {
				t.Fatalf("account=%s app=%s pending=%d want=%d", check.account, check.app, got, check.want)
			}
		}
	})
	t.Run("SM2 released and expired holds have no charge", func(t *testing.T) {
		ctx := context.Background()
		index := 0
		for _, source := range []string{"paid", "test"} {
			for _, live := range []bool{false, true} {
				for _, terminal := range []string{"released", "expired"} {
					index++
					account := fmt.Sprintf("terminal-%d", index)
					lot := fmt.Sprintf("00000000-0000-4000-8007-%012d", index)
					operation := fmt.Sprintf("00000000-0000-4000-8008-%012d", index)
					if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets(business_account_id,customer_email) VALUES($1,'actor@example.com')`, account); err != nil {
						t.Fatal(err)
					}
					if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES($1,$2,$3,'fixture','fixture',$4,'settled',5,5,'open')`, lot, account, source, live); err != nil {
						t.Fatal(err)
					}
					service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "postgres", LiveMode: boolRef(live), Now: func() time.Time { return now }})
					if _, _, err := service.ReserveScopedCreditsWithID(ctx, account, "actor@example.com", "pro", "ai_credits", "app-a", 5, operation); err != nil {
						t.Fatal(err)
					}
					if terminal == "released" {
						if err := service.ReleaseReservation(ctx, account, operation); err != nil {
							t.Fatal(err)
						}
					} else {
						if _, err := db.ExecContext(ctx, `UPDATE business_account_credit_reservations SET expires_at=NOW()-INTERVAL '1 second' WHERE id=$1`, operation); err != nil {
							t.Fatal(err)
						}
						if _, err := service.CleanupExpiredReservations(ctx); err != nil {
							t.Fatal(err)
						}
					}
					if err := service.FinalizeReservationWithCost(ctx, account, operation, 5, "app-a", "model", 101, 1, 2, "provider"); err == nil {
						t.Fatalf("%s hold accepted late settlement", terminal)
					}
					var status string
					var allocations, usageRows, eventRows, ledgerRows, remaining int64
					if err := db.QueryRowContext(ctx, `SELECT status FROM business_account_credit_reservations WHERE id=$1`, operation).Scan(&status); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM credit_lot_allocations WHERE operation_id=$1`, operation).Scan(&allocations); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM business_account_usage_records WHERE business_account_id=$1`, account).Scan(&usageRows); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM business_account_usage_events WHERE operation_id=$1`, operation).Scan(&eventRows); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_ledger_outbox WHERE lot_id=$1`, lot).Scan(&ledgerRows); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT remaining FROM credit_funding_lots WHERE id=$1`, lot).Scan(&remaining); err != nil {
						t.Fatal(err)
					}
					if status != terminal || allocations != 0 || usageRows != 0 || eventRows != 0 || ledgerRows != 0 || remaining != 5 {
						t.Fatalf("terminal=%s got status=%s allocation=%d usage=%d events=%d ledger=%d remaining=%d", terminal, status, allocations, usageRows, eventRows, ledgerRows, remaining)
					}
				}
			}
		}
		if index != 8 {
			t.Fatalf("terminal model cases=%d, want 8", index)
		}
	})
	t.Run("SM1 reversal and finalization order", func(t *testing.T) {
		for _, kind := range []string{"refund", "dispute"} {
			for _, order := range []string{"reverse-first", "finalize-first", "concurrent"} {
				t.Run(kind+"/"+order, func(t *testing.T) {
					ctx := context.Background()
					account := "reversal-" + kind + "-" + order
					lot := fmt.Sprintf("00000000-0000-4000-8004-%012d", len(account)*100+len(kind)*10+len(order))
					operation := fmt.Sprintf("00000000-0000-4000-8005-%012d", len(account)*100+len(kind)*10+len(order))
					if _, err := db.ExecContext(ctx, `INSERT INTO business_account_credit_wallets(business_account_id,customer_email) VALUES($1,'actor@example.com')`, account); err != nil {
						t.Fatal(err)
					}
					payment := "pi-" + account
					if _, err := db.ExecContext(ctx, `INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,paid_amount_minor,paid_currency) VALUES($1,$2,'paid','stripe',$3,TRUE,'settled',5,5,'open',100,'USD')`, lot, account, payment); err != nil {
						t.Fatal(err)
					}
					service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "postgres", LiveMode: boolRef(true), Now: func() time.Time { return now }})
					if _, _, err := service.ReserveScopedCreditsWithID(ctx, account, "actor@example.com", "pro", "ai_credits", "app-a", 5, operation); err != nil {
						t.Fatal(err)
					}
					webhook := &StripeWebhookService{db: db}
					apply := func() error {
						eventKind := "charge.refunded"
						object := map[string]interface{}{"id": "ch-" + account, "payment_intent": payment, "amount_refunded": float64(100)}
						if kind == "dispute" {
							eventKind = "charge.dispute.created"
							object = map[string]interface{}{"id": "dp-" + account, "payment_intent": payment, "charge": "ch-" + account, "status": "needs_response", "amount": float64(100)}
						}
						return webhook.handleFundingReversal(ctx, "event-"+account, eventKind, object, true)
					}
					finalize := func() error {
						return service.FinalizeReservationWithCost(ctx, account, operation, 5, "app-a", "model", 1000000, 1, 2, "provider")
					}
					switch order {
					case "reverse-first":
						if err := apply(); err != nil {
							t.Fatal(err)
						}
						if err := finalize(); err != nil {
							t.Fatal(err)
						}
					case "finalize-first":
						if err := finalize(); err != nil {
							t.Fatal(err)
						}
						if err := apply(); err != nil {
							t.Fatal(err)
						}
					case "concurrent":
						start := make(chan struct{})
						results := make(chan error, 2)
						go func() { <-start; results <- apply() }()
						go func() { <-start; results <- finalize() }()
						close(start)
						for i := 0; i < 2; i++ {
							if err := <-results; err != nil {
								t.Fatal(err)
							}
						}
					}
					if err := apply(); err != nil {
						t.Fatalf("exact reversal replay: %v", err)
					}
					if err := finalize(); err != nil {
						t.Fatalf("exact settlement replay: %v", err)
					}
					var allocated, remaining, usage, charged, ledgerLoss, ledgerReversal, lossCost, lossRows, reversalRows int64
					var status string
					if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations WHERE operation_id=$1`, operation).Scan(&allocated); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT remaining,status FROM credit_funding_lots WHERE id=$1`, lot).Scan(&remaining, &status); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1`, account).Scan(&usage); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events WHERE operation_id=$1`, operation).Scan(&charged); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor),0),COUNT(*) FROM funding_ledger_outbox WHERE lot_id=$1 AND category='loss'`, lot).Scan(&ledgerLoss, &lossRows); err != nil {
						t.Fatal(err)
					}
					category := "refund"
					if kind == "dispute" {
						category = "dispute"
					}
					if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor),0),COUNT(*) FROM funding_ledger_outbox WHERE lot_id=$1 AND category=$2`, lot, category).Scan(&ledgerReversal, &reversalRows); err != nil {
						t.Fatal(err)
					}
					if err := db.QueryRowContext(ctx, `SELECT consumed_cost_micros FROM funding_losses WHERE lot_id=$1 AND kind='funding_reversal'`, lot).Scan(&lossCost); err != nil {
						t.Fatal(err)
					}
					if allocated != 5 || usage != 5 || charged != 5 || ledgerLoss != 100 || ledgerReversal != 100 || lossRows != 1 || reversalRows != 1 || lossCost != 1000000 {
						t.Fatalf("reversal composition: allocated=%d usage=%d charged=%d ledger loss=%d/%d reversal=%d/%d loss cost=%d", allocated, usage, charged, ledgerLoss, lossRows, ledgerReversal, reversalRows, lossCost)
					}
					if kind == "refund" && remaining != 0 {
						t.Fatalf("refund remaining=%d, want zero", remaining)
					}
					if kind == "dispute" && order == "reverse-first" && status != "frozen" {
						t.Fatalf("dispute before settlement status=%s, want frozen", status)
					}
				})
			}
		}
	})
}
