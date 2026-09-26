package orchestration

import (
	"context"
	"errors"
	"strings"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// ContinuationAccepted reads the original owner admission without resending it.
// A missing/expired or pending receipt is not proof that no effect occurred.
func (o *Orchestrator) ContinuationAccepted(ctx context.Context, id uuid.UUID, message, key string) (bool, error) {
	if id == uuid.Nil || strings.TrimSpace(key) == "" || strings.TrimSpace(message) == "" || o.idempotency == nil {
		return false, errors.New("continuation receipt owner unavailable")
	}
	receipt, err := o.idempotency.Check(ctx, key)
	if err != nil || receipt == nil {
		return false, err
	}
	if receipt.Status != domain.IdempotencyStatusComplete {
		return false, nil
	}
	request := ContinueRunRequest{RunID: id, Message: message, IdempotencyKey: key}
	if receipt.EntityType != "Continuation" || receipt.EntityID == nil || *receipt.EntityID != id || string(receipt.Response) != string(continuationRequestHash(request)) {
		return false, errors.New("continuation receipt does not match the requested run")
	}
	return true, nil
}
