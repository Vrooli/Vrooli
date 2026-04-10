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
import { getApiBaseUrl } from "../lib/utils";

export interface UploadResponse {
  id: string;
  file_name: string;
  content_type: string;
  file_size: number;
  storage_path: string;
  url: string;
}

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

/** Serializable attachment metadata for localStorage persistence. */
export interface PersistedAttachment {
  serverId: string;
  fileName: string;
  contentType: string;
  serverPath: string;
  url: string;
}

export interface UseAttachmentsReturn {
  attachments: AttachmentState[];
  addAttachment: (file: File, type?: AttachmentType) => void;
  removeAttachment: (id: string) => void;
  clearAttachments: () => void;
  restoreAttachments: (persisted: PersistedAttachment[]) => void;
  getPersistedAttachments: () => PersistedAttachment[];
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

async function defaultUploadAttachment(file: File): Promise<UploadResponse> {
  const base = getApiBaseUrl();
  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(`${base}/attachments/upload`, {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    if (res.status === 413) throw new Error("File is too large");
    if (res.status === 415) throw new Error("File type not supported");
    throw new Error(`Failed to upload: ${res.status}`);
  }

  return res.json() as Promise<UploadResponse>;
}

export function useAttachments(uploadFn?: (file: File) => Promise<UploadResponse>): UseAttachmentsReturn {
  const [attachments, setAttachments] = useState<AttachmentState[]>([]);

  const upload = uploadFn ?? defaultUploadAttachment;

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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [upload]);

  const uploadFile = async (id: string, file: File) => {
    // Mark as uploading
    setAttachments((prev) =>
      prev.map((att) =>
        att.id === id ? { ...att, uploadStatus: "uploading" } : att
      )
    );

    try {
      const response: UploadResponse = await upload(file);

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
      .filter((att): att is typeof att & { serverId: string } =>
        att.uploadStatus === "uploaded" && !!att.serverId)
      .map((att) => att.serverId);
  }, [attachments]);

  /** Restore previously-uploaded attachments from persisted metadata (e.g. localStorage). */
  const restoreAttachments = useCallback((persisted: PersistedAttachment[]) => {
    const restored: AttachmentState[] = persisted.map((p) => ({
      id: generateLocalId(),
      // Create a minimal placeholder File — the real data lives on the server
      file: new File([], p.fileName, { type: p.contentType }),
      type: (p.contentType.startsWith("image/") ? "image" : "pdf") as AttachmentType,
      previewUrl: p.url,
      uploadStatus: "uploaded" as UploadStatus,
      serverId: p.serverId,
      serverPath: p.serverPath,
    }));
    setAttachments(restored);
  }, []);

  /** Get serializable metadata for all successfully-uploaded attachments. */
  const getPersistedAttachments = useCallback((): PersistedAttachment[] => {
    return attachments
      .filter((att): att is typeof att & { serverId: string; serverPath: string } =>
        att.uploadStatus === "uploaded" && !!att.serverId && !!att.serverPath)
      .map((att) => ({
        serverId: att.serverId,
        fileName: att.file.name,
        contentType: att.file.type,
        serverPath: att.serverPath,
        url: att.previewUrl ?? "",
      }));
  }, [attachments]);

  return {
    attachments,
    addAttachment,
    removeAttachment,
    clearAttachments,
    restoreAttachments,
    getPersistedAttachments,
    isUploading,
    hasErrors,
    allUploaded,
    getUploadedIds,
  };
}
