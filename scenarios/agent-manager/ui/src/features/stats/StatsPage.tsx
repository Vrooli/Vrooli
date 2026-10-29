// Run analytics panels, shown under Usage › Details.

import { Profiler, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { TimeWindowProvider } from "./hooks/useTimeWindow";
import { TimeWindowSelector } from "./components/controls/TimeWindowSelector";
import { ExportButton } from "./components/controls/ExportButton";
import { KPISummary } from "./components/kpi/KPISummary";
import { RunStatusTrends } from "./components/trends/RunStatusTrends";
import { CostDurationTrends } from "./components/trends/CostDurationTrends";
import { RunnerPerformanceTable } from "./components/tables/RunnerPerformanceTable";
import { ProfileActivityTable } from "./components/tables/ProfileActivityTable";
import { ModelUsageBreakdown } from "./components/breakdown/ModelUsageBreakdown";
import { RunClassBreakdown } from "./components/breakdown/RunClassBreakdown";
import { CostDistributionCard } from "./components/breakdown/CostDistributionCard";
import { ToolUsageAnalytics } from "./components/breakdown/ToolUsageAnalytics";
import { TokenAttributionBreakdown } from "./components/breakdown/TokenAttributionBreakdown";
import { ErrorAnalysisSection } from "./components/errors/ErrorAnalysisSection";
import { RecurringWorkloadPanel } from "./components/workload/RecurringWorkloadPanel";
import { FallbackInsightsCard } from "./components/operational/FallbackInsightsCard";
import { ModelFailureAlertBanner } from "./components/operational/ModelFailureAlertBanner";
import { FrictionOverviewCard } from "./components/operational/FrictionOverviewCard";
import { onProfilerRender } from "../../lib/profiler";
import { HistoryBanner } from "../../components/stats/HistoryBanner";
import { fetchDurableRunVolume, statsQueryKeys } from "./api/statsClient";
import { useTimeWindow } from "./hooks/useTimeWindow";
import { ErrorNotice } from "../../components/feedback/ErrorNotice";

function ProfiledStatsSection({ id, children }: { id: string; children: ReactNode }) {
  return (
    <Profiler id={id} onRender={onProfilerRender}>
      {children}
    </Profiler>
  );
}

export function StatsPage() {
  return <TimeWindowProvider defaultPreset="7d"><StatsPageContent /></TimeWindowProvider>;
}

function StatsPageContent() {
  const { filter } = useTimeWindow();
  const volume = useQuery({ queryKey: [...statsQueryKeys.summary(filter), "history"], queryFn: () => fetchDurableRunVolume(filter) });
  return (
    <div className="min-w-0 space-y-3 sm:space-y-4" data-testid="usage-details">
        {/* Header with controls */}
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-base font-semibold">Run analytics</h2>
          <div className="flex flex-wrap items-center gap-3">
            <TimeWindowSelector />
            <ExportButton />
          </div>
        </div>

        {/* Every panel reads the same Measures service; when its first read fails,
            one explanation replaces a wall of identical per-card errors. */}
        {volume.error && !volume.data ? (
          <ErrorNotice error={volume.error} subject="run analytics" onRetry={() => void volume.refetch()} testId="usage-details-error" />
        ) : <>
        {volume.data?.historyFloor && <HistoryBanner coverage={{ historyFloor: volume.data.historyFloor, outsideHistoryRunCount: volume.data.outsideHistoryRunCount }} testId="stats-history-banner" />}

        <ModelFailureAlertBanner />

        {/* KPI Summary Row */}
        <ProfiledStatsSection id="Stats:KPISummary">
          <KPISummary />
        </ProfiledStatsSection>

        {/* Fallback insights (typed event-derived) */}
        <ProfiledStatsSection id="Stats:FallbackInsightsCard">
          <FallbackInsightsCard />
        </ProfiledStatsSection>
        <ProfiledStatsSection id="Stats:FrictionOverviewCard">
          <FrictionOverviewCard />
        </ProfiledStatsSection>

        {/* Charts Row */}
        <div className="grid gap-3 sm:gap-4 lg:grid-cols-2">
          <ProfiledStatsSection id="Stats:RunStatusTrends">
            <RunStatusTrends />
          </ProfiledStatsSection>
          <ProfiledStatsSection id="Stats:CostDurationTrends">
            <CostDurationTrends />
          </ProfiledStatsSection>
        </div>

        <ProfiledStatsSection id="Stats:RunClassBreakdown">
          <RunClassBreakdown />
        </ProfiledStatsSection>

        <ProfiledStatsSection id="Stats:CostDistributionCard">
          <CostDistributionCard />
        </ProfiledStatsSection>

        {/* Tables Row */}
        <div className="grid gap-3 sm:gap-4 lg:grid-cols-2">
          <ProfiledStatsSection id="Stats:RunnerPerformanceTable">
            <RunnerPerformanceTable />
          </ProfiledStatsSection>
          <ProfiledStatsSection id="Stats:ProfileActivityTable">
            <ProfileActivityTable />
          </ProfiledStatsSection>
        </div>

        {/* Breakdowns Row */}
        <div className="grid gap-3 sm:gap-4 lg:grid-cols-2">
          <ProfiledStatsSection id="Stats:ModelUsageBreakdown">
            <ModelUsageBreakdown />
          </ProfiledStatsSection>
          <ProfiledStatsSection id="Stats:ToolUsageAnalytics">
            <ToolUsageAnalytics />
          </ProfiledStatsSection>
        </div>

        <ProfiledStatsSection id="Stats:TokenAttributionBreakdown">
          <TokenAttributionBreakdown />
        </ProfiledStatsSection>

        {/* Error Analysis Section */}
        <ProfiledStatsSection id="Stats:ErrorAnalysisSection">
          <ErrorAnalysisSection />
        </ProfiledStatsSection>
        <ProfiledStatsSection id="Stats:RecurringWorkloadPanel">
          <RecurringWorkloadPanel />
        </ProfiledStatsSection>
        </>}
    </div>
  );
}
