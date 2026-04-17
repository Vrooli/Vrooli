import { useState, useEffect, useCallback, useRef } from "react";
import type { AgentEvent, AgentModeStatus } from "../lib/api";
import { getAgentEvents, getAgentStatus } from "../lib/api";

interface UseAgentWebSocketOptions {
  /** Chat ID to monitor */
  chatId: string | null;
  /** Run ID to subscribe to */
  runId: string | null;
  /** Whether the connection should be active */
  enabled?: boolean;
  /** Callback when new events arrive */
  onEvent?: (event: AgentEvent) => void;
  /** Callback when status changes */
  onStatusChange?: (status: AgentModeStatus) => void;
  /** Polling interval in ms (default: 2000) */
  pollInterval?: number;
}

interface UseAgentWebSocketResult {
  /** All events received so far */
  events: AgentEvent[];
  /** Current run status */
  status: AgentModeStatus | null;
  /** Whether currently connected/polling */
  isConnected: boolean;
  /** Any error that occurred */
  error: string | null;
  /** Manually refresh events */
  refresh: () => Promise<void>;
  /** Clear events */
  clearEvents: () => void;
}

const TERMINAL_STATUSES = ["complete", "failed", "cancelled"];

/**
 * Hook for real-time agent event streaming.
 *
 * Uses polling to fetch events from the agent-manager via the inbox API.
 * In the future, this could be upgraded to WebSocket for true real-time updates.
 *
 * IMPORTANT: onEvent and onStatusChange are stored in refs to prevent
 * dependency instability. Without refs, inline callbacks from the parent
 * component create new references each render, cascading through
 * fetchEvents → fetchStatus → refresh → polling useEffect, causing the
 * effect to reset events and re-fetch on every render (infinite loop).
 */
export function useAgentWebSocket({
  chatId,
  runId,
  enabled = true,
  onEvent,
  onStatusChange,
  pollInterval = 2000
}: UseAgentWebSocketOptions): UseAgentWebSocketResult {
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [status, setStatus] = useState<AgentModeStatus | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Track the last sequence number we've seen
  const lastSequenceRef = useRef<number>(0);
  const isMountedRef = useRef(true);

  // Track whether the run has reached a terminal state
  const terminalRef = useRef(false);

  // Store callbacks in refs to keep useCallback deps stable.
  // This prevents the infinite re-render loop described above.
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;
  const onStatusChangeRef = useRef(onStatusChange);
  onStatusChangeRef.current = onStatusChange;

  // Fetch events since last sequence
  const fetchEvents = useCallback(async (signal?: AbortSignal) => {
    if (!chatId || !enabled) return;

    try {
      const response = await getAgentEvents(chatId, lastSequenceRef.current, signal);

      if (!isMountedRef.current) return;

      if (response.events.length > 0) {
        // Update last sequence
        const maxSequence = Math.max(...response.events.map(e => e.sequence));
        lastSequenceRef.current = maxSequence;

        // Add new events (server-side after_sequence dedup ensures no duplicates)
        setEvents(prev => {
          response.events.forEach(event => {
            onEventRef.current?.(event);
          });
          return [...prev, ...response.events];
        });
      }

      setIsConnected(true);
      setError(null);
    } catch (e) {
      if (e instanceof DOMException && e.name === "AbortError") return;
      if (!isMountedRef.current) return;
      setError(e instanceof Error ? e.message : "Failed to fetch events");
      setIsConnected(false);
    }
  }, [chatId, enabled]);

  // Fetch status
  const fetchStatus = useCallback(async (signal?: AbortSignal) => {
    if (!chatId || !enabled) return;

    try {
      const newStatus = await getAgentStatus(chatId, signal);

      if (!isMountedRef.current) return;

      setStatus(prev => {
        // Check if status changed
        if (prev?.status !== newStatus.status) {
          onStatusChangeRef.current?.(newStatus);
        }
        // Detect terminal status and do one final event fetch
        if (newStatus.status && TERMINAL_STATUSES.includes(newStatus.status) && !terminalRef.current) {
          terminalRef.current = true;
          // Final fetch to pick up any remaining events
          void fetchEvents(signal);
        }
        return newStatus;
      });
    } catch (e) {
      if (e instanceof DOMException && e.name === "AbortError") return;
      // Status errors are less critical, just log
      console.error("Failed to fetch agent status:", e);
    }
  }, [chatId, enabled, fetchEvents]);

  // Combined refresh
  const refresh = useCallback(async (signal?: AbortSignal) => {
    await Promise.all([fetchEvents(signal), fetchStatus(signal)]);
  }, [fetchEvents, fetchStatus]);

  // Expose a stable refresh without signal for external callers
  const publicRefresh = useCallback(async () => {
    await refresh();
  }, [refresh]);

  // Clear events
  const clearEvents = useCallback(() => {
    setEvents([]);
    lastSequenceRef.current = 0;
  }, []);

  // Set up polling with self-scheduling setTimeout
  useEffect(() => {
    isMountedRef.current = true;
    terminalRef.current = false;

    if (!chatId || !runId || !enabled) {
      setIsConnected(false);
      return;
    }

    // Reset state for new run
    setEvents([]);
    lastSequenceRef.current = 0;
    setError(null);

    const abortController = new AbortController();
    const { signal } = abortController;
    let timeoutId: ReturnType<typeof setTimeout>;
    let cancelled = false;

    const poll = async () => {
      await refresh(signal);
      // Only schedule next poll if not cancelled and run hasn't terminated
      if (!cancelled && !terminalRef.current) {
        timeoutId = setTimeout(poll, pollInterval);
      }
    };

    // Initial fetch + start loop
    void poll();

    return () => {
      cancelled = true;
      clearTimeout(timeoutId);
      abortController.abort();
      isMountedRef.current = false;
    };
  }, [chatId, runId, enabled, pollInterval, refresh]);

  return {
    events,
    status,
    isConnected,
    error,
    refresh: publicRefresh,
    clearEvents
  };
}

export default useAgentWebSocket;
