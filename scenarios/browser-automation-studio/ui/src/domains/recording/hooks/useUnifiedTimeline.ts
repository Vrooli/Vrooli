/**
 * useUnifiedTimeline Hook
 *
 * Provides a unified interface for timeline data from both:
 * - Recording mode: Live captured actions from browser recording
 * - Execution mode: Workflow execution events streamed in real-time
 *
 * This hook enables the Record page to display both recording and execution
 * with the same UI components.
 */

import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { useWebSocket, type WebSocketMessage } from '@/contexts/WebSocketContext';
import type { RecordedAction } from '../types/types';
import type { TimelineItem, TimelineMode, ExecutionTimelineItem, ExecutionStatus, UseTimelineEntry } from '../types/timeline-unified';
import {
  recordedActionToTimelineItem,
  timelineEntryToRecordedAction,
  workflowNodesToTimelineItems,
  updateTimelineItemStatus,
  useTimelineEntryToTimelineItem,
} from '../types/timeline-unified';
import { mergeConsecutiveActions } from '../utils/mergeActions';

/** Minimal node type for workflow conversion */
export interface WorkflowNode {
  id: string;
  type?: string;
  data?: Record<string, unknown>;
  action?: { type: string; metadata?: { label?: string }; navigate?: { url?: string } };
}

/** Minimal edge type for workflow conversion */
export interface WorkflowEdge {
  source: string;
  target: string;
}

export interface UseUnifiedTimelineOptions {
  /** Current mode: 'recording' for live recording, 'execution' for workflow playback */
  mode: TimelineMode;
  /** Execution ID for execution mode */
  executionId?: string | null;
  /** Optional: Pre-existing recorded actions (for recording mode) - legacy fallback */
  initialActions?: RecordedAction[];
  /** Optional: Timeline entries from useTimeline hook (preferred for recording mode) */
  initialTimelineEntries?: UseTimelineEntry[];
  /** Optional: Workflow nodes for pre-populating timeline in execution mode */
  workflowNodes?: WorkflowNode[];
  /** Optional: Workflow edges for pre-populating timeline in execution mode */
  workflowEdges?: WorkflowEdge[];
  /** Callback when new items arrive */
  onItemsReceived?: (items: TimelineItem[]) => void;
}

export interface UseUnifiedTimelineReturn {
  /** Unified timeline items for rendering */
  items: TimelineItem[];
  /** Current mode */
  mode: TimelineMode;
  /** Whether the timeline is actively receiving updates */
  isLive: boolean;
  /** Whether we're connected to WebSocket */
  isConnected: boolean;
  /** Error message if any */
  error: string | null;
  /** Clear all items */
  clearItems: () => void;
  /** Convert items back to RecordedActions for legacy compatibility */
  getRecordedActions: () => RecordedAction[];
  /** Subscribe to execution updates */
  subscribeToExecution: (executionId: string) => void;
  /** Unsubscribe from execution updates */
  unsubscribeFromExecution: () => void;
  /** Count of items by status */
  stats: {
    total: number;
    successful: number;
    failed: number;
    pending: number;
  };
}

/**
 * Unified timeline hook that works with both recording and execution modes.
 */
