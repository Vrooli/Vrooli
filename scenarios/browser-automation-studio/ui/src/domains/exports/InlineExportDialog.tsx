/**
 * InlineExportDialog
 *
 * Renders the export dialog for an execution directly in the Exports tab,
 * without requiring navigation to the workflow view.
 * Used when workflows don't have an associated project.
 */

import React, { useEffect } from 'react';
import { useExecutionExport } from '@/domains/executions/viewer/useExecutionExport';
import { useReplayCustomization } from '@/domains/executions/viewer/useReplayCustomization';
import { useExportStore } from '@/domains/exports/store';
import { ExportDialog } from '@/domains/executions/viewer/ExportDialog';
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
  const handleClose = () => {
    exportController.closeExportDialog();
    onClose();
  };

  // Handle success panel dismiss
  const handleDismissSuccess = () => {
    exportController.dismissExportSuccess();
    onClose();
  };

  return (
    <>
      <ExportDialog
        isOpen={exportController.isExportDialogOpen}
        onClose={handleClose}
        onConfirm={exportController.confirmExport}
        dialogTitleId="inline-export-dialog-title"
        dialogDescriptionId="inline-export-dialog-description"
        {...exportController.exportDialogProps}
      />

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
