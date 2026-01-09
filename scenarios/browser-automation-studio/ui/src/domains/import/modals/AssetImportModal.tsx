/**
 * AssetImportModal
 *
 * Modal for uploading assets (images, data files, text) into a project.
 * Features drag-and-drop file upload with validation and preview.
 */

import { useState, useCallback, useId, useRef, useEffect } from 'react';
import { X, Upload, Check, Loader } from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import toast from 'react-hot-toast';

import { DropZone } from '../components/DropZone';
import { AlertBox } from '../components/ValidationStatus';
import { ASSET_EXTENSIONS, MAX_FILE_SIZE } from '../constants';
import type { AssetImportModalProps, SelectedFile } from '../types';
import { getApiBase } from '../../../config';

export function AssetImportModal({
  isOpen,
  onClose,
  folder = '',
  projectId,
  onSuccess,
}: AssetImportModalProps) {
  const titleId = useId();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [selectedFile, setSelectedFile] = useState<SelectedFile | null>(null);
  const [assetName, setAssetName] = useState('');
  const [isUploading, setIsUploading] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  // Reset state when modal closes
  useEffect(() => {
    if (!isOpen) {
      // Clean up preview URL
      if (selectedFile?.previewUrl) {
        URL.revokeObjectURL(selectedFile.previewUrl);
      }
      setSelectedFile(null);
      setAssetName('');
      setValidationError(null);
      setIsUploading(false);
    }
  }, [isOpen, selectedFile?.previewUrl]);

  const handleFilesSelected = useCallback((files: SelectedFile[]) => {
    if (files.length === 0) return;

    const file = files[0];
    if (!file.validation?.isValid) {
      setValidationError(file.validation?.error || 'Invalid file');
      return;
    }

    setSelectedFile(file);
    setValidationError(null);

    // Auto-generate asset name from filename (without extension)
    const nameWithoutExt = file.file.name.replace(/\.[^/.]+$/, '');
    setAssetName(nameWithoutExt);
  }, []);

  const handleUpload = useCallback(async () => {
    if (!selectedFile || !assetName.trim()) {
      setValidationError('Please select a file and enter an asset name');
      return;
    }

    setIsUploading(true);
    setValidationError(null);

    try {
      // Build the asset path
      const ext = selectedFile.file.name.split('.').pop() || '';
      const assetPath = folder
        ? `${folder}/${assetName.trim()}.${ext}`
        : `${assetName.trim()}.${ext}`;

      // Upload the file
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

      toast.success('Asset uploaded successfully');
      onSuccess?.(assetPath);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Upload failed';
      setValidationError(message);
      toast.error(message);
    } finally {
      setIsUploading(false);
    }
  }, [selectedFile, assetName, folder, projectId, onSuccess, onClose]);

  const handleClearFile = useCallback(() => {
    if (selectedFile?.previewUrl) {
      URL.revokeObjectURL(selectedFile.previewUrl);
    }
    setSelectedFile(null);
    setAssetName('');
    setValidationError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [selectedFile?.previewUrl]);

  const handleClose = useCallback(() => {
    if (selectedFile?.previewUrl) {
      URL.revokeObjectURL(selectedFile.previewUrl);
    }
    handleClearFile();
    onClose();
  }, [selectedFile?.previewUrl, handleClearFile, onClose]);

  if (!isOpen) return null;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={handleClose}
      ariaLabelledBy={titleId}
      size="default"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden"
    >
      <div>
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4">
          <button
            type="button"
            onClick={handleClose}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close dialog"
            disabled={isUploading}
          >
            <X size={18} />
          </button>

          <div className="flex flex-col items-center text-center mb-2">
            <div className="w-14 h-14 bg-gradient-to-br from-green-500/30 to-emerald-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-green-500/10">
              <Upload size={28} className="text-green-400" />
            </div>
            <h2
              id={titleId}
              className="text-xl font-bold text-white tracking-tight"
            >
              Upload Asset
            </h2>
            <p className="text-sm text-gray-400 mt-1">
              Add images, data files, or text to your project
            </p>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 pb-6">
          {/* Error Message */}
          {validationError && (
            <AlertBox
              type="error"
              title="Upload Error"
              message={validationError}
              className="mb-4"
            />
          )}

          {/* Drop Zone - Show when no file selected */}
          {!selectedFile ? (
            <DropZone
              variant="file"
              accept={ASSET_EXTENSIONS}
              maxSize={MAX_FILE_SIZE}
              onFilesSelected={handleFilesSelected}
              label="Drag and drop a file"
              description="or click to browse"
              showPreview={false}
              disabled={isUploading}
            />
          ) : (
            <>
              {/* Selected File Preview */}
              <div className="mb-5 p-4 bg-gray-800/50 border border-gray-700/50 rounded-xl">
                <div className="flex items-start gap-4">
                  {/* Preview or Icon */}
                  {selectedFile.previewUrl ? (
                    <img
                      src={selectedFile.previewUrl}
                      alt="Preview"
                      className="w-20 h-20 object-cover rounded-lg border border-gray-700"
                    />
                  ) : (
                    <div className="w-20 h-20 rounded-lg bg-gray-700/50 flex items-center justify-center">
                      <FileTypeIcon mimeType={selectedFile.file.type} />
                    </div>
                  )}

                  {/* File Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-white font-medium truncate">
                        {selectedFile.file.name}
                      </span>
                      <span className="text-xs px-2 py-0.5 rounded-full text-gray-400 bg-gray-700/50">
                        {getFileTypeLabel(selectedFile.file.type)}
                      </span>
                    </div>
                    <p className="text-gray-500 text-sm">
                      {formatFileSize(selectedFile.file.size)}
                    </p>
                  </div>

                  {/* Remove Button */}
                  <button
                    type="button"
                    onClick={handleClearFile}
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
                  onChange={(e) => setAssetName(e.target.value)}
                  className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-green-500 transition-colors"
                  placeholder="Enter asset name"
                />
                <p className="mt-2 text-xs text-gray-500">
                  Saving to: <span className="text-gray-400 font-mono">{folder || 'root'}/</span>
                </p>
              </div>
            </>
          )}

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={handleClose}
              className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
              disabled={isUploading}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleUpload}
              disabled={!selectedFile || !assetName.trim() || isUploading}
              className="px-5 py-2.5 bg-gradient-to-r from-green-500 to-emerald-600 text-white font-medium rounded-xl hover:from-green-400 hover:to-emerald-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-green-500/20 hover:shadow-green-500/30"
            >
              {isUploading ? (
                <span className="flex items-center gap-2">
                  <Loader size={16} className="animate-spin" />
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
      </div>
    </ResponsiveDialog>
  );
}

/** File type icon based on MIME type */
function FileTypeIcon({ mimeType }: { mimeType: string }) {
  const iconClasses = 'text-gray-400';

  if (mimeType.startsWith('image/')) {
    return <span className={`text-blue-400 text-2xl`}>🖼️</span>;
  }
  if (mimeType === 'application/json') {
    return <span className={`text-yellow-400 text-2xl`}>📋</span>;
  }
  if (mimeType === 'text/csv') {
    return <span className={`text-green-400 text-2xl`}>📊</span>;
  }
  if (mimeType.startsWith('text/')) {
    return <span className={`${iconClasses} text-2xl`}>📄</span>;
  }
  return <span className={`${iconClasses} text-2xl`}>📁</span>;
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
