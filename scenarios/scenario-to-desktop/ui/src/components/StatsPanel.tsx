import { useQuery } from "@tanstack/react-query";
import { fetchSystemStatus } from "../lib/api";

export function StatsPanel() {
  const { data } = useQuery({
    queryKey: ["system-status"],
    queryFn: fetchSystemStatus,
    refetchInterval: 30000 // Refresh every 30 seconds
  });

  const stats = data?.statistics || {
    total_builds: 0,
    active_builds: 0,
    completed_builds: 0,
    failed_builds: 0
  };

  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
      <StatCard
        label="Total Builds"
        value={stats.total_builds}
        className="bg-slate-800/50"
      />
      <StatCard
        label="Active Builds"
        value={stats.active_builds}
        className="bg-blue-900/30 border-blue-700"
      />
      <StatCard
        label="Completed"
        value={stats.completed_builds}
        className="bg-green-900/30 border-green-700"
      />
      <StatCard
        label="Failed"
        value={stats.failed_builds}
        className="bg-red-900/30 border-red-700"
      />
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: number;
  className?: string;
}

function StatCard({ label, value, className }: StatCardProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center rounded-lg border border-white/10 p-4 ${className || ""}`}
    >
      <div className="text-3xl font-bold text-slate-50">{value}</div>
      <div className="text-sm text-slate-400">{label}</div>
    </div>
  );
}
