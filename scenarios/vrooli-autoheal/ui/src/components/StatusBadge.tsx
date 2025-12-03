// Status badge component with color coding
// [REQ:UI-HEALTH-002]
import type { HealthStatus } from "../lib/api";

interface StatusBadgeProps {
  status: HealthStatus;
}

const colors: Record<HealthStatus, string> = {
  ok: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
  warning: "bg-amber-500/20 text-amber-400 border-amber-500/30",
  critical: "bg-red-500/20 text-red-400 border-red-500/30",
};

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span className={`px-3 py-1 rounded-full text-sm font-medium border ${colors[status] || "bg-slate-500/20 text-slate-400"}`}>
      {status.toUpperCase()}
    </span>
  );
}
