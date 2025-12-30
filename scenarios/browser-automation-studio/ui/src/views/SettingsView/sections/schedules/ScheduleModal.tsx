import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { X, Clock, Globe, AlertCircle, ChevronDown, Workflow, Link } from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import type { WorkflowSchedule, CreateScheduleInput, UpdateScheduleInput } from '@stores/scheduleStore';
import { CRON_PRESETS, COMMON_TIMEZONES, describeCron } from '@stores/scheduleStore';

interface WorkflowOption {
  id: string;
  name: string;
}

interface ScheduleModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (input: CreateScheduleInput | UpdateScheduleInput, workflowId: string) => Promise<void>;
  schedule?: WorkflowSchedule | null;
  workflowId?: string;
  workflowName?: string;
  workflows?: WorkflowOption[];
}

// Stable empty array for default workflows prop
const EMPTY_WORKFLOWS: WorkflowOption[] = [];

export function ScheduleModal({
  isOpen,
  onClose,
  onSave,
  schedule,
  workflowId: initialWorkflowId,
  workflowName,
  workflows = EMPTY_WORKFLOWS,
}: ScheduleModalProps) {
  const isEditing = Boolean(schedule);

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string>(initialWorkflowId ?? '');
  const [cronExpression, setCronExpression] = useState('0 9 * * *');
  const [timezone, setTimezone] = useState('UTC');
  const [isActive, setIsActive] = useState(true);
  const [usePreset, setUsePreset] = useState(true);
  const [selectedPreset, setSelectedPreset] = useState<string | null>('0 9 * * *');
  const [startUrl, setStartUrl] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  // Track if we've initialized the form for the current open state
  const wasOpenRef = useRef(false);

  // Reset form only when modal first opens (transition from closed to open)
  useEffect(() => {
    // Only reset when transitioning from closed to open
    if (isOpen && !wasOpenRef.current) {
      wasOpenRef.current = true;
      if (schedule) {
        setName(schedule.name);
        setDescription(schedule.description ?? '');
        setSelectedWorkflowId(schedule.workflow_id);
        setCronExpression(schedule.cron_expression);
        setTimezone(schedule.timezone);
        setIsActive(schedule.is_active);
        // Load start_url from parameters if present
        setStartUrl((schedule.parameters?.start_url as string) ?? '');
        // Check if current expression matches a preset
        const matchingPreset = CRON_PRESETS.find(p => p.value === schedule.cron_expression);
        if (matchingPreset) {
          setUsePreset(true);
          setSelectedPreset(matchingPreset.value);
        } else {
          setUsePreset(false);
          setSelectedPreset(null);
        }
      } else {
        // New schedule defaults
        setName('');
        setDescription('');
        setSelectedWorkflowId(initialWorkflowId ?? (workflows.length === 1 ? workflows[0].id : ''));
        setCronExpression('0 9 * * *');
        setTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC');
        setIsActive(true);
        setUsePreset(true);
        setSelectedPreset('0 9 * * *');
        setStartUrl('');
      }
      setError(null);
    } else if (!isOpen) {
      // Reset the ref when modal closes so we reinitialize next time
      wasOpenRef.current = false;
    }
  }, [isOpen, schedule, initialWorkflowId, workflows]);

  const handlePresetChange = useCallback((presetValue: string) => {
    setSelectedPreset(presetValue);
    setCronExpression(presetValue);
    setError(null);
  }, []);

  const handleCustomCronChange = useCallback((value: string) => {
    setCronExpression(value);
    setSelectedPreset(null);
    setError(null);
  }, []);

  const cronDescription = useMemo(() => {
    return describeCron(cronExpression);
  }, [cronExpression]);

  const handleSubmit = useCallback(async () => {
    // Validation
    if (!isEditing && !selectedWorkflowId) {
      setError('Please select a workflow');
      return;
    }
    if (!name.trim()) {
      setError('Schedule name is required');
      return;
    }
    if (!cronExpression.trim()) {
      setError('Cron expression is required');
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      // Build parameters object with start_url if provided
      const parameters: Record<string, unknown> | undefined = startUrl.trim()
        ? { start_url: startUrl.trim() }
        : undefined;

      const input: CreateScheduleInput | UpdateScheduleInput = {
        name: name.trim(),
        description: description.trim() || undefined,
        cron_expression: cronExpression.trim(),
        timezone,
        is_active: isActive,
        parameters,
      };

      await onSave(input, selectedWorkflowId);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save schedule';
      setError(message);
    } finally {
      setIsSaving(false);
    }
  }, [name, description, cronExpression, timezone, isActive, startUrl, selectedWorkflowId, isEditing, onSave, onClose]);

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel={isEditing ? 'Edit schedule' : 'Create schedule'}
      size="wide"
      className="!p-0"
    >
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {isEditing ? 'Edit Schedule' : 'Create Schedule'}
            </h2>
            {workflowName && (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                For workflow: {workflowName}
              </p>
            )}
          </div>
          <button
            onClick={onClose}
            className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-5">
          {error && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800">
              <AlertCircle size={18} className="text-red-600 dark:text-red-400 flex-shrink-0" />
              <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
            </div>
          )}

          {/* Workflow Selector - only show when creating and no workflow pre-selected */}
          {!isEditing && workflows.length > 0 && !initialWorkflowId && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                <span className="flex items-center gap-1.5">
                  <Workflow size={14} />
                  Workflow *
                </span>
              </label>
              <div className="relative">
                <select
                  value={selectedWorkflowId}
                  onChange={(e) => setSelectedWorkflowId(e.target.value)}
                  className="w-full px-3 py-2 pr-10 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 appearance-none"
                >
                  <option value="">Select a workflow...</option>
                  {workflows.map((wf) => (
                    <option key={wf.id} value={wf.id}>
                      {wf.name}
                    </option>
                  ))}
                </select>
                <ChevronDown size={18} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
              </div>
            </div>
          )}

          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
              Schedule Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Daily morning check"
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description..."
              rows={2}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 resize-none"
            />
          </div>

          {/* Start URL */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
              <span className="flex items-center gap-1.5">
                <Link size={14} />
                Start URL
              </span>
            </label>
            <input
              type="url"
              value={startUrl}
              onChange={(e) => setStartUrl(e.target.value)}
              placeholder="https://example.com"
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Required if the workflow doesn't start with a Navigate step
            </p>
          </div>

          {/* Cron Expression */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                Schedule Frequency *
              </label>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setUsePreset(true)}
                  className={`px-2 py-1 text-xs rounded ${
                    usePreset
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300'
                      : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                  }`}
                >
                  Presets
                </button>
                <button
                  type="button"
                  onClick={() => setUsePreset(false)}
                  className={`px-2 py-1 text-xs rounded ${
                    !usePreset
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300'
                      : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                  }`}
                >
                  Custom
                </button>
              </div>
            </div>

            {usePreset ? (
              <div className="relative">
                <select
                  value={selectedPreset ?? ''}
                  onChange={(e) => handlePresetChange(e.target.value)}
                  className="w-full px-3 py-2 pr-10 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 appearance-none"
                >
                  {CRON_PRESETS.map((preset) => (
                    <option key={preset.value} value={preset.value}>
                      {preset.label}
                    </option>
                  ))}
                </select>
                <ChevronDown size={18} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
              </div>
            ) : (
              <input
                type="text"
                value={cronExpression}
                onChange={(e) => handleCustomCronChange(e.target.value)}
                placeholder="* * * * *"
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 font-mono text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            )}

            <p className="mt-1.5 text-sm text-gray-500 dark:text-gray-400 flex items-center gap-1.5">
              <Clock size={14} />
              {cronDescription}
            </p>
          </div>

          {/* Timezone */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
              <span className="flex items-center gap-1.5">
                <Globe size={14} />
                Timezone
              </span>
            </label>
            <div className="relative">
              <select
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
                className="w-full px-3 py-2 pr-10 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 appearance-none"
              >
                {COMMON_TIMEZONES.map((tz) => (
                  <option key={tz} value={tz}>
                    {tz.replace(/_/g, ' ')}
                  </option>
                ))}
              </select>
              <ChevronDown size={18} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
            </div>
          </div>

          {/* Active Toggle */}
          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Active
              </label>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Schedule will run automatically when active
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={isActive}
              onClick={() => setIsActive(!isActive)}
              className={`relative w-11 h-6 rounded-full transition-colors ${
                isActive ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'
              }`}
            >
              <span
                className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${
                  isActive ? 'translate-x-5' : 'translate-x-0'
                }`}
              />
            </button>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={isSaving}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {isSaving ? 'Saving...' : isEditing ? 'Update Schedule' : 'Create Schedule'}
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
