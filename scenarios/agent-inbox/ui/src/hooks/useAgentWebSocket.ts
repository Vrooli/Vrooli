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

/**
 * Hook for real-time agent event streaming.
 *
 * Uses polling to fetch events from the agent-manager via the inbox API.
 * In the future, this could be upgraded to WebSocket for true real-time updates.
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

  // Fetch events since last sequence
  const fetchEvents = useCallback(async () => {
    if (!chatId || !enabled) return;

    try {
      const response = await getAgentEvents(chatId, lastSequenceRef.current);

      if (!isMountedRef.current) return;

      if (response.events && response.events.length > 0) {
        // Update last sequence
        const maxSequence = Math.max(...response.events.map(e => e.sequence));
        lastSequenceRef.current = maxSequence;

        // Add new events
        setEvents(prev => {
          const existingIds = new Set(prev.map(e => e.id));
          const newEvents = response.events.filter(e => !existingIds.has(e.id));

          // Call onEvent for each new event
          newEvents.forEach(event => {
            onEvent?.(event);
          });

          return [...prev, ...newEvents];
        });
      }

      setIsConnected(true);
      setError(null);
    } catch (e) {
      if (!isMountedRef.current) return;
      setError(e instanceof Error ? e.message : "Failed to fetch events");
      setIsConnected(false);
    }
  }, [chatId, enabled, onEvent]);

  // Fetch status
  const fetchStatus = useCallback(async () => {
    if (!chatId || !enabled) return;

    try {
      const newStatus = await getAgentStatus(chatId);

      if (!isMountedRef.current) return;

      setStatus(prev => {
        // Check if status changed
        if (prev?.status !== newStatus.status) {
          onStatusChange?.(newStatus);
        }
        return newStatus;
      });
    } catch (e) {
      // Status errors are less critical, just log
      console.error("Failed to fetch agent status:", e);
    }
  }, [chatId, enabled, onStatusChange]);

  // Combined refresh
  const refresh = useCallback(async () => {
    await Promise.all([fetchEvents(), fetchStatus()]);
  }, [fetchEvents, fetchStatus]);

  // Clear events
  const clearEvents = useCallback(() => {
    setEvents([]);
    lastSequenceRef.current = 0;
  }, []);

  // Set up polling
  useEffect(() => {
    isMountedRef.current = true;

    if (!chatId || !runId || !enabled) {
      setIsConnected(false);
      return;
    }

    // Reset state for new run
    setEvents([]);
    lastSequenceRef.current = 0;
    setError(null);

    // Initial fetch
    refresh();

    // Set up polling interval
    const intervalId = setInterval(refresh, pollInterval);

    return () => {
      isMountedRef.current = false;
      clearInterval(intervalId);
    };
  }, [chatId, runId, enabled, pollInterval, refresh]);

  // Stop polling when run completes
  useEffect(() => {
    if (status?.status && ["complete", "failed", "cancelled"].includes(status.status)) {
      // Do one final fetch to get any remaining events
      fetchEvents();
    }
  }, [status?.status, fetchEvents]);

  return {
    events,
    status,
    isConnected,
    error,
    refresh,
    clearEvents
  };
}

export default useAgentWebSocket;
