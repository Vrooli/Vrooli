import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { StatsPage } from "../../src/features/stats/StatsPage.js";
import { renderWithProviders } from "../../src/test-utils/renderWithProviders.js";

const volume = vi.hoisted(() => vi.fn());
vi.mock("../../src/features/stats/api/statsClient.js", () => ({
  fetchDurableRunVolume: volume,
  statsQueryKeys: {
    summary: (filter: unknown) => ["stats", "summary", filter],
    tokenAttribution: (filter: unknown, groupBy: unknown, view: unknown, limit: unknown) => ["stats", "tokenAttribution", filter, groupBy, view, limit],
  },
}));
vi.mock("../../src/features/stats/components/controls/TimeWindowSelector.js", () => ({ TimeWindowSelector: () => createElement("div", null, "time window") }));
vi.mock("../../src/features/stats/components/controls/ExportButton.js", () => ({ ExportButton: () => createElement("div", null, "export") }));
vi.mock("../../src/features/stats/components/kpi/KPISummary.js", () => ({ KPISummary: () => createElement("div", null, "kpi") }));
vi.mock("../../src/features/stats/components/trends/RunStatusTrends.js", () => ({ RunStatusTrends: () => createElement("div", null, "status trends") }));
vi.mock("../../src/features/stats/components/trends/CostDurationTrends.js", () => ({ CostDurationTrends: () => createElement("div", null, "cost trends") }));
vi.mock("../../src/features/stats/components/tables/RunnerPerformanceTable.js", () => ({ RunnerPerformanceTable: () => createElement("div", null, "runners") }));
vi.mock("../../src/features/stats/components/tables/ProfileActivityTable.js", () => ({ ProfileActivityTable: () => createElement("div", null, "profiles") }));
vi.mock("../../src/features/stats/components/breakdown/ModelUsageBreakdown.js", () => ({ ModelUsageBreakdown: () => createElement("div", null, "models") }));
vi.mock("../../src/features/stats/components/breakdown/ToolUsageAnalytics.js", () => ({ ToolUsageAnalytics: () => createElement("div", null, "tools") }));
vi.mock("../../src/features/stats/components/errors/ErrorAnalysisSection.js", () => ({ ErrorAnalysisSection: () => createElement("div", null, "errors") }));
vi.mock("../../src/features/stats/components/workload/RecurringWorkloadPanel.js", () => ({ RecurringWorkloadPanel: () => createElement("div", null, "workloads") }));
vi.mock("../../src/features/stats/components/operational/FallbackInsightsCard.js", () => ({ FallbackInsightsCard: () => createElement("div", null, "fallback") }));
vi.mock("../../src/features/stats/components/operational/ModelFailureAlertBanner.js", () => ({ ModelFailureAlertBanner: () => createElement("div", null, "failures") }));
vi.mock("../../src/features/stats/components/operational/FrictionOverviewCard.js", () => ({ FrictionOverviewCard: () => createElement("div", null, "friction") }));
vi.mock("../../src/components/stats/HistoryBanner.js", () => ({ HistoryBanner: ({ testId }: { testId?: string }) => createElement("div", { "data-testid": testId }, "history") }));

afterEach(() => vi.clearAllMocks());

test("stats page displays the read-model history banner when coverage metadata exists", async () => {
  volume.mockResolvedValue({ historyFloor: "2026-01-01T00:00:00Z", outsideHistoryRunCount: 4 });
  renderWithProviders(createElement(StatsPage));
  await waitFor(() => assert.equal(screen.getByTestId("stats-history-banner").textContent, "history"));
  assert.ok(screen.getByText("Statistics & Analytics"));
});

test("stats page omits the history banner when the measure has no coverage floor", async () => {
  volume.mockResolvedValue({ historyFloor: "", outsideHistoryRunCount: 0 });
  renderWithProviders(createElement(StatsPage));
  await waitFor(() => assert.ok(screen.getByText("Statistics & Analytics")));
  assert.equal(screen.queryByTestId("stats-history-banner"), null);
});
