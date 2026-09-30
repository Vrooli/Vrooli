package entitlements

import "context"

// BillingAdapter is intentionally provider-neutral. A future billing
// connector must verify signatures/server state before handing events here;
// food-domain backups never receive billing identifiers.
type BillingAdapter interface {
	VerifyAndNormalize(context.Context, []byte) (BillingEvent, error)
}

type BillingEvent struct {
	ID, WorkspaceID string
	Event           Event
	Verified        bool
}

type DisabledBilling struct{}

func (DisabledBilling) VerifyAndNormalize(context.Context, []byte) (BillingEvent, error) {
	return BillingEvent{}, ErrBillingUnavailable
}

var ErrBillingUnavailable = &billingUnavailable{}

type billingUnavailable struct{}

func (*billingUnavailable) Error() string { return "billing provider is not configured" }

// ApplyVerifiedEvent is the only server entry point for billing state. The
// repository's version guard makes retries and out-of-order deliveries safe.
func ApplyVerifiedEvent(ctx context.Context, repo Repository, event BillingEvent) (State, error) {
	if !event.Verified || event.WorkspaceID == "" || event.Event.ID == "" {
		return State{}, ErrUnverifiedBillingEvent
	}
	return repo.ApplyEvent(ctx, event.WorkspaceID, event.Event)
}

var ErrUnverifiedBillingEvent = &unverifiedBillingEvent{}

type unverifiedBillingEvent struct{}

func (*unverifiedBillingEvent) Error() string {
	return "billing event must be verified before application"
}
