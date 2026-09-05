import { FreshnessArc } from "./FreshnessArc";

const ground = {
  display: "grid",
  gap: "var(--space-md, 24px)",
  width: "min(100%, 24rem)",
  padding: "var(--space-lg, 32px)",
  background: "var(--color-background, #05070e)",
  color: "var(--color-foreground, #e8ecf3)",
  font: "var(--text-caption, 600 0.75rem/1.3 system-ui)",
};

/** A reading observed nine seconds ago against a thirty-second TTL: the hairline is two-thirds full and draining. */
export function Draining() {
  return (
    <div style={ground}>
      <span>observed 9s ago · ttl 30s</span>
      <FreshnessArc
        observedAt={new Date(Date.now() - 9_000).toISOString()}
        ttlSeconds={30}
        cached={false}
      />
    </div>
  );
}

/** A reading just fetched: the hairline is full. */
export function Fresh() {
  return (
    <div style={ground}>
      <span>observed now · ttl 60s</span>
      <FreshnessArc observedAt={new Date().toISOString()} ttlSeconds={60} cached={false} />
    </div>
  );
}

/** A cached reading: the rule is dashed and drained; the source is not answering. */
export function Cached() {
  return (
    <div style={ground}>
      <span>last good 4m ago · retrying</span>
      <FreshnessArc
        observedAt={new Date(Date.now() - 240_000).toISOString()}
        ttlSeconds={30}
        cached
      />
    </div>
  );
}
