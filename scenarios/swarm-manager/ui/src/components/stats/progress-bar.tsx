import { cn } from "../../lib/utils";
import { toBarPercent } from "../../lib/stats-format-utils";

interface ProgressBarProps {
  value: number;
  max: number;
  color?: string;
  className?: string;
}

export function ProgressBar({
  value,
  max,
  color = "bg-cyan-500",
  className,
}: ProgressBarProps) {
  const width = toBarPercent(value, max);

  return (
    <div className={cn("h-2 w-full rounded-full bg-slate-800", className)}>
      <div className={cn("h-2 rounded-full transition-all", color)} style={{ width: `${width}%` }} />
    </div>
  );
}
