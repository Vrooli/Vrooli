/**
 * useTimeline Hook
 *
 * Manages unified timeline state for multi-tab recording sessions.
 * Fetches timeline entries (actions + page events) from the backend
 * and handles real-time WebSocket updates.
 */

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { recordingApi } from '../api';
import type {
  TimelineEntry,
  TimelineAction,
  TimelinePageEvent,
} from '../api/schemas';
import type { Page } from './usePages';

// Re-export types for backward compatibility
export type { TimelineEntry, TimelineAction, TimelinePageEvent } from '../api/schemas';
export type { PageEventType } from '../api/schemas';
export type TimelineEntryType = 'action' | 'page_event';

/** WebSocket message for recording action */
interface RecordingActionMessage {
  type: 'recording_action';
  session_id: string;
  entry: {
    id: string;
    type: 'action';
    timestamp: string;
    pageId: string;
    action: TimelineAction;
  };
  timestamp: string;
}

/** WebSocket message for page event */
interface PageEventMessage {
  type: 'page_event';
  session_id: string;
  event: TimelinePageEvent;
  timestamp: string;
}

type WebSocketTimelineMessage = RecordingActionMessage | PageEventMessage;

/** Page color palette for visual distinction */
const PAGE_COLORS = [
  'bg-blue-500',
  'bg-green-500',
  'bg-purple-500',
  'bg-orange-500',
  'bg-pink-500',
  'bg-cyan-500',
  'bg-yellow-500',
  'bg-red-500',
  'bg-gray-500',
] as const;

export type PageColor = typeof PAGE_COLORS[number];

interface UseTimelineOptions {
  /** Session ID to track timeline for */
  sessionId: string | null;
  /** Pages in the session (for color assignment) */
  pages: Page[];
  /** Page ID to filter by (null for all pages) */
  filterPageId?: string | null;
  /** Maximum entries to fetch */
  limit?: number;
  /** Callback when new entries are received */
  onEntryReceived?: (entry: TimelineEntry) => void;
}

interface UseTimelineReturn {
  /** All timeline entries (filtered if filterPageId is set) */
  entries: TimelineEntry[];
  /** Whether entries are being loaded */
  isLoading: boolean;
  /** Error message if any */
  error: string | null;
  /** Map of page ID to color class */
  pageColorMap: Map<string, PageColor>;
  /** Refresh timeline from server */
  refreshTimeline: () => Promise<void>;
  /** Clear all entries */
  clearEntries: () => void;
  /** Total number of entries (unfiltered) */
  totalEntries: number;
  /** Whether there are more entries to load */
  hasMore: boolean;
  /** Get entries for a specific page */
  getEntriesForPage: (pageId: string) => TimelineEntry[];
  /** Count of entries by page */
  entriesByPage: Map<string, number>;
}

