interface StaleBadgeProps {
  ts: string | null | undefined;
}

/**
 * STALE indicator surfaced when an upstream source is cached but last
 * fetch attempt failed. Renders nothing when no staleness timestamp is set.
 */
export function StaleBadge({ ts }: StaleBadgeProps) {
  if (ts === null || ts === undefined) {
    return null;
  }
  return (
    <span
      className="cc-badge cc-badge-stale"
      data-testid="stale-badge"
      title={`Last fetch failed at ${ts}`}
    >
      STALE
    </span>
  );
}
