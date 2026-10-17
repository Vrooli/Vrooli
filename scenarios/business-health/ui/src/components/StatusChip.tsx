import type { ReactNode } from "react";

import { cn } from "../lib/utils";

export type ChipTone = "neutral" | "success" | "warning" | "danger" | "info";

const TONE_CLASSES: Record<ChipTone, string> = {
  neutral: "border-app-border bg-app-surface-muted text-app-muted-foreground",
  success: "border-app-success/40 bg-app-success/10 text-app-success",
  warning: "border-app-warning/40 bg-app-warning/10 text-app-warning",
  danger: "border-app-danger/40 bg-app-danger/10 text-app-danger",
  info: "border-app-info/40 bg-app-info/10 text-app-info",
};

export interface StatusChipProps {
  readonly tone?: ChipTone;
  /** Already-translated label (callers pass `t(strings.x.y)`). */
  readonly children: ReactNode;
  readonly className?: string;
  readonly title?: string;
  readonly "data-testid"?: string;
}

/**
 * Compact semantic status pill. Tone drives color, but the label text always
 * carries the meaning too — status is never communicated by color alone
 * (DESIGN.md accessibility floor). An optional leading dot gives a second,
 * position-based cue.
 */
export function StatusChip({
  tone = "neutral",
  children,
  className,
  title,
  "data-testid": testId,
}: StatusChipProps) {
  return (
    <span
      data-testid={testId}
      title={title}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-pill border px-2 py-0.5 text-xs font-medium",
        TONE_CLASSES[tone],
        className,
      )}
    >
      <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-current" />
      {children}
    </span>
  );
}
