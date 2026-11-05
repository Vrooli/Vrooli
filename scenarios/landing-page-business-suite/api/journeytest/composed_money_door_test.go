package journeytest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	moneyflow "landing-page-business-suite-api/internal/commerce/moneycomposition/flow/generated"
)

// This receipt drives the real Door and wallet against the same independent
// eligibility axes checked exhaustively in commerce's composed Postgres model.
// Only the upstream router is a deterministic fixture.
func TestJ17_ComposedMoneyDoorReceipt(t *testing.T) {
	s := newSpine(t)
	for _, tc := range []struct {
		name, status, settlement, account string
		mode                              bool
		spendableAfter, expiresAt         *time.Time
		remaining                         int64
	}{
		{name: "provider-pending", status: "open", settlement: "pending", account: "account", remaining: 10000},
		{name: "frozen", status: "frozen", settlement: "settled", account: "account", remaining: 10000},
		{name: "expired-status", status: "expired", settlement: "settled", account: "account", remaining: 10000},
		{name: "exhausted", status: "exhausted", settlement: "settled", account: "account", remaining: 10000},
		{name: "reversed", status: "reversed", settlement: "settled", account: "account", remaining: 10000},
		{name: "future", status: "open", settlement: "settled", account: "account", remaining: 10000, spendableAfter: doorTime(time.Now().UTC().Add(time.Hour))},
		{name: "expired-time", status: "open", settlement: "settled", account: "account", remaining: 10000, expiresAt: doorTime(time.Now().UTC().Add(-time.Hour))},
		{name: "wrong-mode", status: "open", settlement: "settled", account: "account", mode: true, remaining: 10000},
		{name: "wrong-account", status: "open", settlement: "settled", account: "other-account", remaining: 10000},
		{name: "zero-value", status: "open", settlement: "settled", account: "account", remaining: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.db.Exec(`UPDATE credit_funding_lots SET status=$1,settlement_state=$2,business_account_id=$3,livemode=$4,spendable_after=$5,expires_at=$6,remaining=$7`, tc.status, tc.settlement, tc.account, tc.mode, tc.spendableAfter, tc.expiresAt, tc.remaining); err != nil {
				t.Fatal(err)
			}
			resp := s.request(t, s.key.Token, "j17-"+tc.name, "hello")
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusPaymentRequired {
				t.Fatalf("%s status=%d, want 402", tc.name, resp.StatusCode)
			}
			var reservations, events, allocations, usageRecords, ledger int64
			for _, item := range []struct {
				query string
				value *int64
			}{
				{`SELECT COUNT(*) FROM business_account_credit_reservations`, &reservations},
				{`SELECT COUNT(*) FROM business_account_usage_events`, &events},
				{`SELECT COUNT(*) FROM credit_lot_allocations`, &allocations},
				{`SELECT COUNT(*) FROM business_account_usage_records`, &usageRecords},
				{`SELECT COUNT(*) FROM funding_ledger_outbox`, &ledger},
			} {
				if err := s.db.QueryRow(item.query).Scan(item.value); err != nil {
					t.Fatal(err)
				}
			}
			if reservations != 0 || events != 0 || allocations != 0 || usageRecords != 0 || ledger != 0 {
				t.Fatalf("%s changed money: reservations=%d events=%d allocations=%d records=%d ledger=%d", tc.name, reservations, events, allocations, usageRecords, ledger)
			}
			var remaining int64
			if err := s.db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE provider_ref='fixture-payment'`).Scan(&remaining); err != nil {
				t.Fatal(err)
			}
			wallet := moneyflow.ProductWalletNonnegative
			if remaining < 0 {
				wallet = moneyflow.ProductWalletNegative
			}
			moneyflow.AssertProductTraceObservations(t, "door_refusal", []moneyflow.ProductObservation{{
				State: moneyflow.ProductState{
					Eligibility: moneyflow.ProductEligibilityUnavailable,
					Reservation: moneyflow.ProductReservationNone,
					Charge:      moneyflow.ProductChargeNone,
					Wallet:      wallet,
					Replay:      moneyflow.ProductReplayFresh,
				},
				Effects: map[string]int64{"reservations": reservations, "allocations": allocations, "usageRecords": usageRecords, "usageEvents": events, "ledgerRows": ledger},
			}})
		})
	}
	if _, err := s.db.Exec(`UPDATE credit_funding_lots SET status='open',settlement_state='settled',business_account_id='account',livemode=FALSE,spendable_after=NULL,expires_at=NULL,remaining=10000`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "j17-spendable", "hello")
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("eligible status=%d, want 200", resp.StatusCode)
	}
	operation := resp.Header.Get("X-Operation-ID")
	if operation == "" {
		t.Fatal("missing settled operation receipt")
	}
	firstObservation := readJ17MoneyProductObservation(t, s, operation, "", resp.StatusCode)
	beforeReplay := snapshotJ17MoneyEffects(t, s, operation)
	receiptBefore := readJ17GatewayReceipt(t, s, operation)
	replay := s.request(t, s.key.Token, "j17-spendable", "hello")
	if replay.StatusCode != http.StatusConflict {
		t.Fatalf("replay status=%d, want established 409 settled-operation response", replay.StatusCode)
	}
	afterReplay := snapshotJ17MoneyEffects(t, s, operation)
	if beforeReplay != afterReplay {
		t.Fatalf("replay changed observed money effects:\nbefore=%+v\nafter=%+v", beforeReplay, afterReplay)
	}
	receiptAfter := readJ17GatewayReceipt(t, s, operation)
	if receiptBefore != receiptAfter {
		t.Fatalf("replay changed the Gateway receipt:\nbefore=%+v\nafter=%+v", receiptBefore, receiptAfter)
	}
	replayedObservation := readJ17MoneyProductObservation(t, s, operation, operation, replay.StatusCode)
	moneyflow.AssertProductTraceObservations(t, "door_settlement_replay", []moneyflow.ProductObservation{firstObservation, replayedObservation})
	var status, account, actor, app string
	var measured, charged, overrun, allocated, usage, eventCredits, eventRows, remaining, ledger int64
	if err := s.db.QueryRow(`SELECT status,business_account_id,actor_identity,app_bundle_key,actual_amount,charged_amount,overrun_amount FROM business_account_credit_reservations WHERE id=$1`, operation).Scan(&status, &account, &actor, &app, &measured, &charged, &overrun); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations WHERE operation_id=$1`, operation).Scan(&allocated); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id='account'`).Scan(&usage); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0),COUNT(*) FROM business_account_usage_events WHERE operation_id=$1`, operation).Scan(&eventCredits, &eventRows); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE provider_ref='fixture-payment'`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM funding_ledger_outbox`).Scan(&ledger); err != nil {
		t.Fatal(err)
	}
	if status != "finalized" || account != "account" || actor != "actor" || app != "journey" || measured != 200 || charged != 200 || overrun != 0 || allocated != 200 || usage != 200 || eventCredits != 200 || eventRows != 1 || remaining != 9800 || ledger != 0 {
		t.Fatalf("door receipt status=%s account=%s actor=%s app=%s measured=%d charged=%d overrun=%d allocation=%d usage=%d event=%d/%d remaining=%d ledger=%d", status, account, actor, app, measured, charged, overrun, allocated, usage, eventCredits, eventRows, remaining, ledger)
	}
}

