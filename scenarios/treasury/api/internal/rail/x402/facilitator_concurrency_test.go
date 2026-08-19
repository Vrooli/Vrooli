package x402

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// [REQ:TRS-P1-002] Concurrent inbound payers retain one admission and one
// outbox event each without SQLite lock-timeout failures. The elapsed value is
// an observation for the declared migration trigger, not a pass threshold.
func TestConcurrentInboundPayersMeasureSQLiteContention(t *testing.T) {
	gate, handle, facilitator, price := newInboundFixture(t)
	facilitator.delay = time.Millisecond
	challenge, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)

	const payers = 32
	started := time.Now()
	errs := make(chan error, payers)
	var group sync.WaitGroup
	for index := range payers {
		group.Add(1)
		go func() {
			defer group.Done()
			_, admitErr := gate.Admit(context.Background(), price.ID, paymentForChallenge(t, challenge, fmt.Sprintf("payer-%d", index)))
			errs <- admitErr
		}()
	}
	group.Wait()
	close(errs)
	lockFailures := 0
	for err := range errs {
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "lock") {
			lockFailures++
		}
		require.NoError(t, err)
	}
	elapsed := time.Since(started)
	var admissions, emissions int
	require.NoError(t, handle.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM x402_inbound_admissions WHERE status='settled'`).Scan(&admissions))
	require.NoError(t, handle.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ledger_emissions WHERE adapter_id='treasury-x402'`).Scan(&emissions))
	require.Equal(t, payers, admissions)
	require.Equal(t, payers, emissions)
	require.Equal(t, 0, lockFailures)
	t.Logf("x402 inbound SQLite observation: payers=%d elapsed=%s lock_timeout_failures=%d", payers, elapsed, lockFailures)
}
