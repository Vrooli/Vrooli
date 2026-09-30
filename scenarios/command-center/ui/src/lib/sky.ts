import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import type { Constellation, Reading } from "./api";

/** Where one signal stands, decided by the one ink resolver: measured, or the reason it is not. */
export type SkyState = "measured" | "failing" | "in-reach" | "missing";

export function skyState(reading: Reading): SkyState {
  const resolution = resolveReading(reading);
  if (resolution.figure === "measured" && !resolution.finding) return "measured";
  // A NOW sensor exists; no number, or an untrusted one, means it is failing.
  if (reading.coverage === "NOW") return "failing";
  if (reading.coverage === "IN-REACH") return "in-reach";
  return "missing";
}

export interface SkyTally {
  rooms: number;
  total: number;
  measured: number;
  /** Measured signals served from cache; a subset of measured. */
  cached: number;
  failing: number;
  inReach: number;
  missing: number;
}

/**
 * Counts each signal once, however many rooms show it. The result is counts,
 * never a ratio: Director Swarm is a production ledger (DECISIONS 2026-09-01).
 */
export function tallySky(constellations: Constellation[]): SkyTally {
  const unique = new Map<string, Reading>();
  for (const constellation of constellations) {
    for (const reading of constellation.readings) {
      if (!unique.has(reading.id)) unique.set(reading.id, reading);
    }
  }
  const tally: SkyTally = { rooms: constellations.length, total: unique.size, measured: 0, cached: 0, failing: 0, inReach: 0, missing: 0 };
  for (const reading of unique.values()) {
    const state = skyState(reading);
    if (state === "measured") {
      tally.measured += 1;
      if (reading.trust === "CACHED") tally.cached += 1;
    } else if (state === "failing") tally.failing += 1;
    else if (state === "in-reach") tally.inReach += 1;
    else tally.missing += 1;
  }
  return tally;
}
