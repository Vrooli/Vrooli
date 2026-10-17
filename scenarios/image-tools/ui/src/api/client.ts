import {
  create,
  fromJson,
  type JsonValue,
} from "@bufbuild/protobuf";
import { resolveApiBase, buildApiUrl, createScenarioConnectTransport } from "@vrooli/api-base";
import {
  ErrorEnvelopeSchema,
  type ErrorEnvelope,
} from "@vrooli/proto-types/image-tools/v1/shared/errors_pb";

export const API_BASE = resolveApiBase();
const REST_API_BASE = resolveApiBase({ appendSuffix: true });
const PROTO_READ_OPTIONS = { ignoreUnknownFields: true } as const;

export const transport = createScenarioConnectTransport({ baseUrl: API_BASE });

/**
 * Typed error thrown when the API returns a non-2xx response. The
 * server-side ErrorEnvelope round-trips through here so callers branch on
 * structured code/status instead of parsing strings.
 */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(envelope: ErrorEnvelope, status: number) {
    super(`${envelope.code}: ${envelope.message}`);
    this.name = "ApiError";
    this.code = envelope.code;
    this.status = status;
  }
}

export function makeApiError(code: string, message: string, status = 500): ApiError {
  const envelope = create(ErrorEnvelopeSchema, { code, message });
  return new ApiError(envelope, status);
}

export async function decodeApiError(res: Response): Promise<ApiError> {
  let envelope: ErrorEnvelope;
  try {
    const json = (await res.json()) as JsonValue;
    envelope = fromJson(ErrorEnvelopeSchema, json, PROTO_READ_OPTIONS);
  } catch {
    envelope = create(ErrorEnvelopeSchema, {
      code: "internal",
      message: `unexpected ${res.status} response (no envelope)`,
    });
  }
  return new ApiError(envelope, res.status);
}

export async function uploadFile(path: string, formData: FormData): Promise<Response> {
  return fetch(buildApiUrl(path, { baseUrl: REST_API_BASE }), {
    method: "POST",
    body: formData,
    cache: "no-store",
  });
}

/**
 * Fetch a managed result blob by its key (a job's `result_ref`, e.g.
 * `out/<id>.png`), served by `GET /api/v1/blobs/{key}`. Used to pull an async
 * AI op's output image once its job is terminal. Throws a typed `ApiError` on
 * a non-2xx response so callers surface a real error state instead of an
 * `<img>` that silently fails to load.
 */
export async function fetchBlob(key: string): Promise<Blob> {
  const res = await fetch(buildApiUrl(`/blobs/${key}`, { baseUrl: REST_API_BASE }), {
    cache: "no-store",
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  return res.blob();
}

/**
 * URL for a managed result blob by its key (`GET /api/v1/blobs/{key}`), for use
 * directly as an `<img src>` (Library thumbnails, Activity result previews).
 * Use `fetchBlob` instead when you need the bytes (e.g. to reopen as a File).
 */
export function blobUrl(key: string): string {
  return buildApiUrl(`/blobs/${key}`, { baseUrl: REST_API_BASE });
}

export { fromJson, PROTO_READ_OPTIONS };
export type { ErrorEnvelope, JsonValue };
