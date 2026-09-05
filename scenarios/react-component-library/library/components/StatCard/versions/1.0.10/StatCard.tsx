/**
 * @libraryId react-component-library:StatCard
 * @displayName StatCard
 * @description The metric surface combining a stat with optional visualization, actions, status, explanation, and responsive layout that keeps the number legible at every width.
 * @version 1.0.10
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:StatCard */
import type { HTMLAttributes, ReactNode } from "react";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import type { StatusTone } from "@vrooli/react-component-library/StatusBadge/1";

export interface StatCardProps extends HTMLAttributes<HTMLElement> {
  label?: string;
  value?: string;
  unit?: string;
  tone?: StatusTone;
  trend?: ReactNode;
}
export const statCardStyles = `
[data-rcl-stat-card] { position: relative; display: grid; min-block-size: 7.375rem; box-sizing: border-box; gap: var(--space-2xs); padding: var(--space-md); border: var(--border-hairline) solid var(--rcl-stat-border, var(--color-border)); border-radius: var(--radius-panel); background: var(--rcl-stat-surface, var(--color-surface)); color: var(--rcl-stat-value, var(--color-foreground)); box-shadow: var(--elev-raised); }
[data-rcl-stat-card][data-tone="success"] { --rcl-stat-value: var(--color-success); --rcl-stat-border: color-mix(in srgb, var(--color-success) 40%, var(--color-border)); --rcl-stat-surface: color-mix(in srgb, var(--color-success) 8%, var(--color-surface)); }
[data-rcl-stat-card][data-tone="warning"] { --rcl-stat-value: var(--color-warning); --rcl-stat-border: color-mix(in srgb, var(--color-warning) 40%, var(--color-border)); --rcl-stat-surface: color-mix(in srgb, var(--color-warning) 8%, var(--color-surface)); }
[data-rcl-stat-card][data-tone="danger"] { --rcl-stat-value: var(--color-danger); --rcl-stat-border: color-mix(in srgb, var(--color-danger) 40%, var(--color-border)); --rcl-stat-surface: color-mix(in srgb, var(--color-danger) 8%, var(--color-surface)); }
[data-rcl-stat-card][data-tone="info"] { --rcl-stat-value: var(--color-info); --rcl-stat-border: color-mix(in srgb, var(--color-info) 40%, var(--color-border)); --rcl-stat-surface: color-mix(in srgb, var(--color-info) 8%, var(--color-surface)); }
[data-rcl-stat-label] { color: var(--color-muted-foreground); font: var(--text-label); text-transform: uppercase; }
[data-rcl-stat-value] { min-inline-size: 4ch; text-wrap: balance; font-size: var(--font-size-lg); font-weight: 700; font-variant-numeric: tabular-nums; }
[data-rcl-stat-unit] { margin-inline-start: var(--space-3xs); color: var(--color-muted-foreground); font-size: var(--font-size-sm); font-weight: 500; }
[data-rcl-stat-trend] { color: var(--rcl-stat-value, var(--color-primary)); font-size: var(--font-size-sm); font-weight: 700; }
`;
export const StatCard = withClassName(function StatCard({
  label,
  value = "—",
  unit,
  tone = "neutral",
  trend,
  className,
  ...props
}: StatCardProps) {
  const strings = useStrings();
  useLibraryStyleSheet("stat-card", statCardStyles);
  return (
    <article
      {...props}
      className={className}
      data-rcl-stat-card
      data-tone={tone}
      data-testid="data-display.stat-card"
    >
      <span data-rcl-stat-label>{label ?? strings("data-display.stat-card.metric", "Metric")}</span>
      <strong data-rcl-stat-value>
        {value}
        {unit && <span data-rcl-stat-unit>{unit}</span>}
      </strong>
      {trend && <span data-rcl-stat-trend>{trend}</span>}
    </article>
  );
});
