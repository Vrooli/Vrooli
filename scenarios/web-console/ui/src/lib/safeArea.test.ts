import { beforeEach, describe, expect, it, vi } from "vitest";
import { readSafeAreaInsets } from "./safeArea";

describe("safe-area reader", () => {
  beforeEach(() => {
    vi.stubGlobal("innerWidth", 390);
    vi.stubGlobal("innerHeight", 844);
  });

  it("reads, caches, and invalidates viewport-safe padding", () => {
    const first = readSafeAreaInsets();
    expect(first).toEqual({ top: 0, right: 0, bottom: 0, left: 0 });
    expect(readSafeAreaInsets()).toBe(first);
    vi.stubGlobal("innerWidth", 800);
    expect(readSafeAreaInsets()).not.toBe(first);
  });
});
