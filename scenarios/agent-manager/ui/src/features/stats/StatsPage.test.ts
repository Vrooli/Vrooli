import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { StatsPage } from "./StatsPage.js";
import { renderWithProviders } from "@vrooli/api-base/testing";

const volume = vi.hoisted(() => vi.fn());
vi.mock("./api/statsClient.js", () => ({
  fetchDurableRunVolume: volume,
  statsQueryKeys: {
    summary: (filter: unknown) => ["stats", "summary", filter],
    tokenAttribution: (filter: unknown, groupBy: unknown, view: unknown, limit: unknown) => ["stats", "tokenAttribution", filter, groupBy, view, limit],
  },
}));
vi.mock("./components/controls/TimeWindowSelector.js", () => ({ TimeWindowSelector: () => createElement("div", null, "time window") }));
vi.mock("./components/controls/ExportButton.js", () => ({ ExportButton: () => createElement("div", null, "export") }));
vi.mock("./components/kpi/KPISummary.js", () => ({ KPISummary: () => createElement("div", null, "kpi") }));
vi.mock("./components/trends/RunStatusTrends.js", () => ({ RunStatusTrends: () => createElement("div", null, "status trends") }));
vi.mock("./components/trends/CostDurationTrends.js", () => ({ CostDurationTrends: () => createElement("div", null, "cost trends") }));
vi.mock("./components/tables/RunnerPerformanceTable.js", () => ({ RunnerPerformanceTable: () => createElement("div", null, "runners") }));
vi.mock("./components/tables/ProfileActivityTable.js", () => ({ ProfileActivityTable: () => createElement("div", null, "profiles") }));
vi.mock("./components/breakdown/ModelUsageBreakdown.js", () => ({ ModelUsageBreakdown: () => createElement("div", null, "models") }));
vi.mock("./components/breakdown/CostDistributionCard.js", () => ({ CostDistributionCard: () => createElement("div", null, "cost distribution") }));
vi.mock("./components/breakdown/ToolUsageAnalytics.js", () => ({ ToolUsageAnalytics: () => createElement("div", null, "tools") }));
vi.mock("./components/errors/ErrorAnalysisSection.js", () => ({ ErrorAnalysisSection: () => createElement("div", null, "errors") }));
vi.mock("./components/workload/RecurringWorkloadPanel.js", () => ({ RecurringWorkloadPanel: () => createElement("div", null, "workloads") }));
vi.mock("./components/operational/FallbackInsightsCard.js", () => ({ FallbackInsightsCard: () => createElement("div", null, "fallback") }));
vi.mock("./components/operational/ModelFailureAlertBanner.js", () => ({ ModelFailureAlertBanner: () => createElement("div", null, "failures") }));
vi.mock("./components/operational/FrictionOverviewCard.js", () => ({ FrictionOverviewCard: () => createElement("div", null, "friction") }));
vi.mock("../../components/stats/HistoryBanner.js", () => ({ HistoryBanner: ({ testId }: { testId?: string }) => createElement("div", { "data-testid": testId }, "history") }));

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
