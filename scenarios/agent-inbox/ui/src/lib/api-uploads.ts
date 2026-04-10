/**
 * Attachment upload API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";

// =============================================================================
// Upload Types
// =============================================================================

export interface UploadResponse {
  id: string;
  file_name: string;
  content_type: string;
  file_size: number;
  storage_path: string;
  url: string;
}

// =============================================================================
// Upload Functions
// =============================================================================

/**
 * Upload a file attachment.
 * @param file - The file to upload
 * @returns Upload response with file metadata and URL
 */
export async function uploadAttachment(file: File): Promise<UploadResponse> {
  const url = buildApiUrl("/attachments/upload", { baseUrl: API_BASE });

  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(url, {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    if (res.status === 413) {
      throw new Error("File is too large");
    }
    if (res.status === 415) {
      throw new Error("File type not supported");
    }
    throw new Error(`Failed to upload file: ${res.status}`);
  }

  return jsonResponse<UploadResponse>(res);
}

/**
 * Upload a file attachment for agent mode (proxied through agent-inbox to agent-manager).
 */
export async function uploadAgentAttachment(file: File): Promise<UploadResponse> {
  const url = buildApiUrl("/agent-attachments/upload", { baseUrl: API_BASE });

  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(url, {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    if (res.status === 413) throw new Error("File is too large");
    if (res.status === 415) throw new Error("File type not supported");
    throw new Error(`Failed to upload file: ${res.status}`);
  }

  return jsonResponse<UploadResponse>(res);
}
