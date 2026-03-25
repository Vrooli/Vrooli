import { describe, it, expect, beforeEach, vi } from "vitest";
import { loadFromStorage, saveToStorage, clearStorage, type StorePersistConfig } from "./store-utils";

const TEST_CONFIG: StorePersistConfig = {
  key: "test-store.v1",
  version: 1,
  maxItems: 5,
};

beforeEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe("loadFromStorage", () => {
  it("returns fallback when localStorage is empty", () => {
    const result = loadFromStorage(TEST_CONFIG, []);
    expect(result).toEqual({ data: [], lastFetchedAt: null });
  });

  it("returns fallback when version mismatches", () => {
    const envelope = { data: ["a", "b"], fetchedAt: Date.now(), version: 99 };
    localStorage.setItem(TEST_CONFIG.key, JSON.stringify(envelope));

    const result = loadFromStorage(TEST_CONFIG, []);
    expect(result).toEqual({ data: [], lastFetchedAt: null });
  });

  it("returns fallback when data is expired", () => {
    const oldTime = Date.now() - 600_000; // 10 minutes ago (exceeds 5-min cacheTimeMs)
    const envelope = { data: ["a"], fetchedAt: oldTime, version: 1 };
    localStorage.setItem(TEST_CONFIG.key, JSON.stringify(envelope));

    const result = loadFromStorage(TEST_CONFIG, []);
    expect(result).toEqual({ data: [], lastFetchedAt: null });
  });

  it("returns fallback on malformed JSON", () => {
    localStorage.setItem(TEST_CONFIG.key, "not-json{{{");

    const result = loadFromStorage(TEST_CONFIG, []);
    expect(result).toEqual({ data: [], lastFetchedAt: null });
  });

  it("returns fallback when envelope is missing fetchedAt", () => {
    const envelope = { data: ["a"], version: 1 };
    localStorage.setItem(TEST_CONFIG.key, JSON.stringify(envelope));

    const result = loadFromStorage(TEST_CONFIG, []);
    expect(result).toEqual({ data: [], lastFetchedAt: null });
  });

  it("returns data and lastFetchedAt when valid and fresh", () => {
    const now = Date.now();
    const envelope = { data: ["a", "b"], fetchedAt: now, version: 1 };
    localStorage.setItem(TEST_CONFIG.key, JSON.stringify(envelope));

    const result = loadFromStorage(TEST_CONFIG, []);
    expect(result).toEqual({ data: ["a", "b"], lastFetchedAt: now });
  });

  it("truncates arrays exceeding maxItems", () => {
    const now = Date.now();
    const envelope = { data: [1, 2, 3, 4, 5, 6, 7, 8], fetchedAt: now, version: 1 };
    localStorage.setItem(TEST_CONFIG.key, JSON.stringify(envelope));

    const result = loadFromStorage(TEST_CONFIG, []);
    expect(result.data).toEqual([1, 2, 3, 4, 5]);
    expect(result.lastFetchedAt).toBe(now);
  });

  it("works with non-array data", () => {
    const now = Date.now();
    const config: StorePersistConfig = { key: "test-obj.v1", version: 1 };
    const envelope = { data: { foo: "bar" }, fetchedAt: now, version: 1 };
    localStorage.setItem(config.key, JSON.stringify(envelope));

    const result = loadFromStorage(config, {});
    expect(result).toEqual({ data: { foo: "bar" }, lastFetchedAt: now });
  });
});

describe("saveToStorage", () => {
  it("writes a valid envelope to localStorage", () => {
    const now = Date.now();
    saveToStorage(TEST_CONFIG, ["a", "b"], now);

    const raw = localStorage.getItem(TEST_CONFIG.key);
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw!);
    expect(parsed).toEqual({ data: ["a", "b"], fetchedAt: now, version: 1 });
  });

  it("truncates arrays to maxItems before saving", () => {
    saveToStorage(TEST_CONFIG, [1, 2, 3, 4, 5, 6, 7], Date.now());

    const parsed = JSON.parse(localStorage.getItem(TEST_CONFIG.key)!);
    expect(parsed.data).toEqual([1, 2, 3, 4, 5]);
  });

  it("silently handles QuotaExceededError", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("quota exceeded", "QuotaExceededError");
    });

    // Should not throw
    expect(() => saveToStorage(TEST_CONFIG, ["a"], Date.now())).not.toThrow();
  });
});

describe("clearStorage", () => {
  it("removes key from localStorage", () => {
    localStorage.setItem(TEST_CONFIG.key, "data");
    clearStorage(TEST_CONFIG.key);
    expect(localStorage.getItem(TEST_CONFIG.key)).toBeNull();
  });

  it("silently handles errors", () => {
    vi.spyOn(Storage.prototype, "removeItem").mockImplementation(() => {
      throw new Error("fail");
    });

    expect(() => clearStorage(TEST_CONFIG.key)).not.toThrow();
  });
});
