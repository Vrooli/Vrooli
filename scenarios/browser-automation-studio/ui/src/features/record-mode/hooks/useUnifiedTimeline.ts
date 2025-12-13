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
import type { RecordedAction } from '../types';
import type { TimelineItem, TimelineMode } from '../types/timeline-unified';
import {
  recordedActionToTimelineItem,
  timelineEventToTimelineItem,
  timelineEventToRecordedAction,
  hasTimelineEvent,
  parseTimelineEvent,
} from '../types/timeline-unified';

export interface UseUnifiedTimelineOptions {
  /** Current mode: 'recording' for live recording, 'execution' for workflow playback */
  mode: TimelineMode;
  /** Execution ID for execution mode */
  executionId?: string | null;
  /** Optional: Pre-existing recorded actions (for recording mode) */
  initialActions?: RecordedAction[];
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
  onItemsReceived,
}: UseUnifiedTimelineOptions): UseUnifiedTimelineReturn {
  const [items, setItems] = useState<TimelineItem[]>(() =>
    initialActions.map(recordedActionToTimelineItem)
  );
  const [isLive, setIsLive] = useState(false);
  // Error state for future error handling
  const error: string | null = null;

  const { isConnected, lastMessage, send } = useWebSocket();
  const subscribedExecutionRef = useRef<string | null>(null);

  // Update items when initialActions change (recording mode)
  useEffect(() => {
    if (mode === 'recording' && initialActions.length > 0) {
      const newItems = initialActions.map(recordedActionToTimelineItem);
      setItems(newItems);
    }
  }, [mode, initialActions]);

  // Handle WebSocket messages for both modes
  useEffect(() => {
    if (!lastMessage) return;

    const msg = lastMessage as WebSocketMessage & { action?: unknown; timeline_event?: unknown };

    // Recording mode: Handle recording_action messages
    if (mode === 'recording' && msg.type === 'recording_action' && msg.action) {
      const action = msg.action as RecordedAction;
      const newItem = recordedActionToTimelineItem(action);

      setItems((prev) => {
        const exists = prev.some((item) => item.id === newItem.id);
        if (exists) return prev;

        const updated = [...prev, newItem];
        if (onItemsReceived) {
          onItemsReceived([newItem]);
        }
        return updated;
      });
    }

    // Execution mode: Handle step events with timeline_event
    if (mode === 'execution' && msg.type === 'step') {
      // Check if the message includes the new timeline_event field
      if (hasTimelineEvent(msg)) {
        const timelineEvent = parseTimelineEvent(msg.timeline_event);
        if (timelineEvent) {
          const newItem = timelineEventToTimelineItem(timelineEvent);

          setItems((prev) => {
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
        };

        if (stepData.node_id) {
          const newItem: TimelineItem = {
            id: stepData.node_id,
            sequenceNum: stepData.step_num ?? 0,
            timestamp: new Date(),
            actionType: stepData.action ?? 'unknown',
            success: stepData.status === 'completed',
            error: stepData.error,
            mode: 'execution',
          };

          setItems((prev) => {
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
      setItems([]); // Clear on new execution
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
      setItems([]); // Clear for new execution
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
        // If we have the legacy action, return it
        if (item.legacyAction) {
          return item.legacyAction;
        }

        // If we have the raw TimelineEvent, convert it
        if (item.rawEvent) {
          return timelineEventToRecordedAction(item.rawEvent);
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
