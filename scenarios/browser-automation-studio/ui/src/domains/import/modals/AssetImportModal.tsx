/**
 * AssetImportModal
 *
 * Modal for uploading assets (images, data files, text) into a project.
 * Features both drag-and-drop and folder browser for flexible file selection.
 * Responsive layout: side-by-side on desktop, stacked on mobile.
 */

import { useState, useCallback, useId, useEffect, useRef } from 'react';
import { X, Upload, Check, Loader2 } from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import toast from 'react-hot-toast';

import { ImportSourceSelector } from '../components/ImportSourceSelector';
import { AlertBox } from '../components/ValidationStatus';
import { ASSET_EXTENSIONS } from '../constants';
import type { AssetImportModalProps, SelectedFile, FolderEntry } from '../types';
import { getApiBase } from '../../../config';

type Step = 'select' | 'preview';

export function AssetImportModal({
  isOpen,
  onClose,
  folder = '',
  projectId,
  onSuccess,
}: AssetImportModalProps) {
  const titleId = useId();

  const [step, setStep] = useState<Step>('select');
  const [selectedFile, setSelectedFile] = useState<SelectedFile | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [assetName, setAssetName] = useState('');
  const [isUploading, setIsUploading] = useState(false);
  const [isLoadingFile, setIsLoadingFile] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [currentBrowsePath, setCurrentBrowsePath] = useState('');

  // Track last file input ref for cleanup
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Reset state when modal closes
  useEffect(() => {
    if (!isOpen) {
      // Clean up preview URL
      if (selectedFile?.previewUrl) {
        URL.revokeObjectURL(selectedFile.previewUrl);
      }
      setStep('select');
      setSelectedFile(null);
      setSelectedPath(null);
      setAssetName('');
      setValidationError(null);
      setIsUploading(false);
      setIsLoadingFile(false);
      setCurrentBrowsePath('');
    }
  }, [isOpen, selectedFile?.previewUrl]);

  // Handle files selected from drop zone
  const handleFilesSelected = useCallback((files: SelectedFile[]) => {
    if (files.length === 0) return;

    const file = files[0];
    if (!file.validation?.isValid) {
      setValidationError(file.validation?.error || 'Invalid file');
      return;
    }

    setSelectedFile(file);
    setSelectedPath(null);
    setValidationError(null);

    // Auto-generate asset name from filename (without extension)
    const nameWithoutExt = file.file.name.replace(/\.[^/.]+$/, '');
    setAssetName(nameWithoutExt);
    setStep('preview');
  }, []);

  // Handle file selected from folder browser
  const handleFolderSelect = useCallback(async (entry: FolderEntry) => {
    // Entry is a file from the file system
    setSelectedPath(entry.path);
    setValidationError(null);
    setIsLoadingFile(true);

    try {
      // Fetch file info from server
      const response = await fetch(`${getApiBase()}/fs/file-info`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: entry.path }),
      });

      if (!response.ok) {
        throw new Error('Failed to get file info');
      }

      const fileInfo = await response.json();

      // Auto-generate asset name from filename
      const nameWithoutExt = entry.name.replace(/\.[^/.]+$/, '');
      setAssetName(nameWithoutExt);

      // Create a mock SelectedFile for display purposes
      setSelectedFile({
        file: new File([], entry.name, { type: fileInfo.mime_type || '' }),
        validation: { isValid: true },
      });

      setStep('preview');
    } catch {
      setValidationError('Failed to load file information');
    } finally {
      setIsLoadingFile(false);
    }
  }, []);

  const handleUpload = useCallback(async () => {
    if ((!selectedFile && !selectedPath) || !assetName.trim()) {
      setValidationError('Please select a file and enter an asset name');
      return;
    }

    setIsUploading(true);
    setValidationError(null);

    try {
      // Build the asset path
      const fileName = selectedFile?.file.name || selectedPath?.split('/').pop() || '';
      const ext = fileName.split('.').pop() || '';
      const assetPath = folder
        ? `${folder}/${assetName.trim()}.${ext}`
        : `${assetName.trim()}.${ext}`;

      if (selectedPath) {
        // Copy file from filesystem path
        const response = await fetch(`${getApiBase()}/projects/${projectId}/assets/copy`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            source_path: selectedPath,
            dest_path: assetPath,
          }),
        });

        if (!response.ok) {
          const error = await response.json();
          throw new Error(error.message || 'Copy failed');
        }
      } else if (selectedFile) {
        // Upload the file via FormData
        const formData = new FormData();
        formData.append('file', selectedFile.file);
        formData.append('path', assetPath);
        formData.append('project_id', projectId);

        const response = await fetch(`${getApiBase()}/projects/assets`, {
          method: 'POST',
          body: formData,
        });

        if (!response.ok) {
          const error = await response.json();
          throw new Error(error.message || 'Upload failed');
        }
      }

      toast.success('Asset added successfully');
      onSuccess?.(assetPath);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Upload failed';
      setValidationError(message);
      toast.error(message);
    } finally {
      setIsUploading(false);
    }
  }, [selectedFile, selectedPath, assetName, folder, projectId, onSuccess, onClose]);

  const handleClearFile = useCallback(() => {
    if (selectedFile?.previewUrl) {
      URL.revokeObjectURL(selectedFile.previewUrl);
    }
    setSelectedFile(null);
    setSelectedPath(null);
    setAssetName('');
    setValidationError(null);
    setStep('select');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [selectedFile?.previewUrl]);

  const handleClose = useCallback(() => {
    if (selectedFile?.previewUrl) {
      URL.revokeObjectURL(selectedFile.previewUrl);
    }
    onClose();
  }, [selectedFile?.previewUrl, onClose]);

  const handleBack = useCallback(() => {
    handleClearFile();
  }, [handleClearFile]);

  const showPreview = step === 'preview' && (selectedFile || selectedPath);

  if (!isOpen) return null;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={handleClose}
      ariaLabelledBy={titleId}
      size="wide"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden !p-0"
    >
      <div className="flex flex-col max-h-[inherit]">
        {/* Header - fixed at top, condensed on mobile */}
        <div className="relative px-4 pt-4 pb-2 md:px-6 md:pt-6 md:pb-4 flex-shrink-0">
          <button
            type="button"
            onClick={handleClose}
            className="absolute top-3 right-3 md:top-4 md:right-4 p-1.5 md:p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close dialog"
            disabled={isUploading}
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
                Upload Asset
              </h2>
              <p className="text-xs md:text-sm text-gray-400 md:mt-1">
                {showPreview
                  ? 'Review and upload the asset'
                  : 'Drag a file or browse to select'}
              </p>
            </div>
          </div>
        </div>

        {/* Content - scrollable */}
        <div className="px-6 pb-6 flex-1 overflow-y-auto min-h-0">
          {/* Error Message */}
          {validationError && (
            <AlertBox
              type="error"
              title="Error"
              message={validationError}
              className="mb-4"
            />
          )}

          {!showPreview ? (
            <SelectStep
              projectId={projectId}
              currentBrowsePath={currentBrowsePath}
              isLoadingFile={isLoadingFile}
              onFilesSelected={handleFilesSelected}
              onFolderSelect={handleFolderSelect}
              onNavigate={setCurrentBrowsePath}
              onClose={handleClose}
            />
          ) : (
            <PreviewStep
              selectedFile={selectedFile}
              selectedPath={selectedPath}
              assetName={assetName}
              folder={folder}
              isUploading={isUploading}
              onNameChange={setAssetName}
              onBack={handleBack}
              onUpload={handleUpload}
              onClose={handleClose}
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
  currentBrowsePath: string;
  isLoadingFile: boolean;
  onFilesSelected: (files: SelectedFile[]) => void;
  onFolderSelect: (entry: FolderEntry) => void;
  onNavigate: (path: string) => void;
  onClose: () => void;
}

function SelectStep({
  projectId,
  currentBrowsePath,
  isLoadingFile,
  onFilesSelected,
  onFolderSelect,
  onNavigate,
  onClose,
}: SelectStepProps) {
  return (
    <>
      <ImportSourceSelector
        // DropZone props
        dropZoneVariant="file"
        dropZoneAccept={ASSET_EXTENSIONS}
        dropZoneLabel="Drag and drop a file"
        dropZoneDescription="or click to browse your device"
        onFilesSelected={onFilesSelected}

        // FolderBrowser props
        folderBrowserMode="files"
        projectId={projectId}
        onFolderSelect={onFolderSelect}
        onNavigate={onNavigate}
        showRegistered={false}

        // Path input - show current browse location
        showPathInput={true}
        pathValue={currentBrowsePath}
        pathLabel="Current Location"
        pathPlaceholder="Browse to select a file..."
        onPathChange={onNavigate}

        // State
        disabled={isLoadingFile}
        isLoading={isLoadingFile}
      />

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 pt-4">
        <button
          type="button"
          onClick={onClose}
          className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
          disabled={isLoadingFile}
        >
          Cancel
        </button>
      </div>
    </>
  );
}

/** Preview step with file info and name input */
interface PreviewStepProps {
  selectedFile: SelectedFile | null;
  selectedPath: string | null;
  assetName: string;
  folder: string;
  isUploading: boolean;
  onNameChange: (name: string) => void;
  onBack: () => void;
  onUpload: () => void;
  onClose: () => void;
}

function PreviewStep({
  selectedFile,
  selectedPath,
  assetName,
  folder,
  isUploading,
  onNameChange,
  onBack,
  onUpload,
  onClose,
}: PreviewStepProps) {
  const fileName = selectedFile?.file.name || selectedPath?.split('/').pop() || 'Unknown file';
  const mimeType = selectedFile?.file.type || '';
  const fileSize = selectedFile?.file.size;

  return (
    <>
      {/* Source Path Display */}
      {selectedPath && (
        <div className="mb-4 p-3 bg-gray-800/50 border border-gray-700 rounded-xl">
          <p className="text-xs text-gray-500 mb-1">Importing from</p>
          <p className="text-sm text-gray-300 font-mono break-all">{selectedPath}</p>
        </div>
      )}

      {/* Selected File Preview */}
      <div className="mb-5 p-4 bg-gray-800/50 border border-gray-700/50 rounded-xl">
        <div className="flex items-start gap-4">
          {/* Preview or Icon */}
          {selectedFile?.previewUrl ? (
            <img
              src={selectedFile.previewUrl}
              alt="Preview"
              className="w-20 h-20 object-cover rounded-lg border border-gray-700"
            />
          ) : (
            <div className="w-20 h-20 rounded-lg bg-gray-700/50 flex items-center justify-center">
              <FileTypeIcon mimeType={mimeType} />
            </div>
          )}

          {/* File Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-white font-medium truncate">{fileName}</span>
              <span className="text-xs px-2 py-0.5 rounded-full text-gray-400 bg-gray-700/50">
                {getFileTypeLabel(mimeType)}
              </span>
            </div>
            {fileSize !== undefined && (
              <p className="text-gray-500 text-sm">{formatFileSize(fileSize)}</p>
            )}
          </div>

          {/* Back/Remove Button */}
          <button
            type="button"
            onClick={onBack}
            className="p-2 text-gray-500 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
            aria-label="Remove file"
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Asset Name Input */}
      <div className="mb-5">
        <label className="block text-sm font-medium text-gray-300 mb-2">
          Asset Name
        </label>
        <input
          type="text"
          value={assetName}
          onChange={(e) => onNameChange(e.target.value)}
          className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-green-500 transition-colors"
          placeholder="Enter asset name"
        />
        <p className="mt-2 text-xs text-gray-500">
          Saving to: <span className="text-gray-400 font-mono">{folder || 'assets'}/</span>
        </p>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between pt-2">
        <button
          type="button"
          onClick={onBack}
          className="px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
          disabled={isUploading}
        >
          Back
        </button>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onClose}
            className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
            disabled={isUploading}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onUpload}
            disabled={!assetName.trim() || isUploading}
            className="px-5 py-2.5 bg-gradient-to-r from-green-500 to-emerald-600 text-white font-medium rounded-xl hover:from-green-400 hover:to-emerald-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-green-500/20 hover:shadow-green-500/30"
          >
            {isUploading ? (
              <span className="flex items-center gap-2">
                <Loader2 size={16} className="animate-spin" />
                Uploading...
              </span>
            ) : (
              <span className="flex items-center gap-2">
                <Check size={16} />
                Upload Asset
              </span>
            )}
          </button>
        </div>
      </div>
    </>
  );
}

/** File type icon based on MIME type */
function FileTypeIcon({ mimeType }: { mimeType: string }) {
  if (mimeType.startsWith('image/')) {
    return <span className="text-blue-400 text-2xl">🖼️</span>;
  }
  if (mimeType === 'application/json') {
    return <span className="text-yellow-400 text-2xl">📋</span>;
  }
  if (mimeType === 'text/csv') {
    return <span className="text-green-400 text-2xl">📊</span>;
  }
  if (mimeType.startsWith('text/')) {
    return <span className="text-gray-400 text-2xl">📄</span>;
  }
  return <span className="text-gray-400 text-2xl">📁</span>;
}

/** Get human-readable file type label */
function getFileTypeLabel(mimeType: string): string {
  const typeMap: Record<string, string> = {
    'image/png': 'PNG Image',
    'image/jpeg': 'JPEG Image',
    'image/gif': 'GIF Image',
    'image/svg+xml': 'SVG Image',
    'image/webp': 'WebP Image',
    'application/json': 'JSON',
    'text/csv': 'CSV',
    'text/plain': 'Text',
    'text/markdown': 'Markdown',
  };
  return typeMap[mimeType] || 'File';
}

/** Format file size for display */
function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default AssetImportModal;
