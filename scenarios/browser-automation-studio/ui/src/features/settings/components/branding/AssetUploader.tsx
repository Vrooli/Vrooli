/**
 * AssetUploader - Drag & drop component for uploading brand assets
 */

import { useState, useRef, useCallback } from 'react';
import { Upload, Image as ImageIcon, AlertCircle, X, Loader2 } from 'lucide-react';
import { useAssetStore } from '@stores/assetStore';
import type { AssetType } from '@lib/storage';
import { ALLOWED_MIME_TYPES, MAX_FILE_SIZE } from '@lib/storage';

interface AssetUploaderProps {
  defaultType?: AssetType;
  onUploadComplete?: () => void;
}

export function AssetUploader({ defaultType = 'other', onUploadComplete }: AssetUploaderProps) {
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { uploadAsset } = useAssetStore();

  const handleFiles = useCallback(
    async (files: FileList | File[]) => {
      const fileArray = Array.from(files);
      if (fileArray.length === 0) return;

      setIsUploading(true);
      setUploadError(null);

      let successCount = 0;
      let lastError: string | null = null;

      for (const file of fileArray) {
        try {
          await uploadAsset(file, defaultType);
          successCount++;
        } catch (err) {
          lastError = err instanceof Error ? err.message : 'Upload failed';
        }
      }

      setIsUploading(false);

      if (lastError && successCount === 0) {
        setUploadError(lastError);
      } else if (successCount > 0) {
        onUploadComplete?.();
      }
    },
    [uploadAsset, defaultType, onUploadComplete],
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(false);

      const { files } = e.dataTransfer;
      handleFiles(files);
    },
    [handleFiles],
  );

  const handleFileInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const { files } = e.target;
      if (files) {
        handleFiles(files);
      }
      // Reset input so the same file can be selected again
      e.target.value = '';
    },
    [handleFiles],
  );

  const handleClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const dismissError = useCallback(() => {
    setUploadError(null);
  }, []);

  const acceptTypes = ALLOWED_MIME_TYPES.join(',');
  const maxSizeMB = (MAX_FILE_SIZE / (1024 * 1024)).toFixed(0);

  return (
    <div className="space-y-3">
      {/* Drop Zone */}
      <button
        type="button"
        onClick={handleClick}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        disabled={isUploading}
        className={`
          relative w-full p-8 border-2 border-dashed rounded-xl transition-all
          flex flex-col items-center justify-center gap-3 cursor-pointer
          ${
            isDragging
              ? 'border-flow-accent bg-flow-accent/10 scale-[1.02]'
              : 'border-gray-700 hover:border-gray-600 hover:bg-gray-800/30'
          }
          ${isUploading ? 'opacity-60 cursor-wait' : ''}
        `}
      >
        {isUploading ? (
          <>
            <Loader2 size={32} className="text-flow-accent animate-spin" />
            <span className="text-sm text-gray-400">Uploading...</span>
          </>
        ) : (
          <>
            <div
              className={`p-3 rounded-full ${isDragging ? 'bg-flow-accent/20' : 'bg-gray-800'}`}
            >
              {isDragging ? (
                <ImageIcon size={24} className="text-flow-accent" />
              ) : (
                <Upload size={24} className="text-gray-400" />
              )}
            </div>
            <div className="text-center">
              <span className="text-sm font-medium text-white block">
                {isDragging ? 'Drop to upload' : 'Click or drag to upload'}
              </span>
              <span className="text-xs text-gray-500 mt-1 block">
                PNG, JPEG, or WebP up to {maxSizeMB}MB
              </span>
            </div>
          </>
        )}
      </button>

      {/* Hidden File Input */}
      <input
        ref={fileInputRef}
        type="file"
        accept={acceptTypes}
        multiple
        onChange={handleFileInputChange}
        className="hidden"
      />

      {/* Error Message */}
      {uploadError && (
        <div className="flex items-start gap-3 p-3 bg-red-500/10 border border-red-500/30 rounded-lg">
          <AlertCircle size={18} className="text-red-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm text-red-300">{uploadError}</p>
          </div>
          <button
            type="button"
            onClick={dismissError}
            className="p-1 text-red-400 hover:text-red-300 transition-colors"
          >
            <X size={16} />
          </button>
        </div>
      )}
    </div>
  );
}

export default AssetUploader;
