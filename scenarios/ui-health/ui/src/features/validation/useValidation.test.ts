import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

import { upsertRecent, recentRunFromResult, useRecentRuns } from "./useValidation";
import type { ValidationResult } from "../../api/validation";

const result: ValidationResult = {
  scenario: "demo",
  passed: false,
  findings: [],
  summary: { errors: 2, warnings: 1, infos: 0 },
  ranAt: "2026-05-20T12:00:00.000Z",
};

describe("upsertRecent", () => {
  it("prepends a new run", () => {
    const out = upsertRecent([], recentRunFromResult(result));
    expect(out).toHaveLength(1);
    expect(out[0]?.scenario).toBe("demo");
  });

  it("replaces an existing run for the same scenario at the top", () => {
    const existing = [
      { scenario: "demo", passed: true, errors: 0, warnings: 0, infos: 0, ranAt: "older" },
      { scenario: "other", passed: true, errors: 0, warnings: 0, infos: 0, ranAt: "older" },
    ];
    const out = upsertRecent(existing, recentRunFromResult(result));
    expect(out).toHaveLength(2);
    expect(out[0]?.scenario).toBe("demo");
    expect(out[0]?.errors).toBe(2);
    expect(out[1]?.scenario).toBe("other");
  });

  it("caps the list at 25 entries", () => {
    const seed = Array.from({ length: 25 }, (_, i) => ({
      scenario: `s${i}`,
      passed: true,
      errors: 0,
      warnings: 0,
      infos: 0,
      ranAt: "x",
    }));
    const out = upsertRecent(seed, recentRunFromResult(result));
    expect(out).toHaveLength(25);
    expect(out[0]?.scenario).toBe("demo");
    expect(out.at(-1)?.scenario).toBe("s23");
  });
});

describe("useRecentRuns", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("starts empty when storage has no entry", () => {
    const { result: hook } = renderHook(() => useRecentRuns());
    expect(hook.current.runs).toEqual([]);
  });

  it("records a result and persists it to localStorage", () => {
    const { result: hook } = renderHook(() => useRecentRuns());
    act(() => {
      hook.current.record(result);
    });
    expect(hook.current.runs).toHaveLength(1);
    const stored = window.localStorage.getItem("ui-health.validation.recent.v1");
    expect(stored).toContain("demo");
  });

  it("clears all entries", () => {
    const { result: hook } = renderHook(() => useRecentRuns());
    act(() => {
      hook.current.record(result);
      hook.current.clear();
    });
    expect(hook.current.runs).toEqual([]);
  });

  it("survives corrupt storage", () => {
    window.localStorage.setItem("ui-health.validation.recent.v1", "not-json");
    const { result: hook } = renderHook(() => useRecentRuns());
    expect(hook.current.runs).toEqual([]);
  });
});
