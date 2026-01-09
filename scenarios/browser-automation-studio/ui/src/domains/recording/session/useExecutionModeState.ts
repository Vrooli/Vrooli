/**
 * useExecutionModeState Hook
 *
 * Encapsulates all execution mode state and handlers for the RecordingSession component.
 * This includes:
 * - Workflow selection state
 * - Execution lifecycle (start, stop, rerun)
 * - Export functionality
 * - Workflow nodes/edges for timeline
 */

import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast';
import { getConfig } from '@/config';
import {
  useExecutionStore,
  useStartWorkflow,
  useExecutionEvents,
  type Execution,
} from '@/domains/executions';
import { useExecutionExport } from '@/domains/executions/viewer/useExecutionExport';
import { useReplayCustomization } from '@/domains/executions/viewer/useReplayCustomization';
import { useExportStore } from '@/domains/exports';
import type { ExecutionConfigSettings } from '../timeline/WorkflowInfoCard';
import type { StreamSettingsValues } from '../capture/StreamSettings';
import { DEFAULT_STREAM_FPS, DEFAULT_STREAM_QUALITY } from '../constants';

/** Workflow node shape from API */
export interface WorkflowNode {
  id: string;
  type?: string;
  data?: Record<string, unknown>;
  action?: {
    type: string;
    metadata?: { label?: string };
    navigate?: { url?: string };
  };
}

/** Workflow edge shape from API */
export interface WorkflowEdge {
  source: string;
  target: string;
}

interface UseExecutionModeStateOptions {
  /** Initial execution ID (from props) */
  initialExecutionId?: string | null;
  /** Initial workflow ID (from props) */
  initialWorkflowId?: string;
  /** Initial project ID (from props) */
  initialProjectId?: string;
  /** Session profile ID for execution */
  sessionProfileId: string | null;
  /** Stream settings ref for frame streaming config */
  streamSettingsRef: React.RefObject<StreamSettingsValues | null>;
}

interface UseExecutionModeStateReturn {
  // Workflow selection
  selectedWorkflowId: string | null;
  selectedProjectId: string | null;
  selectedWorkflowName: string | null;
  setSelectedWorkflowId: (id: string | null) => void;
  setSelectedProjectId: (id: string | null) => void;
  setSelectedWorkflowName: (name: string | null) => void;

  // Execution state
  localExecutionId: string | null;
  setLocalExecutionId: (id: string | null) => void;
  executionStatus: Execution['status'] | null;
  currentExecution: Execution | null;
  isExecuting: boolean;
  canRun: boolean;
  isReadOnly: boolean;

  // Workflow definition (for timeline)
  workflowNodes: WorkflowNode[];
  workflowEdges: WorkflowEdge[];

  // Handlers
  handleWorkflowSelect: (
    workflowId: string,
    projectId: string,
    name: string,
    defaultSessionId?: string | null
  ) => void;
  handleRun: (overrides?: ExecutionConfigSettings) => Promise<void>;
  handleStop: () => Promise<void>;
  handleRerunExecution: () => void;
  handleEditWorkflow: () => void;

  // Export
  exportController: ReturnType<typeof useExecutionExport>;
  exportDialogTitleId: string;
  exportDialogDescriptionId: string;

  // Workflow picker
  showWorkflowPicker: boolean;
  setShowWorkflowPicker: (show: boolean) => void;

  // Sidebar state for execution
  selectedScreenshotIndex: number;
  setSelectedScreenshotIndex: (index: number) => void;
  logsFilter: 'all' | 'error' | 'warning' | 'info' | 'success';
  setLogsFilter: (filter: 'all' | 'error' | 'warning' | 'info' | 'success') => void;

  // Callback to set session profile (passed from parent)
  onSessionProfileSelect?: (profileId: string) => void;
}

