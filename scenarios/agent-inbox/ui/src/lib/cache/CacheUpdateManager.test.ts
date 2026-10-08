import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  CacheUpdateManager,
  type CacheOperations,
  type CachedChatData,
} from "./CacheUpdateManager";
import type { Message } from "../api";

describe("CacheUpdateManager", () => {
  let mockCache: CacheOperations;
  let manager: CacheUpdateManager;
  let setQueryDataCalls: Array<{ key: unknown[]; updater: unknown }>;
  let invalidateQueriesCalls: Array<{ queryKey?: unknown[] }>;

  beforeEach(() => {
    setQueryDataCalls = [];
    invalidateQueriesCalls = [];

    mockCache = {
      setQueryData: vi.fn((key, updater) => {
        setQueryDataCalls.push({ key: key as unknown[], updater });
      }),
      invalidateQueries: vi.fn((filters: { queryKey?: unknown[] }) => {
        invalidateQueriesCalls.push(filters);
        return Promise.resolve();
      }),
    };

    manager = new CacheUpdateManager(mockCache);
  });

  describe("streaming coordination", () => {
    it("defers invalidations while streaming", async () => {
      manager.startStreaming("chat-123");

      await manager.invalidateChat("chat-123", ["chats"]);

      // Should not have called invalidateQueries yet
      expect(invalidateQueriesCalls).toHaveLength(0);
    });

    it("batches multiple invalidation requests", async () => {
      manager.startStreaming("chat-123");

      await manager.invalidateChat("chat-123", ["chats"]);
      await manager.invalidateChat("chat-123", ["labels"]);
      await manager.invalidateChat("chat-123", ["chats"]); // Duplicate

      // Still deferred
      expect(invalidateQueriesCalls).toHaveLength(0);

      // End streaming triggers batched invalidation
      await manager.endStreaming("chat-123");

      // Should have invalidated: ["chat", "chat-123"], ["chats"], ["labels"]
      expect(invalidateQueriesCalls).toHaveLength(3);
      expect(invalidateQueriesCalls).toContainEqual({ queryKey: ["chat", "chat-123"] });
      expect(invalidateQueriesCalls).toContainEqual({ queryKey: ["chats"] });
      expect(invalidateQueriesCalls).toContainEqual({ queryKey: ["labels"] });
    });

    it("performs immediate invalidation when not streaming", async () => {
      await manager.invalidateChat("chat-123", ["chats"]);

      expect(invalidateQueriesCalls).toHaveLength(2);
      expect(invalidateQueriesCalls).toContainEqual({ queryKey: ["chat", "chat-123"] });
      expect(invalidateQueriesCalls).toContainEqual({ queryKey: ["chats"] });
    });

    it("isStreaming returns correct state", () => {
      expect(manager.isStreaming("chat-123")).toBe(false);

      manager.startStreaming("chat-123");
      expect(manager.isStreaming("chat-123")).toBe(true);

      void manager.endStreaming("chat-123");
      expect(manager.isStreaming("chat-123")).toBe(false);
    });

    it("handles streaming errors without orphaning cache state", async () => {
      manager.startStreaming("chat-123");

      // Simulate error during streaming - endStreaming should still work
      await manager.endStreaming("chat-123");

      expect(manager.isStreaming("chat-123")).toBe(false);
      // Should have invalidated at minimum the chat
      expect(invalidateQueriesCalls.some(c =>
        Array.isArray(c.queryKey) && c.queryKey[0] === "chat" && c.queryKey[1] === "chat-123"
      )).toBe(true);
    });

    it("always invalidates chat when streaming ends even with no pending", async () => {
      manager.startStreaming("chat-123");
      // Don't request any invalidations
      await manager.endStreaming("chat-123");

      // Should still invalidate the chat
      expect(invalidateQueriesCalls).toHaveLength(1);
      expect(invalidateQueriesCalls[0]).toEqual({ queryKey: ["chat", "chat-123"] });
    });
  });

  describe("optimistic updates", () => {
    it("includes active_leaf_message_id in optimistic updates", () => {
      const message: Message = {
        id: "msg-123",
        chat_id: "chat-123",
        role: "user",
        content: "Hello",
        token_count: 1,
        sibling_index: 0,
        created_at: new Date().toISOString(),
      };

      manager.optimisticAddMessage("chat-123", message, "msg-123");

      expect(setQueryDataCalls).toHaveLength(1);
      expect(setQueryDataCalls[0]!.key).toEqual(["chat", "chat-123"]);

      // Call the updater with undefined (fresh chat)
      const updater = setQueryDataCalls[0]!.updater as (old: CachedChatData | undefined) => CachedChatData;
      const result = updater(undefined);

      expect(result.chat.active_leaf_message_id).toBe("msg-123");
      expect(result.messages).toHaveLength(1);
      expect(result.messages[0]!.id).toBe("msg-123");
    });

    it("updates existing chat with new message and leaf", () => {
      const existingChat: CachedChatData = {
        chat: {
          id: "chat-123",
          name: "Existing Chat",
          model: "test",
          preview: "",
          view_mode: "bubble",
          chat_mode: "llm",
          is_read: true,
          is_starred: false,
          is_archived: false,
          web_search_enabled: false,
          label_ids: [],
          created_at: "2024-01-01",
          updated_at: "2024-01-01",
          active_leaf_message_id: "old-msg",
        },
        messages: [
          { id: "old-msg", chat_id: "chat-123", role: "user", content: "Old", token_count: 1, sibling_index: 0, created_at: "" },
        ],
        tool_call_records: [],
      };

      const newMessage: Message = {
        id: "new-msg",
        chat_id: "chat-123",
        role: "assistant",
        content: "Response",
        token_count: 5,
        sibling_index: 0,
        created_at: new Date().toISOString(),
      };

      manager.optimisticAddMessage("chat-123", newMessage, "new-msg");

      const updater = setQueryDataCalls[0]!.updater as (old: CachedChatData | undefined) => CachedChatData;
      const result = updater(existingChat);

      expect(result.chat.active_leaf_message_id).toBe("new-msg");
      expect(result.messages).toHaveLength(2);
      expect(result.messages[1]!.id).toBe("new-msg");
    });

    it("optimisticSetActiveLeaf updates only the leaf", () => {
      const existingChat: CachedChatData = {
        chat: {
          id: "chat-123",
          name: "Test",
          model: "test",
          preview: "",
          view_mode: "bubble",
          chat_mode: "llm",
          is_read: true,
          is_starred: false,
          is_archived: false,
          web_search_enabled: false,
          label_ids: [],
          created_at: "",
          updated_at: "",
          active_leaf_message_id: "old-leaf",
        },
        messages: [],
        tool_call_records: [],
      };

      manager.optimisticSetActiveLeaf("chat-123", "new-leaf");

      const updater = setQueryDataCalls[0]!.updater as (old: CachedChatData | undefined) => CachedChatData;
      const result = updater(existingChat);

      expect(result.chat.active_leaf_message_id).toBe("new-leaf");
    });

    it("optimisticSetActiveLeaf handles undefined old data gracefully", () => {
      manager.optimisticSetActiveLeaf("chat-123", "leaf");

      const updater = setQueryDataCalls[0]!.updater as (old: CachedChatData | undefined) => CachedChatData | undefined;
      const result = updater(undefined);

      // Should return undefined if no existing data
      expect(result).toBeUndefined();
    });
  });

  describe("reset", () => {
    it("clears all pending state", async () => {
      manager.startStreaming("chat-123");
      await manager.invalidateChat("chat-123");

      manager.reset();

      expect(manager.isStreaming("chat-123")).toBe(false);

      // New invalidation should work immediately
      await manager.invalidateChat("chat-123");
      expect(invalidateQueriesCalls).toHaveLength(2); // chat + chats
    });
  });

  describe("multiple chats", () => {
    it("handles independent streaming for different chats", async () => {
      manager.startStreaming("chat-1");
      manager.startStreaming("chat-2");

      await manager.invalidateChat("chat-1", ["chats"]);
      await manager.invalidateChat("chat-2", ["chats"]);

      // Nothing invalidated yet
      expect(invalidateQueriesCalls).toHaveLength(0);

      // End streaming for chat-1
      await manager.endStreaming("chat-1");

      // Only chat-1 should be invalidated
      expect(invalidateQueriesCalls.filter(c =>
        Array.isArray(c.queryKey) && c.queryKey[1] === "chat-1"
      )).toHaveLength(1);

      // chat-2 still streaming
      expect(manager.isStreaming("chat-2")).toBe(true);

      // End streaming for chat-2
      await manager.endStreaming("chat-2");

      // Now both should be invalidated
      expect(invalidateQueriesCalls.filter(c =>
        Array.isArray(c.queryKey) && c.queryKey[1] === "chat-2"
      )).toHaveLength(1);
    });
  });
});
