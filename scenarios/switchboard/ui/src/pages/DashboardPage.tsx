import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, RefreshCw, ShieldOff } from "lucide-react";
import { Link } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { consoleApi, consoleKeys, type ChannelHealth } from "../api/console";
import { BudgetMeter } from "../components/console/BudgetMeter";
import { ChannelChip } from "../components/console/ChannelChip";
import { GateCard } from "../components/console/GateCard";
import { Page, StatStrip } from "../components/console/Page";
import { Quiet, Region } from "../components/console/Region";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { relativeTime } from "../lib/time";

/**
 * Overview: the operator's answer to one question — is anything waiting on me?
 * Unanswered gates first, then channel reachability, budget pressure and recent
 * refusals. When none of those exist the page says so and stays quiet.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const overview = useQuery({ queryKey: consoleKeys.overview, queryFn: ({ signal }) => consoleApi.overview(signal), refetchInterval: 30_000 });
  const threads = useQuery({ queryKey: consoleKeys.threads, queryFn: ({ signal }) => consoleApi.threads(signal), staleTime: 15_000 });
  const agents = useQuery({ queryKey: consoleKeys.agents, queryFn: ({ signal }) => consoleApi.agents(signal), staleTime: 60_000 });
  const contacts = useQuery({ queryKey: consoleKeys.contacts, queryFn: ({ signal }) => consoleApi.contacts(signal), staleTime: 60_000 });

  const data = overview.data;
  const pendingGates = data?.gates.filter((gate) => gate.status === "pending") ?? [];
  const attentionState = overview.isPending ? "loading" : overview.isError ? "error" : pendingGates.length === 0 ? "empty" : "ready";
  const channels = data?.channels ?? [];
  const degraded = channels.filter((channel) => channel.bindings > 0 && channel.availability !== "available");
  const channelState = overview.isPending ? "loading" : overview.isError ? "error" : degraded.length > 0 ? "partial" : "ready";
  const pressure = data?.budget.threads_under_pressure ?? [];
  const budgetState = overview.isPending ? "loading" : overview.isError ? "error" : "ready";
  const refusals = data?.refusals ?? [];
  const liveChannels = channels.filter((channel) => channel.availability === "available" && channel.implemented).length;
  const errorDetail = overview.error instanceof Error ? overview.error.message : undefined;

  return (
    <Page
      testId={selectors.pages.dashboard}
      headingId="dashboard-heading"
      title={t(strings.pages.dashboard.title)}
      description={t(strings.pages.dashboard.description)}
      actions={
        <>
          {data ? (
            <span className="text-xs text-app-muted-foreground" data-testid="overview-updated">
              {t(strings.console.overview.updated, { when: relativeTime(data.generated_at) })}
            </span>
          ) : null}
          <Button type="button" variant="secondary" size="sm" data-testid="overview-retry" pending={overview.isFetching} onClick={() => void overview.refetch()}>
            <RefreshCw aria-hidden="true" className="h-3.5 w-3.5" />
            {t(strings.console.overview.refresh)}
          </Button>
        </>
      }
    >
      <StatStrip
        items={[
          { label: t(strings.console.overview.stats.liveChannels), value: overview.isPending ? "…" : liveChannels, hint: t(strings.console.overview.stats.ofTotal, { total: channels.length }), testId: "overview-stat-channels" },
          { label: t(strings.console.overview.stats.threads), value: threads.data ? threads.data.length : "…", testId: "overview-stat-threads" },
          { label: t(strings.console.overview.stats.agents), value: agents.data ? agents.data.agents.length : "…", hint: agents.data && !agents.data.source.ok ? t(strings.console.agents.sourceUnavailableShort) : undefined, tone: agents.data && !agents.data.source.ok ? "warning" : "neutral", testId: "overview-stat-agents" },
          { label: t(strings.console.overview.stats.contacts), value: contacts.data ? contacts.data.length : "…", testId: "overview-stat-contacts" },
        ]}
      />

      <Region
        surfaceId="attention-region"
        testId="dashboard-attention-region"
        state={attentionState}
        title={t(strings.console.overview.waitingOnYou)}
        errorDetail={errorDetail}
        onRetry={() => void overview.refetch()}
        empty={
          <div data-testid="overview-all-clear" role="status" className="flex items-center gap-3 rounded-panel border border-app-success/30 bg-app-success/5 px-4 py-3 text-sm">
            <CheckCircle2 aria-hidden="true" className="h-5 w-5 shrink-0 text-app-success" />
            <div>
              <p className="font-medium text-app-foreground">{t(strings.console.overview.allClear)}</p>
              <p className="text-app-muted-foreground">{t(strings.console.overview.allClearDetail)}</p>
            </div>
          </div>
        }
      >
        <ul data-testid="overview-attention" className="flex flex-col gap-3">
          {pendingGates.map((gate) => (
            <li key={gate.id} data-testid="overview-gate-item">
              <GateCard gate={gate} compact />
              <Link to={`/conversations/${gate.thread_id}`} className="mt-1 inline-block text-xs font-medium text-app-primary">
                {t(strings.console.overview.openThread)}
              </Link>
            </li>
          ))}
        </ul>
      </Region>

      <div className="grid gap-6 lg:grid-cols-3">
        <Region
          surfaceId="channel-health-region"
          testId="dashboard-channel-health-region"
          state={channelState}
          title={t(strings.console.overview.channelHealth)}
          errorDetail={errorDetail}
          onRetry={() => void overview.refetch()}
          actions={
            <Link to="/channels" className="text-xs font-medium text-app-primary">
              {t(strings.console.overview.manageChannels)}
            </Link>
          }
        >
          <ul data-testid="overview-channel-health" className="divide-y divide-app-border overflow-hidden rounded-panel border border-app-border bg-app-surface">
            {channels.map((channel) => (
              <ChannelHealthRow key={channel.id} channel={channel} />
            ))}
          </ul>
        </Region>

        <Region
          surfaceId="budget-region"
          testId="dashboard-budget-region"
          state={budgetState}
          title={t(strings.console.overview.budgetPressure)}
          errorDetail={errorDetail}
          onRetry={() => void overview.refetch()}
        >
          <div data-testid="overview-budget">
            {pressure.length === 0 ? (
              <Quiet title={t(strings.console.overview.noBudgetPressure)} description={t(strings.console.overview.noBudgetPressureDetail)} />
            ) : (
              <ul className="flex flex-col gap-3">
                {pressure.map((budget) => (
                  <li key={budget.thread_id} className="rounded-panel border border-app-border bg-app-surface p-3">
                    <Link to={`/conversations/${budget.thread_id}`} className="mb-2 block truncate text-xs font-medium text-app-primary">
                      {budget.agent_id} · {budget.thread_key}
                    </Link>
                    <BudgetMeter budget={budget} testId="overview-budget-meter" />
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Region>

        <Region
          surfaceId="refusals-region"
          testId="dashboard-refusals-region"
          state={overview.isPending ? "loading" : overview.isError ? "error" : refusals.length === 0 ? "empty" : "ready"}
          title={t(strings.console.overview.recentRefusals)}
          errorDetail={errorDetail}
          onRetry={() => void overview.refetch()}
          empty={<Quiet icon={<ShieldOff className="h-5 w-5" />} title={t(strings.console.overview.noRefusals)} description={t(strings.console.overview.noRefusalsDetail)} />}
        >
          <ul data-testid="overview-refusals" className="divide-y divide-app-border overflow-hidden rounded-panel border border-app-border bg-app-surface">
            {refusals.slice(0, 8).map((refusal, index) => (
              <li key={`${refusal.thread_id}-${refusal.at}-${index}`} className="flex flex-col gap-1 px-3 py-2.5 text-sm">
                <div className="flex items-center justify-between gap-2">
                  <Link to={`/conversations/${refusal.thread_id}`} className="min-w-0 truncate font-medium text-app-foreground hover:text-app-primary">
                    {refusal.sender_address}
                  </Link>
                  <span className="shrink-0 text-xs text-app-muted-foreground">{relativeTime(refusal.at)}</span>
                </div>
                <div className="flex items-center gap-2">
                  <ChannelChip id={refusal.channel_id} name={refusal.channel_display_name} accent={refusal.channel_accent} />
                  <span className="truncate text-xs text-app-muted-foreground">{refusal.reason}</span>
                </div>
              </li>
            ))}
          </ul>
        </Region>
      </div>
    </Page>
  );
}

function ChannelHealthRow({ channel }: { channel: ChannelHealth }) {
  const { t } = useTranslation();
  const live = channel.availability === "available" && channel.implemented;
  const attached = channel.bindings > 0;
  const tone = live ? "success" : attached ? "danger" : "neutral";
  const label = live ? t(strings.console.channels.availability.available) : channel.implemented ? t(strings.console.channels.availability.unavailable) : t(strings.console.channels.availability.unimplemented);
  return (
    <li className="flex items-center gap-3 px-3 py-2.5 text-sm">
      <span aria-hidden="true" className="h-8 w-1 shrink-0 rounded-full" style={{ background: channel.accent ?? "var(--color-accent)" }} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <Link to={`/channels/${channel.id}`} className="truncate font-medium text-app-foreground hover:text-app-primary">
            {channel.display_name}
          </Link>
          <StatusBadge tone={tone} data-testid="channels-health">
            {label}
          </StatusBadge>
        </div>
        <p className="truncate text-xs text-app-muted-foreground">
          {live
            ? t(strings.console.overview.channelCounts, { bindings: channel.bindings, threads: channel.threads })
            : channel.reason ?? t(strings.console.channels.availability.unknown)}
        </p>
      </div>
    </li>
  );
}
