package wiring

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
)

type supervisionRunController struct{ orchestrator *orchestration.Orchestrator }

func (c supervisionRunController) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	return c.orchestrator.GetRun(ctx, id)
}

func (c supervisionRunController) ContinueRun(ctx context.Context, id uuid.UUID, message, idempotencyKey string) error {
	_, err := c.orchestrator.ContinueRun(ctx, orchestration.ContinueRunRequest{RunID: id, Message: message, IdempotencyKey: idempotencyKey})
	return err
}

func (c supervisionRunController) StopRun(ctx context.Context, id uuid.UUID) error {
	return c.orchestrator.StopRun(ctx, id)
}

func (c supervisionRunController) ParkRun(ctx context.Context, id uuid.UUID, watchID string, deadline *time.Time) error {
	_, err := c.orchestrator.ParkRun(ctx, orchestration.ParkRunInput{RunID: id, Producer: orchestration.ProducerSupervision, Key: watchID, Deadline: deadline})
	return err
}

func (c supervisionRunController) WakeRun(ctx context.Context, id uuid.UUID, result string) error {
	_, err := c.orchestrator.WakeRun(ctx, orchestration.WakeRunInput{RunID: id, Result: result})
	return err
}
