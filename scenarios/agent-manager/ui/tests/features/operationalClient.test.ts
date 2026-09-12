import assert from "node:assert/strict";
import { afterEach, test, vi } from "vitest";
import {
  fetchFallbackInsights,
  fetchHealthSummary,
  operationalQueryKeys,
} from "../../src/features/stats/api/operationalClient.js";

afterEach(() => vi.unstubAllGlobals());

test("operational stats client requests the bounded fallback and health resources", async () => {
  const fetch = vi.fn(async () => new Response(JSON.stringify({ generated_at: "2026-07-01T00:00:00Z" }), { status: 200 }));
  vi.stubGlobal("fetch", fetch);

  await fetchFallbackInsights();
  await fetchHealthSummary();

  assert.equal(fetch.mock.calls[0]?.[0], "/api/v1/stats/fallback");
  assert.equal(fetch.mock.calls[1]?.[0], "/api/v1/stats/operational?category=health");
  assert.deepEqual(fetch.mock.calls[0]?.[1], { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  assert.deepEqual(operationalQueryKeys.fallback(), ["operational-stats", "fallback"]);
  assert.deepEqual(operationalQueryKeys.health(), ["operational-stats", "health"]);
});

test("operational stats client includes status and response text in failures", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response("unavailable", { status: 502 })));
  await assert.rejects(fetchFallbackInsights(), /Operational stats API error 502: unavailable/);
});
