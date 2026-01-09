/**
 * InlineExportDialog
 *
 * Renders the export dialog for an execution directly in the Exports tab,
 * without requiring navigation to the workflow view.
 * Used when workflows don't have an associated project.
 */

import React, { useCallback, useEffect, useMemo } from 'react';
import { useExecutionExport } from '@/domains/executions/viewer/useExecutionExport';
import { useReplayCustomization } from '@/domains/executions/viewer/useReplayCustomization';
import { useExportStore } from '@/domains/exports/store';
import {
  ExportDialog,
  ExportDialogProvider,
  buildExportDialogContextValue,
} from '@/domains/executions/export';
import { ExportSuccessPanel } from './ExportSuccessPanel';
import type { Execution } from '@/domains/executions/store';

interface InlineExportDialogProps {
  execution: Execution;
  workflowName: string;
  onClose: () => void;
}

export const InlineExportDialog: React.FC<InlineExportDialogProps> = ({
  execution,
  workflowName,
  onClose,
}) => {
  const { createExport } = useExportStore();
  const replayCustomization = useReplayCustomization({ executionId: execution.id });

  const exportController = useExecutionExport({
    execution,
    replayFrames: execution.timeline ?? [],
    workflowName,
    replayCustomization,
    createExport: createExport as Parameters<typeof useExecutionExport>[0]['createExport'],
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
    onClose();
  };

  // Build context value from hook props
  const contextValue = useMemo(
    () =>
      buildExportDialogContextValue({
        dialogTitleId: 'inline-export-dialog-title',
        dialogDescriptionId: 'inline-export-dialog-description',
        onClose: handleClose,
        onConfirm: exportController.confirmExport,
        ...exportController.exportDialogProps,
      }),
    [handleClose, exportController.confirmExport, exportController.exportDialogProps],
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
