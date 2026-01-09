/**
 * useDropZone Hook
 *
 * Manages drag-and-drop state and file handling for import operations.
 * Supports file validation, multiple file selection, and folder selection.
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import type { SelectedFile, FileValidation } from '../types';
import { MAX_FILE_SIZE } from '../constants';

export interface UseDropZoneOptions {
  /** Accepted MIME types or extensions */
  accept?: readonly string[] | string[];
  /** Allow multiple file selection */
  multiple?: boolean;
  /** Maximum file size in bytes */
  maxSize?: number;
  /** Custom validation function */
  validate?: (file: File) => Promise<FileValidation>;
  /** Callback when files are selected */
  onFilesSelected?: (files: SelectedFile[]) => void;
  /** Callback when a folder path is selected (native directory picker) */
  onFolderSelected?: (path: string) => void;
  /** Whether the drop zone is disabled */
  disabled?: boolean;
}

export interface UseDropZoneReturn {
  /** Whether user is dragging over the zone */
  isDragging: boolean;
  /** Whether files are being processed */
  isProcessing: boolean;
  /** Selected files with validation and preview */
  selectedFiles: SelectedFile[];
  /** Error message if any */
  error: string | null;
  /** Handle drag over event */
  handleDragOver: (e: React.DragEvent) => void;
  /** Handle drag leave event */
  handleDragLeave: (e: React.DragEvent) => void;
  /** Handle drop event */
  handleDrop: (e: React.DragEvent) => void;
  /** Handle file input change */
  handleFileInputChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  /** Open native file picker */
  openFilePicker: () => void;
  /** Open native folder picker (if supported) */
  openFolderPicker: () => Promise<void>;
  /** Clear selected files */
  clearFiles: () => void;
  /** Clear error */
  clearError: () => void;
  /** Ref for hidden file input */
  fileInputRef: React.RefObject<HTMLInputElement>;
}

/** Check if file matches accept patterns */
function matchesAccept(file: File, accept?: readonly string[] | string[]): boolean {
  if (!accept || accept.length === 0) return true;

  const fileName = file.name.toLowerCase();
  const mimeType = file.type.toLowerCase();

  return accept.some((pattern) => {
    const p = pattern.toLowerCase().trim();
    // Extension pattern (e.g., ".json")
    if (p.startsWith('.')) {
      return fileName.endsWith(p);
    }
    // MIME type with wildcard (e.g., "image/*")
    if (p.endsWith('/*')) {
      const prefix = p.slice(0, -1);
      return mimeType.startsWith(prefix);
    }
    // Exact MIME type
    return mimeType === p;
  });
}

/** Validate a single file */
async function validateFile(
  file: File,
  options: UseDropZoneOptions
): Promise<FileValidation> {
  const { accept, maxSize = MAX_FILE_SIZE, validate } = options;

  // Check accept patterns
  if (!matchesAccept(file, accept)) {
    return {
      isValid: false,
      error: `File type not allowed. Accepted: ${accept?.join(', ')}`,
    };
  }

  // Check size
  if (file.size > maxSize) {
    const maxMB = (maxSize / (1024 * 1024)).toFixed(0);
    return {
      isValid: false,
      error: `File size exceeds ${maxMB}MB limit`,
    };
  }

  // Run custom validation if provided
  if (validate) {
    return validate(file);
  }

  return { isValid: true };
}

/** Create preview URL for images */
function createPreviewUrl(file: File): string | undefined {
  if (file.type.startsWith('image/')) {
    return URL.createObjectURL(file);
  }
  return undefined;
}

