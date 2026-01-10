/**
 * InlineExportDialog
 *
 * Renders the export dialog for an execution directly in the Exports tab,
 * without requiring navigation to the workflow view.
 * Used when workflows don't have an associated project.
 *
 * Also supports edit mode when `exportToEdit` is provided - the dialog
 * is pre-populated with the existing export's settings and on confirm,
 * replaces the old export with a new one.
 */

import React, { useCallback, useEffect, useMemo } from 'react';
import { useExecutionExport } from '@/domains/executions/viewer/useExecutionExport';
import { useReplayCustomization } from '@/domains/executions/viewer/useReplayCustomization';
import { useExportStore, type Export } from '@/domains/exports/store';
import {
  ExportDialog,
  ExportDialogProvider,
  buildExportDialogContextValue,
} from '@/domains/executions/export';
import { ExportSuccessPanel } from './ExportSuccessPanel';
import { parseExportSettings } from './utils/parseExportSettings';
import type { Execution } from '@/domains/executions/store';

interface InlineExportDialogProps {
  execution: Execution;
  workflowName: string;
  onClose: () => void;
  /** When provided, the dialog is in edit mode - pre-populated with existing settings */
  exportToEdit?: Export;
  /** Called after successfully replacing the export (edit mode only) */
  onExportUpdated?: () => void;
}

export const InlineExportDialog: React.FC<InlineExportDialogProps> = ({
  execution,
  workflowName,
  onClose,
  exportToEdit,
  onExportUpdated,
}) => {
  const { createExport, replaceExport } = useExportStore();
  const replayCustomization = useReplayCustomization({ executionId: execution.id });

  // Parse initial settings from existing export (edit mode)
  const initialSettings = useMemo(() => {
    if (!exportToEdit) return undefined;
    const parsed = parseExportSettings(exportToEdit);
    return {
      format: parsed.format,
      fileStem: parsed.fileStem,
    };
  }, [exportToEdit]);

  // In edit mode, use replaceExport instead of createExport
  const handleCreateExport = useCallback(
    async (input: Parameters<typeof createExport>[0]) => {
      if (exportToEdit) {
        return replaceExport(exportToEdit.id, input);
      }
      return createExport(input);
    },
    [exportToEdit, createExport, replaceExport]
  );

  const exportController = useExecutionExport({
    execution,
    replayFrames: execution.timeline ?? [],
    workflowName,
    replayCustomization,
    createExport: handleCreateExport as Parameters<typeof useExecutionExport>[0]['createExport'],
    initialSettings,
  });

  // Auto-open the export dialog when component mounts
  useEffect(() => {
    if (!exportController.isExportDialogOpen) {
      exportController.openExportDialog();
    }
  }, []);

  // Handle dialog close - also notify parent
  const handleClose = useCallback(() => {
    exportController.closeExportDialog();
    onClose();
  }, [exportController, onClose]);

  // Handle success panel dismiss
  const handleDismissSuccess = () => {
    exportController.dismissExportSuccess();
    if (exportToEdit && onExportUpdated) {
      onExportUpdated();
    }
    onClose();
  };

  const isEditMode = Boolean(exportToEdit);

  // Build context value from hook props
  const contextValue = useMemo(
    () =>
      buildExportDialogContextValue({
        dialogTitleId: isEditMode ? 'edit-export-dialog-title' : 'inline-export-dialog-title',
        dialogDescriptionId: isEditMode ? 'edit-export-dialog-description' : 'inline-export-dialog-description',
        onClose: handleClose,
        onConfirm: exportController.confirmExport,
        isEditMode,
        ...exportController.exportDialogProps,
      }),
    [handleClose, exportController.confirmExport, exportController.exportDialogProps, isEditMode],
  );

  return (
    <>
      <ExportDialogProvider value={contextValue}>
        <ExportDialog isOpen={exportController.isExportDialogOpen} />
      </ExportDialogProvider>

      {exportController.showExportSuccess && exportController.lastCreatedExport && (
        <ExportSuccessPanel
          export_={exportController.lastCreatedExport}
          onClose={handleDismissSuccess}
          onViewInLibrary={handleDismissSuccess}
          onViewExecution={() => {
            // Already viewing from exports tab, just dismiss
            handleDismissSuccess();
          }}
        />
      )}
    </>
  );
};

export default InlineExportDialog;
