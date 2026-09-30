import type { ReactNode } from "react";
import { StatusBarFill } from "@vrooli/react-component-library/ChromeTheme";

import { cn } from "../lib/classnames";

interface TopSafeAreaProps {
  children: ReactNode;
  className?: string;
  fillClassName?: string;
  testId?: string;
  enabled?: boolean;
}

/**
 * Owns the iOS/PWA top safe area without placing content inside it.
 *
 * The first child surface starts below the status-bar/notch region, while the
 * fill strip gives nav modes full control over the status area's color.
 *
 * The strip itself is the library's `StatusBarFill`, and its colour comes from
 * the `ChromeTheme` service rather than from a class threaded through props:
 * the resting terminal-derived tint is that service's base, and an active
 * banner contributes over it. That direction is deliberate — the colour arrives
 * from the one place that knows what is actually on screen, instead of being
 * recomputed by every caller from conditions that may already have been
 * dismissed.
 */
export default function TopSafeArea({
  children,
  className,
  fillClassName,
  testId,
  enabled = true,
}: TopSafeAreaProps) {
  return (
    <div
      data-testid={testId}
      className={cn("flex shrink-0 flex-col", className)}
    >
      {enabled && (
        <StatusBarFill
          testId={testId ? `${testId}-fill` : undefined}
          className={fillClassName}
        />
      )}
      {children}
    </div>
  );
}
