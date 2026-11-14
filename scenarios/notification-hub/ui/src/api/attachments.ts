import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

import { sessionFetch } from "./notifications";

const REST_API_BASE = resolveApiBase({ appendSuffix: true });

/**
 * Where the Hub serves one stored attachment copy
 * (`GET /api/v1/asks/{ask_id}/attachments/{name}`). A file inside a folder
 * attachment is addressed with `?path=`. The URL is same-origin, so `<img>`,
 * `<video>` and friends carry the owner's HttpOnly session cookie themselves.
 */
export function attachmentUrl(askId: string, name: string, entryPath = ""): string {
  const base = buildApiUrl(`/asks/${encodeURIComponent(askId)}/attachments/${encodeURIComponent(name)}`, { baseUrl: REST_API_BASE });
  return entryPath ? `${base}?path=${encodeURIComponent(entryPath)}` : base;
}

/** Fetch attachment bytes through the session fetch (routed test headers, sign-in expiry). */
export async function fetchAttachment(askId: string, name: string, entryPath = "", signal?: AbortSignal): Promise<Response> {
  const response = await sessionFetch(attachmentUrl(askId, name, entryPath), { signal });
  if (!response.ok) throw new Error(`attachment ${name} unavailable (${response.status})`);
  return response;
}
