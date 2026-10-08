/** @vrooliComponentSource react-component-library:MetricBreakdown */
export interface MetricItem {
  id: string;
  label: string;
  value: number;
  total?: number;
  detail?: string;
}
export function MetricBreakdown({ items = [] }: { items?: MetricItem[] }) {
  return (
    <dl
      aria-label="Metric breakdown"
      style={{ display: "grid", gap: "var(--space-xs)" }}
    >
      {items.map((item) => (
        <div
          key={item.id}
          style={{
            display: "flex",
            justifyContent: "space-between",
            gap: "var(--space-xs)",
            borderBottom: "1px solid var(--color-border)",
            paddingBlock: "var(--space-2xs)",
          }}
        >
          <dt>
            {item.label}
            {item.detail ? <small> · {item.detail}</small> : null}
          </dt>
          <dd>
            {item.value}
            {item.total === undefined ? "" : ` / ${item.total}`}
          </dd>
        </div>
      ))}
    </dl>
  );
}
