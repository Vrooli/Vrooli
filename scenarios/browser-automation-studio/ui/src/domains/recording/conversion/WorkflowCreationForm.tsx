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
 * - Test button to validate before generating
 * - Generate button to create the workflow
 * - Back button to return to Live Preview
 */

import { useEffect, useMemo, useState } from 'react';
import { useProjectStore, type Project } from '@/domains/projects';
import type { RecordedAction, RecordingSessionProfile, ReplayPreviewResponse } from '../types/types';

/** Describes a contiguous range of selected steps */
export interface SelectionRange {
  start: number;
  end: number;
  count: number;
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
  }) => Promise<void>;
  /** Count of actions with low confidence selectors */
  lowConfidenceCount: number;
  /** Count of actions with medium confidence selectors */
  mediumConfidenceCount: number;
}

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
      // Contiguous
      rangeEnd = sorted[i];
    } else {
      // Gap - close current range and start new one
      ranges.push({ start: rangeStart, end: rangeEnd, count: rangeEnd - rangeStart + 1 });
      rangeStart = sorted[i];
      rangeEnd = sorted[i];
    }
  }
  // Close final range
  ranges.push({ start: rangeStart, end: rangeEnd, count: rangeEnd - rangeStart + 1 });

  return ranges;
}

/**
 * Format ranges for display.
 * E.g., [{start: 0, end: 2}, {start: 5, end: 6}] -> "Steps 1-3, 6-7"
 * (Using 1-indexed for user display)
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
  const { projects, fetchProjects, isLoading: projectsLoading, getSmartDefaultProject } = useProjectStore();

  const [workflowName, setWorkflowName] = useState('');
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<ReplayPreviewResponse | null>(null);

  // Fetch projects on mount
  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  // Set default project when projects load
  useEffect(() => {
    if (!selectedProjectId && projects.length > 0) {
      const defaultProject = getSmartDefaultProject();
      if (defaultProject) {
        setSelectedProjectId(defaultProject.id);
      }
    }
  }, [projects, selectedProjectId, getSmartDefaultProject]);

  // Compute selection info
  const selectionRanges = useMemo(() => computeSelectionRanges(selectedIndices), [selectedIndices]);
  const totalSelected = selectedIndices.length;
  const hasMultipleRanges = selectionRanges.length > 1;

  // Check for unstable selectors in selection
  const hasUnstableSelectors = lowConfidenceCount > 0 || mediumConfidenceCount > 0;

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
    if (!selectedProjectId) {
      setError('Please select a project');
      return;
    }
    if (selectedIndices.length === 0) {
      setError('No steps selected');
      return;
    }

    setError(null);
    try {
      await onGenerate({
        name: workflowName.trim(),
        projectId: selectedProjectId,
        defaultSessionId: selectedSessionId,
        actionIndices: selectedIndices,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate workflow');
    }
  };

  const isFormValid = workflowName.trim().length > 0 && selectedProjectId && totalSelected > 0;

  return (
    <div className="flex flex-col h-full bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={onBack}
          className="flex items-center gap-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Back
        </button>
        <div className="flex-1" />
        <h2 className="text-sm font-semibold text-gray-900 dark:text-white">Create Workflow</h2>
        <div className="flex-1" />
        {/* Spacer to center title */}
        <div className="w-[52px]" />
      </div>

      {/* Form Content */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Selection Summary */}
        <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 w-10 h-10 flex items-center justify-center bg-blue-100 dark:bg-blue-800 rounded-full">
              <span className="text-lg font-bold text-blue-600 dark:text-blue-300">{totalSelected}</span>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                {totalSelected === 1 ? '1 step selected' : `${totalSelected} steps selected`}
              </p>
              <p className="text-xs text-blue-700 dark:text-blue-300 mt-0.5">
                {formatRanges(selectionRanges)}
              </p>
              {hasMultipleRanges && (
                <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                  {selectionRanges.length} separate ranges will create {selectionRanges.length} workflow{selectionRanges.length > 1 ? 's' : ''}
                </p>
              )}
            </div>
          </div>
        </div>

        {/* Unstable Selectors Warning */}
        {hasUnstableSelectors && (
          <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              <div>
                <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
                  {lowConfidenceCount > 0
                    ? `${lowConfidenceCount} step${lowConfidenceCount > 1 ? 's have' : ' has'} unstable selectors`
                    : `${mediumConfidenceCount} step${mediumConfidenceCount > 1 ? 's have' : ' has'} potentially unstable selectors`}
                </p>
                <p className="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
                  The workflow may fail on replay. Consider editing selectors before generating.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Workflow Name */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Workflow Name <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={workflowName}
            onChange={(e) => setWorkflowName(e.target.value)}
            placeholder="e.g., Login Flow, Submit Form"
            className="w-full px-4 py-2.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            autoFocus
          />
        </div>

        {/* Project Selection */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Project <span className="text-red-500">*</span>
          </label>
          <select
            value={selectedProjectId || ''}
            onChange={(e) => setSelectedProjectId(e.target.value || null)}
            disabled={projectsLoading}
            className="w-full px-4 py-2.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
          >
            {projectsLoading ? (
              <option value="">Loading projects...</option>
            ) : projects.length === 0 ? (
              <option value="">No projects available</option>
            ) : (
              <>
                <option value="">Select a project</option>
                {projects.map((project: Project) => (
                  <option key={project.id} value={project.id}>
                    {project.name}
                  </option>
                ))}
              </>
            )}
          </select>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            The workflow will be saved to this project.
          </p>
        </div>

        {/* Default Session Selection */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Default Session
          </label>
          <select
            value={selectedSessionId || ''}
            onChange={(e) => setSelectedSessionId(e.target.value || null)}
            disabled={sessionProfilesLoading}
            className="w-full px-4 py-2.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
          >
            <option value="">None (create new each run)</option>
            {sessionProfiles.map((profile) => (
              <option key={profile.id} value={profile.id}>
                {profile.name}
                {profile.hasStorageState ? ' (Auth saved)' : ''}
              </option>
            ))}
          </select>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Optional: Use a saved browser session with existing login state.
          </p>
        </div>

        {/* Test Results */}
        {testResults && (
          <div
            className={`p-4 rounded-lg border ${
              testResults.success
                ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800'
                : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
            }`}
          >
            <div className="flex items-center gap-3">
              {testResults.success ? (
                <svg className="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                    clipRule="evenodd"
                  />
                </svg>
              ) : (
                <svg className="w-5 h-5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
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
                  {(testResults.total_duration_ms / 1000).toFixed(1)}s total
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
          </div>
        )}
      </div>

      {/* Footer Actions */}
      <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
        <div className="flex gap-3">
          <button
            onClick={handleTest}
            disabled={isReplaying || isGenerating || totalSelected === 0}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isReplaying ? (
              <>
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                Testing...
              </>
            ) : (
              <>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                Test First
              </>
            )}
          </button>
          <button
            onClick={handleGenerate}
            disabled={!isFormValid || isReplaying || isGenerating}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isGenerating ? (
              <>
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                Generating...
              </>
            ) : (
              <>
                Generate
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                </svg>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
