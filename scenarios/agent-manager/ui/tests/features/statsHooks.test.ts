import assert from "node:assert/strict";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { afterEach, test, vi } from "vitest";
import { createTestQueryClient } from "@vrooli/api-base/testing";
import { useCostTrends } from "../../src/features/stats/hooks/useCostTrends.js";
import { useErrorAnalysis } from "../../src/features/stats/hooks/useErrorAnalysis.js";
import { useModelBreakdown, useModelUsageRuns } from "../../src/features/stats/hooks/useModelBreakdown.js";
import { useRunTrends } from "../../src/features/stats/hooks/useRunTrends.js";
import { useRunnerPerformance } from "../../src/features/stats/hooks/useRunnerPerformance.js";
import { useProfileBreakdown } from "../../src/features/stats/hooks/useProfileBreakdown.js";
import { useStatsSummary } from "../../src/features/stats/hooks/useStatsSummary.js";
import { getPresetLabel, getPresetShortLabel, TimeWindowProvider, useStatsFilter, useTimeWindow } from "../../src/features/stats/hooks/useTimeWindow.js";
import { useToolUsage, useToolUsageModels, useToolUsageRuns } from "../../src/features/stats/hooks/useToolUsage.js";
import { TimeWindowSelector } from "../../src/features/stats/components/controls/TimeWindowSelector.js";
import { renderWithProviders } from "@vrooli/api-base/testing";

const client = vi.hoisted(() => ({
  durableCost: vi.fn(async () => ({ totalCostUsd: 0, averageCostUsd: 0, totalRuns: 0, totalTokens: 0, inputTokens: 0, outputTokens: 0, cacheReadTokens: 0, cacheCreationTokens: 0, inputCostUsd: 0, outputCostUsd: 0, cacheReadCostUsd: 0, cacheCreationCostUsd: 0, executedQuery: "SELECT" })), errors: vi.fn(async () => ({ value: "errors" })),
  models: vi.fn(async () => ({ value: "models" })), durableModels: vi.fn(async () => ({ rows: [], executedQuery: "SELECT" })),
  profiles: vi.fn(async () => ({ rows: [], executedQuery: "SELECT" })), durableModelCohort: vi.fn(async () => ({ rows: [], truncated: false, executedQuery: "SELECT" })),
  timeSeries: vi.fn(async () => ({ value: "trends" })), durableTrends: vi.fn(async () => ({ rows: [], executedQuery: "SELECT" })),
  runners: vi.fn(async () => ({ value: "runners" })), durableRunners: vi.fn(async () => ({ rows: [], executedQuery: "SELECT" })),
  summary: vi.fn(async () => ({ value: "summary" })), durableTools: vi.fn(async () => ({ rows: [], executedQuery: "SELECT" })),
  durableToolModels: vi.fn(async () => ({ rows: [], executedQuery: "SELECT" })), durableToolCohort: vi.fn(async () => ({ rows: [], truncated: false, executedQuery: "SELECT" })),
  durableSuccess: vi.fn(async () => ({ rate: 0, executedQuery: "SELECT" })), durableDuration: vi.fn(async () => ({ averageDurationMs: 0, p50DurationMs: 0, p95DurationMs: 0, p99DurationMs: 0, minDurationMs: 0, maxDurationMs: 0, count: 0, executedQuery: "SELECT" })),
  durableStatus: vi.fn(async () => ({ rows: [], executedQuery: "SELECT" })),
  definitions: vi.fn(async () => []),
}));
const key = (name: string, ...values: unknown[]) => [name, ...values];
vi.mock("../../src/features/stats/api/statsClient.js", () => ({
  fetchDurableRunCost: client.durableCost, fetchDurableErrorPatterns: client.errors, fetchModelBreakdown: client.models, fetchDurableModelBreakdown: client.durableModels,
  fetchDurableProfileBreakdown: client.profiles,
  fetchDurableModelCohort: client.durableModelCohort, fetchTimeSeries: client.timeSeries, fetchDurableTerminalTrend: client.durableTrends,
  fetchRunnerBreakdown: client.runners, fetchDurableRunnerBreakdown: client.durableRunners,
  fetchStatsSummary: client.summary, fetchDurableRunSuccess: client.durableSuccess, fetchDurableRunDurationStatistics: client.durableDuration, fetchDurableRunStatusDistribution: client.durableStatus, fetchDurableToolUsage: client.durableTools, fetchDurableToolModels: client.durableToolModels,
  fetchDurableToolCohort: client.durableToolCohort,
  fetchMeasureDefinitions: client.definitions,
  statsQueryKeys: { cost: (f: unknown) => key("cost", f), errors: (f: unknown, l: unknown) => key("errors", f, l), models: (f: unknown, l: unknown) => key("models", f, l), profiles: (f: unknown, l: unknown) => key("profiles", f, l), modelRuns: (f: unknown, l: unknown) => key("model-runs", f, l), timeSeries: (f: unknown, b: unknown) => key("trends", f, b), runners: (f: unknown) => key("runners", f), summary: (f: unknown) => key("summary", f), tools: (f: unknown, l: unknown) => key("tools", f, l), toolModels: (f: unknown, n: unknown, l: unknown) => key("tool-models", f, n, l), toolRuns: (f: unknown, n: unknown, l: unknown) => key("tool-runs", f, n, l) },
}));

