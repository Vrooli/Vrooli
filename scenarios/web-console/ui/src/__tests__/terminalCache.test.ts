import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  saveTerminalCache,
  loadTerminalCache,
  clearTerminalCache,
  CACHE_MAX_AGE_MS,
  CACHE_MAX_SIZE,
} from "../lib/terminalCache";
import type { TerminalCacheEntry } from "../lib/terminalCache";

describe("terminalCache", () => {
  beforeEach(() => sessionStorage.clear());

  it("saves and loads cache entry", () => {
    const entry: TerminalCacheEntry = {
      serialized: "some-terminal-state",
      totalBytes: 12345,
      savedAt: Date.now(),
    };
    const saved = saveTerminalCache("sess-1", entry);
    expect(saved).toBe(true);

    const loaded = loadTerminalCache("sess-1");
    expect(loaded).toEqual(entry);
  });

  it("returns null for missing session", () => {
    expect(loadTerminalCache("nonexistent")).toBeNull();
  });

  it("returns null for expired cache", () => {
    const entry: TerminalCacheEntry = {
      serialized: "old-state",
      totalBytes: 100,
      savedAt: Date.now() - CACHE_MAX_AGE_MS - 1,
    };
    saveTerminalCache("sess-expired", entry);

    expect(loadTerminalCache("sess-expired")).toBeNull();

    // Stale entry should have been removed from sessionStorage
    expect(sessionStorage.getItem("wc-terminal-cache-sess-expired")).toBeNull();
  });

  it("returns null for corrupt JSON", () => {
    sessionStorage.setItem("wc-terminal-cache-bad", "not json");
    expect(loadTerminalCache("bad")).toBeNull();
  });

  it("rejects oversized serialized data", () => {
    const entry: TerminalCacheEntry = {
      serialized: "x".repeat(CACHE_MAX_SIZE + 1),
      totalBytes: 0,
      savedAt: Date.now(),
    };
    const saved = saveTerminalCache("sess-big", entry);
    expect(saved).toBe(false);

    expect(sessionStorage.getItem("wc-terminal-cache-sess-big")).toBeNull();
  });

  it("clearTerminalCache removes entry", () => {
    const entry: TerminalCacheEntry = {
      serialized: "state",
      totalBytes: 1,
      savedAt: Date.now(),
    };
    saveTerminalCache("sess-clear", entry);
    expect(loadTerminalCache("sess-clear")).not.toBeNull();

    clearTerminalCache("sess-clear");
    expect(loadTerminalCache("sess-clear")).toBeNull();
  });

  it("handles sessionStorage quota error", () => {
    const original = Storage.prototype.setItem;
    Storage.prototype.setItem = () => {
      throw new DOMException("QuotaExceededError");
    };

    const entry: TerminalCacheEntry = {
      serialized: "data",
      totalBytes: 1,
      savedAt: Date.now(),
    };
    const saved = saveTerminalCache("sess-quota", entry);
    expect(saved).toBe(false);

    Storage.prototype.setItem = original;
  });
});
