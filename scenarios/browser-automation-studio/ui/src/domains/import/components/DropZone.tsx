/**
 * DropZone Component
 *
 * Shared drag-and-drop zone for file and folder selection.
 * Used across project, asset, and routine import modals.
 */

import { Upload, FolderOpen, Image as ImageIcon, File, Loader2, AlertCircle, X } from 'lucide-react';
import { useDropZone, type UseDropZoneOptions } from '../hooks/useDropZone';
import type { DropZoneVariant, SelectedFile } from '../types';
import { formatFileSize, getFileTypeInfo, getAcceptString } from '../constants';

export interface DropZoneProps extends Omit<UseDropZoneOptions, 'onFilesSelected' | 'onFolderSelected'> {
  /** What can be dropped: file, folder, or both */
  variant?: DropZoneVariant;
  /** Callback when files are selected */
  onFilesSelected?: (files: SelectedFile[]) => void;
  /** Callback when folder is selected (path for server-side scanning) */
  onFolderSelected?: (path: string) => void;
  /** Custom label for the drop zone */
  label?: string;
  /** Custom description text */
  description?: string;
  /** Whether to show file preview after selection */
  showPreview?: boolean;
  /** Custom class name */
  className?: string;
  /** Test ID for testing */
  testId?: string;
}

export function DropZone({
  variant = 'file',
  accept,
  multiple = false,
  maxSize,
  validate,
  onFilesSelected,
  onFolderSelected,
  disabled = false,
  label,
  description,
  showPreview = true,
  className = '',
  testId,
}: DropZoneProps) {
  const {
    isDragging,
    isProcessing,
    selectedFiles,
    error,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleFileInputChange,
    openFilePicker,
    openFolderPicker,
    clearFiles,
    clearError,
    fileInputRef,
  } = useDropZone({
    accept,
    multiple,
    maxSize,
    validate,
    onFilesSelected,
    onFolderSelected,
    disabled,
  });

  const hasFiles = selectedFiles.length > 0;
  const showFiles = showPreview && hasFiles;
  const supportsFolder = variant === 'folder' || variant === 'both';
  const supportsFile = variant === 'file' || variant === 'both';

  // Default labels based on variant
  const defaultLabel = variant === 'folder'
    ? 'Select a folder'
    : variant === 'both'
    ? 'Drag files or select a folder'
    : 'Drag and drop files';

  const defaultDescription = variant === 'folder'
    ? 'or browse to select'
    : 'or click to browse';

  const displayLabel = label || defaultLabel;
  const displayDescription = description || defaultDescription;

  return (
    <div className={`space-y-3 ${className}`} data-testid={testId}>
      {/* Drop Zone */}
      {!showFiles && (
        <div
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          onClick={supportsFile ? openFilePicker : undefined}
          role="button"
          tabIndex={disabled ? -1 : 0}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              if (supportsFile) openFilePicker();
            }
          }}
          className={`
            relative w-full border-2 border-dashed rounded-xl transition-all
            flex flex-col items-center justify-center gap-3
            ${supportsFile ? 'cursor-pointer' : 'cursor-default'}
            ${isDragging
              ? 'border-flow-accent bg-flow-accent/10 scale-[1.02]'
              : 'border-gray-700 hover:border-gray-600 hover:bg-gray-800/30'
            }
            ${isProcessing || disabled ? 'opacity-60 cursor-wait pointer-events-none' : ''}
            ${variant === 'folder' ? 'p-6' : 'p-8'}
          `}
          aria-label={displayLabel}
          aria-disabled={disabled}
        >
          {isProcessing ? (
            <>
              <Loader2 size={32} className="text-flow-accent animate-spin" />
              <span className="text-sm text-gray-400">Processing...</span>
            </>
          ) : (
            <>
              <div className={`p-3 rounded-full ${isDragging ? 'bg-flow-accent/20' : 'bg-gray-800'}`}>
                {variant === 'folder' ? (
                  <FolderOpen size={24} className={isDragging ? 'text-flow-accent' : 'text-gray-400'} />
                ) : isDragging ? (
                  <ImageIcon size={24} className="text-flow-accent" />
                ) : (
                  <Upload size={24} className="text-gray-400" />
                )}
              </div>
              <div className="text-center">
                <span className="text-sm font-medium text-white block">
                  {isDragging ? 'Drop here' : displayLabel}
                </span>
                <span className="text-xs text-gray-500 mt-1 block">
                  {displayDescription}
                </span>
                {accept && accept.length > 0 && (
                  <span className="text-xs text-gray-600 mt-2 block">
                    Supported: {accept.join(', ')}
                  </span>
                )}
              </div>

              {/* Folder picker button when variant includes folder */}
              {supportsFolder && onFolderSelected && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    openFolderPicker();
                  }}
                  className="mt-2 px-4 py-2 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors flex items-center gap-2"
                >
                  <FolderOpen size={16} />
                  Browse Folders
                </button>
              )}
            </>
          )}

          {/* Hidden file input */}
          {supportsFile && (
            <input
              ref={fileInputRef}
              type="file"
              accept={accept ? getAcceptString(accept) : undefined}
              multiple={multiple}
              onChange={handleFileInputChange}
              className="hidden"
              disabled={disabled}
            />
          )}
        </div>
      )}

      {/* Selected Files Preview */}
      {showFiles && (
        <div className="space-y-2">
          {selectedFiles.map((sf, index) => (
            <FilePreviewCard
              key={`${sf.file.name}-${index}`}
              selectedFile={sf}
              onRemove={() => {
                // For single file, just clear all
                if (!multiple) {
                  clearFiles();
                } else {
                  // For multiple, we'd need to update the array
                  // For now, just clear all (can be enhanced later)
                  clearFiles();
                }
              }}
            />
          ))}
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className="flex items-start gap-3 p-3 bg-red-500/10 border border-red-500/30 rounded-lg">
          <AlertCircle size={18} className="text-red-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm text-red-300">{error}</p>
          </div>
          <button
            type="button"
            onClick={clearError}
            className="p-1 text-red-400 hover:text-red-300 transition-colors"
            aria-label="Dismiss error"
          >
            <X size={16} />
          </button>
        </div>
      )}
    </div>
  );
}

