import { describe, it, expect, beforeEach } from "vitest";
import { useClarificationStore } from "./clarification-store";
import type { ClarificationThread } from "../types/domain";

const { getState, setState } = useClarificationStore;

const MOCK_TARGET = {
  backlogKind: "idea" as const,
  backlogName: "test-item",
  roundNumber: 1,
  itemId: "d1",
  itemTopic: "Test decision",
};

const MOCK_THREAD: ClarificationThread = {
  id: "thread-1",
  round_number: 1,
  item_id: "d1",
  run_id: "run-abc",
  messages: [{ role: "user", content: "Why?", created_at: "2026-04-01T00:00:00Z" }],
  status: "active",
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
};

beforeEach(() => {
  setState({
    isOpen: false,
    target: null,
    thread: null,
    isCreating: false,
    isLoading: false,
  });
});

describe("clarification-store", () => {
  describe("open()", () => {
    it("sets isOpen, target, and clears thread", () => {
      getState().open(MOCK_TARGET);

      const s = getState();
      expect(s.isOpen).toBe(true);
      expect(s.target).toEqual(MOCK_TARGET);
      expect(s.thread).toBeNull();
      expect(s.isCreating).toBe(false);
    });

    it("sets isLoading false when no clarificationId", () => {
      getState().open(MOCK_TARGET);
      expect(getState().isLoading).toBe(false);
    });

    it("sets isLoading true when clarificationId is present", () => {
      getState().open({ ...MOCK_TARGET, clarificationId: "thread-1" });
      expect(getState().isLoading).toBe(true);
    });
  });

  describe("close()", () => {
    it("resets all state", () => {
      getState().open({ ...MOCK_TARGET, clarificationId: "thread-1" });
      getState().setThread(MOCK_THREAD);
      getState().close();

      const s = getState();
      expect(s.isOpen).toBe(false);
      expect(s.target).toBeNull();
      expect(s.thread).toBeNull();
      expect(s.isCreating).toBe(false);
      expect(s.isLoading).toBe(false);
    });
  });

  describe("setThread()", () => {
    it("updates thread", () => {
      getState().setThread(MOCK_THREAD);
      expect(getState().thread).toEqual(MOCK_THREAD);
    });
  });

  describe("setCreating()", () => {
    it("toggles isCreating", () => {
      getState().setCreating(true);
      expect(getState().isCreating).toBe(true);
      getState().setCreating(false);
      expect(getState().isCreating).toBe(false);
    });
  });

  describe("setLoading()", () => {
    it("toggles isLoading", () => {
      getState().setLoading(true);
      expect(getState().isLoading).toBe(true);
      getState().setLoading(false);
      expect(getState().isLoading).toBe(false);
    });
  });
});
