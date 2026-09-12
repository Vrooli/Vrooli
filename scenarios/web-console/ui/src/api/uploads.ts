// Multipart file uploads for session attachments. Stays on plain HTTP
// because Connect-RPC binary payloads are not yet wired for this path.

import { buildApiUrl } from "@vrooli/api-base";
import { extractAPIError } from "../lib/errors";
import { API_BASE_WITH_SUFFIX } from "./client";

export async function uploadFile(sessionId: string, file: File | Blob, filename?: string): Promise<string> {
  const url = buildApiUrl(`/sessions/${sessionId}/upload`, { baseUrl: API_BASE_WITH_SUFFIX });
  const formData = new FormData();
  formData.append("file", file, filename ?? (file instanceof File ? file.name : "image.png"));
  const res = await fetch(url, { method: "POST", body: formData });
  if (!res.ok) throw await extractAPIError(res, "File upload failed");
  const data = (await res.json()) as { path: string };
  return data.path;
}
