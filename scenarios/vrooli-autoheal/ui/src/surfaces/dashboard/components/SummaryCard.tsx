// Summary card component for health statistics
// [REQ:UI-HEALTH-001]
import type { ElementType } from "react";
import { Card } from "../../../shared/ui/primitives";

interface SummaryCardProps {
  title: string;
  value: number;
  icon: ElementType;
  tone: "neutral" | "success" | "warning" | "danger";
}

const toneClasses: Record<SummaryCardProps["tone"], string> = {
  neutral: "bg-surface-overlay/80 text-text-muted",
  success: "bg-accent-success/20 text-accent-success",
  warning: "bg-accent-warning/20 text-accent-warning",
  danger: "bg-accent-danger/20 text-accent-danger",
};

export function SummaryCard({ title, value, icon: Icon, tone }: SummaryCardProps) {
  return (
    <Card className="min-h-[5.25rem] min-w-0 p-3 sm:min-h-[5.5rem] sm:p-4">
      <div className="flex min-w-0 items-center gap-2.5 sm:gap-3">
        <div className={`shrink-0 rounded-md p-2 ${toneClasses[tone]}`}>
          <Icon size={20} className="h-5 w-5" />
        </div>
        <div className="min-w-0">
          <p className="truncate text-2xl font-bold leading-tight">{value}</p>
          <p className="truncate text-sm text-text-muted">{title}</p>
        </div>
      </div>
    </Card>
  );
}
