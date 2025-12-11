import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';

export interface WorkflowSchedule {
  id: string;
  workflow_id: string;
  name: string;
  description?: string;
  cron_expression: string;
  timezone: string;
  is_active: boolean;
  parameters?: Record<string, unknown>;
  next_run_at?: string;
  last_run_at?: string;
  created_at?: string;
  updated_at?: string;
  // Computed fields from API
  workflow_name?: string;
  next_run_human?: string;
  last_run_status?: string;
}

export interface CreateScheduleInput {
  name: string;
  description?: string;
  cron_expression: string;
  timezone?: string;
  parameters?: Record<string, unknown>;
  is_active?: boolean;
}

export interface UpdateScheduleInput {
  name?: string;
  description?: string;
  cron_expression?: string;
  timezone?: string;
  parameters?: Record<string, unknown>;
  is_active?: boolean;
}

interface ScheduleState {
  schedules: WorkflowSchedule[];
  selectedScheduleId: string | null;
  isLoading: boolean;
  error: string | null;

  // Actions
  fetchSchedules: (workflowId?: string) => Promise<void>;
  fetchSchedulesByWorkflow: (workflowId: string) => Promise<void>;
  createSchedule: (workflowId: string, input: CreateScheduleInput) => Promise<WorkflowSchedule | null>;
  updateSchedule: (scheduleId: string, input: UpdateScheduleInput) => Promise<WorkflowSchedule | null>;
  deleteSchedule: (scheduleId: string) => Promise<boolean>;
  toggleSchedule: (scheduleId: string) => Promise<WorkflowSchedule | null>;
  triggerSchedule: (scheduleId: string) => Promise<{ execution_id: string } | null>;
  selectSchedule: (scheduleId: string | null) => void;
  clearError: () => void;
  reset: () => void;
}

const initialState = {
  schedules: [] as WorkflowSchedule[],
  selectedScheduleId: null as string | null,
  isLoading: false,
  error: null as string | null,
};

export const useScheduleStore = create<ScheduleState>((set) => ({
  ...initialState,

  fetchSchedules: async (workflowId?: string) => {
    set({ isLoading: true, error: null });

    try {
      const config = await getConfig();
      let url = `${config.API_URL}/schedules`;
      if (workflowId) {
        url += `?workflow_id=${workflowId}`;
      }

      const response = await fetch(url);
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load schedules (${response.status})`);
      }

      const payload = await response.json();
      const schedules = Array.isArray(payload?.schedules) ? payload.schedules : [];

      set({ schedules, isLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to load schedules';
      logger.error('Failed to load schedules', { component: 'ScheduleStore', action: 'fetchSchedules' }, error);
      set({ error: message, isLoading: false });
    }
  },

  fetchSchedulesByWorkflow: async (workflowId: string) => {
    set({ isLoading: true, error: null });

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${workflowId}/schedules`);

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load workflow schedules (${response.status})`);
      }

      const payload = await response.json();
      const schedules = Array.isArray(payload?.schedules) ? payload.schedules : [];

      set({ schedules, isLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to load workflow schedules';
      logger.error('Failed to load workflow schedules', { component: 'ScheduleStore', action: 'fetchSchedulesByWorkflow' }, error);
      set({ error: message, isLoading: false });
    }
  },

  createSchedule: async (workflowId: string, input: CreateScheduleInput) => {
    set({ isLoading: true, error: null });

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${workflowId}/schedules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(input),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to create schedule (${response.status})`);
      }

      const payload = await response.json();
      const newSchedule = payload.schedule as WorkflowSchedule;

      // Add to local state
      set((state) => ({
        schedules: [newSchedule, ...state.schedules],
        isLoading: false,
      }));

      return newSchedule;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to create schedule';
      logger.error('Failed to create schedule', { component: 'ScheduleStore', action: 'createSchedule' }, error);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  updateSchedule: async (scheduleId: string, input: UpdateScheduleInput) => {
    set({ isLoading: true, error: null });

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/schedules/${scheduleId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(input),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to update schedule (${response.status})`);
      }

      const payload = await response.json();
      const updatedSchedule = payload.schedule as WorkflowSchedule;

      // Update in local state
      set((state) => ({
        schedules: state.schedules.map((s) =>
          s.id === scheduleId ? updatedSchedule : s
        ),
        isLoading: false,
      }));

      return updatedSchedule;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to update schedule';
      logger.error('Failed to update schedule', { component: 'ScheduleStore', action: 'updateSchedule' }, error);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  deleteSchedule: async (scheduleId: string) => {
    set({ isLoading: true, error: null });

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/schedules/${scheduleId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to delete schedule (${response.status})`);
      }

      // Remove from local state
      set((state) => ({
        schedules: state.schedules.filter((s) => s.id !== scheduleId),
        selectedScheduleId: state.selectedScheduleId === scheduleId ? null : state.selectedScheduleId,
        isLoading: false,
      }));

      return true;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to delete schedule';
      logger.error('Failed to delete schedule', { component: 'ScheduleStore', action: 'deleteSchedule' }, error);
      set({ error: message, isLoading: false });
      return false;
    }
  },

  toggleSchedule: async (scheduleId: string) => {
    set({ isLoading: true, error: null });

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/schedules/${scheduleId}/toggle`, {
        method: 'POST',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to toggle schedule (${response.status})`);
      }

      const payload = await response.json();
      const updatedSchedule = payload.schedule as WorkflowSchedule;

      // Update in local state
      set((state) => ({
        schedules: state.schedules.map((s) =>
          s.id === scheduleId ? updatedSchedule : s
        ),
        isLoading: false,
      }));

      return updatedSchedule;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to toggle schedule';
      logger.error('Failed to toggle schedule', { component: 'ScheduleStore', action: 'toggleSchedule' }, error);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  triggerSchedule: async (scheduleId: string) => {
    set({ isLoading: true, error: null });

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/schedules/${scheduleId}/trigger`, {
        method: 'POST',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to trigger schedule (${response.status})`);
      }

      const payload = await response.json();

      // Update last_run_at in local state
      set((state) => ({
        schedules: state.schedules.map((s) =>
          s.id === scheduleId
            ? { ...s, last_run_at: payload.triggered_at, last_run_status: 'running' }
            : s
        ),
        isLoading: false,
      }));

      return { execution_id: payload.execution_id };
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to trigger schedule';
      logger.error('Failed to trigger schedule', { component: 'ScheduleStore', action: 'triggerSchedule' }, error);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  selectSchedule: (scheduleId: string | null) => {
    set({ selectedScheduleId: scheduleId });
  },

  clearError: () => {
    set({ error: null });
  },

  reset: () => {
    set(initialState);
  },
}));

