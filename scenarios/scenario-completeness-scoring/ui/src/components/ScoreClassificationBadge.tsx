import { cn } from "../lib/utils";

interface ScoreClassificationBadgeProps {
  classification: string;
  score?: number;
  className?: string;
}

// Color coding based on classification
export function ScoreClassificationBadge({
  classification,
  score,
  className,
}: ScoreClassificationBadgeProps) {
  const normalizedClass = classification.toLowerCase().replace(/[_\s]+/g, "_");

  const config: Record<string, { color: string; bg: string; emoji: string }> = {
    production_ready: {
      color: "text-emerald-300",
      bg: "bg-emerald-400/20",
      emoji: "🟢",
    },
    nearly_ready: {
      color: "text-emerald-400",
      bg: "bg-emerald-400/15",
      emoji: "🟢",
    },
    mostly_complete: {
      color: "text-amber-300",
      bg: "bg-amber-400/15",
      emoji: "🟡",
    },
    functional_incomplete: {
      color: "text-amber-400",
      bg: "bg-amber-400/10",
      emoji: "🟡",
    },
    foundation_laid: {
      color: "text-orange-400",
      bg: "bg-orange-400/10",
      emoji: "🟠",
    },
    early_stage: {
      color: "text-red-400",
      bg: "bg-red-400/10",
      emoji: "🔴",
    },
  };

  const { color, bg, emoji } = config[normalizedClass] || config.early_stage;

  const displayName = classification.replace(/_/g, " ");

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-sm font-medium capitalize",
        color,
        bg,
        className
      )}
      data-testid="score-classification-badge"
    >
      <span>{emoji}</span>
      <span>{displayName}</span>
      {score !== undefined && (
        <span className="ml-1 text-xs opacity-75">({score})</span>
      )}
    </span>
  );
}
