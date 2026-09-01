import { useEffect, useState } from "react";

interface FreshnessHairlineProps {
  observedAt: string | null;
  ttlSeconds: number;
  /** Cached readings draw the rule dashed and fully drained. */
  cached: boolean;
}

/** The board's own pulse: a hairline that drains over the source TTL and refills on fetch. */
export function FreshnessHairline({ observedAt, ttlSeconds, cached }: FreshnessHairlineProps) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  const age = observedAt ? (now - Date.parse(observedAt)) / 1000 : ttlSeconds;
  const remaining = cached ? 0 : Math.max(0, Math.min(1, 1 - age / Math.max(1, ttlSeconds)));
  return (
    <span className={cached ? "cc-hairline cc-hairline-cached" : "cc-hairline"} data-testid="freshness-hairline" aria-hidden="true">
      <span style={{ transform: `scaleX(${remaining.toFixed(3)})` }} />
    </span>
  );
}
