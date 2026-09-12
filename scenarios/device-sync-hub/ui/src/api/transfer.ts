import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import {
  TransferService,
  ItemKind,
  Retention,
  UploadItemResponseSchema,
  type Item,
} from "@vrooli/proto-types/device-sync-hub/v1/transfer/transfer_pb";

import {
  authedFetch,
  buildApiUrl,
  decodeApiError,
  makeApiError,
  PROTO_READ_OPTIONS,
  REST_API_BASE,
  transport,
} from "./client";
import { DEVICE_TOKEN_HEADER } from "./transport";
import { readSessionCredentials } from "../features/session/store";

/** Typed client for the device-token-authed TransferService Connect RPCs. */
export const transferClient = createClient(TransferService, transport);

/** Wire value the REST upload endpoint expects for the `retention` field. */
export const RETENTION_FORM_VALUE: Record<Retention, string> = {
  [Retention.UNSPECIFIED]: "",
  [Retention.LIVE]: "live",
  [Retention.HELD]: "held",
  [Retention.PINNED]: "pinned",
};

export interface UploadOptions {
  retention?: Retention;
  targetDeviceId?: string;
  /** 0..1 progress callback driven by the XHR upload-progress events. */
  onProgress?: (fraction: number) => void;
  /** Abort signal so a staged send can be cancelled mid-flight. */
  signal?: AbortSignal;
  sessionId?: string;
  onSession?: (id: string) => void;
}

const RESUMABLE_THRESHOLD = 32 << 20;
const RESUMABLE_CHUNK_BYTES = 8 << 20;

/**
 * Upload a file to the REST multipart endpoint. The scaffold's generic
 * `uploadFile` rides `authedFetch` (so it carries the device token), but file
 * sends want byte-level progress, so this uses XHR directly and sets the
 * `X-Device-Token` header explicitly. Returns the proto-typed Item.
 */
export function uploadItem(file: File, options: UploadOptions = {}): Promise<Item> {
	if (file.size > RESUMABLE_THRESHOLD) return uploadResumable(file, options);
  const { retention = Retention.HELD, targetDeviceId = "", onProgress, signal } = options;
  const { deviceToken } = readSessionCredentials();

  return new Promise<Item>((resolve, reject) => {
    const form = new FormData();
    form.set("file", file, file.name);
    form.set("name", file.name);
    form.set("retention", RETENTION_FORM_VALUE[retention] || "held");
    if (targetDeviceId) {
      form.set("target_device_id", targetDeviceId);
    }

    const xhr = new XMLHttpRequest();
    xhr.open("POST", buildApiUrl("/transfer/items", { baseUrl: REST_API_BASE }));
    if (deviceToken) {
      xhr.setRequestHeader(DEVICE_TOKEN_HEADER, deviceToken);
    }
    xhr.responseType = "text";

    if (onProgress) {
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable && event.total > 0) {
          onProgress(event.loaded / event.total);
        }
      };
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const json = JSON.parse(xhr.responseText) as JsonValue;
          const decoded = fromJson(UploadItemResponseSchema, json, PROTO_READ_OPTIONS);
          if (!decoded.item) {
            reject(makeApiError("internal", "upload returned no item"));
            return;
          }
          resolve(decoded.item);
        } catch {
          reject(makeApiError("internal", "upload returned a malformed response", xhr.status));
        }
        return;
      }
      // Decode the structured ErrorEnvelope from the failure body.
      void decodeApiError(new Response(xhr.responseText, { status: xhr.status })).then(reject);
    };

    xhr.onerror = () => reject(makeApiError("unavailable", "network error during upload", 0));
    xhr.onabort = () => reject(makeApiError("canceled", "upload canceled", 0));

    if (signal) {
      if (signal.aborted) {
        reject(makeApiError("canceled", "upload canceled", 0));
        return;
      }
      signal.addEventListener("abort", () => xhr.abort(), { once: true });
    }

    xhr.send(form);
  });
}

