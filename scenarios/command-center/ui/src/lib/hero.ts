import { hasValue, type Reading } from "./api";
import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";

/** The room picks its hero: the first measured reading in registry order, else the first illustrative one. */
export function pickHero(readings: Reading[], heroId?: string): Reading | null {
  if (heroId) return readings.find((reading) => reading.id === heroId) ?? null;
  return readings.find(hasValue) ?? readings.find((reading) => resolveReading(reading).figure === "sample") ?? readings[0] ?? null;
}

/**
 * The beat to show while illustrative figures are hidden: the next beat whose
 * hero is measured, wrapping once. Returns -1 when no beat has a measured hero,
 * so a room whose beats are all hidden stops instead of advancing forever.
 */
export function nextMeasuredBeat(beats: Array<{ hero?: string }>, measuredIDs: Set<string>, currentIndex: number): number {
  const measuredHero = (beat: { hero?: string }) => Boolean(beat.hero && measuredIDs.has(beat.hero));
  const after = beats.findIndex((beat, index) => index > currentIndex && measuredHero(beat));
  return after >= 0 ? after : beats.findIndex(measuredHero);
}
