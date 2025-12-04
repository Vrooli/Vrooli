/**
 * useExecutionUpdates Hook
 * Listens to WebSocket messages and updates execution/dashboard stores
 */

import { useEffect } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { useDashboardStore } from '@stores/dashboardStore';
import { useExecutionStore } from '@stores/executionStore';
import { logger } from '@utils/logger';

export function useExecutionUpdates() {
  const { lastMessage } = useWebSocket();
  const { fetchRunningExecutions, fetchRecentExecutions } = useDashboardStore();
  const { refreshTimeline, currentExecution } = useExecutionStore();

  useEffect(() => {
    if (!lastMessage) return;

    const { type, execution_id: executionId } = lastMessage;

    const refreshDashboard = () => {
      void fetchRunningExecutions();
      void fetchRecentExecutions();
    };

    const refreshCurrentExecution = () => {
      if (executionId && currentExecution?.id === executionId) {
        void refreshTimeline(executionId);
      }
    };

    // Handle different WebSocket message types
    switch (type) {
      case 'connected':
        // Initial handshake - no action needed
        logger.debug('WebSocket handshake received', { component: 'useExecutionUpdates' });
        break;

      case 'execution_started':
      case 'execution_running':
        refreshDashboard();
        break;

      case 'execution_completed':
      case 'execution_failed':
      case 'execution_cancelled':
        refreshDashboard();
        refreshCurrentExecution();
        break;

      case 'step_started':
      case 'step_completed':
      case 'step_failed':
      case 'progress':
        refreshCurrentExecution();
        break;

      case 'workflow_created':
      case 'workflow_updated':
      case 'workflow_deleted':
        // Workflow changes - could refresh workflows list if needed
        logger.debug('Workflow update received', { component: 'useExecutionUpdates', type });
        break;

      default:
        // Unknown message type - check if it's an envelope event
        if (lastMessage.data && typeof lastMessage.data === 'object') {
          const data = lastMessage.data as { kind?: string };
          if (data.kind) {
            // This is an event envelope, handle based on kind
            handleEnvelopeEvent(data.kind, executionId);
          }
        }
    }

    function handleEnvelopeEvent(kind: string, execId?: string) {
      switch (kind) {
        case 'execution.started':
        case 'execution.running':
          refreshDashboard();
          break;

        case 'execution.completed':
        case 'execution.failed':
        case 'execution.cancelled':
          refreshDashboard();
          if (execId && currentExecution?.id === execId) {
            void refreshTimeline(execId);
          }
          break;

        case 'step.started':
        case 'step.completed':
        case 'step.failed':
        case 'step.progress':
          if (execId && currentExecution?.id === execId) {
            void refreshTimeline(execId);
          }
          break;

        default:
          logger.debug('Unknown envelope event', { component: 'useExecutionUpdates', kind });
      }
    }
  }, [lastMessage, fetchRunningExecutions, fetchRecentExecutions, refreshTimeline, currentExecution]);
}
