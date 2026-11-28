package duplex

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type accountCreditReserveRequest struct {
	budgetMicros int64
	model        string
	provider     string
	userBearer   string
	ownerKey     string
}

func newAccountCreditReserveRequest(ctx context.Context, budgetMicros uint64, model, provider string) (accountCreditReserveRequest, error) {
	if ctx == nil {
		return accountCreditReserveRequest{}, errors.New("duplex: nil account-credit reservation context")
	}
	if budgetMicros == 0 || budgetMicros > uint64(^uint64(0)>>1) || strings.TrimSpace(model) == "" || strings.TrimSpace(provider) == "" {
		return accountCreditReserveRequest{}, errors.New("duplex: valid USD budget, model, and provider are required")
	}
	userToken, hasUser := humanBearer(ctx)
	if !hasUser {
		return accountCreditReserveRequest{}, ErrHumanBearerRequired
	}
	ownerKey, err := verifiedHumanOwnerKey(ctx)
	if err != nil {
		return accountCreditReserveRequest{}, err
	}
	return accountCreditReserveRequest{
		budgetMicros: int64(budgetMicros),
		model:        strings.TrimSpace(model),
		provider:     strings.TrimSpace(provider),
		userBearer:   userToken,
		ownerKey:     ownerKey,
	}, nil
}

func (request accountCreditReserveRequest) reservation(client *AccountCreditClient, operationID string) *AccountCreditReservation {
	return &AccountCreditReservation{
		client:       client,
		operationID:  operationID,
		budgetMicros: request.budgetMicros,
		model:        request.model,
		provider:     request.provider,
		userBearer:   request.userBearer,
		ownerKey:     request.ownerKey,
		journal:      client.journal,
	}
}

func (r *AccountCreditReservation) createJournalIntent(ctx context.Context) error {
	if r.journal == nil {
		return nil
	}
	if err := r.journal.Create(ctx, JournalOperation{
		OperationID:  r.operationID,
		OwnerKey:     r.ownerKey,
		ProviderID:   r.provider,
		ModelID:      r.model,
		BudgetMicros: r.budgetMicros,
	}); err != nil {
		return err
	}
	r.journalPhase = "reserve_pending"
	return nil
}

func (c *AccountCreditClient) createAccountCreditHold(ctx context.Context, operationID string, budgetMicros int64) error {
	var response lifecycleReservationResponse
	if err := c.call(ctx, http.MethodPost, accountCreditUSDV1ReservePath, map[string]any{
		"operation_id":  operationID,
		"budget_micros": budgetMicros,
	}, &response); err != nil {
		return err
	}
	if response.ReservationID != operationID || !response.Created || response.Phase != "reserved" || response.ContractVersion != 1 || response.BudgetMicros != budgetMicros || response.ReservedAmount <= 0 || response.CostMultiplier <= 0 {
		// Replays and ambiguous/non-reserved responses must never authorize a
		// second dispatch. The original durable hold remains owner-managed.
		return errors.New("duplex: LPBS did not create a fresh reserved account-credit hold")
	}
	return nil
}
