/**
 * Byte-format helpers for human-readable retention size input.
 *
 * `parseHumanBytes` accepts mixed-unit input (e.g. "10 GiB", "500MB",
 * "1024", "0") and returns a non-negative integer byte count, or null
 * for invalid input.
 *
 * `bytesToHumanReadable` is the inverse: it renders the canonical IEC
 * form (KiB, MiB, GiB, TiB) so a save → reload doesn't introduce
 * silent drift in the displayed value.
 */

const SIZE_UNITS: Record<string, number> = {
  "": 1,
  b: 1,
  kb: 1_000,
  k: 1_000,
  kib: 1_024,
  mb: 1_000_000,
  m: 1_000_000,
  mib: 1_048_576,
  gb: 1_000_000_000,
  g: 1_000_000_000,
  gib: 1_073_741_824,
  tb: 1_000_000_000_000,
  t: 1_000_000_000_000,
  tib: 1_099_511_627_776,
};

export function parseHumanBytes(input: string): number | null {
  const trimmed = input.trim();
  if (trimmed === "") return null;
  const match = trimmed.match(/^(-?\d+(?:\.\d+)?)\s*([A-Za-z]*)$/);
  if (!match) return null;
  const value = parseFloat(match[1]);
  if (!Number.isFinite(value) || value < 0) return null;
  const unit = match[2].toLowerCase();
  const multiplier = SIZE_UNITS[unit];
  if (multiplier === undefined) return null;
  return Math.round(value * multiplier);
}

export function bytesToHumanReadable(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"] as const;
  let value = bytes;
  let i = 0;
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024;
    i++;
  }
  return `${parseFloat(value.toFixed(2))} ${units[i]}`;
}

export function parseNonNegativeInt(input: string): number | null {
  const trimmed = input.trim();
  if (trimmed === "") return null;
  if (!/^\d+$/.test(trimmed)) return null;
  const n = Number(trimmed);
  return Number.isInteger(n) && n >= 0 ? n : null;
}
