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
  TimelinePageEvent,
} from '../api/schemas';
import type { Page } from './usePages';
import { useSessionStore } from '../stores';
import { fromJson, type JsonValue } from '@bufbuild/protobuf';
import { TimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import { timelineEntryToRecordedAction } from '../types/timeline-unified';

// Re-export types for backward compatibility
export type { TimelineEntry, TimelineAction, TimelinePageEvent } from '../api/schemas';
export type { PageEventType } from '../api/schemas';
export type TimelineEntryType = 'action' | 'page_event';

/** WebSocket message for page event */
interface PageEventMessage {
  type: 'page_event';
  session_id: string;
  event: TimelinePageEvent;
  timestamp: string;
}

type WebSocketTimelineMessage = PageEventMessage;

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
  sessionId: propSessionId,
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

  // Read session validation state from store
  // This ensures we only subscribe to WebSocket when session is validated
  const storeSessionId = useSessionStore((s) => s.sessionId);
  const isValidated = useSessionStore((s) => s.isValidated);

  // Use store session ID if validated, otherwise fall back to prop
  // This handles the case where prop is stale but store has the current session
  const sessionId = isValidated ? storeSessionId : propSessionId;

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

  // Fetch timeline when session changes and is validated
  useEffect(() => {
    if (sessionId && isValidated) {
      void refreshTimeline();
    } else {
      setEntries([]);
      setTotalEntries(0);
      setHasMore(false);
    }
  }, [sessionId, isValidated, refreshTimeline]);

  // Subscribe to recording session for real-time timeline updates
  // CRITICAL: Only subscribe when session is validated (confirmed to exist on server)
  // This prevents subscribing to stale session IDs from URL after driver restart
  //
  // NOTE: We use a separate effect for subscription management to avoid flapping.
  // The subscription should only change when:
  // 1. We connect/disconnect from WebSocket
  // 2. The actual validated session ID changes
  // 3. Component unmounts
  useEffect(() => {
    // Determine the target session to subscribe to
    // Only subscribe if we have a valid, validated session and are connected
    const targetSession = isConnected && sessionId && isValidated ? sessionId : null;
    const currentSub = subscribedSessionRef.current;

    console.log('[useTimeline] Subscription check:', {
      isConnected,
      sessionId,
      isValidated,
      targetSession,
      currentSub,
    });

    // Case 1: We need to unsubscribe (was subscribed, now shouldn't be)
    if (currentSub && currentSub !== targetSession) {
      console.log('[useTimeline] Unsubscribing from session:', currentSub);
      send({ type: 'unsubscribe_recording', session_id: currentSub });
      subscribedSessionRef.current = null;
    }

    // Case 2: We need to subscribe (target exists and we're not subscribed to it)
    if (targetSession && subscribedSessionRef.current !== targetSession) {
      console.log('[useTimeline] Subscribing to session:', targetSession);
      send({ type: 'subscribe_recording', session_id: targetSession });
      subscribedSessionRef.current = targetSession;
    }

    // Cleanup: only unsubscribe on unmount, not on every re-render
    // This prevents the subscribe/unsubscribe/subscribe flapping pattern
    return () => {
      // Only cleanup on actual unmount by checking if we still have a subscription
      // that matches what we set in this effect run
      if (subscribedSessionRef.current === targetSession && targetSession) {
        console.log('[useTimeline] Cleanup (unmount): unsubscribing from session:', targetSession);
        send({ type: 'unsubscribe_recording', session_id: targetSession });
        subscribedSessionRef.current = null;
      }
    };
  }, [isConnected, sessionId, isValidated, send]);

  // Handle WebSocket messages for real-time updates
  useEffect(() => {
    if (!lastMessage || !sessionId) return;

    const msg = lastMessage as unknown as WebSocketTimelineMessage | { type: string; session_id?: string; entry?: unknown };

    // Debug: Log all received messages
    console.log('[useTimeline] Received WebSocket message:', {
      type: msg.type,
      session_id: 'session_id' in msg ? (msg as { session_id?: string }).session_id : undefined,
      currentSessionId: sessionId,
      isValidated,
    });

    // Canonical V2 timeline stream. The entry is protobuf JSON, so decode it
    // before adapting it to this hook's presentation model.
    if (msg.type === 'TIMELINE_MESSAGE_TYPE_ENTRY' && msg.session_id === sessionId) {
      try {
        const raw = (msg as unknown as { entry?: unknown }).entry;
        if (!raw) return;
        const recorded = timelineEntryToRecordedAction(fromJson(TimelineEntrySchema, raw as JsonValue));
        if (!recorded) return;
        const entry: TimelineEntry = {
          id: recorded.id,
          type: 'action',
          timestamp: recorded.timestamp,
          pageId: '',
          action: {
            id: recorded.id,
            actionType: recorded.actionType,
            sequenceNum: recorded.sequenceNum,
            timestamp: recorded.timestamp,
            confidence: recorded.confidence,
            url: recorded.url,
            pageTitle: recorded.pageTitle,
            selector: recorded.selector,
            payload: recorded.payload,
          },
        };
        setEntries((prev) => prev.some((existing) => existing.id === entry.id) ? prev : [...prev, entry].sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()));
        setTotalEntries((prev) => prev + 1);
      } catch (error) {
        console.warn('[useTimeline] Invalid V2 timeline stream entry', error);
      }
      return;
    }
    // Handle page event
	const pageEventMsg = msg as PageEventMessage;
    if (pageEventMsg.type === 'page_event' && pageEventMsg.session_id === sessionId) {
		const event = pageEventMsg.event;

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
  }, [lastMessage, sessionId, isValidated]);

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
