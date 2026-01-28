/**
 * FileUpload Component
 *
 * Provides drag-and-drop and click-to-upload functionality for adding files
 * to an idea folder.
 *
 * Features:
 * - Drag and drop support with visual feedback
 * - Click to open file picker
 * - Multiple file upload
 * - Upload progress indication
 * - Error handling with retry
 *
 * [REQ:REQ-P0-004] Drag-and-drop file upload for idea details page
 */

import { useCallback, useState, useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Upload, X, Loader2, CheckCircle2, AlertCircle } from "lucide-react";
import { cn, formatFileSize } from "../../lib";
import { ideasService } from "../../services";
import { Button } from "./button";

export interface FileUploadProps {
  /** Idea name to upload files to */
  ideaName: string;
  /** Optional subdirectory path within the idea */
  targetPath?: string;
  /** Called when upload completes successfully */
  onUploadComplete?: () => void;
  /** Optional className for styling */
  className?: string;
  /** data-testid attribute */
  "data-testid"?: string;
}

interface UploadState {
  file: File;
  status: "pending" | "uploading" | "success" | "error";
  error?: string;
}

/**
 * FileUpload component with drag-and-drop support.
 */
export function FileUpload({
  ideaName,
  targetPath,
  onUploadComplete,
  className,
  "data-testid": testId,
}: FileUploadProps) {
  const [isDragging, setIsDragging] = useState(false);
  const [uploads, setUploads] = useState<UploadState[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const queryClient = useQueryClient();

  // Mutation for uploading a single file
  const uploadMutation = useMutation({
    mutationFn: async ({ file, index }: { file: File; index: number }) => {
      setUploads((prev) =>
        prev.map((u, i) => (i === index ? { ...u, status: "uploading" } : u))
      );
      return ideasService.uploadFile(ideaName, file, targetPath);
    },
    onSuccess: (_, { index }) => {
      setUploads((prev) =>
        prev.map((u, i) => (i === index ? { ...u, status: "success" } : u))
      );
      // Invalidate files query to refresh the file tree
      queryClient.invalidateQueries({ queryKey: ["ideas", ideaName, "files"] });
      onUploadComplete?.();
    },
    onError: (error, { index }) => {
      const errorMessage = error instanceof Error ? error.message : "Upload failed";
      setUploads((prev) =>
        prev.map((u, i) =>
          i === index ? { ...u, status: "error", error: errorMessage } : u
        )
      );
    },
  });

  const handleFiles = useCallback(
    (files: FileList | File[]) => {
      const fileArray = Array.from(files);
      const newUploads: UploadState[] = fileArray.map((file) => ({
        file,
        status: "pending" as const,
      }));

      setUploads((prev) => [...prev, ...newUploads]);

      // Start uploading each file
      const startIndex = uploads.length;
      fileArray.forEach((file, i) => {
        uploadMutation.mutate({ file, index: startIndex + i });
      });
    },
    [uploads.length, uploadMutation]
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

      const files = e.dataTransfer.files;
      if (files.length > 0) {
        handleFiles(files);
      }
    },
    [handleFiles]
  );

  const handleClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleFileInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files;
      if (files && files.length > 0) {
        handleFiles(files);
      }
      // Reset input so the same file can be uploaded again
      e.target.value = "";
    },
    [handleFiles]
  );

  const removeUpload = useCallback((index: number) => {
    setUploads((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const retryUpload = useCallback(
    (index: number) => {
      const upload = uploads[index];
      if (upload) {
        uploadMutation.mutate({ file: upload.file, index });
      }
    },
    [uploads, uploadMutation]
  );

  const clearCompleted = useCallback(() => {
    setUploads((prev) => prev.filter((u) => u.status !== "success"));
  }, []);

  const hasCompletedUploads = uploads.some((u) => u.status === "success");

  return (
    <div className={cn("space-y-4", className)} data-testid={testId ?? "file-upload"}>
      {/* Drop zone */}
      <div
        className={cn(
          "relative rounded-lg border-2 border-dashed transition-colors cursor-pointer",
          isDragging
            ? "border-cyan-400 bg-cyan-400/10"
            : "border-slate-600 hover:border-slate-500 hover:bg-slate-800/30"
        )}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={handleClick}
        data-testid="file-upload-dropzone"
      >
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={handleFileInputChange}
          data-testid="file-upload-input"
        />

        <div className="flex flex-col items-center justify-center py-8 px-4">
          <Upload
            className={cn(
              "h-10 w-10 mb-3 transition-colors",
              isDragging ? "text-cyan-400" : "text-slate-500"
            )}
          />
          <p className="text-sm text-slate-300 text-center">
            <span className="font-medium">Click to upload</span> or drag and drop
          </p>
          <p className="text-xs text-slate-500 mt-1">
            Any file type supported
          </p>
        </div>
      </div>

      {/* Upload list */}
      {uploads.length > 0 && (
        <div className="space-y-2" data-testid="file-upload-list">
          {/* Clear completed button */}
          {hasCompletedUploads && (
            <div className="flex justify-end">
              <button
                type="button"
                onClick={clearCompleted}
                className="text-xs text-slate-500 hover:text-slate-400"
                data-testid="file-upload-clear"
              >
                Clear completed
              </button>
            </div>
          )}

          {uploads.map((upload, index) => (
            <div
              key={`${upload.file.name}-${index}`}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-lg border",
                upload.status === "success" && "border-green-500/30 bg-green-500/10",
                upload.status === "error" && "border-red-500/30 bg-red-500/10",
                upload.status === "uploading" && "border-cyan-500/30 bg-cyan-500/10",
                upload.status === "pending" && "border-slate-600 bg-slate-800/30"
              )}
              data-testid={`file-upload-item-${index}`}
            >
              {/* Status icon */}
              {upload.status === "uploading" && (
                <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
              )}
              {upload.status === "success" && (
                <CheckCircle2 className="h-4 w-4 text-green-400" />
              )}
              {upload.status === "error" && (
                <AlertCircle className="h-4 w-4 text-red-400" />
              )}
              {upload.status === "pending" && (
                <div className="h-4 w-4 rounded-full border-2 border-slate-500" />
              )}

              {/* File name */}
              <span className="flex-1 text-sm text-slate-300 truncate">
                {upload.file.name}
              </span>

              {/* File size */}
              <span className="text-xs text-slate-500">
                {formatFileSize(upload.file.size)}
              </span>

              {/* Actions */}
              {upload.status === "error" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation();
                    retryUpload(index);
                  }}
                  className="h-6 px-2 text-xs"
                  data-testid={`file-upload-retry-${index}`}
                >
                  Retry
                </Button>
              )}

              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  removeUpload(index);
                }}
                className="text-slate-500 hover:text-slate-300"
                data-testid={`file-upload-remove-${index}`}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
