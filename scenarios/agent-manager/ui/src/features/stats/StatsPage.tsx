// Stats Page - main page component for the Stats tab

import { Profiler, type ReactNode } from "react";
import { TimeWindowProvider } from "./hooks/useTimeWindow";
import { TimeWindowSelector } from "./components/controls/TimeWindowSelector";
import { ExportButton } from "./components/controls/ExportButton";
import { KPISummary } from "./components/kpi/KPISummary";
import { RunStatusTrends } from "./components/trends/RunStatusTrends";
import { CostDurationTrends } from "./components/trends/CostDurationTrends";
import { RunnerPerformanceTable } from "./components/tables/RunnerPerformanceTable";
import { ProfileActivityTable } from "./components/tables/ProfileActivityTable";
import { ModelUsageBreakdown } from "./components/breakdown/ModelUsageBreakdown";
import { ToolUsageAnalytics } from "./components/breakdown/ToolUsageAnalytics";
import { ErrorAnalysisSection } from "./components/errors/ErrorAnalysisSection";
import { onProfilerRender } from "../../lib/profiler";

function ProfiledStatsSection({ id, children }: { id: string; children: ReactNode }) {
  return (
    <Profiler id={id} onRender={onProfilerRender}>
      {children}
    </Profiler>
  );
}

export function StatsPage() {
  return (
    <TimeWindowProvider defaultPreset="24h">
      <div className="h-full overflow-y-auto overflow-x-hidden px-3 py-3 sm:px-6 lg:px-10 space-y-3 sm:space-y-4">
        {/* Header with controls */}
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-lg font-semibold">Statistics & Analytics</h2>
          <div className="flex items-center gap-3">
            <TimeWindowSelector />
            <ExportButton />
          </div>
        </div>

        {/* KPI Summary Row */}
        <ProfiledStatsSection id="Stats:KPISummary">
          <KPISummary />
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
      </div>
    </TimeWindowProvider>
  );
}
