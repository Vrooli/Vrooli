/**
 * @libraryId react-component-library:HealthIndicator
 * @displayName HealthIndicator
 * @description A semantic health label with score, staleness, and non-color status treatment.
 * @version 1.0.3
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:HealthIndicator */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

export type HealthState = "healthy" | "degraded" | "blocked" | "unknown";
export const HealthIndicator = withClassName(function HealthIndicator({
  state = "unknown",
  score,
  stalenessDays = 0,
}: {
  state?: HealthState;
  score?: number;
  stalenessDays?: number;
}) {
  const text =
    state === "unknown"
      ? "Unknown"
      : `${state[0]?.toUpperCase() ?? ""}${state.slice(1)}`;
  return (
    <span
      data-testid="data-display.health-indicator"
      role="status"
      aria-label={`Health ${text}`}
      data-health={state}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: "var(--space-2xs)",
        padding: "var(--space-3xs) var(--space-2xs)",
        border: "1px solid currentColor",
        borderRadius: "var(--radius-pill)",
      }}
    >
      <span aria-hidden="true">●</span>
      {text}
      {score === undefined ? null : ` · ${score.toFixed(0)}%`}
      {stalenessDays > 0 ? ` · ${stalenessDays.toFixed(1)}d old` : null}
    </span>
  );
});
