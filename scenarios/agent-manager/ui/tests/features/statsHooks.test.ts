import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { afterEach, test, vi } from "vitest";
import { createTestQueryClient } from "../../src/test-utils/renderWithProviders.js";
import { useCostTrends } from "../../src/features/stats/hooks/useCostTrends.js";
import { useErrorAnalysis } from "../../src/features/stats/hooks/useErrorAnalysis.js";
import { useModelBreakdown, useModelUsageRuns } from "../../src/features/stats/hooks/useModelBreakdown.js";
import { useRunTrends } from "../../src/features/stats/hooks/useRunTrends.js";
import { useRunnerPerformance } from "../../src/features/stats/hooks/useRunnerPerformance.js";
import { useStatsSummary } from "../../src/features/stats/hooks/useStatsSummary.js";
import { getPresetLabel, getPresetShortLabel, TimeWindowProvider, useStatsFilter, useTimeWindow } from "../../src/features/stats/hooks/useTimeWindow.js";
import { useToolUsage, useToolUsageModels, useToolUsageRuns } from "../../src/features/stats/hooks/useToolUsage.js";

const client = vi.hoisted(() => ({
  cost: vi.fn(async () => ({ value: "cost" })), errors: vi.fn(async () => ({ value: "errors" })),
  models: vi.fn(async () => ({ value: "models" })), modelRuns: vi.fn(async () => ({ value: "model-runs" })),
  timeSeries: vi.fn(async () => ({ value: "trends" })), runners: vi.fn(async () => ({ value: "runners" })),
  summary: vi.fn(async () => ({ value: "summary" })), tools: vi.fn(async () => ({ value: "tools" })),
  toolModels: vi.fn(async () => ({ value: "tool-models" })), toolRuns: vi.fn(async () => ({ value: "tool-runs" })),
}));
const key = (name: string, ...values: unknown[]) => [name, ...values];
vi.mock("../../src/features/stats/api/statsClient.js", () => ({
  fetchCostStats: client.cost, fetchErrorPatterns: client.errors, fetchModelBreakdown: client.models,
  fetchModelUsageRuns: client.modelRuns, fetchTimeSeries: client.timeSeries, fetchRunnerBreakdown: client.runners,
  fetchStatsSummary: client.summary, fetchToolUsage: client.tools, fetchToolUsageModels: client.toolModels,
  fetchToolUsageRuns: client.toolRuns,
  statsQueryKeys: { cost: (f: unknown) => key("cost", f), errors: (f: unknown, l: unknown) => key("errors", f, l), models: (f: unknown, l: unknown) => key("models", f, l), modelRuns: (f: unknown, l: unknown) => key("model-runs", f, l), timeSeries: (f: unknown, b: unknown) => key("trends", f, b), runners: (f: unknown) => key("runners", f), summary: (f: unknown) => key("summary", f), tools: (f: unknown, l: unknown) => key("tools", f, l), toolModels: (f: unknown, n: unknown, l: unknown) => key("tool-models", f, n, l), toolRuns: (f: unknown, n: unknown, l: unknown) => key("tool-runs", f, n, l) },
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
  ];
  await waitFor(() => hooks.forEach(({ result }) => assert.equal(result.current.isSuccess, true)));
  assert.deepEqual(client.errors.mock.calls[0], [filter, 4]);
  assert.deepEqual(client.models.mock.calls[0], [filter, 3]);
  assert.deepEqual(client.modelRuns.mock.calls[0], [{ ...filter, model: "gpt-5" }, 2]);
  assert.deepEqual(client.timeSeries.mock.calls[0], [filter, "hour"]);
  assert.deepEqual(client.tools.mock.calls[0], [filter, 6]);
  assert.deepEqual(client.toolModels.mock.calls[0], [filter, "bash", 2]);
  assert.deepEqual(client.toolRuns.mock.calls[0], [filter, "bash", 2]);
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
  assert.equal(getPresetShortLabel("12h"), "12h");
  assert.equal(getPresetLabel("custom" as never), "custom");
  assert.equal(getPresetShortLabel("custom" as never), "custom");
});
