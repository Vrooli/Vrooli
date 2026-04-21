import { describe, expect, it } from "vitest";
import { normalizeAISearchResponse } from "./ai-search";

// The server should always send `results: []` for no matches, but we harden
// the client side of the seam so a single server regression (or a buggy proxy
// that drops the field) can't crash the page.
describe("normalizeAISearchResponse", () => {
  it("coerces null results to an empty array", () => {
    const normalized = normalizeAISearchResponse(
      { results: null, total: 0, query: "x", entity: "both", fallback: "none", latencyMs: 12 },
      "both",
    );
    expect(normalized.results).toEqual([]);
    expect(normalized.total).toBe(0);
  });

  it("coerces missing results field to an empty array", () => {
    const normalized = normalizeAISearchResponse(
      { total: 0, query: "x", entity: "both", fallback: "none", latencyMs: 0 },
      "both",
    );
    expect(normalized.results).toEqual([]);
  });

  it("coerces a non-array results value to an empty array", () => {
    const normalized = normalizeAISearchResponse(
      { results: "oops" as unknown, total: 0, query: "x", entity: "both", fallback: "none", latencyMs: 0 },
      "both",
    );
    expect(normalized.results).toEqual([]);
  });

  it("preserves a well-formed response", () => {
    const item = {
      entity: "backlog" as const,
      id: "a",
      score: 0.8,
      scorePercent: 80,
      payload: { title: "A" },
    };
    const normalized = normalizeAISearchResponse(
      { results: [item], total: 1, query: "x", entity: "backlog", fallback: "none", latencyMs: 42 },
      "backlog",
    );
    expect(normalized.results).toEqual([item]);
    expect(normalized.total).toBe(1);
    expect(normalized.entity).toBe("backlog");
    expect(normalized.fallback).toBe("none");
    expect(normalized.latencyMs).toBe(42);
  });

  it("derives total from results length when server total is missing", () => {
    const item = {
      entity: "backlog" as const,
      id: "a",
      score: 0.8,
      scorePercent: 80,
      payload: {},
    };
    const normalized = normalizeAISearchResponse(
      { results: [item] },
      "backlog",
    );
    expect(normalized.total).toBe(1);
  });

  it("falls back to requested entity when server entity is invalid", () => {
    const normalized = normalizeAISearchResponse(
      { results: [], entity: "bogus" },
      "initiative",
    );
    expect(normalized.entity).toBe("initiative");
  });

  it("defaults fallback to 'unavailable' for invalid values", () => {
    const normalized = normalizeAISearchResponse(
      { results: [], fallback: "nonsense" },
      "both",
    );
    expect(normalized.fallback).toBe("unavailable");
  });

  it("handles null or undefined raw response", () => {
    expect(normalizeAISearchResponse(null, "both").results).toEqual([]);
    expect(normalizeAISearchResponse(undefined, "both").results).toEqual([]);
  });
});
