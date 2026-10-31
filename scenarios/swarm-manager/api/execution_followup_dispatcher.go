package main

import (
	"context"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

type executionFollowUpDispatcher struct{ service *execution.Service }

func (d executionFollowUpDispatcher) DispatchFollowUp(ctx context.Context, kind backlog.BacklogKind, name, steering string) error {
	_, err := d.service.FollowUpLatestForBacklog(ctx, string(kind), name, steering)
	return err
}
