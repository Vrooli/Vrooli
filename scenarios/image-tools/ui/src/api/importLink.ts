import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

import { decodeApiError, fetchBlob } from "./client";

interface ImportLinkResponse {
  input_ref: string;
  format: string;
  mime_type: string;
  width: number;
  height: number;
}

/** Import a public image URL through the server's SSRF-safe downloader. */
export async function importImageLink(url: string): Promise<File> {
  const response = await fetch(
    buildApiUrl("/import/link", { baseUrl: resolveApiBase({ appendSuffix: true }) }),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url }),
      cache: "no-store",
    },
  );
  if (!response.ok) {
    throw await decodeApiError(response);
  }
  const imported = (await response.json()) as ImportLinkResponse;
  if (!imported.input_ref || !imported.format || !imported.mime_type) {
    throw new Error("Image Tools returned an incomplete import result");
  }
  const blob = await fetchBlob(imported.input_ref);
  return new File([blob], `imported.${imported.format}`, {
    type: imported.mime_type || blob.type || "application/octet-stream",
  });
}
