/**
 * useAttachments - Hook for managing image attachments in the agent message input.
 *
 * Handles:
 * - Adding files (with preview generation for images)
 * - Uploading files to the server via the GCT proxy
 * - Tracking upload status
 * - Removing attachments
 *
 * Attachments are uploaded immediately when added, not on message send.
 */
import { useState, useCallback } from "react";

export type UploadStatus = "pending" | "uploading" | "uploaded" | "error";

export interface AttachmentState {
  id: string;
  file: File;
  previewUrl?: string;
  uploadStatus: UploadStatus;
  serverId?: string;
  error?: string;
}

export interface UploadResponse {
  id: string;
  fileName: string;
  contentType: string;
  fileSize: number;
}

export interface UseAttachmentsReturn {
  attachments: AttachmentState[];
  addAttachment: (file: File) => void;
  removeAttachment: (id: string) => void;
  clearAttachments: () => void;
  isUploading: boolean;
  getUploadedIds: () => string[];
}

let attachmentIdCounter = 0;

export function useAttachments(
  uploadFn: (file: File) => Promise<UploadResponse>
): UseAttachmentsReturn {
  const [attachments, setAttachments] = useState<AttachmentState[]>([]);

  const addAttachment = useCallback(
    (file: File) => {
      const id = `local-${++attachmentIdCounter}-${Date.now()}`;

      const newAttachment: AttachmentState = {
        id,
        file,
        uploadStatus: "pending",
      };

      // Generate preview for images
      if (file.type.startsWith("image/")) {
        const reader = new FileReader();
        reader.onload = (e) => {
          setAttachments((prev) =>
            prev.map((att) =>
              att.id === id
                ? { ...att, previewUrl: e.target?.result as string }
                : att
            )
          );
        };
        reader.readAsDataURL(file);
      }

      setAttachments((prev) => [...prev, newAttachment]);

      // Start upload immediately
      (async () => {
        setAttachments((prev) =>
          prev.map((att) =>
            att.id === id ? { ...att, uploadStatus: "uploading" } : att
          )
        );

        try {
          const response = await uploadFn(file);
          setAttachments((prev) =>
            prev.map((att) =>
              att.id === id
                ? { ...att, uploadStatus: "uploaded", serverId: response.id }
                : att
            )
          );
        } catch (error) {
          setAttachments((prev) =>
            prev.map((att) =>
              att.id === id
                ? {
                    ...att,
                    uploadStatus: "error",
                    error:
                      error instanceof Error ? error.message : "Upload failed",
                  }
                : att
            )
          );
        }
      })();
    },
    [uploadFn]
  );

  const removeAttachment = useCallback((id: string) => {
    setAttachments((prev) => prev.filter((att) => att.id !== id));
  }, []);

  const clearAttachments = useCallback(() => {
    setAttachments([]);
  }, []);

  const isUploading = attachments.some(
    (att) => att.uploadStatus === "uploading"
  );

  const getUploadedIds = useCallback((): string[] => {
    return attachments
      .filter(
        (att): att is typeof att & { serverId: string } =>
          att.uploadStatus === "uploaded" && !!att.serverId
      )
      .map((att) => att.serverId);
  }, [attachments]);

  return {
    attachments,
    addAttachment,
    removeAttachment,
    clearAttachments,
    isUploading,
    getUploadedIds,
  };
}
