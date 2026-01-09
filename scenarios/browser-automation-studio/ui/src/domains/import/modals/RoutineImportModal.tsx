/**
 * RoutineImportModal
 *
 * Modal for importing workflow/routine files into a project.
 * Supports both drag-and-drop file upload and folder browsing.
 * Responsive layout: side-by-side on desktop, stacked on mobile.
 */

import { useState, useEffect, useCallback, useId } from 'react';
import { FileCode, Loader2, X, Upload } from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import toast from 'react-hot-toast';

import { ImportSourceSelector } from '../components/ImportSourceSelector';
import { StatusBadge, AlertBox } from '../components/ValidationStatus';
import { useRoutineImport, type InspectRoutineResponse } from '../hooks/useRoutineImport';
import type { RoutineImportModalProps, FolderEntry, SelectedFile } from '../types';
import { WORKFLOW_EXTENSIONS } from '../constants';
import { getApiBase } from '../../../config';

type Step = 'select' | 'preview';

export function RoutineImportModal({
  isOpen,
  onClose,
  projectId,
  onSuccess,
}: RoutineImportModalProps) {
  const titleId = useId();

  const [step, setStep] = useState<Step>('select');
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [currentFolderPath, setCurrentFolderPath] = useState('');
  const [workflowName, setWorkflowName] = useState('');
  const [overwrite, setOverwrite] = useState(false);
  const [isUploadingFile, setIsUploadingFile] = useState(false);

  const {
    isInspecting,
    isImporting,
    inspectResult,
    error,
    inspectFile,
    importRoutine,
    clearError,
    reset,
  } = useRoutineImport({ projectId });

  // Reset state when modal closes
  useEffect(() => {
    if (!isOpen) {
      setStep('select');
      setSelectedPath(null);
      setCurrentFolderPath('');
      setWorkflowName('');
      setOverwrite(false);
      setIsUploadingFile(false);
      reset();
    }
  }, [isOpen, reset]);

  // Update name from preview
  useEffect(() => {
    if (inspectResult?.preview?.name && !workflowName) {
      setWorkflowName(inspectResult.preview.name);
    }
  }, [inspectResult, workflowName]);

  // Handle file selection from drop zone - upload to temp then inspect
  const handleFilesSelected = useCallback(
    async (files: SelectedFile[]) => {
      if (files.length === 0) return;

      const file = files[0];
      if (!file.validation?.isValid) {
        return;
      }

      setIsUploadingFile(true);
      clearError();

      try {
        // Upload file to temp location on server
        const formData = new FormData();
        formData.append('file', file.file);
        formData.append('project_id', projectId);

        const uploadResponse = await fetch(`${getApiBase()}/projects/${projectId}/routines/upload-temp`, {
          method: 'POST',
          body: formData,
        });

        if (!uploadResponse.ok) {
          const errorData = await uploadResponse.json();
          throw new Error(errorData.message || 'Failed to upload file');
        }

        const { temp_path } = await uploadResponse.json();
        setSelectedPath(temp_path);

        // Inspect the uploaded file
        const result = await inspectFile(temp_path);
        if (result && result.exists && result.is_valid) {
          setStep('preview');
        }
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to upload file');
      } finally {
        setIsUploadingFile(false);
      }
    },
    [projectId, inspectFile, clearError]
  );

  // Handle folder navigation in browser
  const handleFolderNavigate = useCallback((path: string) => {
    setCurrentFolderPath(path);
  }, []);

  // Handle folder path input change
  const handleFolderPathChange = useCallback((newPath: string) => {
    setCurrentFolderPath(newPath);
    clearError();
  }, [clearError]);

  // Handle folder selection from browser
  const handleFolderSelect = useCallback(
    async (entry: FolderEntry) => {
      setSelectedPath(entry.path);
      clearError();

      const result = await inspectFile(entry.path);
      if (result && result.exists && result.is_valid) {
        setStep('preview');
      }
    },
    [inspectFile, clearError]
  );

  // Handle import
  const handleImport = useCallback(async () => {
    if (!selectedPath) return;

    const result = await importRoutine({
      file_path: selectedPath,
      name: workflowName.trim() || undefined,
      overwrite_if_exists: overwrite,
    });

    if (result) {
      toast.success(`Workflow "${result.name}" imported successfully`);
      onSuccess?.({ id: result.workflow_id, name: result.name });
      onClose();
    }
  }, [selectedPath, workflowName, overwrite, importRoutine, onSuccess, onClose]);

  // Handle back to selection
  const handleBack = useCallback(() => {
    setStep('select');
    setSelectedPath(null);
    setWorkflowName('');
    clearError();
  }, [clearError]);

  const showPreview = step === 'preview' && inspectResult;
  const isLoading = isInspecting || isUploadingFile;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="wide"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden !p-0"
    >
      <div className="flex flex-col max-h-[inherit]">
        {/* Header - fixed at top, condensed on mobile */}
        <div className="relative px-4 pt-4 pb-2 md:px-6 md:pt-6 md:pb-4 flex-shrink-0">
          {/* Close button */}
          <button
            type="button"
            onClick={onClose}
            className="absolute top-3 right-3 md:top-4 md:right-4 p-1.5 md:p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close"
            disabled={isImporting}
          >
            <X size={18} />
          </button>

          {/* Icon & Title - inline on mobile, stacked on desktop */}
          <div className="flex items-center gap-3 md:flex-col md:items-center md:text-center md:gap-0 mb-1 md:mb-2">
            <div className="w-10 h-10 md:w-14 md:h-14 bg-gradient-to-br from-purple-500/30 to-violet-500/20 rounded-xl md:rounded-2xl flex items-center justify-center md:mb-4 shadow-lg shadow-purple-500/10 flex-shrink-0">
              <FileCode size={20} className="md:hidden text-purple-400" />
              <FileCode size={28} className="hidden md:block text-purple-400" />
            </div>
            <div>
              <h2 id={titleId} className="text-lg md:text-xl font-bold text-white tracking-tight">
                Import Workflow
              </h2>
              <p className="text-xs md:text-sm text-gray-400 md:mt-1">
                {showPreview
                  ? 'Review and import the workflow'
                  : 'Drag a file or browse to select'}
              </p>
            </div>
          </div>
        </div>

        {/* Content - scrollable */}
        <div className="px-6 pb-6 flex-1 overflow-y-auto min-h-0">
          {/* Error */}
          {error && (
            <AlertBox
              type="error"
              title="Error"
              message={error}
              className="mb-4"
            />
          )}

          {!showPreview ? (
            <SelectStep
              projectId={projectId}
              folderPath={currentFolderPath}
              isLoading={isLoading}
              onFolderPathChange={handleFolderPathChange}
              onFolderNavigate={handleFolderNavigate}
              onFilesSelected={handleFilesSelected}
              onFolderSelect={handleFolderSelect}
              onClose={onClose}
            />
          ) : (
            <PreviewStep
              inspectResult={inspectResult}
              workflowName={workflowName}
              overwrite={overwrite}
              isImporting={isImporting}
              error={error}
              onNameChange={setWorkflowName}
              onOverwriteChange={setOverwrite}
              onBack={handleBack}
              onImport={handleImport}
              onClose={onClose}
            />
          )}
        </div>
      </div>
    </ResponsiveDialog>
  );
}

