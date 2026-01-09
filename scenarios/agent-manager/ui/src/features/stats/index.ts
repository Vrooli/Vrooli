// Public exports for the stats feature module

// Main page
export { StatsPage } from "./StatsPage";

// API types
export type {
  StatsFilter,
  TimePreset,
  TimeWindow,
  RunStatusCounts,
  DurationStats,
  CostStats,
  RunnerBreakdown,
  ProfileBreakdown,
  ModelBreakdown,
  ToolUsageStats,
  ErrorPattern,
  TimeSeriesBucket,
  StatsSummary,
} from "./api/types";

// API client
export {
  fetchStatsSummary,
  fetchStatusDistribution,
  fetchSuccessRate,
  fetchDurationStats,
  fetchCostStats,
  fetchRunnerBreakdown,
  fetchProfileBreakdown,
  fetchModelBreakdown,
  fetchToolUsage,
  fetchErrorPatterns,
  fetchTimeSeries,
  statsQueryKeys,
} from "./api/statsClient";

// Hooks
export { useStatsSummary } from "./hooks/useStatsSummary";
export { useRunTrends } from "./hooks/useRunTrends";
export { useCostTrends } from "./hooks/useCostTrends";
export { useRunnerPerformance } from "./hooks/useRunnerPerformance";
export { useProfileBreakdown } from "./hooks/useProfileBreakdown";
export { useModelBreakdown } from "./hooks/useModelBreakdown";
export { useToolUsage } from "./hooks/useToolUsage";
export { useErrorAnalysis } from "./hooks/useErrorAnalysis";
export {
  TimeWindowProvider,
  useTimeWindow,
  useStatsFilter,
  getPresetLabel,
  getPresetShortLabel,
} from "./hooks/useTimeWindow";

// Components - Controls
export { TimeWindowSelector } from "./components/controls/TimeWindowSelector";
export { ExportButton } from "./components/controls/ExportButton";

// Components - KPI
export { KPICard } from "./components/kpi/KPICard";
export { KPISummary } from "./components/kpi/KPISummary";

// Components - Trends
export { RunStatusTrends } from "./components/trends/RunStatusTrends";
export { CostDurationTrends } from "./components/trends/CostDurationTrends";

// Components - Tables
export { RunnerPerformanceTable } from "./components/tables/RunnerPerformanceTable";
export { ProfileActivityTable } from "./components/tables/ProfileActivityTable";

// Components - Breakdowns
export { ModelUsageBreakdown } from "./components/breakdown/ModelUsageBreakdown";
export { ToolUsageAnalytics } from "./components/breakdown/ToolUsageAnalytics";

// Components - Errors
export { ErrorAnalysisSection } from "./components/errors/ErrorAnalysisSection";

// Utils
export * from "./utils/formatters";
export * from "./utils/calculations";
export * from "./utils/chartConfig";
