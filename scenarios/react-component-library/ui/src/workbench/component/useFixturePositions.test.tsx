import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useFixturePositions } from "./useFixturePositions";

const key = (id: string) => `rcl.workbench.fixtures.${id}.v1`;

describe("useFixturePositions", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("loads saved positions and persists edits after the debounce", () => {
    localStorage.setItem(key("alpha"), JSON.stringify({ primary: { x: 32, y: 48 } }));
    const { result } = renderHook(() => useFixturePositions("alpha"));

    expect(result.current.positions).toEqual({ primary: { x: 32, y: 48 } });
    act(() => result.current.persistPositions({ primary: { x: 80, y: 96 } }));
    expect(result.current.positions).toEqual({ primary: { x: 80, y: 96 } });
    expect(localStorage.getItem(key("alpha"))).toEqual(JSON.stringify({ primary: { x: 32, y: 48 } }));

    act(() => { vi.advanceTimersByTime(199); });
    expect(localStorage.getItem(key("alpha"))).toEqual(JSON.stringify({ primary: { x: 32, y: 48 } }));
    act(() => { vi.advanceTimersByTime(1); });
    expect(localStorage.getItem(key("alpha"))).toEqual(JSON.stringify({ primary: { x: 80, y: 96 } }));
  });

  it("replaces the in-memory arrangement when the asset id changes", () => {
    localStorage.setItem(key("alpha"), JSON.stringify({ primary: { x: 11, y: 12 } }));
    localStorage.setItem(key("beta"), JSON.stringify({ compact: { x: 120, y: 130 } }));
    const { result, rerender } = renderHook(({ id }) => useFixturePositions(id), { initialProps: { id: "alpha" } });

    expect(result.current.positions).toEqual({ primary: { x: 11, y: 12 } });
    rerender({ id: "beta" });
    expect(result.current.positions).toEqual({ compact: { x: 120, y: 130 } });
  });

  it.each([
    "not-json",
    "[]",
    "null",
    "42",
    "true",
    JSON.stringify("position"),
    JSON.stringify({ primary: null }),
    JSON.stringify({ primary: { x: 1 } }),
    JSON.stringify({ primary: { x: "1", y: 2 } }),
  ])("ignores malformed stored payload %s", (raw) => {
    localStorage.setItem(key("alpha"), raw);
    const { result } = renderHook(() => useFixturePositions("alpha"));
    expect(result.current.positions).toEqual({});
  });

  it("keeps an in-memory draft when storage writes fail", () => {
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("storage unavailable");
    });
    const { result } = renderHook(() => useFixturePositions("alpha"));
    act(() => result.current.persistPositions({ primary: { x: 40, y: 50 } }));
    act(() => { vi.advanceTimersByTime(200); });

    expect(result.current.positions).toEqual({ primary: { x: 40, y: 50 } });
    expect(setItem).toHaveBeenCalledOnce();
  });
});
