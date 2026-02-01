/**
 * useTimeline Hook
 *
 * Manages unified timeline state for multi-tab recording sessions.
 * Fetches timeline entries (actions + page events) from the backend
 * and handles real-time WebSocket updates.
 */

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { getApiBase } from '@/config';
import type { Page } from './usePages';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isString = (value: unknown): value is string => typeof value === 'string';
const isNumber = (value: unknown): value is number => typeof value === 'number';

const safeJson = async (response: Response): Promise<unknown> => {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
};

/** Timeline entry types */
export type TimelineEntryType = 'action' | 'page_event';

/** Page event types */
export type PageEventType = 'page_created' | 'page_navigated' | 'page_closed';

/** Recorded action from timeline */
export interface TimelineAction {
  id: string;
  actionType: string;
  url?: string;
  sequenceNum: number;
  timestamp: string;
  selector?: { primary: string };
  payload?: Record<string, unknown>;
  confidence: number;
  pageTitle?: string;
}

/** Page event from timeline */
export interface TimelinePageEvent {
  id: string;
  type: PageEventType;
  pageId: string;
  url?: string;
  title?: string;
  openerId?: string;
  timestamp: string;
}

/** Unified timeline entry */
export interface TimelineEntry {
  id: string;
  type: TimelineEntryType;
  timestamp: string;
  pageId: string;
  action?: TimelineAction;
  pageEvent?: TimelinePageEvent;
}

/** API response for timeline */
interface TimelineResponse {
  entries: TimelineEntry[];
  hasMore: boolean;
  totalEntries: number;
}

const parseTimelineAction = (value: unknown): TimelineAction | undefined => {
  if (!isRecord(value)) return undefined;
  if (!isString(value.id)) return undefined;
  if (!isString(value.actionType)) return undefined;
  if (!isNumber(value.sequenceNum)) return undefined;
  if (!isString(value.timestamp)) return undefined;
  if (!isNumber(value.confidence)) return undefined;
  const selector = isRecord(value.selector) && isString(value.selector.primary)
    ? { primary: value.selector.primary }
    : undefined;
  return {
    id: value.id,
    actionType: value.actionType,
    sequenceNum: value.sequenceNum,
    timestamp: value.timestamp,
    confidence: value.confidence,
    url: isString(value.url) ? value.url : undefined,
    selector,
    payload: isRecord(value.payload) ? value.payload : undefined,
    pageTitle: isString(value.pageTitle) ? value.pageTitle : undefined,
  };
};

const parseTimelinePageEvent = (value: unknown): TimelinePageEvent | undefined => {
  if (!isRecord(value)) return undefined;
  if (!isString(value.id)) return undefined;
  if (!isString(value.type)) return undefined;
  if (!isString(value.pageId)) return undefined;
  if (!isString(value.timestamp)) return undefined;
  if (value.type !== 'page_created' && value.type !== 'page_navigated' && value.type !== 'page_closed') {
    return undefined;
  }
  return {
    id: value.id,
    type: value.type,
    pageId: value.pageId,
    timestamp: value.timestamp,
    url: isString(value.url) ? value.url : undefined,
    title: isString(value.title) ? value.title : undefined,
    openerId: isString(value.openerId) ? value.openerId : undefined,
  };
};

const parseTimelineEntry = (value: unknown): TimelineEntry | null => {
  if (!isRecord(value)) return null;
  if (!isString(value.id)) return null;
  if (!isString(value.type)) return null;
  if (!isString(value.timestamp)) return null;
  if (!isString(value.pageId)) return null;
  const action = parseTimelineAction(value.action);
  const pageEvent = parseTimelinePageEvent(value.pageEvent);
  return {
    id: value.id,
    type: value.type as TimelineEntryType,
    timestamp: value.timestamp,
    pageId: value.pageId,
    action,
    pageEvent,
  };
};

const parseTimelineResponse = (value: unknown): TimelineResponse | null => {
  if (!isRecord(value)) return null;
  const entries = Array.isArray(value.entries)
    ? value.entries.map(parseTimelineEntry).filter((entry): entry is TimelineEntry => entry !== null)
    : [];
  const hasMore = typeof value.hasMore === 'boolean' ? value.hasMore : false;
  const totalEntries = typeof value.totalEntries === 'number' ? value.totalEntries : entries.length;
  return { entries, hasMore, totalEntries };
};

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
  const apiUrl = getApiBase();
  const onEntryReceivedRef = useRef(onEntryReceived);
  onEntryReceivedRef.current = onEntryReceived;
  const subscribedSessionRef = useRef<string | null>(null);

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

    try {
      const params = new URLSearchParams({ limit: limit.toString() });
      if (filterPageId) {
        params.set('pageId', filterPageId);
      }

      const response = await fetch(`${apiUrl}/recordings/live/${sessionId}/timeline?${params}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch timeline: ${response.statusText}`);
      }

      const payload = await safeJson(response);
      const data = parseTimelineResponse(payload);
      if (!data) {
        throw new Error('Invalid timeline response');
      }
      setEntries(data.entries || []);
      setTotalEntries(data.totalEntries || 0);
      setHasMore(data.hasMore || false);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch timeline';
      setError(message);
      console.error('[useTimeline] Error fetching timeline:', err);
    } finally {
      setIsLoading(false);
    }
  }, [apiUrl, sessionId, filterPageId, limit]);

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
