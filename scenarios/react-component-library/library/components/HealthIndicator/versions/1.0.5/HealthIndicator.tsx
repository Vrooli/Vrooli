/**
 * @libraryId react-component-library:HealthIndicator
 * @displayName HealthIndicator
 * @description A semantic health label with score, staleness, and non-color status treatment.
 * @version 1.0.5
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:HealthIndicator */
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import type { HTMLAttributes } from "react";

export type HealthState = "healthy" | "degraded" | "blocked" | "unknown";
const tones: Record<HealthState, StatusTone> = {
  healthy: "success",
  degraded: "warning",
  blocked: "danger",
  unknown: "neutral",
};
export interface HealthIndicatorProps extends HTMLAttributes<HTMLSpanElement> {
  state?: HealthState;
  score?: number;
  stalenessDays?: number;
}
export const HealthIndicator = withClassName(function HealthIndicator({
  state = "unknown",
  score,
  stalenessDays = 0,
  ...props
}: HealthIndicatorProps) {
  const text =
    state === "unknown" ? "Unknown" : `${state[0]?.toUpperCase() ?? ""}${state.slice(1)}`;
  const details = `${text}${score === undefined ? "" : ` · ${score.toFixed(0)}%`}${stalenessDays > 0 ? ` · ${stalenessDays.toFixed(1)}d old` : ""}`;
  return (
    <StatusBadge
      {...props}
      data-testid="data-display.health-indicator"
      role="status"
      aria-label={`Health ${details}`}
      data-health={state}
      tone={tones[state]}
    >
      {details}
    </StatusBadge>
  );
});
