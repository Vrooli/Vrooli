import { type JsonValue, fromJson } from "@bufbuild/protobuf";
import { buildApiUrl } from "@vrooli/api-base";
import { ResponseSchema } from "@vrooli/proto-types/data-backup-manager/v1/health/health_pb";
import type { Response as HealthResponse } from "@vrooli/proto-types/data-backup-manager/v1/health/health_pb";

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
