import assert from "node:assert/strict";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { RecurringWorkloadPanel } from "../../src/features/stats/components/workload/RecurringWorkloadPanel.js";
import { renderWithProviders } from "@vrooli/api-base/testing";

const mocks = vi.hoisted(() => ({
  workload: vi.fn(),
  definitions: vi.fn(),
}));

vi.mock("../../src/features/stats/api/statsClient.js", () => ({
  fetchDurableWorkloadBreakdown: mocks.workload,
  statsQueryKeys: { workload: (filter: unknown, limit: number) => ["workload", filter, limit] },
}));
vi.mock("../../src/features/stats/hooks/useMeasureDefinitions.js", () => ({ useMeasureDefinitions: mocks.definitions }));
vi.mock("../../src/features/stats/hooks/useTimeWindow.js", () => ({ useTimeWindow: () => ({ filter: { preset: "7d" } }) }));

const evidence = { state: "available" as const, reason: "enough evidence", sampleSize: 5, largestFingerprintShare: 0 };

afterEach(() => {
  vi.clearAllMocks();
});

test("recurring workloads shows linked workload evidence and successful consumption", async () => {
  mocks.definitions.mockReturnValue({ data: [{ id: "throughput.workload_breakdown", counts: "runs", numerator: "success", denominator: "runs", sourceTable: "agent_runs", limitation: "observational" }] });
  mocks.workload.mockResolvedValue({
    rows: [{ key: "build", value: "Build", runCount: 3, successCount: 2, failedCount: 1, totalCostUsd: 0, totalTokens: 1200, totalChargeMicroUsd: 0, averageDurationMs: 1, consumptionPerSuccessfulCompletion: 600, completionRate: 2 / 3 }],
    validity: evidence,
    definitionId: "throughput.workload_breakdown",
    executedQuery: "SELECT",
  });
  renderWithProviders(createElement(RecurringWorkloadPanel));
  await waitFor(() => assert.equal(screen.getByTestId("recurring-workload-panel").textContent?.includes("Build"), true));
  assert.equal(screen.getByRole("link", { name: "Build" }).getAttribute("href")?.includes("workloadKey=build"), true);
  assert.equal(screen.getByText("600").textContent, "600");
});

test("recurring workloads distinguishes empty and unsuccessful workload evidence", async () => {
  mocks.definitions.mockReturnValue({ data: undefined });
  mocks.workload.mockResolvedValue({ rows: [], validity: evidence, definitionId: "throughput.workload_breakdown", executedQuery: "SELECT" });
  renderWithProviders(createElement(RecurringWorkloadPanel));
  await waitFor(() => assert.equal(screen.getByText("No workload keys recorded in this window.").textContent, "No workload keys recorded in this window."));

  cleanup();

  mocks.workload.mockResolvedValue({
    rows: [{ key: "review", value: "", runCount: 1, successCount: 0, failedCount: 1, totalCostUsd: 0, totalTokens: 2, totalChargeMicroUsd: 0, averageDurationMs: 1, consumptionPerSuccessfulCompletion: 0, completionRate: 0 }],
    validity: evidence,
    definitionId: "throughput.workload_breakdown",
    executedQuery: "SELECT",
  });
  renderWithProviders(createElement(RecurringWorkloadPanel));
  await waitFor(() => assert.equal(screen.getByText("No successful completion").textContent, "No successful completion"));
});
