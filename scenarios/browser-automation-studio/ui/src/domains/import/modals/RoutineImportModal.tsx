/**
 * RoutineImportModal
 *
 * Modal for importing workflow/routine files into a project.
 * Supports both drag-and-drop file upload and folder browsing.
 */

import { useState, useEffect, useCallback, useId, useRef } from 'react';
import { FileCode, Loader2, X, Upload } from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import toast from 'react-hot-toast';

import { DropZone } from '../components/DropZone';
import { FolderBrowser } from '../components/FolderBrowser';
import { StatusBadge, AlertBox } from '../components/ValidationStatus';
import { useRoutineImport, type InspectRoutineResponse } from '../hooks/useRoutineImport';
import type { RoutineImportModalProps, FolderEntry, SelectedFile } from '../types';
import { WORKFLOW_EXTENSIONS } from '../constants';

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

  const folderInputRef = useRef<HTMLInputElement>(null);

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
      reset();
    }
  }, [isOpen, reset]);

  // Update name from preview
  useEffect(() => {
    if (inspectResult?.preview?.name && !workflowName) {
      setWorkflowName(inspectResult.preview.name);
    }
  }, [inspectResult, workflowName]);

  // Handle file selection from drop zone
  const handleFilesSelected = useCallback(
    async (files: SelectedFile[]) => {
      if (files.length === 0) return;

      const file = files[0];
      if (!file.validation?.isValid) {
        return;
      }

      // For dropped files, we need the path - in browser context we can't get it
      // This would work with Electron/Tauri where we have file system access
      // For now, show a message that folder browsing is preferred
      toast.error('Please use folder browser to select workflow files');
    },
    []
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

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="default"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden"
    >
      <div>
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4">
          {/* Close button */}
          <button
            type="button"
            onClick={onClose}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close"
            disabled={isImporting}
          >
            <X size={18} />
          </button>

          {/* Icon & Title */}
          <div className="flex flex-col items-center text-center mb-2">
            <div className="w-14 h-14 bg-gradient-to-br from-purple-500/30 to-violet-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-purple-500/10">
              <FileCode size={28} className="text-purple-400" />
            </div>
            <h2 id={titleId} className="text-xl font-bold text-white tracking-tight">
              Import Workflow
            </h2>
            <p className="text-sm text-gray-400 mt-1">
              {showPreview
                ? 'Review and import the workflow'
                : 'Select a workflow file to import'}
            </p>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 pb-6">
          {!showPreview ? (
            <SelectStep
              projectId={projectId}
              folderPath={currentFolderPath}
              folderInputRef={folderInputRef}
              isInspecting={isInspecting}
              error={error}
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

/** Selection step */
interface SelectStepProps {
  projectId: string;
  folderPath: string;
  folderInputRef: React.RefObject<HTMLInputElement>;
  isInspecting: boolean;
  error: string | null;
  onFolderPathChange: (path: string) => void;
  onFolderNavigate: (path: string) => void;
  onFilesSelected: (files: SelectedFile[]) => void;
  onFolderSelect: (entry: FolderEntry) => void;
  onClose: () => void;
}

function SelectStep({
  projectId,
  folderPath,
  folderInputRef,
  isInspecting,
  error,
  onFolderPathChange,
  onFolderNavigate,
  onFilesSelected,
  onFolderSelect,
  onClose,
}: SelectStepProps) {
  return (
    <>
      {/* Drop Zone */}
      <DropZone
        variant="file"
        accept={WORKFLOW_EXTENSIONS as unknown as string[]}
        onFilesSelected={onFilesSelected}
        label="Drop workflow file here"
        description="or browse below"
        showPreview={false}
        disabled={isInspecting}
      />

      {/* Divider */}
      <div className="flex items-center gap-3 my-4">
        <div className="flex-1 h-px bg-gray-700/50" />
        <span className="text-xs text-gray-500">or browse project files</span>
        <div className="flex-1 h-px bg-gray-700/50" />
      </div>

      {/* Folder Path Input */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-300 mb-2">
          Current Location
        </label>
        <div className="relative">
          <input
            ref={folderInputRef}
            type="text"
            value={folderPath}
            onChange={(e) => onFolderPathChange(e.target.value)}
            className="w-full px-4 py-3 pr-10 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-purple-500 font-mono text-sm transition-colors"
            placeholder="Browsing workflows folder..."
            disabled={isInspecting}
            readOnly
          />
        </div>
      </div>

      {/* Folder Browser */}
      <FolderBrowser
        mode="workflows"
        projectId={projectId}
        onSelect={onFolderSelect}
        onNavigate={onFolderNavigate}
        showRegistered={false}
        className="max-h-48"
      />

      {/* Error */}
      {error && (
        <AlertBox
          type="error"
          title="Error"
          message={error}
          className="mt-4"
        />
      )}

      {/* Loading indicator */}
      {isInspecting && (
        <div className="mt-4 flex items-center justify-center gap-2 text-gray-400">
          <Loader2 size={16} className="animate-spin" />
          <span className="text-sm">Inspecting workflow...</span>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 pt-4">
        <button
          type="button"
          onClick={onClose}
          className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
          disabled={isInspecting}
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
  error,
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

      {/* Error */}
      {error && (
        <AlertBox
          type="error"
          title="Import Error"
          message={error}
          className="mb-4"
        />
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
