import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { clearHistory, loadHistory, recordSearch } from "./searchHistory";

describe("searchHistory", () => {
  beforeEach(() => {
    window.localStorage.clear();
    clearHistory();
  });
  afterEach(() => {
    window.localStorage.clear();
  });

  it("records a search most-recent-first", () => {
    recordSearch("first", "live");
    const after = recordSearch("second", "learnings");
    expect(after.map((e) => e.query)).toEqual(["second", "first"]);
    expect(after[0]?.mode).toBe("learnings");
  });

  it("de-duplicates the same (query, mode) and bubbles it to the front", () => {
    recordSearch("a", "live");
    recordSearch("b", "live");
    const after = recordSearch("a", "live");
    expect(after.map((e) => e.query)).toEqual(["a", "b"]);
  });

  it("treats the same query under different modes as distinct entries", () => {
    recordSearch("a", "live");
    const after = recordSearch("a", "learnings");
    expect(after).toHaveLength(2);
  });

  it("ignores blank queries", () => {
    const after = recordSearch("   ", "live");
    expect(after).toHaveLength(0);
  });

  it("persists across loadHistory and clears", () => {
    recordSearch("x", "live");
    expect(loadHistory()).toHaveLength(1);
    expect(clearHistory()).toHaveLength(0);
    expect(loadHistory()).toHaveLength(0);
  });
  it("discards malformed persisted history and preserves valid entries", () => {
    const key = "web-search:history";
    localStorage.setItem(key, "{broken");
    expect(loadHistory()).toEqual([]);
    localStorage.setItem(key, JSON.stringify({ query: "not an array" }));
    expect(loadHistory()).toEqual([]);
    const valid = { query: "verified", mode: "live", at: 100 };
    localStorage.setItem(key, JSON.stringify([null, "bad", { query: 4 }, { query: "bad", mode: "unsupported" }, { query: "bad", mode: "live", at: null }, valid]));
    expect(loadHistory()).toEqual([valid]);
  });

});
