/**
 * TimelineFullView Component - Multi-Workflow Builder
 *
 * A workflow-creation-centric view that allows users to:
 * - Create multiple workflows from a single recording session
 * - Visually select ranges of steps for each workflow
 * - See workflow boundaries on a horizontal mini-timeline
 * - Test and generate all workflows at once
 *
 * Design philosophy: The workflow creation form is the main focus,
 * with the timeline serving as a selection tool rather than the primary content.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useProjectStore, type Project } from '@/stores/projectStore';
import { getConfig } from '@/config';
import type { RecordedAction, RecordingSessionProfile, ReplayPreviewResponse, SelectorValidation } from '../types';

/** Composition mode for workflow generation */
type CompositionMode = 'inline' | 'reference';

/** Existing workflow summary for reference mode */
interface ExistingWorkflow {
  id: string;
  name: string;
  workflow_type?: string;
  folder_path?: string;
}

/** A pending workflow to be created */
interface PendingWorkflow {
  id: string;
  name: string;
  workflowType: 'action' | 'flow' | 'case';
  /** Relative folder path within project (no leading slash) */
  folderPath: string;
  projectId: string | null;
  sessionId: string | null;
  /** Start index (inclusive) */
  startIndex: number;
  /** End index (inclusive) */
  endIndex: number;
  /** Color for visual identification */
  color: string;
  /** Whether currently being tested */
  isTesting: boolean;
  /** Test result if available */
  testResult?: { success: boolean; message: string };
  /** Composition mode: inline nodes directly or reference existing workflows */
  compositionMode: CompositionMode;
  /** ID of existing workflow to reference (only for reference mode) */
  referenceWorkflowId?: string;
}

/** Available colors for workflow segments */
const WORKFLOW_COLORS = [
  { bg: 'bg-blue-500', light: 'bg-blue-100 dark:bg-blue-900/30', border: 'border-blue-300 dark:border-blue-700', text: 'text-blue-700 dark:text-blue-300' },
  { bg: 'bg-emerald-500', light: 'bg-emerald-100 dark:bg-emerald-900/30', border: 'border-emerald-300 dark:border-emerald-700', text: 'text-emerald-700 dark:text-emerald-300' },
  { bg: 'bg-violet-500', light: 'bg-violet-100 dark:bg-violet-900/30', border: 'border-violet-300 dark:border-violet-700', text: 'text-violet-700 dark:text-violet-300' },
  { bg: 'bg-amber-500', light: 'bg-amber-100 dark:bg-amber-900/30', border: 'border-amber-300 dark:border-amber-700', text: 'text-amber-700 dark:text-amber-300' },
  { bg: 'bg-rose-500', light: 'bg-rose-100 dark:bg-rose-900/30', border: 'border-rose-300 dark:border-rose-700', text: 'text-rose-700 dark:text-rose-300' },
  { bg: 'bg-cyan-500', light: 'bg-cyan-100 dark:bg-cyan-900/30', border: 'border-cyan-300 dark:border-cyan-700', text: 'text-cyan-700 dark:text-cyan-300' },
];

function normalizeRelFolder(value: string): string {
  const normalized = value.replace(/\\/g, '/').trim().replace(/^\/+/, '').replace(/\/+$/, '');
  return normalized;
}

type ProjectPresetConfig = {
  version: 'v1';
  namingTemplates?: Partial<Record<PendingWorkflow['workflowType'], string>>;
};

function fileBasename(relPath: string): string {
  const normalized = relPath.replace(/\\/g, '/').trim().replace(/^\/+/, '').replace(/\/+$/, '');
  const parts = normalized.split('/').filter(Boolean);
  return parts.length > 0 ? parts[parts.length - 1] : '';
}

function fileDirname(relPath: string): string {
  const normalized = relPath.replace(/\\/g, '/').trim().replace(/^\/+/, '').replace(/\/+$/, '');
  const idx = normalized.lastIndexOf('/');
  return idx > 0 ? normalized.slice(0, idx) : '';
}

function renderNamingTemplate(template: string, name: string, kind: PendingWorkflow['workflowType']): string {
  const slug = slugifyWorkflowFilename(name) || 'workflow';
  return template
    .replace(/\{name\}/g, slug)
    .replace(/\{type\}/g, kind)
    .replace(/\{kind\}/g, kind);
}

function typedWorkflowFileSuffix(workflowType: PendingWorkflow['workflowType']): string {
  switch (workflowType) {
    case 'action':
      return '.action.json';
    case 'case':
      return '.case.json';
    default:
      return '.flow.json';
  }
}

function slugifyWorkflowFilename(value: string): string {
  const lowered = value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
  return lowered || 'workflow';
}