export function useDropZone(options: UseDropZoneOptions = {}): UseDropZoneReturn {
  const {
    accept,
    multiple = false,
    maxSize = MAX_FILE_SIZE,
    validate,
    onFilesSelected,
    onFolderSelected,
    disabled = false,
  } = options;

  const [isDragging, setIsDragging] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<SelectedFile[]>([]);
  const [error, setError] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const dragCounterRef = useRef(0);

  // Cleanup preview URLs on unmount
  useEffect(() => {
    return () => {
      selectedFiles.forEach((sf) => {
        if (sf.previewUrl) {
          URL.revokeObjectURL(sf.previewUrl);
        }
      });
    };
  }, [selectedFiles]);

  const processFiles = useCallback(
    async (files: FileList | File[]) => {
      if (disabled) return;

      const fileArray = Array.from(files);
      if (fileArray.length === 0) return;

      // If not multiple, only take first file
      const filesToProcess = multiple ? fileArray : [fileArray[0]];

      setIsProcessing(true);
      setError(null);

      try {
        const results: SelectedFile[] = [];

        for (const file of filesToProcess) {
          const validation = await validateFile(file, {
            accept,
            maxSize,
            validate,
          });
          const previewUrl = createPreviewUrl(file);

          results.push({
            file,
            previewUrl,
            validation,
          });
        }

        // Check if any files are invalid
        const invalidFiles = results.filter((r) => !r.validation?.isValid);
        if (invalidFiles.length > 0 && invalidFiles.length === results.length) {
          // All files invalid - show first error
          setError(invalidFiles[0].validation?.error || 'Invalid file');
          // Still set files so UI can show them with errors
        }

        setSelectedFiles(results);
        onFilesSelected?.(results);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to process files');
      } finally {
        setIsProcessing(false);
      }
    },
    [accept, disabled, maxSize, multiple, onFilesSelected, validate]
  );

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (disabled) return;

      // Use dragCounter to handle nested elements
      if (dragCounterRef.current === 0) {
        setIsDragging(true);
      }
      dragCounterRef.current++;
    },
    [disabled]
  );

  const handleDragLeave = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (disabled) return;

      dragCounterRef.current--;
      if (dragCounterRef.current === 0) {
        setIsDragging(false);
      }
    },
    [disabled]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragCounterRef.current = 0;
      setIsDragging(false);

      if (disabled) return;

      const { files } = e.dataTransfer;
      processFiles(files);
    },
    [disabled, processFiles]
  );

  const handleFileInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const { files } = e.target;
      if (files) {
        processFiles(files);
      }
      // Reset input so same file can be selected again
      e.target.value = '';
    },
    [processFiles]
  );

  const openFilePicker = useCallback(() => {
    if (disabled) return;
    fileInputRef.current?.click();
  }, [disabled]);

  const openFolderPicker = useCallback(async () => {
    if (disabled || !onFolderSelected) return;

    // Check for native directory picker support
    if ('showDirectoryPicker' in window) {
      try {
        // @ts-expect-error - showDirectoryPicker is not in all TS definitions
        const handle = await window.showDirectoryPicker();
        // In browser context, we can only get the directory name
        // For full path, we'd need electron/tauri integration
        onFolderSelected(handle.name);
      } catch (err) {
        // User cancelled or error
        if (err instanceof Error && err.name !== 'AbortError') {
          setError('Failed to select folder');
        }
      }
    } else {
      // Fallback: use webkitdirectory attribute
      const input = document.createElement('input');
      input.type = 'file';
      input.webkitdirectory = true;
      input.addEventListener('change', () => {
        const files = input.files;
        if (files && files.length > 0) {
          // Get common path prefix from first file
          const firstPath = files[0].webkitRelativePath;
          const folderName = firstPath.split('/')[0];
          onFolderSelected(folderName);
        }
      });
      input.click();
    }
  }, [disabled, onFolderSelected]);

  const clearFiles = useCallback(() => {
    // Revoke preview URLs
    selectedFiles.forEach((sf) => {
      if (sf.previewUrl) {
        URL.revokeObjectURL(sf.previewUrl);
      }
    });
    setSelectedFiles([]);
    setError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [selectedFiles]);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
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
  };
}
