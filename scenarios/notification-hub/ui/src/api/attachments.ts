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

/** Same-origin URL for an image saved with the recipient's answer. */
export function answerImageUrl(askId: string, name: string): string {
  return buildApiUrl(`/asks/${encodeURIComponent(askId)}/answer-images/${encodeURIComponent(name)}`, { baseUrl: REST_API_BASE });
}

/** Fetch attachment bytes through the session fetch (routed test headers, sign-in expiry). */
export async function fetchAttachment(askId: string, name: string, entryPath = "", signal?: AbortSignal): Promise<Response> {
  const response = await sessionFetch(attachmentUrl(askId, name, entryPath), { signal });
  if (!response.ok) throw new Error(`attachment ${name} unavailable (${response.status})`);
  return response;
}

export interface LinkedArtifact {
  name: string;
  kind: string;
  mimeType: string;
  sizeBytes: number;
  contentUrl: string;
}

/** Resolve a local path from this owner's decision into a same-origin Hub viewer source. */
export async function resolveLinkedArtifact(askId: string, path: string): Promise<LinkedArtifact> {
  const url = new URL(buildApiUrl(`/asks/${encodeURIComponent(askId)}/linked-artifact/metadata`, { baseUrl: REST_API_BASE }), window.location.origin);
  url.searchParams.set("path", path);
  const response = await sessionFetch(url);
  if (!response.ok) throw new Error(`linked artifact unavailable (${response.status})`);
  const result = await response.json() as LinkedArtifact;
  if (!result.name || !result.kind || !result.mimeType || !Number.isFinite(result.sizeBytes)) {
    throw new Error("linked artifact metadata is incomplete");
  }
  const content = new URL(result.contentUrl, window.location.origin);
  if (content.origin !== window.location.origin || !content.pathname.includes("/linked-artifact/content")) {
    throw new Error("linked artifact returned an unsafe content address");
  }
  content.searchParams.set("path", path);
  return { ...result, contentUrl: content.href };
}
