/**
 * AttachmentPreview - Shows attached files above the message input.
 *
 * Displays:
 * - Thumbnail/icon for each attachment
 * - Upload progress/status
 * - Remove button
 */
import { X, FileText, Loader2, AlertCircle, Check } from "lucide-react";
import type { AttachmentState } from "../../hooks/useAttachments";

interface AttachmentPreviewProps {
  attachments: AttachmentState[];
  onRemove: (id: string) => void;
  isUploading: boolean;
}

export function AttachmentPreview({
  attachments,
  onRemove,
  isUploading: _isUploading,
}: AttachmentPreviewProps) {
  if (attachments.length === 0) {
    return null;
  }

  return (
    <div
      className="flex gap-2 px-4 py-2 overflow-x-auto scrollbar-thin scrollbar-thumb-white/20"
      data-testid="attachment-preview-container"
    >
      {attachments.map((attachment) => (
        <AttachmentThumbnail
          key={attachment.id}
          attachment={attachment}
          onRemove={() => onRemove(attachment.id)}
        />
      ))}
    </div>
  );
}

interface AttachmentThumbnailProps {
  attachment: AttachmentState;
  onRemove: () => void;
}

function AttachmentThumbnail({ attachment, onRemove }: AttachmentThumbnailProps) {
  const isImage = attachment.type === "image";
  const isPending = attachment.uploadStatus === "pending";
  const isUploading = attachment.uploadStatus === "uploading";
  const isError = attachment.uploadStatus === "error";
  const isUploaded = attachment.uploadStatus === "uploaded";

  return (
    <div
      className="relative group shrink-0"
      data-testid="attachment-thumbnail"
    >
      {/* Thumbnail container */}
      <div
        className={`
          relative w-20 h-20 rounded-lg overflow-hidden border
          ${isError ? "border-red-500/50 bg-red-500/10" : "border-white/10 bg-white/5"}
        `}
      >
        {/* Image preview */}
        {isImage && attachment.previewUrl && (
          <img
            src={attachment.previewUrl}
            alt={attachment.file.name}
            className="w-full h-full object-cover"
          />
        )}

        {/* PDF icon */}
        {!isImage && (
          <div className="w-full h-full flex flex-col items-center justify-center">
            <FileText className="h-8 w-8 text-orange-400" />
            <span className="text-xs text-slate-400 mt-1 px-1 truncate max-w-full">
              PDF
            </span>
          </div>
        )}

        {/* Upload overlay */}
        {(isPending || isUploading) && (
          <div className="absolute inset-0 bg-black/60 flex items-center justify-center">
            <Loader2 className="h-6 w-6 text-white animate-spin" />
          </div>
        )}

        {/* Error overlay */}
        {isError && (
          <div className="absolute inset-0 bg-red-500/20 flex items-center justify-center">
            <AlertCircle className="h-6 w-6 text-red-400" />
          </div>
        )}

        {/* Success indicator */}
        {isUploaded && (
          <div className="absolute bottom-1 right-1 bg-green-500 rounded-full p-0.5">
            <Check className="h-3 w-3 text-white" />
          </div>
        )}
      </div>

      {/* Remove button */}
      <button
        onClick={onRemove}
        className="absolute -top-2 -right-2 bg-slate-800 border border-white/20 rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-500/20 hover:border-red-500/50"
        data-testid="attachment-remove-button"
      >
        <X className="h-3 w-3 text-white" />
      </button>

      {/* File name tooltip on hover */}
      <div className="absolute -bottom-1 left-0 right-0 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
        <div className="bg-slate-900 border border-white/10 rounded px-1 py-0.5 text-xs text-slate-300 truncate mx-1">
          {attachment.file.name}
        </div>
      </div>

      {/* Error message */}
      {isError && attachment.error && (
        <div className="absolute -bottom-6 left-0 right-0">
          <div className="text-xs text-red-400 truncate text-center">
            {attachment.error}
          </div>
        </div>
      )}
    </div>
  );
}
