import type { ReactNode } from "react";
import { Badge, type BadgeProps } from "../primitives/Badge";
import { cn } from "../../lib/utils";

export type VerdictKind = "pass" | "stale" | "unexpected" | "failure" | "neutral";

const kindToVariant: Record<VerdictKind, BadgeProps["variant"]> = {
  pass: "verdict-pass",
  stale: "verdict-stale",
  unexpected: "verdict-unexpected",
  failure: "verdict-failure",
  neutral: "neutral",
};

const kindToLabel: Record<VerdictKind, string> = {
  pass: "pass",
  stale: "stale",
  unexpected: "unexpected",
  failure: "failure",
  neutral: "—",
};

export interface VerdictCellProps {
  kind: VerdictKind;
  /** Optional metric tail (e.g. duration, cost). */
  metric?: ReactNode;
  /** Click handler — when present the cell becomes a button. */
  onClick?: () => void;
  /** Optional override for the badge label. */
  label?: ReactNode;
  /** Custom test id (e.g. row+column-keyed). */
  testId?: string;
  className?: string;
}

/**
 * Single verdict-grid cell. Renders a colored badge + optional metric tail.
 * Becomes interactive when `onClick` is provided.
 */
export function VerdictCell({
  kind,
  metric,
  onClick,
  label,
  testId,
  className,
}: VerdictCellProps) {
  const inner = (
    <>
      <Badge variant={kindToVariant[kind]}>{label ?? kindToLabel[kind]}</Badge>
      {metric ? (
        <span className="text-xs text-app-muted-foreground tabular-nums">{metric}</span>
      ) : null}
    </>
  );

  const baseClass = "flex items-center justify-between gap-2 rounded-control px-2 py-1.5";

  if (onClick) {
    return (
      <button
        type="button"
        data-testid={testId}
        data-verdict={kind}
        onClick={onClick}
        className={cn(
          baseClass,
          "w-full text-left transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-accent/60",
          className,
        )}
      >
        {inner}
      </button>
    );
  }

  return (
    <div
      data-testid={testId}
      data-verdict={kind}
      className={cn(baseClass, className)}
    >
      {inner}
    </div>
  );
}
