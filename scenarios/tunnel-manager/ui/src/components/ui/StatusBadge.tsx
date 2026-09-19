import { cn } from "../../lib/utils";

/** Semantic tone → token-driven badge styling. */
export type BadgeTone = "success" | "danger" | "warning" | "info" | "neutral";

interface StatusBadgeProps {
  tone: BadgeTone;
  children: React.ReactNode;
  className?: string;
  "data-testid"?: string;
}

const TONE_CLASS: Record<BadgeTone, string> = {
  success: "bg-app-success/15 text-app-success border-app-success/30",
  danger: "bg-app-danger/15 text-app-danger border-app-danger/30",
  warning: "bg-app-warning/15 text-app-warning border-app-warning/30",
  info: "bg-app-info/15 text-app-info border-app-info/30",
  neutral: "bg-app-surface-muted text-app-muted-foreground border-app-border",
};

/**
 * StatusBadge is the shared pill used wherever a surface renders a categorical
 * health/status value (tier, probe outcome, audit finding, recovery state).
 * Tone maps to the design-token palette so the same severity reads identically
 * everywhere; the visible text always carries the meaning (color is secondary,
 * for a11y).
 */
export function StatusBadge({ tone, children, className, ...rest }: StatusBadgeProps) {
  return (
    <span
      data-testid={rest["data-testid"]}
      className={cn(
        "inline-flex items-center rounded-pill border px-2 py-0.5 text-xs font-medium",
        TONE_CLASS[tone],
        className,
      )}
    >
      {children}
    </span>
  );
}
