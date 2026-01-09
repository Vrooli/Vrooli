// Mini sparkline bar showing recent status history
// Shared component for trends grid and detail modal
import { HealthStatus } from "../lib/api";

interface StatusSparklineProps {
  statuses: HealthStatus[];
  maxBars?: number;
  barHeight?: number;
}

export function StatusSparkline({ statuses, maxBars = 12, barHeight = 16 }: StatusSparklineProps) {
  // Show last N statuses as small bars
  const displayStatuses = statuses.slice(0, maxBars);

  return (
    <div className="flex items-center gap-0.5">
      {displayStatuses.map((status, idx) => (
        <div
          key={idx}
          className={`w-1.5 rounded-sm transition-all ${
            status === "ok"
              ? "bg-emerald-500"
              : status === "warning"
              ? "bg-amber-500"
              : "bg-red-500"
          }`}
          style={{
            height: barHeight,
            opacity: 0.4 + (idx / displayStatuses.length) * 0.6,
          }}
        />
      ))}
      {displayStatuses.length === 0 && (
        <span className="text-xs text-slate-500">No data</span>
      )}
    </div>
  );
}
