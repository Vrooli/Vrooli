/**
 * StatusBadge displays the health/activity status of domains and services.
 * Used throughout the dashboard to indicate current state.
 *
 * [REQ:LD-DOMAIN-HEALTH] - Visual indicator for domain health status
 */

interface StatusBadgeProps {
  status: string;
}

const statusStyles: Record<string, string> = {
  healthy: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
  active: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
  degraded: "bg-amber-500/20 text-amber-400 border-amber-500/30",
  inactive: "bg-slate-500/20 text-slate-400 border-slate-500/30",
  unhealthy: "bg-red-500/20 text-red-400 border-red-500/30",
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const styles = statusStyles[status] || "bg-slate-500/20 text-slate-400 border-slate-500/30";

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${styles}`}>
      {status}
    </span>
  );
}
