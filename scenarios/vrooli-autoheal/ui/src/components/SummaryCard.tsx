// Summary card component for health statistics
// [REQ:UI-HEALTH-001]
import type { ElementType } from "react";

interface SummaryCardProps {
  title: string;
  value: number;
  icon: ElementType;
  color: string;
}

export function SummaryCard({ title, value, icon: Icon, color }: SummaryCardProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4">
      <div className="flex items-center gap-3">
        <div className={`p-2 rounded-lg ${color}`}>
          <Icon size={20} />
        </div>
        <div>
          <p className="text-2xl font-bold">{value}</p>
          <p className="text-sm text-slate-400">{title}</p>
        </div>
      </div>
    </div>
  );
}
