// Mini sparkline bar showing recent status history
// Shared component for trends grid and detail modal
import { memo } from "react";
import { HealthStatus } from "../../lib/api";

interface StatusSparklineProps {
  statuses: HealthStatus[];
  maxBars?: number;
  barHeight?: number;
}

function StatusSparklineImpl({ statuses, maxBars = 12, barHeight = 16 }: StatusSparklineProps) {
  // Show last N statuses as small bars
  const displayStatuses = statuses.slice(0, maxBars);

  return (
    <div className="flex min-w-0 items-center gap-0.5">
      {displayStatuses.map((status, idx) => (
        <div
          key={idx}
          className={`w-1.5 rounded-sm transition-all ${
            status === "ok"
              ? "bg-accent-success"
              : status === "warning"
              ? "bg-accent-warning"
              : status === "critical"
              ? "bg-accent-danger"
              : "bg-text-muted/50"
          }`}
          style={{
            height: barHeight,
            opacity: 0.4 + (idx / displayStatuses.length) * 0.6,
          }}
        />
      ))}
      {displayStatuses.length === 0 && (
        <span className="text-xs text-text-muted">No data</span>
      )}
    </div>
  );
}

export const StatusSparkline = memo(StatusSparklineImpl);
