import { useState, useCallback, useId, useRef } from "react";
import {
  X,
  Upload,
  AlertCircle,
  FileImage,
  FileJson,
  FileText,
  File,
  Check,
  Loader,
} from "lucide-react";
import { ResponsiveDialog } from "@shared/layout";
import toast from "react-hot-toast";

// Supported file types
const SUPPORTED_TYPES = {
  // Images
  "image/png": { icon: FileImage, label: "PNG Image", color: "text-blue-400" },
  "image/jpeg": { icon: FileImage, label: "JPEG Image", color: "text-blue-400" },
  "image/gif": { icon: FileImage, label: "GIF Image", color: "text-blue-400" },
  "image/svg+xml": { icon: FileImage, label: "SVG Image", color: "text-blue-400" },
  "image/webp": { icon: FileImage, label: "WebP Image", color: "text-blue-400" },
  // Data
  "application/json": { icon: FileJson, label: "JSON", color: "text-yellow-400" },
  "text/csv": { icon: FileText, label: "CSV", color: "text-green-400" },
  // Text
  "text/plain": { icon: FileText, label: "Text", color: "text-gray-400" },
  "text/markdown": { icon: FileText, label: "Markdown", color: "text-purple-400" },
} as const;

type SupportedMimeType = keyof typeof SUPPORTED_TYPES;

const SUPPORTED_EXTENSIONS = [".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".json", ".csv", ".txt", ".md"];

interface AssetUploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  folder: string;
  projectId: string;
  onSuccess?: (assetPath: string) => void;
}

function getFileTypeInfo(file: File) {
  const mimeType = file.type as SupportedMimeType;
  if (SUPPORTED_TYPES[mimeType]) {
    return SUPPORTED_TYPES[mimeType];
  }
  // Fallback based on extension
  const ext = file.name.toLowerCase().split(".").pop();
  if (ext === "md") {
    return SUPPORTED_TYPES["text/markdown"];
  }
  return { icon: File, label: "File", color: "text-gray-400" };
}

