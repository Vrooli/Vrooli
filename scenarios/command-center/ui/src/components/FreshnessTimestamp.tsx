type FreshnessTimestampProps = {
  label: string;
  value?: string | null;
  loading?: boolean;
};

/** Show when owner-provided data was observed, without treating a missing time as fresh. */
export function FreshnessTimestamp({ label, value, loading = false }: FreshnessTimestampProps) {
  const observedAt = value ? new Date(value) : null;
  const valid = observedAt && !Number.isNaN(observedAt.getTime());

  return (
    <p className="cc-panel-note cc-freshness-timestamp" data-testid="freshness-timestamp">
      {label}: {loading ? "checking" : valid ? <time dateTime={observedAt.toISOString()}>{observedAt.toLocaleString()}</time> : "unknown"}
    </p>
  );
}
