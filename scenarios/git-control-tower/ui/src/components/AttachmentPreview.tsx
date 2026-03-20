import { X, Loader2, AlertCircle, Check } from "lucide-react";
import type { AttachmentState } from "../hooks/useAttachments";

interface AttachmentPreviewProps {
  attachments: AttachmentState[];
  onRemove: (id: string) => void;
}

export function AttachmentPreview({
  attachments,
  onRemove,
}: AttachmentPreviewProps) {
  if (attachments.length === 0) {
    return null;
  }

  return (
    <div className="flex gap-2 px-4 py-2 overflow-x-auto">
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

function AttachmentThumbnail({
  attachment,
  onRemove,
}: {
  attachment: AttachmentState;
  onRemove: () => void;
}) {
  const isPending = attachment.uploadStatus === "pending";
  const isUploading = attachment.uploadStatus === "uploading";
  const isError = attachment.uploadStatus === "error";
  const isUploaded = attachment.uploadStatus === "uploaded";

  return (
    <div className="relative group shrink-0">
      <div
        className={`relative w-16 h-16 rounded-lg overflow-hidden border ${
          isError
            ? "border-red-500/50 bg-red-500/10"
            : "border-slate-700 bg-slate-800"
        }`}
      >
        {attachment.previewUrl && (
          <img
            src={attachment.previewUrl}
            alt={attachment.file.name}
            className="w-full h-full object-cover"
          />
        )}

        {(isPending || isUploading) && (
          <div className="absolute inset-0 bg-black/60 flex items-center justify-center">
            <Loader2 className="h-5 w-5 text-white animate-spin" />
          </div>
        )}

        {isError && (
          <div className="absolute inset-0 bg-red-500/20 flex items-center justify-center">
            <AlertCircle className="h-5 w-5 text-red-400" />
          </div>
        )}

        {isUploaded && (
          <div className="absolute bottom-0.5 right-0.5 bg-green-500 rounded-full p-0.5">
            <Check className="h-2.5 w-2.5 text-white" />
          </div>
        )}
      </div>

      <button
        type="button"
        onClick={onRemove}
        className="absolute -top-1.5 -right-1.5 bg-slate-800 border border-slate-600 rounded-full p-0.5 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-500/20 hover:border-red-500/50"
      >
        <X className="h-3 w-3 text-white" />
      </button>

      {isError && attachment.error && (
        <div className="absolute -bottom-5 left-0 right-0">
          <div className="text-[10px] text-red-400 truncate text-center">
            {attachment.error}
          </div>
        </div>
      )}
    </div>
  );
}
