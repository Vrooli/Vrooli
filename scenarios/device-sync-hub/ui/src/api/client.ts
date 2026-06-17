/**
 * Public API surface for REST + Connect plumbing.
 *
 * The dual-credential transport, the `ApiError` envelope type, and the
 * device-token-aware `fetch` live in `./transport`. This module re-exports the
 * stable pieces feature code imports, plus the generic device-token-aware
 * multipart `uploadFile` helper used by the file-send path.
 */
import {
  ApiError,
  authedFetch,
  buildApiUrl,
  decodeApiError,
  fromJson,
  makeApiError,
  PROTO_READ_OPTIONS,
  REST_API_BASE,
  transport,
  API_BASE,
} from "./transport";

/**
 * POST a multipart form to a REST edge. Unlike a bare `fetch`, this rides the
 * dual-credential `authedFetch` so the `X-Device-Token` header (required by the
 * transfer upload endpoint) is attached automatically. Returns the raw Response
 * so callers can branch on `res.ok` and decode the typed envelope on failure.
 */
export async function uploadFile(path: string, formData: FormData): Promise<Response> {
  return authedFetch(buildApiUrl(path, { baseUrl: REST_API_BASE }), {
    method: "POST",
    body: formData,
    cache: "no-store",
  });
}

export {
  ApiError,
  authedFetch,
  buildApiUrl,
  decodeApiError,
  fromJson,
  makeApiError,
  PROTO_READ_OPTIONS,
  REST_API_BASE,
  transport,
  API_BASE,
};
export type { ErrorEnvelope, JsonValue } from "./transport";
