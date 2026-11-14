/**
 * Pure sort/filter model tests — the worst-first ranking and AND-composed
 * filters, exercised without React or the network.
 */
import { describe, expect, it } from "vitest";

import {
  DEFAULT_SORT,
  EMPTY_FILTERS,
  defaultDirectionFor,
  filterEntries,
  sortEntries,
  toggleSort,
} from "./fleetModel";
import { makeFleetEntry } from "./mocks/factories";

const alpha = makeFleetEntry({ scenario: "alpha", debtScore: 50, errorCount: 5, starterRegistry: true });
const beta = makeFleetEntry({ scenario: "beta", debtScore: 30, errorCount: 2, templateLaggard: true });
const gamma = makeFleetEntry({ scenario: "gamma", debtScore: 10, errorCount: 0, unprovenClaims: 3 });
const entries = [beta, gamma, alpha];

describe("fleetModel.sortEntries", () => {
  it("defaults to worst-first by debt score descending", () => {
    const sorted = sortEntries(entries, DEFAULT_SORT);
    expect(sorted.map((e) => e.scenario)).toEqual(["alpha", "beta", "gamma"]);
  });

  it("sorts the scenario column ascending by slug", () => {
    const sorted = sortEntries(entries, { column: "scenario", direction: "asc" });
    expect(sorted.map((e) => e.scenario)).toEqual(["alpha", "beta", "gamma"]);
  });

  it("does not mutate the source array", () => {
    const source = [...entries];
    sortEntries(source, DEFAULT_SORT);
    expect(source.map((e) => e.scenario)).toEqual(["beta", "gamma", "alpha"]);
  });
});

describe("fleetModel.toggleSort", () => {
  it("flips direction when the same column is reselected", () => {
    expect(toggleSort({ column: "debt", direction: "desc" }, "debt")).toEqual({
      column: "debt",
      direction: "asc",
    });
  });

  it("switches columns at the natural default direction", () => {
    expect(toggleSort({ column: "debt", direction: "desc" }, "scenario")).toEqual({
      column: "scenario",
      direction: "asc",
    });
    expect(toggleSort({ column: "scenario", direction: "asc" }, "errors")).toEqual({
      column: "errors",
      direction: "desc",
    });
  });

  it("gives alphabetic columns ascending defaults, numeric descending", () => {
    expect(defaultDirectionFor("scenario")).toBe("asc");
    expect(defaultDirectionFor("template")).toBe("asc");
    expect(defaultDirectionFor("debt")).toBe("desc");
    expect(defaultDirectionFor("errors")).toBe("desc");
  });
});

describe("fleetModel.filterEntries", () => {
  it("returns every entry with no active filters", () => {
    expect(filterEntries(entries, EMPTY_FILTERS)).toHaveLength(3);
  });

  it("filters by scenario substring, case-insensitively", () => {
    const result = filterEntries(entries, { ...EMPTY_FILTERS, text: "ALP" });
    expect(result.map((e) => e.scenario)).toEqual(["alpha"]);
  });

  it("keeps only starter registries when that toggle is on", () => {
    const result = filterEntries(entries, { ...EMPTY_FILTERS, starter: true });
    expect(result.map((e) => e.scenario)).toEqual(["alpha"]);
  });

  it("keeps only template laggards when that toggle is on", () => {
    const result = filterEntries(entries, { ...EMPTY_FILTERS, laggard: true });
    expect(result.map((e) => e.scenario)).toEqual(["beta"]);
  });

  it("keeps only entries with unproven claims when that toggle is on", () => {
    const result = filterEntries(entries, { ...EMPTY_FILTERS, unproven: true });
    expect(result.map((e) => e.scenario)).toEqual(["gamma"]);
  });

  it("composes multiple filters with AND semantics", () => {
    const result = filterEntries(entries, { ...EMPTY_FILTERS, starter: true, laggard: true });
    expect(result).toHaveLength(0);
  });
});