/** Generate a unique ID */
function generateId(): string {
  return `wf_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}

/** Get a short action label for the mini-timeline */
function getShortActionLabel(action: RecordedAction): string {
  const rawType: string = action.actionType;
  const normalizedType =
    rawType === 'type' ? 'input' : rawType === 'keypress' ? 'keyboard' : rawType;
  switch (normalizedType) {
    case 'click': return 'Click';
    case 'input': return 'Type';
    case 'navigate': return 'Nav';
    case 'scroll': return 'Scroll';
    case 'keyboard': return 'Key';
    case 'select': return 'Select';
    default: return normalizedType.slice(0, 4);
  }
}

interface TimelineFullViewProps {
  /** All recorded actions */
  actions: RecordedAction[];
  /** Whether recording is active */
  isRecording: boolean;
  /** Whether replaying */
  isReplaying: boolean;
  /** Whether generating */
  isGenerating: boolean;
  /** Session profiles */
  sessionProfiles: RecordingSessionProfile[];
  /** Whether profiles are loading */
  sessionProfilesLoading: boolean;
  /** Low confidence count */
  lowConfidenceCount: number;
  /** Medium confidence count */
  mediumConfidenceCount: number;
  /** Callback to delete an action */
  onDeleteAction?: (index: number) => void;
  /** Callback to validate a selector */
  onValidateSelector?: (selector: string) => Promise<SelectorValidation>;
  /** Callback to edit a selector */
  onEditSelector?: (index: number, newSelector: string) => void;
  /** Callback to edit payload */
  onEditPayload?: (index: number, payload: Record<string, unknown>) => void;
  /** Selection mode state (unused but kept for API compat) */
  isSelectionMode: boolean;
  /** Selected action indices (unused but kept for API compat) */
  selectedIndices: Set<number>;
  /** Callback to toggle selection mode (unused but kept for API compat) */
  onToggleSelectionMode: () => void;
  /** Callback when action is clicked (unused but kept for API compat) */
  onActionClick: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
  /** Callback to select all (unused but kept for API compat) */
  onSelectAll: () => void;
  /** Callback to select none (unused but kept for API compat) */
  onSelectNone: () => void;
  /** Callback to test selected actions */
  onTest: (actionIndices: number[]) => Promise<ReplayPreviewResponse>;
  /** Callback to generate workflow */
  onGenerate: (params: {
    name: string;
    projectId: string;
    defaultSessionId: string | null;
    actionIndices: number[];
    workflowType?: 'action' | 'flow' | 'case';
    path?: string;
    /** For reference mode: ID of existing workflow to create subflow reference to */
    referenceWorkflowId?: string;
    compositionMode?: CompositionMode;
  }) => Promise<void>;
  /** Callback to go back to live view */
  onBack: () => void;
}

export function TimelineFullView({
  actions,
  isRecording,
  isReplaying,
  isGenerating,
  sessionProfiles,
  sessionProfilesLoading,
  lowConfidenceCount,
  mediumConfidenceCount,
  onTest,
  onGenerate,
  onBack,
}: TimelineFullViewProps) {
  const { projects, fetchProjects, isLoading: projectsLoading, getSmartDefaultProject } = useProjectStore();
  const [presetConfigsByProjectId, setPresetConfigsByProjectId] = useState<
    Record<string, ProjectPresetConfig | null>
  >({});

  // Existing workflows per project for reference mode
  const [existingWorkflowsByProjectId, setExistingWorkflowsByProjectId] = useState<
    Record<string, ExistingWorkflow[]>
  >({});

  // Workflow queue state
  const [workflows, setWorkflows] = useState<PendingWorkflow[]>([]);
  const [editingWorkflowId, setEditingWorkflowId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isGeneratingAll, setIsGeneratingAll] = useState(false);
  const [generationProgress, setGenerationProgress] = useState<{ current: number; total: number } | null>(null);

  // Mini-timeline interaction state
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState<number | null>(null);
  const [dragEnd, setDragEnd] = useState<number | null>(null);
  const timelineRef = useRef<HTMLDivElement>(null);

  // Fetch projects on mount
  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  useEffect(() => {
    const ids = Array.from(
      new Set(workflows.map((wf) => wf.projectId).filter((id): id is string => Boolean(id)))
    );
    if (ids.length === 0) return;

    let cancelled = false;
    const fetchMissing = async () => {
      const config = await getConfig();
      await Promise.all(
        ids.map(async (projectId) => {
          if (presetConfigsByProjectId[projectId] !== undefined) return;
          try {
            const response = await fetch(`${config.API_URL}/projects/${projectId}/preset-config`);
            if (!response.ok) {
              if (!cancelled) {
                setPresetConfigsByProjectId((prev) => ({ ...prev, [projectId]: null }));
              }
              return;
            }
            const payload = (await response.json()) as ProjectPresetConfig;
            if (!payload || payload.version !== 'v1') {
              if (!cancelled) {
                setPresetConfigsByProjectId((prev) => ({ ...prev, [projectId]: null }));
              }
              return;
            }
            if (!cancelled) {
              setPresetConfigsByProjectId((prev) => ({ ...prev, [projectId]: payload }));
            }
          } catch (err) {
            console.warn('Failed to fetch project preset config', err);
            if (!cancelled) {
              setPresetConfigsByProjectId((prev) => ({ ...prev, [projectId]: null }));
            }
          }
        })
      );
    };

    void fetchMissing();
    return () => {
      cancelled = true;
    };
  }, [workflows, presetConfigsByProjectId]);

  // Fetch existing workflows for projects when reference mode is used
  useEffect(() => {
    const projectsWithReferenceMode = workflows
      .filter((wf) => wf.compositionMode === 'reference' && wf.projectId)
      .map((wf) => wf.projectId!)
      .filter((id, idx, arr) => arr.indexOf(id) === idx); // unique

    if (projectsWithReferenceMode.length === 0) return;

    let cancelled = false;
    const fetchExisting = async () => {
      const config = await getConfig();
      await Promise.all(
        projectsWithReferenceMode.map(async (projectId) => {
          if (existingWorkflowsByProjectId[projectId] !== undefined) return;
          try {
            const response = await fetch(`${config.API_URL}/projects/${projectId}/workflows`);
            if (!response.ok) {
              if (!cancelled) {
                setExistingWorkflowsByProjectId((prev) => ({ ...prev, [projectId]: [] }));
              }
              return;
            }
            const payload = (await response.json()) as { workflows?: ExistingWorkflow[] };
            if (!cancelled) {
              setExistingWorkflowsByProjectId((prev) => ({
                ...prev,
                [projectId]: Array.isArray(payload.workflows) ? payload.workflows : [],
              }));
            }
          } catch (err) {
            console.warn('Failed to fetch project workflows for reference mode', err);
            if (!cancelled) {
              setExistingWorkflowsByProjectId((prev) => ({ ...prev, [projectId]: [] }));
            }
          }
        })
      );
    };

    void fetchExisting();
    return () => {
      cancelled = true;
    };
  }, [workflows, existingWorkflowsByProjectId]);

  const selectionHasAssert = useCallback((startIndex: number, endIndex: number) => {
    const start = Math.max(0, Math.min(startIndex, endIndex));
    const end = Math.min(actions.length - 1, Math.max(startIndex, endIndex));
    for (let i = start; i <= end; i++) {
      const action = actions[i];
      if (!action) continue;
      if (action.actionKind === 'RECORDED_ACTION_TYPE_ASSERT') return true;
      const typed = action.typedAction as Record<string, unknown> | undefined;
      if (typed && typeof typed === 'object' && 'assert' in typed) return true;
    }
    return false;
  }, [actions]);

  const suggestWorkflowType = useCallback((startIndex: number, endIndex: number): PendingWorkflow['workflowType'] => {
    if (selectionHasAssert(startIndex, endIndex)) {
      return 'case';
    }
    const len = Math.abs(endIndex - startIndex) + 1;
    if (len <= 5) {
      return 'action';
    }
    return 'flow';
  }, [selectionHasAssert]);

  const getDefaultFolderFor = useCallback((
    projectId: string | null,
    workflowType: PendingWorkflow['workflowType'],
  ) => {
    const presetConfig = projectId ? presetConfigsByProjectId[projectId] : null;
    const template = presetConfig?.namingTemplates?.[workflowType];
    const rendered = template ? renderNamingTemplate(template, workflowType, workflowType) : '';
    const templateFolder = rendered ? fileDirname(rendered) : '';
    if (templateFolder) return templateFolder;
    return workflowType === 'action' ? 'actions' : workflowType === 'case' ? 'cases' : 'flows';
  }, [presetConfigsByProjectId]);

  // Initialize with a single workflow covering all actions
  useEffect(() => {
    if (actions.length > 0 && workflows.length === 0) {
      const defaultProject = getSmartDefaultProject();
      const projectId = defaultProject?.id || null;
      setWorkflows([{
        id: generateId(),
        name: '',
        workflowType: 'flow',
        folderPath: getDefaultFolderFor(projectId, 'flow'),
        projectId,
        sessionId: null,
        startIndex: 0,
        endIndex: actions.length - 1,
        color: WORKFLOW_COLORS[0].bg,
        isTesting: false,
        compositionMode: 'inline',
      }]);
    }
  }, [actions.length, workflows.length, getSmartDefaultProject, getDefaultFolderFor]);

  // Get steps assigned to workflows
  const assignedSteps = useMemo(() => {
    const assigned = new Map<number, string>(); // stepIndex -> workflowId
    for (const wf of workflows) {
      for (let i = wf.startIndex; i <= wf.endIndex; i++) {
        assigned.set(i, wf.id);
      }
    }
    return assigned;
  }, [workflows]);

  // Find unassigned steps
  const unassignedSteps = useMemo(() => {
    const unassigned: number[] = [];
    for (let i = 0; i < actions.length; i++) {
      if (!assignedSteps.has(i)) {
        unassigned.push(i);
      }
    }
    return unassigned;
  }, [actions.length, assignedSteps]);

  // Check for overlapping workflows
  const hasOverlaps = useMemo(() => {
    const stepCount = new Map<number, number>();
    for (const wf of workflows) {
      for (let i = wf.startIndex; i <= wf.endIndex; i++) {
        stepCount.set(i, (stepCount.get(i) || 0) + 1);
      }
    }
    return Array.from(stepCount.values()).some(c => c > 1);
  }, [workflows]);

  // Get color for a workflow by index
  const getWorkflowColor = useCallback((index: number) => {
    return WORKFLOW_COLORS[index % WORKFLOW_COLORS.length];
  }, []);

  // Add a new workflow
  const handleAddWorkflow = useCallback(() => {
    // Find first unassigned range
    let startIdx = 0;
    let endIdx = actions.length - 1;

    if (unassignedSteps.length > 0) {
      // Find contiguous range of unassigned steps
      startIdx = unassignedSteps[0];
      endIdx = startIdx;
      for (let i = 1; i < unassignedSteps.length; i++) {
        if (unassignedSteps[i] === endIdx + 1) {
          endIdx = unassignedSteps[i];
        } else {
          break;
        }
      }
    }

    const defaultProject = getSmartDefaultProject();
    const suggestedType = suggestWorkflowType(startIdx, endIdx);
    const newWorkflow: PendingWorkflow = {
      id: generateId(),
      name: '',
      workflowType: suggestedType,
      folderPath: getDefaultFolderFor(defaultProject?.id || null, suggestedType),
      projectId: defaultProject?.id || null,
      sessionId: null,
      startIndex: startIdx,
      endIndex: endIdx,
      color: getWorkflowColor(workflows.length).bg,
      isTesting: false,
      compositionMode: 'inline',
    };

    setWorkflows([...workflows, newWorkflow]);
    setEditingWorkflowId(newWorkflow.id);
  }, [actions.length, unassignedSteps, workflows, getSmartDefaultProject, getWorkflowColor, getDefaultFolderFor, suggestWorkflowType]);

  // Remove a workflow
  const handleRemoveWorkflow = useCallback((id: string) => {
    setWorkflows(workflows.filter(wf => wf.id !== id));
    if (editingWorkflowId === id) {
      setEditingWorkflowId(null);
    }
  }, [workflows, editingWorkflowId]);

  // Update a workflow field
  const handleUpdateWorkflow = useCallback((id: string, updates: Partial<PendingWorkflow>) => {
    setWorkflows(workflows.map(wf =>
      wf.id === id ? { ...wf, ...updates } : wf
    ));
  }, [workflows]);

  // Test a single workflow
  const handleTestWorkflow = useCallback(async (workflow: PendingWorkflow) => {
    handleUpdateWorkflow(workflow.id, { isTesting: true, testResult: undefined });
    try {
      const indices = Array.from(
        { length: workflow.endIndex - workflow.startIndex + 1 },
        (_, i) => workflow.startIndex + i
      );
      const result = await onTest(indices);
      handleUpdateWorkflow(workflow.id, {
        isTesting: false,
        testResult: {
          success: result.success,
          message: result.success
            ? `Passed (${(result.total_duration_ms / 1000).toFixed(1)}s)`
            : `${result.failed_actions} failed`,
        },
      });
    } catch (err) {
      handleUpdateWorkflow(workflow.id, {
        isTesting: false,
        testResult: {
          success: false,
          message: err instanceof Error ? err.message : 'Test failed',
        },
      });
    }
  }, [onTest, handleUpdateWorkflow]);

  // Generate all workflows
  const handleGenerateAll = useCallback(async () => {
    // Validate all workflows
    const invalidWorkflows = workflows.filter(wf => !wf.name.trim() || !wf.projectId);
    if (invalidWorkflows.length > 0) {
      setError(`${invalidWorkflows.length} workflow(s) are missing name or project`);
      return;
    }

    // Validate reference mode workflows have a selected workflow
    const invalidRefWorkflows = workflows.filter(
      wf => wf.compositionMode === 'reference' && !wf.referenceWorkflowId
    );
    if (invalidRefWorkflows.length > 0) {
      setError(`${invalidRefWorkflows.length} workflow(s) in reference mode need a workflow selected`);
      return;
    }

    if (hasOverlaps) {
      setError('Some steps are assigned to multiple workflows. Please fix overlaps first.');
      return;
    }

    setError(null);
    setIsGeneratingAll(true);
    setGenerationProgress({ current: 0, total: workflows.length });

    try {
      for (let i = 0; i < workflows.length; i++) {
        const wf = workflows[i];
        setGenerationProgress({ current: i + 1, total: workflows.length });

        const indices = Array.from(
          { length: wf.endIndex - wf.startIndex + 1 },
          (_, j) => wf.startIndex + j
        );

        const folder = normalizeRelFolder(wf.folderPath);
        const presetConfig = wf.projectId ? presetConfigsByProjectId[wf.projectId] : null;
        const template = presetConfig?.namingTemplates?.[wf.workflowType];
        const templatePath = template ? renderNamingTemplate(template, wf.name, wf.workflowType) : '';
        const baseName = templatePath ? fileBasename(templatePath) : `${slugifyWorkflowFilename(wf.name)}${typedWorkflowFileSuffix(wf.workflowType)}`;
        const relPath = folder ? `${folder}/${baseName}` : baseName;

        await onGenerate({
          name: wf.name.trim(),
          projectId: wf.projectId!,
          defaultSessionId: wf.sessionId,
          actionIndices: indices,
          workflowType: wf.workflowType,
          path: relPath,
          compositionMode: wf.compositionMode,
          referenceWorkflowId: wf.referenceWorkflowId,
        });
      }

      // Success - go back to live view
      onBack();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate workflows');
    } finally {
      setIsGeneratingAll(false);
      setGenerationProgress(null);
    }
  }, [workflows, hasOverlaps, onGenerate, onBack, presetConfigsByProjectId]);

  // Mini-timeline drag handlers
  const handleTimelineMouseDown = useCallback((e: React.MouseEvent, stepIndex: number) => {
    e.preventDefault();
    setIsDragging(true);
    setDragStart(stepIndex);
    setDragEnd(stepIndex);
  }, []);

  const handleTimelineMouseMove = useCallback((_e: React.MouseEvent, stepIndex: number) => {
    if (isDragging) {
      setDragEnd(stepIndex);
    }
  }, [isDragging]);

  const handleTimelineMouseUp = useCallback(() => {
    if (isDragging && dragStart !== null && dragEnd !== null) {
      const start = Math.min(dragStart, dragEnd);
      const end = Math.max(dragStart, dragEnd);

      // If we have an editing workflow, update its range
      if (editingWorkflowId) {
        handleUpdateWorkflow(editingWorkflowId, { startIndex: start, endIndex: end });
      } else {
        // Create a new workflow with this range
        const defaultProject = getSmartDefaultProject();
        const suggestedType = suggestWorkflowType(start, end);
        const newWorkflow: PendingWorkflow = {
          id: generateId(),
          name: '',
          workflowType: suggestedType,
          folderPath: getDefaultFolderFor(defaultProject?.id || null, suggestedType),
          projectId: defaultProject?.id || null,
          sessionId: null,
          startIndex: start,
          endIndex: end,
          color: getWorkflowColor(workflows.length).bg,
          isTesting: false,
          compositionMode: 'inline',
        };
        setWorkflows([...workflows, newWorkflow]);
        setEditingWorkflowId(newWorkflow.id);
      }
    }

    setIsDragging(false);
    setDragStart(null);
    setDragEnd(null);
  }, [isDragging, dragStart, dragEnd, editingWorkflowId, handleUpdateWorkflow, workflows, getSmartDefaultProject, getWorkflowColor, getDefaultFolderFor, suggestWorkflowType]);

  // Calculate which workflow owns each step for coloring
  const stepColors = useMemo(() => {
    const colors: (typeof WORKFLOW_COLORS[0] | null)[] = new Array(actions.length).fill(null);
    workflows.forEach((wf, wfIdx) => {
      const color = getWorkflowColor(wfIdx);
      for (let i = wf.startIndex; i <= wf.endIndex; i++) {
        if (i < colors.length) {
          colors[i] = color;
        }
      }
    });
    return colors;
  }, [actions.length, workflows, getWorkflowColor]);

  // Get drag selection range for highlighting
  const dragRange = useMemo(() => {
    if (!isDragging || dragStart === null || dragEnd === null) return null;
    return { start: Math.min(dragStart, dragEnd), end: Math.max(dragStart, dragEnd) };
  }, [isDragging, dragStart, dragEnd]);

  const hasUnstableSelectors = lowConfidenceCount > 0 || mediumConfidenceCount > 0;
  const allWorkflowsValid = workflows.every(wf => wf.name.trim() && wf.projectId);
  const canGenerate = workflows.length > 0 && allWorkflowsValid && !hasOverlaps;

  return (
    <div className="flex flex-col h-full bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-4">
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            Back to Recording
          </button>
          <div className="w-px h-6 bg-gray-200 dark:bg-gray-700" />
          <h1 className="text-lg font-semibold text-gray-900 dark:text-white">
            Workflow Builder
          </h1>
        </div>

        <div className="flex items-center gap-3">
          {isRecording && (
            <div className="flex items-center gap-1.5 px-3 py-1 text-xs text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 rounded-full">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-red-500" />
              </span>
              Recording
            </div>
          )}
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {actions.length} total steps
          </span>
        </div>
      </div>

      {/* Main content area */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {/* Workflow queue */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="max-w-4xl mx-auto space-y-4">
            {/* Instructions */}
            <div className="flex items-start gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
              <svg className="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div className="text-sm text-blue-800 dark:text-blue-200">
                <p className="font-medium">Create multiple workflows from your recording</p>
                <p className="text-blue-700 dark:text-blue-300 mt-1">
                  Drag on the timeline below to select steps, or click a workflow card to edit its range.
                  Each workflow will be generated as a separate automation.
                </p>
              </div>
            </div>

            {/* Warnings */}
            {hasUnstableSelectors && (
              <div className="flex items-start gap-3 p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
                <svg className="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
                <div className="text-sm">
                  <p className="font-medium text-yellow-800 dark:text-yellow-200">
                    {lowConfidenceCount > 0
                      ? `${lowConfidenceCount} unstable selector${lowConfidenceCount > 1 ? 's' : ''} detected`
                      : `${mediumConfidenceCount} potentially unstable selector${mediumConfidenceCount > 1 ? 's' : ''}`}
                  </p>
                  <p className="text-yellow-700 dark:text-yellow-300 mt-0.5">
                    Consider reviewing selectors before generating workflows
                  </p>
                </div>
              </div>
            )}

            {hasOverlaps && (
              <div className="flex items-start gap-3 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                <svg className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
                <div className="text-sm">
                  <p className="font-medium text-red-800 dark:text-red-200">Overlapping workflows detected</p>
                  <p className="text-red-700 dark:text-red-300 mt-0.5">
                    Some steps are assigned to multiple workflows. Adjust ranges to fix this.
                  </p>
                </div>
              </div>
            )}

            {/* Workflow cards */}
            <div className="space-y-3">
              {workflows.map((workflow, wfIdx) => {
                const color = getWorkflowColor(wfIdx);
                const isEditing = editingWorkflowId === workflow.id;
                const stepCount = workflow.endIndex - workflow.startIndex + 1;

                return (
                  <div
                    key={workflow.id}
                    className={`relative rounded-xl border-2 transition-all ${
                      isEditing
                        ? `${color.border} ${color.light} shadow-md`
                        : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600'
                    }`}
                  >
                    {/* Color indicator bar */}
                    <div className={`absolute left-0 top-0 bottom-0 w-1.5 rounded-l-xl ${color.bg}`} />

                    <div className="pl-5 pr-4 py-4">
                      <div className="flex items-start gap-4">
                        {/* Workflow number badge */}
                        <div className={`flex-shrink-0 w-8 h-8 rounded-lg ${color.bg} flex items-center justify-center`}>
                          <span className="text-sm font-bold text-white">{wfIdx + 1}</span>
                        </div>

                        {/* Form fields */}
                        <div className="flex-1 min-w-0 space-y-3">
                          {/* Name and range row */}
                          <div className="flex items-center gap-3">
                            <input
                              type="text"
                              value={workflow.name}
                              onChange={(e) => handleUpdateWorkflow(workflow.id, { name: e.target.value })}
                              onFocus={() => setEditingWorkflowId(workflow.id)}
                              placeholder="Workflow name (e.g., Login Flow)"
                              className={`flex-1 px-3 py-2 text-sm border rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                                !workflow.name.trim() && !isEditing ? 'border-red-300 dark:border-red-700' : 'border-gray-300 dark:border-gray-600'
                              }`}
                            />
                            <div className={`flex-shrink-0 px-3 py-2 text-xs font-medium rounded-lg ${color.light} ${color.text}`}>
                              Steps {workflow.startIndex + 1}-{workflow.endIndex + 1} ({stepCount})
                            </div>
                          </div>

                          {/* Project and session row */}
                          <div className="flex items-center gap-3">
                            <select
                              value={workflow.projectId || ''}
                              onChange={(e) => handleUpdateWorkflow(workflow.id, { projectId: e.target.value || null })}
                              onFocus={() => setEditingWorkflowId(workflow.id)}
                              disabled={projectsLoading}
                              className={`flex-1 px-3 py-2 text-sm border rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50 ${
                                !workflow.projectId && !isEditing ? 'border-red-300 dark:border-red-700' : 'border-gray-300 dark:border-gray-600'
                              }`}
                            >
                              {projectsLoading ? (
                                <option value="">Loading...</option>
                              ) : projects.length === 0 ? (
                                <option value="">No projects</option>
                              ) : (
                                <>
                                  <option value="">Select project</option>
                                  {projects.map((project: Project) => (
                                    <option key={project.id} value={project.id}>
                                      {project.name}
                                    </option>
                                  ))}
                                </>
                              )}
                            </select>

                            <select
                              value={workflow.sessionId || ''}
                              onChange={(e) => handleUpdateWorkflow(workflow.id, { sessionId: e.target.value || null })}
                              onFocus={() => setEditingWorkflowId(workflow.id)}
                              disabled={sessionProfilesLoading}
                              className="flex-1 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                            >
                              <option value="">No saved session</option>
                              {sessionProfiles.map((profile) => (
                                <option key={profile.id} value={profile.id}>
                                  {profile.name}{profile.hasStorageState ? ' (Auth)' : ''}
                                </option>
                              ))}
                            </select>
                          </div>

                          {/* Type and path row */}
                          <div className="flex items-center gap-3">
                            <select
                              value={workflow.workflowType}
                              onChange={(e) => {
                                const nextType = e.target.value as PendingWorkflow['workflowType'];
                                const presetConfig = workflow.projectId ? presetConfigsByProjectId[workflow.projectId] : null;
                                const template = presetConfig?.namingTemplates?.[nextType];
                                const templatePath = template ? renderNamingTemplate(template, workflow.name || nextType, nextType) : '';
                                const templateFolder = templatePath ? fileDirname(templatePath) : '';
                                const nextDefaultFolder = templateFolder || (nextType === 'action' ? 'actions' : nextType === 'case' ? 'cases' : 'flows');
                                const currentFolder = normalizeRelFolder(workflow.folderPath);
                                const shouldResetFolder = currentFolder === '' || ['actions', 'flows', 'cases'].includes(currentFolder);
                                handleUpdateWorkflow(workflow.id, {
                                  workflowType: nextType,
                                  folderPath: shouldResetFolder ? nextDefaultFolder : workflow.folderPath,
                                });
                              }}
                              onFocus={() => setEditingWorkflowId(workflow.id)}
                              className="w-32 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            >
                              <option value="action">Action</option>
                              <option value="flow">Flow</option>
                              <option value="case">Case</option>
                            </select>
                            <input
                              type="text"
                              value={workflow.folderPath}
                              onChange={(e) => handleUpdateWorkflow(workflow.id, { folderPath: e.target.value })}
                              onFocus={() => setEditingWorkflowId(workflow.id)}
                              placeholder="Folder (e.g., flows/auth)"
                              className="flex-1 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            />
                          </div>

                          {/* Composition mode toggle */}
                          <div className="flex items-center gap-3 flex-wrap">
                            <span className="text-xs text-gray-500 dark:text-gray-400">Composition:</span>
                            <div className="flex items-center gap-1 p-0.5 bg-gray-100 dark:bg-gray-700 rounded-lg">
                              <button
                                type="button"
                                onClick={() => handleUpdateWorkflow(workflow.id, { compositionMode: 'inline', referenceWorkflowId: undefined })}
                                className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors ${
                                  workflow.compositionMode === 'inline'
                                    ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                                    : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                                }`}
                                title="Include all steps directly in this workflow"
                              >
                                Inline
                              </button>
                              <button
                                type="button"
                                onClick={() => handleUpdateWorkflow(workflow.id, { compositionMode: 'reference' })}
                                disabled={!workflow.projectId}
                                className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors ${
                                  !workflow.projectId ? 'cursor-not-allowed opacity-50' : ''
                                } ${
                                  workflow.compositionMode === 'reference'
                                    ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                                    : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                                }`}
                                title={workflow.projectId ? "Reference an existing workflow as a subflow" : "Select a project first to use reference mode"}
                              >
                                Reference
                              </button>
                            </div>
                            <span className="text-[10px] text-gray-400 dark:text-gray-500 italic">
                              {workflow.compositionMode === 'inline'
                                ? 'Steps copied directly'
                                : 'Creates subflow reference'}
                            </span>
                          </div>

                          {/* Reference workflow selector (shown in reference mode) */}
                          {workflow.compositionMode === 'reference' && workflow.projectId && (
                            <div className="flex items-center gap-2">
                              <span className="text-xs text-gray-500 dark:text-gray-400">Reference workflow:</span>
                              <select
                                value={workflow.referenceWorkflowId || ''}
                                onChange={(e) => handleUpdateWorkflow(workflow.id, { referenceWorkflowId: e.target.value || undefined })}
                                className="flex-1 px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                              >
                                <option value="">Select existing workflow...</option>
                                {(existingWorkflowsByProjectId[workflow.projectId] || []).map((existing) => (
                                  <option key={existing.id} value={existing.id}>
                                    {existing.name}{existing.workflow_type ? ` (${existing.workflow_type})` : ''}
                                  </option>
                                ))}
                              </select>
                              {(!existingWorkflowsByProjectId[workflow.projectId] || existingWorkflowsByProjectId[workflow.projectId]?.length === 0) && (
                                <span className="text-xs text-amber-500 dark:text-amber-400">
                                  No existing workflows in project
                                </span>
                              )}
                            </div>
                          )}

                          <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
                            File: {(() => {
                              const folder = normalizeRelFolder(workflow.folderPath);
                              const presetConfig = workflow.projectId ? presetConfigsByProjectId[workflow.projectId] : null;
                              const template = presetConfig?.namingTemplates?.[workflow.workflowType];
                              const templatePath = template ? renderNamingTemplate(template, workflow.name, workflow.workflowType) : '';
                              const baseName = templatePath ? fileBasename(templatePath) : `${slugifyWorkflowFilename(workflow.name)}${typedWorkflowFileSuffix(workflow.workflowType)}`;
                              return folder ? `${folder}/${baseName}` : baseName;
                            })()}
                          </div>

                          {/* Test result */}
                          {workflow.testResult && (
                            <div className={`inline-flex items-center gap-1.5 px-2 py-1 text-xs rounded ${
                              workflow.testResult.success
                                ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300'
                                : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300'
                            }`}>
                              {workflow.testResult.success ? (
                                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                                </svg>
                              ) : (
                                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                                </svg>
                              )}
                              {workflow.testResult.message}
                            </div>
                          )}
                        </div>

                        {/* Actions */}
                        <div className="flex-shrink-0 flex items-center gap-1">
                          <button
                            onClick={() => handleTestWorkflow(workflow)}
                            disabled={workflow.isTesting || isReplaying}
                            className="p-2 text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
                            title="Test this workflow"
                          >
                            {workflow.isTesting ? (
                              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                              </svg>
                            ) : (
                              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                              </svg>
                            )}
                          </button>
                          {workflows.length > 1 && (
                            <button
                              onClick={() => handleRemoveWorkflow(workflow.id)}
                              className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                              title="Remove workflow"
                            >
                              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                              </svg>
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}

              {/* Add workflow button */}
              <button
                onClick={handleAddWorkflow}
                className="w-full flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium text-gray-600 dark:text-gray-400 bg-white dark:bg-gray-800 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-xl hover:border-gray-400 dark:hover:border-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add Another Workflow
              </button>
            </div>

            {/* Error */}
            {error && (
              <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
              </div>
            )}
          </div>
        </div>

        {/* Mini-timeline */}
        <div className="border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
          <div className="px-6 py-4">
            <div className="max-w-4xl mx-auto">
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                  Timeline - Drag to select steps
                </span>
                {editingWorkflowId && (
                  <span className="text-xs text-blue-600 dark:text-blue-400">
                    Editing: {workflows.find(w => w.id === editingWorkflowId)?.name || 'Workflow'}
                  </span>
                )}
              </div>

              <div
                ref={timelineRef}
                className="flex gap-1 overflow-x-auto pb-2 select-none"
                onMouseLeave={handleTimelineMouseUp}
                onMouseUp={handleTimelineMouseUp}
              >
                {actions.map((action, idx) => {
                  const color = stepColors[idx];
                  const isInDragRange = dragRange && idx >= dragRange.start && idx <= dragRange.end;
                  const isUnassigned = !color && !isInDragRange;

                  return (
                    <div
                      key={action.id}
                      className={`
                        flex-shrink-0 flex flex-col items-center justify-center
                        w-12 h-14 rounded-lg cursor-pointer transition-all
                        ${isInDragRange
                          ? 'bg-blue-200 dark:bg-blue-700 ring-2 ring-blue-500'
                          : color
                            ? `${color.light} ${color.border} border`
                            : 'bg-gray-100 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600'
                        }
                        ${isUnassigned ? 'opacity-60' : ''}
                      `}
                      onMouseDown={(e) => handleTimelineMouseDown(e, idx)}
                      onMouseMove={(e) => handleTimelineMouseMove(e, idx)}
                      title={`Step ${idx + 1}: ${getShortActionLabel(action)}`}
                    >
                      <span className={`text-[10px] font-medium ${color?.text || 'text-gray-600 dark:text-gray-400'}`}>
                        {idx + 1}
                      </span>
                      <span className={`text-[9px] ${color?.text || 'text-gray-500 dark:text-gray-500'}`}>
                        {getShortActionLabel(action)}
                      </span>
                    </div>
                  );
                })}
              </div>

              {/* Legend */}
              {workflows.length > 1 && (
                <div className="flex flex-wrap gap-3 mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
                  {workflows.map((wf, idx) => {
                    const color = getWorkflowColor(idx);
                    return (
                      <button
                        key={wf.id}
                        onClick={() => setEditingWorkflowId(wf.id)}
                        className={`flex items-center gap-1.5 px-2 py-1 text-xs rounded-lg transition-colors ${
                          editingWorkflowId === wf.id
                            ? `${color.light} ${color.border} border-2`
                            : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                        }`}
                      >
                        <div className={`w-2.5 h-2.5 rounded ${color.bg}`} />
                        <span className={color.text}>{wf.name || `Workflow ${idx + 1}`}</span>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Footer actions */}
      <div className="px-6 py-4 bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700">
        <div className="max-w-4xl mx-auto flex items-center justify-between">
          <div className="text-sm text-gray-500 dark:text-gray-400">
            {workflows.length} workflow{workflows.length !== 1 ? 's' : ''} queued
            {unassignedSteps.length > 0 && (
              <span className="ml-2 text-yellow-600 dark:text-yellow-400">
                ({unassignedSteps.length} unassigned step{unassignedSteps.length !== 1 ? 's' : ''})
              </span>
            )}
          </div>

          <div className="flex items-center gap-3">
            {generationProgress && (
              <span className="text-sm text-blue-600 dark:text-blue-400">
                Generating {generationProgress.current}/{generationProgress.total}...
              </span>
            )}
            <button
              onClick={handleGenerateAll}
              disabled={!canGenerate || isGeneratingAll || isReplaying || isGenerating}
              className="flex items-center gap-2 px-6 py-2.5 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isGeneratingAll ? (
                <>
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Generating...
                </>
              ) : (
                <>
                  Generate All Workflows
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                  </svg>
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