/** Selection step with dual-input pattern */
interface SelectStepProps {
  projectId: string;
  folderPath: string;
  isLoading: boolean;
  onFolderPathChange: (path: string) => void;
  onFolderNavigate: (path: string) => void;
  onFilesSelected: (files: SelectedFile[]) => void;
  onFolderSelect: (entry: FolderEntry) => void;
  onClose: () => void;
}

function SelectStep({
  projectId,
  folderPath,
  isLoading,
  onFolderPathChange,
  onFolderNavigate,
  onFilesSelected,
  onFolderSelect,
  onClose,
}: SelectStepProps) {
  return (
    <>
      <ImportSourceSelector
        // DropZone props
        dropZoneVariant="file"
        dropZoneAccept={WORKFLOW_EXTENSIONS as unknown as string[]}
        dropZoneLabel="Drag workflow file here"
        dropZoneDescription="or click to browse your device"
        onFilesSelected={onFilesSelected}

        // FolderBrowser props
        folderBrowserMode="workflows"
        projectId={projectId}
        onFolderSelect={onFolderSelect}
        onNavigate={onFolderNavigate}
        showRegistered={false}

        // Path input
        showPathInput={true}
        pathValue={folderPath}
        pathLabel="Current Location"
        pathPlaceholder="Browsing workflows folder..."
        onPathChange={onFolderPathChange}

        // State
        disabled={isLoading}
        isLoading={isLoading}
      />

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 pt-4">
        <button
          type="button"
          onClick={onClose}
          className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
          disabled={isLoading}
        >
          Cancel
        </button>
      </div>
    </>
  );
}

