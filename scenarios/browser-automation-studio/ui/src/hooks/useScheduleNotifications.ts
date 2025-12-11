/**
 * useScheduleNotifications Hook
 * Listens to WebSocket messages for schedule execution events and shows notifications.
 * Also updates the tray with schedule information.
 */

import { useEffect, useCallback } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { useScheduleStore } from '@stores/scheduleStore';
import { logger } from '@utils/logger';
import {
  showScheduleNotification,
  updateTrayWithSchedules,
  isDesktopEnvironment,
} from '@/lib/desktop/tray';

interface ScheduleExecutionPayload {
  schedule_id: string;
  schedule_name: string;
  execution_id?: string;
  status: 'started' | 'completed' | 'failed';
  duration_ms?: number;
  error?: string;
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = ((ms % 60000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
}

export function useScheduleNotifications() {
  const { lastMessage } = useWebSocket();
  const { schedules, fetchSchedules } = useScheduleStore();

  // Update tray whenever schedules change
  useEffect(() => {
    if (isDesktopEnvironment()) {
      void updateTrayWithSchedules(schedules);
    }
  }, [schedules]);

  const handleScheduleEvent = useCallback((payload: ScheduleExecutionPayload) => {
    const { schedule_name, status, duration_ms, error } = payload;

    switch (status) {
      case 'started':
        logger.info('Schedule execution started', {
          component: 'useScheduleNotifications',
          scheduleName: schedule_name,
        });
        break;

      case 'completed':
        showScheduleNotification(
          `${schedule_name} completed`,
          `Execution finished${duration_ms ? ` in ${formatDuration(duration_ms)}` : ''}`,
          { urgency: 'normal' }
        );
        // Refresh schedules to update last_run_at
        void fetchSchedules();
        break;

      case 'failed':
        showScheduleNotification(
          `${schedule_name} failed`,
          error || 'Unknown error occurred',
          { urgency: 'critical' }
        );
        // Refresh schedules to update status
        void fetchSchedules();
        break;
    }
  }, [fetchSchedules]);

  useEffect(() => {
    if (!lastMessage) return;

    const { type, data } = lastMessage;

    // Check for schedule-specific event types
    if (type === 'schedule.execution.started' ||
        type === 'schedule.execution.completed' ||
        type === 'schedule.execution.failed') {
      const payload = data as ScheduleExecutionPayload;
      if (payload) {
        handleScheduleEvent(payload);
      }
      return;
    }

    // Check for envelope-style events
    if (data && typeof data === 'object') {
      const envelope = data as { kind?: string; payload?: ScheduleExecutionPayload };
      if (envelope.kind?.startsWith('schedule.execution.') && envelope.payload) {
        handleScheduleEvent(envelope.payload);
      }
    }
  }, [lastMessage, handleScheduleEvent]);
}
