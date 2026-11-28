import type { BoardRoom } from "./api";

type Beat = NonNullable<BoardRoom["beats"]>[number];

/**
 * Resolve the readings owned by one beat while preserving registry order.
 * Legacy beats without readingIds intentionally retain whole-room behavior;
 * authored beats opt into bounded presentation by declaring their group.
 */
export function readingsForBeat<T extends { id: string }>(beat: Beat | undefined, readings: T[]): T[] {
  if (!beat || beat.readingIds === undefined) return readings;
  const allowed = new Set([beat.hero, ...beat.readingIds]);
  return readings.filter((reading) => allowed.has(reading.id));
}
