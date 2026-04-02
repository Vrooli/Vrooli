import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { useSnoozeStore } from "./snooze-store";

const { getState, setState } = useSnoozeStore;

beforeEach(() => {
  // Reset to empty state before each test
  setState({ entries: new Map() });
  vi.restoreAllMocks();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("snooze-store", () => {
  describe("snooze()", () => {
    it("adds an entry to the map", () => {
      const future = Date.now() + 60_000;
      getState().snooze("backlog:idea/test", future);

      const entry = getState().entries.get("backlog:idea/test");
      expect(entry).toEqual({ key: "backlog:idea/test", expiresAt: future });
    });

    it("overwrites an existing entry", () => {
      getState().snooze("backlog:idea/test", Date.now() + 60_000);
      const newExpiry = Date.now() + 120_000;
      getState().snooze("backlog:idea/test", newExpiry);

      expect(getState().entries.get("backlog:idea/test")?.expiresAt).toBe(newExpiry);
      expect(getState().entries.size).toBe(1);
    });
  });

  describe("unsnooze()", () => {
    it("removes an existing entry", () => {
      getState().snooze("backlog:idea/test", Date.now() + 60_000);
      getState().unsnooze("backlog:idea/test");

      expect(getState().entries.has("backlog:idea/test")).toBe(false);
    });

    it("no-ops when key does not exist", () => {
      getState().unsnooze("nonexistent");
      expect(getState().entries.size).toBe(0);
    });
  });

  describe("isSnoozed()", () => {
    it("returns true for a valid non-expired entry", () => {
      getState().snooze("backlog:idea/test", Date.now() + 60_000);
      expect(getState().isSnoozed("backlog:idea/test")).toBe(true);
    });

    it("returns false for a nonexistent key", () => {
      expect(getState().isSnoozed("nonexistent")).toBe(false);
    });

    it("returns false for an expired entry", () => {
      getState().snooze("backlog:idea/test", Date.now() - 1000);
      expect(getState().isSnoozed("backlog:idea/test")).toBe(false);
    });
  });

  describe("snoozedKeys()", () => {
    it("returns empty set when no entries", () => {
      expect(getState().snoozedKeys().size).toBe(0);
    });

    it("returns only non-expired keys", () => {
      getState().snooze("key-a", Date.now() + 60_000);
      getState().snooze("key-b", Date.now() - 1000); // expired
      getState().snooze("key-c", Date.now() + 120_000);

      const keys = getState().snoozedKeys();
      expect(keys.has("key-a")).toBe(true);
      expect(keys.has("key-b")).toBe(false);
      expect(keys.has("key-c")).toBe(true);
      expect(keys.size).toBe(2);
    });
  });

  describe("purgeExpired()", () => {
    it("removes expired entries", () => {
      getState().snooze("active", Date.now() + 60_000);
      getState().snooze("expired-1", Date.now() - 1000);
      getState().snooze("expired-2", Date.now() - 5000);

      getState().purgeExpired();

      expect(getState().entries.size).toBe(1);
      expect(getState().entries.has("active")).toBe(true);
    });

    it("no-ops when nothing is expired", () => {
      getState().snooze("a", Date.now() + 60_000);
      getState().snooze("b", Date.now() + 120_000);

      const entriesBefore = getState().entries;
      getState().purgeExpired();

      // Should return same reference (no change)
      expect(getState().entries).toBe(entriesBefore);
    });
  });
});
