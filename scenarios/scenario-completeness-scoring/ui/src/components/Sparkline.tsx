import { cn } from "../lib/utils";

interface SparklineProps {
  data: number[];
  width?: number;
  height?: number;
  className?: string;
}

// [REQ:SCS-UI-006] Sparkline trend charts for score history
export function Sparkline({ data, width = 100, height = 24, className }: SparklineProps) {
  if (!data || data.length < 2) {
    return <span className="text-xs text-slate-500">No history</span>;
  }

  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;

  const points = data.map((value, index) => {
    const x = (index / (data.length - 1)) * width;
    const y = height - ((value - min) / range) * (height - 4) - 2;
    return `${x},${y}`;
  });

  const pathD = `M ${points.join(" L ")}`;
  const isImproving = data[data.length - 1] >= data[0];

  return (
    <svg
      width={width}
      height={height}
      className={cn("overflow-visible", className)}
      data-testid="sparkline"
    >
      <path
        d={pathD}
        fill="none"
        stroke={isImproving ? "#10b981" : "#ef4444"}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle
        cx={(data.length - 1) / (data.length - 1) * width}
        cy={height - ((data[data.length - 1] - min) / range) * (height - 4) - 2}
        r="2"
        fill={isImproving ? "#10b981" : "#ef4444"}
      />
    </svg>
  );
}
