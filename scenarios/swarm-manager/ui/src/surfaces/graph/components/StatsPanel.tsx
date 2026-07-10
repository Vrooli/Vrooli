/**
 * StatsContent - tabbed operational metrics content shared by the routed
 * Stats view and focused tests.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import {
  Activity,
  AlertCircle,
  BarChart3,
  CheckCircle2,
  Clock3,
  Gauge,
  Info,
  LayoutDashboard,
  Layers,
  ListChecks,
  Loader2,
  MessageSquare,
  Target,
  Timer,
  TrendingUp,
  Users,
  Zap,
} from "lucide-react";
import { HistoryBanner } from "../../../components/stats/history-banner";
import { InsufficientDataCard } from "../../../components/stats/insufficient-data-card";
import { KeyValueList } from "../../../components/stats/key-value-list";
import { MiniBarChart } from "../../../components/stats/mini-bar-chart";
import { ProgressBar } from "../../../components/stats/progress-bar";
import { SectionLabel } from "../../../components/stats/section-label";
import { StatsCard as StatCard } from "../../../components/stats/stats-card";
import { EtaExplainerTrigger, type EtaExplainerBand } from "../../../components/stats/EtaExplainer";
import { StatsEmptyState } from "../../../components/stats/stats-empty-state";
import { StatsMetricCard } from "../../../components/stats/stats-metric-card";
import { CompactTabBar } from "../../../components/ui/compact-tab-bar";
import { cn } from "../../../lib/utils";
import { Popover } from "../../../components/ui/popover";
import { InitiativeSummaryCard } from "../../../components/initiative/initiative-summary-card";
import { useInitiativeStore } from "../../../stores/initiative-store";
import { useNavigate } from "react-router-dom";
import { initiativeDetailPath } from "../../../app/routes/route-paths";
import { backlogDetailPath } from "../../../app/routes/route-paths";
import { BoardCard } from "../../../components/cards/BoardCard";
import {
  formatDelta,
  formatHours,
  formatRate,
} from "../../../lib/stats-format-utils";
import type {
  AgentStats,
  BlockingStats,
  DashboardStats,
  HistoryWindow,
  ModeStats,
  ScopeStats,
  SessionStats,
  StatsCategory,
  StatsResponse,
  ThroughputStats,
  TimingStats,
} from "../../../types/stats";
import type { CompactTabItem } from "../../../components/ui/compact-tab-bar";
import type { StatsEtaBand } from "../../../types/stats";

// ---------------------------------------------------------------------------
// Tab config
// ---------------------------------------------------------------------------

/** Map the snake_case Stats ETA band onto the shared explainer's normalized shape. */
function toExplainerBand(eta: StatsEtaBand): EtaExplainerBand {
  return {
    p50Label: eta.p50_label,
    p80Label: eta.p80_label,
    remainingItems: eta.remaining_items,
    laneCapacity: eta.lane_capacity,
    basis: eta.basis,
    basisLabel: eta.basis_label,
    confidence: eta.confidence,
  };
}

const STATS_TABS: CompactTabItem<StatsCategory>[] = [
  { value: "dashboard", label: "Dashboard", icon: LayoutDashboard },
  { value: "throughput", label: "Throughput", icon: TrendingUp },
  { value: "agent", label: "Agent", icon: Zap },
  { value: "timing", label: "Timing", icon: Timer },
  { value: "blocking", label: "Blocking", icon: AlertCircle },
  { value: "scope", label: "Scope", icon: Target },
  { value: "modes", label: "Modes", icon: Layers },
  { value: "sessions", label: "Sessions", icon: MessageSquare },
];

// Default min sample threshold used when the response does not include one.
// The Go engine exports this via stats.MinSampleMeaningful; the response
// echoes it so UI and backend stay aligned.
const DEFAULT_MIN_SAMPLE = 5;

