import { TrendingUp, TrendingDown, Minus, AlertTriangle, HelpCircle } from "lucide-react";
import { cn } from "../lib/utils";
import type { TrendAnalysis } from "../lib/api";

interface TrendIndicatorProps {
  trend: TrendAnalysis["trend"];
  delta?: number;
  className?: string;
}

// [REQ:SCS-UI-006] Trend indicators for score changes
export function TrendIndicator({ trend, delta, className }: TrendIndicatorProps) {
  const config = {
    improving: {
      icon: TrendingUp,
      color: "text-emerald-400",
      label: "Improving",
    },
    declining: {
      icon: TrendingDown,
      color: "text-red-400",
      label: "Declining",
    },
    stable: {
      icon: Minus,
      color: "text-slate-400",
      label: "Stable",
    },
    stalled: {
      icon: AlertTriangle,
      color: "text-amber-400",
      label: "Stalled",
    },
    insufficient_data: {
      icon: HelpCircle,
      color: "text-slate-500",
      label: "No data",
    },
  };

  const { icon: Icon, color, label } = config[trend] || config.insufficient_data;

  return (
    <div
      className={cn("inline-flex items-center gap-1", color, className)}
      data-testid="trend-indicator"
      title={label}
    >
      <Icon className="h-4 w-4" />
      {delta !== undefined && delta !== 0 && (
        <span className="text-sm">
          {delta > 0 ? "+" : ""}
          {delta}
        </span>
      )}
    </div>
  );
}
