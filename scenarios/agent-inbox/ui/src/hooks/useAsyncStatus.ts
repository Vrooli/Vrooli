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
 * - Maintains a map of active operations with their current status
 * - Auto-cleans completed operations after a delay
 *
 * USAGE:
 * ```tsx
 * const { operations, isConnected } = useAsyncStatus(chatId);
 * // operations is an array of AsyncStatusUpdate
 * ```
 */
import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { resolveApiBase } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

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

/** Options for the useAsyncStatus hook */
export interface UseAsyncStatusOptions {
  /** Delay in ms before removing completed operations from display (default: 3000) */
  completedRemovalDelay?: number;
  /** Whether to auto-connect when chatId is provided (default: true) */
  autoConnect?: boolean;
}

// Stable empty array for when there are no operations
// CRITICAL: Using inline [] creates new array on every render, causing infinite re-render loops
const EMPTY_OPERATIONS: AsyncStatusUpdate[] = [];

/**
 * Hook for tracking async tool operations via Server-Sent Events.
 *
 * @param chatId - The chat ID to track operations for (null to disconnect)
 * @param options - Configuration options
 * @returns Object containing operations array and connection status
 */
export function useAsyncStatus(
  chatId: string | null,
  options: UseAsyncStatusOptions = {}
) {
  const { completedRemovalDelay = 3000, autoConnect = true } = options;

  const [operations, setOperations] = useState<Map<string, AsyncStatusUpdate>>(
    new Map()
  );
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Use ref to track cleanup timers
  const cleanupTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  // Use ref for the removal delay to avoid re-creating callbacks
  const completedRemovalDelayRef = useRef(completedRemovalDelay);
  completedRemovalDelayRef.current = completedRemovalDelay;

  // Schedule removal of a completed operation
  // Using ref pattern to avoid dependency issues
  const scheduleRemoval = useCallback((toolCallId: string) => {
    // Clear any existing timer for this operation
    const existingTimer = cleanupTimers.current.get(toolCallId);
    if (existingTimer) {
      clearTimeout(existingTimer);
    }

    // Schedule new removal
    const timer = setTimeout(() => {
      setOperations((prev) => {
        const next = new Map(prev);
        next.delete(toolCallId);
        return next;
      });
      cleanupTimers.current.delete(toolCallId);
    }, completedRemovalDelayRef.current);

    cleanupTimers.current.set(toolCallId, timer);
  }, []); // Empty deps - uses ref for delay

  // Handle incoming status update
  // Using ref pattern to keep callback stable
  const handleStatusUpdateRef = useRef<(update: AsyncStatusUpdate) => void>();
  handleStatusUpdateRef.current = (update: AsyncStatusUpdate) => {
    setOperations((prev) => {
      const next = new Map(prev);
      next.set(update.tool_call_id, update);
      return next;
    });

    // Schedule removal for terminal operations
    if (update.is_terminal) {
      scheduleRemoval(update.tool_call_id);
    }
  };

  // Connect to SSE endpoint
  useEffect(() => {
    if (!chatId || !autoConnect) {
      setOperations(new Map());
      setIsConnected(false);
      setError(null);
      return;
    }

    const url = `${API_BASE}/chats/${chatId}/async-status`;
    const eventSource = new EventSource(url);

    eventSource.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    eventSource.onerror = (e) => {
      console.error("[useAsyncStatus] SSE error:", e);
      setIsConnected(false);
      setError("Connection lost. Reconnecting...");
    };

    // Handle 'connected' event
    eventSource.addEventListener("connected", () => {
      setIsConnected(true);
      setError(null);
    });

    // Handle 'status' events
    eventSource.addEventListener("status", (event) => {
      try {
        const update = JSON.parse(event.data) as AsyncStatusUpdate;
        handleStatusUpdateRef.current?.(update);
      } catch (err) {
        console.error("[useAsyncStatus] Failed to parse status event:", err);
      }
    });

    // Cleanup on unmount or chatId change
    return () => {
      eventSource.close();
      setIsConnected(false);

      // Clear all cleanup timers
      cleanupTimers.current.forEach((timer) => clearTimeout(timer));
      cleanupTimers.current.clear();
    };
  }, [chatId, autoConnect]); // Removed handleStatusUpdate from deps - using ref instead

  // Manually cancel an operation
  const cancelOperation = useCallback(
    async (toolCallId: string) => {
      if (!chatId) return;

      try {
        const response = await fetch(
          `${API_BASE}/chats/${chatId}/async-operations/${toolCallId}/cancel`,
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
    [chatId]
  );

  // Memoize the operations array to prevent creating new array references on every render
  // This is critical to prevent infinite re-render loops in consuming components
  // CRITICAL: When Map is empty, return stable EMPTY_OPERATIONS instead of new []
  const operationsArray = useMemo(
    () => operations.size === 0 ? EMPTY_OPERATIONS : Array.from(operations.values()),
    [operations]
  );

  return {
    /** Array of active async operations */
    operations: operationsArray,
    /** Whether SSE connection is established */
    isConnected,
    /** Connection error message, if any */
    error,
    /** Cancel a running operation */
    cancelOperation,
    /** Number of active operations */
    activeCount: operations.size,
  };
}
