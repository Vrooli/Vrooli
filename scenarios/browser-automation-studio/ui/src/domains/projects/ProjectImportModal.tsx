/**
 * ProjectImportModal
 *
 * Modal for importing existing project folders into the browser-automation-studio.
 * Two-step flow: validate folder path, then preview and import.
 */

import { useState, useEffect, useId, useRef, useCallback } from "react";
import { logger } from "@utils/logger";
import {
  X,
  Upload,
  AlertCircle,
  FolderSearch,
  Check,
  AlertTriangle,
  Loader2,
} from "lucide-react";
import { ResponsiveDialog } from "@shared/layout";
import { selectors } from "@constants/selectors";
import { useProjectImport } from "./hooks/useProjectImport";
import type { Project } from "./store";
import toast from "react-hot-toast";

interface ProjectImportModalProps {
  isOpen?: boolean;
  onClose: () => void;
  onSuccess?: (project: Project) => void;
}

function StatusBadge({
  success,
  label,
  warningLabel,
}: {
  success: boolean;
  label: string;
  warningLabel?: string;
}) {
  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm ${
        success
          ? "bg-green-500/10 border border-green-500/30 text-green-300"
          : "bg-yellow-500/10 border border-yellow-500/30 text-yellow-300"
      }`}
    >
      {success ? <Check size={14} /> : <AlertTriangle size={14} />}
      <span>{success ? label : warningLabel || label}</span>
    </div>
  );
}

function ProjectImportModal({
  isOpen = true,
  onClose,
  onSuccess,
}: ProjectImportModalProps) {
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

  const [folderPath, setFolderPath] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);

  const titleId = useId();
  const folderInputRef = useRef<HTMLInputElement>(null);

  // Track if we're in preview mode (after successful inspection)
  const showPreview =
    inspectResult &&
    inspectResult.exists &&
    inspectResult.is_dir &&
    !error;

  // Focus folder input when modal opens
  useEffect(() => {
    if (isOpen && folderInputRef.current && !showPreview) {
      const timer = setTimeout(() => {
        folderInputRef.current?.focus();
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [isOpen, showPreview]);

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
      setFolderPath("");
      setName("");
      setDescription("");
      setValidationError(null);
      reset();
    }
  }, [isOpen, reset]);

  const validateFolderPath = useCallback((path: string): string | null => {
    const trimmed = path.trim();
    if (!trimmed) {
      return "Folder path is required";
    }
    if (!trimmed.startsWith("/")) {
      return "Folder path must be an absolute path starting with /";
    }
    return null;
  }, []);

  const handleValidate = async () => {
    const error = validateFolderPath(folderPath);
    if (error) {
      setValidationError(error);
      return;
    }
    setValidationError(null);
    clearError();

    const result = await inspectFolder(folderPath.trim());
    if (result) {
      if (!result.exists) {
        setValidationError("Folder does not exist");
      } else if (!result.is_dir) {
        setValidationError("Path is not a directory");
      }
    }
  };

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
    reset();
    setName("");
    setDescription("");
    setValidationError(null);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !showPreview && !isInspecting) {
      e.preventDefault();
      handleValidate();
    }
  };

  const renderInputStep = () => (
    <>
      {/* Folder Path Input */}
      <div className="mb-5">
        <label className="block text-sm font-medium text-gray-300 mb-2">
          Project Folder Path
        </label>
        <input
          ref={folderInputRef}
          type="text"
          data-testid={selectors.dialogs.projectImport.folderPathInput}
          value={folderPath}
          onChange={(e) => {
            setFolderPath(e.target.value);
            setValidationError(null);
            clearError();
          }}
          onKeyDown={handleKeyDown}
          className={`w-full px-4 py-3 bg-gray-800/50 border-2 rounded-xl text-white placeholder-gray-500 focus:outline-none font-mono text-sm transition-colors ${
            validationError || error
              ? "border-red-500/50 focus:border-red-500"
              : "border-gray-700/50 focus:border-flow-accent"
          }`}
          placeholder="/path/to/existing/project"
          disabled={isInspecting}
        />
        {(validationError || error) && (
          <p
            className="mt-2 text-red-400 text-xs flex items-center gap-1"
            data-testid={selectors.dialogs.projectImport.folderPathError}
          >
            <AlertCircle size={12} />
            {validationError || error}
          </p>
        )}
        <p className="mt-2 text-xs text-gray-500">
          Enter the absolute path to an existing project folder
        </p>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 pt-2">
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
          onClick={handleValidate}
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

  const renderPreviewStep = () => {
    if (!inspectResult) return null;

    return (
      <>
        {/* Folder Path Display */}
        <div className="mb-4 p-3 bg-gray-800/50 border border-gray-700 rounded-xl">
          <p className="text-xs text-gray-500 mb-1">Importing from</p>
          <p className="text-sm text-gray-300 font-mono break-all">
            {inspectResult.folder_path}
          </p>
        </div>

        {/* Status Badges */}
        <div
          className="flex flex-wrap gap-2 mb-5"
          data-testid={selectors.dialogs.projectImport.preview.section}
        >
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

        {/* Already indexed warning */}
        {inspectResult.already_indexed && (
          <div
            className="mb-5 p-3 bg-yellow-500/10 border border-yellow-500/30 rounded-xl flex items-start gap-3"
            data-testid={selectors.dialogs.projectImport.preview.alreadyIndexedWarning}
          >
            <AlertTriangle size={18} className="text-yellow-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-yellow-300 text-sm font-medium">
                Project already indexed
              </p>
              <p className="text-yellow-300/70 text-xs mt-1">
                This folder is already registered. Importing will return the existing project.
              </p>
            </div>
          </div>
        )}

        {/* Metadata error warning */}
        {inspectResult.metadata_error && (
          <div className="mb-5 p-3 bg-yellow-500/10 border border-yellow-500/30 rounded-xl flex items-start gap-3">
            <AlertTriangle size={18} className="text-yellow-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-yellow-300 text-sm font-medium">
                Metadata parsing issue
              </p>
              <p className="text-yellow-300/70 text-xs mt-1">
                {inspectResult.metadata_error}
              </p>
            </div>
          </div>
        )}

        {/* Name Input */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Project Name
          </label>
          <input
            type="text"
            data-testid={selectors.dialogs.projectImport.nameInput}
            value={name}
            onChange={(e) => setName(e.target.value)}
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
            Description{" "}
            <span className="text-gray-600 font-normal">(optional)</span>
          </label>
          <textarea
            data-testid={selectors.dialogs.projectImport.descriptionInput}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent resize-none transition-colors"
            placeholder="Describe what this project is for..."
          />
        </div>

        {/* API Error */}
        {error && (
          <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-xl flex items-center gap-3">
            <div className="w-8 h-8 bg-red-500/20 rounded-lg flex items-center justify-center flex-shrink-0">
              <AlertCircle size={16} className="text-red-400" />
            </div>
            <span className="text-red-300 text-sm">{error}</span>
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center justify-between pt-2">
          <button
            type="button"
            onClick={handleBack}
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
              onClick={handleImport}
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
  };

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="default"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden"
    >
      <div data-testid={selectors.dialogs.projectImport.root}>
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4">
          {/* Close button */}
          <button
            data-testid={selectors.dialogs.base.closeButton}
            onClick={() => {
              logger.debug("ProjectImportModal X button clicked", {
                component: "ProjectImportModal",
              });
              onClose();
            }}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close import modal"
          >
            <X size={18} />
          </button>

          {/* Icon & Title */}
          <div className="flex flex-col items-center text-center mb-2">
            <div className="w-14 h-14 bg-gradient-to-br from-green-500/30 to-emerald-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-green-500/10">
              <Upload size={28} className="text-green-400" />
            </div>
            <h2
              id={titleId}
              className="text-xl font-bold text-white tracking-tight"
            >
              Import Project
            </h2>
            <p className="text-sm text-gray-400 mt-1">
              {showPreview
                ? "Review and import the project"
                : "Import an existing project folder"}
            </p>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 pb-6">
          {showPreview ? renderPreviewStep() : renderInputStep()}
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default ProjectImportModal;
