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

const API_BASE = resolveApiBase({ appendSuffix: true });
const PROTO_READ_OPTIONS = { ignoreUnknownFields: true } as const;
const PROTO_WRITE_OPTIONS = { useProtoFieldName: true } as const;

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

export interface ProtoFetchOptions<
  ReqDesc extends DescMessage | undefined,
  RespDesc extends DescMessage,
> {
  requestSchema?: ReqDesc;
  request?: ReqDesc extends DescMessage ? MessageInitShape<ReqDesc> : never;
  responseSchema: RespDesc;
}

/**
 * Single proto-typed fetch helper. `src/api/client.ts` is the only
 * production UI module that calls raw fetch; domain clients in
 * `src/api/<domain>.ts` pass generated request/response descriptors.
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
    init.body = toJsonString(reqSchema, reqMsg, PROTO_WRITE_OPTIONS);
  }
  const res = await fetch(url, init);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  return fromJson(opts.responseSchema, json, PROTO_READ_OPTIONS);
}

export type { ErrorEnvelope };