export function useTimeline({
  sessionId,
  pages,
  filterPageId = null,
  limit = 100,
  onEntryReceived,
}: UseTimelineOptions): UseTimelineReturn {
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [totalEntries, setTotalEntries] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  const { lastMessage, send, isConnected } = useWebSocket();
  const onEntryReceivedRef = useRef(onEntryReceived);
  onEntryReceivedRef.current = onEntryReceived;
  const subscribedSessionRef = useRef<string | null>(null);

  // AbortController for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null);

  // Clean up on unmount
  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
    };
  }, []);

  // Reset abort controller when session changes
  useEffect(() => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = new AbortController();
  }, [sessionId]);

  // Assign colors to pages based on creation order
  const pageColorMap = useMemo(() => {
    const defaultColor = PAGE_COLORS[0] ?? 'bg-gray-500';
    const map = new Map<string, PageColor>();
    pages.forEach((page, index) => {
      const color = PAGE_COLORS[index % PAGE_COLORS.length];
      map.set(page.id, color ?? defaultColor);
    });
    return map;
  }, [pages]);

  // Fetch timeline from API
  const refreshTimeline = useCallback(async () => {
    if (!sessionId) {
      setEntries([]);
      setTotalEntries(0);
      setHasMore(false);
      return;
    }

    setIsLoading(true);
    setError(null);

    const result = await recordingApi.getTimeline(
      sessionId,
      { limit, pageId: filterPageId ?? undefined },
      { signal: abortControllerRef.current?.signal }
    );

    setIsLoading(false);

    if (!result.success) {
      // Don't set error for aborted requests
      if (result.error !== 'Request cancelled') {
        setError(result.error);
        console.error('[useTimeline] Error fetching timeline:', result.error);
      }
      return;
    }

    setEntries(result.data.entries);
    setTotalEntries(result.data.totalEntries);
    setHasMore(result.data.hasMore);
  }, [sessionId, filterPageId, limit]);

  // Fetch timeline when session changes
  useEffect(() => {
    if (sessionId) {
      void refreshTimeline();
    } else {
      setEntries([]);
      setTotalEntries(0);
      setHasMore(false);
    }
  }, [sessionId, refreshTimeline]);

  // Subscribe to recording session for real-time timeline updates
  useEffect(() => {
    if (!isConnected || !sessionId) {
      return;
    }

    // Unsubscribe from previous session if different
    if (subscribedSessionRef.current && subscribedSessionRef.current !== sessionId) {
      send({ type: 'unsubscribe_recording', session_id: subscribedSessionRef.current });
      subscribedSessionRef.current = null;
    }

    // Subscribe to new session
    if (subscribedSessionRef.current !== sessionId) {
      send({ type: 'subscribe_recording', session_id: sessionId });
      subscribedSessionRef.current = sessionId;
    }

    // Cleanup: unsubscribe when unmounting or sessionId changes
    return () => {
      if (subscribedSessionRef.current) {
        send({ type: 'unsubscribe_recording', session_id: subscribedSessionRef.current });
        subscribedSessionRef.current = null;
      }
    };
  }, [isConnected, sessionId, send]);

  // Handle WebSocket messages for real-time updates
  useEffect(() => {
    if (!lastMessage || !sessionId) return;

    const msg = lastMessage as unknown as WebSocketTimelineMessage;

    // Handle recording action
    if (msg.type === 'recording_action' && msg.session_id === sessionId) {
      const entry: TimelineEntry = {
        id: msg.entry.id,
        type: 'action',
        timestamp: msg.entry.timestamp,
        pageId: msg.entry.pageId,
        action: msg.entry.action,
      };

      setEntries((prev) => {
        // Check if entry already exists
        if (prev.some((e) => e.id === entry.id)) {
          return prev;
        }

        const updated = [...prev, entry];
        // Sort by timestamp
        updated.sort((a, b) =>
          new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
        );

        if (onEntryReceivedRef.current) {
          onEntryReceivedRef.current(entry);
        }

        return updated;
      });

      setTotalEntries((prev) => prev + 1);
    }

    // Handle page event
    if (msg.type === 'page_event' && msg.session_id === sessionId) {
      const event = msg.event;

      // Create timeline entry from page event
      const entry: TimelineEntry = {
        id: event.id,
        type: 'page_event',
        timestamp: event.timestamp,
        pageId: event.pageId,
        pageEvent: event,
      };

      setEntries((prev) => {
        // Check if entry already exists
        if (prev.some((e) => e.id === entry.id)) {
          return prev;
        }

        const updated = [...prev, entry];
        // Sort by timestamp
        updated.sort((a, b) =>
          new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
        );

        if (onEntryReceivedRef.current) {
          onEntryReceivedRef.current(entry);
        }

        return updated;
      });

      setTotalEntries((prev) => prev + 1);
    }
  }, [lastMessage, sessionId]);

  // Filter entries by page if filter is set
  const filteredEntries = useMemo(() => {
    if (!filterPageId) return entries;
    return entries.filter((e) => e.pageId === filterPageId);
  }, [entries, filterPageId]);

  // Get entries for a specific page
  const getEntriesForPage = useCallback((pageId: string) => {
    return entries.filter((e) => e.pageId === pageId);
  }, [entries]);

  // Count entries by page
  const entriesByPage = useMemo(() => {
    const counts = new Map<string, number>();
    entries.forEach((entry) => {
      const count = counts.get(entry.pageId) || 0;
      counts.set(entry.pageId, count + 1);
    });
    return counts;
  }, [entries]);

  // Clear all entries
  const clearEntries = useCallback(() => {
    setEntries([]);
    setTotalEntries(0);
    setHasMore(false);
  }, []);

  return {
    entries: filteredEntries,
    isLoading,
    error,
    pageColorMap,
    refreshTimeline,
    clearEntries,
    totalEntries,
    hasMore,
    getEntriesForPage,
    entriesByPage,
  };
}
