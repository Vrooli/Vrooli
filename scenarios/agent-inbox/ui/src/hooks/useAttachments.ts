/**
 * useAttachments - Hook for managing file attachments in the message input.
 *
 * Handles:
 * - Adding files (with preview generation for images)
 * - Uploading files to the server
 * - Tracking upload status
 * - Removing attachments
 *
 * DESIGN: Attachments are uploaded immediately when added, not on message send.
 * This provides better UX (progress feedback) and simplifies the send flow.
 */
import { useState, useCallback } from "react";
import { uploadAttachment, type UploadResponse } from "../lib/api";

export type AttachmentType = "image" | "pdf";

export type UploadStatus = "pending" | "uploading" | "uploaded" | "error";

export interface AttachmentState {
  id: string;           // Local ID for tracking before upload completes
  file: File;
  type: AttachmentType;
  previewUrl?: string;  // Data URL for image preview
  uploadStatus: UploadStatus;
  serverId?: string;    // Server-assigned ID after upload
  serverPath?: string;  // Storage path on server
  error?: string;
}

export interface UseAttachmentsReturn {
  attachments: AttachmentState[];
  addAttachment: (file: File, type: AttachmentType) => void;
  removeAttachment: (id: string) => void;
  clearAttachments: () => void;
  isUploading: boolean;
  hasErrors: boolean;
  allUploaded: boolean;
  getUploadedIds: () => string[];
}

let attachmentIdCounter = 0;

function generateLocalId(): string {
  return `local-${++attachmentIdCounter}-${Date.now()}`;
}

function getAttachmentType(file: File): AttachmentType {
  if (file.type.startsWith("image/")) {
    return "image";
  }
  return "pdf";
}

export function useAttachments(): UseAttachmentsReturn {
  const [attachments, setAttachments] = useState<AttachmentState[]>([]);

  const addAttachment = useCallback((file: File, type?: AttachmentType) => {
    const id = generateLocalId();
    const resolvedType = type || getAttachmentType(file);

    // Create initial state
    const newAttachment: AttachmentState = {
      id,
      file,
      type: resolvedType,
      uploadStatus: "pending",
    };

    // Generate preview for images
    if (resolvedType === "image") {
      const reader = new FileReader();
      reader.onload = (e) => {
        setAttachments((prev) =>
          prev.map((att) =>
            att.id === id ? { ...att, previewUrl: e.target?.result as string } : att
          )
        );
      };
      reader.readAsDataURL(file);
    }

    // Add to state
    setAttachments((prev) => [...prev, newAttachment]);

    // Start upload immediately
    uploadFile(id, file);
  }, []);

  const uploadFile = async (id: string, file: File) => {
    // Mark as uploading
    setAttachments((prev) =>
      prev.map((att) =>
        att.id === id ? { ...att, uploadStatus: "uploading" } : att
      )
    );

    try {
      const response: UploadResponse = await uploadAttachment(file);

      // Mark as uploaded with server data
      setAttachments((prev) =>
        prev.map((att) =>
          att.id === id
            ? {
                ...att,
                uploadStatus: "uploaded",
                serverId: response.id,
                serverPath: response.storage_path,
              }
            : att
        )
      );
    } catch (error) {
      // Mark as error
      setAttachments((prev) =>
        prev.map((att) =>
          att.id === id
            ? {
                ...att,
                uploadStatus: "error",
                error: error instanceof Error ? error.message : "Upload failed",
              }
            : att
        )
      );
    }
  };

  const removeAttachment = useCallback((id: string) => {
    setAttachments((prev) => {
      const attachment = prev.find((att) => att.id === id);
      // Revoke object URL if it exists to prevent memory leaks
      if (attachment?.previewUrl?.startsWith("blob:")) {
        URL.revokeObjectURL(attachment.previewUrl);
      }
      return prev.filter((att) => att.id !== id);
    });
  }, []);

  const clearAttachments = useCallback(() => {
    setAttachments((prev) => {
      // Revoke all object URLs
      prev.forEach((att) => {
        if (att.previewUrl?.startsWith("blob:")) {
          URL.revokeObjectURL(att.previewUrl);
        }
      });
      return [];
    });
  }, []);

  const isUploading = attachments.some((att) => att.uploadStatus === "uploading");
  const hasErrors = attachments.some((att) => att.uploadStatus === "error");
  const allUploaded = attachments.length > 0 && attachments.every((att) => att.uploadStatus === "uploaded");

  const getUploadedIds = useCallback((): string[] => {
    return attachments
      .filter((att) => att.uploadStatus === "uploaded" && att.serverId)
      .map((att) => att.serverId!);
  }, [attachments]);

  return {
    attachments,
    addAttachment,
    removeAttachment,
    clearAttachments,
    isUploading,
    hasErrors,
    allUploaded,
    getUploadedIds,
  };
}
