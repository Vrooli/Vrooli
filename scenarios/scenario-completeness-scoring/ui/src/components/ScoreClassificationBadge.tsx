import { cn } from "../lib/utils";

interface ScoreClassificationBadgeProps {
  classification: string;
  score?: number;
  className?: string;
  /** Show tooltip with classification description on hover */
  showTooltip?: boolean;
}

// Classification descriptions to help first-time users understand what each level means
const classificationDescriptions: Record<string, string> = {
  production_ready: "96-100 — Ready for production deployment with comprehensive coverage",
  nearly_ready: "81-95 — Almost complete, minor gaps to address before production",
  mostly_complete: "66-80 — Core functionality works, needs testing and polish",
  functional_incomplete: "51-65 — Basic functionality present, significant gaps remain",
  foundation_laid: "26-50 — Initial structure in place, major work still needed",
  early_stage: "0-25 — Early development, fundamental features not yet implemented",
};

// Color coding based on classification
export function ScoreClassificationBadge({
  classification,
  score,
  className,
  showTooltip = true,
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
  const description = showTooltip ? (classificationDescriptions[normalizedClass] || classificationDescriptions.early_stage) : undefined;

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
      title={description}
    >
      <span>{emoji}</span>
      <span>{displayName}</span>
      {score !== undefined && (
        <span className="ml-1 text-xs opacity-75">({score})</span>
      )}
    </span>
  );
}
