/**
 * useTaskUpdates Hook
 * Listens to WebSocket messages and invalidates queries on task updates
 */

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useWebSocket } from '../contexts/WebSocketContext';
import { queryKeys } from '../lib/queryKeys';

export function useTaskUpdates() {
  const { lastMessage } = useWebSocket();
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!lastMessage) return;

    const { type, data } = lastMessage;
    const payload = (data ?? {}) as Record<string, unknown>;
    const taskId =
      typeof payload.task_id === 'string'
        ? payload.task_id
        : typeof payload.id === 'string'
          ? payload.id
          : undefined;

    const invalidateTasks = () => {
      if (taskId) {
        queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(taskId) });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
      queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
    };

    const invalidateProcesses = () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.processes.running() });
    };

    const invalidateAutoSteerState = () => {
      if (taskId) {
        queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.executionState(taskId) });
      }
    };

    // Handle different WebSocket message types
    switch (type) {
      case 'task_status_changed':
      case 'task_status_updated':
      case 'task_updated':
      case 'task_recycled':
      case 'task_deleted':
        invalidateTasks();
        break;

      case 'task_started':
      case 'task_executing':
        invalidateTasks();
        invalidateProcesses();
        invalidateAutoSteerState();
        break;

      case 'task_completed':
      case 'task_failed':
      case 'claude_execution_complete':
      case 'process_terminated':
        invalidateTasks();
        invalidateProcesses();
        invalidateAutoSteerState();
        break;

      case 'task_progress':
        if (taskId) {
          queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(taskId) });
        }
        break;

      case 'log_entry':
        // High-volume streaming event; UI components subscribe directly elsewhere.
        break;

      case 'settings_updated':
      case 'settings_reset':
        queryClient.invalidateQueries({ queryKey: queryKeys.settings.get() });
        queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
        break;

      case 'rate_limit_pause':
      case 'rate_limit_pause_started':
      case 'rate_limit_resume':
      case 'rate_limit_manual_reset':
      case 'rate_limit_hit':
        queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
        break;

      case 'connected':
        // Initial handshake event; no invalidation needed
        break;

      default:
        // Unknown message type - log for debugging
        console.log('[useTaskUpdates] Unknown WebSocket message type:', type);
    }
  }, [lastMessage, queryClient]);
}