/** Preview step */
interface PreviewStepProps {
  inspectResult: InspectRoutineResponse;
  workflowName: string;
  overwrite: boolean;
  isImporting: boolean;
  error: string | null;
  onNameChange: (name: string) => void;
  onOverwriteChange: (overwrite: boolean) => void;
  onBack: () => void;
  onImport: () => void;
  onClose: () => void;
}

function PreviewStep({
  inspectResult,
  workflowName,
  overwrite,
  isImporting,
  onNameChange,
  onOverwriteChange,
  onBack,
  onImport,
  onClose,
}: PreviewStepProps) {
  const preview = inspectResult.preview;

  return (
    <>
      {/* File Path Display */}
      <div className="mb-4 p-3 bg-gray-800/50 border border-gray-700 rounded-xl">
        <p className="text-xs text-gray-500 mb-1">Importing from</p>
        <p className="text-sm text-gray-300 font-mono break-all">
          {inspectResult.file_path}
        </p>
      </div>

      {/* Status Badges */}
      <div className="flex flex-wrap gap-2 mb-5">
        <StatusBadge
          success={inspectResult.is_valid}
          label="Valid workflow"
          warningLabel="Invalid workflow"
        />
        {preview && (
          <>
            <StatusBadge
              success={preview.has_start_node}
              label="Has start node"
              warningLabel="No start node"
            />
            <StatusBadge
              success={(preview.node_count || 0) > 0}
              label={`${preview.node_count} nodes`}
              warningLabel="No nodes"
            />
          </>
        )}
      </div>

      {/* Already indexed warning */}
      {inspectResult.already_indexed && (
        <AlertBox
          type="warning"
          title="Workflow already indexed"
          message="This workflow is already registered. Enable overwrite to replace it."
          className="mb-5"
        />
      )}

      {/* Workflow Name Input */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-300 mb-2">
          Workflow Name
        </label>
        <input
          type="text"
          value={workflowName}
          onChange={(e) => onNameChange(e.target.value)}
          className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-purple-500 transition-colors"
          placeholder="Workflow name"
        />
        <p className="mt-1 text-xs text-gray-500">
          Leave empty to use name from file
        </p>
      </div>

      {/* Workflow Preview */}
      {preview && (
        <div className="mb-5 p-4 bg-gray-800/30 border border-gray-700/50 rounded-xl">
          <h4 className="text-sm font-medium text-gray-300 mb-3">Workflow Details</h4>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <span className="text-gray-500">Nodes:</span>{' '}
              <span className="text-white">{preview.node_count}</span>
            </div>
            <div>
              <span className="text-gray-500">Edges:</span>{' '}
              <span className="text-white">{preview.edge_count}</span>
            </div>
            <div>
              <span className="text-gray-500">Version:</span>{' '}
              <span className="text-white">{preview.version}</span>
            </div>
            {preview.tags && preview.tags.length > 0 && (
              <div className="col-span-2">
                <span className="text-gray-500">Tags:</span>{' '}
                <span className="text-white">{preview.tags.join(', ')}</span>
              </div>
            )}
          </div>
          {preview.description && (
            <p className="mt-3 text-sm text-gray-400">{preview.description}</p>
          )}
        </div>
      )}

      {/* Overwrite checkbox */}
      {inspectResult.already_indexed && (
        <label className="flex items-center gap-2 mb-4 text-sm text-gray-400 cursor-pointer">
          <input
            type="checkbox"
            checked={overwrite}
            onChange={(e) => onOverwriteChange(e.target.checked)}
            className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-purple-500 focus:ring-purple-500"
          />
          Overwrite existing workflow
        </label>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between pt-2">
        <button
          type="button"
          onClick={onBack}
          className="px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
          disabled={isImporting}
        >
          Back
        </button>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onClose}
            className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
            disabled={isImporting}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onImport}
            disabled={isImporting || (inspectResult.already_indexed && !overwrite)}
            className="px-5 py-2.5 bg-gradient-to-r from-purple-500 to-violet-600 text-white font-medium rounded-xl hover:from-purple-400 hover:to-violet-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-purple-500/20 hover:shadow-purple-500/30"
          >
            {isImporting ? (
              <span className="flex items-center gap-2">
                <Loader2 size={16} className="animate-spin" />
                Importing...
              </span>
            ) : (
              <span className="flex items-center gap-2">
                <Upload size={16} />
                Import Workflow
              </span>
            )}
          </button>
        </div>
      </div>
    </>
  );
}

export default RoutineImportModal;
