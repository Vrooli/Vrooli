/**
 * Frecency algorithm for ordering items by usage frequency and recency.
 * Score = count * decay_factor, where decay halves every 24 hours.
 */

import type { ModeHistoryEntry } from "./types/templates";

const HALF_LIFE_HOURS = 24;

/**
 * Calculate frecency score for a history entry.
 * Higher scores = more frequently/recently used.
 */
export function calculateFrecencyScore(entry: ModeHistoryEntry): number {
  const now = Date.now();
  const lastUsedMs = new Date(entry.lastUsed).getTime();
  const hoursSinceUse = (now - lastUsedMs) / (1000 * 60 * 60);

  // Decay factor: halve score every HALF_LIFE_HOURS
  const decayFactor = Math.pow(0.5, hoursSinceUse / HALF_LIFE_HOURS);

  return entry.count * decayFactor;
}

/**
 * Sort items by frecency score (highest first).
 * Items not in history get score 0 and sort to the end.
 */
export function sortByFrecency<T extends { path: string }>(
  items: T[],
  history: ModeHistoryEntry[]
): T[] {
  const historyMap = new Map(history.map((h) => [h.path, h]));

  return [...items].sort((a, b) => {
    const historyA = historyMap.get(a.path);
    const historyB = historyMap.get(b.path);
    const scoreA = historyA ? calculateFrecencyScore(historyA) : 0;
    const scoreB = historyB ? calculateFrecencyScore(historyB) : 0;
    return scoreB - scoreA;
  });
}

/**
 * Record usage of a path, updating count and lastUsed.
 * Returns updated history array.
 */
export function recordPathUsage(
  history: ModeHistoryEntry[],
  path: string
): ModeHistoryEntry[] {
  const now = new Date().toISOString();
  const existing = history.find((h) => h.path === path);

  if (existing) {
    return history.map((h) =>
      h.path === path ? { ...h, count: h.count + 1, lastUsed: now } : h
    );
  }

  return [...history, { path, count: 1, lastUsed: now }];
}

/**
 * Serialize a mode path array to a string for storage.
 * ["Research", "Codebase Structure"] -> "Research/Codebase Structure"
 */
export function serializeModePath(modes: string[]): string {
  return modes.join("/");
}

/**
 * Deserialize a mode path string to an array.
 * "Research/Codebase Structure" -> ["Research", "Codebase Structure"]
 */
export function deserializeModePath(path: string): string[] {
  return path.split("/");
}
