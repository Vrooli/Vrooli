/**
 * WorkflowCreationForm Component
 *
 * A full-panel form for creating a workflow from selected recorded actions.
 * Replaces the Live Preview panel when the user clicks "Create Workflow →".
 *
 * Features:
 * - Workflow name input
 * - Project selection dropdown
 * - Default session selection (for workflow execution)
 * - Selection summary (step count, ranges)
 * - Advanced settings (navigation, timeouts, viewport)
 * - Test button to validate before generating
 * - Generate button to create the workflow
 * - Back button to return to Live Preview
 */

import { useEffect, useMemo, useState, useCallback } from 'react';
import { ChevronDown, ChevronRight, Settings2, ArrowLeft, Play, Sparkles, CheckCircle2, XCircle, AlertTriangle, FolderTree, ChevronRight as ChevronRightIcon } from 'lucide-react';
import { useProjectStore, type Project } from '@/domains/projects';
import ProjectModal from '@/domains/projects/ProjectModal';
import type { RecordedAction, RecordingSessionProfile, ReplayPreviewResponse } from '../types/types';
import type { NavigationWaitUntil, WorkflowSettingsTyped } from '@/types/workflow';
import { ProjectPickerModal } from './ProjectPickerModal';

/** Describes a contiguous range of selected steps */
export interface SelectionRange {
  start: number;
  end: number;
  count: number;
}

/** Advanced workflow settings from the form */
export interface WorkflowAdvancedSettings {
  navigationWaitUntil: NavigationWaitUntil;
  actionTimeoutSeconds: number;
  viewportWidth: number;
  viewportHeight: number;
  continueOnError: boolean;
}

interface WorkflowCreationFormProps {
  /** All recorded actions (for context) */
  actions: RecordedAction[];
  /** Indices of selected actions (sorted) */
  selectedIndices: number[];
  /** Available session profiles for default session dropdown */
  sessionProfiles: RecordingSessionProfile[];
  /** Whether profiles are still loading */
  sessionProfilesLoading: boolean;
  /** Whether a test is in progress */
  isReplaying: boolean;
  /** Whether workflow generation is in progress */
  isGenerating: boolean;
  /** Callback to go back to Live Preview */
  onBack: () => void;
  /** Callback to run a test on selected actions */
  onTest: (actionIndices: number[]) => Promise<ReplayPreviewResponse>;
  /** Callback to generate workflow from selected actions */
  onGenerate: (params: {
    name: string;
    projectId: string;
    defaultSessionId: string | null;
    actionIndices: number[];
    settings?: WorkflowSettingsTyped;
  }) => Promise<void>;
  /** Count of actions with low confidence selectors */
  lowConfidenceCount: number;
  /** Count of actions with medium confidence selectors */
  mediumConfidenceCount: number;
}

// Default values for advanced settings
const DEFAULT_SETTINGS: WorkflowAdvancedSettings = {
  navigationWaitUntil: 'domcontentloaded',
  actionTimeoutSeconds: 30,
  viewportWidth: 1920,
  viewportHeight: 1080,
  continueOnError: false,
};

// Navigation wait options
const NAVIGATION_WAIT_OPTIONS: { value: NavigationWaitUntil; label: string; description: string }[] = [
  { value: 'domcontentloaded', label: 'DOM Ready', description: 'Wait for HTML to parse (fast, recommended)' },
  { value: 'load', label: 'Page Load', description: 'Wait for all resources to load' },
  { value: 'networkidle', label: 'Network Idle', description: 'Wait for no network activity (slow on heavy sites)' },
  { value: 'commit', label: 'Commit', description: 'Wait for first response byte (fastest)' },
];

/**
 * Compute contiguous ranges from sorted indices.
 * E.g., [0, 1, 2, 5, 6, 10] -> [{start: 0, end: 2}, {start: 5, end: 6}, {start: 10, end: 10}]
 */
export function computeSelectionRanges(indices: number[]): SelectionRange[] {
  if (indices.length === 0) return [];

  const sorted = [...indices].sort((a, b) => a - b);
  const ranges: SelectionRange[] = [];
  let rangeStart = sorted[0];
  let rangeEnd = sorted[0];

  for (let i = 1; i < sorted.length; i++) {
    if (sorted[i] === rangeEnd + 1) {
      rangeEnd = sorted[i];
    } else {
      ranges.push({ start: rangeStart, end: rangeEnd, count: rangeEnd - rangeStart + 1 });
      rangeStart = sorted[i];
      rangeEnd = sorted[i];
    }
  }
  ranges.push({ start: rangeStart, end: rangeEnd, count: rangeEnd - rangeStart + 1 });

  return ranges;
}

