import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useCommandHistory } from "./useCommandHistory";

describe("useCommandHistory", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers({ now: Date.parse("2026-09-11T12:00:00Z"), toFake: ["Date"] });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("persists sends with when they happened, trims whitespace, and deduplicates adjacent entries", () => {
    const { result } = renderHook(() => useCommandHistory());
    act(() => {
      result.current.push("  pwd  ");
      result.current.push("pwd");
      result.current.push("ls");
      result.current.push("   ");
    });
    expect(result.current.entries.map((entry) => entry.text)).toEqual(["pwd", "ls"]);
    expect(result.current.entries[1]?.at).toBe(Date.parse("2026-09-11T12:00:00Z"));
    const stored = JSON.parse(localStorage.getItem("wc-command-history") ?? "null") as { text: string }[];
    expect(stored.map((entry) => entry.text)).toEqual(["pwd", "ls"]);
  });

  it("navigates older and newer entries and returns to the draft", () => {
    const { result } = renderHook(() => useCommandHistory());
    act(() => {
      result.current.push("one");
      result.current.push("two");
    });
    expect(result.current.navigateDown()).toBeNull();
    expect(result.current.navigateUp()).toBe("two");
    expect(result.current.navigateUp()).toBe("one");
    expect(result.current.navigateUp()).toBe("one");
    expect(result.current.navigateDown()).toBe("two");
    expect(result.current.navigateDown()).toBeNull();
    expect(result.current.navigateDown()).toBeNull();
  });

  it("recovers from malformed persisted history and resets navigation", () => {
    localStorage.setItem("wc-command-history", "{broken");
    const { result } = renderHook(() => useCommandHistory());
    expect(result.current.entries).toEqual([]);
    expect(result.current.navigateUp()).toBeNull();
    act(() => { result.current.push("echo ready"); });
    expect(result.current.navigateUp()).toBe("echo ready");
    act(() => { result.current.resetNavigation(); });
    expect(result.current.navigateDown()).toBeNull();
  });

  it("[REQ:P0-017f] clears the device's history", () => {
    const { result } = renderHook(() => useCommandHistory());
    act(() => { result.current.push("secret-ish"); });
    act(() => { result.current.clear(); });
    expect(result.current.entries).toEqual([]);
    expect(JSON.parse(localStorage.getItem("wc-command-history") ?? "null")).toEqual([]);
  });

  it("[REQ:P0-017f] shares one history between every mounted control", () => {
    // The toolbar and the expanded composer each hold the hook; a send from
    // one shows in the other's History, and neither overwrites the other.
    const toolbar = renderHook(() => useCommandHistory());
    const composer = renderHook(() => useCommandHistory());
    act(() => { toolbar.result.current.push("echo from toolbar"); });
    expect(composer.result.current.entries.map((entry) => entry.text)).toEqual(["echo from toolbar"]);
    act(() => { composer.result.current.push("echo from composer"); });
    expect(toolbar.result.current.entries.map((entry) => entry.text)).toEqual(["echo from toolbar", "echo from composer"]);
    const stored = JSON.parse(localStorage.getItem("wc-command-history") ?? "null") as { text: string }[];
    expect(stored.map((entry) => entry.text)).toEqual(["echo from toolbar", "echo from composer"]);
  });

  it("[REQ:P0-017f] keeps the last 50 sends", () => {
    const { result } = renderHook(() => useCommandHistory());
    act(() => {
      for (let index = 0; index < 55; index += 1) result.current.push(`command ${String(index)}`);
    });
    expect(result.current.entries).toHaveLength(50);
    expect(result.current.entries[0]?.text).toBe("command 5");
  });
});
