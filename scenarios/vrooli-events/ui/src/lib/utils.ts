// DOC: docs/internal/COHERENCE-NOTES.md#styling-patterns
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Format a byte count to a human-readable string (e.g. "1.5 MB"). */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

/** Truncate a string to a max length, appending ellipsis if truncated. */
export function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max) + "\u2026" : s;
}

/** Format an ISO timestamp to a locale-appropriate string. Returns "Invalid date" on bad input. */
export function formatTimestamp(iso: string): string {
  try {
    const d = new Date(iso);
    if (isNaN(d.getTime())) return "Invalid date";
    return d.toLocaleString();
  } catch {
    return "Invalid date";
  }
}

/** Safely JSON-stringify a value, returning a fallback on circular refs or errors. */
export function safeStringify(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return "[Unable to serialize value]";
  }
}
