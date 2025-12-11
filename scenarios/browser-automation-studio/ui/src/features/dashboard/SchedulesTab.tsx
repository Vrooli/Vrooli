import React, { useEffect, useState, useCallback } from 'react';
import {
  CalendarClock,
  Play,
  Pause,
  PlayCircle,
  Trash2,
  Plus,
  RefreshCw,
  Edit2,
  Clock,
  Loader2,
  XCircle,
  ChevronRight,
  X,
  CalendarDays,
  Globe2,
} from 'lucide-react';
import {
  useScheduleStore,
  type WorkflowSchedule,
  type CreateScheduleInput,
  type UpdateScheduleInput,
  formatNextRun,
  describeCron,
} from '@stores/scheduleStore';
import { useWorkflowStore } from '@stores/workflowStore';
import { TabEmptyState } from './TabEmptyState';
import { ScheduleModal } from '@features/settings/components/schedules';

interface SchedulesTabProps {
  onNavigateToWorkflow?: (workflowId: string) => void;
  onNavigateToExecution?: (executionId: string, workflowId: string) => void;
  onNavigateToHome?: () => void;
}

// Schedule card component
const ScheduleCard: React.FC<{
  schedule: WorkflowSchedule;
  onToggle: () => void;
  onTrigger: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onViewWorkflow?: () => void;
  isLoading?: boolean;
}> = ({ schedule, onToggle, onTrigger, onEdit, onDelete, onViewWorkflow, isLoading }) => {
  const statusColor = schedule.is_active
    ? 'bg-green-500/10 border-green-500/30 text-green-400'
    : 'bg-gray-500/10 border-gray-500/30 text-gray-400';

  const nextRunLabel = schedule.is_active
    ? formatNextRun(schedule.next_run_at, schedule.next_run_human)
    : 'Paused';

  return (
    <div className="group p-4 rounded-lg border border-gray-700/50 bg-gray-800/50 hover:border-gray-600 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 min-w-0 flex-1">
          <div className={`p-2 rounded-lg ${statusColor}`}>
            <CalendarClock size={18} />
          </div>
          <div className="min-w-0 flex-1">
            <div className="font-medium text-surface truncate">{schedule.name}</div>
            {schedule.workflow_name && (
              <button
                onClick={onViewWorkflow}
                className="text-xs text-gray-500 hover:text-gray-300 truncate flex items-center gap-1 transition-colors"
              >
                {schedule.workflow_name}
                <ChevronRight size={12} />
              </button>
            )}
            {schedule.description && (
              <div className="text-xs text-gray-500 mt-1 line-clamp-1">
                {schedule.description}
              </div>
            )}
            <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
              <span className="flex items-center gap-1">
                <Clock size={12} />
                {describeCron(schedule.cron_expression)}
              </span>
              <span className="flex items-center gap-1">
                <Globe2 size={12} />
                {schedule.timezone}
              </span>
            </div>
            <div className="flex items-center gap-2 mt-2">
              <span
                className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium ${
                  schedule.is_active
                    ? 'bg-green-500/10 text-green-400'
                    : 'bg-gray-500/10 text-gray-400'
                }`}
              >
                {schedule.is_active ? (
                  <>
                    <div className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
                    Active
                  </>
                ) : (
                  <>
                    <Pause size={10} />
                    Paused
                  </>
                )}
              </span>
              <span className="text-xs text-gray-500">{nextRunLabel}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={onTrigger}
            disabled={isLoading}
            className="p-2 text-blue-400 hover:text-blue-300 hover:bg-blue-500/10 rounded-lg transition-colors disabled:opacity-50"
            title="Run now"
          >
            <PlayCircle size={16} />
          </button>
          <button
            onClick={onToggle}
            disabled={isLoading}
            className={`p-2 rounded-lg transition-colors disabled:opacity-50 ${
              schedule.is_active
                ? 'text-amber-400 hover:text-amber-300 hover:bg-amber-500/10'
                : 'text-green-400 hover:text-green-300 hover:bg-green-500/10'
            }`}
            title={schedule.is_active ? 'Pause schedule' : 'Resume schedule'}
          >
            {schedule.is_active ? <Pause size={16} /> : <Play size={16} />}
          </button>
          <button
            onClick={onEdit}
            disabled={isLoading}
            className="p-2 text-gray-400 hover:text-gray-300 hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
            title="Edit schedule"
          >
            <Edit2 size={16} />
          </button>
          <button
            onClick={onDelete}
            disabled={isLoading}
            className="p-2 text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-colors disabled:opacity-50"
            title="Delete schedule"
          >
            <Trash2 size={16} />
          </button>
        </div>
      </div>
    </div>
  );
};

// Empty state preview component
const SchedulesEmptyPreview: React.FC = () => {
  return (
    <div className="space-y-3">
      {[
        { name: 'Daily health check', time: '9:00 AM', status: 'active' },
        { name: 'Weekly report', time: 'Monday 8:00 AM', status: 'active' },
        { name: 'Monthly cleanup', time: '1st of month', status: 'paused' },
      ].map((item, idx) => (
        <div
          key={idx}
          className="flex items-center gap-3 p-3 rounded-lg bg-gray-800/50 border border-gray-700/50"
        >
          <div
            className={`p-1.5 rounded-lg ${
              item.status === 'active'
                ? 'bg-green-500/10 text-green-400'
                : 'bg-gray-500/10 text-gray-400'
            }`}
          >
            <CalendarClock size={14} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm text-surface truncate">{item.name}</div>
            <div className="text-xs text-gray-500">{item.time}</div>
          </div>
          <div
            className={`w-2 h-2 rounded-full ${
              item.status === 'active' ? 'bg-green-400 animate-pulse' : 'bg-gray-500'
            }`}
          />
        </div>
      ))}
    </div>
  );
};

export const SchedulesTab: React.FC<SchedulesTabProps> = ({
  onNavigateToWorkflow,
  onNavigateToExecution,
  onNavigateToHome,
}) => {
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

  const { workflows, loadWorkflows } = useWorkflowStore();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<WorkflowSchedule | null>(null);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  useEffect(() => {
    void fetchSchedules();
    void loadWorkflows();
  }, [fetchSchedules, loadWorkflows]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await fetchSchedules();
    setIsRefreshing(false);
  }, [fetchSchedules]);

  const handleCreateSchedule = () => {
    setEditingSchedule(null);
    setSelectedWorkflowId(null);
    setIsModalOpen(true);
  };

  const handleEditSchedule = (schedule: WorkflowSchedule) => {
    setEditingSchedule(schedule);
    setSelectedWorkflowId(schedule.workflow_id);
    setIsModalOpen(true);
  };

  const handleCloseModal = useCallback(() => {
    setIsModalOpen(false);
    setEditingSchedule(null);
    setSelectedWorkflowId(null);
  }, []);

  const handleSaveSchedule = async (input: CreateScheduleInput | UpdateScheduleInput, workflowId: string) => {
    if (editingSchedule) {
      await updateSchedule(editingSchedule.id, input as UpdateScheduleInput);
    } else if (workflowId) {
      await createSchedule(workflowId, input as CreateScheduleInput);
    }
  };

  const handleToggleSchedule = async (scheduleId: string) => {
    await toggleSchedule(scheduleId);
  };

  const handleTriggerSchedule = async (scheduleId: string) => {
    const result = await triggerSchedule(scheduleId);
    if (result?.execution_id && onNavigateToExecution) {
      const schedule = schedules.find((s) => s.id === scheduleId);
      if (schedule) {
        onNavigateToExecution(result.execution_id, schedule.workflow_id);
      }
    }
  };

  const handleDeleteSchedule = async (scheduleId: string) => {
    await deleteSchedule(scheduleId);
    setDeleteConfirm(null);
  };

  const activeSchedules = schedules.filter((s) => s.is_active);
  const inactiveSchedules = schedules.filter((s) => !s.is_active);

  return (
    <div className="space-y-6">
      {/* Header with stats */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between bg-gradient-to-r from-gray-900/80 via-gray-900 to-purple-900/10 border border-gray-800/80 rounded-2xl p-4">
        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-green-500/10 border border-green-500/30 text-green-100">
            <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            {activeSchedules.length} active
          </div>
          <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-gray-700/50 text-gray-100 border border-gray-700">
            <Pause size={14} />
            {inactiveSchedules.length} paused
          </div>
          <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-gray-700/50 text-gray-100 border border-gray-700">
            <CalendarDays size={14} />
            {schedules.length} total
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleRefresh}
            disabled={isRefreshing || isLoading}
            className="flex items-center gap-2 px-3 py-1.5 text-sm text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
          >
            <RefreshCw size={14} className={isRefreshing ? 'animate-spin' : ''} />
            Refresh
          </button>
          <button
            onClick={handleCreateSchedule}
            className="hero-button-primary flex items-center gap-2"
          >
            <Plus size={16} />
            New Schedule
          </button>
        </div>
      </div>

      {/* Error display */}
      {error && (
        <div className="flex items-center justify-between p-3 bg-red-500/10 border border-red-500/30 rounded-lg">
          <div className="flex items-center gap-2 text-red-400 text-sm">
            <XCircle size={16} />
            {error}
          </div>
          <button
            onClick={clearError}
            className="text-red-400 hover:text-red-300 transition-colors"
          >
            <X size={16} />
          </button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && schedules.length === 0 ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 size={24} className="animate-spin text-gray-500" />
        </div>
      ) : schedules.length === 0 ? (
        // Empty state
        <TabEmptyState
          icon={<CalendarClock size={22} />}
          title="Automate on autopilot"
          subtitle="Schedule workflows to run automatically at specific times or intervals."
          preview={<SchedulesEmptyPreview />}
          primaryCta={{
            label: 'Create your first schedule',
            onClick: handleCreateSchedule,
          }}
          secondaryCta={
            onNavigateToHome
              ? { label: 'Browse workflows', onClick: onNavigateToHome }
              : undefined
          }
          progressPath={[
            { label: 'Create workflow', completed: true },
            { label: 'Add schedule', active: true },
            { label: 'Run automatically' },
          ]}
          features={[
            {
              title: 'Cron scheduling',
              description: 'Use flexible cron expressions or simple presets.',
              icon: <Clock size={16} />,
            },
            {
              title: 'Timezone support',
              description: 'Schedule in any timezone, run reliably.',
              icon: <Globe2 size={16} />,
            },
            {
              title: 'Manual triggers',
              description: 'Run scheduled workflows anytime with one click.',
              icon: <PlayCircle size={16} />,
            },
          ]}
        />
      ) : (
        // Schedule list
        <div className="space-y-6">
          {/* Active schedules */}
          {activeSchedules.length > 0 && (
            <div className="space-y-3">
              <h3 className="text-sm font-medium text-gray-300 flex items-center gap-2">
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                Active Schedules ({activeSchedules.length})
              </h3>
              <div className="space-y-2">
                {activeSchedules.map((schedule) => (
                  <ScheduleCard
                    key={schedule.id}
                    schedule={schedule}
                    onToggle={() => handleToggleSchedule(schedule.id)}
                    onTrigger={() => handleTriggerSchedule(schedule.id)}
                    onEdit={() => handleEditSchedule(schedule)}
                    onDelete={() => setDeleteConfirm(schedule.id)}
                    onViewWorkflow={
                      onNavigateToWorkflow
                        ? () => onNavigateToWorkflow(schedule.workflow_id)
                        : undefined
                    }
                    isLoading={isLoading}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Inactive schedules */}
          {inactiveSchedules.length > 0 && (
            <div className="space-y-3">
              <h3 className="text-sm font-medium text-gray-400">
                Paused Schedules ({inactiveSchedules.length})
              </h3>
              <div className="space-y-2">
                {inactiveSchedules.map((schedule) => (
                  <ScheduleCard
                    key={schedule.id}
                    schedule={schedule}
                    onToggle={() => handleToggleSchedule(schedule.id)}
                    onTrigger={() => handleTriggerSchedule(schedule.id)}
                    onEdit={() => handleEditSchedule(schedule)}
                    onDelete={() => setDeleteConfirm(schedule.id)}
                    onViewWorkflow={
                      onNavigateToWorkflow
                        ? () => onNavigateToWorkflow(schedule.workflow_id)
                        : undefined
                    }
                    isLoading={isLoading}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Schedule modal */}
      <ScheduleModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onSave={handleSaveSchedule}
        schedule={editingSchedule}
        workflowId={selectedWorkflowId ?? undefined}
        workflowName={selectedWorkflowId ? workflows.find(w => w.id === selectedWorkflowId)?.name : undefined}
        workflows={workflows.map(w => ({ id: w.id, name: w.name }))}
      />

      {/* Delete confirmation */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/60 backdrop-blur-sm"
            onClick={() => setDeleteConfirm(null)}
          />
          <div className="relative bg-gray-900 rounded-xl border border-gray-700 shadow-2xl p-6 max-w-sm mx-4">
            <h3 className="text-lg font-semibold text-surface mb-2">Delete Schedule?</h3>
            <p className="text-gray-400 text-sm mb-4">
              This will permanently delete this schedule. The workflow itself will not be affected.
            </p>
            <div className="flex items-center gap-3">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleDeleteSchedule(deleteConfirm)}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SchedulesTab;