export function useExecutionModeState({
  initialExecutionId,
  initialWorkflowId,
  initialProjectId,
  sessionProfileId,
  streamSettingsRef,
}: UseExecutionModeStateOptions): UseExecutionModeStateReturn {
  const navigate = useNavigate();

  // Workflow selection state
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string | null>(
    initialWorkflowId ?? null
  );
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(
    initialProjectId ?? null
  );
  const [selectedWorkflowName, setSelectedWorkflowName] = useState<string | null>(null);
  const [localExecutionId, setLocalExecutionId] = useState<string | null>(
    initialExecutionId ?? null
  );
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false);

  // Workflow nodes/edges for timeline pre-population
  const [workflowNodes, setWorkflowNodes] = useState<WorkflowNode[]>([]);
  const [workflowEdges, setWorkflowEdges] = useState<WorkflowEdge[]>([]);

  // Execution sidebar state
  const [selectedScreenshotIndex, setSelectedScreenshotIndex] = useState(0);
  const [logsFilter, setLogsFilter] = useState<'all' | 'error' | 'warning' | 'info' | 'success'>(
    'all'
  );

  // Session profile callback ref (set by parent)
  const onSessionProfileSelectRef = useRef<((profileId: string) => void) | undefined>();

  // Execution store
  const currentExecution = useExecutionStore((s) => s.currentExecution);
  const stopExecution = useExecutionStore((s) => s.stopExecution);

  // Derived execution status
  const executionStatus =
    currentExecution?.id === localExecutionId ? currentExecution.status : null;
  const canRun = !!selectedWorkflowId && !localExecutionId;
  const isExecuting = executionStatus === 'running';
  const isReadOnly = !!(localExecutionId && executionStatus);

  // Subscribe to execution events
  useExecutionEvents(
    localExecutionId && executionStatus
      ? { id: localExecutionId, status: executionStatus }
      : undefined
  );

  // Export infrastructure
  const { createExport } = useExportStore();
  const exportDialogTitleId = useId();
  const exportDialogDescriptionId = useId();
  const replayCustomization = useReplayCustomization({
    executionId: localExecutionId ?? '',
  });

  // Default execution for hook
  const defaultExecution: Execution = useMemo(
    () => ({
      id: '',
      workflowId: '',
      status: 'pending' as const,
      startedAt: new Date(),
      screenshots: [],
      timeline: [],
      logs: [],
      progress: 0,
    }),
    []
  );

  const exportController = useExecutionExport({
    execution: currentExecution ?? defaultExecution,
    replayFrames: currentExecution?.timeline ?? [],
    workflowName: selectedWorkflowName ?? 'Workflow',
    replayCustomization,
    createExport: createExport as Parameters<typeof useExecutionExport>[0]['createExport'],
  });

  // Start workflow hook
  const { startWorkflow } = useStartWorkflow({
    onSuccess: (execId) => {
      setLocalExecutionId(execId);
    },
    onError: (error) => {
      toast.error(error.message || 'Failed to start workflow execution');
    },
  });

  // Handle workflow selection from picker
  const handleWorkflowSelect = useCallback(
    (
      workflowId: string,
      projectId: string,
      name: string,
      defaultSessionId?: string | null
    ) => {
      setSelectedWorkflowId(workflowId);
      setSelectedProjectId(projectId);
      setSelectedWorkflowName(name);
      setLocalExecutionId(null);
      setShowWorkflowPicker(false);

      // Auto-select the workflow's default session if provided
      if (defaultSessionId && onSessionProfileSelectRef.current) {
        onSessionProfileSelectRef.current(defaultSessionId);
      }
    },
    []
  );

  // Handle Run button click
  const handleRun = useCallback(
    async (overrides?: ExecutionConfigSettings) => {
      if (!selectedWorkflowId) return;
      try {
        const currentStreamSettings = streamSettingsRef.current;
        await startWorkflow({
          workflowId: selectedWorkflowId,
          sessionProfileId,
          overrides: overrides
            ? {
                viewport_width: overrides.viewportWidth,
                viewport_height: overrides.viewportHeight,
                timeout_ms: overrides.actionTimeoutSeconds * 1000,
                navigation_wait_until: overrides.navigationWaitUntil,
                continue_on_error: overrides.continueOnError,
              }
            : undefined,
          artifactProfile: overrides?.artifactProfile,
          frameStreaming: {
            enabled: true,
            quality: currentStreamSettings?.quality ?? DEFAULT_STREAM_QUALITY,
            fps: currentStreamSettings?.fps ?? DEFAULT_STREAM_FPS,
          },
        });
      } catch {
        // Error handled by onError callback
      }
    },
    [selectedWorkflowId, sessionProfileId, startWorkflow, streamSettingsRef]
  );

  // Handle Stop button click
  const handleStop = useCallback(async () => {
    if (localExecutionId) {
      await stopExecution(localExecutionId);
    }
  }, [localExecutionId, stopExecution]);

  // Handle Re-run button click
  const handleRerunExecution = useCallback(() => {
    setLocalExecutionId(null);
    setTimeout(() => {
      if (selectedWorkflowId) {
        void handleRun();
      }
    }, 100);
  }, [selectedWorkflowId, handleRun]);

  // Handle Edit Workflow button click
  const handleEditWorkflow = useCallback(() => {
    if (selectedWorkflowId && selectedProjectId) {
      navigate(`/projects/${selectedProjectId}/workflows/${selectedWorkflowId}`);
    }
  }, [selectedWorkflowId, selectedProjectId, navigate]);

  // Fetch workflow definition when workflow is selected
  useEffect(() => {
    if (!selectedWorkflowId) {
      setWorkflowNodes([]);
      setWorkflowEdges([]);
      return;
    }

    const fetchWorkflowDefinition = async () => {
      try {
        const config = await getConfig();
        const res = await fetch(`${config.API_URL}/workflows/${selectedWorkflowId}`);
        if (!res.ok) {
          console.warn('Failed to fetch workflow definition');
          return;
        }
        const data = await res.json();

        // Parse workflow definition
        if (data.definition) {
          const def =
            typeof data.definition === 'string'
              ? JSON.parse(data.definition)
              : data.definition;
          setWorkflowNodes(def.nodes || []);
          setWorkflowEdges(def.edges || []);
        }

        // Set workflow name if not already set
        if (!selectedWorkflowName && data.name) {
          setSelectedWorkflowName(data.name);
        }
      } catch (err) {
        console.warn('Error fetching workflow definition:', err);
      }
    };

    void fetchWorkflowDefinition();
  }, [selectedWorkflowId, selectedWorkflowName]);

  return {
    // Workflow selection
    selectedWorkflowId,
    selectedProjectId,
    selectedWorkflowName,
    setSelectedWorkflowId,
    setSelectedProjectId,
    setSelectedWorkflowName,

    // Execution state
    localExecutionId,
    setLocalExecutionId,
    executionStatus,
    currentExecution,
    isExecuting,
    canRun,
    isReadOnly,

    // Workflow definition
    workflowNodes,
    workflowEdges,

    // Handlers
    handleWorkflowSelect,
    handleRun,
    handleStop,
    handleRerunExecution,
    handleEditWorkflow,

    // Export
    exportController,
    exportDialogTitleId,
    exportDialogDescriptionId,

    // Workflow picker
    showWorkflowPicker,
    setShowWorkflowPicker,

    // Sidebar state
    selectedScreenshotIndex,
    setSelectedScreenshotIndex,
    logsFilter,
    setLogsFilter,
  };
}