export function useUnifiedTimeline({
  mode,
  executionId,
  initialActions = [],
  initialTimelineEntries,
  workflowNodes,
  workflowEdges,
  onItemsReceived,
}: UseUnifiedTimelineOptions): UseUnifiedTimelineReturn {
  const [items, setItems] = useState<TimelineItem[]>(() =>
    mergeConsecutiveActions(initialActions).map((action) => recordedActionToTimelineItem(action))
  );
  const [isLive, setIsLive] = useState(false);
  // Error state for future error handling
  const error: string | null = null;

  const { isConnected, lastMessage, send } = useWebSocket();
  const subscribedExecutionRef = useRef<string | null>(null);
  // Track if we've pre-populated from workflow to avoid re-populating on each render
  const prePopulatedWorkflowRef = useRef<string | null>(null);

  // Track previous mode to detect mode switches
  const prevModeRef = useRef<TimelineMode>(mode);

  // Clear timeline when switching from recording to execution mode
  useEffect(() => {
    if (prevModeRef.current === 'recording' && mode === 'execution') {
      // Clear items immediately when switching to execution mode
      // They will be repopulated from workflow nodes when available
      setItems([]);
      prePopulatedWorkflowRef.current = null;
      console.log('[useUnifiedTimeline] Cleared timeline for execution mode');
    }
    prevModeRef.current = mode;
  }, [mode]);

  // Update items when timeline entries or actions change (recording mode)
  // Prefer timeline entries (from useTimeline) over actions (from useRecordMode)
  useEffect(() => {
    if (mode === 'recording') {
      if (initialTimelineEntries && initialTimelineEntries.length > 0) {
        // Preferred path: use timeline entries from useTimeline
        // Separate actions and page events
        const actionEntries = initialTimelineEntries.filter(e => e.type === 'action' && e.action);
        const pageEventEntries = initialTimelineEntries.filter(e => e.type === 'page_event');

        // Convert action entries to RecordedActions for merging
        const actionsForMerging: RecordedAction[] = actionEntries.map(e => ({
          id: e.action!.id,
          sessionId: '',
          sequenceNum: e.action!.sequenceNum,
          timestamp: e.action!.timestamp,
          actionType: e.action!.actionType as RecordedAction['actionType'],
          url: e.action!.url ?? '',
          selector: e.action!.selector ? { primary: e.action!.selector.primary, candidates: [] } : undefined,
          payload: e.action!.payload,
          confidence: e.action!.confidence,
          pageId: e.pageId,
          pageTitle: e.action!.pageTitle,
        }));

        // Merge consecutive actions (typing) for readability
        const mergedActionRecords = mergeConsecutiveActions(actionsForMerging);
        const actionItems = mergedActionRecords.map(action => recordedActionToTimelineItem(action));

        // Convert page events (no merging needed)
        const pageEventItems = pageEventEntries.map(useTimelineEntryToTimelineItem);

        // Combine and sort by timestamp
        const allItems = [...actionItems, ...pageEventItems].sort(
          (a, b) => a.timestamp.getTime() - b.timestamp.getTime()
        );
        setItems(allItems);
      } else if (initialActions.length > 0) {
        // Fallback: use legacy actions from useRecordMode
        const newItems = mergeConsecutiveActions(initialActions).map((action) => recordedActionToTimelineItem(action));
        setItems(newItems);
      } else {
        // No data - clear items
        setItems([]);
      }
      prePopulatedWorkflowRef.current = null; // Reset when switching to recording
    }
  }, [mode, initialActions, initialTimelineEntries]);

  // Pre-populate timeline with workflow steps when entering execution mode
  useEffect(() => {
    console.log('[useUnifiedTimeline] Pre-populate effect triggered:', {
      mode,
      hasWorkflowNodes: !!workflowNodes,
      workflowNodesLength: workflowNodes?.length ?? 0,
      hasWorkflowEdges: !!workflowEdges,
      workflowEdgesLength: workflowEdges?.length ?? 0,
    });

    if (mode === 'execution' && workflowNodes && workflowNodes.length > 0 && workflowEdges) {
      // Create a key to track which workflow we've pre-populated
      const workflowKey = workflowNodes.map(n => n.id).join(',');
      console.log('[useUnifiedTimeline] Workflow key:', workflowKey, 'Previous:', prePopulatedWorkflowRef.current);

      if (prePopulatedWorkflowRef.current !== workflowKey) {
        const workflowItems = workflowNodesToTimelineItems(workflowNodes, workflowEdges);
        console.log('[useUnifiedTimeline] Setting items:', workflowItems);
        setItems(workflowItems);
        prePopulatedWorkflowRef.current = workflowKey;
        console.log('[useUnifiedTimeline] Pre-populated timeline with', workflowItems.length, 'workflow steps');
      } else {
        console.log('[useUnifiedTimeline] Skipping - already populated for this workflow');
      }
    } else {
      console.log('[useUnifiedTimeline] Conditions not met for pre-population');
    }
  }, [mode, workflowNodes, workflowEdges]);

  // Handle WebSocket messages for both modes
  useEffect(() => {
    if (!lastMessage) return;

    const msg = lastMessage as WebSocketMessage & { action?: unknown; timeline_entry?: unknown; node_id?: string; status?: string };

    // Log all messages in execution mode for debugging
    if (mode === 'execution') {
      console.log('[useUnifiedTimeline] WebSocket message received:', msg);
    }

    // Recording mode timeline is driven by initialActions from useRecordMode.

    // Execution mode: Handle timeline entry messages
    // Support both 'step' (legacy) and 'TIMELINE_MESSAGE_TYPE_ENTRY' (current) formats
    if (mode === 'execution' && (msg.type === 'step' || msg.type === 'TIMELINE_MESSAGE_TYPE_ENTRY')) {
      // The message has entry data in the 'entry' field
      const msgAny = msg as unknown as Record<string, unknown>;
      const entryData = msgAny.entry as Record<string, unknown> | undefined;

      if (entryData) {
        console.log('[useUnifiedTimeline] Processing entry:', entryData);

        // Extract node_id from the entry - it might be nested in context or at top level
        const context = entryData.context as Record<string, unknown> | undefined;
        const stepIndex = entryData.step_index as number | undefined;

        // The actual workflow node_id is in context.node_id (for completed steps)
        // For "step started" messages, we need to use step_index to find the matching item
        const actualNodeId = (context?.node_id as string) ?? (entryData.node_id as string);

        // Extract status from context
        const isSuccess = context?.success as boolean | undefined;
        const errorMsg = context?.error as string | undefined;

        // Determine execution status
        let execStatus: ExecutionStatus = 'running';
        if (isSuccess === true) {
          execStatus = 'completed';
        } else if (isSuccess === false || errorMsg) {
          execStatus = 'failed';
        }

        console.log('[useUnifiedTimeline] Extracted:', { actualNodeId, stepIndex, execStatus, isSuccess, errorMsg });

        setItems((prev) => {
          console.log('[useUnifiedTimeline] Current items nodeIds:', prev.map(item => (item as ExecutionTimelineItem).nodeId));

          // Try to find the matching item
          let prePopIndex = -1;

          // First, try by actual node_id (for completed steps)
          if (actualNodeId) {
            console.log('[useUnifiedTimeline] Looking for actualNodeId:', actualNodeId);
            prePopIndex = prev.findIndex((item) =>
              (item as ExecutionTimelineItem).nodeId === actualNodeId
            );
          }

          // If not found and we have step_index, try by index (for "step started" messages)
          if (prePopIndex === -1 && stepIndex !== undefined) {
            console.log('[useUnifiedTimeline] Looking by stepIndex:', stepIndex);
            // step_index is 0-based, matches the order of pre-populated items
            if (stepIndex >= 0 && stepIndex < prev.length) {
              const item = prev[stepIndex] as ExecutionTimelineItem;
              // Only match if it's a pre-populated item (has a nodeId that's not an execution ID)
              if (item.nodeId && !item.nodeId.includes('-step-')) {
                prePopIndex = stepIndex;
              }
            }
          }

          console.log('[useUnifiedTimeline] Found at index:', prePopIndex);

          if (prePopIndex >= 0) {
            const targetNodeId = (prev[prePopIndex] as ExecutionTimelineItem).nodeId;
            const updated = updateTimelineItemStatus(
              prev as ExecutionTimelineItem[],
              targetNodeId,
              execStatus,
              errorMsg,
              entryData.duration_ms as number | undefined
            );
            console.log('[useUnifiedTimeline] Updated step', targetNodeId, 'to status', execStatus);
            return updated;
          }

          // Node not found - DON'T add new items for execution steps
          // We only want to show pre-populated workflow steps
          console.log('[useUnifiedTimeline] Ignoring unmatched entry (not adding new item)');
          return prev;
        });
      }
    }

    // Execution started/stopped
    if (msg.type === 'execution_started') {
      setIsLive(true);
      // Don't clear items if we have pre-populated from workflow
      // Instead, reset all items to pending status
      if (prePopulatedWorkflowRef.current) {
        setItems((prev) => prev.map(item => ({
          ...item,
          success: undefined,
          error: undefined,
          executionStatus: 'pending' as ExecutionStatus,
        } as ExecutionTimelineItem)));
      } else {
        setItems([]); // Clear on new execution if no workflow pre-populated
      }
    }

    if (msg.type === 'execution_completed' || msg.type === 'execution_failed') {
      setIsLive(false);
    }
  }, [lastMessage, mode, onItemsReceived]);

  // Subscribe to execution updates
  const subscribeToExecution = useCallback(
    (execId: string) => {
      if (subscribedExecutionRef.current === execId) return;

      if (subscribedExecutionRef.current) {
        send({ type: 'unsubscribe_execution', execution_id: subscribedExecutionRef.current });
      }

      send({ type: 'subscribe_execution', execution_id: execId });
      subscribedExecutionRef.current = execId;
      setIsLive(true);
      // Don't clear items if pre-populated from workflow
      if (!prePopulatedWorkflowRef.current) {
        setItems([]);
      }
      console.log('[useUnifiedTimeline] Subscribed to execution:', execId);
    },
    [send]
  );

  // Unsubscribe from execution updates
  const unsubscribeFromExecution = useCallback(() => {
    if (subscribedExecutionRef.current) {
      send({ type: 'unsubscribe_execution', execution_id: subscribedExecutionRef.current });
      subscribedExecutionRef.current = null;
      setIsLive(false);
      console.log('[useUnifiedTimeline] Unsubscribed from execution');
    }
  }, [send]);

  // Auto-subscribe to execution when executionId is provided
  useEffect(() => {
    if (mode === 'execution' && executionId && isConnected) {
      subscribeToExecution(executionId);
    }

    return () => {
      if (mode === 'execution') {
        unsubscribeFromExecution();
      }
    };
  }, [mode, executionId, isConnected, subscribeToExecution, unsubscribeFromExecution]);

  // Clear items
  const clearItems = useCallback(() => {
    setItems([]);
  }, []);

  // Convert items back to RecordedActions for legacy component compatibility
  const getRecordedActions = useCallback((): RecordedAction[] => {
    return items
      .map((item) => {
        // If we have the raw TimelineEntry, convert it
        if (item.rawEntry) {
          return timelineEntryToRecordedAction(item.rawEntry);
        }

        // Otherwise create a minimal RecordedAction from TimelineItem
        return {
          id: item.id,
          sessionId: '',
          sequenceNum: item.sequenceNum,
          timestamp: item.timestamp.toISOString(),
          durationMs: item.durationMs,
          actionType: item.actionType as RecordedAction['actionType'],
          confidence: 1.0,
          url: item.url ?? '',
          selector: item.selector ? { primary: item.selector, candidates: [] } : undefined,
        } as RecordedAction;
      })
      .filter((a): a is RecordedAction => a !== null);
  }, [items]);

  // Calculate stats
  const stats = useMemo(() => {
    const total = items.length;
    const successful = items.filter((item) => item.success === true).length;
    const failed = items.filter((item) => item.success === false).length;
    const pending = items.filter((item) => item.success === undefined).length;
    return { total, successful, failed, pending };
  }, [items]);

  return {
    items,
    mode,
    isLive: isLive || mode === 'recording',
    isConnected,
    error,
    clearItems,
    getRecordedActions,
    subscribeToExecution,
    unsubscribeFromExecution,
    stats,
  };
}
