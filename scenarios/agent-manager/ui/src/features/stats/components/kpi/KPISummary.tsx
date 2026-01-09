// KPI Summary - row of 5 key metric cards

import {
  Activity,
  Clock,
  DollarSign,
  CheckCircle2,
  ListTodo,
} from "lucide-react";
import { KPICard } from "./KPICard";
import { useStatsSummary } from "../../hooks/useStatsSummary";
import {
  formatPercent,
  formatCurrency,
  formatDuration,
  formatNumber,
} from "../../utils/formatters";
import { calculateThroughput, getWindowHours } from "../../utils/calculations";
import { useTimeWindow } from "../../hooks/useTimeWindow";

export function KPISummary() {
  const { data, isLoading, error } = useStatsSummary();
  const { preset } = useTimeWindow();
  const windowHours = getWindowHours(preset);

  const summary = data?.summary;
  const counts = summary?.statusCounts;
  const duration = summary?.duration;
  const cost = summary?.cost;

  // Calculate throughput (runs/hour)
  const throughput = summary?.runnerBreakdown
    ? summary.runnerBreakdown.reduce((sum, r) => sum + r.runCount, 0) / windowHours
    : 0;

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
      <KPICard
        title="Success Rate"
        value={summary ? formatPercent(summary.successRate) : "-"}
        subtitle="completed runs"
        icon={<CheckCircle2 className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
        variant={
          summary
            ? summary.successRate >= 0.9
              ? "success"
              : summary.successRate >= 0.7
              ? "warning"
              : "error"
            : "default"
        }
      />
      <KPICard
        title="Total Cost"
        value={cost ? formatCurrency(cost.totalCostUsd) : "-"}
        subtitle={cost ? `${formatNumber(cost.totalTokens)} tokens` : undefined}
        icon={<DollarSign className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
      />
      <KPICard
        title="Avg Duration"
        value={duration ? formatDuration(duration.avgMs) : "-"}
        subtitle={duration ? `p95: ${formatDuration(duration.p95Ms)}` : undefined}
        icon={<Clock className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
      />
      <KPICard
        title="Throughput"
        value={throughput > 0 ? `${throughput.toFixed(1)}/hr` : "-"}
        subtitle="runs per hour"
        icon={<Activity className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
      />
      <KPICard
        title="Queue"
        value={counts ? formatNumber(counts.pending + counts.running) : "-"}
        subtitle={
          counts
            ? `${counts.pending} pending, ${counts.running} running`
            : undefined
        }
        icon={<ListTodo className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
        variant={
          counts
            ? counts.pending > 10
              ? "warning"
              : "default"
            : "default"
        }
      />
    </div>
  );
}
