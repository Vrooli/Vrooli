import { useCallback, useEffect, useState, useMemo } from 'react';
import {
  Calendar,
  Clock,
  Play,
  Pause,
  Trash2,
  Edit2,
  PlayCircle,
  RefreshCw,
  Plus,
  ChevronRight,
  AlertCircle,
} from 'lucide-react';
import { useScheduleStore, formatNextRun, describeCron } from '@stores/scheduleStore';
import type { WorkflowSchedule, CreateScheduleInput, UpdateScheduleInput } from '@stores/scheduleStore';
import { useWorkflowStore } from '@stores/workflowStore';
import { ScheduleModal } from './ScheduleModal';

export function SchedulesTab() {
  const { workflows, loadWorkflows } = useWorkflowStore();
  const {
    schedules,
    isLoading,
    error,
    fetchSchedules,
    createSchedule,
    updateSchedule,
    deleteSchedule,
    toggleSchedule,
    triggerSchedule,
    clearError,
  } = useScheduleStore();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<WorkflowSchedule | null>(null);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string | null>(null);
  const [triggeringId, setTriggeringId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  // Load schedules and workflows on mount
  useEffect(() => {
    void fetchSchedules();
    void loadWorkflows();
  }, [fetchSchedules, loadWorkflows]);

  const sortedSchedules = useMemo(() => {
    return [...schedules].sort((a, b) => {
      // Active schedules first
      if (a.is_active !== b.is_active) {
        return a.is_active ? -1 : 1;
      }
      // Then by next run time
      const aNext = a.next_run_at ? new Date(a.next_run_at).getTime() : Infinity;
      const bNext = b.next_run_at ? new Date(b.next_run_at).getTime() : Infinity;
      return aNext - bNext;
    });
  }, [schedules]);

  const handleOpenCreate = useCallback((workflowId?: string) => {
    setEditingSchedule(null);
    setSelectedWorkflowId(workflowId ?? null);
    setIsModalOpen(true);
  }, []);

  const handleOpenEdit = useCallback((schedule: WorkflowSchedule) => {
    setEditingSchedule(schedule);
    setSelectedWorkflowId(schedule.workflow_id);
    setIsModalOpen(true);
  }, []);

  const handleCloseModal = useCallback(() => {
    setIsModalOpen(false);
    setEditingSchedule(null);
    setSelectedWorkflowId(null);
  }, []);

  const handleSave = useCallback(async (input: CreateScheduleInput | UpdateScheduleInput, workflowId: string) => {
    if (editingSchedule) {
      await updateSchedule(editingSchedule.id, input);
    } else if (workflowId) {
      await createSchedule(workflowId, input as CreateScheduleInput);
    }
  }, [editingSchedule, createSchedule, updateSchedule]);

  const handleToggle = useCallback(async (schedule: WorkflowSchedule) => {
    await toggleSchedule(schedule.id);
  }, [toggleSchedule]);

  const handleTrigger = useCallback(async (schedule: WorkflowSchedule) => {
    setTriggeringId(schedule.id);
    try {
      await triggerSchedule(schedule.id);
    } finally {
      setTriggeringId(null);
    }
  }, [triggerSchedule]);

  const handleDelete = useCallback(async (schedule: WorkflowSchedule) => {
    const confirmed = window.confirm(`Delete schedule "${schedule.name}"? This cannot be undone.`);
    if (!confirmed) return;

    setDeletingId(schedule.id);
    try {
      await deleteSchedule(schedule.id);
    } finally {
      setDeletingId(null);
    }
  }, [deleteSchedule]);

  const getWorkflowName = useCallback((workflowId: string) => {
    const workflow = workflows.find(w => w.id === workflowId);
    return workflow?.name ?? 'Unknown Workflow';
  }, [workflows]);

  const formatLastRun = useCallback((lastRunAt?: string, lastRunStatus?: string) => {
    if (!lastRunAt) return 'Never';
    const date = new Date(lastRunAt);
    const timeStr = date.toLocaleString();
    if (lastRunStatus) {
      return `${timeStr} (${lastRunStatus})`;
    }
    return timeStr;
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Workflow Schedules
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Automate your workflows to run on a schedule.
          </p>
        </div>
        <div className="flex gap-2">
          {workflows.length > 0 && (
            <button
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-60"
              onClick={() => handleOpenCreate()}
              disabled={isLoading}
            >
              <Plus size={16} />
              New Schedule
            </button>
          )}
          <button
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-800 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-60"
            onClick={() => void fetchSchedules()}
            disabled={isLoading}
          >
            <RefreshCw size={16} className={isLoading ? 'animate-spin' : ''} />
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="flex items-center justify-between rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/40 dark:text-red-200">
          <span className="flex items-center gap-2">
            <AlertCircle size={16} />
            {error}
          </span>
          <button
            onClick={clearError}
            className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-200"
          >
            Dismiss
          </button>
        </div>
      )}

      {workflows.length === 0 && (
        <div className="rounded-lg border border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-900/30 p-4">
          <p className="text-sm text-yellow-800 dark:text-yellow-200">
            Create a workflow first to set up schedules.
          </p>
        </div>
      )}

      <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
        <div className="flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-700">
          <span className="text-sm text-gray-700 dark:text-gray-200">
            {isLoading
              ? 'Loading schedules...'
              : `${schedules.length} schedule${schedules.length === 1 ? '' : 's'}`}
          </span>
        </div>

        {sortedSchedules.length === 0 && !isLoading ? (
          <div className="px-4 py-8 text-center">
            <Calendar size={48} className="mx-auto mb-3 text-gray-300 dark:text-gray-600" />
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-1">
              No schedules yet
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Create a schedule to automate your workflow executions
            </p>
          </div>
        ) : (
          <ul className="divide-y divide-gray-200 dark:divide-gray-800">
            {sortedSchedules.map((schedule) => (
              <li
                key={schedule.id}
                className="px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <span
                        className={`inline-flex items-center gap-1 text-sm font-semibold ${
                          schedule.is_active
                            ? 'text-gray-900 dark:text-gray-100'
                            : 'text-gray-500 dark:text-gray-400'
                        }`}
                      >
                        {schedule.name}
                      </span>
                      <span
                        className={`text-[10px] px-2 py-0.5 rounded-full ${
                          schedule.is_active
                            ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-200'
                            : 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
                        }`}
                      >
                        {schedule.is_active ? 'Active' : 'Paused'}
                      </span>
                    </div>

                    {schedule.workflow_name && (
                      <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400 mb-1">
                        <ChevronRight size={12} />
                        <span>{schedule.workflow_name}</span>
                      </div>
                    )}

                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                      <span className="flex items-center gap-1" title={`Cron: ${schedule.cron_expression}`}>
                        <Clock size={12} />
                        {describeCron(schedule.cron_expression)}
                      </span>
                      <span title="Next run">
                        Next: {formatNextRun(schedule.next_run_at, schedule.next_run_human)}
                      </span>
                      <span title="Last run">
                        Last: {formatLastRun(schedule.last_run_at, schedule.last_run_status)}
                      </span>
                    </div>

                    {schedule.description && (
                      <p className="mt-1 text-xs text-gray-400 dark:text-gray-500 truncate">
                        {schedule.description}
                      </p>
                    )}
                  </div>

                  <div className="flex items-center gap-1 flex-shrink-0">
                    <button
                      onClick={() => void handleTrigger(schedule)}
                      disabled={triggeringId === schedule.id || !schedule.is_active}
                      className="p-1.5 text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded disabled:opacity-40 disabled:cursor-not-allowed"
                      title="Run now"
                    >
                      <PlayCircle size={18} className={triggeringId === schedule.id ? 'animate-pulse' : ''} />
                    </button>

                    <button
                      onClick={() => void handleToggle(schedule)}
                      className={`p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-800 ${
                        schedule.is_active
                          ? 'text-yellow-600 hover:text-yellow-800 dark:text-yellow-400 dark:hover:text-yellow-200'
                          : 'text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-200'
                      }`}
                      title={schedule.is_active ? 'Pause' : 'Resume'}
                    >
                      {schedule.is_active ? <Pause size={18} /> : <Play size={18} />}
                    </button>

                    <button
                      onClick={() => handleOpenEdit(schedule)}
                      className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
                      title="Edit"
                    >
                      <Edit2 size={18} />
                    </button>

                    <button
                      onClick={() => void handleDelete(schedule)}
                      disabled={deletingId === schedule.id}
                      className="p-1.5 text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-200 hover:bg-red-50 dark:hover:bg-red-900/30 rounded disabled:opacity-40"
                      title="Delete"
                    >
                      <Trash2 size={18} />
                    </button>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      <ScheduleModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onSave={handleSave}
        schedule={editingSchedule}
        workflowId={selectedWorkflowId ?? undefined}
        workflowName={selectedWorkflowId ? getWorkflowName(selectedWorkflowId) : undefined}
        workflows={workflows.map(w => ({ id: w.id, name: w.name }))}
      />
    </div>
  );
}
