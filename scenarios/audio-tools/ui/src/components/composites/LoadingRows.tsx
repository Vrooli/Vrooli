interface Props {
  rows?: number;
  /** Optional accessible label announced via aria-live="polite". */
  label?: string;
}

/**
 * Skeleton block for tabular loading states. Pure CSS pulse — avoids the
 * "spinner over empty area" pattern that scores poorly on perceived latency.
 */
export function LoadingRows({ rows = 4, label = "Loading…" }: Props) {
  return (
    <div role="status" aria-live="polite" aria-label={label} className="flex flex-col gap-2">
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          className="h-9 animate-pulse rounded-control bg-app-surface-muted"
          aria-hidden="true"
        />
      ))}
    </div>
  );
}