function minSample(history: HistoryWindow | undefined): number {
  return history?.min_sample_meaningful && history.min_sample_meaningful > 0
    ? history.min_sample_meaningful
    : DEFAULT_MIN_SAMPLE;
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface StatsContentProps {
  data: StatsResponse | undefined;
  isLoading: boolean;
  error: Error | null;
  activeTab: StatsCategory;
  onTabChange: (tab: StatsCategory) => void;
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function StatsContent({
  data,
  isLoading,
  error,
  activeTab,
  onTabChange,
}: StatsContentProps) {
  return (
    <div className="space-y-4" data-testid="stats-content">
      <CompactTabBar
        items={STATS_TABS}
        activeValue={activeTab}
        onValueChange={onTabChange}
        aria-label="Stats sections"
        className="border-b border-slate-700/50"
        tabTestIdPrefix="stats-tab"
      />

      {isLoading && (
        <div className="flex items-center justify-center gap-2 py-12 text-sm text-slate-400" data-testid="stats-loading">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading stats...
        </div>
      )}

      {error && (
        <div className="flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300" data-testid="stats-error">
          <AlertCircle className="h-4 w-4 shrink-0" />
          Failed to load stats: {error.message}
        </div>
      )}

      {data && (
        <>
          <HistoryBanner history={data.history} testId="stats-history-banner" />
          <TabContent tab={activeTab} data={data} />
        </>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Tab content dispatcher
// ---------------------------------------------------------------------------

function TabContent({ tab, data }: { tab: StatsCategory; data: StatsResponse }) {
  switch (tab) {
    case "dashboard":
      return <DashboardTab data={data.dashboard} eventCount={data.event_count} history={data.history} />;
    case "throughput":
      return <ThroughputTab data={data.throughput} dashboard={data.dashboard} />;
    case "agent":
      return <AgentTab data={data.agent} history={data.history} />;
    case "timing":
      return <TimingTab data={data.timing} history={data.history} />;
    case "blocking":
      return <BlockingTab data={data.blocking} />;
    case "scope":
      return <ScopeTab data={data.scope} />;
    case "modes":
      return <ModesTab data={data.mode} />;
    case "sessions":
      return <SessionsTab data={data.session} history={data.history} />;
  }
}

// ---------------------------------------------------------------------------
// Dashboard tab
// ---------------------------------------------------------------------------

function DashboardTab({ data, eventCount, history }: { data: DashboardStats; eventCount: number; history: HistoryWindow }) {
  const velocityTrend = data.velocity_trend ?? [];
  const navigate = useNavigate();
  const [selectedWeek, setSelectedWeek] = useState<string | null>(null);
  const selectedPoint = velocityTrend.find((point) => point.week_start === selectedWeek);

  const eta = data.estimated_remaining;

  return (
    <div className="space-y-4" data-testid="stats-content-dashboard">
      <div className="grid gap-3 sm:grid-cols-3">
        <StatCard label="Backlog" value={data.total_backlog_size.toLocaleString()} icon={ListChecks} testId="stat-backlog-size" />
        <StatCard
          label="Completed"
          value={data.total_completed_all_time.toLocaleString()}
          subtext="all time"
          icon={CheckCircle2}
          testId="stat-completed-all-time"
        />
        {eta ? (
          <StatCard
            label="Est. Remaining"
            value={`${eta.p50_label} - ${eta.p80_label}`}
            subtext={`${eta.remaining_items.toLocaleString()} items · ${eta.basis_label}`}
            icon={Clock3}
            testId="stat-weeks-remaining"
          >
            <EtaExplainerTrigger band={toExplainerBand(eta)} testId="stat-weeks-remaining-explainer" />
          </StatCard>
        ) : (
          <InsufficientDataCard
            label="Est. Remaining"
            reason="No estimable backlog closure is available."
            testId="stat-weeks-remaining"
          />
        )}
      </div>

      <div>
        <SectionLabel icon={TrendingUp}>Velocity (last {velocityTrend.length} weeks)</SectionLabel>
        {velocityTrend.length === 0 ? (
          <StatsEmptyState>No velocity data yet</StatsEmptyState>
        ) : (
          <MiniBarChart
            points={velocityTrend.map((point) => ({
              key: point.week_start,
              label: point.week_start,
              value: point.completed,
            }))}
            testId="stats-velocity-chart"
            onSelect={(point) => setSelectedWeek(point.key)}
          />
        )}
        {selectedWeek && selectedPoint && (
          <div className="mt-3 space-y-2" data-testid="stats-velocity-drilldown">
            <p className="text-xs font-medium text-slate-300">Completed that week ({selectedWeek})</p>
            {selectedPoint.completed_items?.length ? (
              selectedPoint.completed_items.map((item) => (
                <BoardCard
                  key={`${item.kind}/${item.name}`}
                  title={item.name}
                  subtitle={item.kind}
                  tone="positive"
                  onClick={() => navigate(backlogDetailPath(item.kind, item.name))}
                />
              ))
            ) : (
              <StatsEmptyState>No completed items recorded for this week</StatsEmptyState>
            )}
          </div>
        )}
      </div>

      <p className="text-xs text-slate-500">
        {eventCount.toLocaleString()} events processed
        {history.has_history && (
          <> · {Math.max(1, Math.round(history.history_days))}d of history</>
        )}
      </p>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Throughput tab
// ---------------------------------------------------------------------------

function formatItemsPerWeek(count: number, days: number): string {
  return `${((count / days) * 7).toFixed(1)} / wk`;
}

function deltaClassName(value: number): string {
  if (value > 0) return "text-amber-400";
  if (value < 0) return "text-emerald-400";
  return "text-slate-300";
}

function ThroughputTab({ data, dashboard }: { data: ThroughputStats; dashboard: DashboardStats }) {
  const trend = data.throughput_trend ?? [];
  const hasTrendData = trend.some((point) => point.created > 0 || point.completed > 0);
  const eta = dashboard.estimated_remaining;

  return (
    <div className="space-y-4" data-testid="stats-content-throughput">
      <div className="grid gap-3 sm:grid-cols-3">
        <StatCard
          label="Created"
          value={data.created_last_7_days.toLocaleString()}
          subtext="last 7 days"
          icon={ListChecks}
          testId="stat-throughput-created-7d"
        />
        <StatCard
          label="Completed"
          value={data.completed_last_7_days.toLocaleString()}
          subtext="last 7 days"
          icon={CheckCircle2}
          testId="stat-throughput-completed-7d"
        />
        <StatCard
          label="Net Delta"
          value={formatDelta(data.net_delta_7_days)}
          subtext={data.net_delta_7_days > 0 ? "backlog grew" : data.net_delta_7_days < 0 ? "backlog shrank" : "balanced"}
          icon={TrendingUp}
          valueClassName={deltaClassName(data.net_delta_7_days)}
          testId="stat-throughput-net-7d"
        />
      </div>

      <div>
        <div className="mb-2 flex items-center justify-between gap-3">
          <SectionLabel icon={BarChart3}>Created vs completed</SectionLabel>
          <div className="flex items-center gap-3 text-[11px] text-slate-500">
            <span className="inline-flex items-center gap-1"><span className="h-2 w-2 rounded-sm bg-cyan-400/70" /> Created</span>
            <span className="inline-flex items-center gap-1"><span className="h-2 w-2 rounded-sm bg-emerald-400/70" /> Completed</span>
          </div>
        </div>
        {!hasTrendData ? (
          <StatsEmptyState testId="stats-throughput-empty">
            No created or completed work recorded in the trend window
          </StatsEmptyState>
        ) : (
          <MiniBarChart
            points={trend.map((point) => ({
              key: point.week_start,
              label: point.week_start,
              value: point.created,
              secondaryValue: point.completed,
            }))}
            valueLabel="created"
            secondaryValueLabel="completed"
            testId="stats-throughput-chart"
          />
        )}
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        {eta ? (
          <StatCard
            label="Burndown"
            value={`${eta.p50_label} - ${eta.p80_label}`}
            subtext={`${eta.remaining_items.toLocaleString()} items · ${eta.basis_label}`}
            icon={Clock3}
            testId="stat-throughput-burndown"
          >
            <EtaExplainerTrigger band={toExplainerBand(eta)} testId="stat-throughput-burndown-explainer" />
          </StatCard>
        ) : (
          <InsufficientDataCard
            label="Burndown"
            reason="No estimable backlog closure is available."
            testId="stat-throughput-burndown"
          />
        )}
        <StatCard
          label="30d Flow Rate"
          value={`${formatItemsPerWeek(data.completed_last_30_days, 30)} done`}
          subtext={`${formatItemsPerWeek(data.created_last_30_days, 30)} created · ${formatDelta(data.net_delta_30_days)} net`}
          icon={Gauge}
          testId="stat-throughput-rate"
        />
      </div>

      <div>
        <SectionLabel icon={Activity}>Window detail</SectionLabel>
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-xs text-slate-500">
              <th className="pb-2 font-medium" />
              <th className="pb-2 font-medium">7 days</th>
              <th className="pb-2 font-medium">30 days</th>
            </tr>
          </thead>
          <tbody className="text-slate-200">
            <tr>
              <td className="py-1 text-slate-400">Created</td>
              <td className="py-1">{data.created_last_7_days}</td>
              <td className="py-1">{data.created_last_30_days}</td>
            </tr>
            <tr>
              <td className="py-1 text-slate-400">Completed</td>
              <td className="py-1">{data.completed_last_7_days}</td>
              <td className="py-1">{data.completed_last_30_days}</td>
            </tr>
            <tr className="border-t border-slate-700/50">
              <td className="py-1 text-slate-400">Net delta</td>
              <td className={cn("py-1 font-medium", deltaClassName(data.net_delta_7_days))}>
                {formatDelta(data.net_delta_7_days)}
              </td>
              <td className={cn("py-1 font-medium", deltaClassName(data.net_delta_30_days))}>
                {formatDelta(data.net_delta_30_days)}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Agent tab
// ---------------------------------------------------------------------------

function AgentTab({ data, history }: { data: AgentStats; history: HistoryWindow }) {
  const threshold = minSample(history);
  const rateSample = data.success_rate_sample_size;
  const rateReady = rateSample >= Math.max(1, threshold);
  const durationReady = data.execution_duration_samples >= Math.max(1, threshold);
  const workshopReady = data.workshop_rounds_sample_size >= Math.max(1, threshold);

  return (
    <div className="space-y-4" data-testid="stats-content-agent">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <StatCard
          label="Total Executions"
          value={String(data.total_executions)}
          subtext={`${data.completed_count} completed · ${data.failed_count} failed`}
          icon={Zap}
        />
        <StatsMetricCard
          label="Avg Duration"
          value={`${data.avg_execution_minutes.toFixed(1)} min`}
          sampleSize={data.execution_duration_samples}
          minSample={threshold}
          sampleNoun="completed runs"
          insufficientReason={`Need at least ${threshold} finished runs.`}
          icon={Timer}
        />
      </div>

      {rateReady ? (
        <div className="space-y-3">
          <div>
            <div className="mb-1 flex justify-between text-xs">
              <span className="text-slate-400">Success rate</span>
              <span className="text-emerald-400">{formatRate(data.success_rate)} <span className="text-slate-500">({data.completed_count} of {rateSample})</span></span>
            </div>
            <ProgressBar value={data.success_rate} max={1} color="bg-emerald-500" />
          </div>
          <div>
            <div className="mb-1 flex justify-between text-xs">
              <span className="text-slate-400">Failure rate</span>
              <span className="text-red-400">{formatRate(data.failure_rate)} <span className="text-slate-500">({data.failed_count} of {rateSample})</span></span>
            </div>
            <ProgressBar value={data.failure_rate} max={1} color="bg-red-500" />
          </div>
          {data.manually_accepted_count > 0 && (
            <div>
              <div className="mb-1 flex justify-between text-xs">
                <span className="text-slate-400">Manually accepted</span>
                <span className="text-cyan-300">{formatRate(data.manual_accept_rate)} <span className="text-slate-500">({data.manually_accepted_count} of {rateSample})</span></span>
              </div>
              <ProgressBar value={data.manual_accept_rate} max={1} color="bg-cyan-500" />
              <p className="mt-1 text-[11px] text-slate-500">
                Runs the agent flagged as failed but you accepted as good enough.
              </p>
            </div>
          )}
          <div>
            <div className="mb-1 flex justify-between text-xs">
              <span className="text-slate-400">Follow-up rate</span>
              <span className="text-amber-400">{formatRate(data.follow_up_rate)}</span>
            </div>
            <ProgressBar value={data.follow_up_rate} max={1} color="bg-amber-500" />
          </div>
        </div>
      ) : (
        <InsufficientDataCard
          label="Success / failure rate"
          reason={`Need at least ${threshold} finished runs before rates are meaningful.`}
          have={rateSample}
          required={threshold}
        />
      )}

      {workshopReady || durationReady ? (
        <StatsMetricCard
          label="Avg Workshop Rounds"
          value={data.avg_workshop_rounds.toFixed(1)}
          sampleSize={data.workshop_rounds_sample_size}
          minSample={threshold}
          sampleNoun="workshop runs"
          insufficientReason={`Need at least ${threshold} items with workshop rounds.`}
        />
      ) : (
        <InsufficientDataCard
          label="Avg Workshop Rounds"
          reason="No workshop rounds recorded yet."
          have={data.workshop_rounds_sample_size}
          required={threshold}
        />
      )}

      <RecommendationAcceptanceSection data={data} threshold={threshold} />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Recommendation acceptance (Agent tab subsection)
// ---------------------------------------------------------------------------

function RecommendationAcceptanceSection({
  data,
  threshold,
}: {
  data: AgentStats;
  threshold: number;
}) {
  const sample = data.recommendation_acceptance_sample_size;
  const ready = sample >= Math.max(1, threshold);
  const byKind = data.recommendation_acceptance_by_kind ?? {};
  const kindEntries = Object.entries(byKind).sort(([a], [b]) => a.localeCompare(b));
  const [showByKind, setShowByKind] = useState(false);

  return (
    <div className="space-y-3" data-testid="stats-recommendation-acceptance">
      <SectionLabel icon={MessageSquare}>Decision Recommendations</SectionLabel>
      {ready ? (
        <div className="space-y-3">
          <div>
            <div className="mb-1 flex justify-between text-xs">
              <span className="text-slate-400">Recommendation acceptance</span>
              <span className="text-emerald-400">
                {formatRate(data.recommendation_acceptance_rate)}{" "}
                <span className="text-slate-500">(n={sample})</span>
              </span>
            </div>
            <ProgressBar value={data.recommendation_acceptance_rate} max={1} color="bg-emerald-500" />
            <p className="mt-1 text-[11px] text-slate-500">
              Of decisions you answered, the share where you picked the agent&apos;s recommended option.
              Picking &quot;Other&quot; counts as rejecting the recommendation.
            </p>
          </div>
          <div>
            <div className="mb-1 flex justify-between text-xs">
              <span className="text-slate-400">Freeform override</span>
              <span className="text-amber-400">
                {formatRate(data.freeform_override_rate)}{" "}
                <span className="text-slate-500">(n={sample})</span>
              </span>
            </div>
            <ProgressBar value={data.freeform_override_rate} max={1} color="bg-amber-500" />
            <p className="mt-1 text-[11px] text-slate-500">
              Share of answers that picked &quot;Other&quot;. A high rate means the offered options miss the mark.
            </p>
          </div>
          {kindEntries.length > 0 && (
            <div>
              <button
                type="button"
                onClick={() => setShowByKind((v) => !v)}
                className="text-xs text-slate-400 hover:text-slate-200"
                data-testid="stats-rec-by-kind-toggle"
              >
                {showByKind ? "Hide breakdown by kind" : "Show breakdown by kind"}
              </button>
              {showByKind && (
                <div className="mt-2 space-y-2">
                  {kindEntries.map(([kind, kr]) => {
                    const kindReady = kr.sample_size >= Math.max(1, threshold);
                    return kindReady ? (
                      <div key={kind} className="text-xs">
                        <div className="mb-1 flex justify-between">
                          <span className="capitalize text-slate-400">{kind}</span>
                          <span className="text-emerald-300">
                            {formatRate(kr.rate)}{" "}
                            <span className="text-slate-500">(n={kr.sample_size})</span>
                          </span>
                        </div>
                        <ProgressBar value={kr.rate} max={1} color="bg-emerald-500" />
                      </div>
                    ) : (
                      <InsufficientDataCard
                        key={kind}
                        label={kind.charAt(0).toUpperCase() + kind.slice(1)}
                        reason={`Need at least ${threshold} answered decisions in this kind.`}
                        have={kr.sample_size}
                        required={threshold}
                      />
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      ) : (
        <InsufficientDataCard
          label="Recommendation acceptance"
          reason={`Need at least ${threshold} answered decisions.`}
          have={sample}
          required={threshold}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Timing tab
// ---------------------------------------------------------------------------

function TimingTab({ data, history }: { data: TimingStats; history: HistoryWindow }) {
  const threshold = minSample(history);
  return (
    <div className="space-y-3" data-testid="stats-content-timing">
      <SectionLabel icon={Clock3}>Lead Time (created → complete)</SectionLabel>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <StatsMetricCard
          label="Average"
          value={formatHours(data.avg_lead_time_hours)}
          sampleSize={data.lead_time_sample_size}
          minSample={threshold}
          sampleNoun="items"
          insufficientReason={`Need at least ${threshold} items tracked from creation to completion.`}
          icon={Gauge}
        />
        <StatsMetricCard
          label="Median"
          value={formatHours(data.median_lead_time_hours)}
          sampleSize={data.lead_time_sample_size}
          minSample={threshold}
          sampleNoun="items"
          insufficientReason={`Need at least ${threshold} items tracked from creation to completion.`}
          icon={BarChart3}
        />
      </div>

      <SectionLabel icon={Timer}>Execution Duration (running → complete)</SectionLabel>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <StatsMetricCard
          label="Average"
          value={`${data.avg_execution_minutes.toFixed(1)} min`}
          sampleSize={data.execution_duration_samples}
          minSample={threshold}
          sampleNoun="finished runs"
          insufficientReason={`Need at least ${threshold} finished executions.`}
          icon={Timer}
        />
        <StatsMetricCard
          label="Median"
          value={`${data.median_execution_minutes.toFixed(1)} min`}
          sampleSize={data.execution_duration_samples}
          minSample={threshold}
          sampleNoun="finished runs"
          insufficientReason={`Need at least ${threshold} finished executions.`}
          icon={BarChart3}
        />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Blocking tab
// ---------------------------------------------------------------------------

function BlockingTab({ data }: { data: BlockingStats }) {
  const topReasons = data.top_reasons ?? [];

  return (
    <div className="space-y-4" data-testid="stats-content-blocking">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <StatCard label="Blocked" value={String(data.currently_blocked)} icon={AlertCircle} />
        <StatCard label="Blocked %" value={formatRate(data.blocked_ratio)} icon={Gauge} />
        <StatCard label="Avg Block Time" value={formatHours(data.avg_block_hours)} icon={Clock3} />
      </div>

      <div>
        <SectionLabel icon={AlertCircle}>Top Blocking Reasons</SectionLabel>
        {topReasons.length === 0 ? (
          <StatsEmptyState>No blocking reasons recorded</StatsEmptyState>
        ) : (
          <ul className="space-y-1">
            {topReasons.map((r) => (
              <li key={r.reason} className="flex items-center justify-between rounded px-2 py-1 text-sm hover:bg-slate-800/50">
                <span className="truncate text-slate-300">{r.reason}</span>
                <span className="ml-2 shrink-0 rounded bg-slate-700/60 px-1.5 py-0.5 text-xs text-slate-400">
                  {r.count}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Scope tab
// ---------------------------------------------------------------------------

function ScopeTab({ data }: { data: ScopeStats }) {
  const initiatives = data.initiatives ?? [];
  const navigate = useNavigate();
  const storeItems = useInitiativeStore((s) => s.items);
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);
  const [explainerOpen, setExplainerOpen] = useState(false);
  const explainerRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    void fetchInitiatives();
  }, [fetchInitiatives]);

  const itemByName = useMemo(
    () => new Map(storeItems.map((item) => [item.initiative.name, item])),
    [storeItems],
  );

  return (
    <div className="space-y-4" data-testid="stats-content-scope">
      <div className="flex items-start justify-between gap-3">
        <p className="text-xs leading-5 text-slate-400">
          Scope shows initiative health and how its tracked item count has changed.
          {data.max_dependency_depth > 0 && ` Maximum dependency depth: ${data.max_dependency_depth}.`}
        </p>
        <button
          ref={explainerRef}
          type="button"
          onClick={() => setExplainerOpen((open) => !open)}
          aria-label="Explain scope statistics"
          aria-expanded={explainerOpen}
          className="shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-800 hover:text-cyan-300"
        >
          <Info className="h-3.5 w-3.5" aria-hidden />
        </button>
        <Popover
          isOpen={explainerOpen}
          onClose={() => setExplainerOpen(false)}
          triggerRef={explainerRef}
          placement="bottom-end"
          className="w-72 p-3 text-xs text-slate-300"
        >
          <h3 className="mb-1 text-sm font-semibold text-slate-100">How scope is computed</h3>
          <p>Each row summarizes the initiative’s tracked items, completion, active work, blockers, and scope change.</p>
        </Popover>
      </div>

      {initiatives.length === 0 ? (
        <StatsEmptyState>No initiatives yet</StatsEmptyState>
      ) : (
        <ul className="space-y-3">
          {initiatives.map((init) => {
            const item = itemByName.get(init.name);
            if (!item) {
              const pct = init.total > 0 ? (init.completed / init.total) * 100 : 0;
              return (
                <li key={init.name} className="rounded-lg border border-slate-700/50 bg-slate-900/40 p-3">
                  <div className="mb-1 flex items-center justify-between">
                    <span className="text-sm font-medium text-slate-200">{init.name}</span>
                    <span className="text-xs text-slate-400">{Math.round(pct)}%</span>
                  </div>
                  <ProgressBar value={init.completed} max={init.total} color="bg-cyan-500" />
                  <div className="mt-1.5 flex gap-3 text-xs text-slate-500">
                    <span>{init.total} total</span>
                    <span>{init.in_progress} active</span>
                    {init.blocked > 0 && <span className="text-red-400">{init.blocked} blocked</span>}
                    {init.scope_creep !== 0 && (
                      <span className={init.scope_creep > 0 ? "text-amber-400" : "text-emerald-400"}>
                        scope {init.scope_creep > 0 ? "+" : ""}{Math.round(init.scope_creep * 100)}%
                      </span>
                    )}
                  </div>
                </li>
              );
            }
            return (
              <li key={init.name}>
                <InitiativeSummaryCard item={item} onOpen={() => navigate(initiativeDetailPath(init.name))} />
                <div className="mt-1 flex flex-wrap gap-2 px-1 text-[11px] text-slate-500">
                  <span>{init.total} tracked</span>
                  {init.scope_creep !== 0 && (
                    <span className={init.scope_creep > 0 ? "text-amber-400" : "text-emerald-400"}>
                      scope {init.scope_creep > 0 ? "+" : ""}{Math.round(init.scope_creep * 100)}%
                    </span>
                  )}
                  {init.blocked > 0 && <span className="text-red-400">{init.blocked} blocked</span>}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Modes tab
// ---------------------------------------------------------------------------

function ModesTab({ data }: { data: ModeStats }) {
  const usageEntries = Object.entries(data?.usage_by_mode ?? {}).sort(([a], [b]) => a.localeCompare(b));
  const phaseEntries = Object.entries(data?.phase_runs_by_mode ?? {}).sort(([a], [b]) => a.localeCompare(b));
  const profileEntries = Object.entries(data?.usage_by_profile ?? {}).sort(([, a], [, b]) => b - a);
  const syncEntries = Object.entries(data?.backlog_sync_by_mode ?? {}).sort(([a], [b]) => a.localeCompare(b));

  return (
    <div className="space-y-4" data-testid="stats-content-modes">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <StatCard label="Mode Switches" value={String(data?.mode_switch_count ?? 0)} icon={Layers} />
        <StatCard label="Profiles Used" value={String(profileEntries.length)} icon={Users} />
      </div>

      <div>
        <SectionLabel icon={Activity}>Current Mode Usage</SectionLabel>
        {usageEntries.length === 0 ? (
          <StatsEmptyState>No initiatives recorded yet</StatsEmptyState>
        ) : (
          <ul className="space-y-2">
            {usageEntries.map(([mode, count]) => (
              <li key={mode} className="flex items-center justify-between rounded-lg border border-slate-700/50 bg-slate-900/40 px-3 py-2 text-sm">
                <span className="text-slate-300">{formatModeLabel(mode)}</span>
                <span className="font-medium text-slate-100">{count}</span>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div>
        <SectionLabel icon={BarChart3}>Phase Runs</SectionLabel>
        {phaseEntries.length === 0 ? (
          <StatsEmptyState>No operating-mode phase runs yet</StatsEmptyState>
        ) : (
          <div className="space-y-3">
            {phaseEntries.map(([mode, phases]) => (
              <div key={mode} className="rounded-lg border border-slate-700/50 bg-slate-900/40 p-3">
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-sm font-medium text-slate-200">{formatModeLabel(mode)}</span>
                  <span className="text-xs text-slate-500">{sumValues(phases)} runs</span>
                </div>
                <div className="space-y-2">
                  {Object.entries(phases).sort(([a], [b]) => a.localeCompare(b)).map(([phase, count]) => (
                    <div key={phase} className="text-xs">
                      <div className="mb-1 flex justify-between">
                        <span className="text-slate-400">{formatModeLabel(phase)}</span>
                        <span className="text-slate-300">{count}</span>
                      </div>
                      <ProgressBar value={count} max={Math.max(1, sumValues(phases))} color="bg-cyan-500" />
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {Object.entries(data?.replan_rate_by_mode ?? {}).map(([mode, rate]) => (
          <StatCard
            key={`replan-${mode}`}
            label={`${formatModeLabel(mode)} Replan`}
            value={formatRate(rate.rate)}
            subtext={`n=${rate.sample_size}`}
          />
        ))}
        {Object.entries(data?.acceptance_rate_by_mode ?? {}).map(([mode, rate]) => (
          <StatCard
            key={`acceptance-${mode}`}
            label={`${formatModeLabel(mode)} Acceptance`}
            value={formatRate(rate.rate)}
            subtext={`n=${rate.sample_size}`}
          />
        ))}
      </div>

      <div>
        <SectionLabel icon={Users}>Profile Usage</SectionLabel>
        {profileEntries.length === 0 ? (
          <StatsEmptyState>No profile usage recorded yet</StatsEmptyState>
        ) : (
          <ul className="space-y-1">
            {profileEntries.map(([profile, count]) => (
              <li key={profile} className="flex items-center justify-between rounded px-2 py-1 text-sm hover:bg-slate-800/50">
                <span className="truncate text-slate-300">{profile}</span>
                <span className="ml-2 shrink-0 rounded bg-slate-700/60 px-1.5 py-0.5 text-xs text-slate-400">{count}</span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {syncEntries.length > 0 && (
        <div>
          <SectionLabel icon={ListChecks}>Backlog Sync</SectionLabel>
          <ul className="space-y-2">
            {syncEntries.map(([mode, sync]) => (
              <li key={mode} className="rounded-lg border border-slate-700/50 bg-slate-900/40 p-3 text-xs text-slate-400">
                <div className="mb-1 font-medium text-slate-200">{formatModeLabel(mode)}</div>
                <div className="flex flex-wrap gap-x-3 gap-y-1">
                  <span>{sync.events} events</span>
                  <span>{sync.items_completed} completed</span>
                  <span>{sync.items_created} created</span>
                  <span>{sync.items_updated} updated</span>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Sessions tab
// ---------------------------------------------------------------------------

function SessionsTab({ data, history }: { data: SessionStats; history: HistoryWindow }) {
  const threshold = minSample(history);
  const kindEntries = Object.entries(data?.sessions_by_kind ?? {}).sort(([a], [b]) => a.localeCompare(b));
  const statusEntries = Object.entries(data?.sessions_by_status ?? {}).sort(([a], [b]) => a.localeCompare(b));
  const proposalEntries = Object.entries(data?.proposal_apply_rate_by_kind ?? {}).sort(([a], [b]) => a.localeCompare(b));
  const artifactEntries = Object.entries(data?.artifacts_by_type ?? {}).sort(([a], [b]) => a.localeCompare(b));

  return (
    <div className="space-y-4" data-testid="stats-content-sessions">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <StatCard label="Sessions" value={String(data?.total_sessions ?? 0)} subtext={`${data?.active_sessions ?? 0} active`} icon={MessageSquare} />
        <StatsMetricCard
          label="Messages / Session"
          value={(data?.avg_messages_per_session ?? 0).toFixed(1)}
          sampleSize={data?.total_sessions ?? 0}
          minSample={1}
          sampleNoun="sessions"
          insufficientReason="No sessions recorded yet."
          icon={MessageSquare}
        />
        <StatsMetricCard
          label="Failed Sessions"
          value={formatRate(data?.failed_session_rate ?? 0)}
          sampleSize={data?.failed_session_sample_size ?? 0}
          minSample={Math.max(1, threshold)}
          sampleNoun="terminal sessions"
          insufficientReason={`Need at least ${threshold} terminal sessions.`}
          icon={AlertCircle}
        />
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <StatCard label="Backlog Artifacts" value={String(data?.session_created_backlog_items ?? 0)} subtext="created by sessions" icon={ListChecks} />
        <StatCard label="Initiative Artifacts" value={String(data?.session_created_initiatives ?? 0)} subtext="created by sessions" icon={Target} />
      </div>

      <StatsMetricCard
        label="Time to First Proposal"
        value={formatDurationSeconds(data?.avg_time_to_first_proposal_seconds ?? 0)}
        sampleSize={data?.first_proposal_sample_size ?? 0}
        minSample={Math.max(1, threshold)}
        sampleNoun="sessions with proposals"
        insufficientReason={`Need at least ${threshold} sessions with proposals.`}
        icon={Timer}
      />

      <div>
        <SectionLabel icon={MessageSquare}>Sessions By Kind</SectionLabel>
        {kindEntries.length === 0 ? (
          <StatsEmptyState>No agent sessions recorded yet</StatsEmptyState>
        ) : (
          <KeyValueList entries={kindEntries} formatKey={formatModeLabel} />
        )}
      </div>

      <div>
        <SectionLabel icon={Activity}>Current Status</SectionLabel>
        {statusEntries.length === 0 ? (
          <StatsEmptyState>No status data yet</StatsEmptyState>
        ) : (
          <KeyValueList entries={statusEntries} formatKey={formatModeLabel} />
        )}
      </div>

      <div>
        <SectionLabel icon={CheckCircle2}>Proposal Apply Rate</SectionLabel>
        {proposalEntries.length === 0 ? (
          <StatsEmptyState>No proposals recorded yet</StatsEmptyState>
        ) : (
          <div className="space-y-2">
            {proposalEntries.map(([kind, rate]) => (
              <div key={kind} className="text-xs">
                <div className="mb-1 flex justify-between">
                  <span className="text-slate-400">{formatModeLabel(kind)}</span>
                  <span className="text-emerald-300">{formatRate(rate.rate)} <span className="text-slate-500">(n={rate.sample_size})</span></span>
                </div>
                <ProgressBar value={rate.rate} max={1} color="bg-emerald-500" />
              </div>
            ))}
          </div>
        )}
      </div>

      <div>
        <SectionLabel icon={ListChecks}>Artifacts By Type</SectionLabel>
        {artifactEntries.length === 0 ? (
          <StatsEmptyState>No session artifacts recorded yet</StatsEmptyState>
        ) : (
          <KeyValueList entries={artifactEntries} formatKey={formatModeLabel} />
        )}
      </div>
    </div>
  );
}

function formatDurationSeconds(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "0m";
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${(seconds / 3600).toFixed(1)}h`;
}

function formatModeLabel(value: string): string {
  return value
    .split(/[-_]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function sumValues(values: Record<string, number>): number {
  return Object.values(values).reduce((sum, value) => sum + value, 0);
}
