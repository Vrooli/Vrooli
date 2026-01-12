/**
 * ProjectImportModal
 *
 * Modal for importing existing project folders into the browser-automation-studio.
 * Features both drag-and-drop and folder browser for easy navigation and project discovery.
 * Responsive layout: side-by-side on desktop, stacked on mobile.
 * Two-step flow: browse/validate folder path, then preview and import.
 */

import { useState, useEffect, useId, useRef, useCallback } from 'react';
import {
  X,
  Upload,
  FolderSearch,
  Loader2,
} from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import { selectors } from '@constants/selectors';
import toast from 'react-hot-toast';

import { ImportSourceSelector } from '../components/ImportSourceSelector';
import { StatusBadge, AlertBox } from '../components/ValidationStatus';
import { ValidationAccordion } from '../components/ValidationAccordion';
import { useProjectImport, type InspectFolderResponse } from '../hooks/useProjectImport';
import { getApiBase } from '../../../config';
import type { ProjectImportModalProps, FolderEntry } from '../types';
import { DEFAULT_PROJECTS_ROOT } from '../constants';

type Step = 'select' | 'preview';

export function ProjectImportModal({
  isOpen,
  onClose,
  onSuccess,
}: ProjectImportModalProps) {
  const titleId = useId();

  const [step, setStep] = useState<Step>('select');
  const [folderPath, setFolderPath] = useState(DEFAULT_PROJECTS_ROOT);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);
  const [pathValidationStatus, setPathValidationStatus] = useState<'valid' | 'invalid' | 'checking' | null>(null);

  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const {
    isInspecting,
    isImporting,
    inspectResult,
    error,
    inspectFolder,
    importProject,
    clearError,
    reset,
  } = useProjectImport();

  const showPreview = step === 'preview' && inspectResult;

  // Debounced path validation as user types
  useEffect(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    const trimmed = folderPath.trim();
    if (!trimmed || !trimmed.startsWith('/')) {
      setPathValidationStatus(null);
      return;
    }

    setPathValidationStatus('checking');
    debounceTimerRef.current = setTimeout(async () => {
      try {
        const response = await fetch(`${getApiBase()}/fs/scan`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode: 'projects', path: trimmed }),
        });
        setPathValidationStatus(response.ok ? 'valid' : 'invalid');
      } catch {
        setPathValidationStatus('invalid');
      }
    }, 300);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [folderPath]);

  // Update name/description from inspection result
  useEffect(() => {
    if (inspectResult) {
      if (inspectResult.suggested_name && !name) {
        setName(inspectResult.suggested_name);
      }
      if (inspectResult.suggested_description && !description) {
        setDescription(inspectResult.suggested_description);
      }
    }
  }, [inspectResult, name, description]);

  // Reset state when modal closes
  useEffect(() => {
    if (!isOpen) {
      setStep('select');
      setFolderPath(DEFAULT_PROJECTS_ROOT);
      setName('');
      setDescription('');
      setValidationError(null);
      setPathValidationStatus(null);
      reset();
    }
  }, [isOpen, reset]);

  const validateFolderPath = useCallback((path: string): string | null => {
    const trimmed = path.trim();
    if (!trimmed) {
      return 'Folder path is required';
    }
    if (!trimmed.startsWith('/')) {
      return 'Folder path must be an absolute path starting with /';
    }
    return null;
  }, []);

  const handleValidate = useCallback(async (pathToValidate?: string) => {
    const targetPath = pathToValidate ?? folderPath;
    const validationErr = validateFolderPath(targetPath);
    if (validationErr) {
      setValidationError(validationErr);
      return;
    }
    setValidationError(null);
    clearError();

    const result = await inspectFolder(targetPath.trim());
    if (result) {
      if (!result.exists) {
        setValidationError('Folder does not exist');
      } else if (!result.is_dir) {
        setValidationError('Path is not a directory');
      } else {
        setStep('preview');
      }
    }
  }, [folderPath, validateFolderPath, clearError, inspectFolder]);

  const handleImport = async () => {
    if (!inspectResult) return;

    const project = await importProject({
      folder_path: inspectResult.folder_path,
      name: name.trim() || undefined,
      description: description.trim() || undefined,
    });

    if (project) {
      toast.success(`Project "${project.name}" imported successfully`);
      onSuccess?.(project);
      onClose();
    }
  };

  const handleBack = () => {
    setStep('select');
    setName('');
    setDescription('');
    setValidationError(null);
    reset();
  };

  // Handle folder selection from browser
  const handleFolderSelect = useCallback(async (entry: FolderEntry) => {
    setFolderPath(entry.path);
    setValidationError(null);
    clearError();
    await handleValidate(entry.path);
  }, [clearError, handleValidate]);

  // Handle folder path input change
  const handleFolderPathChange = useCallback((newPath: string) => {
    setFolderPath(newPath);
    setValidationError(null);
    clearError();
  }, [clearError]);

  // Handle folder path submit (Enter key or validate button)
  const handlePathSubmit = useCallback(() => {
    if (!isInspecting) {
      handleValidate();
    }
  }, [isInspecting, handleValidate]);

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="wide"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden !p-0"
    >
      <div data-testid={selectors.dialogs.projectImport.root} className="flex flex-col max-h-[inherit]">
        {/* Header - fixed at top, condensed on mobile */}
        <div className="relative px-4 pt-4 pb-2 md:px-6 md:pt-6 md:pb-4 flex-shrink-0">
          {/* Close button */}
          <button
            type="button"
            data-testid={selectors.dialogs.base.closeButton}
            onClick={onClose}
            className="absolute top-3 right-3 md:top-4 md:right-4 p-1.5 md:p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close import modal"
            disabled={isImporting}
          >
            <X size={18} />
          </button>

          {/* Icon & Title - inline on mobile, stacked on desktop */}
          <div className="flex items-center gap-3 md:flex-col md:items-center md:text-center md:gap-0 mb-1 md:mb-2">
            <div className="w-10 h-10 md:w-14 md:h-14 bg-gradient-to-br from-green-500/30 to-emerald-500/20 rounded-xl md:rounded-2xl flex items-center justify-center md:mb-4 shadow-lg shadow-green-500/10 flex-shrink-0">
              <Upload size={20} className="md:hidden text-green-400" />
              <Upload size={28} className="hidden md:block text-green-400" />
            </div>
            <div>
              <h2
                id={titleId}
                className="text-lg md:text-xl font-bold text-white tracking-tight"
              >
                Import Project
              </h2>
              <p className="text-xs md:text-sm text-gray-400 md:mt-1">
                {showPreview
                  ? 'Review and import the project'
                  : 'Drag a folder or browse to select'}
              </p>
            </div>
          </div>
        </div>

        {/* Content - scrollable */}
        <div className="px-6 pb-6 flex-1 overflow-y-auto min-h-0">
          {!showPreview ? (
            <SelectStep
              folderPath={folderPath}
              validationError={validationError}
              pathValidationStatus={pathValidationStatus}
              error={error}
              isInspecting={isInspecting}
              onFolderPathChange={handleFolderPathChange}
              onPathSubmit={handlePathSubmit}
              onFolderSelect={handleFolderSelect}
              onValidate={() => handleValidate()}
              onClose={onClose}
            />
          ) : (
            <PreviewStep
              inspectResult={inspectResult!}
              name={name}
              description={description}
              isImporting={isImporting}
              error={error}
              onNameChange={setName}
              onDescriptionChange={setDescription}
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
  folderPath: string;
  validationError: string | null;
  pathValidationStatus: 'valid' | 'invalid' | 'checking' | null;
  error: string | null;
  isInspecting: boolean;
  onFolderPathChange: (path: string) => void;
  onPathSubmit: () => void;
  onFolderSelect: (entry: FolderEntry) => void;
  onValidate: () => void;
  onClose: () => void;
}

function SelectStep({
  folderPath,
  validationError,
  pathValidationStatus,
  error,
  isInspecting,
  onFolderPathChange,
  onPathSubmit,
  onFolderSelect,
  onValidate,
  onClose,
}: SelectStepProps) {
  return (
    <>
      <ImportSourceSelector
        // DropZone props
        dropZoneVariant="folder"
        dropZoneLabel="Drop project folder here"
        dropZoneDescription="or use the browser on the right"
        onDropZoneFolderSelected={(path) => {
          onFolderPathChange(path);
          // Note: Native folder picker gives folder name, not full path
          // User will need to use browser for full path selection
        }}

        // FolderBrowser props
        folderBrowserMode="projects"
        onFolderSelect={onFolderSelect}
        onNavigate={onFolderPathChange}
        initialPath={DEFAULT_PROJECTS_ROOT}
        showRegistered={false}

        // Path input
        showPathInput={true}
        pathValue={folderPath}
        pathLabel="Project Folder Path"
        pathPlaceholder="/path/to/existing/project"
        onPathChange={onFolderPathChange}
        onPathSubmit={onPathSubmit}
        pathValidationStatus={pathValidationStatus}
        pathError={validationError || error}

        // State
        disabled={isInspecting}
        isLoading={isInspecting}
      />

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 pt-4">
        <button
          type="button"
          data-testid={selectors.dialogs.projectImport.cancelButton}
          onClick={onClose}
          className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
          disabled={isInspecting}
        >
          Cancel
        </button>
        <button
          type="button"
          data-testid={selectors.dialogs.projectImport.validateButton}
          onClick={onValidate}
          disabled={isInspecting || !folderPath.trim()}
          className="px-5 py-2.5 bg-gradient-to-r from-flow-accent to-blue-600 text-white font-medium rounded-xl hover:from-blue-500 hover:to-blue-700 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-flow-accent/20 hover:shadow-flow-accent/30"
        >
          {isInspecting ? (
            <span className="flex items-center gap-2">
              <Loader2 size={16} className="animate-spin" />
              Validating...
            </span>
          ) : (
            <span className="flex items-center gap-2">
              <FolderSearch size={16} />
              Validate Folder
            </span>
          )}
        </button>
      </div>
    </>
  );
}

/** Preview step */
interface PreviewStepProps {
  inspectResult: InspectFolderResponse;
  name: string;
  description: string;
  isImporting: boolean;
  error: string | null;
  onNameChange: (name: string) => void;
  onDescriptionChange: (description: string) => void;
  onBack: () => void;
  onImport: () => void;
  onClose: () => void;
}

function PreviewStep({
  inspectResult,
  name,
  description,
  isImporting,
  error,
  onNameChange,
  onDescriptionChange,
  onBack,
  onImport,
  onClose,
}: PreviewStepProps) {
  return (
    <>
      {/* Folder Path Display */}
      <div className="mb-4 p-3 bg-gray-800/50 border border-gray-700 rounded-xl">
        <p className="text-xs text-gray-500 mb-1">Importing from</p>
        <p className="text-sm text-gray-300 font-mono break-all">
          {inspectResult.folder_path}
        </p>
      </div>

      {/* Validation Accordion */}
      <div
        className="mb-5"
        data-testid={selectors.dialogs.projectImport.preview.section}
      >
        {inspectResult.validation ? (
          <ValidationAccordion validation={inspectResult.validation} />
        ) : (
          /* Fallback for legacy API responses without validation */
          <div className="flex flex-wrap gap-2">
            <StatusBadge
              success={inspectResult.has_bas_metadata}
              label="Has project metadata"
              warningLabel="No project metadata"
            />
            <StatusBadge
              success={inspectResult.has_workflows}
              label="Workflows detected"
              warningLabel="No workflows found"
            />
          </div>
        )}
      </div>

      {/* Name Input */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-300 mb-2">
          Project Name
        </label>
        <input
          type="text"
          data-testid={selectors.dialogs.projectImport.nameInput}
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent transition-colors"
          placeholder="Project name"
        />
        <p className="mt-1 text-xs text-gray-500">
          Leave empty to use name from metadata or folder name
        </p>
      </div>

      {/* Description Input */}
      <div className="mb-5">
        <label className="block text-sm font-medium text-gray-400 mb-2">
          Description <span className="text-gray-600 font-normal">(optional)</span>
        </label>
        <textarea
          data-testid={selectors.dialogs.projectImport.descriptionInput}
          value={description}
          onChange={(e) => onDescriptionChange(e.target.value)}
          rows={2}
          className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent resize-none transition-colors"
          placeholder="Describe what this project is for..."
        />
      </div>

      {/* API Error */}
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
            data-testid={selectors.dialogs.projectImport.cancelButton}
            onClick={onClose}
            className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
            disabled={isImporting}
          >
            Cancel
          </button>
          <button
            type="button"
            data-testid={selectors.dialogs.projectImport.importButton}
            onClick={onImport}
            disabled={isImporting}
            className="px-5 py-2.5 bg-gradient-to-r from-green-500 to-emerald-600 text-white font-medium rounded-xl hover:from-green-400 hover:to-emerald-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-green-500/20 hover:shadow-green-500/30"
          >
            {isImporting ? (
              <span className="flex items-center gap-2">
                <Loader2 size={16} className="animate-spin" />
                Importing...
              </span>
            ) : (
              <span className="flex items-center gap-2">
                <Upload size={16} />
                Import Project
              </span>
            )}
          </button>
        </div>
      </div>
    </>
  );
}

export default ProjectImportModal;
