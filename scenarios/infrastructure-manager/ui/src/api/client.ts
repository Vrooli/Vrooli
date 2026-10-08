import {
  create,
  fromJson,
  type JsonValue,
} from "@bufbuild/protobuf";
import { resolveApiBase, buildApiUrl, createScenarioConnectTransport } from "@vrooli/api-base";
import {
  ErrorEnvelopeSchema,
  type ErrorEnvelope,
} from "@vrooli/proto-types/infrastructure-manager/v1/shared/errors_pb";

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

export { fromJson, PROTO_READ_OPTIONS };
export type { ErrorEnvelope, JsonValue };
