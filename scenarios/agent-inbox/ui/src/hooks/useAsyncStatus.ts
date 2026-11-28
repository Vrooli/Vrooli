/**
 * useAsyncStatus - Hook for tracking async tool operations via SSE.
 *
 * This hook connects to the async-status SSE endpoint for a chat and
 * provides real-time updates about long-running tool operations like
 * spawn_coding_agent.
 *
 * ARCHITECTURE:
 * - Connects to /api/v1/chats/{id}/async-status via EventSource
 * - Receives real-time status updates (progress, phase, completion)
 * - Maintains a map of operations with their current status
 * - Completed operations are NOT auto-removed (available for history view)
 *
 * USAGE:
 * ```tsx
 * const { operations, activeOperations, completedOperations, isConnected } = useAsyncStatus(chatId);
 * ```
 */
import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import {
  API_BASE,
  EMPTY_OPERATIONS,
  isAsyncStatusUpdate,
  isAsyncHistoryResponse,
  type AsyncStatusUpdate,
  type UseAsyncStatusOptions,
  type UseAsyncStatusReturn,
} from "./useAsyncStatus.types";

// Re-export all types for consumers
export type {
  AsyncStatusUpdate,
  AsyncHistoryResponse,
  UseAsyncStatusOptions,
  UseAsyncStatusReturn,
} from "./useAsyncStatus.types";

/**
 * Hook for tracking async tool operations via Server-Sent Events.
 *
 * @param chatId - The chat ID to track operations for (null to disconnect)
 * @param options - Configuration options
 * @returns Object containing operations arrays and connection status
 */
