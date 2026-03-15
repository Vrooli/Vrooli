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
  formatDuration,
  formatNumber,
} from "../../utils/formatters";
import { getWindowHours } from "../../utils/calculations";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import { formatUsdFixed } from "../../../../lib/currency";

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
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-5">
      <KPICard
        title="Success Rate"
        value={summary ? formatPercent(summary.successRate) : "-"}
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
        value={cost ? formatUsdFixed(cost.totalCostUsd, 2) : "-"}
        icon={<DollarSign className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
      />
      <KPICard
        title="Avg Duration"
        value={duration ? formatDuration(duration.avgMs) : "-"}
        icon={<Clock className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
      />
      <KPICard
        title="Throughput"
        value={throughput > 0 ? `${throughput.toFixed(1)}/hr` : "-"}
        icon={<Activity className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
      />
      <KPICard
        title="Queue"
        value={counts ? formatNumber(counts.pending + counts.running) : "-"}
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
