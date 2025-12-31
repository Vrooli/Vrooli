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
import type { TimelineItem, TimelineMode, ExecutionTimelineItem, ExecutionStatus } from '../types/timeline-unified';
import {
  recordedActionToTimelineItem,
  timelineEntryToTimelineItem,
  timelineEntryToRecordedAction,
  hasTimelineEntry,
  parseTimelineEntry,
  workflowNodesToTimelineItems,
  updateTimelineItemStatus,
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
  /** Optional: Pre-existing recorded actions (for recording mode) */
  initialActions?: RecordedAction[];
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

  // Update items when initialActions change (recording mode)
  useEffect(() => {
    if (mode === 'recording') {
      const newItems = mergeConsecutiveActions(initialActions).map((action) => recordedActionToTimelineItem(action));
      setItems(newItems);
      prePopulatedWorkflowRef.current = null; // Reset when switching to recording
    }
  }, [mode, initialActions]);

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

    const msg = lastMessage as WebSocketMessage & { action?: unknown; timeline_entry?: unknown };

    // Recording mode timeline is driven by initialActions from useRecordMode.

    // Execution mode: Handle step events with timeline_entry (V2 unified format)
    if (mode === 'execution' && msg.type === 'step') {
      // Check if the message includes the new timeline_entry field
      if (hasTimelineEntry(msg)) {
        const timelineEntry = parseTimelineEntry(msg.timeline_entry);
        if (timelineEntry) {
          const newItem = timelineEntryToTimelineItem(timelineEntry);

          setItems((prev) => {
            // First, try to update a pre-populated item by node_id
            const nodeId = (msg as { node_id?: string }).node_id;
            if (nodeId) {
              const prePopIndex = prev.findIndex((item) =>
                (item as ExecutionTimelineItem).nodeId === nodeId
              );
              if (prePopIndex >= 0) {
                // Determine status from context
                const status: ExecutionStatus = newItem.success === true ? 'completed'
                  : newItem.success === false ? 'failed'
                  : 'running';
                const updated = [...prev];
                updated[prePopIndex] = {
                  ...prev[prePopIndex],
                  ...newItem,
                  nodeId,
                  executionStatus: status,
                } as ExecutionTimelineItem;
                return updated;
              }
            }

            // Check if we're updating an existing item (retry) or adding new
            const existingIndex = prev.findIndex((item) => item.id === newItem.id);
            if (existingIndex >= 0) {
              // Update existing item (e.g., on retry)
              const updated = [...prev];
              updated[existingIndex] = newItem;
              return updated;
            }

            const updated = [...prev, newItem];
            if (onItemsReceived) {
              onItemsReceived([newItem]);
            }
            return updated;
          });
        }
      } else {
        // Fall back to legacy step event format
        const stepData = msg as {
          node_id?: string;
          step_num?: number;
          action?: string;
          status?: string;
          error?: string;
          duration_ms?: number;
        };

        if (stepData.node_id) {
          // Map status string to ExecutionStatus
          const status: ExecutionStatus = stepData.status === 'completed' ? 'completed'
            : stepData.status === 'failed' ? 'failed'
            : stepData.status === 'running' || stepData.status === 'started' ? 'running'
            : 'pending';

          setItems((prev) => {
            // First, try to update a pre-populated item by node_id
            const prePopIndex = prev.findIndex((item) =>
              (item as ExecutionTimelineItem).nodeId === stepData.node_id
            );
            if (prePopIndex >= 0) {
              const updated = updateTimelineItemStatus(
                prev as ExecutionTimelineItem[],
                stepData.node_id!,
                status,
                stepData.error,
                stepData.duration_ms
              );
              console.log('[useUnifiedTimeline] Updated step', stepData.node_id, 'to status', status);
              return updated;
            }

            // Otherwise, add as a new item (fallback for backwards compatibility)
            const newItem: TimelineItem = {
              id: stepData.node_id!,
              sequenceNum: stepData.step_num ?? 0,
              timestamp: new Date(),
              actionType: stepData.action ?? 'unknown',
              success: status === 'completed' ? true : status === 'failed' ? false : undefined,
              error: stepData.error,
              mode: 'execution',
              durationMs: stepData.duration_ms,
            };

            const existingIndex = prev.findIndex((item) => item.id === newItem.id);
            if (existingIndex >= 0) {
              const updated = [...prev];
              updated[existingIndex] = newItem;
              return updated;
            }

            const updated = [...prev, newItem];
            if (onItemsReceived) {
              onItemsReceived([newItem]);
            }
            return updated;
          });
        }
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
