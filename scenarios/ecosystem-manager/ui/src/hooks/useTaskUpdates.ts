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

    const { type, task_id } = lastMessage;

    // Handle different WebSocket message types
    switch (type) {
      case 'task_status_update':
        // Task status changed - invalidate task lists and specific task
        if (task_id) {
          queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(task_id) });
        }
        queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
        break;

      case 'process_started':
      case 'process_completed':
        // Process state changed - invalidate task details and lists
        if (task_id) {
          queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(task_id) });
        }
        queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
        queryClient.invalidateQueries({ queryKey: queryKeys.processes.running() });
        break;

      case 'queue_status_update':
        // Queue status changed - invalidate queue status
        queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
        break;

      case 'rate_limit_notification':
        // Rate limit hit - invalidate queue status
        queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
        break;

      default:
        // Unknown message type - log for debugging
        console.log('[useTaskUpdates] Unknown WebSocket message type:', type);
    }
  }, [lastMessage, queryClient]);
}
