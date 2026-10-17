// KPI Summary - row of 5 key metric cards

import {
  Activity,
  Clock,
  DollarSign,
  CheckCircle2,
  ListTodo,
  Coins,
} from "lucide-react";
import { KPICard } from "./KPICard";
import { useQuery } from "@tanstack/react-query";
import { fetchDurableRunCost, fetchDurableRunCycleTime, fetchDurableRunStatusDistribution, fetchDurableRunSuccess, fetchDurableRunVolume, statsQueryKeys } from "../../api/statsClient";
import {
  formatPercent,
  formatDuration,
  formatTokens,
} from "../../utils/formatters";
import { getWindowHours } from "../../utils/calculations";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import { MeasureFrame } from "../measure/MeasureFrame";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";

export function KPISummary() {
  const { preset, filter } = useTimeWindow();
  const definitions = useMeasureDefinitions();
  const windowHours = getWindowHours(preset);
  const success = useQuery({ queryKey: [...statsQueryKeys.successRate(filter), "durable"] as const, queryFn: () => fetchDurableRunSuccess(filter) });
  const cost = useQuery({ queryKey: [...statsQueryKeys.cost(filter), "durable"] as const, queryFn: () => fetchDurableRunCost(filter) });
  const duration = useQuery({ queryKey: [...statsQueryKeys.duration(filter), "durable"] as const, queryFn: () => fetchDurableRunCycleTime(filter) });
  const volume = useQuery({ queryKey: [...statsQueryKeys.summary(filter), "volume", "durable"] as const, queryFn: () => fetchDurableRunVolume(filter) });
  const statuses = useQuery({ queryKey: [...statsQueryKeys.statusDistribution(filter), "durable"] as const, queryFn: () => fetchDurableRunStatusDistribution(filter) });
  const throughput = volume.data ? volume.data.totalRuns / windowHours : 0;
  const queued = statuses.data?.rows.reduce((count, row) => count + (row.status === "pending" || row.status === "starting" ? row.count : 0), 0) ?? 0;
  const chargeLabel = cost.data ? formatCharge(cost.data.chargeByBasis ?? []) : "-";

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-6">
      <MeasureFrame label="Success rate" result={success.data} definition={definitions.data?.find((item) => item.id === "throughput.run_success_rate")} loading={success.isLoading} error={success.error?.message}>
      <KPICard
        title="Success Rate"
        value={success.data ? formatPercent(success.data.rate) : "-"}
        icon={<CheckCircle2 className="h-4 w-4" />}
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
      </MeasureFrame>
      <MeasureFrame label="Tokens" result={cost.data} definition={definitions.data?.find((item) => item.id === "throughput.run_cost")} loading={cost.isLoading} error={cost.error?.message}>
        <KPICard
          title="Tokens"
          value={cost.data ? formatTokens(cost.data.totalTokens) : "-"}
          subtitle={cost.data ? `in ${formatTokens(cost.data.inputTokens)} · out ${formatTokens(cost.data.outputTokens)} · cache ${formatTokens(cost.data.cacheReadTokens)}` : undefined}
          icon={<Coins className="h-4 w-4" />}
        />
      </MeasureFrame>
      <MeasureFrame label="Total charge" result={cost.data} definition={definitions.data?.find((item) => item.id === "throughput.run_cost")} loading={cost.isLoading} error={cost.error?.message}>
      <KPICard
        title="Total Cost"
        value={chargeLabel}
        subtitle={cost.data?.chargeByBasis?.length ? cost.data.chargeByBasis.map((item) => `${item.basis}: ${item.runCount} runs`).join(" · ") : undefined}
        icon={<DollarSign className="h-4 w-4" />}
      />
      </MeasureFrame>
      <MeasureFrame label="Average duration" result={duration.data} definition={definitions.data?.find((item) => item.id === "throughput.run_cycle_time")} loading={duration.isLoading} error={duration.error?.message}>
      <KPICard
        title="Avg Duration"
        value={duration.data ? formatDuration(duration.data.rate) : "-"}
        icon={<Clock className="h-4 w-4" />}
      />
      </MeasureFrame>
      <MeasureFrame label="Throughput" result={volume.data} definition={definitions.data?.find((item) => item.id === "throughput.run_volume")} loading={volume.isLoading} error={volume.error?.message}>
      <KPICard
        title="Throughput"
        value={throughput > 0 ? `${throughput.toFixed(1)}/hr` : "-"}
        icon={<Activity className="h-4 w-4" />}
      />
      </MeasureFrame>
      <MeasureFrame label="Queue" result={statuses.data} definition={definitions.data?.find((item) => item.id === "throughput.run_status_distribution")} loading={statuses.isLoading} error={statuses.error?.message}>
      <KPICard
        title="Queue"
        value={statuses.data ? String(queued) : "-"}
        icon={<ListTodo className="h-4 w-4" />}
        variant="default"
      />
      </MeasureFrame>
    </div>
  );
}

function formatCharge(rows: Array<{ basis: string; chargeMicroUsd: number; chargeReason: string }>): string {
  if (rows.length === 0) return "Billing mode not declared";
  if (rows.length > 1) return rows.map((row) => `${row.basis} ${(row.chargeMicroUsd / 1_000_000).toFixed(2)}`).join(" + ");
  const row = rows[0];
  if (!row) return "Billing mode not declared";
  if (row.basis === "unknown" || row.basis === "unpriced") return row.basis === "unpriced" ? `Not priced${row.chargeReason ? ` (${row.chargeReason})` : ""}` : "Billing mode not declared";
  if (row.basis === "subscription") return `Covered by subscription${row.chargeReason ? ` (${row.chargeReason})` : ""}`;
  return `$${(row.chargeMicroUsd / 1_000_000).toFixed(2)} (${row.basis})`;
}
