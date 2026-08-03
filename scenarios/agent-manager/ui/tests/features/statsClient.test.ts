import assert from "node:assert/strict";
import { afterEach, test, vi } from "vitest";
import { timestampDate } from "@bufbuild/protobuf/wkt";
const measures = vi.hoisted(() => ({
  externalToolShare: vi.fn(async () => ({ share: 0.2, externalCalls: 2n, resolvedCalls: 10n, unknownCalls: 1n, executedQuery: "SELECT durable" })),
  retryRate: vi.fn(async () => ({ rate: 0.2, retryCalls: 2n, totalCalls: 10n, executedQuery: "SELECT durable" })),
  helpRecoveryRate: vi.fn(async () => ({ rate: 0.2, helpRecoveries: 2n, totalCalls: 10n, executedQuery: "SELECT durable" })),
  repeatedWorkRate: vi.fn(async () => ({ rate: 0.2, repeatedCalls: 2n, totalCalls: 10n, executedQuery: "SELECT durable" })),
  fileRereadRate: vi.fn(async () => ({ rate: 0.2, filesReadMoreThanOnce: 2n, readCalls: 10n, executedQuery: "SELECT durable" })),
  findingRecurrenceRate: vi.fn(async () => ({ rate: 0.2, recurringFindings: 2n, totalFindings: 10n, recurringFingerprints: 1n, executedQuery: "SELECT durable" })),
  errorPatterns: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  runnerBreakdown: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  profileBreakdown: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  modelBreakdown: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  terminalRunTrend: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  toolUsage: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  runCost: vi.fn(async () => ({ totalCostUsd: 1, averageCostUsd: 1, totalRuns: 1n, totalTokens: 2n, inputTokens: 1n, outputTokens: 1n, cacheReadTokens: 0n, cacheCreationTokens: 0n, inputCostUsd: 0.5, outputCostUsd: 0.5, cacheReadCostUsd: 0, cacheCreationCostUsd: 0, authoritativeCostUsd: 1, estimatedCostUsd: 0, unknownCostUsd: 0, executedQuery: "SELECT durable" })),
  runSuccessRate: vi.fn(async () => ({ rate: 1, executedQuery: "SELECT durable" })),
  runCycleTime: vi.fn(async () => ({ averageDurationMs: 1, executedQuery: "SELECT durable" })),
  runDurationStatistics: vi.fn(async () => ({ averageDurationMs: 1, p50DurationMs: 1, p95DurationMs: 1, p99DurationMs: 1, minDurationMs: 1n, maxDurationMs: 1n, count: 1n, executedQuery: "SELECT durable" })),
  runVolume: vi.fn(async () => ({ totalRuns: 1n, terminalRuns: 1n, executedQuery: "SELECT durable" })),
  runStatusDistribution: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  selectCohort: vi.fn(async () => ({ runIds: [], truncated: false, executedQuery: "SELECT durable" })),
  workloadBreakdown: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  toolCommandBreakdown: vi.fn(async () => ({ rows: [], executedQuery: "SELECT durable" })),
  allMeasureDefinitions: vi.fn(async () => ({ definitions: [] })),
}));
vi.mock("../../src/features/stats/api/measuresClient.js", () => ({ measuresClient: measures }));
import {
  fetchDurableModelBreakdown,
  fetchDurableModelCohort,
  fetchDurableProfileBreakdown,
  fetchDurableRunCost,
  fetchDurableRunCycleTime,
  fetchDurableRunDurationStatistics,
  fetchDurableRunStatusDistribution,
  fetchDurableRunSuccess,
  fetchDurableRunVolume,
  fetchDurableRunnerBreakdown,
  fetchDurableTerminalTrend,
  fetchDurableToolCohort,
  fetchDurableToolModels,
  fetchDurableToolUsage,
  fetchDurableToolCommands,
  fetchDurableWorkloadBreakdown,
  fetchMeasureDefinitions,
  fetchExternalToolShare,
  fetchFileRereadRate,
  fetchFindingRecurrenceRate,
  fetchHelpRecoveryRate,
  fetchModelCostComparison,
  fetchRepeatedWorkRate,
  fetchRetryRate,
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

test("durable stats adapters send the same typed filter to each analytics question", async () => {
  await Promise.all([
    fetchDurableRunnerBreakdown(filter), fetchDurableProfileBreakdown(filter), fetchDurableModelBreakdown(filter),
    fetchDurableTerminalTrend(filter), fetchDurableToolUsage(filter), fetchDurableRunCost(filter), fetchDurableRunSuccess(filter),
    fetchDurableRunCycleTime(filter), fetchDurableRunDurationStatistics(filter), fetchDurableRunVolume(filter), fetchDurableRunStatusDistribution(filter),
    fetchDurableToolCohort(filter, "workspace-sandbox", 11), fetchDurableModelCohort(filter, "gpt-5", 12), fetchDurableToolModels(filter, "workspace-sandbox"),
    fetchDurableWorkloadBreakdown(filter), fetchDurableToolCommands(filter, "workspace-sandbox", 9), fetchMeasureDefinitions(),
  ]);
  for (const method of [measures.runnerBreakdown, measures.profileBreakdown, measures.modelBreakdown, measures.terminalRunTrend, measures.toolUsage, measures.runCost, measures.runSuccessRate, measures.runCycleTime, measures.runDurationStatistics, measures.runVolume, measures.runStatusDistribution]) {
    const request = method.mock.calls[0]?.[0];
    assert.equal(timestampDate(request.window.window.value.from).toISOString(), new Date(filter.start).toISOString());
    assert.equal(request.filter.runnerType, "codex");
    assert.equal(request.filter.profileId, "profile/a");
    assert.equal(request.filter.model, "gpt-5");
    assert.equal(request.filter.tagPrefix, "investigation");
  }
  assert.equal(measures.selectCohort.mock.calls[0]?.[0]?.limit, 11);
  assert.equal(measures.selectCohort.mock.calls[0]?.[0]?.filter.toolName, "workspace-sandbox");
  assert.equal(measures.selectCohort.mock.calls[1]?.[0]?.limit, 12);
  assert.equal(measures.selectCohort.mock.calls[1]?.[0]?.filter.model, "gpt-5");
  assert.equal(measures.workloadBreakdown.mock.calls[0]?.[0]?.filter.workloadKey, "");
  assert.equal(measures.toolCommandBreakdown.mock.calls[0]?.[0]?.limit, 9);
  assert.equal(measures.toolCommandBreakdown.mock.calls[0]?.[0]?.filter.toolName, "workspace-sandbox");
  assert.equal(measures.allMeasureDefinitions.mock.calls.length, 1);
});

test("durable measure requests support every preset and leave omitted dimensions unfiltered", async () => {
  measures.runnerBreakdown.mockClear();
  for (const preset of ["6h", "12h", "24h", "7d", "30d"] as const) {
    await fetchDurableRunnerBreakdown({ preset, end: "2026-07-02T00:00:00Z" });
  }
  await fetchDurableRunnerBreakdown({ end: "2026-07-02T00:00:00Z" });
  await fetchDurableRunnerBreakdown({ start: "2026-07-01T00:00:00Z", end: "2026-07-02T00:00:00Z" });

  assert.equal(measures.runnerBreakdown.mock.calls.length, 7);
  for (const [request] of measures.runnerBreakdown.mock.calls.slice(0, 6)) {
    assert.equal(request.window.window.case, "custom");
    assert.equal(request.filter.profileId, "");
    assert.equal(request.filter.runnerType, "");
    assert.equal(request.filter.model, "");
    assert.equal(request.filter.tagPrefix, "");
  }
  assert.equal(
    timestampDate(measures.runnerBreakdown.mock.calls[6]?.[0]?.window.window.value.from).toISOString(),
    "2026-07-01T00:00:00.000Z",
  );
});

test("stats client posts model comparisons and exposes API failures", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(new Response("backend down", { status: 503 }))
    .mockResolvedValueOnce(new Response(JSON.stringify({ comparisons: [] }), { status: 200 }));
  vi.stubGlobal("fetch", fetch);

  await assert.rejects(fetchModelCostComparison({ modelList: ["gpt-5"], actualModel: "gpt-5", inputTokens: 10, outputTokens: 20, cacheReadTokens: 0, cacheCreationTokens: 0 }), /API error 503: backend down/);
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

test("typed friction measures use one explicit Connect procedure per question", async () => {
  const window = { window: { custom: { from: "2026-07-01T00:00:00Z", to: "2026-07-02T00:00:00Z" } } };

  await Promise.all([fetchExternalToolShare(window), fetchRetryRate(window), fetchHelpRecoveryRate(window), fetchRepeatedWorkRate(window), fetchFileRereadRate(window), fetchFindingRecurrenceRate(window)]);

  for (const method of [measures.externalToolShare, measures.retryRate, measures.helpRecoveryRate, measures.repeatedWorkRate, measures.fileRereadRate, measures.findingRecurrenceRate]) {
    assert.equal(method.mock.calls.length, 1);
    const request = method.mock.calls[0]?.[0];
    assert.equal(request.window.window.case, "custom");
    assert.equal(timestampDate(request.window.window.value.from).toISOString(), new Date(window.window.custom.from).toISOString());
    assert.equal(timestampDate(request.window.window.value.to).toISOString(), new Date(window.window.custom.to).toISOString());
  }
});

test("typed friction measures reject malformed custom-window timestamps before invoking Connect", async () => {
  measures.externalToolShare.mockClear();
  await assert.rejects(
    fetchExternalToolShare({ window: { custom: { from: "not-a-timestamp", to: "2026-07-02T00:00:00Z" } } }),
    /Invalid measure timestamp: not-a-timestamp/,
  );
  assert.equal(measures.externalToolShare.mock.calls.length, 0);
});

test("durable adapters preserve validity, provenance, optional cohort fields, and charge bases", async () => {
  const provenance = { sourceTable: "agent_runs", windowStart: "from", windowEnd: "to", rowCount: 2n, appliedFilters: [{ field: "model", value: "gpt-5" }] };
  for (const state of ["available", "unreliable", "unavailable", "unexpected"]) {
    measures.runnerBreakdown.mockResolvedValueOnce({ rows: [], validity: { state, reason: "fixture", sampleSize: 2n, largestFingerprintShare: 0.1 }, provenance, definitionId: "throughput.runner_breakdown", executedQuery: "SELECT" });
    const result = await fetchDurableRunnerBreakdown(filter);
    assert.equal(result.validity.state, state === "unexpected" ? "unavailable" : state);
    assert.equal(result.provenance?.appliedFilters[0]?.field, "model");
  }

  measures.selectCohort.mockResolvedValueOnce({
    runIds: ["run-1", "run-2"], truncated: false, executedQuery: "SELECT", definitionId: "throughput.cohort",
    rows: [
      { runId: "run-1", taskTitle: "", profileId: "p", profileName: "Profile", status: "complete", createdAt: "", model: "gpt-5", runnerType: "codex", workloadKey: "build", totalTokens: 5n, totalChargeMicroUsd: 7n, chargeBasis: "metered", toolCallCount: 2n },
      { runId: "run-2", totalTokens: 0n },
    ],
    validity: { state: "available", reason: "fixture", sampleSize: 2n, largestFingerprintShare: 0 }, provenance,
  });
  const cohort = await fetchDurableToolCohort(filter, "shell", 2);
  assert.equal(cohort.rows[0]?.taskTitle, undefined);
  assert.equal(cohort.rows[0]?.totalChargeMicroUsd, 7);
  assert.equal(cohort.rows[1]?.totalChargeMicroUsd, undefined);

  measures.runCost.mockResolvedValueOnce({ totalCostUsd: 1, averageCostUsd: 1, totalRuns: 2n, totalTokens: 5n, inputTokens: 2n, outputTokens: 3n, cacheReadTokens: 0n, cacheCreationTokens: 0n, inputCostUsd: 0, outputCostUsd: 0, cacheReadCostUsd: 0, cacheCreationCostUsd: 0, totalChargeMicroUsd: 7n, unpricedTokenCount: 0n, chargeByBasis: [{ basis: "metered", runCount: 2n, chargeMicroUsd: 7n, tokenCount: 5n, chargeReason: "priced" }], validity: { state: "available", reason: "fixture", sampleSize: 2n, largestFingerprintShare: 0 }, provenance, definitionId: "throughput.run_cost", executedQuery: "SELECT" });
  const cost = await fetchDurableRunCost(filter);
  assert.deepEqual(cost.chargeByBasis, [{ basis: "metered", runCount: 2, chargeMicroUsd: 7, tokenCount: 5, chargeReason: "priced" }]);
});
