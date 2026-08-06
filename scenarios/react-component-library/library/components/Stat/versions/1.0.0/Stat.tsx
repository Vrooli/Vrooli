/** @vrooliComponentSource react-component-library:Stat */
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function Stat({
  label = "Metric",
  value = "—",
}: {
  label?: string;
  value?: string;
}) {
  return (
    <div style={{ display: "grid", gap: 4 }}>
      <span
        style={{
          ...muted,
          fontSize: 12,
          fontWeight: 700,
          textTransform: "uppercase",
        }}
      >
        {label}
      </span>
      <strong data-stat-value style={{ fontSize: 24 }}>
        {value}
      </strong>
    </div>
  );
}
