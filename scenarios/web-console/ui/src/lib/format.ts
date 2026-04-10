// DOC: docs/internal/SEAMS.md#6-cross-cutting
/** Extract the shell name from a full path (e.g., "/bin/bash" → "bash"). */
export function getShellName(shellPath: string): string {
  const name = shellPath.split("/").pop();
  return name || "shell";
}

/** Format an ISO timestamp as a locale time string (e.g., "3:42:15 PM"). */
export function formatSessionTime(isoString: string): string {
  const date = new Date(isoString);
  if (isNaN(date.getTime())) return "—";
  return date.toLocaleTimeString();
}

/** Truncate a UUID for display (e.g., "abc12345-..." → "abc12345..."). */
export function truncateId(id: string, length = 8): string {
  if (!id) return "—";
  return `${id.slice(0, length)}...`;
}

/**
 * Parse a short duration string (e.g. "1h", "30m", "15s") into milliseconds.
 * Returns 0 for unrecognized formats. Only supports single-unit durations.
 */
export function parseDurationMs(duration?: string): number {
  if (!duration) return 0;
  const match = duration.match(/^(\d+)(h|m|s)$/);
  if (!match) return 0;
  const val = parseInt(match[1] ?? "0", 10);
  const unit = match[2];
  if (unit === "h") return val * 3600000;
  if (unit === "m") return val * 60000;
  if (unit === "s") return val * 1000;
  return 0;
}

/**
 * Format a remaining-seconds value as a human-readable countdown.
 * Examples: "2h 15m", "4m 30s", "12s", "expired"
 */
export function formatCountdown(seconds: number): string {
  if (seconds <= 0) return "expired";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}
