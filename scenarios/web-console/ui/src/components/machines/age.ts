/**
 * Age wording for the machines surface.
 *
 * Staleness is stated, never implied. A machine that stopped answering must
 * say when, because presence has historically reported a week-old node as
 * fresh, and any surface that hides age inherits that defect.
 */

function plural(value: number, unit: string): string {
  return value === 1 ? `1 ${unit} ago` : `${String(value)} ${unit}s ago`;
}

/** Render a duration the way a person says it, changing unit with magnitude. */
export function humanAge(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "just now";
  if (seconds < 60) return plural(Math.floor(seconds), "second");
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return plural(minutes, "minute");
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return plural(hours, "hour");
  return plural(Math.floor(hours / 24), "day");
}

/** Render a remaining lifetime as a countdown an operator can act on. */
export function humanCountdown(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(total / 60);
  const remainder = total % 60;
  return `${String(minutes)}:${String(remainder).padStart(2, "0")}`;
}