afterEach(() => vi.clearAllMocks());
function wrapper({ children }: { children: ReactNode }) { return createElement(QueryClientProvider, { client: createTestQueryClient() }, children); }
const filter = { preset: "7d" as const };

test("stats query hooks fetch the intended analytics with explicit filters and limits", async () => {
  const hooks = [
    renderHook(() => useCostTrends({ filter }), { wrapper }), renderHook(() => useErrorAnalysis({ filter, limit: 4 }), { wrapper }),
    renderHook(() => useModelBreakdown({ filter, limit: 3 }), { wrapper }), renderHook(() => useModelUsageRuns({ filter, model: "gpt-5", limit: 2 }), { wrapper }),
    renderHook(() => useRunTrends({ filter, bucket: "hour" }), { wrapper }), renderHook(() => useRunnerPerformance({ filter }), { wrapper }),
    renderHook(() => useStatsSummary({ filter }), { wrapper }), renderHook(() => useToolUsage({ filter, limit: 6 }), { wrapper }),
    renderHook(() => useToolUsageModels({ filter, toolName: "bash", limit: 2 }), { wrapper }), renderHook(() => useToolUsageRuns({ filter, toolName: "bash", limit: 2 }), { wrapper }),
    renderHook(() => useProfileBreakdown({ filter, limit: 4 }), { wrapper }),
  ];
  await waitFor(() => hooks.forEach(({ result }) => assert.equal(result.current.isSuccess, true)));
  assert.equal(client.errors.mock.calls[0]?.[0]?.window?.custom?.from.endsWith("Z"), true);
  assert.equal(client.errors.mock.calls[0]?.[0]?.window?.custom?.to.endsWith("Z"), true);
  assert.deepEqual(client.durableModels.mock.calls[0], [filter]);
	assert.deepEqual(client.durableCost.mock.calls[0], [filter]);
  assert.deepEqual(client.durableSuccess.mock.calls[0], [filter]);
  assert.deepEqual(client.durableDuration.mock.calls[0], [filter]);
  assert.deepEqual(client.durableStatus.mock.calls[0], [filter]);
  assert.deepEqual(client.durableModelCohort.mock.calls[0], [filter, "gpt-5", 2]);
  assert.deepEqual(client.durableTrends.mock.calls[0], [filter]);
  assert.deepEqual(client.durableRunners.mock.calls[0], [filter]);
  assert.deepEqual(client.durableTools.mock.calls[0], [filter]);
  assert.deepEqual(client.durableToolModels.mock.calls[0], [filter, "bash"]);
  assert.deepEqual(client.durableToolCohort.mock.calls[0], [filter, "bash", 2]);
  assert.deepEqual(client.profiles.mock.calls[0], [filter]);
});

test("optional model and tool detail hooks remain disabled until an operator selects an item", () => {
  const model = renderHook(() => useModelUsageRuns({ filter }), { wrapper });
  const toolRuns = renderHook(() => useToolUsageRuns({ filter }), { wrapper });
  const toolModels = renderHook(() => useToolUsageModels({ filter }), { wrapper });
  assert.equal(model.result.current.fetchStatus, "idle");
  assert.equal(toolRuns.result.current.fetchStatus, "idle");
  assert.equal(toolModels.result.current.fetchStatus, "idle");
});

test("time-window state shares presets, merges filters, and labels all supported choices", async () => {
  const time = renderHook(() => ({ window: useTimeWindow(), filter: useStatsFilter({ runnerType: "codex" }) }), { wrapper: ({ children }) => createElement(TimeWindowProvider, { defaultPreset: "6h" }, children) });
  await waitFor(() => assert.equal(time.result.current.window.preset, "6h"));
  act(() => { time.result.current.window.setPreset("30d"); });
  await waitFor(() => assert.deepEqual(time.result.current.filter, { preset: "30d", runnerType: "codex" }));
  assert.equal(getPresetLabel("7d"), "Last 7 days");
  assert.equal(getPresetLabel("6h"), "Last 6 hours");
  assert.equal(getPresetLabel("24h"), "Last 24 hours");
  assert.equal(getPresetLabel("30d"), "Last 30 days");
  assert.equal(getPresetShortLabel("12h"), "12h");
  assert.equal(getPresetShortLabel("6h"), "6h");
  assert.equal(getPresetShortLabel("24h"), "24h");
  assert.equal(getPresetShortLabel("30d"), "30d");
  assert.equal(getPresetLabel("custom" as never), "custom");
  assert.equal(getPresetShortLabel("custom" as never), "custom");
});

test("time-window selector marks the active option and changes the shared window", async () => {
  renderWithProviders(createElement(TimeWindowProvider, { defaultPreset: "7d" }, createElement(TimeWindowSelector)));
  assert.equal(screen.getByRole("button", { name: "7d" }).getAttribute("aria-pressed"), "true");
  fireEvent.click(screen.getByRole("button", { name: "6h" }));
  await waitFor(() => assert.equal(screen.getByRole("button", { name: "6h" }).getAttribute("aria-pressed"), "true"));
  fireEvent.click(screen.getByRole("button", { name: "6h" }));
});
