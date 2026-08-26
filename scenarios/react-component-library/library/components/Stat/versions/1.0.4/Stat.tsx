/**
 * @libraryId react-component-library:Stat
 * @displayName Stat
 * @description
 * @version 1.0.4
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:Stat */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { ReactNode } from "react";

export type StatTrendTone = "positive" | "negative" | "neutral";

const styles = `
  [data-rcl-stat] { min-inline-size: 0; display: grid; gap: var(--space-md, 1rem); padding: var(--space-lg, 1.5rem); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, .75rem); background: linear-gradient(145deg, var(--color-surface-raised, #fff), color-mix(in srgb, var(--color-primary, #2563eb) 4%, var(--color-surface-raised, #fff))); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08)); }
  [data-rcl-stat-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm, .75rem); }
  [data-rcl-stat-label] { color: var(--color-muted-foreground, #64748b); font: var(--text-label, 600 .8125rem/1rem system-ui, sans-serif); letter-spacing: .04em; }
  [data-rcl-stat-mark] { display: grid; place-items: center; inline-size: 2rem; block-size: 2rem; border: 1px solid color-mix(in srgb, var(--color-primary, #2563eb) 24%, var(--color-border, #cbd5e1)); border-radius: var(--radius-control, .5rem); background: color-mix(in srgb, var(--color-primary, #2563eb) 10%, transparent); color: var(--color-primary, #2563eb); font: 700 .875rem/1 system-ui, sans-serif; }
  [data-rcl-stat-value] { display: block; color: var(--color-foreground, #0f172a); font: var(--text-display, 750 clamp(1.75rem, 5vw, 2.5rem)/1 system-ui, sans-serif); letter-spacing: -.035em; }
  [data-rcl-stat-footer] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs, .75rem); min-block-size: 1.25rem; }
  [data-rcl-stat-trend] { font: var(--text-label, 600 .8125rem/1rem system-ui, sans-serif); }
  [data-rcl-stat-trend][data-tone="positive"] { color: var(--color-success, #15803d); }
  [data-rcl-stat-trend][data-tone="negative"] { color: var(--color-danger, #dc2626); }
  [data-rcl-stat-trend][data-tone="neutral"] { color: var(--color-muted-foreground, #64748b); }
  [data-rcl-stat-caption] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 500 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-stat-skeleton] { display: grid; gap: var(--space-2xs, .5rem); }
  [data-rcl-stat-skeleton] span { display: block; block-size: .8rem; border-radius: var(--radius-pill, 999px); background: var(--color-surface-muted, #f1f5f9); }
  [data-rcl-stat-skeleton] span:first-child { inline-size: min(70%, 8rem); block-size: 2.25rem; }
`;

export interface StatProps {
  label?: ReactNode;
  value?: ReactNode;
  trend?: ReactNode;
  trendTone?: StatTrendTone;
  caption?: ReactNode;
  icon?: ReactNode;
  loading?: boolean;
  className?: string;
  style?: React.CSSProperties;
  "aria-label"?: string;
}

export const Stat = withClassName(function Stat({
  label,
  value = "—",
  trend,
  trendTone = "neutral",
  caption,
  icon = "↗",
  loading = false,
  className,
  style,
  "aria-label": ariaLabel,
}: StatProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.stat.metric", "Metric");
  const strings = useStrings();
  return (
    <>
      <style data-rcl-stat-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <article
        data-testid="data-display.stat"
        className={className}
        style={style}
        data-rcl-stat
        aria-label={ariaLabel}
        aria-busy={loading || undefined}
      >
        {loading ? (
          <div
            data-rcl-stat-skeleton
            aria-label={strings("data-display.stat.loading-metric", "Loading metric")}
            role="status"
          >
            <span />
            <span />
          </div>
        ) : (
          <>
            <div data-rcl-stat-header>
              <span data-rcl-stat-label>{label}</span>
              <span data-rcl-stat-mark aria-hidden="true">
                {icon}
              </span>
            </div>
            <strong data-rcl-stat-value>{value}</strong>
            {(trend || caption) && (
              <div data-rcl-stat-footer>
                {trend && (
                  <span data-rcl-stat-trend data-tone={trendTone}>
                    {trend}
                  </span>
                )}
                {caption && <span data-rcl-stat-caption>{caption}</span>}
              </div>
            )}
          </>
        )}
      </article>
    </>
  );
});
