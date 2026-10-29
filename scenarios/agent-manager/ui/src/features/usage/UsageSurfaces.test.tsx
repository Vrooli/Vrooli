import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import { ConnectError, Code } from "@connectrpc/connect";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { DashboardUsageStrip } from "./DashboardUsageStrip.js";
import { RunUsagePanel } from "./RunUsagePanel.js";
import { renderWithProviders } from "../../test-utils/index.js";
import { BURNER_RUN_ID, makeUsageBurners, makeUsageForecast, makeUsageRecommendations } from "../../test-utils/fixtures/usage.js";

const NO_COST = { hasCost: false, costLoading: false };

const measures = vi.hoisted(() => ({ usageForecast: vi.fn(), usageBurners: vi.fn(), usageRecommendations: vi.fn() }));
vi.mock("../stats/api/measuresClient.js", () => ({ measuresClient: measures }));

afterEach(() => Object.values(measures).forEach((fn) => fn.mockReset()));

test("the dashboard strip shows one line per window: at-risk with run-out and fixes, no-reading quietly, all linking to Usage", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast());
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());
  renderWithProviders(createElement(DashboardUsageStrip));
  const lines = await screen.findAllByTestId("dashboard-usage-line");
  assert.equal(lines.length, 2);
  const [codex, claude] = lines as [HTMLElement, HTMLElement];
  assert.equal(codex.getAttribute("href"), "/usage");
  assert.match(codex.textContent ?? "", /Will run out.*Codex · weekly.*33%.*Runs out [A-Z][a-z]{2} \d{1,2}.* · resets [A-Z][a-z]{2} \d{1,2}/);
  // Chip, name, bar and percent share the first line; the RCL bar's 12rem minimum is lifted so it fits 390 px.
  const firstLine = codex.firstElementChild as HTMLElement;
  assert.match(firstLine.textContent ?? "", /Will run out.*Codex · weekly.*33%/);
  const bar = firstLine.querySelector("[data-rcl-progress]") as HTMLElement;
  assert.match(bar.style.minWidth, /^0(px)?$/);
  await waitFor(() => assert.match(codex.textContent ?? "", /2 recommendations →/));
  assert.match(claude.textContent ?? "", /No reading.*Readings appear after the next reply/);
});

test("a window with no reading since its reset is one muted line in the strip, not a card", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ dormant: true }));
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());
  renderWithProviders(createElement(DashboardUsageStrip));
  const dormant = await screen.findByTestId("usage-window-dormant");
  assert.match(dormant.textContent ?? "", /^Codex · 5-hour: no reading since /);
  assert.equal(screen.getAllByTestId("dashboard-usage-line").length, 2);
});

test("an on-track window is one quiet line and the strip is absent without windows or when the forecast fails", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ status: "on_track" }));
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());
  const first = renderWithProviders(createElement(DashboardUsageStrip));
  const [codex] = await screen.findAllByTestId("dashboard-usage-line");
  assert.doesNotMatch(codex?.textContent ?? "", /Runs out|recommendation/);
  first.unmount();

  measures.usageForecast.mockRejectedValue(new ConnectError("down", Code.Unavailable));
  renderWithProviders(createElement(DashboardUsageStrip));
  await waitFor(() => assert.ok(measures.usageForecast.mock.calls.length > 1));
  assert.equal(screen.queryByTestId("dashboard-usage-strip"), null);
});

test("the run usage panel reads the run's own row and the recommendations that name it", async () => {
  measures.usageBurners.mockResolvedValue(makeUsageBurners("run"));
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());

  renderWithProviders(createElement(RunUsagePanel, { runId: BURNER_RUN_ID, ...NO_COST }));
  const usage = within(await screen.findByTestId("run-usage-window"));
  assert.deepEqual(measures.usageBurners.mock.calls[0]?.[0], { groupBy: "run", runId: BURNER_RUN_ID });
  assert.deepEqual(measures.usageRecommendations.mock.calls[0]?.[0], { runId: BURNER_RUN_ID });
  assert.ok(usage.getByText("≈ 2.2 pts"));
  assert.ok(usage.getByText("of Codex · weekly"));
  assert.ok(usage.getAllByText("1.1 B").length > 0);
  assert.ok(usage.getByText("3,335 requests"));
  assert.ok(await screen.findByText("Recommendations for this run"));
  assert.equal(screen.getByRole("link", { name: "All recommendations →" }).getAttribute("href"), "/usage");
});

test("a run with neither quota nor cost data shows one empty message, only after both have loaded", async () => {
  measures.usageBurners.mockResolvedValue(makeUsageBurners("run"));
  measures.usageRecommendations.mockResolvedValue({ groups: [], rules: [] });
  const { rerender } = renderWithProviders(createElement(RunUsagePanel, { runId: "other-run", hasCost: false, costLoading: true }));
  await waitFor(() => assert.ok(measures.usageBurners.mock.calls.length > 0));
  assert.ok(screen.getByTestId("data-state-skeleton"), "cost still loading: a skeleton, not an empty claim");
  assert.equal(screen.queryByText("No usage recorded for this run"), null);
  rerender(createElement(RunUsagePanel, { runId: "other-run", ...NO_COST }));
  assert.ok(await screen.findByText("No usage recorded for this run"));
  assert.equal(screen.getAllByText(/No usage recorded|No cost events|No quota attributed/).length, 1);
  assert.equal(screen.queryByText("Recommendations for this run"), null);
});

test("cost events without attributed quota show the cost, never an empty message beside it", async () => {
  measures.usageBurners.mockResolvedValue(makeUsageBurners("run"));
  measures.usageRecommendations.mockResolvedValue({ groups: [], rules: [] });
  renderWithProviders(createElement(RunUsagePanel, { runId: "other-run", hasCost: true, costLoading: false }, createElement("div", { "data-testid": "run-cost" }, "$0.12")));
  assert.ok(await screen.findByTestId("run-cost"));
  await waitFor(() => assert.ok(measures.usageBurners.mock.calls.length > 0));
  assert.equal(screen.queryByText(/No usage recorded|No cost events|No quota attributed/), null);
});
