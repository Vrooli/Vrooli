/**
 * useNewExportFlow Hook
 *
 * Orchestrates the "New Export" flow from the Exports tab.
 * Manages the execution picker dialog and inline export dialog.
 */

import { useState, useCallback, useEffect } from 'react';
import { useExecutionStore, type ExecutionWithExportability, type Execution } from '@/domains/executions/store';
import { useExportStore } from '@/domains/exports/store';
import { useExecutionFilters, type UseExecutionFiltersReturn } from './useExecutionFilters';
import { getConfig } from '@/config';
import { logger } from '@utils/logger';

export interface UseNewExportFlowReturn {
  // Selection dialog state
  isSelectDialogOpen: boolean;
  openSelectDialog: () => void;
  closeSelectDialog: () => void;

  // Execution data for selection
  executions: ExecutionWithExportability[];
  isLoadingExecutions: boolean;

  // Filter state
  filters: UseExecutionFiltersReturn;

  // Selection handling
  handleExecutionSelected: (executionId: string, workflowId: string) => void;

  // Helpers
  getExportCountForExecution: (executionId: string) => number;

  // Inline export state (for when we can't navigate)
  selectedExecution: Execution | null;
  selectedWorkflowName: string;
  isLoadingSelectedExecution: boolean;
  clearSelectedExecution: () => void;
}

export interface UseNewExportFlowOptions {
  /** Called when an execution is selected and can be navigated to */
  onViewExecution: (executionId: string, workflowId: string) => void;
}

export function useNewExportFlow({ onViewExecution }: UseNewExportFlowOptions): UseNewExportFlowReturn {
  const [isSelectDialogOpen, setIsSelectDialogOpen] = useState(false);
  const [executions, setExecutions] = useState<ExecutionWithExportability[]>([]);
  const [isLoadingExecutions, setIsLoadingExecutions] = useState(false);

  // State for inline export (when workflow has no project)
  const [selectedExecution, setSelectedExecution] = useState<Execution | null>(null);
  const [selectedWorkflowName, setSelectedWorkflowName] = useState('');
  const [isLoadingSelectedExecution, setIsLoadingSelectedExecution] = useState(false);

  const { loadExecutionsWithExportability, loadExecution, refreshTimeline, setAutoOpenExport } = useExecutionStore();
  const { fetchExports, getExportsByExecutionId } = useExportStore();
  const filters = useExecutionFilters();

  // Load executions with exportability when dialog opens
  useEffect(() => {
    if (!isSelectDialogOpen) return;

    const loadData = async () => {
      setIsLoadingExecutions(true);
      try {
        // Load executions with exportability info
        const result = await loadExecutionsWithExportability();
        setExecutions(result);

        // Also ensure exports are loaded for the "already exported" badges
        await fetchExports();
      } finally {
        setIsLoadingExecutions(false);
      }
    };

    loadData();
  }, [isSelectDialogOpen, loadExecutionsWithExportability, fetchExports]);

  const openSelectDialog = useCallback(() => {
    setIsSelectDialogOpen(true);
  }, []);

  const closeSelectDialog = useCallback(() => {
    setIsSelectDialogOpen(false);
    filters.resetFilters();
  }, [filters]);

  const clearSelectedExecution = useCallback(() => {
    setSelectedExecution(null);
    setSelectedWorkflowName('');
  }, []);

  const handleExecutionSelected = useCallback(
    async (executionId: string, workflowId: string) => {
      // Close the selection dialog
      closeSelectDialog();
      setIsLoadingSelectedExecution(true);

      try {
        // Load the execution and its timeline (timeline is loaded async by loadExecution,
        // so we must explicitly wait for it to be ready for export)
        await loadExecution(executionId);
        await refreshTimeline(executionId);

        // Check if workflow has a project
        const config = await getConfig();
        const workflowResponse = await fetch(`${config.API_URL}/workflows/${workflowId}`);

        if (!workflowResponse.ok) {
          throw new Error(`Failed to fetch workflow: ${workflowResponse.status}`);
        }

        const workflowData = await workflowResponse.json();
        const projectId = workflowData.project_id ?? workflowData.projectId;
        const workflowName = workflowData.name ?? 'Workflow';

        if (projectId) {
          // Workflow has a project - navigate to it and auto-open export dialog
          setAutoOpenExport(true);
          onViewExecution(executionId, workflowId);
        } else {
          // No project - show inline export dialog
          // Get the loaded execution from the store
          const currentExecution = useExecutionStore.getState().currentExecution;
          if (currentExecution) {
            setSelectedExecution(currentExecution);
            setSelectedWorkflowName(workflowName);
          } else {
            throw new Error('Failed to load execution');
          }
        }
      } catch (error) {
        logger.error('Failed to select execution for export', { executionId, workflowId }, error);
        // Clear any partial state
        setSelectedExecution(null);
        setSelectedWorkflowName('');
      } finally {
        setIsLoadingSelectedExecution(false);
      }
    },
    [closeSelectDialog, loadExecution, refreshTimeline, onViewExecution, setAutoOpenExport]
  );

  const getExportCountForExecution = useCallback(
    (executionId: string): number => {
      return getExportsByExecutionId(executionId).length;
    },
    [getExportsByExecutionId]
  );

  return {
    isSelectDialogOpen,
    openSelectDialog,
    closeSelectDialog,
    executions,
    isLoadingExecutions,
    filters,
    handleExecutionSelected,
    getExportCountForExecution,
    // Inline export state
    selectedExecution,
    selectedWorkflowName,
    isLoadingSelectedExecution,
    clearSelectedExecution,
  };
}
