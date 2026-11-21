/** @vrooliComponentSource react-component-library:HealthIndicator */
export type HealthState = "healthy" | "degraded" | "blocked" | "unknown";
export function HealthIndicator({
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
}
