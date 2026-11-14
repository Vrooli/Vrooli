package serviceoffer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func sandboxTestOffer(t *testing.T) Record {
	t.Helper()
	offer, _, err := Prototype()
	require.NoError(t, err)
	return offer
}

func TestSandboxOrderFreezesStableIdentityAndDoorBeforeProviderCalls(t *testing.T) {
	offer := sandboxTestOffer(t)
	order, err := NewSandboxOrder("ord-test-01", "job-test-01", offer, DoorMCP)
	require.NoError(t, err)
	require.Equal(t, "ord-test-01", order.OrderID)
	require.Equal(t, "job-test-01", order.JobID)
	require.Equal(t, offer.ID, order.OfferID)
	require.Equal(t, offer.Version, order.OfferVersion)
	require.Equal(t, DoorMCP, order.Door)
	require.Equal(t, CheckoutModeSandbox, order.Mode)
	require.Equal(t, offer.ProposedPrice.AmountMinor, order.AmountMinor)
	require.Equal(t, offer.ProposedPrice.Currency, order.Currency)
	require.Equal(t, OrderCreated, order.State)
	require.Empty(t, order.AuthorizationID)
	require.Empty(t, order.CaptureID)
	require.Empty(t, order.LedgerPostingID)
}

func TestSandboxOrderAcceptsOnlyTheFourDeclaredPurchaseDoors(t *testing.T) {
	offer := sandboxTestOffer(t)
	for _, door := range []string{DoorBrowser, DoorBrowserAgent, DoorMCP, DoorAgentManager} {
		_, err := NewSandboxOrder("ord-test-01", "job-test-01", offer, door)
		require.NoError(t, err, "door %q", door)
	}
	_, err := NewSandboxOrder("ord-test-01", "job-test-01", offer, " ")
	require.ErrorContains(t, err, "purchase door is required")
	for _, door := range []string{"plugin", "x402", "mpp_x402", "unknown"} {
		_, err := NewSandboxOrder("ord-test-01", "job-test-01", offer, door)
		require.ErrorContains(t, err, "unsupported sandbox purchase door")
	}
}

func TestSandboxContractRefusesCaptureBeforeSuccessfulDeliveryVerification(t *testing.T) {
	order := newSandboxOrder(t)
	_, err := ApplyOrderEvent(order, OrderEvent{EventID: "capture-1", Kind: EventCaptureRequested})
	require.ErrorContains(t, err, "while order is created")

	order = apply(t, order, OrderEvent{EventID: "auth-1", Kind: EventAuthorized, ReceiptID: "sandbox-auth-1"})
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "verify-without-receipt", Kind: EventVerified})
	require.ErrorContains(t, err, "receipt_id is required")
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "capture-2", Kind: EventCaptureRequested})
	require.ErrorContains(t, err, "while order is authorized")

	order = apply(t, order, OrderEvent{EventID: "verify-failed-1", Kind: EventVerificationFailed, ReceiptID: "delivery-proof-failed-1"})
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "capture-3", Kind: EventCaptureRequested})
	require.ErrorContains(t, err, "while order is verification_failed")
	order = apply(t, order, OrderEvent{EventID: "void-1", Kind: EventVoided, ReceiptID: "sandbox-void-1"})
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "capture-4", Kind: EventCaptureRequested})
	require.ErrorContains(t, err, "while order is voided")
}

func TestSandboxContractReplaysProviderCallbacksAndLedgerReceiptIdempotently(t *testing.T) {
	order := newSandboxOrder(t)
	events := []OrderEvent{
		{EventID: "auth-event-1", Kind: EventAuthorized, ReceiptID: "auth-sandbox-1"},
		{EventID: "delivery-check-1", Kind: EventVerified, ReceiptID: "delivery-proof-1"},
		{EventID: "capture-request-1", Kind: EventCaptureRequested},
		{EventID: "capture-callback-1", Kind: EventCaptured, ReceiptID: "capture-sandbox-1"},
		{EventID: "ledger-outbox-1", Kind: EventLedgerPending},
		{EventID: "ledger-ack-1", Kind: EventReconciled, ReceiptID: "posting-test-1"},
	}
	for _, event := range events {
		order = apply(t, order, event)
	}
	require.Equal(t, OrderReconciled, order.State)
	require.Equal(t, "capture-sandbox-1", order.CaptureID)
	require.Equal(t, "delivery-proof-1", order.VerificationID)
	require.Equal(t, "posting-test-1", order.LedgerPostingID)

	// A duplicate provider callback or acknowledgement returns the already
	// persisted state without creating another transition.
	duplicate, err := ApplyOrderEvent(order, events[3])
	require.NoError(t, err)
	require.Equal(t, len(order.Events), len(duplicate.Events))
	require.Equal(t, order, duplicate)
	duplicate, err = ApplyOrderEvent(order, events[5])
	require.NoError(t, err)
	require.Equal(t, order, duplicate)

	// Reusing the callback identity for a different result is not a new event.
	conflict := events[3]
	conflict.ReceiptID = "different-capture"
	_, err = ApplyOrderEvent(order, conflict)
	require.ErrorContains(t, err, "replayed with a different payload")

	// The append-only order can be serialized and folded back to the same state.
	data, err := json.Marshal(order)
	require.NoError(t, err)
	var restored SandboxOrder
	require.NoError(t, json.Unmarshal(data, &restored))
	require.Equal(t, order, restored)
	replayed := newSandboxOrder(t)
	for _, event := range restored.Events {
		replayed = apply(t, replayed, event)
	}
	require.Equal(t, restored, replayed)
}

