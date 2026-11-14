// Sanctioned REST exception: RESTReasonOpsProbe.
//
// The /health endpoint is the liveness probe for the API binary itself
// and must answer before Connect-RPC routing is wired up. Per the
// react-vite template (see api/internal/module/module.go::RESTReasonOpsProbe
// and packages/proto/schemas/web-console/v1/health/health.proto), this is
// one of the four enumerated allowed REST shapes. See
// docs/internal/SEAMS.md for the full REST-exception registry.
//
// The proto carries the JSON wire shape so the response decodes through
// fromJson(ResponseSchema, ...) for type safety; no hand-rolled types.

import { type JsonValue, fromJson } from "@bufbuild/protobuf";
import { buildApiUrl } from "@vrooli/api-base";
import {
  ResponseSchema,
  type Response as HealthResponse,
} from "@vrooli/proto-types/web-console/v1/health/health_pb";

import { API_BASE, PROTO_READ_OPTIONS, decodeApiError } from "./client";

export async function fetchHealth(): Promise<HealthResponse> {
  const res = await fetch(buildApiUrl("/health", { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  return fromJson(ResponseSchema, (await res.json()) as JsonValue, PROTO_READ_OPTIONS);
}

export type { HealthResponse };
