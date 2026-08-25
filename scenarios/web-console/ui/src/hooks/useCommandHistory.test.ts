import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { useCommandHistory } from "./useCommandHistory";

describe("useCommandHistory", () => {
  beforeEach(() => localStorage.clear());

  it("persists commands, trims whitespace, and deduplicates adjacent entries", () => {
    const { result } = renderHook(() => useCommandHistory());
    act(() => {
      result.current.push("  pwd  ");
      result.current.push("pwd");
      result.current.push("ls");
      result.current.push("   ");
    });
    expect(result.current.entries).toEqual(["pwd", "ls"]);
    expect(JSON.parse(localStorage.getItem("wc-command-history") ?? "null")).toEqual(["pwd", "ls"]);
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
    act(() => result.current.push("echo ready"));
    expect(result.current.navigateUp()).toBe("echo ready");
    act(() => result.current.resetNavigation());
    expect(result.current.navigateDown()).toBeNull();
  });
});
