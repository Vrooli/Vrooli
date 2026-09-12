import { ConnectError } from '@connectrpc/connect';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { create } from 'zustand';

import { schedulesClient } from '../api/schedules';
import { logger } from '../utils/logger';
import { protoTimestampToISOString } from '../utils/timestamps';
import type {
  WorkflowSchedule as PbWorkflowSchedule,
  ScheduleOccurrence as PbScheduleOccurrence,
  ScheduleAggregate as PbScheduleAggregate,
} from '@vrooli/proto-types/browser-automation-studio/v1/schedules/schedules_pb';

// =============================================================================
// UI-facing types (snake_case, preserved for component compatibility)
// =============================================================================

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

export interface ScheduleOccurrence {
  schedule_id: string;
  schedule_name: string;
  workflow_id: string;
  workflow_name: string;
  run_at: string;
  is_active: boolean;
  cron_expression: string;
  timezone: string;
}

export interface ScheduleAggregate {
  schedule_id: string;
  schedule_name: string;
  total_runs: number;
  truncated: boolean;
  cron_expression: string;
}

// =============================================================================
// Mappers (proto → snake_case UI shape)
// =============================================================================

const emptyToUndefined = (s: string | undefined): string | undefined =>
  s && s.length > 0 ? s : undefined;

const mapSchedule = (s: PbWorkflowSchedule): WorkflowSchedule => ({
  id: s.id,
  workflow_id: s.workflowId,
  name: s.name,
  description: emptyToUndefined(s.description),
  cron_expression: s.cronExpression,
  timezone: s.timezone,
  is_active: s.isActive,
  parameters: s.parameters ? (s.parameters as Record<string, unknown>) : undefined,
  next_run_at: protoTimestampToISOString(s.nextRunAt),
  last_run_at: protoTimestampToISOString(s.lastRunAt),
  created_at: protoTimestampToISOString(s.createdAt),
  updated_at: protoTimestampToISOString(s.updatedAt),
  workflow_name: emptyToUndefined(s.workflowName),
  next_run_human: emptyToUndefined(s.nextRunHuman),
  last_run_status: emptyToUndefined(s.lastRunStatus),
});

const mapOccurrence = (o: PbScheduleOccurrence): ScheduleOccurrence => ({
  schedule_id: o.scheduleId,
  schedule_name: o.scheduleName,
  workflow_id: o.workflowId,
  workflow_name: o.workflowName,
  run_at: protoTimestampToISOString(o.runAt) ?? '',
  is_active: o.isActive,
  cron_expression: o.cronExpression,
  timezone: o.timezone,
});

const mapAggregate = (a: PbScheduleAggregate): ScheduleAggregate => ({
  schedule_id: a.scheduleId,
  schedule_name: a.scheduleName,
  total_runs: a.totalRuns,
  truncated: a.truncated,
  cron_expression: a.cronExpression,
});

const errorMessage = (err: unknown, fallback: string): string => {
  if (err instanceof ConnectError) return err.rawMessage || err.message;
  if (err instanceof Error) return err.message;
  return fallback;
};

// =============================================================================
// Store
// =============================================================================

interface ScheduleState {
  schedules: WorkflowSchedule[];
  selectedScheduleId: string | null;
  isLoading: boolean;
  error: string | null;
  occurrences: ScheduleOccurrence[];
  aggregates: Record<string, ScheduleAggregate>;
  isLoadingOccurrences: boolean;

  fetchSchedules: (workflowId?: string) => Promise<void>;
  fetchSchedulesByWorkflow: (workflowId: string) => Promise<void>;
  fetchOccurrences: (start: Date, end: Date, workflowId?: string) => Promise<void>;
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
  occurrences: [] as ScheduleOccurrence[],
  aggregates: {} as Record<string, ScheduleAggregate>,
  isLoadingOccurrences: false,
};

