// This file implements user-requested actions on existing runs.
package orchestration

import (
	"context"
	"net/url"
	"strings"

	"agent-manager/internal/domain"
)

// webConsoleSessionURL builds the run-detail deep link to the live web-console
// session for an interactive run. Returns "" when the run is not interactive,
// carries no session id, or the web-console UI base was not resolved at wiring
// time. web-console has no per-session route today; the session surfaces in the
// sidebar's Programmatic tab by its display label, so the link targets the UI
// base with the session id as a forward-compatible reference param.
func (o *Orchestrator) webConsoleSessionURL(run *domain.Run) string {
	if o.webConsoleUIBase == "" || run == nil {
		return ""
	}
	if run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.WebConsoleSessionID == "" {
		return ""
	}
	base := strings.TrimRight(o.webConsoleUIBase, "/")
	return base + "/?session=" + url.QueryEscape(run.WebConsoleSessionID)
}

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
	run.WebConsoleSessionURL = o.webConsoleSessionURL(run)
	return run
}

func (o *Orchestrator) attachRunActionsList(ctx context.Context, runs []*domain.Run) []*domain.Run {
	if len(runs) == 0 {
		return runs
	}
	actx := o.runActionContext(ctx)
	for _, run := range runs {
		if run == nil {
			continue
		}
		actions := domain.RunActionsFor(run, actx)
		run.Actions = &actions
		run.WebConsoleSessionURL = o.webConsoleSessionURL(run)
	}
	return runs
}
