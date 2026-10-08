/**
 * Format the duration between two ISO timestamps as a short human label.
 * Returns "—" when either timestamp is missing or invalid, or when the
 * computed duration is negative (clock skew).
 */
export function formatBootDuration(firstEntryISO: string, lastEntryISO: string): string {
  if (!firstEntryISO || !lastEntryISO) return '—';
  const start = Date.parse(firstEntryISO);
  const end = Date.parse(lastEntryISO);
  if (Number.isNaN(start) || Number.isNaN(end)) return '—';
  const ms = end - start;
  if (ms < 0) return '—';

  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ${minutes % 60}m`;
  const days = Math.floor(hours / 24);
  return `${days}d ${hours % 24}h`;
}