export const useScheduleStore = create<ScheduleState>((set) => ({
  ...initialState,

  fetchSchedules: async (workflowId?: string) => {
    set({ isLoading: true, error: null });
    try {
      const res = await schedulesClient.list({ workflowId: workflowId ?? '' });
      set({ schedules: res.schedules.map(mapSchedule), isLoading: false });
    } catch (err) {
      const message = errorMessage(err, 'Unable to load schedules');
      logger.error('Failed to load schedules', { component: 'ScheduleStore', action: 'fetchSchedules' }, err);
      set({ error: message, isLoading: false });
    }
  },

  fetchSchedulesByWorkflow: async (workflowId: string) => {
    set({ isLoading: true, error: null });
    try {
      const res = await schedulesClient.listByWorkflow({ workflowId });
      set({ schedules: res.schedules.map(mapSchedule), isLoading: false });
    } catch (err) {
      const message = errorMessage(err, 'Unable to load workflow schedules');
      logger.error('Failed to load workflow schedules', { component: 'ScheduleStore', action: 'fetchSchedulesByWorkflow' }, err);
      set({ error: message, isLoading: false });
    }
  },

  fetchOccurrences: async (start: Date, end: Date, workflowId?: string) => {
    set({ isLoadingOccurrences: true });
    try {
      const res = await schedulesClient.occurrences({
        start: timestampFromDate(start),
        end: timestampFromDate(end),
        workflowId: workflowId ?? '',
      });
      const aggregates: Record<string, ScheduleAggregate> = {};
      for (const [k, v] of Object.entries(res.aggregates)) {
        aggregates[k] = mapAggregate(v as PbScheduleAggregate);
      }
      set({
        occurrences: res.occurrences.map(mapOccurrence),
        aggregates,
        isLoadingOccurrences: false,
      });
    } catch (err) {
      const message = errorMessage(err, 'Unable to load schedule occurrences');
      logger.error('Failed to load schedule occurrences', { component: 'ScheduleStore', action: 'fetchOccurrences' }, err);
      set({ occurrences: [], aggregates: {}, isLoadingOccurrences: false, error: message });
    }
  },

  createSchedule: async (workflowId, input) => {
    set({ isLoading: true, error: null });
    try {
      const res = await schedulesClient.create({
        workflowId,
        name: input.name,
        description: input.description ?? '',
        cronExpression: input.cron_expression,
        timezone: input.timezone ?? '',
        parameters: input.parameters as never,
        isActive: input.is_active,
      });
      if (!res.schedule) throw new Error('Server omitted schedule');
      const created = mapSchedule(res.schedule);
      set((state) => ({ schedules: [created, ...state.schedules], isLoading: false }));
      return created;
    } catch (err) {
      const message = errorMessage(err, 'Unable to create schedule');
      logger.error('Failed to create schedule', { component: 'ScheduleStore', action: 'createSchedule' }, err);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  updateSchedule: async (scheduleId, input) => {
    set({ isLoading: true, error: null });
    try {
      const res = await schedulesClient.update({
        scheduleId,
        name: input.name,
        description: input.description,
        cronExpression: input.cron_expression,
        timezone: input.timezone,
        parameters: input.parameters as never,
        isActive: input.is_active,
      });
      if (!res.schedule) throw new Error('Server omitted schedule');
      const updated = mapSchedule(res.schedule);
      set((state) => ({
        schedules: state.schedules.map((s) => (s.id === scheduleId ? updated : s)),
        isLoading: false,
      }));
      return updated;
    } catch (err) {
      const message = errorMessage(err, 'Unable to update schedule');
      logger.error('Failed to update schedule', { component: 'ScheduleStore', action: 'updateSchedule' }, err);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  deleteSchedule: async (scheduleId) => {
    set({ isLoading: true, error: null });
    try {
      await schedulesClient.delete({ scheduleId });
      set((state) => ({
        schedules: state.schedules.filter((s) => s.id !== scheduleId),
        selectedScheduleId: state.selectedScheduleId === scheduleId ? null : state.selectedScheduleId,
        isLoading: false,
      }));
      return true;
    } catch (err) {
      const message = errorMessage(err, 'Unable to delete schedule');
      logger.error('Failed to delete schedule', { component: 'ScheduleStore', action: 'deleteSchedule' }, err);
      set({ error: message, isLoading: false });
      return false;
    }
  },

  toggleSchedule: async (scheduleId) => {
    set({ isLoading: true, error: null });
    try {
      const res = await schedulesClient.toggle({ scheduleId });
      if (!res.schedule) throw new Error('Server omitted schedule');
      const updated = mapSchedule(res.schedule);
      set((state) => ({
        schedules: state.schedules.map((s) => (s.id === scheduleId ? updated : s)),
        isLoading: false,
      }));
      return updated;
    } catch (err) {
      const message = errorMessage(err, 'Unable to toggle schedule');
      logger.error('Failed to toggle schedule', { component: 'ScheduleStore', action: 'toggleSchedule' }, err);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  triggerSchedule: async (scheduleId) => {
    set({ isLoading: true, error: null });
    try {
      const res = await schedulesClient.trigger({ scheduleId });
      const triggeredAt = protoTimestampToISOString(res.triggeredAt);
      set((state) => ({
        schedules: state.schedules.map((s) =>
          s.id === scheduleId ? { ...s, last_run_at: triggeredAt, last_run_status: 'running' } : s,
        ),
        isLoading: false,
      }));
      return { execution_id: res.executionId };
    } catch (err) {
      const message = errorMessage(err, 'Unable to trigger schedule');
      logger.error('Failed to trigger schedule', { component: 'ScheduleStore', action: 'triggerSchedule' }, err);
      set({ error: message, isLoading: false });
      return null;
    }
  },

  selectSchedule: (scheduleId) => set({ selectedScheduleId: scheduleId }),
  clearError: () => set({ error: null }),
  reset: () => set(initialState),
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

export function describeCron(expr: string): string {
  const preset = CRON_PRESETS.find((p) => p.value === expr);
  if (preset) return preset.description;
  return `Cron: ${expr}`;
}
