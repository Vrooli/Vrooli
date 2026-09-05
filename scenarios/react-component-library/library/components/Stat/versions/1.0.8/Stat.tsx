/**
 * @libraryId react-component-library:Stat
 * @displayName Stat
 * @description The focused metric with value, label, comparison, trend, confidence, loading treatment, locale-aware formatting, and accessible context for what the number means.
 * @version 1.0.8
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:Stat */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";

export type StatTrendTone = "positive" | "negative" | "neutral";
export type StatProvenance = "measured" | "cached" | "sample" | "absent";

const styles = `
  [data-rcl-stat] { min-inline-size: 0; display: grid; gap: var(--space-md, 24px); padding: var(--space-lg, 32px); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); background: linear-gradient(145deg, var(--color-surface-raised, #ffffff), color-mix(in srgb, var(--color-primary, #2563eb) 4%, var(--color-surface-raised, #ffffff))); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10)); }
  [data-rcl-stat][data-scale="hero"] { gap: var(--space-lg, 32px); padding: var(--space-xl, 48px); }
  [data-rcl-stat][data-scale="wall"] { gap: var(--space-xl, 48px); padding: var(--space-2xl, 64px); border-width: var(--border-hairline, 1px); background: transparent; box-shadow: none; }
  [data-rcl-stat-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm, 16px); }
  [data-rcl-stat-label] { color: var(--color-muted-foreground, #64748b); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); letter-spacing: .04em; }
  [data-rcl-stat-mark] { display: grid; place-items: center; inline-size: 2rem; block-size: 2rem; border: 1px solid color-mix(in srgb, var(--color-primary, #2563eb) 24%, var(--color-border, #cbd5e1)); border-radius: var(--radius-control, 0.375rem); background: color-mix(in srgb, var(--color-primary, #2563eb) 10%, transparent); color: var(--color-primary, #2563eb); font: 700 .875rem/1 system-ui, sans-serif; }
  [data-rcl-stat-value] { display: block; color: var(--color-foreground, #0f172a); font: var(--text-display, 700 var(--text-display-size) / var(--text-display-line) var(--font-sans)); letter-spacing: -.035em; }
  [data-rcl-stat][data-scale="hero"] [data-rcl-stat-value] { font-size: clamp(2.5rem, 8vw, 6rem); }
  [data-rcl-stat][data-scale="wall"] [data-rcl-stat-value] { font-size: var(--text-wall, clamp(5rem, 16vw, 20rem)); }
  [data-rcl-stat][data-provenance="sample"] [data-rcl-stat-value] { outline: 2px solid currentColor; outline-offset: .15em; }
  [data-rcl-stat][data-provenance="absent"] [data-rcl-stat-value] { outline: 2px dotted currentColor; outline-offset: .15em; opacity: .72; }
  [data-rcl-stat-footer] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs, 12px); min-block-size: 1.25rem; }
  [data-rcl-stat-trend] { font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); }
  [data-rcl-stat-trend][data-tone="positive"] { color: var(--color-success, #16a34a); }
  [data-rcl-stat-trend][data-tone="negative"] { color: var(--color-danger, #dc2626); }
  [data-rcl-stat-trend][data-tone="neutral"] { color: var(--color-muted-foreground, #64748b); }
  [data-rcl-stat-caption] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-stat-skeleton] { display: grid; gap: var(--space-2xs, 8px); }
  [data-rcl-stat-skeleton] span { display: block; block-size: .8rem; border-radius: var(--radius-pill, 9999px); background: var(--color-surface-muted, #f1f5f9); }
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
  provenance?: StatProvenance;
  scale?: "card" | "hero" | "wall";
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
  provenance = "measured",
  scale = "card",
}: StatProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.stat.metric", "Metric");
  const strings = useStrings();
  return (
    <>
      <StyleSheet name="stat-1-0-4-1" css={styles} />
      <article
        data-testid="data-display.stat"
        className={className}
        style={style}
        data-rcl-stat
        aria-label={ariaLabel}
        aria-busy={loading || undefined}
        data-provenance={provenance}
        data-scale={scale}
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
