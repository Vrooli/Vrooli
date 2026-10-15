// This file implements user-requested actions on existing runs.
package orchestration

import (
	"context"
	"net/url"
	"strings"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/discovery"
)

// resolveWebConsoleUIBase resolves the browser-facing web-console base URL used
// to build links shown to, or sent to, the operator.
//
// An explicit override (set via WithWebConsoleUIBase — tests, or an
// operator-pinned WEB_CONSOLE_UI_URL) wins. Otherwise the canonical resolver
// derives it so every agent-manager link agrees with the rest of the platform:
// from the request origin when one is present (a tunnel subdomain on mobile,
// localhost on the same machine), and from Tunnel Manager's public URL for a
// background caller that carries no origin — exactly the case for the approval
// and run-outcome notifications, which a localhost port would strand on a phone.
// Returns "" when nothing resolves, which callers surface as an omitted link.
// resolveExternalURL / resolveExternalURLForNotice are seams over the discovery
// resolvers so tests can drive the no-base path deterministically without a live
// discovery/tunnel.
var (
	resolveExternalURL          = discovery.ResolveExternalURL
	resolveExternalURLForNotice = discovery.ResolveExternalURLForNotice
)

// resolveWebConsoleUIBase resolves the base for a link shown in a live browser
// context (the interactive-run session link). It is request-cheap: it uses the
// request origin when present and a localhost port otherwise, and never calls
// Tunnel Manager — this runs on the hot run get/list path.
func (o *Orchestrator) resolveWebConsoleUIBase(ctx context.Context) string {
	if o.webConsoleUIBase != "" {
		return o.webConsoleUIBase
	}
	base, err := resolveExternalURL(ctx, "web-console", discovery.ExternalHostFromContext(ctx))
	if err != nil {
		return ""
	}
	return base
}

// resolveWebConsoleNoticeBase resolves the base for a link sent to the operator
// out of band (the approval and run-outcome notifications). These carry no
// browser origin and may reach a phone, so it prefers web-console's public
// tunnel URL, falling back to localhost. It may make a Tunnel Manager call and
// so is used only on the infrequent run-end notification path.
func (o *Orchestrator) resolveWebConsoleNoticeBase(ctx context.Context) string {
	if o.webConsoleUIBase != "" {
		return o.webConsoleUIBase
	}
	base, err := resolveExternalURLForNotice(ctx, "web-console")
	if err != nil {
		return ""
	}
	return base
}

// sessionURLForBase builds the run-detail deep link to the live web-console
// session for an interactive run. Returns "" when base is empty, or the run is
// not interactive / carries no session id. web-console has no per-session route
// today; the session surfaces in the sidebar's Programmatic tab by its display
// label, so the link targets the UI base with the session id as a
// forward-compatible reference param.
func sessionURLForBase(base string, run *domain.Run) string {
	if base == "" || run == nil {
		return ""
	}
	if run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.WebConsoleSessionID == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/?session=" + url.QueryEscape(run.WebConsoleSessionID)
}

func (o *Orchestrator) webConsoleSessionURL(ctx context.Context, run *domain.Run) string {
	return sessionURLForBase(o.resolveWebConsoleUIBase(ctx), run)
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
	run.WebConsoleSessionURL = o.webConsoleSessionURL(ctx, run)
	return run
}

func (o *Orchestrator) attachRunActionsList(ctx context.Context, runs []*domain.Run) []*domain.Run {
	if len(runs) == 0 {
		return runs
	}
	actx := o.runActionContext(ctx)
	// Resolve the UI base once for the whole list rather than per run.
	base := o.resolveWebConsoleUIBase(ctx)
	for _, run := range runs {
		if run == nil {
			continue
		}
		actions := domain.RunActionsFor(run, actx)
		run.Actions = &actions
		run.WebConsoleSessionURL = sessionURLForBase(base, run)
	}
	return runs
}
