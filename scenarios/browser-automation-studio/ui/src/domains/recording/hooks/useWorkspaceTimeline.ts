/** Owns recording journal state and execution timeline state for the workspace. */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { fromJson, type JsonValue } from '@bufbuild/protobuf';
import { TimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import { useWebSocket, useWebSocketMessage, type WebSocketMessage } from '@/contexts/WebSocketContext';
import { recordingApi } from '../api';
import type { TimelineEntry, TimelinePageEvent } from '../api/schemas';
import { timelineEntryId, timelineEntryTimestamp } from '../api/schemas';
import type { Page } from './usePages';
import { useSessionStore } from '../stores';
import type { ExecutionStatus, ExecutionTimelineItem, TimelineItem, TimelineMode } from '../types/timeline-unified';
import { updateTimelineItemStatus, workflowNodesToTimelineItems } from '../types/timeline-unified';

export type { TimelineEntry, TimelineAction, TimelinePageEvent } from '../api/schemas';
export type { PageEventType } from '../api/schemas';
export type TimelineEntryType = 'action' | 'page_event';
export type PageColor = (typeof PAGE_COLORS)[number];

const PAGE_COLORS = [
  'bg-blue-500', 'bg-green-500', 'bg-purple-500', 'bg-orange-500',
  'bg-pink-500', 'bg-cyan-500', 'bg-yellow-500', 'bg-red-500', 'bg-gray-500',
] as const;

export interface WorkflowNode {
  id: string;
  type?: string;
  data?: Record<string, unknown>;
  action?: { type: string; metadata?: { label?: string }; navigate?: { url?: string } };
}

export interface WorkflowEdge {
  source: string;
  target: string;
}

export interface UseWorkspaceTimelineOptions {
  mode: TimelineMode;
  isRecording?: boolean;
  sessionId: string | null;
  pages: Page[];
  executionId?: string | null;
  workflowNodes?: WorkflowNode[];
  workflowEdges?: WorkflowEdge[];
  filterPageId?: string | null;
  limit?: number;
  onEntryReceived?: (entry: TimelineEntry) => void;
}

export interface UseWorkspaceTimelineReturn {
  mode: TimelineMode;
  entries: TimelineEntry[];
  executionItems: TimelineItem[];
  isLoading: boolean;
  isLive: boolean;
  isConnected: boolean;
  error: string | null;
  pageColorMap: Map<string, PageColor>;
  totalEntries: number;
  hasMore: boolean;
  entriesByPage: Map<string, number>;
  refreshTimeline: () => Promise<void>;
  clearEntries: () => void;
  clearExecutionItems: () => void;
  updateEntries: (update: (entries: TimelineEntry[]) => TimelineEntry[]) => void;
  getEntriesForPage: (pageId: string) => TimelineEntry[];
  subscribeToExecution: (executionId: string) => void;
  unsubscribeFromExecution: () => void;
  stats: { total: number; successful: number; failed: number; pending: number };
}

export function useWorkspaceTimeline({
  mode,
  isRecording = false,
  sessionId: propSessionId,
  pages,
  executionId,
  workflowNodes,
  workflowEdges,
  filterPageId = null,
  limit = 100,
  onEntryReceived,
}: UseWorkspaceTimelineOptions): UseWorkspaceTimelineReturn {
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [executionItems, setExecutionItems] = useState<TimelineItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [entryError, setEntryError] = useState<string | null>(null);
  const [totalEntries, setTotalEntries] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [isExecutionLive, setIsExecutionLive] = useState(false);

  const storeSessionId = useSessionStore((state) => state.sessionId);
  const isValidated = useSessionStore((state) => state.isValidated);
  const activeSessionId = isValidated ? storeSessionId : propSessionId;
  const { send, isConnected } = useWebSocket();
  const onEntryReceivedRef = useRef(onEntryReceived);
  onEntryReceivedRef.current = onEntryReceived;
  const subscribedSessionRef = useRef<string | null>(null);
  const subscribedExecutionRef = useRef<string | null>(null);
  const prePopulatedWorkflowRef = useRef<string | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  useEffect(() => () => abortControllerRef.current?.abort(), []);
  useEffect(() => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = new AbortController();
  }, [activeSessionId]);

  const pageColorMap = useMemo(() => {
    const colors = new Map<string, PageColor>();
    pages.forEach((page, index) => colors.set(page.id, PAGE_COLORS[index % PAGE_COLORS.length] ?? PAGE_COLORS[0]));
    return colors;
  }, [pages]);

  const refreshTimeline = useCallback(async () => {
    if (!activeSessionId) {
      setEntries([]);
      setTotalEntries(0);
      setHasMore(false);
      return;
    }
    setIsLoading(true);
    setEntryError(null);
    const result = await recordingApi.getTimeline(
      activeSessionId,
      { limit, pageId: filterPageId ?? undefined },
      { signal: abortControllerRef.current?.signal },
    );
    setIsLoading(false);
    if (!result.success) {
      if (result.error !== 'Request cancelled') {
        setEntryError(result.error);
        console.error('[useWorkspaceTimeline] Error fetching timeline:', result.error);
      }
      return;
    }
    setEntries(result.data.entries);
    setTotalEntries(result.data.totalEntries);
    setHasMore(result.data.hasMore);
  }, [activeSessionId, filterPageId, limit]);

  useEffect(() => {
    if (activeSessionId && isValidated) void refreshTimeline();
    else {
      setEntries([]);
      setTotalEntries(0);
      setHasMore(false);
    }
  }, [activeSessionId, isValidated, refreshTimeline]);

  useEffect(() => {
    const target = isConnected && activeSessionId && isValidated ? activeSessionId : null;
    const current = subscribedSessionRef.current;
    if (current && current !== target) {
      send({ type: 'unsubscribe_recording', session_id: current });
      subscribedSessionRef.current = null;
    }
    if (target && subscribedSessionRef.current !== target) {
      send({ type: 'subscribe_recording', session_id: target, frames: false });
      subscribedSessionRef.current = target;
    }
    return () => {
      if (target && subscribedSessionRef.current === target) {
        send({ type: 'unsubscribe_recording', session_id: target });
        subscribedSessionRef.current = null;
      }
    };
  }, [isConnected, activeSessionId, isValidated, send]);

  useEffect(() => {
    if (mode === 'recording' && executionItems.length) {
      setExecutionItems([]);
      prePopulatedWorkflowRef.current = null;
    }
  }, [mode]);

  useEffect(() => {
    if (mode !== 'execution' || !workflowNodes?.length || !workflowEdges) return;
    const workflowKey = workflowNodes.map((node) => node.id).join(',');
    if (prePopulatedWorkflowRef.current === workflowKey) return;
    setExecutionItems(workflowNodesToTimelineItems(workflowNodes, workflowEdges));
    prePopulatedWorkflowRef.current = workflowKey;
  }, [mode, workflowNodes, workflowEdges]);

  useWebSocketMessage((lastMessage) => {
    if (activeSessionId) {
      const recordingMessage = lastMessage as unknown as {
        type: string;
        session_id?: string;
        entry?: unknown;
        event?: TimelinePageEvent;
      };
      if (recordingMessage.type === 'TIMELINE_MESSAGE_TYPE_ENTRY' && recordingMessage.session_id === activeSessionId) {
        try {
          if (!recordingMessage.entry) return;
          const protoEntry = fromJson(TimelineEntrySchema, recordingMessage.entry as JsonValue);
          if (!protoEntry.action) return;
          const entry: TimelineEntry = { type: 'action', entry: protoEntry, pageId: '' };
          setEntries((current) => current.some((existing) => timelineEntryId(existing) === timelineEntryId(entry))
            ? current
            : [...current, entry].sort((a, b) => new Date(timelineEntryTimestamp(a)).getTime() - new Date(timelineEntryTimestamp(b)).getTime()));
          setTotalEntries((count) => count + 1);
        } catch (error) {
          console.warn('[useWorkspaceTimeline] Invalid V2 timeline stream entry', error);
        }
        return;
      }
      if (recordingMessage.type === 'page_event' && recordingMessage.session_id === activeSessionId) {
        const event = recordingMessage.event;
        if (!event) return;
        const entry: TimelineEntry = { type: 'page_event', pageId: event.pageId, pageEvent: event };
        setEntries((current) => {
          if (current.some((existing) => timelineEntryId(existing) === timelineEntryId(entry))) return current;
          onEntryReceivedRef.current?.(entry);
          return [...current, entry].sort((a, b) => new Date(timelineEntryTimestamp(a)).getTime() - new Date(timelineEntryTimestamp(b)).getTime());
        });
        setTotalEntries((count) => count + 1);
      }
    }

    const message = lastMessage as WebSocketMessage & {
      entry?: unknown;
      node_id?: string;
      status?: string;
    };
    if (mode === 'execution' && (message.type === 'step' || message.type === 'TIMELINE_MESSAGE_TYPE_ENTRY')) {
      const data = (message as unknown as Record<string, unknown>).entry as Record<string, unknown> | undefined;
      if (data) {
        const context = data.context as Record<string, unknown> | undefined;
        const stepIndex = data.step_index as number | undefined;
        const nodeId = (context?.node_id as string) ?? (data.node_id as string);
        const success = context?.success as boolean | undefined;
        const error = context?.error as string | undefined;
        const status: ExecutionStatus = success === true ? 'completed' : success === false || error ? 'failed' : 'running';
        setExecutionItems((current) => {
          let index = nodeId ? current.findIndex((item) => (item as ExecutionTimelineItem).nodeId === nodeId) : -1;
          if (index < 0 && stepIndex !== undefined && stepIndex >= 0 && stepIndex < current.length) {
            const item = current[stepIndex] as ExecutionTimelineItem;
            if (item.nodeId && !item.nodeId.includes('-step-')) index = stepIndex;
          }
          if (index < 0) return current;
          const targetNodeId = (current[index] as ExecutionTimelineItem).nodeId;
          return updateTimelineItemStatus(
            current as ExecutionTimelineItem[], targetNodeId, status, error,
            data.duration_ms as number | undefined,
          );
        });
      }
    }
    if (mode === 'execution' && message.type === 'execution_started') {
      setIsExecutionLive(true);
      if (prePopulatedWorkflowRef.current) {
        setExecutionItems((current) => current.map((item) => ({
          ...item, success: undefined, error: undefined, executionStatus: 'pending' as ExecutionStatus,
        } as ExecutionTimelineItem)));
      } else setExecutionItems([]);
    }
    if (mode === 'execution' && (message.type === 'execution_completed' || message.type === 'execution_failed')) {
      setIsExecutionLive(false);
    }
  });

  const subscribeToExecution = useCallback((id: string) => {
    if (subscribedExecutionRef.current === id) return;
    if (subscribedExecutionRef.current) {
      send({ type: 'unsubscribe_execution', execution_id: subscribedExecutionRef.current });
    }
    send({ type: 'subscribe_execution', execution_id: id });
    subscribedExecutionRef.current = id;
    setIsExecutionLive(true);
    if (!prePopulatedWorkflowRef.current) setExecutionItems([]);
  }, [send]);

  const unsubscribeFromExecution = useCallback(() => {
    if (!subscribedExecutionRef.current) return;
    send({ type: 'unsubscribe_execution', execution_id: subscribedExecutionRef.current });
    subscribedExecutionRef.current = null;
    setIsExecutionLive(false);
  }, [send]);

  useEffect(() => {
    if (mode === 'execution' && executionId && isConnected) subscribeToExecution(executionId);
    return () => {
      if (mode === 'execution') unsubscribeFromExecution();
    };
  }, [mode, executionId, isConnected, subscribeToExecution, unsubscribeFromExecution]);

  const filteredEntries = useMemo(
    () => filterPageId ? entries.filter((entry) => entry.pageId === filterPageId) : entries,
    [entries, filterPageId],
  );
  const getEntriesForPage = useCallback((pageId: string) => entries.filter((entry) => entry.pageId === pageId), [entries]);
  const entriesByPage = useMemo(() => {
    const counts = new Map<string, number>();
    entries.forEach((entry) => counts.set(entry.pageId, (counts.get(entry.pageId) ?? 0) + 1));
    return counts;
  }, [entries]);
  const clearEntries = useCallback(() => {
    setEntries([]);
    setTotalEntries(0);
    setHasMore(false);
  }, []);
  const clearExecutionItems = useCallback(() => setExecutionItems([]), []);
  const updateEntries = useCallback((update: (current: TimelineEntry[]) => TimelineEntry[]) => {
    setEntries((current) => {
      const updated = update(current);
      setTotalEntries(updated.length);
      return updated;
    });
  }, []);
  const stats = useMemo(() => ({
    total: executionItems.length,
    successful: executionItems.filter((item) => item.success === true).length,
    failed: executionItems.filter((item) => item.success === false).length,
    pending: executionItems.filter((item) => item.success === undefined).length,
  }), [executionItems]);

  return {
    mode,
    entries: filteredEntries,
    executionItems,
    isLoading,
    isLive: mode === 'execution' ? isExecutionLive : isRecording,
    isConnected,
    error: entryError,
    pageColorMap,
    totalEntries,
    hasMore,
    entriesByPage,
    refreshTimeline,
    clearEntries,
    clearExecutionItems,
    updateEntries,
    getEntriesForPage,
    subscribeToExecution,
    unsubscribeFromExecution,
    stats,
  };
}
