package main

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// This integration boundary proves grant attribution and replay against the
// declarative Postgres schema. Spending is exercised through account-owned
// reservations and funding-lot allocations in internal/commerce, never an
// independent email-wallet debit API.
func TestStripeService_AccountFundingGrants(t *testing.T) {
	db := setupTestDB(t)
	service := ConfigureStripeServiceSimple(t, db)
	wallet := service.creditWallet
	const account = "acct-grants"
	const email = "grants@example.com"
	metadata := map[string]interface{}{"livemode": false, "charge_id": "ch_grant", "plan_tier": "pro"}
	grant := func(event string, amount int64) error {
		return wallet.AddCreditsForBusinessAccount(account, email, amount, "credit_topup", "stripe", event, metadata)
	}
	require.NoError(t, grant("evt_grant_1", 1000))
	require.NoError(t, grant("evt_grant_2", 500))
	require.NoError(t, grant("evt_grant_1", 1000))
	require.Error(t, grant("evt_grant_1", 1001), "an event replay must not silently change amount")
	require.Error(t, wallet.AddCreditsForBusinessAccount("other-account", email, 1000, "credit_topup", "stripe", "evt_grant_1", metadata))
	balance, err := wallet.BalanceForBusinessAccount(account, "GRANTS@EXAMPLE.COM")
	require.NoError(t, err)
	require.Equal(t, int64(1500), balance)
	var lots, transactions, legacy int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots WHERE business_account_id=$1 AND provider='stripe' AND livemode=FALSE AND provider_ref IN ('evt_grant_1','evt_grant_2') AND provider_ref2='ch_grant'`, account).Scan(&lots))
	require.Equal(t, 2, lots)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_transactions WHERE business_account_id=$1 AND customer_email=$2 AND metadata->>'plan_tier'='pro' AND lot_id IS NOT NULL`, account, email).Scan(&transactions))
	require.Equal(t, 2, transactions)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM credit_wallets WHERE customer_email=$1`, email).Scan(&legacy))
	require.Zero(t, legacy, "account grants must never recreate an email wallet")
	absent, err := wallet.BalanceForBusinessAccount("absent", email)
	require.NoError(t, err)
	require.Zero(t, absent)
	_, err = wallet.BalanceForBusinessAccount("", email)
	require.Error(t, err)
	for _, amount := range []int64{0, -1} {
		require.Error(t, grant(fmt.Sprintf("invalid-%d", amount), amount))
	}
	require.Error(t, wallet.AddCreditsForBusinessAccount(account, "", 1, "credit_topup", "stripe", "empty-customer", metadata))
	require.Error(t, wallet.AddCreditsForBusinessAccount(account, email, 1, "credit_topup", "stripe", "unknown-mode", nil))
}

func TestStripeService_ConcurrentAccountFundingGrants(t *testing.T) {
	db := setupTestDB(t)
	service := ConfigureStripeServiceSimple(t, db)
	const attempts = 10
	const account = "acct-concurrent-grants"
	const email = "concurrent-grants@example.com"
	grant := func(event string) error {
		return service.creditWallet.AddCreditsForBusinessAccount(account, email, 100, "credit_topup", "stripe", event, map[string]interface{}{"livemode": false})
	}
	for _, sameEvent := range []bool{true, false} {
		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make(chan error, attempts)
		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				event := "evt_same"
				if !sameEvent {
					event = fmt.Sprintf("evt_unique_%d", i)
				}
				errs <- grant(event)
			}(i)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
	}
	balance, err := service.creditWallet.BalanceForBusinessAccount(account, email)
	require.NoError(t, err)
	require.Equal(t, int64((attempts+1)*100), balance, "same-event races grant once; independent events accumulate")
	var transactions, lots int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_transactions WHERE business_account_id=$1`, account).Scan(&transactions))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots WHERE business_account_id=$1`, account).Scan(&lots))
	require.Equal(t, attempts+1, transactions)
	require.Equal(t, transactions, lots, "every committed account grant has exactly one source-attributed lot")
}
