import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { beforeEach, test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { KPISummary } from "../../src/features/stats/components/kpi/KPISummary.js";
import { KPICard } from "../../src/features/stats/components/kpi/KPICard.js";
import type { TimePreset } from "../../src/features/stats/api/types.js";
import { useTimeWindow } from "../../src/features/stats/hooks/useTimeWindow.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
const measures = vi.hoisted(() => ({ success: vi.fn(), cost: vi.fn(), duration: vi.fn(), volume: vi.fn(), statuses: vi.fn() }));
vi.mock("../../src/features/stats/api/statsClient.js", () => ({
  fetchDurableRunSuccess: measures.success, fetchDurableRunCost: measures.cost, fetchDurableRunCycleTime: measures.duration, fetchDurableRunVolume: measures.volume, fetchDurableRunStatusDistribution: measures.statuses,
  statsQueryKeys: { successRate: (f: unknown) => ["success", f], cost: (f: unknown) => ["cost", f], duration: (f: unknown) => ["duration", f], summary: (f: unknown) => ["summary", f], statusDistribution: (f: unknown) => ["statuses", f] },
}));

vi.mock("../../src/features/stats/hooks/useTimeWindow.js", () => ({
  useTimeWindow: vi.fn(),
}));

type QueryResult = ReturnType<typeof useQuery<unknown, Error>>;

const presetOptions: readonly TimePreset[] = ["6h", "12h", "24h", "7d", "30d"];

function queryResult(overrides: Partial<QueryResult>): QueryResult {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as QueryResult;
}

beforeEach(() => {
	measures.success.mockResolvedValue({ rate: 0.875, executedQuery: "SELECT" }); measures.cost.mockResolvedValue({ totalCostUsd: 12.35, executedQuery: "SELECT" }); measures.duration.mockResolvedValue({ rate: 90_000, executedQuery: "SELECT" }); measures.volume.mockResolvedValue({ totalRuns: 24, executedQuery: "SELECT" }); measures.statuses.mockResolvedValue({ rows: [{ status: "pending", count: 2 }, { status: "starting", count: 1 }], executedQuery: "SELECT" });
  vi.mocked(useTimeWindow).mockReturnValue({
    preset: "24h",
    setPreset: vi.fn(),
    filter: { preset: "24h" },
    presetOptions,
  });
});

test("KPISummary renders formatted stats and queue totals", async () => {
  renderWithProviders(createElement(KPISummary));

  await waitFor(() => assert.ok(screen.getByText("Success Rate")));
  assert.ok(screen.getByText("87.5%"));
  assert.ok(screen.getByText("Total Cost"));
  assert.ok(screen.getByText("$12.35"));
  assert.ok(screen.getByText("Avg Duration"));
  assert.ok(screen.getByText("1.5m"));
  assert.ok(screen.getByText("Throughput"));
  assert.ok(screen.getByText("1.0/hr"));
  assert.ok(screen.getByText("Queue"));
  assert.ok(screen.getByText("3"));
});

test("KPISummary uses the selected stats window when calculating throughput", async () => {
  vi.mocked(useTimeWindow).mockReturnValue({
    preset: "6h",
    setPreset: vi.fn(),
    filter: { preset: "6h" },
    presetOptions,
  });
  measures.volume.mockResolvedValue({ totalRuns: 12, executedQuery: "SELECT" });

  renderWithProviders(createElement(KPISummary));

  await waitFor(() => assert.ok(screen.getByText("2.0/hr")));
});

test("KPISummary renders loading and error states through metric cards", async () => {
  measures.success.mockReturnValue(new Promise(() => {}));

  const { unmount } = renderWithProviders(createElement(KPISummary));

  assert.equal(screen.queryByText("Success Rate"), null);
  assert.equal(document.querySelectorAll(".animate-pulse").length, 5);

  unmount();
  measures.success.mockRejectedValue(new Error("stats unavailable"));

  renderWithProviders(createElement(KPISummary));

  await waitFor(() => assert.equal(screen.getAllByText("Error loading").length, 5));
});

test("KPICard communicates directional and neutral trends, visual variants, and its loading/error fallbacks", () => {
  const { rerender } = renderWithProviders(createElement(KPICard, {
    title: "Completed", value: "12", trend: 4.25, trendLabel: "vs prior", variant: "success",
    icon: createElement("span", null, "icon"),
  }));
  assert.ok(screen.getByText("4.3%"));
  assert.ok(screen.getByText("vs prior"));
  assert.ok(screen.getByText("icon"));
  assert.match(screen.getByText("Completed").closest("div.rounded-lg")?.className ?? "", /emerald/);

  rerender(createElement(KPICard, { title: "Failures", value: "2", trend: -1, variant: "error" }));
  assert.ok(screen.getByText("1.0%"));
  assert.match(screen.getByText("Failures").closest("div.rounded-lg")?.className ?? "", /red/);

  rerender(createElement(KPICard, { title: "Queue", value: "0", trend: 0, variant: "warning" }));
  assert.ok(screen.getByText("0.0%"));
  assert.match(screen.getByText("Queue").closest("div.rounded-lg")?.className ?? "", /amber/);

  rerender(createElement(KPICard, { title: "Loading", value: "", loading: true }));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 1);
  rerender(createElement(KPICard, { title: "Broken", value: "", error: "unavailable" }));
  assert.ok(screen.getByText("Error loading"));
});
