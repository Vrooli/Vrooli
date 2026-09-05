import {
  create,
  fromJson,
  type JsonValue,
} from "@bufbuild/protobuf";
import { resolveApiBase, buildApiUrl, createScenarioConnectTransport } from "@vrooli/api-base";
import {
  ErrorEnvelopeSchema,
  type ErrorEnvelope,
} from "@vrooli/proto-types/vrooli-bridge/v1/errors/errors_pb";

import { notifySessionExpired, readOwnerToken, restoreLocalSession } from "../features/session/store";

export const API_BASE = resolveApiBase();
export const REST_API_BASE = resolveApiBase({ appendSuffix: true });
const PROTO_READ_OPTIONS = { ignoreUnknownFields: true } as const;

/**
 * `@vrooli/api-base` has no built-in auth interceptor. `authedFetch` reads the
 * ephemeral LocalSession fresh from the session store on every request and
 * attaches it as `Authorization: LocalSession`. Enrollment metadata and the
 * signing key persist locally; provider bearer tokens do not. The one-time
 * enrollment RPC uses an explicit in-memory bootstrap token.
 */
export async function authedFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const ownerToken = readOwnerToken() ?? (await restoreLocalSession())?.ownerToken ?? null;
  const headers = new Headers(init?.headers);
  if (ownerToken && !headers.has("Authorization")) {
    const scheme = ownerToken.startsWith("OS1.") ? "LocalSession" : "Bearer";
    headers.set("Authorization", `${scheme} ${ownerToken}`);
  }
  const res = await fetch(input, { ...init, headers });
  // A 401 on a request that carried our token means the token is expired or
  // revoked — tell the session so the gate returns to sign-in instead of
  // leaving a shell where every panel errors. IdentityService calls are
  // excluded: their 401 means "wrong password on this sign-in attempt", which
  // must not sign out an existing session.
  if (res.status === 401 && ownerToken && !isIdentityCall(input)) {
    notifySessionExpired();
  }
  return res;
}

function isIdentityCall(input: RequestInfo | URL): boolean {
  const url = typeof input === "object" && "url" in input ? input.url : String(input);
  return url.includes(".identity.IdentityService/");
}

/**
 * Connect transport every RPC client is built against. The custom `fetch`
 * attaches the enrollment-based local credential when present; the server
 * enforces which RPCs require it.
 */
export const transport = createScenarioConnectTransport({
  baseUrl: API_BASE,
  fetch: authedFetch,
});

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
 * Resolve a downloadable URL for a run artifact reference. Artifact bytes live
 * in device-sync-hub (DATA.md) and never transit the control-plane store; the
 * REST gateway streams them by ref, so the dashboard links directly to this URL
 * rather than buffering the bytes through Connect.
 */
export function artifactDownloadUrl(ref: string): string {
  return buildApiUrl(`/artifacts/${encodeURIComponent(ref)}/download`, {
    baseUrl: REST_API_BASE,
  });
}

export { fromJson, PROTO_READ_OPTIONS };
export type { ErrorEnvelope, JsonValue };
