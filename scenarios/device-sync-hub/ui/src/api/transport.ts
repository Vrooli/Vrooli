import {
  create,
  fromJson,
  type JsonValue,
} from "@bufbuild/protobuf";
import { resolveApiBase, buildApiUrl, createScenarioConnectTransport } from "@vrooli/api-base";
import {
  ErrorEnvelopeSchema,
  type ErrorEnvelope,
} from "@vrooli/proto-types/device-sync-hub/v1/errors/errors_pb";

import { readSessionCredentials } from "../features/session/store";

export const API_BASE = resolveApiBase();
export const REST_API_BASE = resolveApiBase({ appendSuffix: true });
const PROTO_READ_OPTIONS = { ignoreUnknownFields: true } as const;

/**
 * The hub has two independent credentials and no built-in auth interceptor in
 * `@vrooli/api-base`:
 *
 *   - Device token (`X-Device-Token`) — this browser's membership in the trust
 *     group; required by the transfer RPCs + the realtime SSE stream.
 *   - Owner JWT (`Authorization: Bearer`) — required by the owner-gated devices
 *     RPCs.
 *
 * `authedFetch` reads BOTH fresh from the session store on every request and
 * attaches whichever is present. The server reads the header the RPC needs, so
 * sending both when present is correct. We read fresh per call (not at
 * transport-construction time) so a pairing/sign-in mid-session takes effect
 * without rebuilding the transport.
 */
export const DEVICE_TOKEN_HEADER = "X-Device-Token";

export function authedFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const { deviceToken, ownerToken } = readSessionCredentials();
  const headers = new Headers(init?.headers);
  if (deviceToken && !headers.has(DEVICE_TOKEN_HEADER)) {
    headers.set(DEVICE_TOKEN_HEADER, deviceToken);
  }
  if (ownerToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${ownerToken}`);
  }
  return fetch(input, { ...init, headers });
}

/**
 * Connect transport every RPC client is built against. The custom `fetch`
 * attaches the dual credentials; the server enforces which one each RPC needs.
 */
export const transport = createScenarioConnectTransport({
  baseUrl: API_BASE,
  fetch: authedFetch,
});

/**
 * Typed error thrown when a REST edge returns a non-2xx response. The
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

export { fromJson, PROTO_READ_OPTIONS, buildApiUrl };
export type { ErrorEnvelope, JsonValue };
