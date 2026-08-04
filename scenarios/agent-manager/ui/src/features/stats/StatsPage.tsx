// Stats Page - main page component for the Stats tab

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
import { ToolUsageAnalytics } from "./components/breakdown/ToolUsageAnalytics";
import { ErrorAnalysisSection } from "./components/errors/ErrorAnalysisSection";
import { RecurringWorkloadPanel } from "./components/workload/RecurringWorkloadPanel";
import { FallbackInsightsCard } from "./components/operational/FallbackInsightsCard";
import { ModelFailureAlertBanner } from "./components/operational/ModelFailureAlertBanner";
import { FrictionOverviewCard } from "./components/operational/FrictionOverviewCard";
import { onProfilerRender } from "../../lib/profiler";
import { HistoryBanner } from "../../components/stats/HistoryBanner";
import { fetchDurableRunVolume, statsQueryKeys } from "./api/statsClient";
import { useTimeWindow } from "./hooks/useTimeWindow";

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
    <div className="h-full overflow-y-auto overflow-x-hidden px-3 py-3 sm:px-6 lg:px-10 space-y-3 sm:space-y-4">
        {/* Header with controls */}
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-lg font-semibold">Statistics & Analytics</h2>
          <div className="flex items-center gap-3">
            <TimeWindowSelector />
            <ExportButton />
          </div>
        </div>

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

        {/* Error Analysis Section */}
        <ProfiledStatsSection id="Stats:ErrorAnalysisSection">
          <ErrorAnalysisSection />
        </ProfiledStatsSection>
        <ProfiledStatsSection id="Stats:RecurringWorkloadPanel">
          <RecurringWorkloadPanel />
        </ProfiledStatsSection>
    </div>
  );
}
