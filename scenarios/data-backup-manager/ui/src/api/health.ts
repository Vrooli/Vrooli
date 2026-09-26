import { fromJson } from "@bufbuild/protobuf";
import { buildApiUrl } from "@vrooli/api-base";
import { ResponseSchema } from "@vrooli/proto-types/data-backup-manager/v1/health/health_pb";
import type { Response } from "@vrooli/proto-types/data-backup-manager/v1/health/health_pb";

import { API_BASE, PROTO_READ_OPTIONS, decodeApiError, parseJsonValue } from "./client";

type HealthResponse = Response;

export async function fetchHealth(): Promise<HealthResponse> {
  const res = await fetch(buildApiUrl("/health", { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  return fromJson(ResponseSchema, parseJsonValue(await res.text()), PROTO_READ_OPTIONS);
}

export type { HealthResponse };