// Common cron presets for UI dropdowns
export const CRON_PRESETS = [
  { label: 'Every minute', value: '* * * * *', description: 'Runs at the start of every minute' },
  { label: 'Every 5 minutes', value: '*/5 * * * *', description: 'Runs every 5 minutes' },
  { label: 'Every 15 minutes', value: '*/15 * * * *', description: 'Runs every 15 minutes' },
  { label: 'Every 30 minutes', value: '*/30 * * * *', description: 'Runs every 30 minutes' },
  { label: 'Every hour', value: '0 * * * *', description: 'Runs at the start of every hour' },
  { label: 'Every 2 hours', value: '0 */2 * * *', description: 'Runs every 2 hours' },
  { label: 'Every 6 hours', value: '0 */6 * * *', description: 'Runs every 6 hours' },
  { label: 'Daily at midnight', value: '0 0 * * *', description: 'Runs every day at 00:00' },
  { label: 'Daily at 6 AM', value: '0 6 * * *', description: 'Runs every day at 06:00' },
  { label: 'Daily at 9 AM', value: '0 9 * * *', description: 'Runs every day at 09:00' },
  { label: 'Daily at noon', value: '0 12 * * *', description: 'Runs every day at 12:00' },
  { label: 'Weekdays at 9 AM', value: '0 9 * * 1-5', description: 'Runs Monday-Friday at 09:00' },
  { label: 'Weekly on Monday', value: '0 0 * * 1', description: 'Runs every Monday at midnight' },
  { label: 'Monthly on the 1st', value: '0 0 1 * *', description: 'Runs on the 1st of each month' },
] as const;

// Common timezones for UI dropdowns
export const COMMON_TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'America/Toronto',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Asia/Singapore',
  'Australia/Sydney',
  'Pacific/Auckland',
] as const;

// Helper to format next run time
export function formatNextRun(nextRunAt?: string, nextRunHuman?: string): string {
  if (nextRunHuman) return nextRunHuman;
  if (!nextRunAt) return 'Not scheduled';

  const date = new Date(nextRunAt);
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();

  if (diffMs < 0) return 'Overdue';
  if (diffMs < 60000) return 'In less than a minute';
  if (diffMs < 3600000) return `In ${Math.round(diffMs / 60000)} minutes`;
  if (diffMs < 86400000) return `In ${Math.round(diffMs / 3600000)} hours`;
  return `In ${Math.round(diffMs / 86400000)} days`;
}

// Helper to describe cron expression
export function describeCron(expr: string): string {
  const preset = CRON_PRESETS.find(p => p.value === expr);
  if (preset) return preset.description;
  return `Cron: ${expr}`;
}