func doorTime(value time.Time) *time.Time { return &value }

type j17GatewayReceipt struct {
	OperationID      string `json:"operation_id"`
	FeatureID        string `json:"feature_id"`
	Content          string `json:"content,omitempty"`
	State            string `json:"state"`
	ErrorCode        string `json:"error_code,omitempty"`
	Message          string `json:"message,omitempty"`
	RouteID          string `json:"route_id,omitempty"`
	Supplier         string `json:"supplier,omitempty"`
	Model            string `json:"model,omitempty"`
	CreatedAt        string `json:"created_at"`
	CostMicros       int64  `json:"cost_micros"`
	Credits          int64  `json:"credits"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	CostKnown        bool   `json:"cost_known"`
}

func readJ17GatewayReceipt(t *testing.T, s *spine, operation string) j17GatewayReceipt {
	t.Helper()
	response, err := http.Get(s.gateway + "/api/v1/private/provider-routes/receipt/" + operation)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Gateway receipt status=%d for operation %q", response.StatusCode, operation)
	}
	var receipt j17GatewayReceipt
	if err := json.NewDecoder(response.Body).Decode(&receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.OperationID != operation || receipt.State != "completed" || !receipt.CostKnown {
		t.Fatalf("Gateway receipt=%+v, want completed known-cost receipt for operation %q", receipt, operation)
	}
	return receipt
}

func readJ17MoneyProductObservation(t *testing.T, s *spine, operation, previousOperation string, responseStatus int) moneyflow.ProductObservation {
	t.Helper()
	var reservationID, reservationStatus string
	var actual, charged, overrun, remaining int64
	if err := s.db.QueryRow(`SELECT id::text,status,actual_amount,charged_amount,overrun_amount FROM business_account_credit_reservations WHERE id=$1`, operation).Scan(&reservationID, &reservationStatus, &actual, &charged, &overrun); err != nil {
		t.Fatal(err)
	}
	var spendable int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots WHERE business_account_id='account' AND status='open' AND settlement_state='settled' AND livemode=FALSE AND remaining>0`).Scan(&spendable); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots WHERE business_account_id='account'`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	var allocations, chargedCredits, usageCredits, usageEvents, ledgerRows int64
	for _, query := range []struct {
		sql       string
		operation bool
		value     *int64
	}{
		{`SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations WHERE operation_id=$1`, true, &allocations},
		{`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events WHERE operation_id=$1`, true, &chargedCredits},
		{`SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id='account'`, false, &usageCredits},
		{`SELECT COUNT(*) FROM business_account_usage_events WHERE operation_id=$1`, true, &usageEvents},
		{`SELECT COUNT(*) FROM funding_ledger_outbox`, false, &ledgerRows},
	} {
		var err error
		if query.operation {
			err = s.db.QueryRow(query.sql, operation).Scan(query.value)
		} else {
			err = s.db.QueryRow(query.sql).Scan(query.value)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	eligibility := moneyflow.ProductEligibilityUnavailable
	if spendable > 0 {
		eligibility = moneyflow.ProductEligibilitySpendable
	}
	reservation := moneyflow.ProductReservationNone
	charge := moneyflow.ProductChargeNone
	if reservationStatus == "finalized" {
		reservation = moneyflow.ProductReservationFinalized
	}
	if charged == 0 {
		charge = moneyflow.ProductChargeNone
	} else if charged <= actual && overrun == 0 {
		charge = moneyflow.ProductChargeWithin
	} else {
		charge = moneyflow.ProductChargeOver
	}
	wallet := moneyflow.ProductWalletNonnegative
	if remaining < 0 {
		wallet = moneyflow.ProductWalletNegative
	}
	replay := moneyflow.ProductReplayFresh
	switch {
	case previousOperation == "" && responseStatus == http.StatusOK && reservationID == operation:
		replay = moneyflow.ProductReplayFresh
	case previousOperation != "" && responseStatus == http.StatusConflict && reservationID == previousOperation:
		replay = moneyflow.ProductReplayReplayed
	default:
		t.Fatalf("cannot classify response as a fresh/replayed operation: prior=%q response_status=%d persisted_operation=%q", previousOperation, responseStatus, reservationID)
	}
	return moneyflow.ProductObservation{
		State: moneyflow.ProductState{Eligibility: eligibility, Reservation: reservation, Charge: charge, Wallet: wallet, Replay: replay},
		Effects: map[string]int64{
			"allocations": allocations, "chargedCredits": chargedCredits, "usageCredits": usageCredits,
			"usageEvents": usageEvents, "ledgerRows": ledgerRows,
		},
	}
}

type j17MoneyEffects struct {
	AccountReservations, AccountPending, AccountFinalized, AccountOverrunRows, AccountReleasedRows, AccountExpiredRows int64
	AccountReserved, AccountActual, AccountCharged, AccountOverrunAmount                                               int64
	RequestReservations, RequestFinalized, RequestCharged                                                              int64
	GlobalReservations, GlobalAllocations, AccountAllocations, RequestAllocations                                      int64
	GlobalAllocationCredits, AccountAllocationCredits, RequestAllocationCredits                                        int64
	GlobalUsageRows, AccountUsageRows                                                                                  int64
	GlobalUsageCredits, AccountUsageCredits                                                                            int64
	GlobalEventRows, AccountEventRows, RequestEventRows                                                                int64
	GlobalEventCredits, AccountEventCredits, RequestEventCredits                                                       int64
	GlobalLedgerRows, AccountLedgerRows                                                                                int64
	GlobalLedgerAmount, AccountLedgerAmount                                                                            int64
	GlobalKeyChargeRows, AccountKeyChargeRows, RequestKeyChargeRows                                                    int64
	GlobalKeyReserved, AccountKeyReserved, RequestKeyReserved                                                          int64
	GlobalKeyActual, AccountKeyActual, RequestKeyActual                                                                int64
	AccountLots, WalletRemaining                                                                                       int64
}

func snapshotJ17MoneyEffects(t *testing.T, s *spine, operation string) j17MoneyEffects {
	t.Helper()
	var got j17MoneyEffects
	scan := func(query string, args []any, value *int64) {
		t.Helper()
		var err error
		if len(args) == 0 {
			err = s.db.QueryRow(query).Scan(value)
		} else {
			err = s.db.QueryRow(query, args...).Scan(value)
		}
		if err != nil {
			t.Fatalf("snapshot J17 money query %q: %v", query, err)
		}
	}
	account := "business_account_id='account'"
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE `+account, nil, &got.AccountReservations)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE `+account+` AND status='pending'`, nil, &got.AccountPending)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE `+account+` AND status='finalized'`, nil, &got.AccountFinalized)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE `+account+` AND status='overrun'`, nil, &got.AccountOverrunRows)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE `+account+` AND status='released'`, nil, &got.AccountReleasedRows)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE `+account+` AND status='expired'`, nil, &got.AccountExpiredRows)
	scan(`SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE `+account, nil, &got.AccountReserved)
	scan(`SELECT COALESCE(SUM(actual_amount),0) FROM business_account_credit_reservations WHERE `+account, nil, &got.AccountActual)
	scan(`SELECT COALESCE(SUM(charged_amount),0) FROM business_account_credit_reservations WHERE `+account, nil, &got.AccountCharged)
	scan(`SELECT COALESCE(SUM(overrun_amount),0) FROM business_account_credit_reservations WHERE `+account, nil, &got.AccountOverrunAmount)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE id=$1`, []any{operation}, &got.RequestReservations)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE id=$1 AND status='finalized'`, []any{operation}, &got.RequestFinalized)
	scan(`SELECT COALESCE(SUM(charged_amount),0) FROM business_account_credit_reservations WHERE id=$1`, []any{operation}, &got.RequestCharged)
	scan(`SELECT COUNT(*) FROM business_account_credit_reservations`, nil, &got.GlobalReservations)
	scan(`SELECT COUNT(*) FROM credit_lot_allocations a JOIN credit_funding_lots l ON l.id=a.lot_id WHERE l.business_account_id='account'`, nil, &got.AccountAllocations)
	scan(`SELECT COALESCE(SUM(a.credits),0) FROM credit_lot_allocations a JOIN credit_funding_lots l ON l.id=a.lot_id WHERE l.business_account_id='account'`, nil, &got.AccountAllocationCredits)
	scan(`SELECT COUNT(*) FROM credit_lot_allocations`, nil, &got.GlobalAllocations)
	scan(`SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations`, nil, &got.GlobalAllocationCredits)
	scan(`SELECT COUNT(*) FROM credit_lot_allocations WHERE operation_id=$1`, []any{operation}, &got.RequestAllocations)
	scan(`SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations WHERE operation_id=$1`, []any{operation}, &got.RequestAllocationCredits)
	scan(`SELECT COUNT(*) FROM business_account_usage_records WHERE business_account_id='account'`, nil, &got.AccountUsageRows)
	scan(`SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id='account'`, nil, &got.AccountUsageCredits)
	scan(`SELECT COUNT(*) FROM business_account_usage_records`, nil, &got.GlobalUsageRows)
	scan(`SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records`, nil, &got.GlobalUsageCredits)
	scan(`SELECT COUNT(*) FROM business_account_usage_events WHERE business_account_id='account'`, nil, &got.AccountEventRows)
	scan(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events WHERE business_account_id='account'`, nil, &got.AccountEventCredits)
	scan(`SELECT COUNT(*) FROM business_account_usage_events`, nil, &got.GlobalEventRows)
	scan(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events`, nil, &got.GlobalEventCredits)
	scan(`SELECT COUNT(*) FROM business_account_usage_events WHERE operation_id=$1`, []any{operation}, &got.RequestEventRows)
	scan(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events WHERE operation_id=$1`, []any{operation}, &got.RequestEventCredits)
	scan(`SELECT COUNT(*) FROM funding_ledger_outbox o JOIN credit_funding_lots l ON l.id=o.lot_id WHERE l.business_account_id='account'`, nil, &got.AccountLedgerRows)
	scan(`SELECT COALESCE(SUM(o.amount_minor),0) FROM funding_ledger_outbox o JOIN credit_funding_lots l ON l.id=o.lot_id WHERE l.business_account_id='account'`, nil, &got.AccountLedgerAmount)
	scan(`SELECT COUNT(*) FROM funding_ledger_outbox`, nil, &got.GlobalLedgerRows)
	scan(`SELECT COALESCE(SUM(amount_minor),0) FROM funding_ledger_outbox`, nil, &got.GlobalLedgerAmount)
	scan(`SELECT COUNT(*) FROM ai_key_charges`, nil, &got.GlobalKeyChargeRows)
	scan(`SELECT COALESCE(SUM(reserved_credits),0) FROM ai_key_charges`, nil, &got.GlobalKeyReserved)
	scan(`SELECT COALESCE(SUM(actual_credits),0) FROM ai_key_charges`, nil, &got.GlobalKeyActual)
	scan(`SELECT COUNT(*) FROM ai_key_charges c JOIN ai_access_keys k ON k.id=c.key_id WHERE k.business_account_id='account'`, nil, &got.AccountKeyChargeRows)
	scan(`SELECT COALESCE(SUM(c.reserved_credits),0) FROM ai_key_charges c JOIN ai_access_keys k ON k.id=c.key_id WHERE k.business_account_id='account'`, nil, &got.AccountKeyReserved)
	scan(`SELECT COALESCE(SUM(c.actual_credits),0) FROM ai_key_charges c JOIN ai_access_keys k ON k.id=c.key_id WHERE k.business_account_id='account'`, nil, &got.AccountKeyActual)
	scan(`SELECT COUNT(*) FROM ai_key_charges WHERE operation_id=$1`, []any{operation}, &got.RequestKeyChargeRows)
	scan(`SELECT COALESCE(SUM(reserved_credits),0) FROM ai_key_charges WHERE operation_id=$1`, []any{operation}, &got.RequestKeyReserved)
	scan(`SELECT COALESCE(SUM(actual_credits),0) FROM ai_key_charges WHERE operation_id=$1`, []any{operation}, &got.RequestKeyActual)
	scan(`SELECT COUNT(*) FROM credit_funding_lots WHERE business_account_id='account'`, nil, &got.AccountLots)
	scan(`SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots WHERE business_account_id='account'`, nil, &got.WalletRemaining)
	return got
}
