import { hasValue, type Reading } from "./api";
import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.1";

/** The room picks its hero: the first measured reading in registry order, else the first illustrative one. */
export function pickHero(readings: Reading[]): Reading | null {
  return readings.find(hasValue) ?? readings.find((reading) => resolveReading(reading).figure === "sample") ?? readings[0] ?? null;
}
