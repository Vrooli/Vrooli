import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, Bot, Plus, TerminalSquare } from "lucide-react";
import { Link } from "react-router-dom";

import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { consoleApi, consoleKeys, type Agent } from "../api/console";
import { AgentMark } from "../components/console/AgentMark";
import { ChannelChip } from "../components/console/ChannelChip";
import { Page } from "../components/console/Page";
import { Quiet, Region } from "../components/console/Region";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const TERMINAL_SCOPES = new Set(["terminal", "shell", "exec", "owner"]);

export function agentReachesTerminal(agent: Agent): boolean {
  return agent.grant.scopes.some((scope) => TERMINAL_SCOPES.has(scope) || scope.startsWith("terminal"));
}

/**
 * The roster of every agent this install operates: where each is reachable
 * and what it may do, scannable by shape (mark, chips, grant) before names.
 */
export function AgentsPage() {
  const { t } = useTranslation();
  const roster = useQuery({ queryKey: consoleKeys.agents, queryFn: ({ signal }) => consoleApi.agents(signal), refetchInterval: 60_000 });
  const agents = roster.data?.agents ?? [];
  const sourceDown = roster.data ? !roster.data.source.ok : false;
  const state = roster.isPending ? "loading" : roster.isError ? "error" : agents.length === 0 ? "empty" : sourceDown ? "partial" : "ready";

  return (
    <Page
      headingId="agents-heading"
      testId="page-agents"
      title={t(strings.console.agents.title)}
      description={t(strings.console.agents.description)}
      actions={
        <Link
          to="/agents/new"
          data-testid="agents-create"
          className="inline-flex min-h-11 items-center gap-1.5 rounded-control bg-app-primary px-3 text-sm font-medium text-app-primary-foreground hover:opacity-90 md:min-h-9"
        >
          <Plus aria-hidden="true" className="h-4 w-4" />
          {t(strings.console.agents.newAgent)}
        </Link>
      }
    >
      {sourceDown ? (
        <div role="alert" className="flex items-start gap-2 rounded-panel border border-app-warning/50 bg-app-warning/10 px-3 py-2 text-sm">
          <AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-app-warning" />
          <p>
            <span className="font-medium">{t(strings.console.agents.sourceUnavailable)}</span>{" "}
            <span className="text-app-muted-foreground">{roster.data?.source.reason}</span>
          </p>
        </div>
      ) : null}
      <Region
        surfaceId="roster-region"
        testId="agents-roster-region"
        state={state}
        errorDetail={roster.error instanceof Error ? roster.error.message : undefined}
        onRetry={() => void roster.refetch()}
        skeletonRows={4}
        empty={
          <Quiet
            icon={<Bot className="h-6 w-6" />}
            title={t(strings.console.agents.emptyTitle)}
            description={t(strings.console.agents.empty)}
            action={
              <Link to="/agents/new" data-testid="agents-empty-cta" className="inline-flex min-h-11 items-center rounded-control bg-app-primary px-3 text-sm font-medium text-app-primary-foreground">
                {t(strings.console.agents.createFirst)}
              </Link>
            }
          />
        }
      >
        <ul data-testid="agents-roster" className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {agents.map((agent) => (
            <AgentCard key={agent.id} agent={agent} />
          ))}
        </ul>
      </Region>
    </Page>
  );
}

function AgentCard({ agent }: { agent: Agent }) {
  const { t } = useTranslation();
  const live = agent.bindings.some((binding) => binding.live);
  const reachable = agent.bindings.length > 0;
  return (
    <li data-testid="agents-card" data-agent-id={agent.id}>
      <Link
        to={`/agents/${encodeURIComponent(agent.id)}`}
        className="flex h-full flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4 transition-colors hover:border-app-primary/50 hover:shadow-subtle"
      >
        <div className="flex items-start gap-3">
          <AgentMark name={agent.display_name} appearance={agent.appearance} size="lg" live={reachable ? live : undefined} testId="agents-avatar" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="truncate text-sm font-semibold text-app-foreground">{agent.display_name}</h3>
              {agent.broken ? (
                <StatusBadge tone="danger">{t(strings.console.agents.brokenReference)}</StatusBadge>
              ) : (
                <StatusBadge tone={live ? "success" : reachable ? "warning" : "neutral"} data-testid="agents-live">
                  {live ? t(strings.console.agents.live) : reachable ? t(strings.console.agents.paused) : t(strings.console.agents.unbound)}
                </StatusBadge>
              )}
            </div>
            <p className="mt-0.5 font-mono text-[11px] text-app-muted-foreground">{agent.id}</p>
          </div>
        </div>
        {agent.broken ? (
          <p className="text-sm text-app-danger">{agent.broken}</p>
        ) : agent.description ? (
          <p className="line-clamp-2 text-sm text-app-muted-foreground">{agent.description}</p>
        ) : null}
        <div data-testid="agents-channels" role="group" aria-label={t(strings.console.agents.reachableAt)} className="flex flex-wrap items-center gap-1.5">
          {agent.bindings.length === 0 ? (
            <span className="text-xs text-app-muted-foreground">{t(strings.console.agents.notReachable)}</span>
          ) : (
            agent.bindings.map((binding) => (
              <ChannelChip key={binding.id} id={binding.channel_id} name={binding.channel_display_name} accent={binding.channel_accent} className={binding.live ? "" : "opacity-60"} />
            ))
          )}
        </div>
        <div data-testid="agents-grant" role="status" className="mt-auto flex flex-wrap items-center gap-1.5 border-t border-app-border pt-3 text-xs">
          {agent.grant.scopes.map((scope) => (
            <code key={scope} className="rounded-sm bg-app-surface-muted px-1.5 py-0.5 font-mono text-[11px] text-app-foreground">
              {scope}
            </code>
          ))}
          {agentReachesTerminal(agent) ? (
            <span className="inline-flex items-center gap-1 rounded-sm bg-app-danger/10 px-1.5 py-0.5 font-medium text-app-danger">
              <TerminalSquare aria-hidden="true" className="h-3 w-3" />
              {t(strings.console.agents.reachesTerminal)}
            </span>
          ) : null}
          {agent.grant.source === "default" ? <span className="text-app-muted-foreground">{t(strings.console.agents.defaultGrantShort)}</span> : null}
          {agent.activity ? (
            <span className="ml-auto text-app-muted-foreground">{t(strings.console.agents.activity24h, { turns: agent.activity.turns_24h, refusals: agent.activity.refusals_24h })}</span>
          ) : null}
        </div>
      </Link>
    </li>
  );
}
