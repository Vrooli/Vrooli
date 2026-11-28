export interface IncidentLogRange {
  seconds: number;
  since: string;
  until: string;
}

/** Adds the selected dashboard event's time neighborhood to a Logs deep link. */
export function incidentLogsSearch(search: string, eventAt: string | null, range: IncidentLogRange): string {
  const params = new URLSearchParams(search);
  const eventTime = eventAt ? Date.parse(eventAt) : Number.NaN;
  if (Number.isFinite(eventTime)) {
    const halfWindowMs = Math.max(1, range.seconds) * 500;
    params.set('since', new Date(eventTime - halfWindowMs).toISOString());
    params.set('until', new Date(eventTime + halfWindowMs).toISOString());
  } else {
    params.set('since', range.since);
    params.set('until', range.until);
  }
  const query = params.toString();
  return query ? `?${query}` : '';
}
