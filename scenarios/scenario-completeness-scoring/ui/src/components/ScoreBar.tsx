import { useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import { cn } from "../lib/utils";

interface ScoreBarProps {
  label: string;
  value: number;
  max: number;
  className?: string;
  details?: Record<string, number>;
}

// [REQ:SCS-UI-005] Score progress bars for each dimension with expandable details
export function ScoreBar({ label, value, max, className, details }: ScoreBarProps) {
  const [expanded, setExpanded] = useState(false);
  const percentage = max > 0 ? Math.min(100, (value / max) * 100) : 0;
  const hasDetails = details && Object.keys(details).length > 0;

  const getBarColor = () => {
    if (percentage >= 80) return "bg-emerald-500";
    if (percentage >= 60) return "bg-amber-500";
    return "bg-red-500";
  };

  const formatDetailKey = (key: string) => {
    return key
      .replace(/_/g, " ")
      .replace(/\b\w/g, (c) => c.toUpperCase());
  };

  return (
    <div className={cn("space-y-1", className)} data-testid={`score-bar-${label.toLowerCase().replace(/\s+/g, "-")}`}>
      <button
        className={cn(
          "w-full flex justify-between items-center text-sm",
          hasDetails && "cursor-pointer hover:opacity-80"
        )}
        onClick={() => hasDetails && setExpanded(!expanded)}
        disabled={!hasDetails}
        type="button"
      >
        <span className="text-slate-300 flex items-center gap-1">
          {label}
          {hasDetails && (
            expanded ? (
              <ChevronUp className="h-3.5 w-3.5 text-slate-500" />
            ) : (
              <ChevronDown className="h-3.5 w-3.5 text-slate-500" />
            )
          )}
        </span>
        <span className="text-slate-400">
          {value.toFixed(1)}/{max} ({percentage.toFixed(0)}%)
        </span>
      </button>
      <div className="h-2 rounded-full bg-slate-700 overflow-hidden">
        <div
          className={cn("h-full rounded-full transition-all duration-300", getBarColor())}
          style={{ width: `${percentage}%` }}
        />
      </div>
      {/* Expandable detail breakdown */}
      {expanded && hasDetails && (
        <div className="mt-2 ml-2 pl-2 border-l-2 border-slate-700 space-y-1.5" data-testid="score-bar-details">
          {Object.entries(details).map(([key, val]) => {
            const isDeduction = val < 0;
            return (
              <div key={key} className="flex justify-between text-xs">
                <span className="text-slate-400">{formatDetailKey(key)}</span>
                <span className={cn(
                  isDeduction ? "text-red-400" : "text-slate-300"
                )}>
                  {isDeduction ? val.toFixed(1) : `+${val.toFixed(1)}`}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
