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
    <Card className="p-4">
      <div className="flex items-center gap-3">
        <div className={`rounded-md p-2 ${toneClasses[tone]}`}>
          <Icon size={20} />
        </div>
        <div>
          <p className="text-2xl font-bold">{value}</p>
          <p className="text-sm text-text-muted">{title}</p>
        </div>
      </div>
    </Card>
  );
}