async function uploadResumable(file: File, options: UploadOptions): Promise<Item> {
  const { retention = Retention.HELD, targetDeviceId = "", onProgress, sessionId, onSession, signal } = options;
  let id = sessionId;
  let received = new Set<number>();
  if (id) {
    const response = await authedFetch(buildApiUrl(`/transfer/uploads/${encodeURIComponent(id)}`, { baseUrl: REST_API_BASE }), { cache: "no-store", signal });
    if (!response.ok) throw await decodeApiError(response);
    const status = await response.json() as { received: number[] };
    received = new Set(status.received);
  } else {
    const response = await authedFetch(buildApiUrl("/transfer/uploads", { baseUrl: REST_API_BASE }), {
      method: "POST", cache: "no-store", signal,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: file.name, mime: file.type || "application/octet-stream", size_bytes: file.size, retention: RETENTION_FORM_VALUE[retention] || "held", target_device_id: targetDeviceId }),
    });
    if (!response.ok) throw await decodeApiError(response);
    const status = await response.json() as { id: string; received: number[] };
    id = status.id; received = new Set(status.received); onSession?.(id);
  }
  const count = Math.ceil(file.size / RESUMABLE_CHUNK_BYTES);
  for (let index = 0; index < count; index += 1) {
    if (received.has(index)) { onProgress?.(Math.min(1, ((index + 1) * RESUMABLE_CHUNK_BYTES) / file.size)); continue; }
    await putChunk(id!, index, file.slice(index * RESUMABLE_CHUNK_BYTES, Math.min(file.size, (index + 1) * RESUMABLE_CHUNK_BYTES)), (fraction) => onProgress?.((index * RESUMABLE_CHUNK_BYTES + fraction * Math.min(RESUMABLE_CHUNK_BYTES, file.size - index * RESUMABLE_CHUNK_BYTES)) / file.size), signal);
    onProgress?.(Math.min(1, ((index + 1) * RESUMABLE_CHUNK_BYTES) / file.size));
  }
  const response = await authedFetch(buildApiUrl(`/transfer/uploads/${encodeURIComponent(id!)}/complete`, { baseUrl: REST_API_BASE }), { method: "POST", cache: "no-store", signal });
  if (!response.ok) throw await decodeApiError(response);
  const decoded = fromJson(UploadItemResponseSchema, await response.json() as JsonValue, PROTO_READ_OPTIONS);
  if (!decoded.item) throw makeApiError("internal", "upload returned no item");
  return decoded.item;
}

function putChunk(sessionId: string, index: number, body: Blob, onProgress: (fraction: number) => void, signal?: AbortSignal): Promise<void> {
  const { deviceToken } = readSessionCredentials();
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest(); xhr.open("PUT", buildApiUrl(`/transfer/uploads/${encodeURIComponent(sessionId)}/chunks/${index}`, { baseUrl: REST_API_BASE }));
    if (deviceToken) xhr.setRequestHeader(DEVICE_TOKEN_HEADER, deviceToken);
    xhr.upload.onprogress = (event) => { if (event.lengthComputable && event.total > 0) onProgress(event.loaded / event.total); };
    xhr.onload = () => { if (xhr.status >= 200 && xhr.status < 300) resolve(); else void decodeApiError(new Response(xhr.responseText, { status: xhr.status })).then(reject); };
    xhr.onerror = () => reject(makeApiError("unavailable", "network error during upload", 0)); xhr.onabort = () => reject(makeApiError("canceled", "upload canceled", 0));
    if (signal) signal.addEventListener("abort", () => xhr.abort(), { once: true }); xhr.send(body);
  });
}

/**
 * Fetch an item's bytes through the device-token-authed download endpoint.
 * A plain `<a href>` can't set the `X-Device-Token` header, so we fetch the
 * blob ourselves. `thumb` requests the server-generated thumbnail variant.
 */
export async function fetchItemBlob(itemId: string, opts: { thumb?: boolean } = {}): Promise<Blob> {
  const path = `/transfer/items/${encodeURIComponent(itemId)}/content${opts.thumb ? "?thumb=1" : ""}`;
  const res = await authedFetch(buildApiUrl(path, { baseUrl: REST_API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  return res.blob();
}

/**
 * Download an item to the user's device: fetch the blob with the device token,
 * then synthesise a temporary object URL + `<a download>` click so the browser
 * saves it under the original filename.
 */
export async function downloadItem(itemId: string, filename: string): Promise<void> {
  const blob = await fetchItemBlob(itemId);
  const url = URL.createObjectURL(blob);
  try {
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = filename || "download";
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}

export { ItemKind, Retention };
export type { Item };
