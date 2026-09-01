import type { Reading } from "../lib/api";

/** One place to author a reading for tests; every field the payload carries, overridable per case. */
export const makeReading = (overrides: Partial<Reading> = {}): Reading => ({
  id: "metric",
  label: "Metric",
  unit: "count",
  format: "integer",
  source: { team: "director-swarm", binding: "scenario:swarm-manager" },
  coverage: "NOW",
  trust: "VALID",
  empirical: "NONE",
  value: null,
  observedAt: null,
  ttlSeconds: 60,
  target: null,
  owner: null,
  whatIsNeeded: null,
  firstObservedMissing: null,
  gapOpenDays: null,
  sample: null,
  prediction: null,
  ...overrides,
});

export const authoredSample = (value: number, series: number[] = [value]) => ({ value, series, basis: "hand-authored, reviewed 2026-09-01" });
