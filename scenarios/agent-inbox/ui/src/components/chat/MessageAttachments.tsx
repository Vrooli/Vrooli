import { useState } from "react";
import { XCircle } from "lucide-react";
import { resolveAttachmentUrl, type Attachment } from "../../lib/api";

// Component for displaying message attachments (images)
export interface MessageAttachmentsProps {
  attachments?: Attachment[];
  isUser?: boolean;
  compact?: boolean;
}

export function MessageAttachments({ attachments, isUser = false, compact = false }: MessageAttachmentsProps) {
  const [expandedImage, setExpandedImage] = useState<string | null>(null);

  if (!attachments || attachments.length === 0) {
    return null;
  }

  // Filter to only show images
  const images = attachments.filter(att =>
    att.content_type?.startsWith("image/")
  );

  if (images.length === 0) {
    return null;
  }

  const imageSize = compact ? "max-w-[150px] max-h-[150px]" : "max-w-[300px] max-h-[300px]";

  return (
    <>
      <div className={`flex flex-wrap gap-2 ${compact ? "mt-1 mb-1" : "mt-2 mb-2"}`}>
        {images.map((attachment) => {
          const resolvedUrl = resolveAttachmentUrl(attachment.url);
          return (
            <button
              key={attachment.id}
              onClick={() => setExpandedImage(resolvedUrl || null)}
              className={`relative group/img rounded-lg overflow-hidden border ${
                isUser
                  ? "border-indigo-400/30 hover:border-indigo-300/50"
                  : "border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500"
              } transition-colors cursor-pointer`}
            >
              <img
                src={resolvedUrl}
                alt={attachment.file_name}
                className={`${imageSize} object-cover`}
                loading="lazy"
              />
              <div className="absolute inset-0 bg-black/0 group-hover/img:bg-black/10 transition-colors" />
            </button>
          );
        })}
      </div>

      {/* Expanded image modal */}
      {expandedImage && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
          onClick={() => setExpandedImage(null)}
        >
          <button
            className="absolute top-4 right-4 p-2 text-white hover:text-gray-300 transition-colors"
            onClick={() => setExpandedImage(null)}
            aria-label="Close"
          >
            <XCircle className="h-8 w-8" />
          </button>
          <img
            src={expandedImage}
            alt="Expanded view"
            className="max-w-[90vw] max-h-[90vh] object-contain rounded-lg"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}
    </>
  );
}
