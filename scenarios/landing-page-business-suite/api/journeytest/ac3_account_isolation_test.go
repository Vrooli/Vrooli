package journeytest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/aidoor"
)

// TestAC3_ConcurrentAccountsKeepWalletReservationAndReceiptReadsScoped drives
// two independently issued account keys through the real LPBS door at once.
// It checks both persisted ownership and the public receipt read boundary.
func TestAC3_ConcurrentAccountsKeepWalletReservationAndReceiptReadsScoped(t *testing.T) {
	s := newSpine(t)
	for _, seed := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO business_accounts(id,display_name,billing_email) VALUES('account-b','Journey B','actor-b@example.invalid')`, nil},
		{`INSERT INTO business_account_members(business_account_id,user_id,role) VALUES('account-b','actor-b','owner')`, nil},
		{`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES(gen_random_uuid(),'account-b','paid','stripe','fixture-payment-b',FALSE,'settled',10000,10000,'open',$1)`, []any{time.Now().UTC()}},
	} {
		if _, err := s.db.Exec(seed.query, seed.args...); err != nil {
			t.Fatal(err)
		}
	}
	keyB, err := (aidoor.KeyStore{DB: s.db}).IssueDevice(context.Background(), "account-b", "actor-b", "web-console", 10000)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		key       aidoor.Key
		operation string
		status    int
	}
	results := make(chan result, 2)
	var group sync.WaitGroup
	for _, key := range []aidoor.Key{s.key, keyB} {
		group.Add(1)
		go func(key aidoor.Key) {
			defer group.Done()
			body := `{"model":"fixture-model","stream":true,"messages":[{"role":"user","content":"parallel"}]}`
			req, requestErr := http.NewRequest(http.MethodPost, s.server.URL+"/api/v1/ai/chat/completions", strings.NewReader(body))
			if requestErr != nil {
				t.Errorf("build request for %s: %v", key.Account, requestErr)
				return
			}
			req.Header.Set("Authorization", "Bearer "+key.Token)
			req.Header.Set("Idempotency-Key", "same-client-operation")
			req.Header.Set("Content-Type", "application/json")
			response, requestErr := http.DefaultClient.Do(req)
			if requestErr != nil {
				t.Errorf("request for %s: %v", key.Account, requestErr)
				return
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if readErr != nil {
				t.Errorf("read response for %s: %v", key.Account, readErr)
				return
			}
			results <- result{key: key, operation: response.Header.Get("X-Operation-ID"), status: response.StatusCode}
		}(key)
	}
	group.Wait()
	close(results)

	byAccount := map[string]result{}
	for got := range results {
		if got.status != http.StatusOK || got.operation == "" {
			t.Fatalf("account %s response status=%d operation=%q", got.key.Account, got.status, got.operation)
		}
		byAccount[got.key.Account] = got
	}
	if len(byAccount) != 2 || byAccount["account"].operation == byAccount["account-b"].operation {
		t.Fatalf("concurrent account operations were not isolated: %+v", byAccount)
	}

	for account, wantActor := range map[string]string{"account": "actor", "account-b": "actor-b"} {
		got := byAccount[account]
		var reservationAccount, reservationActor, state string
		var reservationAmount int64
		if err := s.db.QueryRow(`SELECT business_account_id,actor_identity,status,reserved_amount FROM business_account_credit_reservations WHERE id=$1`, got.operation).Scan(&reservationAccount, &reservationActor, &state, &reservationAmount); err != nil {
			t.Fatalf("reservation for %s: %v", account, err)
		}
		if reservationAccount != account || reservationActor != wantActor || state != "finalized" {
			t.Errorf("reservation for %s belongs to account=%s actor=%s state=%s", account, reservationAccount, reservationActor, state)
		}
		var eventAccount, eventActor string
		var eventRows, charged int64
		if err := s.db.QueryRow(`SELECT business_account_id,actor_identity,COUNT(*),COALESCE(SUM(credits),0) FROM business_account_usage_events WHERE operation_id=$1 GROUP BY business_account_id,actor_identity`, got.operation).Scan(&eventAccount, &eventActor, &eventRows, &charged); err != nil {
			t.Fatalf("wallet event for %s: %v", account, err)
		}
		if eventAccount != account || eventActor != wantActor || eventRows != 1 || charged != 200 {
			t.Errorf("wallet event for %s belongs to account=%s actor=%s rows=%d charged=%d", account, eventAccount, eventActor, eventRows, charged)
		}
		var chargeKey string
		var reserved int64
		var actual int64
		if err := s.db.QueryRow(`SELECT key_id,reserved_credits,actual_credits FROM ai_key_charges WHERE operation_id=$1`, got.operation).Scan(&chargeKey, &reserved, &actual); err != nil {
			t.Fatalf("key recovery ledger for %s: %v", account, err)
		}
		if chargeKey != got.key.ID || reserved != reservationAmount || actual != 200 {
			t.Errorf("key recovery ledger for %s belongs to key=%s reserved=%d (reservation=%d) actual=%d", account, chargeKey, reserved, reservationAmount, actual)
		}
		var ownRemaining int64
		if err := s.db.QueryRow(`SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots WHERE business_account_id=$1`, account).Scan(&ownRemaining); err != nil {
			t.Fatal(err)
		}
		if ownRemaining != 9800 {
			t.Errorf("wallet for %s remaining=%d want=9800", account, ownRemaining)
		}
	}

	for _, account := range []string{"account", "account-b"} {
		caller := byAccount[account]
		for _, targetAccount := range []string{account, map[string]string{"account": "account-b", "account-b": "account"}[account]} {
			target := byAccount[targetAccount]
			req, err := http.NewRequest(http.MethodGet, s.server.URL+"/api/v1/ai/receipt", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+caller.key.Token)
			idempotencyKey := "same-client-operation"
			if targetAccount != account {
				// A different account cannot address this operation by replaying
				// its public operation ID as an idempotency key.
				idempotencyKey = target.operation
			}
			req.Header.Set("Idempotency-Key", idempotencyKey)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			var receipt struct {
				Status         string `json:"status"`
				ChargedCredits int64  `json:"charged_credits"`
			}
			_ = json.NewDecoder(response.Body).Decode(&receipt)
			response.Body.Close()
			want := http.StatusNotFound
			if targetAccount == account {
				want = http.StatusOK
				if receipt.Status != "settled" || receipt.ChargedCredits != 200 {
					t.Errorf("own receipt status=%+v", receipt)
				}
			}
			if response.StatusCode != want {
				t.Errorf("account %s addressing %s receipt got HTTP %d, want %d", caller.key.Account, target.key.Account, response.StatusCode, want)
			}
		}
	}
}
