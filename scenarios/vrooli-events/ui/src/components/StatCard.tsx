// DOC: docs/internal/COHERENCE-NOTES.md
import { type LucideIcon } from "lucide-react";
import { Panel } from "./Panel";

interface StatCardProps {
  label: string;
  value: string | number;
  icon: LucideIcon;
  detail?: string;
}

export function StatCard({ label, value, icon: Icon, detail }: StatCardProps) {
  return (
    <Panel className="p-4">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">{label}</span>
        <Icon className="h-4 w-4 text-[var(--text-faint)]" />
      </div>
      <p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p>
      {detail && <p className="mt-1 text-xs text-[var(--text-muted)]">{detail}</p>}
    </Panel>
  );
}
