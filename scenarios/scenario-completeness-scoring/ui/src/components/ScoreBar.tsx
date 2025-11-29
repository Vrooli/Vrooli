import { cn } from "../lib/utils";

interface ScoreBarProps {
  label: string;
  value: number;
  max: number;
  className?: string;
}

// [REQ:SCS-UI-005] Score progress bars for each dimension
export function ScoreBar({ label, value, max, className }: ScoreBarProps) {
  const percentage = max > 0 ? Math.min(100, (value / max) * 100) : 0;

  const getBarColor = () => {
    if (percentage >= 80) return "bg-emerald-500";
    if (percentage >= 60) return "bg-amber-500";
    return "bg-red-500";
  };

  return (
    <div className={cn("space-y-1", className)} data-testid={`score-bar-${label.toLowerCase().replace(/\s+/g, "-")}`}>
      <div className="flex justify-between text-sm">
        <span className="text-slate-300">{label}</span>
        <span className="text-slate-400">
          {value.toFixed(1)}/{max} ({percentage.toFixed(0)}%)
        </span>
      </div>
      <div className="h-2 rounded-full bg-slate-700 overflow-hidden">
        <div
          className={cn("h-full rounded-full transition-all duration-300", getBarColor())}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}
