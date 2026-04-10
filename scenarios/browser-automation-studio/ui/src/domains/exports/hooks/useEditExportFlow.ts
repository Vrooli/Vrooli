/**
 * useEditExportFlow Hook
 *
 * Orchestrates the "Edit Export" flow from the Exports tab.
 * Loads the execution for an existing export and opens the inline export dialog.
 *
 * Uses a testable seam for workflow API calls via WorkflowApiClient.
 */

import { useState, useCallback } from 'react';
import { useExecutionStore, type Execution } from '@/domains/executions/store';
import { type Export } from '@/domains/exports/store';
import {
  defaultWorkflowApiClient,
  type WorkflowApiClient,
} from '../api/workflowClient';
import { logger } from '@utils/logger';

export interface UseEditExportFlowReturn {
  // Export being edited
  exportToEdit: Export | null;

  // Loaded execution for the export
  executionForEdit: Execution | null;
  workflowNameForEdit: string;

  // Loading state
  isLoadingExecution: boolean;

  // Actions
  openEditDialog: (export_: Export) => Promise<void>;
  closeEditDialog: () => void;
  clearEditState: () => void;
}

export interface UseEditExportFlowOptions {
  /** Optional workflow API client for testing. Defaults to defaultWorkflowApiClient. */
  workflowApiClient?: WorkflowApiClient;
}

export function useEditExportFlow({
  workflowApiClient = defaultWorkflowApiClient,
}: UseEditExportFlowOptions = {}): UseEditExportFlowReturn {
  const [exportToEdit, setExportToEdit] = useState<Export | null>(null);
  const [executionForEdit, setExecutionForEdit] = useState<Execution | null>(null);
  const [workflowNameForEdit, setWorkflowNameForEdit] = useState('');
  const [isLoadingExecution, setIsLoadingExecution] = useState(false);

  const { loadExecution, refreshTimeline } = useExecutionStore();

  const openEditDialog = useCallback(
    async (export_: Export) => {
      setIsLoadingExecution(true);
      setExportToEdit(export_);

      try {
        // Load the execution and its timeline
        await loadExecution(export_.executionId);
        await refreshTimeline(export_.executionId);

        // Get workflow name if available
        let workflowName = export_.workflowName ?? '';
        if (!workflowName && export_.workflowId) {
          try {
            const workflowInfo = await workflowApiClient.fetchWorkflow(export_.workflowId);
            workflowName = workflowInfo.name;
          } catch {
            // Workflow might not exist anymore, use fallback
            workflowName = 'Workflow';
          }
        }

        // Get the loaded execution from the store
        const currentExecution = useExecutionStore.getState().currentExecution;
        if (currentExecution) {
          setExecutionForEdit(currentExecution);
          setWorkflowNameForEdit(workflowName);
        } else {
          throw new Error('Failed to load execution');
        }
      } catch (error) {
        logger.error('Failed to load execution for export edit', { exportId: export_.id, executionId: export_.executionId }, error);
        // Clear state on error
        setExportToEdit(null);
        setExecutionForEdit(null);
        setWorkflowNameForEdit('');
      } finally {
        setIsLoadingExecution(false);
      }
    },
    [loadExecution, refreshTimeline, workflowApiClient]
  );

  const closeEditDialog = useCallback(() => {
    setExportToEdit(null);
    setExecutionForEdit(null);
    setWorkflowNameForEdit('');
  }, []);

  const clearEditState = useCallback(() => {
    setExportToEdit(null);
    setExecutionForEdit(null);
    setWorkflowNameForEdit('');
  }, []);

  return {
    exportToEdit,
    executionForEdit,
    workflowNameForEdit,
    isLoadingExecution,
    openEditDialog,
    closeEditDialog,
    clearEditState,
  };
}
