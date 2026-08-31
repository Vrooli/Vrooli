/**
 * @libraryId react-component-library:StatCard
 * @displayName StatCard
 * @description The metric surface combining a stat with optional visualization, actions, status, explanation, and responsive layout that keeps the number legible at every width.
 * @version 1.0.9
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:StatCard */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 0.5rem)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export const StatCard = withClassName(function StatCard({
  label,
  value = "—",
  trend,
}: {
  label?: string;
  value?: string;
  trend?: string;
}) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.stat-card.metric", "Metric");
  return (
    <article
      data-testid="data-display.stat-card"
      style={{
        ...panel,
        position: "relative",
        display: "grid",
        gap: 8,
        minHeight: 118,
        boxSizing: "border-box",
      }}
    >
      <span
        aria-hidden
        style={{
          position: "absolute",
          top: 20,
          right: 20,
          width: 10,
          height: 10,
          borderRadius: "50%",
          background: "var(--color-primary, #2563eb)",
        }}
      />
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
      <strong data-stat-value style={{ fontSize: 28 }}>
        {value}
      </strong>
      {trend && (
        <span
          style={{
            color: "var(--color-primary, #2563eb)",
            fontSize: 13,
            fontWeight: 700,
          }}
        >
          {trend}
        </span>
      )}
    </article>
  );
});
