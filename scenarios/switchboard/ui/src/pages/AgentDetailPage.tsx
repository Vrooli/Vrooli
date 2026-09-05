import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ExternalLink, Lock, Plus } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { ConsoleApiError, consoleApi, consoleKeys, type AgentActivityEntry } from "../api/console";
import { AgentMark } from "../components/console/AgentMark";
import { BudgetMeter } from "../components/console/BudgetMeter";
import { ChannelChip } from "../components/console/ChannelChip";
import { Page } from "../components/console/Page";
import { Quiet, Region } from "../components/console/Region";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { relativeTime } from "../lib/time";

const ACTIVITY_KIND_KEY = {
  turn: strings.console.agentDetail.activityKind.turn,
  refusal: strings.console.agentDetail.activityKind.refusal,
  suppressed: strings.console.agentDetail.activityKind.suppressed,
  gate: strings.console.agentDetail.activityKind.gate,
} as const;

/**
 * One agent in full. The grant section is the centre of gravity because it is
 * the only thing here that can cause harm: owner-only scopes render as a fact,
 * never as a disabled control that suggests they sit on a movable spectrum.
 */
export function AgentDetailPage() {
  const { t } = useTranslation();
  const { agentId = "" } = useParams<{ agentId: string }>();
  const detail = useQuery({ queryKey: consoleKeys.agent(agentId), queryFn: ({ signal }) => consoleApi.agent(agentId, signal), enabled: agentId !== "" });
  const threads = useQuery({ queryKey: consoleKeys.threads, queryFn: ({ signal }) => consoleApi.threads(signal), staleTime: 15_000 });
  const agent = detail.data;
  const agentThreads = (threads.data ?? []).filter((thread) => thread.agent_id === agentId);
  const notFound = detail.error instanceof ConsoleApiError && detail.error.status === 404;
  const grantState = detail.isPending ? "loading" : notFound ? "ready" : detail.isError ? "error" : "ready";
  const log = agent?.activity_log ?? [];
  const activityState = detail.isPending ? "loading" : notFound ? "empty" : detail.isError ? "error" : log.length === 0 ? "empty" : "ready";

  return (
    <Page
      headingId="agent-heading"
      testId="page-agent"
      eyebrow={
        <Link to="/agents" className="inline-flex items-center gap-1 text-app-primary">
          <ArrowLeft aria-hidden="true" className="h-3 w-3" />
          {t(strings.console.agents.title)}
        </Link>
      }
      title={agent?.display_name ?? agentId}
      description={agent?.description}
      actions={
        <a
          href={`/prompt-manager/agents/${encodeURIComponent(agentId)}`}
          data-testid="agent-profile-link"
          className="inline-flex min-h-11 items-center gap-1.5 rounded-control border border-app-border px-3 text-sm font-medium text-app-foreground hover:bg-app-surface-muted md:min-h-9"
        >
          {t(strings.console.agentDetail.editInPromptManager)}
          <ExternalLink aria-hidden="true" className="h-3.5 w-3.5" />
        </a>
      }
    >
      <div data-testid="agent-header" className="flex flex-wrap items-center gap-4 rounded-panel border border-app-border bg-app-surface p-4">
        <AgentMark name={agent?.display_name ?? agentId} appearance={agent?.appearance} size="xl" testId="agent-avatar" />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <code className="font-mono text-xs text-app-muted-foreground">{agentId}</code>
            {agent?.status ? <StatusBadge tone={agent.status === "active" ? "success" : "neutral"}>{agent.status}</StatusBadge> : null}
            {agent?.broken ? <StatusBadge tone="danger">{t(strings.console.agents.brokenReference)}</StatusBadge> : null}
          </div>
          {agent?.tags?.length ? (
            <ul className="mt-2 flex flex-wrap gap-1.5">
              {agent.tags.map((tag) => (
                <li key={tag} className="rounded-pill bg-app-surface-muted px-2 py-0.5 text-xs text-app-muted-foreground">
                  {tag}
                </li>
              ))}
            </ul>
          ) : null}
          {agent?.broken ? <p className="mt-2 text-sm text-app-danger">{agent.broken}</p> : null}
          <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.console.agentDetail.profileOwnedElsewhere)}</p>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <div className="flex flex-col gap-6">
          <Region
            surfaceId="grant-region"
            testId="agent-detail-grant-region"
            state={grantState}
            title={t(strings.console.agentDetail.whatItMayDo)}
            errorDetail={detail.error instanceof Error ? detail.error.message : undefined}
            onRetry={() => void detail.refetch()}
          >
            {notFound ? <Quiet title={t(strings.console.agentDetail.notFound)} description={t(strings.console.agentDetail.notFoundDetail)} /> : null}
            {agent ? (
              <div data-testid="agent-grant" role="region" aria-label={t(strings.console.agentDetail.whatItMayDo)} className="rounded-panel border border-app-border bg-app-surface">
                <ul className="divide-y divide-app-border">
                  {agent.grant.scopes.map((scope) => (
                    <li key={scope} className="flex items-center gap-3 px-4 py-3 text-sm">
                      <code className="rounded-sm bg-app-surface-muted px-1.5 py-0.5 font-mono text-xs">{scope}</code>
                      <span className="text-app-muted-foreground">{t(strings.console.agentDetail.scopeGranted)}</span>
                    </li>
                  ))}
                  {agent.grant.program_bindings?.map((binding) => (
                    <li key={binding} className="flex items-center gap-3 px-4 py-3 text-sm">
                      <code className="rounded-sm bg-app-surface-muted px-1.5 py-0.5 font-mono text-xs">{binding}</code>
                      <span className="text-app-muted-foreground">{t(strings.console.agentDetail.programBinding)}</span>
                    </li>
                  ))}
                </ul>
                <div className="border-t border-app-border px-4 py-3 text-xs text-app-muted-foreground">
                  {agent.grant.source === "default" ? t(strings.console.agentDetail.defaultGrant) : t(strings.console.agentDetail.descriptorGrant)}
                </div>
                <div data-testid="agent-owner-only" role="note" className="flex items-start gap-2 border-t border-app-border bg-app-surface-muted/60 px-4 py-3 text-xs text-app-muted-foreground">
                  <Lock aria-hidden="true" className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                  <p>
                    {t(strings.console.agentDetail.ownerOnlyNote)}
                    {agent.grant.owner_only.length > 0 ? (
                      <>
                        {" "}
                        {agent.grant.owner_only.map((scope) => (
                          <code key={scope} className="mr-1 rounded-sm bg-app-surface px-1 py-0.5 font-mono">
                            {scope}
                          </code>
                        ))}
                      </>
                    ) : null}
                  </p>
                </div>
              </div>
            ) : null}
          </Region>

          <Region surfaceId="channel-list-region" testId="agent-detail-channels-region" state={grantState} title={t(strings.console.agents.reachableAt)} actions={
            <Link to="/channels" className="inline-flex items-center gap-1 text-xs font-medium text-app-primary">
              <Plus aria-hidden="true" className="h-3 w-3" />
              {t(strings.console.agentDetail.attachToChannel)}
            </Link>
          }>
            {agent ? (
              agent.bindings.length === 0 ? (
                <Quiet title={t(strings.console.agents.notReachable)} description={t(strings.console.agentDetail.notReachableDetail)} />
              ) : (
                <ul data-testid="agent-channels" className="divide-y divide-app-border rounded-panel border border-app-border bg-app-surface">
                  {agent.bindings.map((binding) => (
                    <li key={binding.id} className="flex flex-wrap items-center gap-3 px-4 py-2.5 text-sm">
                      <ChannelChip id={binding.channel_id} name={binding.channel_display_name} accent={binding.channel_accent} size="md" />
                      <code className="truncate font-mono text-xs text-app-muted-foreground">{binding.address}{binding.thread_key ? ` · ${binding.thread_key}` : ""}</code>
                      <StatusBadge tone={binding.live ? "success" : "warning"} className="ml-auto">
                        {binding.live ? t(strings.console.agents.live) : t(strings.console.agents.paused)}
                      </StatusBadge>
                    </li>
                  ))}
                </ul>
              )
            ) : null}
          </Region>
        </div>

        <div className="flex flex-col gap-6">
          <Region surfaceId="budget-section" testId="agent-budgets" state={threads.isPending ? "loading" : "ready"} title={t(strings.console.agentDetail.budgets)}>
            {agentThreads.length === 0 ? (
              <Quiet title={t(strings.console.agentDetail.noThreads)} />
            ) : (
              <ul className="flex flex-col gap-2">
                {agentThreads.slice(0, 6).map((thread) => (
                  <li key={thread.id} className="rounded-panel border border-app-border bg-app-surface p-3">
                    <Link to={`/conversations/${thread.id}`} className="mb-2 flex items-center gap-2 text-xs">
                      <ChannelChip id={thread.channel_id} name={thread.channel_display_name} accent={thread.channel_accent} />
                      <span className="truncate font-mono text-app-muted-foreground">{thread.thread_key}</span>
                    </Link>
                    <BudgetMeter budget={thread.budget} compact />
                  </li>
                ))}
              </ul>
            )}
          </Region>

          <Region
            surfaceId="activity-region"
            testId="agent-detail-activity-region"
            state={activityState}
            title={t(strings.console.agentDetail.recentActivity)}
            empty={<Quiet title={t(strings.console.agentDetail.noActivity)} />}
          >
            <ol data-testid="agent-activity" className="relative flex flex-col gap-0 border-l border-app-border pl-4">
              {log.map((entry, index) => (
                <ActivityRow key={`${entry.at}-${index}`} entry={entry} />
              ))}
            </ol>
          </Region>
        </div>
      </div>
    </Page>
  );
}

function ActivityRow({ entry }: { entry: AgentActivityEntry }) {
  const { t } = useTranslation();
  const tone = entry.kind === "refusal" ? "bg-app-danger" : entry.kind === "gate" ? "bg-app-warning" : entry.kind === "suppressed" ? "bg-app-muted-foreground" : "bg-app-success";
  const label = t(ACTIVITY_KIND_KEY[entry.kind]);
  return (
    <li className="relative pb-4 text-sm">
      <span aria-hidden="true" className={["absolute -left-[21px] top-1.5 h-2.5 w-2.5 rounded-full ring-4 ring-app-background", tone].join(" ")} />
      <div className="flex items-baseline justify-between gap-2">
        <span className="font-medium text-app-foreground">{label}</span>
        <time dateTime={entry.at} className="text-xs text-app-muted-foreground">
          {relativeTime(entry.at)}
        </time>
      </div>
      <Link to={`/conversations/${entry.thread_id}`} className="block truncate text-xs text-app-muted-foreground hover:text-app-primary">
        {entry.reason ?? entry.text ?? entry.channel_id}
      </Link>
    </li>
  );
}
