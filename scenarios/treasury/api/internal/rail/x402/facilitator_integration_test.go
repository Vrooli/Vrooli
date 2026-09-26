package x402

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// [REQ:TRS-P1-002] A facilitator-confirmed inbound payment atomically queues
// one authoritative positive Money Ledger event.
func TestInboundSettlementQueuesOneInflow(t *testing.T) {
	gate, handle, _, price := newInboundFixture(t)
	challenge, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	payment := paymentForChallenge(t, challenge, "inflow-nonce")
	admission, err := gate.Admit(context.Background(), price.ID, payment)
	require.NoError(t, err)

	var count int
	var amount int64
	var basis, externalID string
	err = handle.QueryRowContext(context.Background(), `SELECT COUNT(*),amount_minor,basis,external_id FROM ledger_emissions WHERE settlement_id=?`, admission.ID).Scan(&count, &amount, &basis, &externalID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.EqualValues(t, 1, amount)
	require.Equal(t, "authoritative", basis)
	require.Equal(t, "treasury-x402-inflow:"+admission.TransactionID, externalID)

	_, err = gate.Admit(context.Background(), price.ID, payment)
	require.NoError(t, err)
	require.NoError(t, handle.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ledger_emissions WHERE settlement_id=?`, admission.ID).Scan(&count))
	require.Equal(t, 1, count)
}
