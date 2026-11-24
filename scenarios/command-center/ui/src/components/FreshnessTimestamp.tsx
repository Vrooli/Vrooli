type FreshnessTimestampProps = {
  label: string;
  value?: string | null;
  loading?: boolean;
};

const explicitTimezone = /(?:Z|[+-]\d{2}:\d{2})$/i;
const utcFormatter = new Intl.DateTimeFormat("en-US", {
  year: "numeric",
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
  timeZone: "UTC",
  timeZoneName: "short",
});

function parseObservedAt(value?: string | null): Date | null {
  if (!value || !explicitTimezone.test(value)) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

/** Show when owner-provided data was observed, without treating a missing time as fresh. */
export function FreshnessTimestamp({ label, value, loading = false }: FreshnessTimestampProps) {
  const observedAt = parseObservedAt(value);

  return (
    <p className="cc-panel-note cc-freshness-timestamp" data-testid="freshness-timestamp">
      {label}: {loading ? "checking" : observedAt ? <time dateTime={observedAt.toISOString()}>{utcFormatter.format(observedAt)}</time> : "unknown"}
    </p>
  );
}
