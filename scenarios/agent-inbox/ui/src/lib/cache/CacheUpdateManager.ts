/**
 * CacheUpdateManager - Coordinates cache invalidation with streaming operations.
 *
 * PROBLEM SOLVED: Multiple invalidateQueries calls racing with ongoing streaming
 * updates cause:
 * - React "too many re-renders" errors
 * - Stale active_leaf_message_id during streaming
 * - Message tree viewer out of sync with content
 *
 * SOLUTION: Defer invalidations while streaming, then batch them at the end.
 *
 * SEAM: Injectable CacheOperations interface for testing.
 */

import type { QueryClient, QueryKey, InvalidateQueryFilters } from "@tanstack/react-query";
import type { Message, Chat } from "../api";

/**
 * Interface for cache operations - enables mocking in tests.
 */
export interface CacheOperations {
  setQueryData<T>(key: QueryKey, updater: (old: T | undefined) => T): void;
  invalidateQueries(filters: InvalidateQueryFilters): Promise<void>;
}

/**
 * Adapter to wrap React Query's QueryClient for our interface.
 */
export function createQueryClientAdapter(queryClient: QueryClient): CacheOperations {
  return {
    setQueryData: <T>(key: QueryKey, updater: (old: T | undefined) => T) => {
      queryClient.setQueryData(key, updater);
    },
    invalidateQueries: (filters: InvalidateQueryFilters) => {
      return queryClient.invalidateQueries(filters);
    },
  };
}

/**
 * Chat data structure as stored in the cache.
 */
export interface CachedChatData {
  chat: Chat & { active_leaf_message_id?: string };
  messages: Message[];
  tool_call_records?: unknown[];
}

/**
 * Manages cache updates with streaming coordination.
 *
 * Key features:
 * - Defers invalidations while streaming is active
 * - Batches multiple invalidation requests
 * - Tracks active_leaf_message_id in optimistic updates
 * - Ensures single invalidation point when streaming ends
 */
export class CacheUpdateManager {
  private readonly cache: CacheOperations;
  private readonly streamingChatIds = new Set<string>();
  private readonly pendingInvalidations = new Map<string, Set<string>>();

  constructor(cache: CacheOperations) {
    this.cache = cache;
  }

  /**
   * Mark a chat as actively streaming.
   *
   * While streaming, all invalidation requests are deferred.
   * Call endStreaming() when streaming completes.
   */
  startStreaming(chatId: string): void {
    this.streamingChatIds.add(chatId);
    // Initialize pending invalidations set for this chat
    if (!this.pendingInvalidations.has(chatId)) {
      this.pendingInvalidations.set(chatId, new Set());
    }
  }

  /**
   * Check if a chat is currently streaming.
   */
  isStreaming(chatId: string): boolean {
    return this.streamingChatIds.has(chatId);
  }

  /**
   * End streaming for a chat and process all pending invalidations.
   *
   * This consolidates multiple invalidation requests into a single batch,
   * preventing cascading re-renders.
   */
  async endStreaming(chatId: string): Promise<void> {
    this.streamingChatIds.delete(chatId);

    // Get and clear pending invalidations
    const pending = this.pendingInvalidations.get(chatId);
    this.pendingInvalidations.delete(chatId);

    if (pending && pending.size > 0) {
      // Perform batched invalidations
      await this.performBatchedInvalidation(chatId, pending);
    } else {
      // Always invalidate the chat at minimum when streaming ends
      await this.invalidateChatImmediate(chatId);
    }
  }

  /**
   * Request cache invalidation for a chat.
   *
   * If streaming is active, the request is queued.
   * Otherwise, it's performed immediately.
   *
   * @param chatId - The chat to invalidate
   * @param queryKeys - Additional query keys to invalidate (e.g., "chats" for list)
   */
  async invalidateChat(chatId: string, queryKeys: string[] = ["chats"]): Promise<void> {
    if (this.streamingChatIds.has(chatId)) {
      // Queue for later
      const pending = this.pendingInvalidations.get(chatId) || new Set();
      pending.add("chat");
      for (const key of queryKeys) {
        pending.add(key);
      }
      this.pendingInvalidations.set(chatId, pending);
    } else {
      // Perform immediately
      await this.invalidateChatImmediate(chatId);
      for (const key of queryKeys) {
        await this.cache.invalidateQueries({ queryKey: [key] });
      }
    }
  }

  /**
   * Optimistically add a message to the cache with active_leaf tracking.
   *
   * This updates the cache immediately without waiting for a refetch,
   * providing instant UI feedback while tracking the expected leaf message.
   *
   * @param chatId - The chat to update
   * @param message - The new message to add
   * @param expectedLeafId - The expected active leaf message ID after this update
   */
  optimisticAddMessage(chatId: string, message: Message, expectedLeafId?: string): void {
    this.cache.setQueryData(["chat", chatId], (old: CachedChatData | undefined) => {
      if (!old) {
        // Fresh chat - create minimal structure with all required Chat fields
        const now = new Date().toISOString();
        return {
          chat: {
            id: chatId,
            name: "New Chat",
            model: "default",
            preview: "",
            view_mode: "bubble" as const,
            chat_mode: "llm" as const,
            is_read: true,
            is_starred: false,
            is_archived: false,
            web_search_enabled: false,
            label_ids: [],
            created_at: now,
            updated_at: now,
            active_leaf_message_id: expectedLeafId || message.id,
          },
          messages: [message],
          tool_call_records: [],
        } as CachedChatData;
      }

      return {
        ...old,
        chat: {
          ...old.chat,
          active_leaf_message_id: expectedLeafId || message.id,
        },
        messages: [...old.messages, message],
      };
    });
  }

  /**
   * Optimistically update the active leaf message ID.
   *
   * Used when the UI needs to track the current position in the message tree
   * without waiting for a server response.
   */
  optimisticSetActiveLeaf(chatId: string, messageId: string): void {
    this.cache.setQueryData(["chat", chatId], (old: CachedChatData | undefined) => {
      if (!old) return undefined as unknown as CachedChatData;
      return {
        ...old,
        chat: {
          ...old.chat,
          active_leaf_message_id: messageId,
        },
      };
    });
  }

  /**
   * Clear all pending state (useful for cleanup).
   */
  reset(): void {
    this.streamingChatIds.clear();
    this.pendingInvalidations.clear();
  }

  /**
   * Perform immediate invalidation for a single chat.
   */
  private async invalidateChatImmediate(chatId: string): Promise<void> {
    await this.cache.invalidateQueries({ queryKey: ["chat", chatId] });
  }

  /**
   * Perform batched invalidation for multiple query keys.
   */
  private async performBatchedInvalidation(chatId: string, keys: Set<string>): Promise<void> {
    const promises: Promise<void>[] = [];

    // Always invalidate the specific chat
    promises.push(this.cache.invalidateQueries({ queryKey: ["chat", chatId] }));

    // Invalidate other requested keys
    for (const key of keys) {
      if (key !== "chat") {
        promises.push(this.cache.invalidateQueries({ queryKey: [key] }));
      }
    }

    await Promise.all(promises);
  }
}
