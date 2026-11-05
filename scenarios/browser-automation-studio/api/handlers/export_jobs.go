package handlers

import (
	"context"
	"github.com/google/uuid"
)

// DrainExports must precede shutdown of the driver and repository.
func (h *Handler) DrainExports(ctx context.Context) error {
	if h.exportJobs == nil {
		return nil
	}
	return h.exportJobs.Drain(ctx)
}

// CancelExport acknowledges cancellation of an active owned job. The worker
// persists its terminal result; acknowledgement is not a completed cancellation.
func (h *Handler) CancelExport(id uuid.UUID) bool {
	return h.exportJobs != nil && h.exportJobs.Cancel(id)
}
