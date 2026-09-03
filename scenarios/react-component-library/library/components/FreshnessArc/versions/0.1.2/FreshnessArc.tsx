/**
 * @libraryId react-component-library:FreshnessArc
 * @displayName FreshnessArc
 * @description The board's own pulse: a hairline that drains over a source TTL and refills on fetch, dashed when the reading is cached.
 * @version 0.1.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:FreshnessArc */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useEffect, useState } from "react";

export interface FreshnessArcProps {
  observedAt: string | null;
  ttlSeconds: number;
  /** Cached readings draw the rule dashed and fully drained. */
  cached: boolean;
  className?: string;
}

const styles = `
  [data-rcl-freshness] { display: block; width: 100%; height: 2px; margin: 0.35em 0 0.5em; background: color-mix(in srgb, var(--color-primary, #2563eb) 18%, transparent); border-radius: 1px; overflow: hidden; }
  [data-rcl-freshness] > span { display: block; height: 100%; background: var(--color-primary, #2563eb); box-shadow: 0 0 8px var(--glow-primary, rgba(51,214,255,.5)); transform-origin: left; transition: transform 1s linear; }
  [data-rcl-freshness="cached"] { background: repeating-linear-gradient(90deg, var(--color-warning, #d97706) 0 4px, transparent 4px 9px); opacity: 0.7; }
  [data-rcl-freshness="cached"] > span { display: none; }
  @media (prefers-reduced-motion: reduce) { [data-rcl-freshness] > span { transition: none; } }
`;

/** The board's own pulse: a hairline that drains over the source TTL and refills on fetch. Replaces a STALE badge. */
export const FreshnessArc = withClassName(function FreshnessArc({
  observedAt,
  ttlSeconds,
  cached,
  className,
}: FreshnessArcProps) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  const age = observedAt ? (now - Date.parse(observedAt)) / 1000 : ttlSeconds;
  const remaining = cached ? 0 : Math.max(0, Math.min(1, 1 - age / Math.max(1, ttlSeconds)));
  return (
    <>
      <StyleSheet name="freshness-arc-1" css={styles} />
      <span
        data-rcl-freshness={cached ? "cached" : "live"}
        data-testid="freshness-hairline"
        className={className}
        aria-hidden="true"
      >
        <span style={{ transform: `scaleX(${remaining.toFixed(3)})` }} />
      </span>
    </>
  );
});
