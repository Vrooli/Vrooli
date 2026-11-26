import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Remove a group or folder prefix so a row can show the part of the path that
 * distinguishes it from its siblings.
 */
export function stripPathPrefix(path: string, prefix?: string) {
  const normalized = (prefix ?? "").replace(/^\/+|\/+$/g, "");
  if (!normalized) return path;
  return path.startsWith(`${normalized}/`) ? path.slice(normalized.length + 1) : path;
}

/** Compact a count for dense list chrome: 999, 1.2K, 3.4M. */
export function compactCount(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 10_000) return `${Math.round(value / 1_000)}K`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
}

/**
 * Truncate a file path intelligently, keeping the most informative parts visible.
 * Prioritises showing the first segment and the last 1-2 segments, collapsing
 * the middle with "...".
 */
export function formatPath(path: string, maxChars: number) {  if (path.length <= maxChars) return path;

  const segments = path.split("/");
  const last = segments[segments.length - 1] || path;
  const lastTwo = segments.length > 1 ? segments.slice(-2).join("/") : last;
  const first = segments[0] || path;

  const candidateMiddle = `${first}/.../${lastTwo}`;
  if (candidateMiddle.length <= maxChars) return candidateMiddle;

  const candidateMiddleShort = `${first}/.../${last}`;
  if (candidateMiddleShort.length <= maxChars) return candidateMiddleShort;

  const candidateEnd = `.../${lastTwo}`;
  if (candidateEnd.length <= maxChars) return candidateEnd;

  const candidateEndShort = `.../${last}`;
  if (candidateEndShort.length <= maxChars) return candidateEndShort;

  const tailMax = Math.max(1, maxChars - 4);
  return `.../${last.slice(-tailMax)}`;
}