/** Individual file preview card */
interface FilePreviewCardProps {
  selectedFile: SelectedFile;
  onRemove: () => void;
}

function FilePreviewCard({ selectedFile, onRemove }: FilePreviewCardProps) {
  const { file, previewUrl, validation } = selectedFile;
  const fileInfo = getFileTypeInfo(file);
  const FileIcon = file.type.startsWith('image/') ? ImageIcon : File;
  const isInvalid = validation && !validation.isValid;

  return (
    <div
      className={`p-4 bg-gray-800/50 border rounded-xl ${
        isInvalid ? 'border-red-500/50' : 'border-gray-700/50'
      }`}
    >
      <div className="flex items-start gap-4">
        {/* Preview or Icon */}
        {previewUrl ? (
          <img
            src={previewUrl}
            alt="Preview"
            className="w-20 h-20 object-cover rounded-lg border border-gray-700"
          />
        ) : (
          <div
            className={`w-20 h-20 rounded-lg bg-gray-700/50 flex items-center justify-center ${fileInfo.color}`}
          >
            <FileIcon size={32} />
          </div>
        )}

        {/* File Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-white font-medium truncate">{file.name}</span>
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${fileInfo.color} bg-gray-700/50`}
            >
              {fileInfo.label}
            </span>
          </div>
          <p className="text-gray-500 text-sm">{formatFileSize(file.size)}</p>
          {validation?.dimensions && (
            <p className="text-gray-500 text-xs mt-1">
              {validation.dimensions.width} x {validation.dimensions.height}
            </p>
          )}
          {isInvalid && (
            <p className="text-red-400 text-sm mt-2 flex items-center gap-1">
              <AlertCircle size={12} />
              {validation.error}
            </p>
          )}
        </div>

        {/* Remove Button */}
        <button
          type="button"
          onClick={onRemove}
          className="p-2 text-gray-500 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
          aria-label="Remove file"
        >
          <X size={16} />
        </button>
      </div>
    </div>
  );
}

export default DropZone;
