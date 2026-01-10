package orchestration

import (
	"context"

	"agent-manager/internal/domain"
)

func (o *Orchestrator) runActionContext(ctx context.Context) domain.RunActionContext {
	return domain.RunActionContext{
		InvestigationTagAllowlist: o.investigationTagAllowlist(ctx),
	}
}

func (o *Orchestrator) runActionsFor(ctx context.Context, run *domain.Run) domain.RunActions {
	return domain.RunActionsFor(run, o.runActionContext(ctx))
}

func (o *Orchestrator) attachRunActions(ctx context.Context, run *domain.Run) *domain.Run {
	if run == nil {
		return nil
	}
	actions := o.runActionsFor(ctx, run)
	run.Actions = &actions
	return run
}

func (o *Orchestrator) attachRunActionsList(ctx context.Context, runs []*domain.Run) []*domain.Run {
	for _, run := range runs {
		o.attachRunActions(ctx, run)
	}
	return runs
}
