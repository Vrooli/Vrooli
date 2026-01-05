// Stats Page - main page component for the Stats tab

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

export function StatsPage() {
  return (
    <TimeWindowProvider defaultPreset="24h">
      <div className="space-y-6">
        {/* Header with controls */}
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">Statistics & Analytics</h2>
            <p className="text-sm text-muted-foreground">
              Performance metrics and trends for agent runs
            </p>
          </div>
          <div className="flex items-center gap-3">
            <TimeWindowSelector />
            <ExportButton />
          </div>
        </div>

        {/* KPI Summary Row */}
        <KPISummary />

        {/* Charts Row */}
        <div className="grid gap-6 lg:grid-cols-2">
          <RunStatusTrends />
          <CostDurationTrends />
        </div>

        {/* Tables Row */}
        <div className="grid gap-6 lg:grid-cols-2">
          <RunnerPerformanceTable />
          <ProfileActivityTable />
        </div>

        {/* Breakdowns Row */}
        <div className="grid gap-6 lg:grid-cols-2">
          <ModelUsageBreakdown />
          <ToolUsageAnalytics />
        </div>

        {/* Error Analysis Section */}
        <ErrorAnalysisSection />
      </div>
    </TimeWindowProvider>
  );
}
