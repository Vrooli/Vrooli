import { ResponseSchema } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";
import type { Response as HealthResponse } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";

import { protoFetch } from "./client";

export async function fetchHealth(): Promise<HealthResponse> {
  return protoFetch("GET", "/health", { responseSchema: ResponseSchema });
}

export type { HealthResponse };

