/**
 * StatCard displays a single statistic with icon, value, and optional trend.
 * Used in the dashboard header for key metrics.
 *
 * [REQ:LD-DASHBOARD-TIMELINE] - Summary statistics display
 */
import type { LucideIcon } from "lucide-react";
import { Card } from "../ui";

interface StatCardProps {
  label: string;
  value: string | number;
  icon: LucideIcon;
  trend?: string;
}

export function StatCard({ label, value, icon: Icon, trend }: StatCardProps) {
  return (
    <Card>
      <div className="flex items-center justify-between">
        <Icon className="w-5 h-5 text-slate-400" />
        {trend && <span className="text-xs text-emerald-400">{trend}</span>}
      </div>
      <p className="mt-3 text-2xl font-semibold text-slate-100">{value}</p>
      <p className="text-sm text-slate-500">{label}</p>
    </Card>
  );
}
