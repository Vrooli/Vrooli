import { useState, useRef, useCallback } from 'react';
import { Upload, X, Loader2, AlertCircle, Image as ImageIcon } from 'lucide-react';
import { Button } from './button';
import { uploadAsset, getAssetUrl, type AssetCategory, type Asset } from '../api';

export interface ImageUploaderProps {
  /** Current image URL (can be external URL or uploaded asset URL) */
  value?: string | null;
  /** Called when image changes (upload complete or cleared) */
  onChange: (url: string | null) => void;
  /** Asset category for organization */
  category?: AssetCategory;
  /** Placeholder text when no image */
  placeholder?: string;
  /** Label for the upload button */
  uploadLabel?: string;
  /** Preview size: 'sm' (32px), 'md' (64px), 'lg' (96px), 'xl' (128px) */
  previewSize?: 'sm' | 'md' | 'lg' | 'xl';
  /** Accepted file types */
  accept?: string;
  /** Max file size in bytes (default 5MB) */
  maxSize?: number;
  /** Whether to allow clearing the image */
  clearable?: boolean;
  /** Whether to show URL input as fallback */
  allowUrlInput?: boolean;
  /** Alt text for accessibility */
  alt?: string;
  /** Additional CSS classes */
  className?: string;
  /** Disabled state */
  disabled?: boolean;
  /** Callback with full asset response (derivatives, metadata) */
  onUploadComplete?: (asset: Asset) => void;
}

const sizeClasses = {
  sm: 'h-8 w-8',
  md: 'h-16 w-16',
  lg: 'h-24 w-24',
  xl: 'h-32 w-32',
};

export function ImageUploader({
  value,
  onChange,
  category = 'general',
  placeholder = 'No image',
  uploadLabel = 'Upload',
  previewSize = 'md',
  accept = 'image/png,image/jpeg,image/gif,image/webp,image/svg+xml,image/x-icon',
  maxSize = 5 * 1024 * 1024,
  clearable = true,
  allowUrlInput = true,
  alt = 'Uploaded image',
  className = '',
  disabled = false,
  onUploadComplete,
}: ImageUploaderProps) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showUrlInput, setShowUrlInput] = useState(false);
  const [urlInputValue, setUrlInputValue] = useState('');
  const [imageError, setImageError] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    // Validate file size
    if (file.size > maxSize) {
      setError(`File too large. Maximum size is ${Math.round(maxSize / 1024 / 1024)}MB`);
      return;
    }

    setError(null);
    setUploading(true);

    try {
      const asset = await uploadAsset(file, { category });
      onChange(asset.url);
      onUploadComplete?.(asset);
      setImageError(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
      // Reset file input so same file can be selected again
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  }, [category, maxSize, onChange]);

  const handleUrlSubmit = useCallback(() => {
    const url = urlInputValue.trim();
    if (url) {
      onChange(url);
      setImageError(false);
    }
    setShowUrlInput(false);
    setUrlInputValue('');
  }, [urlInputValue, onChange]);

  const handleClear = useCallback(() => {
    onChange(null);
    setError(null);
    setImageError(false);
  }, [onChange]);

  const handleDrop = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();

    const file = event.dataTransfer.files?.[0];
    if (!file) return;

    // Check if file is an image
    if (!file.type.startsWith('image/')) {
      setError('Please drop an image file');
      return;
    }

    // Create a synthetic event for the file handler
    const dataTransfer = new DataTransfer();
    dataTransfer.items.add(file);
    const syntheticEvent = {
      target: { files: dataTransfer.files },
    } as React.ChangeEvent<HTMLInputElement>;
    handleFileSelect(syntheticEvent);
  }, [handleFileSelect]);

  const handleDragOver = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
  }, []);

  const resolvedUrl = value ? getAssetUrl(value) : null;

  return (
    <div className={`space-y-2 ${className}`}>
      {/* Preview and Upload Area */}
      <div
        className="flex items-center gap-4"
        onDrop={handleDrop}
        onDragOver={handleDragOver}
      >
        {/* Image Preview */}
        <div
          className={`${sizeClasses[previewSize]} flex-shrink-0 rounded-lg border border-white/10 bg-slate-800 overflow-hidden flex items-center justify-center`}
        >
          {resolvedUrl && !imageError ? (
            <img
              src={resolvedUrl}
              alt={alt}
              className="h-full w-full object-contain"
              onError={() => setImageError(true)}
            />
          ) : (
            <ImageIcon className="h-6 w-6 text-slate-500" />
          )}
        </div>

        {/* Actions */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            {/* Hidden file input */}
            <input
              ref={fileInputRef}
              type="file"
              accept={accept}
              onChange={handleFileSelect}
              className="hidden"
              disabled={disabled || uploading}
            />

            {/* Upload button */}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
              disabled={disabled || uploading}
              className="gap-2"
            >
              {uploading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Upload className="h-4 w-4" />
              )}
              {uploading ? 'Uploading...' : uploadLabel}
            </Button>

            {/* URL input toggle */}
            {allowUrlInput && !showUrlInput && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setShowUrlInput(true)}
                disabled={disabled || uploading}
                className="text-slate-400 hover:text-slate-200"
              >
                or enter URL
              </Button>
            )}

            {/* Clear button */}
            {clearable && value && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleClear}
                disabled={disabled || uploading}
                className="text-rose-400 hover:text-rose-300"
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>

          {/* URL input */}
          {showUrlInput && (
            <div className="flex items-center gap-2">
              <input
                type="url"
                value={urlInputValue}
                onChange={(e) => setUrlInputValue(e.target.value)}
                placeholder="https://example.com/image.png"
                className="flex-1 rounded-lg border border-white/10 bg-slate-900/70 px-3 py-1.5 text-sm text-white"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    handleUrlSubmit();
                  } else if (e.key === 'Escape') {
                    setShowUrlInput(false);
                    setUrlInputValue('');
                  }
                }}
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleUrlSubmit}
              >
                Set
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  setShowUrlInput(false);
                  setUrlInputValue('');
                }}
              >
                Cancel
              </Button>
            </div>
          )}

          {/* Current value display */}
          {value && !showUrlInput && (
            <p className="text-xs text-slate-500 truncate max-w-[300px]" title={value}>
              {value}
            </p>
          )}

          {/* Placeholder when no image */}
          {!value && !showUrlInput && (
            <p className="text-xs text-slate-500">{placeholder}</p>
          )}
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="flex items-center gap-2 text-sm text-rose-400">
          <AlertCircle className="h-4 w-4" />
          {error}
        </div>
      )}

      {/* Image load error */}
      {imageError && value && (
        <div className="flex items-center gap-2 text-sm text-amber-400">
          <AlertCircle className="h-4 w-4" />
          Failed to load image preview
        </div>
      )}
    </div>
  );
}
