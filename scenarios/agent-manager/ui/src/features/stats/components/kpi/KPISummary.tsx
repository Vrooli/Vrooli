// KPI Summary - row of 5 key metric cards

import {
  Activity,
  Clock,
  DollarSign,
  CheckCircle2,
  ListTodo,
} from "lucide-react";
import { KPICard } from "./KPICard";
import { useQuery } from "@tanstack/react-query";
import { fetchDurableRunCost, fetchDurableRunCycleTime, fetchDurableRunStatusDistribution, fetchDurableRunSuccess, fetchDurableRunVolume, statsQueryKeys } from "../../api/statsClient";
import {
  formatPercent,
  formatDuration,
} from "../../utils/formatters";
import { getWindowHours } from "../../utils/calculations";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import { formatUsdFixed } from "../../../../lib/currency";

export function KPISummary() {
  const { preset, filter } = useTimeWindow();
  const windowHours = getWindowHours(preset);
  const success = useQuery({ queryKey: [...statsQueryKeys.successRate(filter), "durable"] as const, queryFn: () => fetchDurableRunSuccess(filter) });
  const cost = useQuery({ queryKey: [...statsQueryKeys.cost(filter), "durable"] as const, queryFn: () => fetchDurableRunCost(filter) });
  const duration = useQuery({ queryKey: [...statsQueryKeys.duration(filter), "durable"] as const, queryFn: () => fetchDurableRunCycleTime(filter) });
  const volume = useQuery({ queryKey: [...statsQueryKeys.summary(filter), "volume", "durable"] as const, queryFn: () => fetchDurableRunVolume(filter) });
  const statuses = useQuery({ queryKey: [...statsQueryKeys.statusDistribution(filter), "durable"] as const, queryFn: () => fetchDurableRunStatusDistribution(filter) });
  const isLoading = success.isLoading || cost.isLoading || duration.isLoading || volume.isLoading || statuses.isLoading;
  const error = success.error ?? cost.error ?? duration.error ?? volume.error ?? statuses.error;
  const throughput = volume.data ? volume.data.totalRuns / windowHours : 0;
  const queued = statuses.data?.rows.reduce((count, row) => count + (row.status === "pending" || row.status === "starting" ? row.count : 0), 0) ?? 0;

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-5">
      <KPICard
        title="Success Rate"
        value={success.data ? formatPercent(success.data.rate) : "-"}
        icon={<CheckCircle2 className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
        variant={
          success.data
            ? success.data.rate >= 0.9
              ? "success"
              : success.data.rate >= 0.7
              ? "warning"
              : "error"
            : "default"
        }
      />
      <KPICard
        title="Total Cost"
        value={cost.data ? formatUsdFixed(cost.data.totalCostUsd, 2) : "-"}
        icon={<DollarSign className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
      />
      <KPICard
        title="Avg Duration"
        value={duration.data ? formatDuration(duration.data.rate) : "-"}
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
        value={statuses.data ? String(queued) : "-"}
        icon={<ListTodo className="h-4 w-4" />}
        loading={isLoading}
        error={error?.message}
        variant="default"
      />
    </div>
  );
}
