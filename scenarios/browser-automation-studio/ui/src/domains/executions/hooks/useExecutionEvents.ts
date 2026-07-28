import { useEffect, useRef, useCallback, useMemo } from 'react';
import { useWebSocket, type WebSocketMessage } from '@/contexts/WebSocketContext';
import { useExecutionStore, type Execution } from '../store';
import {
  parseStreamMessage,
  streamMessageToExecutionEvent,
} from '../live/executionEvents';
import {
  processExecutionEvent,
  type ExecutionEventHandlers,
} from '../utils/eventProcessor';
import { logger } from '@utils/logger';

/**
 * Hook that bridges WebSocketContext to the execution store.
 *
 * Subscribes to execution updates when an execution is running,
 * processes incoming messages through ExecutionEventsClient,
 * and calls store methods to update state.
 *
 * This replaces the store's direct WebSocket connection with the
 * shared WebSocketContext, following the @vrooli/api-base pattern.
 */
export function useExecutionEvents(execution?: Pick<Execution, 'id' | 'status'>) {
  const { lastMessage, isConnected, send } = useWebSocket();

  // Get store methods (stable references via selector)
  const updateExecutionStatus = useExecutionStore(s => s.updateExecutionStatus);
  const updateProgress = useExecutionStore(s => s.updateProgress);
  const addLog = useExecutionStore(s => s.addLog);
  const addScreenshot = useExecutionStore(s => s.addScreenshot);
  const recordHeartbeat = useExecutionStore(s => s.recordHeartbeat);

  // Track current subscription to avoid duplicate subscribe messages
  const subscribedIdRef = useRef<string | null>(null);

  // Memoize handlers object to prevent infinite re-render loops
  // Store methods from Zustand are stable references, but the object itself must be memoized
  const handlers: ExecutionEventHandlers = useMemo(() => ({
    updateExecutionStatus,
    updateProgress,
    addLog,
    addScreenshot,
    recordHeartbeat,
  }), [updateExecutionStatus, updateProgress, addLog, addScreenshot, recordHeartbeat]);

  // Process a WebSocket message for execution events
  const processMessage = useCallback((message: WebSocketMessage) => {
    // Try to parse as proto TimelineStreamMessage first
    const envelope = parseStreamMessage(message);
    if (envelope) {
      const event = streamMessageToExecutionEvent(envelope);
      if (event) {
        processExecutionEvent(handlers, event, {
          fallbackProgress: event.progress,
        });
      }
      return;
    }

    // Try nested envelope (message.data contains the proto)
    if (message.data && typeof message.data === 'object') {
      const nestedEnvelope = parseStreamMessage(message.data);
      if (nestedEnvelope) {
        const event = streamMessageToExecutionEvent(nestedEnvelope);
        if (event) {
          processExecutionEvent(handlers, event, {
            fallbackProgress: event.progress,
          });
        }
        return;
      }
    }

  }, [handlers]);

  // Subscribe/unsubscribe based on execution state
  useEffect(() => {
    const executionId = execution?.id;
    const status = execution?.status;
    // Subscribe when execution is active (pending or running)
    // Unsubscribe when completed, failed, or cancelled
    const isActive = status === 'pending' || status === 'running';

    // Unsubscribe if no execution or not active
    if (!executionId || !isActive) {
      if (subscribedIdRef.current) {
        send({ type: 'unsubscribe', execution_id: subscribedIdRef.current });
        subscribedIdRef.current = null;
      }
      return;
    }

    // Subscribe to new execution
    if (subscribedIdRef.current !== executionId) {
      // Unsubscribe from previous if different
      if (subscribedIdRef.current) {
        send({ type: 'unsubscribe', execution_id: subscribedIdRef.current });
      }
      send({ type: 'subscribe', execution_id: executionId });
      subscribedIdRef.current = executionId;
      logger.debug('Subscribed to execution updates', {
        component: 'useExecutionEvents',
        executionId,
      });
    }

    // Cleanup on unmount or when execution changes
    return () => {
      if (subscribedIdRef.current) {
        send({ type: 'unsubscribe', execution_id: subscribedIdRef.current });
        subscribedIdRef.current = null;
      }
    };
  }, [execution?.id, execution?.status, send]);

  // Process incoming messages for this execution
  useEffect(() => {
    if (!lastMessage || !execution?.id) return;

    // Filter messages that aren't for this execution
    if (lastMessage.execution_id && lastMessage.execution_id !== execution.id) {
      return;
    }

    try {
      processMessage(lastMessage);
    } catch (err) {
      logger.error('Failed to process execution event', {
        component: 'useExecutionEvents',
        executionId: execution.id,
      }, err);
    }
  }, [lastMessage, execution?.id, processMessage]);

  return { isConnected };
}
