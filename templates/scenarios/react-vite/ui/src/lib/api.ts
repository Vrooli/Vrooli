import {
  create,
  fromJson,
  toJsonString,
  type DescMessage,
  type JsonValue,
  type MessageInitShape,
  type MessageShape,
} from "@bufbuild/protobuf";
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import {
  ErrorEnvelopeSchema,
  type ErrorEnvelope,
} from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/errors/errors_pb";
import { ResponseSchema } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";
import type { Response as HealthResponse } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";

// Specify whether to append the /api/v1 suffix; true for versioned routes.
const API_BASE = resolveApiBase({ appendSuffix: true });

/**
 * Typed error thrown when the API returns a non-2xx response. The
 * server-side error envelope (proto: ErrorEnvelope) round-trips through
 * here so callers see a structured `code` + `message` rather than a
 * raw HTTP status.
 *
 * Lives in `lib/api.ts` (not a per-domain file) so every UI client
 * reads the same shape — fetchHealth and listNotes throw the same
 * type, so call-site error handling doesn't fork by domain.
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

/**
 * Build a synthetic ApiError without a network round-trip. Use when
 * client-side validation surfaces a failure that's structurally
 * indistinguishable from a server envelope (e.g., a 2xx response that
 * omits a required field).
 */
export function makeApiError(code: string, message: string, status = 500): ApiError {
  const envelope = create(ErrorEnvelopeSchema, { code, message });
  return new ApiError(envelope, status);
}

/**
 * Build an ApiError from a non-2xx response. Returns the error rather
 * than throwing it so callers express the throw at the call site
 * (`throw await decodeApiError(res)`) — a `Promise<never>` returner
 * reads as if the call site is doing nothing, when in fact every
 * caller relies on the thrown error to short-circuit decoding.
 */
export async function decodeApiError(res: Response): Promise<ApiError> {
  let envelope: ErrorEnvelope;
  try {
    const json = (await res.json()) as JsonValue;
    envelope = fromJson(ErrorEnvelopeSchema, json, { ignoreUnknownFields: true });
  } catch {
    envelope = create(ErrorEnvelopeSchema, {
      code: "internal",
      message: `unexpected ${res.status} response (no envelope)`,
    });
  }
  return new ApiError(envelope, res.status);
}

/**
 * Options for protoFetch. requestSchema + request together encode the
 * outbound proto body; responseSchema decodes the inbound JSON.
 *
 * For GET-style calls, omit requestSchema/request.
 */
export interface ProtoFetchOptions<
  ReqDesc extends DescMessage | undefined,
  RespDesc extends DescMessage,
> {
  requestSchema?: ReqDesc;
  request?: ReqDesc extends DescMessage ? MessageInitShape<ReqDesc> : never;
  responseSchema: RespDesc;
}

/**
 * Single proto-typed fetch helper. Replaces the per-domain
 * `if (!res.ok) throw await decodeApiError(res); fromJson(...)` ribbon
 * that previously lived in lib/notes.ts × N domains.
 *
 * On non-2xx, throws ApiError carrying the typed envelope. On 2xx,
 * returns the decoded response message. Caller then guards optional
 * fields ("if (!resp.note) ...") since proto3 makes everything optional.
 */
export async function protoFetch<
  RespDesc extends DescMessage,
  ReqDesc extends DescMessage | undefined = undefined,
>(
  method: string,
  path: string,
  opts: ProtoFetchOptions<ReqDesc, RespDesc>,
): Promise<MessageShape<RespDesc>> {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const init: RequestInit = {
    method,
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  };
  if (opts.requestSchema && opts.request !== undefined) {
    const reqSchema: DescMessage = opts.requestSchema;
    const reqMsg = create(reqSchema, opts.request as MessageInitShape<DescMessage>);
    init.body = toJsonString(reqSchema, reqMsg);
  }
  const res = await fetch(url, init);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  return fromJson(opts.responseSchema, json, { ignoreUnknownFields: true });
}

/**
 * Fetch the API health endpoint. Re-implemented on top of protoFetch so
 * health and every domain client share one error path.
 *
 * Test code mocks this function via `vi.mock("./lib/api", ...)`. See
 * `ui/src/App.test.tsx` for the canonical pattern and
 * `ui/src/test-utils/factories.ts::makeHealthResponse` for typed
 * fixture construction.
 */
export async function fetchHealth(): Promise<HealthResponse> {
  return protoFetch("GET", "/health", { responseSchema: ResponseSchema });
}

export type { HealthResponse, ErrorEnvelope };
