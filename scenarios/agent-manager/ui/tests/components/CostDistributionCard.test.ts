import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { beforeEach, test, vi } from "vitest";
import { CostDistributionCard } from "../../src/features/stats/components/breakdown/CostDistributionCard.js";
import type { TimePreset } from "../../src/features/stats/api/types.js";
import { useTimeWindow } from "../../src/features/stats/hooks/useTimeWindow.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const cost = vi.hoisted(() => vi.fn());
vi.mock("../../src/features/stats/api/statsClient.js", () => ({
  fetchDurableRunCost: cost,
  fetchMeasureDefinitions: vi.fn(async () => []),
  statsQueryKeys: { cost: (f: unknown) => ["cost", f] },
}));

vi.mock("../../src/features/stats/hooks/useTimeWindow.js", () => ({
  useTimeWindow: vi.fn(),
}));

const presetOptions: readonly TimePreset[] = ["6h", "12h", "24h", "7d", "30d"];
const validMeasure = { validity: { state: "available", reason: "fixture", sampleSize: 512, largestFingerprintShare: 0 }, executedQuery: "SELECT", definitionId: "throughput.run_cost" } as const;

const distribution = {
  totalCostUsd: 126.5,
  averageCostUsd: 0.23,
  totalRuns: 551,
  totalTokens: 4_105_504_396,
  inputTokens: 198_353_285,
  outputTokens: 12_777_939,
  cacheReadTokens: 3_872_247_788,
  cacheCreationTokens: 22_125_384,
  inputCostUsd: 0,
  outputCostUsd: 0,
  cacheReadCostUsd: 0,
  cacheCreationCostUsd: 0,
  totalChargeMicroUsd: 126_508_762,
  unpricedTokenCount: 0,
  chargeByBasis: [],
  p50Tokens: 1_236_010,
  p90Tokens: 15_397_815,
  p95Tokens: 38_370_977,
  p99Tokens: 121_389_138,
  maxTokens: 281_839_369,
  p50CostUsd: 0,
  p90CostUsd: 1.488852,
  p95CostUsd: 1.91577,
  p99CostUsd: 2.665219,
  maxCostUsd: 3.30845,
  tokenBuckets: [
    { label: "<100K", minTokens: 0, maxTokens: 100_000, runCount: 81 },
    { label: "25M+", minTokens: 25_000_000, maxTokens: 0, runCount: 44 },
  ],
  distributionSampleSize: 512,
  distributionUnobservedRuns: 39,
  ...validMeasure,
};

beforeEach(() => {
  cost.mockReset();
  cost.mockResolvedValue(distribution);
  vi.mocked(useTimeWindow).mockReturnValue({
    preset: "7d",
    setPreset: vi.fn(),
    filter: { preset: "7d" },
    presetOptions,
  });
});

test("CostDistributionCard renders token and cost percentiles with an explicit denominator", async () => {
  renderWithProviders(createElement(CostDistributionCard));

  await waitFor(() => assert.ok(screen.getByText("Per-run token & cost distribution")));
  assert.ok(screen.getByText("512 runs with observed usage · 39 uncounted"));
  assert.equal(screen.getAllByText("P50").length, 2);
  assert.equal(screen.getAllByText("Max").length, 2);
  assert.ok(screen.getByText("1.24M"));
  assert.ok(screen.getByText("15.40M"));
  assert.ok(screen.getByText("38.37M"));
  assert.ok(screen.getByText("121.39M"));
  assert.ok(screen.getByText("281.84M"));
  assert.ok(screen.getByText("$0.0000"));
  assert.ok(screen.getByText("$1.4889"));
  assert.ok(screen.getByText("$1.9158"));
  assert.ok(screen.getByText("$2.6652"));
  assert.ok(screen.getByText("$3.3085"));
});

test("CostDistributionCard shows an empty state when no run has observed usage", async () => {
  cost.mockResolvedValue({ ...distribution, distributionSampleSize: 0, tokenBuckets: [] });
  renderWithProviders(createElement(CostDistributionCard));

  await waitFor(() => assert.ok(screen.getByText("No runs with observed token usage in the selected window")));
  assert.equal(screen.queryByText("Tokens per run"), null);
});
