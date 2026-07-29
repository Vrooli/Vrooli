import assert from "node:assert/strict";
import { afterEach, test, vi } from "vitest";
import {
  fetchCostStats,
  fetchDurationStats,
  fetchErrorPatterns,
  fetchModelBreakdown,
  fetchModelCostComparison,
  fetchModelUsageRuns,
  fetchProfileBreakdown,
  fetchRunnerBreakdown,
  fetchStatsSummary,
  fetchStatusDistribution,
  fetchSuccessRate,
  fetchTimeSeries,
  fetchToolUsage,
  fetchToolUsageModels,
  fetchToolUsageRuns,
  statsQueryKeys,
} from "../../src/features/stats/api/statsClient.js";

afterEach(() => vi.unstubAllGlobals());

const filter = {
  preset: "24h" as const,
  start: "2026-07-01T00:00:00Z",
  end: "2026-07-02T00:00:00Z",
  runnerType: "codex",
  profileId: "profile/a",
  model: "gpt-5",
  tagPrefix: "investigation",
};

test("stats client sends every filter and endpoint-specific limit in GET requests", async () => {
  const fetch = vi.fn(async () => new Response(JSON.stringify({ ok: true }), { status: 200 }));
  vi.stubGlobal("fetch", fetch);

  await fetchStatsSummary(filter);
  await fetchStatusDistribution(filter);
  await fetchSuccessRate(filter);
  await fetchDurationStats(filter);
  await fetchCostStats(filter);
  await fetchRunnerBreakdown(filter);
  await fetchProfileBreakdown(filter, 7);
  await fetchModelBreakdown(filter, 8);
  await fetchModelUsageRuns(filter, 9);
  await fetchToolUsage(filter, 10);
  await fetchToolUsageModels(filter, "workspace-sandbox", 11);
  await fetchToolUsageRuns(filter, "workspace-sandbox", 12);
  await fetchErrorPatterns(filter, 13);
  await fetchTimeSeries(filter, "1h");

  assert.equal(fetch.mock.calls.length, 14);
  const urls = fetch.mock.calls.map(([url]) => String(url));
  assert.match(urls[0]!, /^\/api\/v1\/stats\/summary\?/);
  assert.match(urls[0]!, /preset=24h/);
  assert.match(urls[0]!, /profile_id=profile%2Fa/);
  assert.match(urls[6]!, /\/profiles\?.*limit=7/);
  assert.match(urls[7]!, /\/models\?.*limit=8/);
  assert.match(urls[8]!, /\/models\/runs\?.*limit=9/);
  assert.match(urls[9]!, /\/tools\?.*limit=10/);
  assert.match(urls[10]!, /limit=11/);
  assert.match(urls[10]!, /tool_name=workspace-sandbox/);
  assert.match(urls[11]!, /limit=12/);
  assert.match(urls[11]!, /tool_name=workspace-sandbox/);
  assert.match(urls[12]!, /\/errors\?.*limit=13/);
  assert.match(urls[13]!, /\/time-series\?.*bucket=1h/);
});

test("stats client exposes API failures and posts model comparisons", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(new Response("backend down", { status: 503 }))
    .mockResolvedValueOnce(new Response(JSON.stringify({ comparisons: [] }), { status: 200 }));
  vi.stubGlobal("fetch", fetch);

  await assert.rejects(fetchStatsSummary({}), /API error 503: backend down/);
  await fetchModelCostComparison({ modelList: ["gpt-5"], actualModel: "gpt-5", inputTokens: 10, outputTokens: 20, cacheReadTokens: 0, cacheCreationTokens: 0 });

  assert.deepEqual(fetch.mock.calls[1]?.[0], "/api/v1/pricing/compare");
  assert.deepEqual(fetch.mock.calls[1]?.[1], {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ modelList: ["gpt-5"], actualModel: "gpt-5", inputTokens: 10, outputTokens: 20, cacheReadTokens: 0, cacheCreationTokens: 0 }),
  });
});

test("stats query keys partition requests by their meaningful inputs", () => {
  assert.deepEqual(statsQueryKeys.summary(filter), ["stats", "summary", filter]);
  assert.deepEqual(statsQueryKeys.toolModels(filter, "bash", 25), ["stats", "toolModels", filter, "bash", 25]);
  assert.deepEqual(statsQueryKeys.timeSeries(filter, "1d"), ["stats", "timeSeries", filter, "1d"]);
});

test("stats query keys cover every analytics surface and preserve optional time buckets", () => {
  assert.deepEqual(statsQueryKeys.all, ["stats"]);
  assert.deepEqual(statsQueryKeys.statusDistribution(filter), ["stats", "statusDistribution", filter]);
  assert.deepEqual(statsQueryKeys.successRate(filter), ["stats", "successRate", filter]);
  assert.deepEqual(statsQueryKeys.duration(filter), ["stats", "duration", filter]);
  assert.deepEqual(statsQueryKeys.cost(filter), ["stats", "cost", filter]);
  assert.deepEqual(statsQueryKeys.runners(filter), ["stats", "runners", filter]);
  assert.deepEqual(statsQueryKeys.profiles(filter, 4), ["stats", "profiles", filter, 4]);
  assert.deepEqual(statsQueryKeys.models(filter, 5), ["stats", "models", filter, 5]);
  assert.deepEqual(statsQueryKeys.modelRuns(filter, 6), ["stats", "modelRuns", filter, 6]);
  assert.deepEqual(statsQueryKeys.tools(filter, 7), ["stats", "tools", filter, 7]);
  assert.deepEqual(statsQueryKeys.toolRuns(filter, "shell", 8), ["stats", "toolRuns", filter, "shell", 8]);
  assert.deepEqual(statsQueryKeys.errors(filter, 9), ["stats", "errors", filter, 9]);
  assert.deepEqual(statsQueryKeys.timeSeries(filter), ["stats", "timeSeries", filter, undefined]);
  assert.deepEqual(statsQueryKeys.modelCostComparison({ modelList: ["gpt-5"], actualModel: "gpt-5", inputTokens: 1, outputTokens: 2, cacheReadTokens: 3, cacheCreationTokens: 4 }), ["pricing", "compare", ["gpt-5"], "gpt-5", 1, 2, 3, 4]);
});
