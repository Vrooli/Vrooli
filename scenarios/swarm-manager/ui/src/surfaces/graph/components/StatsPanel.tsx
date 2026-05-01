/**
 * StatsPanel - Floating panel displaying operational metrics.
 *
 * Follows the SettingsDrawer pattern: FloatingPanel + custom tab bar + per-tab content.
 * Data is fetched via React Query (useStats hook) with 60s auto-refresh while open.
 */

import { useState } from "react";
import { AlertCircle, Loader2 } from "lucide-react";
import { FloatingPanel } from "../../../components/ui/floating-panel";
import { cn } from "../../../lib/utils";
import { useStats } from "../../../hooks/useStats";
import { HistoryBanner } from "../../../components/stats/history-banner";
import { InsufficientDataCard } from "../../../components/stats/insufficient-data-card";
import { StatsMetricCard } from "../../../components/stats/stats-metric-card";
import {
  formatDelta,
  formatHours,
  formatRate,
  formatWeeksRemaining,
  toBarPercent,
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

// ---------------------------------------------------------------------------
// Tab config
// ---------------------------------------------------------------------------

const STATS_TABS: { id: StatsCategory; label: string }[] = [
  { id: "dashboard", label: "Dashboard" },
  { id: "throughput", label: "Throughput" },
  { id: "agent", label: "Agent" },
  { id: "timing", label: "Timing" },
  { id: "blocking", label: "Blocking" },
  { id: "scope", label: "Scope" },
  { id: "modes", label: "Modes" },
  { id: "sessions", label: "Sessions" },
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

interface StatsPanelProps {
  isOpen: boolean;
  onClose: () => void;
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function StatsPanel({ isOpen, onClose }: StatsPanelProps) {
  const [activeTab, setActiveTab] = useState<StatsCategory>("dashboard");
  const { data, isLoading, error } = useStats(isOpen);

  return (
    <FloatingPanel
      isOpen={isOpen}
      onClose={onClose}
      title="Stats"
      className="max-w-3xl"
      initialPosition={{ x: 340, y: 76 }}
      testId="stats-panel"
    >
      {/* Tab bar — same pattern as SettingsDrawer */}
      <div className="-mx-4 mb-4 flex overflow-x-auto border-b border-slate-700/50 px-4" role="tablist">
        {STATS_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={cn(
              "shrink-0 border-b-2 px-4 py-2 text-sm font-medium transition-colors",
              activeTab === tab.id
                ? "border-cyan-400 text-cyan-300"
                : "border-transparent text-slate-400 hover:text-slate-200",
            )}
            data-testid={`stats-tab-${tab.id}`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content area */}
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
    </FloatingPanel>
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
      return <ThroughputTab data={data.throughput} />;
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
// Shared presentational components
// ---------------------------------------------------------------------------

function StatCard({ label, value, subtext, testId }: { label: string; value: string; subtext?: string; testId?: string }) {
  return (
    <div className="rounded-lg border border-slate-700/50 bg-slate-900/40 p-3" data-testid={testId}>
      <p className="text-xs text-slate-400">{label}</p>
      <p className="text-lg font-semibold text-slate-100">{value}</p>
      {subtext && <p className="text-xs text-slate-500">{subtext}</p>}
    </div>
  );
}

function ProgressBar({ value, max, color = "bg-cyan-500" }: { value: number; max: number; color?: string }) {
  const width = toBarPercent(value, max);
  return (
    <div className="h-2 w-full rounded-full bg-slate-800">
      <div className={cn("h-2 rounded-full transition-all", color)} style={{ width: `${width}%` }} />
    </div>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <h3 className="mb-2 text-xs font-medium uppercase tracking-wider text-slate-500">{children}</h3>;
}

// ---------------------------------------------------------------------------
// Dashboard tab
// ---------------------------------------------------------------------------

function DashboardTab({ data, eventCount, history }: { data: DashboardStats; eventCount: number; history: HistoryWindow }) {
  const maxCompleted = data.velocity_trend.length > 0
    ? Math.max(...data.velocity_trend.map((p) => p.completed), 1)
    : 1;

  // The estimated-remaining pill is based on the last 4 full weeks of
  // velocity; if we have <4 non-zero weeks of history the estimate is
  // very noisy, so render an insufficient-data card instead.
  const velocityReady = data.velocity_weeks_covered >= 4;

  return (
    <div className="space-y-4" data-testid="stats-content-dashboard">
      <div className="grid grid-cols-3 gap-3">
        <StatCard label="Backlog" value={String(data.total_backlog_size)} testId="stat-backlog-size" />
        <StatCard label="Completed" value={String(data.total_completed_all_time)} subtext="all time" testId="stat-completed-all-time" />
        {velocityReady ? (
          <StatCard label="Est. Remaining" value={formatWeeksRemaining(data.estimated_weeks_remaining)} testId="stat-weeks-remaining" />
        ) : (
          <InsufficientDataCard
            label="Est. Remaining"
            reason="Need at least 4 weeks of completed work."
            have={data.velocity_weeks_covered}
            required={4}
            testId="stat-weeks-remaining"
          />
        )}
      </div>

      <div>
        <SectionLabel>Velocity (last {data.velocity_trend.length} weeks)</SectionLabel>
        {data.velocity_trend.length === 0 ? (
          <p className="text-sm text-slate-500">No velocity data yet</p>
        ) : (
          <div className="flex items-end gap-1" style={{ height: 80 }}>
            {data.velocity_trend.map((point) => (
              <div
                key={point.week_start}
                className="flex flex-1 flex-col items-center gap-1"
              >
                <div
                  className="w-full rounded-t bg-cyan-500/70"
                  style={{ height: `${toBarPercent(point.completed, maxCompleted)}%` }}
                  title={`${point.week_start}: ${point.completed} completed`}
                />
                <span className="text-[10px] text-slate-500">{point.completed}</span>
              </div>
            ))}
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

function ThroughputTab({ data }: { data: ThroughputStats }) {
  return (
    <div className="space-y-3" data-testid="stats-content-throughput">
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
            <td className={cn("py-1 font-medium", data.net_delta_7_days > 0 ? "text-amber-400" : data.net_delta_7_days < 0 ? "text-emerald-400" : "text-slate-300")}>
              {formatDelta(data.net_delta_7_days)}
            </td>
            <td className={cn("py-1 font-medium", data.net_delta_30_days > 0 ? "text-amber-400" : data.net_delta_30_days < 0 ? "text-emerald-400" : "text-slate-300")}>
              {formatDelta(data.net_delta_30_days)}
            </td>
          </tr>
        </tbody>
      </table>
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
      <div className="grid grid-cols-2 gap-3">
        <StatCard
          label="Total Executions"
          value={String(data.total_executions)}
          subtext={`${data.completed_count} completed · ${data.failed_count} failed`}
        />
        <StatsMetricCard
          label="Avg Duration"
          value={`${data.avg_execution_minutes.toFixed(1)} min`}
          sampleSize={data.execution_duration_samples}
          minSample={threshold}
          sampleNoun="completed runs"
          insufficientReason={`Need at least ${threshold} finished runs.`}
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
      <SectionLabel>Decision Recommendations</SectionLabel>
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
      <SectionLabel>Lead Time (created → complete)</SectionLabel>
      <div className="grid grid-cols-2 gap-3">
        <StatsMetricCard
          label="Average"
          value={formatHours(data.avg_lead_time_hours)}
          sampleSize={data.lead_time_sample_size}
          minSample={threshold}
          sampleNoun="items"
          insufficientReason={`Need at least ${threshold} items tracked from creation to completion.`}
        />
        <StatsMetricCard
          label="Median"
          value={formatHours(data.median_lead_time_hours)}
          sampleSize={data.lead_time_sample_size}
          minSample={threshold}
          sampleNoun="items"
          insufficientReason={`Need at least ${threshold} items tracked from creation to completion.`}
        />
      </div>

      <SectionLabel>Execution Duration (running → complete)</SectionLabel>
      <div className="grid grid-cols-2 gap-3">
        <StatsMetricCard
          label="Average"
          value={`${data.avg_execution_minutes.toFixed(1)} min`}
          sampleSize={data.execution_duration_samples}
          minSample={threshold}
          sampleNoun="finished runs"
          insufficientReason={`Need at least ${threshold} finished executions.`}
        />
        <StatsMetricCard
          label="Median"
          value={`${data.median_execution_minutes.toFixed(1)} min`}
          sampleSize={data.execution_duration_samples}
          minSample={threshold}
          sampleNoun="finished runs"
          insufficientReason={`Need at least ${threshold} finished executions.`}
        />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Blocking tab
// ---------------------------------------------------------------------------

function BlockingTab({ data }: { data: BlockingStats }) {
  return (
    <div className="space-y-4" data-testid="stats-content-blocking">
      <div className="grid grid-cols-3 gap-3">
        <StatCard label="Blocked" value={String(data.currently_blocked)} />
        <StatCard label="Blocked %" value={formatRate(data.blocked_ratio)} />
        <StatCard label="Avg Block Time" value={formatHours(data.avg_block_hours)} />
      </div>

      <div>
        <SectionLabel>Top Blocking Reasons</SectionLabel>
        {data.top_reasons.length === 0 ? (
          <p className="text-sm text-slate-500">No blocking reasons recorded</p>
        ) : (
          <ul className="space-y-1">
            {data.top_reasons.map((r) => (
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
  return (
    <div className="space-y-4" data-testid="stats-content-scope">
      {data.max_dependency_depth > 0 && (
        <p className="text-xs text-slate-500">Max dependency depth: {data.max_dependency_depth}</p>
      )}

      {data.initiatives.length === 0 ? (
        <p className="text-sm text-slate-500">No initiatives yet</p>
      ) : (
        <ul className="space-y-3">
          {data.initiatives.map((init) => {
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
      <div className="grid grid-cols-2 gap-3">
        <StatCard label="Mode Switches" value={String(data?.mode_switch_count ?? 0)} />
        <StatCard label="Profiles Used" value={String(profileEntries.length)} />
      </div>

      <div>
        <SectionLabel>Current Mode Usage</SectionLabel>
        {usageEntries.length === 0 ? (
          <p className="text-sm text-slate-500">No initiatives recorded yet</p>
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
        <SectionLabel>Phase Runs</SectionLabel>
        {phaseEntries.length === 0 ? (
          <p className="text-sm text-slate-500">No operating-mode phase runs yet</p>
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

      <div className="grid grid-cols-2 gap-3">
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
        <SectionLabel>Profile Usage</SectionLabel>
        {profileEntries.length === 0 ? (
          <p className="text-sm text-slate-500">No profile usage recorded yet</p>
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
          <SectionLabel>Backlog Sync</SectionLabel>
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
      <div className="grid grid-cols-3 gap-3">
        <StatCard label="Sessions" value={String(data?.total_sessions ?? 0)} subtext={`${data?.active_sessions ?? 0} active`} />
        <StatsMetricCard
          label="Messages / Session"
          value={(data?.avg_messages_per_session ?? 0).toFixed(1)}
          sampleSize={data?.total_sessions ?? 0}
          minSample={1}
          sampleNoun="sessions"
          insufficientReason="No sessions recorded yet."
        />
        <StatsMetricCard
          label="Failed Sessions"
          value={formatRate(data?.failed_session_rate ?? 0)}
          sampleSize={data?.failed_session_sample_size ?? 0}
          minSample={Math.max(1, threshold)}
          sampleNoun="terminal sessions"
          insufficientReason={`Need at least ${threshold} terminal sessions.`}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <StatCard label="Backlog Artifacts" value={String(data?.session_created_backlog_items ?? 0)} subtext="created by sessions" />
        <StatCard label="Initiative Artifacts" value={String(data?.session_created_initiatives ?? 0)} subtext="created by sessions" />
      </div>

      <StatsMetricCard
        label="Time to First Proposal"
        value={formatDurationSeconds(data?.avg_time_to_first_proposal_seconds ?? 0)}
        sampleSize={data?.first_proposal_sample_size ?? 0}
        minSample={Math.max(1, threshold)}
        sampleNoun="sessions with proposals"
        insufficientReason={`Need at least ${threshold} sessions with proposals.`}
      />

      <div>
        <SectionLabel>Sessions By Kind</SectionLabel>
        {kindEntries.length === 0 ? (
          <p className="text-sm text-slate-500">No agent sessions recorded yet</p>
        ) : (
          <KeyValueList entries={kindEntries} />
        )}
      </div>

      <div>
        <SectionLabel>Current Status</SectionLabel>
        {statusEntries.length === 0 ? (
          <p className="text-sm text-slate-500">No status data yet</p>
        ) : (
          <KeyValueList entries={statusEntries} />
        )}
      </div>

      <div>
        <SectionLabel>Proposal Apply Rate</SectionLabel>
        {proposalEntries.length === 0 ? (
          <p className="text-sm text-slate-500">No proposals recorded yet</p>
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
        <SectionLabel>Artifacts By Type</SectionLabel>
        {artifactEntries.length === 0 ? (
          <p className="text-sm text-slate-500">No session artifacts recorded yet</p>
        ) : (
          <KeyValueList entries={artifactEntries} />
        )}
      </div>
    </div>
  );
}

function KeyValueList({ entries }: { entries: [string, number][] }) {
  return (
    <ul className="space-y-1">
      {entries.map(([key, value]) => (
        <li key={key} className="flex items-center justify-between rounded px-2 py-1 text-sm hover:bg-slate-800/50">
          <span className="truncate text-slate-300">{formatModeLabel(key)}</span>
          <span className="ml-2 shrink-0 rounded bg-slate-700/60 px-1.5 py-0.5 text-xs text-slate-400">{value}</span>
        </li>
      ))}
    </ul>
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
