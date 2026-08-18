/** @vrooliComponentSource react-component-library:ScoreGauge */
export interface ScoreGaugeProps {
  value?: number;
  label?: string;
  threshold?: number;
}
export function ScoreGauge({
  value = 0,
  label = "Score",
  threshold = 90,
}: ScoreGaugeProps) {
  const bounded = Math.max(0, Math.min(100, value));
  const status =
    bounded >= threshold
      ? "passing"
      : bounded >= threshold / 2
        ? "watch"
        : "blocked";
  return (
    <section
      aria-label={label}
      data-status={status}
      style={{
        display: "grid",
        gap: "var(--space-2xs)",
        padding: "var(--space-sm)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface)",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          gap: "var(--space-xs)",
        }}
      >
        <span>{label}</span>
        <strong aria-label={`${bounded.toFixed(0)} percent`}>
          {bounded.toFixed(0)}%
        </strong>
      </div>
      <meter
        min={0}
        max={100}
        value={bounded}
        aria-label={`${label} progress`}
      />
      <small>
        {status} · threshold {threshold}%
      </small>
    </section>
  );
}
