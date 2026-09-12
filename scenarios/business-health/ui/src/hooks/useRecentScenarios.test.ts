import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useRecentScenarios } from "./useRecentScenarios";

describe("useRecentScenarios", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("remembers most-recent-first and de-duplicates", () => {
    const { result } = renderHook(() => useRecentScenarios());
    act(() => result.current.remember("a"));
    act(() => result.current.remember("b"));
    act(() => result.current.remember("a"));
    expect(result.current.recents).toEqual(["a", "b"]);
  });

  it("ignores blank slugs", () => {
    const { result } = renderHook(() => useRecentScenarios());
    act(() => result.current.remember("   "));
    expect(result.current.recents).toEqual([]);
  });

  it("persists across hook instances via localStorage", () => {
    const first = renderHook(() => useRecentScenarios());
    act(() => first.result.current.remember("persisted"));
    const second = renderHook(() => useRecentScenarios());
    expect(second.result.current.recents).toContain("persisted");
  });

  it("clears recents", () => {
    const { result } = renderHook(() => useRecentScenarios());
    act(() => result.current.remember("x"));
    act(() => result.current.clear());
    expect(result.current.recents).toEqual([]);
  });
});
