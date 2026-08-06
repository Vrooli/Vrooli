/** @vrooliComponentSource react-component-library:DescriptionList */
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function DescriptionList({
  entries = [],
}: {
  entries?: Array<{ term: string; description: string }>;
}) {
  return (
    <dl
      style={{
        display: "grid",
        margin: 0,
        border: "1px solid var(--color-border, #cbd5e1)",
        borderRadius: "var(--radius-panel, .75rem)",
        overflow: "hidden",
      }}
    >
      {entries.map((entry, index) => (
        <div
          key={entry.term}
          style={{
            display: "grid",
            gridTemplateColumns: "minmax(120px, .35fr) 1fr",
            gap: 24,
            padding: 16,
            background:
              index % 2
                ? "var(--color-surface-muted, #f1f5f9)"
                : "var(--color-surface, #fff)",
          }}
        >
          <dt style={muted}>{entry.term}</dt>
          <dd style={{ margin: 0, fontWeight: 600 }}>{entry.description}</dd>
        </div>
      ))}
    </dl>
  );
}
