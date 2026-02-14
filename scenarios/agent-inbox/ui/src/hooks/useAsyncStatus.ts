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
import { resolveApiBase } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

function isAsyncStatusUpdate(v: unknown): v is AsyncStatusUpdate {
  return typeof v === 'object' && v !== null
    && typeof (v as Record<string, unknown>).tool_call_id === 'string'
    && typeof (v as Record<string, unknown>).is_terminal === 'boolean';
}

function isAsyncHistoryResponse(v: unknown): v is AsyncHistoryResponse {
  return typeof v === 'object' && v !== null
    && Array.isArray((v as Record<string, unknown>).operations);
}

/** Status update received via SSE */
export interface AsyncStatusUpdate {
  tool_call_id: string;
  chat_id: string;
  tool_name: string;
  status: string;
  progress?: number;
  message?: string;
  phase?: string;
  result?: unknown;
  error?: string;
  is_terminal: boolean;
  updated_at: string;
}

/** History response from the API */
export interface AsyncHistoryResponse {
  operations: AsyncStatusUpdate[];
  total: number;
  limit: number;
  offset: number;
  has_more: boolean;
}

/** Options for the useAsyncStatus hook */
export interface UseAsyncStatusOptions {
  /** Whether to auto-connect when chatId is provided (default: true) */
  autoConnect?: boolean;
}

/** Return type for the useAsyncStatus hook */
export interface UseAsyncStatusReturn {
  /** All operations (active + recently completed) */
  operations: AsyncStatusUpdate[];
  /** Only non-terminal operations */
  activeOperations: AsyncStatusUpdate[];
  /** Only terminal operations */
  completedOperations: AsyncStatusUpdate[];
  /** Whether SSE connection is established */
  isConnected: boolean;
  /** Connection error message, if any */
  error: string | null;
  /** Cancel a running operation */
  cancelOperation: (toolCallId: string) => Promise<void>;
  /** Force-refresh a specific operation (immediate status poll) */
  refreshOperation: (toolCallId: string) => Promise<AsyncStatusUpdate>;
  /** Fetch completed operation history from server */
  fetchHistory: (limit?: number, offset?: number) => Promise<{
    operations: AsyncStatusUpdate[];
    total: number;
    hasMore: boolean;
  }>;
  /** Number of non-terminal operations */
  activeCount: number;
}

// Stable empty array for when there are no operations
// CRITICAL: Using inline [] creates new array on every render, causing infinite re-render loops
const EMPTY_OPERATIONS: AsyncStatusUpdate[] = [];

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
    // NOTE: Completed operations are no longer auto-removed
    // They stay in state for the drawer/history view
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

      // Auto-fetch recent history on connect to populate completed operations
      // This ensures the status bar shows recent operations even after page refresh
      // or when all operations completed before SSE connection was established
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
            // Merge history into operations map
            // Don't overwrite active operations that came in via SSE
            setOperations((prev) => {
              const next = new Map(prev);
              for (const op of data.operations || []) {
                // Only add if we don't already have this operation
                // (SSE might have sent it already)
                if (!next.has(op.tool_call_id)) {
                  next.set(op.tool_call_id, op);
                }
              }
              return next;
            });
            setHistoryLoaded(true);
          })
          .catch((err) => {
            console.warn("[useAsyncStatus] Failed to fetch initial history:", err);
            // Don't set error state - this is a non-critical fetch
            // The SSE connection is still working
            setHistoryLoaded(true); // Mark as loaded to prevent retries
          });
      }
    };

    eventSource.onerror = (e) => {
      console.error("[useAsyncStatus] SSE error:", e);
      setIsConnected(false);
      setError("Connection lost. Reconnecting...");
    };

    // Handle 'status' events
    eventSource.addEventListener("status", (event) => {
      try {
        const raw = JSON.parse(event.data);
        if (!isAsyncStatusUpdate(raw)) {
          console.warn("[useAsyncStatus] Unexpected status update shape");
          return;
        }
        const update = raw;
        handleStatusUpdateRef.current?.(update);
      } catch (err) {
        console.error("[useAsyncStatus] Failed to parse status event:", err);
      }
    });

    // Cleanup on unmount or chatId change
    return () => {
      eventSource.close();
      setIsConnected(false);
    };
  }, [chatId, autoConnect, historyLoaded]);

  // Manually cancel an operation
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
          const data = await response.json();
          throw new Error(data.error || "Failed to cancel operation");
        }

        // Remove from local state immediately
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

  // Force-refresh a specific operation (immediate status poll)
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
          const data = await response.json();
          throw new Error(data.error || "Failed to refresh operation");
        }

        const raw = await response.json();
        if (!isAsyncStatusUpdate(raw)) {
          throw new Error("Unexpected response shape from refresh");
        }
        const update = raw;

        // Update local state with the refreshed data
        setOperations((prev) => {
          const next = new Map(prev);
          next.set(update.tool_call_id, update);
          return next;
        });

        return update;
      } catch (err) {
        console.error("[useAsyncStatus] Failed to refresh operation:", err);
        throw err;
      }
    },
    []
  );

  // Fetch completed operation history from server
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
          const data = await response.json();
          throw new Error(data.error || "Failed to fetch history");
        }

        const raw = await response.json();
        if (!isAsyncHistoryResponse(raw)) {
          throw new Error("Unexpected response shape from history");
        }
        const data = raw;
        return {
          operations: data.operations || [],
          total: data.total,
          hasMore: data.has_more,
        };
      } catch (err) {
        console.error("[useAsyncStatus] Failed to fetch history:", err);
        throw err;
      }
    },
    []
  );

  // Memoize the operations array to prevent creating new array references on every render
  // CRITICAL: When Map is empty, return stable EMPTY_OPERATIONS instead of new []
  const operationsArray = useMemo(
    () => operations.size === 0 ? EMPTY_OPERATIONS : Array.from(operations.values()),
    [operations]
  );

  // Separate active and completed operations
  const activeOperations = useMemo(
    () => operationsArray.filter((op) => !op.is_terminal),
    [operationsArray]
  );

  const completedOperations = useMemo(
    () => operationsArray.filter((op) => op.is_terminal),
    [operationsArray]
  );

  // Compute activeCount from active operations
  const activeCount = activeOperations.length;

  // CRITICAL: Memoize the return object to prevent creating new object references
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