/**
 * Format ranges for display.
 * E.g., [{start: 0, end: 2}, {start: 5, end: 6}] -> "Steps 1-3, 6-7"
 */
function formatRanges(ranges: SelectionRange[]): string {
  if (ranges.length === 0) return 'No steps selected';
  if (ranges.length === 1) {
    const r = ranges[0];
    if (r.start === r.end) {
      return `Step ${r.start + 1}`;
    }
    return `Steps ${r.start + 1}-${r.end + 1}`;
  }
  return ranges
    .map((r) => (r.start === r.end ? `${r.start + 1}` : `${r.start + 1}-${r.end + 1}`))
    .join(', ');
}

export function WorkflowCreationForm({
  actions: _actions,
  selectedIndices,
  sessionProfiles,
  sessionProfilesLoading,
  isReplaying,
  isGenerating,
  onBack,
  onTest,
  onGenerate,
  lowConfidenceCount,
  mediumConfidenceCount,
}: WorkflowCreationFormProps) {
  const { projects, fetchProjects, getSmartDefaultProject } = useProjectStore();

  // Basic form state
  const [workflowName, setWorkflowName] = useState('');
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<ReplayPreviewResponse | null>(null);

  // Project picker modal state
  const [isProjectPickerOpen, setIsProjectPickerOpen] = useState(false);
  const [isProjectModalOpen, setIsProjectModalOpen] = useState(false);

  // Advanced settings state
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [advancedSettings, setAdvancedSettings] = useState<WorkflowAdvancedSettings>(DEFAULT_SETTINGS);

  // Fetch projects on mount
  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  // Set default project when projects load
  useEffect(() => {
    if (!selectedProject && projects.length > 0) {
      const defaultProject = getSmartDefaultProject();
      if (defaultProject) {
        setSelectedProject(defaultProject);
      }
    }
  }, [projects, selectedProject, getSmartDefaultProject]);

  // Compute selection info
  const selectionRanges = useMemo(() => computeSelectionRanges(selectedIndices), [selectedIndices]);
  const totalSelected = selectedIndices.length;

  // Check for unstable selectors in selection
  const hasUnstableSelectors = lowConfidenceCount > 0 || mediumConfidenceCount > 0;

  // Update a single advanced setting
  const updateSetting = useCallback(<K extends keyof WorkflowAdvancedSettings>(
    key: K,
    value: WorkflowAdvancedSettings[K]
  ) => {
    setAdvancedSettings(prev => ({ ...prev, [key]: value }));
  }, []);

  const handleTest = async () => {
    setError(null);
    setTestResults(null);
    try {
      const results = await onTest(selectedIndices);
      setTestResults(results);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Test failed');
    }
  };

  const handleGenerate = async () => {
    if (!workflowName.trim()) {
      setError('Please enter a workflow name');
      return;
    }
    if (!selectedProject) {
      setError('Please select a project');
      return;
    }
    if (selectedIndices.length === 0) {
      setError('No steps selected');
      return;
    }

    setError(null);
    try {
      // Convert form settings to WorkflowSettingsTyped
      const settings: WorkflowSettingsTyped = {
        navigation_wait_until: advancedSettings.navigationWaitUntil,
        entry_selector_timeout_ms: advancedSettings.actionTimeoutSeconds * 1000,
        viewport_width: advancedSettings.viewportWidth,
        viewport_height: advancedSettings.viewportHeight,
        continue_on_error: advancedSettings.continueOnError,
      };

      await onGenerate({
        name: workflowName.trim(),
        projectId: selectedProject.id,
        defaultSessionId: selectedSessionId,
        actionIndices: selectedIndices,
        settings,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate workflow');
    }
  };

  const isFormValid = workflowName.trim().length > 0 && selectedProject !== null && totalSelected > 0;

  // Handle project creation from ProjectModal
  const handleProjectCreated = useCallback((project: Project) => {
    setSelectedProject(project);
    setIsProjectModalOpen(false);
  }, []);

  return (
    <div className="flex flex-col h-full bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={onBack}
          className="flex items-center gap-1.5 px-2 py-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
        >
          <ArrowLeft size={16} />
          <span>Back</span>
        </button>
        <div className="flex-1" />
        <h2 className="text-sm font-semibold text-gray-900 dark:text-white">Create Workflow</h2>
        <div className="flex-1" />
        <div className="w-[72px]" /> {/* Spacer to center title */}
      </div>

      {/* Form Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-4">
          {/* Selection Summary Card */}
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
            <div className="flex items-center gap-3">
              <div className="flex-shrink-0 w-12 h-12 flex items-center justify-center bg-blue-50 dark:bg-blue-900/30 rounded-lg">
                <span className="text-xl font-bold text-blue-600 dark:text-blue-400">{totalSelected}</span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-white">
                  {totalSelected === 1 ? '1 step selected' : `${totalSelected} steps selected`}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                  {formatRanges(selectionRanges)}
                </p>
              </div>
            </div>
          </div>

          {/* Unstable Selectors Warning */}
          {hasUnstableSelectors && (
            <div className="flex items-start gap-3 p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
              <AlertTriangle size={18} className="text-amber-500 flex-shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-amber-800 dark:text-amber-200">
                  {lowConfidenceCount > 0
                    ? `${lowConfidenceCount} step${lowConfidenceCount > 1 ? 's have' : ' has'} unstable selectors`
                    : `${mediumConfidenceCount} step${mediumConfidenceCount > 1 ? 's have' : ' has'} potentially unstable selectors`}
                </p>
                <p className="text-xs text-amber-700 dark:text-amber-300 mt-0.5">
                  Consider editing selectors before generating.
                </p>
              </div>
            </div>
          )}

          {/* Main Form Card */}
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
            {/* Workflow Name */}
            <div className="p-4">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Workflow Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={workflowName}
                onChange={(e) => setWorkflowName(e.target.value)}
                placeholder="e.g., Login Flow, Submit Form"
                className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                autoFocus
              />
            </div>

            {/* Project Selection */}
            <div className="p-4">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Project <span className="text-red-500">*</span>
              </label>
              <button
                type="button"
                onClick={() => setIsProjectPickerOpen(true)}
                className="w-full flex items-center gap-3 px-3 py-2.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-left hover:bg-gray-50 dark:hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
              >
                <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${
                  selectedProject
                    ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-400'
                }`}>
                  <FolderTree size={16} />
                </div>
                <div className="flex-1 min-w-0">
                  {selectedProject ? (
                    <>
                      <div className="font-medium text-gray-900 dark:text-white truncate">
                        {selectedProject.name}
                      </div>
                      {selectedProject.description && (
                        <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
                          {selectedProject.description}
                        </div>
                      )}
                    </>
                  ) : (
                    <div className="text-gray-400 dark:text-gray-500">
                      Select a project...
                    </div>
                  )}
                </div>
                <ChevronRightIcon size={16} className="text-gray-400 flex-shrink-0" />
              </button>
            </div>

            {/* Default Session Selection */}
            <div className="p-4">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Default Session
              </label>
              <select
                value={selectedSessionId || ''}
                onChange={(e) => setSelectedSessionId(e.target.value || null)}
                disabled={sessionProfilesLoading}
                className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
              >
                <option value="">None (create new each run)</option>
                {sessionProfiles.map((profile) => (
                  <option key={profile.id} value={profile.id}>
                    {profile.name}
                    {profile.has_storage_state ? ' (Auth saved)' : ''}
                  </option>
                ))}
              </select>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                Use a saved browser session with existing login state.
              </p>
            </div>
          </div>

          {/* Advanced Settings Accordion */}
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="w-full flex items-center justify-between p-4 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
            >
              <div className="flex items-center gap-2">
                <Settings2 size={16} className="text-gray-500 dark:text-gray-400" />
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Advanced Settings</span>
              </div>
              {showAdvanced ? (
                <ChevronDown size={16} className="text-gray-400" />
              ) : (
                <ChevronRight size={16} className="text-gray-400" />
              )}
            </button>

            {showAdvanced && (
              <div className="border-t border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
                {/* Navigation Wait */}
                <div className="p-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Navigation Wait
                  </label>
                  <select
                    value={advancedSettings.navigationWaitUntil}
                    onChange={(e) => updateSetting('navigationWaitUntil', e.target.value as NavigationWaitUntil)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    {NAVIGATION_WAIT_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                    {NAVIGATION_WAIT_OPTIONS.find(o => o.value === advancedSettings.navigationWaitUntil)?.description}
                  </p>
                </div>

                {/* Action Timeout */}
                <div className="p-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Action Timeout
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min={1}
                      max={300}
                      value={advancedSettings.actionTimeoutSeconds}
                      onChange={(e) => updateSetting('actionTimeoutSeconds', Math.max(1, Math.min(300, parseInt(e.target.value) || 30)))}
                      className="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <span className="text-sm text-gray-500 dark:text-gray-400">seconds</span>
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                    How long to wait for elements before timing out.
                  </p>
                </div>

                {/* Viewport */}
                <div className="p-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Viewport Size
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min={320}
                      max={3840}
                      value={advancedSettings.viewportWidth}
                      onChange={(e) => updateSetting('viewportWidth', Math.max(320, Math.min(3840, parseInt(e.target.value) || 1920)))}
                      className="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <span className="text-sm text-gray-500 dark:text-gray-400">×</span>
                    <input
                      type="number"
                      min={240}
                      max={2160}
                      value={advancedSettings.viewportHeight}
                      onChange={(e) => updateSetting('viewportHeight', Math.max(240, Math.min(2160, parseInt(e.target.value) || 1080)))}
                      className="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <span className="text-sm text-gray-500 dark:text-gray-400">px</span>
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                    Browser window dimensions for workflow execution.
                  </p>
                </div>

                {/* Continue on Error */}
                <div className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                        Continue on Error
                      </label>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                        Keep running even if a step fails.
                      </p>
                    </div>
                    <button
                      type="button"
                      role="switch"
                      aria-checked={advancedSettings.continueOnError}
                      onClick={() => updateSetting('continueOnError', !advancedSettings.continueOnError)}
                      className={`relative w-11 h-6 rounded-full transition-colors ${
                        advancedSettings.continueOnError
                          ? 'bg-blue-500'
                          : 'bg-gray-300 dark:bg-gray-600'
                      }`}
                    >
                      <span
                        className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${
                          advancedSettings.continueOnError ? 'translate-x-5' : 'translate-x-0'
                        }`}
                      />
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Test Results */}
          {testResults && (
            <div
              className={`flex items-center gap-3 p-4 rounded-lg border ${
                testResults.success
                  ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800'
                  : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
              }`}
            >
              {testResults.success ? (
                <CheckCircle2 size={20} className="text-green-500 flex-shrink-0" />
              ) : (
                <XCircle size={20} className="text-red-500 flex-shrink-0" />
              )}
              <div className="flex-1">
                <p
                  className={`text-sm font-medium ${
                    testResults.success
                      ? 'text-green-800 dark:text-green-200'
                      : 'text-red-800 dark:text-red-200'
                  }`}
                >
                  {testResults.success
                    ? 'All tests passed!'
                    : `${testResults.failed_actions} of ${testResults.total_actions} steps failed`}
                </p>
                <p
                  className={`text-xs mt-0.5 ${
                    testResults.success
                      ? 'text-green-700 dark:text-green-300'
                      : 'text-red-700 dark:text-red-300'
                  }`}
                >
                  Completed in {(testResults.total_duration_ms / 1000).toFixed(1)}s
                </p>
              </div>
            </div>
          )}

          {/* Error Message */}
          {error && (
            <div className="flex items-start gap-3 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
              <XCircle size={18} className="text-red-500 flex-shrink-0 mt-0.5" />
              <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
            </div>
          )}
        </div>
      </div>

      {/* Footer Actions */}
      <div className="px-4 py-3 border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <div className="flex gap-3">
          <button
            onClick={handleTest}
            disabled={isReplaying || isGenerating || totalSelected === 0}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isReplaying ? (
              <>
                <div className="w-4 h-4 border-2 border-gray-400 border-t-transparent rounded-full animate-spin" />
                Testing...
              </>
            ) : (
              <>
                <Play size={16} />
                Test First
              </>
            )}
          </button>
          <button
            onClick={handleGenerate}
            disabled={!isFormValid || isReplaying || isGenerating}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isGenerating ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <Sparkles size={16} />
                Generate
              </>
            )}
          </button>
        </div>
      </div>

      {/* Project Picker Modal */}
      <ProjectPickerModal
        isOpen={isProjectPickerOpen}
        onClose={() => setIsProjectPickerOpen(false)}
        onSelect={setSelectedProject}
        onCreateNew={() => setIsProjectModalOpen(true)}
        selectedProject={selectedProject}
      />

      {/* Project Creation Modal */}
      <ProjectModal
        isOpen={isProjectModalOpen}
        onClose={() => setIsProjectModalOpen(false)}
        onSuccess={handleProjectCreated}
      />
    </div>
  );
}
