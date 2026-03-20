import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Truncate a file path intelligently, keeping the most informative parts visible.
 * Prioritises showing the first segment and the last 1-2 segments, collapsing
 * the middle with "...".
 */
export function formatPath(path: string, maxChars: number) {
  if (path.length <= maxChars) return path;

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
