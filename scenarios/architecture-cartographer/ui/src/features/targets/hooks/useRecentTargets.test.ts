import { describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useRecentTargets } from "./useRecentTargets";

function makeStorage(initial: Record<string, string> = {}) {
  const store: Record<string, string> = { ...initial };
  return {
    getItem: (key: string) => (key in store ? store[key]! : null),
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    snapshot: () => ({ ...store }),
  };
}

describe("useRecentTargets", () => {
  it("returns an empty list when nothing is stored", () => {
    const storage = makeStorage();
    const { result } = renderHook(() => useRecentTargets({ storage }));
    expect(result.current.recent).toEqual([]);
  });

  it("records a scenario at the head with the current timestamp", () => {
    const storage = makeStorage();
    const fixedNow = new Date("2026-01-02T03:04:05.000Z");
    const { result } = renderHook(() =>
      useRecentTargets({ storage, now: () => fixedNow }),
    );

    act(() => {
      result.current.record("architecture-cartographer");
    });

    expect(result.current.recent).toEqual([
      { scenario: "architecture-cartographer", lastOpenedAt: fixedNow.toISOString() },
    ]);
  });

  it("moves an existing scenario to the head rather than duplicating it", () => {
    const storage = makeStorage();
    const t1 = new Date("2026-01-02T00:00:00.000Z");
    const t2 = new Date("2026-01-03T00:00:00.000Z");
    let n = 0;
    const now = () => (++n === 1 ? t1 : t2);
    const { result } = renderHook(() => useRecentTargets({ storage, now }));

    act(() => result.current.record("a"));
    act(() => result.current.record("b"));
    act(() => result.current.record("a"));

    expect(result.current.recent.map((entry) => entry.scenario)).toEqual(["a", "b"]);
    expect(result.current.recent[0]?.lastOpenedAt).toBe(t2.toISOString());
  });

  it("removes a scenario", () => {
    const storage = makeStorage();
    const { result } = renderHook(() =>
      useRecentTargets({ storage, now: () => new Date(0) }),
    );

    act(() => result.current.record("a"));
    act(() => result.current.record("b"));
    act(() => result.current.remove("a"));

    expect(result.current.recent.map((e) => e.scenario)).toEqual(["b"]);
  });

  it("clears the whole list", () => {
    const storage = makeStorage();
    const { result } = renderHook(() =>
      useRecentTargets({ storage, now: () => new Date(0) }),
    );

    act(() => result.current.record("a"));
    act(() => result.current.clear());

    expect(result.current.recent).toEqual([]);
  });

  it("caps the list at MAX_ENTRIES (8)", () => {
    const storage = makeStorage();
    let counter = 0;
    const now = () => new Date(`2026-01-01T00:00:${String(counter++).padStart(2, "0")}.000Z`);
    const { result } = renderHook(() => useRecentTargets({ storage, now }));

    act(() => {
      for (let i = 0; i < 12; i++) {
        result.current.record(`scenario-${i}`);
      }
    });

    expect(result.current.recent).toHaveLength(8);
    expect(result.current.recent[0]?.scenario).toBe("scenario-11");
  });

  it("ignores corrupt persisted entries during validation", () => {
    const storage = makeStorage({
      "cartographer.recentTargets": JSON.stringify([
        { scenario: "good", lastOpenedAt: "2026-01-01T00:00:00.000Z" },
        { scenario: "" },
        { lastOpenedAt: "x" },
        "not-an-object",
      ]),
    });
    const { result } = renderHook(() => useRecentTargets({ storage }));
    expect(result.current.recent).toEqual([
      { scenario: "good", lastOpenedAt: "2026-01-01T00:00:00.000Z" },
    ]);
  });

  it("ignores an empty-string scenario on record()", () => {
    const storage = makeStorage();
    const { result } = renderHook(() =>
      useRecentTargets({ storage, now: () => new Date(0) }),
    );
    act(() => result.current.record(""));
    expect(result.current.recent).toEqual([]);
  });
});