func TestSandboxContractRequiresCaptureBeforeLedgerReconciliation(t *testing.T) {
	order := newSandboxOrder(t)
	_, err := ApplyOrderEvent(order, OrderEvent{EventID: "ledger-1", Kind: EventReconciled, ReceiptID: "posting-test-1"})
	require.ErrorContains(t, err, "while order is created")

	order = apply(t, order, OrderEvent{EventID: "auth-2", Kind: EventAuthorized, ReceiptID: "auth-sandbox-2"})
	order = apply(t, order, OrderEvent{EventID: "verify-2", Kind: EventVerified, ReceiptID: "delivery-proof-2"})
	order = apply(t, order, OrderEvent{EventID: "capture-request-2", Kind: EventCaptureRequested})
	order = apply(t, order, OrderEvent{EventID: "capture-2", Kind: EventCaptured, ReceiptID: "capture-sandbox-2"})
	order = apply(t, order, OrderEvent{EventID: "ledger-pending-2", Kind: EventLedgerPending})
	order = apply(t, order, OrderEvent{EventID: "ledger-ack-2", Kind: EventReconciled, ReceiptID: "posting-test-2"})
	require.Equal(t, OrderReconciled, order.State)

	// Once captured, a provider authorization cannot be voided as if it were
	// still only a hold; a refund/reversal is a separate owner contract.
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "void-after-capture", Kind: EventVoided, ReceiptID: "void-2"})
	require.ErrorContains(t, err, "while order is reconciled")
}

func TestSandboxContractRejectsMissingOrConflictingIdentities(t *testing.T) {
	offer := sandboxTestOffer(t)
	_, err := NewSandboxOrder("", "job", offer, "browser")
	require.ErrorContains(t, err, "order_id and job_id are required")
	_, err = NewSandboxOrder("order", "job", offer, " ")
	require.ErrorContains(t, err, "purchase door is required")

	order := newSandboxOrder(t)
	_, err = ApplyOrderEvent(order, OrderEvent{Kind: EventAuthorized, ReceiptID: "auth"})
	require.ErrorContains(t, err, "event_id is required")
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "auth", Kind: EventAuthorized})
	require.ErrorContains(t, err, "receipt_id is required")

	order.Mode = "live"
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "auth-live", Kind: EventAuthorized, ReceiptID: "auth"})
	require.ErrorContains(t, err, "only sandbox orders")

	order = newSandboxOrder(t)
	order.AmountMinor++
	_, err = ApplyOrderEvent(order, OrderEvent{EventID: "auth-changed-terms", Kind: EventAuthorized, ReceiptID: "auth"})
	require.ErrorContains(t, err, "immutable terms changed")
}

func TestSandboxContractUsesStableOperationAndLedgerIdentities(t *testing.T) {
	order, err := ProviderIdempotencyKey("ord-test-01", ProviderAuthorize)
	require.NoError(t, err)
	require.Equal(t, "service-order:ord-test-01:authorize", order)
	retried, err := ProviderIdempotencyKey("ord-test-01", ProviderAuthorize)
	require.NoError(t, err)
	require.Equal(t, order, retried, "authorization retries keep the original provider key")

	capture, err := ProviderIdempotencyKey("ord-test-01", ProviderCapture)
	require.NoError(t, err)
	require.Equal(t, "service-order:ord-test-01:capture", capture)
	void, err := ProviderIdempotencyKey("ord-test-01", ProviderVoid)
	require.NoError(t, err)
	require.NotEqual(t, capture, void)
	_, err = ProviderIdempotencyKey("ord-test-01", "refund")
	require.ErrorContains(t, err, "unsupported provider operation")
	_, err = ProviderIdempotencyKey(" ", ProviderCapture)
	require.ErrorContains(t, err, "order_id is required")

	externalID, err := LedgerExternalID("ord-test-01", "capture-1")
	require.NoError(t, err)
	require.Equal(t, "service-order:ord-test-01:capture:capture-1", externalID)
	replayID, err := LedgerExternalID("ord-test-01", "capture-1")
	require.NoError(t, err)
	require.Equal(t, externalID, replayID, "ledger outbox retries keep the original external identity")
	_, err = LedgerExternalID("ord-test-01", "capture-2")
	require.NoError(t, err)
	_, err = LedgerExternalID("ord-test-01", " ")
	require.ErrorContains(t, err, "order_id and capture_id are required")
}

func newSandboxOrder(t *testing.T) SandboxOrder {
	t.Helper()
	order, err := NewSandboxOrder("ord-test-01", "job-test-01", sandboxTestOffer(t), "browser")
	require.NoError(t, err)
	return order
}

func apply(t *testing.T, order SandboxOrder, event OrderEvent) SandboxOrder {
	t.Helper()
	next, err := ApplyOrderEvent(order, event)
	require.NoError(t, err)
	return next
}