function isFileSupported(file: File): boolean {
  const mimeType = file.type as SupportedMimeType;
  if (SUPPORTED_TYPES[mimeType]) {
    return true;
  }
  // Check extension for markdown
  const ext = file.name.toLowerCase().split(".").pop();
  return ext === "md";
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function AssetUploadModal({
  isOpen,
  onClose,
  folder,
  projectId,
  onSuccess,
}: AssetUploadModalProps) {
  const titleId = useId();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [assetName, setAssetName] = useState("");
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);

  const handleFileSelect = useCallback((file: File) => {
    if (!isFileSupported(file)) {
      setValidationError(`Unsupported file type. Supported: ${SUPPORTED_EXTENSIONS.join(", ")}`);
      return;
    }

    // Max 10MB
    if (file.size > 10 * 1024 * 1024) {
      setValidationError("File size must be under 10MB");
      return;
    }

    setSelectedFile(file);
    setValidationError(null);

    // Auto-generate asset name from filename (without extension)
    const nameWithoutExt = file.name.replace(/\.[^/.]+$/, "");
    setAssetName(nameWithoutExt);

    // Create preview for images
    if (file.type.startsWith("image/")) {
      const url = URL.createObjectURL(file);
      setPreviewUrl(url);
    } else {
      setPreviewUrl(null);
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    const file = e.dataTransfer.files[0];
    if (file) {
      handleFileSelect(file);
    }
  }, [handleFileSelect]);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      handleFileSelect(file);
    }
  }, [handleFileSelect]);

  const handleUpload = useCallback(async () => {
    if (!selectedFile || !assetName.trim()) {
      setValidationError("Please select a file and enter an asset name");
      return;
    }

    setIsUploading(true);
    setValidationError(null);

    try {
      // Build the asset path
      const ext = selectedFile.name.split(".").pop() || "";
      const assetPath = folder
        ? `${folder}/${assetName.trim()}.${ext}`
        : `${assetName.trim()}.${ext}`;

      // Upload the file
      const formData = new FormData();
      formData.append("file", selectedFile);
      formData.append("path", assetPath);
      formData.append("project_id", projectId);

      const response = await fetch("/api/projects/assets", {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || "Upload failed");
      }

      toast.success("Asset uploaded successfully");
      onSuccess?.(assetPath);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Upload failed";
      setValidationError(message);
      toast.error(message);
    } finally {
      setIsUploading(false);
    }
  }, [selectedFile, assetName, folder, projectId, onSuccess, onClose]);

  const handleClearFile = useCallback(() => {
    setSelectedFile(null);
    setAssetName("");
    setPreviewUrl(null);
    setValidationError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  }, []);

  // Cleanup preview URL on unmount
  const handleClose = useCallback(() => {
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    handleClearFile();
    onClose();
  }, [previewUrl, handleClearFile, onClose]);

  if (!isOpen) return null;

  const fileInfo = selectedFile ? getFileTypeInfo(selectedFile) : null;
  const FileIcon = fileInfo?.icon || File;

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
            onClick={handleClose}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close dialog"
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
            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-xl flex items-center gap-3">
              <div className="w-8 h-8 bg-red-500/20 rounded-lg flex items-center justify-center flex-shrink-0">
                <AlertCircle size={16} className="text-red-400" />
              </div>
              <span className="text-red-300 text-sm">{validationError}</span>
            </div>
          )}

          {/* Drop Zone */}
          {!selectedFile ? (
            <div
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onClick={() => fileInputRef.current?.click()}
              className={`
                relative border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-all
                ${isDragging
                  ? "border-green-500 bg-green-500/10"
                  : "border-gray-700 bg-gray-800/30 hover:border-gray-600 hover:bg-gray-800/50"
                }
              `}
            >
              <input
                ref={fileInputRef}
                type="file"
                accept={SUPPORTED_EXTENSIONS.join(",")}
                onChange={handleInputChange}
                className="hidden"
              />

              <div className="w-16 h-16 mx-auto mb-4 rounded-xl bg-gray-700/50 flex items-center justify-center">
                <Upload size={32} className={isDragging ? "text-green-400" : "text-gray-400"} />
              </div>

              <p className="text-gray-300 font-medium mb-1">
                {isDragging ? "Drop file here" : "Drag and drop a file"}
              </p>
              <p className="text-gray-500 text-sm mb-3">
                or click to browse
              </p>
              <p className="text-gray-600 text-xs">
                Supports: Images (PNG, JPG, GIF, SVG, WebP), JSON, CSV, TXT, MD
              </p>
              <p className="text-gray-600 text-xs mt-1">
                Max size: 10MB
              </p>
            </div>
          ) : (
            <>
              {/* Selected File Preview */}
              <div className="mb-5 p-4 bg-gray-800/50 border border-gray-700/50 rounded-xl">
                <div className="flex items-start gap-4">
                  {/* Preview or Icon */}
                  {previewUrl ? (
                    <img
                      src={previewUrl}
                      alt="Preview"
                      className="w-20 h-20 object-cover rounded-lg border border-gray-700"
                    />
                  ) : (
                    <div className={`w-20 h-20 rounded-lg bg-gray-700/50 flex items-center justify-center ${fileInfo?.color}`}>
                      <FileIcon size={32} />
                    </div>
                  )}

                  {/* File Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-white font-medium truncate">
                        {selectedFile.name}
                      </span>
                      <span className={`text-xs px-2 py-0.5 rounded-full ${fileInfo?.color} bg-gray-700/50`}>
                        {fileInfo?.label}
                      </span>
                    </div>
                    <p className="text-gray-500 text-sm">
                      {formatFileSize(selectedFile.size)}
                    </p>
                  </div>

                  {/* Remove Button */}
                  <button
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
                  Saving to: <span className="text-gray-400 font-mono">{folder || "root"}/</span>
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

export default AssetUploadModal;
