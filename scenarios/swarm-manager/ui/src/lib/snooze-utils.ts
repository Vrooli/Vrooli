/**
 * Snooze Utilities
 *
 * Pure functions for snooze key generation, preset computation, and filtering.
 * No store dependency — consumed by snooze-store and command-post-utils.
 */

import type { BacklogKind } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface SnoozeEntry {
  key: string;
  expiresAt: number; // epoch ms
}

export interface SnoozePreset {
  label: string;
  ms: number | null; // null = compute dynamically (e.g. "tomorrow")
  compute?: () => number;
}

// ---------------------------------------------------------------------------
// Presets
// ---------------------------------------------------------------------------

export const SNOOZE_PRESETS: SnoozePreset[] = [
  { label: "1 hour", ms: 3_600_000 },
  { label: "4 hours", ms: 14_400_000 },
  { label: "Tomorrow", ms: null, compute: tomorrowAt9am },
];

// ---------------------------------------------------------------------------
// Key builders
// ---------------------------------------------------------------------------

export function snoozeKeyForBacklog(kind: BacklogKind, name: string): string {
  return `backlog:${kind}/${name}`;
}

export function snoozeKeyForExecution(id: string): string {
  return `execution:${id}`;
}

export function snoozeKeyForCapture(id: string): string {
  return `capture:${id}`;
}

// ---------------------------------------------------------------------------
// Expiry helpers
// ---------------------------------------------------------------------------

/**
 * Compute the expiry epoch ms for a preset.
 * For fixed-duration presets, adds ms to Date.now().
 * For dynamic presets (e.g. "Tomorrow"), calls the compute function.
 */
export function getPresetExpiry(preset: SnoozePreset): number {
  if (preset.compute) return preset.compute();
  return Date.now() + (preset.ms ?? 0);
}

/** Check if a snooze entry has expired. */
export function isExpired(entry: SnoozeEntry): boolean {
  return Date.now() >= entry.expiresAt;
}

/**
 * Compute epoch ms for tomorrow at 9:00 AM local time.
 * If it's currently before 9 AM, "tomorrow" still means the next calendar day.
 */
export function tomorrowAt9am(): number {
  const now = new Date();
  const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 9, 0, 0, 0);
  return tomorrow.getTime();
}

// ---------------------------------------------------------------------------
// Filtering
// ---------------------------------------------------------------------------

/**
 * Filter out items whose snooze key is in the snoozed set.
 * Returns only non-snoozed items.
 */
export function filterSnoozed<T>(
  items: T[],
  keyFn: (item: T) => string,
  snoozedKeys: Set<string>,
): T[] {
  return items.filter((item) => !snoozedKeys.has(keyFn(item)));
}