export function useAsyncStatus(
  chatId: string | null,
  options: UseAsyncStatusOptions = {}
): UseAsyncStatusReturn {
  const { autoConnect = true } = options;

  const [operations, setOperations] = useState<Map<string, AsyncStatusUpdate>>(
    new Map()
  );
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Track if we've loaded initial history to prevent duplicate fetches
  const [historyLoaded, setHistoryLoaded] = useState(false);

  // Store chatId in ref for stable callbacks
  const chatIdRef = useRef(chatId);
  chatIdRef.current = chatId;

  // Handle incoming status update
  // Using ref pattern to keep callback stable
  const handleStatusUpdateRef = useRef<(update: AsyncStatusUpdate) => void>();
  handleStatusUpdateRef.current = (update: AsyncStatusUpdate) => {
    setOperations((prev) => {
      const next = new Map(prev);
      next.set(update.tool_call_id, update);
      return next;
    });
  };

  // Connect to SSE endpoint
  useEffect(() => {
    if (!chatId || !autoConnect) {
      setOperations(new Map());
      setIsConnected(false);
      setError(null);
      setHistoryLoaded(false);
      return;
    }

    const url = `${API_BASE}/chats/${chatId}/async-status`;
    const eventSource = new EventSource(url);

    eventSource.onopen = () => {
      setIsConnected(true);
      setError(null);

      if (!historyLoaded) {
        fetch(`${API_BASE}/chats/${chatId}/async-operations/history?limit=5&offset=0`)
          .then((response) => {
            if (!response.ok) {
              throw new Error("Failed to fetch history");
            }
            return response.json();
          })
          .then((raw: unknown) => {
            if (!isAsyncHistoryResponse(raw)) {
              console.warn("[useAsyncStatus] Unexpected history response shape");
              return;
            }
            const data = raw;
            setOperations((prev) => {
              const next = new Map(prev);
              for (const op of data.operations) {
                if (!next.has(op.tool_call_id)) {
                  next.set(op.tool_call_id, op);
                }
              }
              return next;
            });
            setHistoryLoaded(true);
          })
          .catch((err: unknown) => {
            console.warn("[useAsyncStatus] Failed to fetch initial history:", err);
            setHistoryLoaded(true);
          });
      }
    };

    eventSource.onerror = (e) => {
      console.error("[useAsyncStatus] SSE error:", e);
      setIsConnected(false);
      setError("Connection lost. Reconnecting...");
    };

    eventSource.addEventListener("status", (event) => {
      try {
        const raw: unknown = JSON.parse(event.data as string) as unknown;
        if (!isAsyncStatusUpdate(raw)) {
          console.warn("[useAsyncStatus] Unexpected status update shape");
          return;
        }
        handleStatusUpdateRef.current?.(raw);
      } catch (err) {
        console.error("[useAsyncStatus] Failed to parse status event:", err);
      }
    });

    return () => {
      eventSource.close();
      setIsConnected(false);
    };
  }, [chatId, autoConnect, historyLoaded]);

  const cancelOperation = useCallback(
    async (toolCallId: string) => {
      const currentChatId = chatIdRef.current;
      if (!currentChatId) return;

      try {
        const response = await fetch(
          `${API_BASE}/chats/${currentChatId}/async-operations/${toolCallId}/cancel`,
          { method: "POST" }
        );
        if (!response.ok) {
          const data = (await response.json()) as { error?: string };
          throw new Error(data.error || "Failed to cancel operation");
        }
        setOperations((prev) => {
          const next = new Map(prev);
          next.delete(toolCallId);
          return next;
        });
      } catch (err) {
        console.error("[useAsyncStatus] Failed to cancel operation:", err);
        throw err;
      }
    },
    []
  );

  const refreshOperation = useCallback(
    async (toolCallId: string): Promise<AsyncStatusUpdate> => {
      const currentChatId = chatIdRef.current;
      if (!currentChatId) {
        throw new Error("No chat selected");
      }

      try {
        const response = await fetch(
          `${API_BASE}/chats/${currentChatId}/async-operations/${toolCallId}/refresh`,
          { method: "POST" }
        );
        if (!response.ok) {
          const data = (await response.json()) as { error?: string };
          throw new Error(data.error || "Failed to refresh operation");
        }
        const raw: unknown = (await response.json()) as unknown;
        if (!isAsyncStatusUpdate(raw)) {
          throw new Error("Unexpected response shape from refresh");
        }
        setOperations((prev) => {
          const next = new Map(prev);
          next.set(raw.tool_call_id, raw);
          return next;
        });
        return raw;
      } catch (err) {
        console.error("[useAsyncStatus] Failed to refresh operation:", err);
        throw err;
      }
    },
    []
  );

  const fetchHistory = useCallback(
    async (limit = 20, offset = 0) => {
      const currentChatId = chatIdRef.current;
      if (!currentChatId) {
        return { operations: [], total: 0, hasMore: false };
      }

      try {
        const response = await fetch(
          `${API_BASE}/chats/${currentChatId}/async-operations/history?limit=${limit}&offset=${offset}`
        );
        if (!response.ok) {
          const data = (await response.json()) as { error?: string };
          throw new Error(data.error || "Failed to fetch history");
        }
        const raw: unknown = (await response.json()) as unknown;
        if (!isAsyncHistoryResponse(raw)) {
          throw new Error("Unexpected response shape from history");
        }
        return {
          operations: raw.operations,
          total: raw.total,
          hasMore: raw.has_more,
        };
      } catch (err) {
        console.error("[useAsyncStatus] Failed to fetch history:", err);
        throw err;
      }
    },
    []
  );

  const operationsArray = useMemo(
    () => operations.size === 0 ? EMPTY_OPERATIONS : Array.from(operations.values()),
    [operations]
  );

  const activeOperations = useMemo(
    () => operationsArray.filter((op) => !op.is_terminal),
    [operationsArray]
  );

  const completedOperations = useMemo(
    () => operationsArray.filter((op) => op.is_terminal),
    [operationsArray]
  );

  const activeCount = activeOperations.length;

  return useMemo(
    () => ({
      operations: operationsArray,
      activeOperations: activeOperations.length === 0 ? EMPTY_OPERATIONS : activeOperations,
      completedOperations: completedOperations.length === 0 ? EMPTY_OPERATIONS : completedOperations,
      isConnected,
      error,
      cancelOperation,
      refreshOperation,
      fetchHistory,
      activeCount,
    }),
    [operationsArray, activeOperations, completedOperations, isConnected, error, cancelOperation, refreshOperation, fetchHistory, activeCount]
  );
}
