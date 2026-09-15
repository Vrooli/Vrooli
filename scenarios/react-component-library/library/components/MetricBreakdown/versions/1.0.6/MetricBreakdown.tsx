/**
 * @libraryId react-component-library:MetricBreakdown
 * @displayName MetricBreakdown
 * @version 1.0.6
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:MetricBreakdown */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

export interface MetricItem {
  id: string;
  label: string;
  value: number;
  total?: number;
  detail?: string;
}
export const MetricBreakdown = withClassName(function MetricBreakdown({
  items = [],
}: {
  items?: MetricItem[];
}) {
  const strings = useStrings();
  return (
    <dl
      data-testid="data-display.metric-breakdown"
      aria-label={strings("data-display.metric-breakdown.metric-breakdown", "Metric breakdown")}
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
});
