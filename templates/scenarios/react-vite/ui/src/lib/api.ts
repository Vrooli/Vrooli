import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import { ResponseSchema } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";
import type { Response as HealthResponse } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";

// Specify whether to append the /api/v1 suffix; true for versioned routes.
const API_BASE = resolveApiBase({ appendSuffix: true });

/**
 * Fetch the API health endpoint.
 *
 * The wire shape lives in `packages/proto/schemas/{{SCENARIO_ID}}/v1/health/health.proto`
 * and is consumed here via the generated `ResponseSchema` descriptor.
 * `fromJson` accepts both snake_case (proto names like `uptime_seconds`,
 * `latency_ms`) and lowerCamelCase (`uptimeSeconds`, `latencyMs`) by
 * default — matching what the api-core/health JSON encoder emits — so
 * the wire and the runtime type agree without translation.
 *
 * `ignoreUnknownFields: true` mirrors the interop-steer guidance: the
 * UI keeps decoding successfully when the wire grows fields the proto
 * hasn't caught up to. The reverse (failing every time api-core adds a
 * field) would force unrelated UI churn whenever the wire grows.
 *
 * Test code mocks this function via `vi.mock("./lib/api", ...)`. See
 * `ui/src/App.test.tsx` for the canonical pattern and
 * `ui/src/test-utils/factories.ts::makeHealthResponse` for typed
 * fixture construction.
 */
export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  const json = (await res.json()) as JsonValue;
  return fromJson(ResponseSchema, json, { ignoreUnknownFields: true });
}

export type { HealthResponse };
