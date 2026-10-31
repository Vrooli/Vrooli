import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { UsagePage } from "./UsagePage.js";
import { renderWithProviders } from "../../test-utils/index.js";
import { makeUsageCompare, makeUsageOutcomes, makeUsageParity, type OutcomesKind } from "../../test-utils/fixtures/usage.js";

const measures = vi.hoisted(() => ({
  usageOutcomes: vi.fn(), usageParity: vi.fn(), allMeasureDefinitions: vi.fn(), usageCompare: vi.fn(), usageExperiment: vi.fn(),
  usageForecast: vi.fn(), usageBurners: vi.fn(), usageRecommendations: vi.fn(), usageCoverage: vi.fn(), listUsageModes: vi.fn(),
}));
vi.mock("./measuresClient.js", () => ({ measuresClient: measures }));

function serve(kind: OutcomesKind, definitions: string[] = []) {
  measures.usageOutcomes.mockImplementation(async ({ by }: { by: string }) => makeUsageOutcomes(kind, by));
  measures.usageParity.mockResolvedValue(makeUsageParity());
  measures.allMeasureDefinitions.mockResolvedValue({ definitions: definitions.map((id) => ({ id })) });
  measures.usageCompare.mockResolvedValue(makeUsageCompare());
  measures.usageCoverage.mockResolvedValue({ roots: [] });
  measures.usageExperiment.mockResolvedValue({ experiments: [] });
}

function open() {
  return renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
}

beforeEach(() => serve("observational"));
afterEach(() => Object.values(measures).forEach((fn) => fn.mockReset()));

test("the conclusion comes first with its strength pill, then the break-even bar and the comparison, from live values", async () => {
  open();
  const conclusion = await screen.findByTestId("evidence-conclusion");
  assert.equal(within(conclusion).getByTestId("evidence-pill").textContent, "Observational");
  assert.match(within(conclusion).getByTestId("evidence-text").textContent ?? "", /luna cost about 3\.6x less per accepted outcome/);
  assert.match(within(conclusion).getByTestId("evidence-strength").textContent ?? "", /an association to look into, not what the model caused/);
  const order = screen.getAllByTestId(/^evidence-(conclusion|break-even|over-time|models)$/).map((el) => el.dataset.testid);
  assert.deepEqual(order, ["evidence-conclusion", "evidence-break-even", "evidence-over-time", "evidence-models"]);
  const bar = screen.getByTestId("break-even");
  assert.match(bar.textContent ?? "", /luna cost 5\.0x less per reference token than terra on these units/);
  assert.match(bar.textContent ?? "", /The model price alone is 4\.0x less at one fixed token mix \(a prior from its tier/);
  assert.match(bar.textContent ?? "", /Break-even on these units 5\.0x/);
  assert.match(bar.textContent ?? "", /Used 1\.4x \(1\.1x to 1\.9x\)/);
  const rows = screen.getAllByTestId("model-comparison-row");
  assert.match(rows[1]?.textContent ?? "", /2\.0 pts to 3\.4 pts/);
});

test("a wide range reads inconclusive everywhere, and the bar does not call it settled", async () => {
  serve("inconclusive");
  open();
  assert.match((await screen.findByTestId("evidence-text")).textContent ?? "", /^Inconclusive on operator sessions: luna cost/);
  const rows = screen.getAllByTestId("model-comparison-row");
  assert.match(rows[1]?.textContent ?? "", /Inconclusive/);
  assert.doesNotMatch(rows[1]?.textContent ?? "", /About the same|cheaper/i);
  assert.match(screen.getByTestId("break-even").textContent ?? "", /does not settle whether it cost less/);
});

test("the table states the path, the completed tier apart from accepted, and the weaker unit under its own label", async () => {
  measures.usageOutcomes.mockImplementation(async ({ by }: { by: string }) => {
    const response = makeUsageOutcomes("early", by);
    if (by === "model") {
      response.rows = [{ ...response.rows[0]!, stratum: "headless", units: 47n, accepted: 0n, completed: 47n, unknownRate: 0, statusReason: "0 accepted outcomes; need 5", pointsPerCompleted: { available: true, value: 0.7, low: 0.1, high: 1.6, samples: 47n } as never, completedReading: "inconclusive", completedBaseline: "terra", cheaperByCompleted: { available: true, value: 0.97, low: 0.13, high: 49.5, samples: 69n } as never }];
    }
    return response;
  });
  open();
  const table = await screen.findByTestId("model-comparison");
  assert.match(within(table).getByRole("columnheader", { name: /Cost per completed \(weaker\)/ }).textContent ?? "", /weaker/);
  const row = within(table).getAllByTestId("model-comparison-row")[0]?.textContent ?? "";
  assert.match(row, /codex, headless runs/);
  assert.match(row, /0 of 47/);
  assert.match(row, /47 only finished/);
  assert.match(row, /retries only: no operator to correct it/);
  assert.match(row, /Per completed only, not a verdict: inconclusive/);
  assert.match(row, /Not enough data/);
});

test("the request asks the API for the same window and grouping the CLI defaults to, 30 days", async () => {
  open();
  await screen.findByTestId("evidence-conclusion");
  const request = measures.usageOutcomes.mock.calls.find((call) => call[0].by === "model")?.[0];
  assert.equal(request.window.window.case, "token");
  assert.equal(request.by, "model");
});

test("a randomized verdict carries the Randomized pill", async () => {
  serve("randomized");
  open();
  assert.equal((await screen.findByTestId("evidence-pill")).textContent, "Randomized");
  assert.doesNotMatch(screen.getByTestId("break-even").textContent ?? "", /prior/);
});

test("too early to say shows counts only, no ratios, and no break-even bar", async () => {
  serve("early");
  open();
  assert.equal((await screen.findByTestId("evidence-pill")).textContent, "Not enough data");
  assert.match(screen.getByTestId("evidence-text").textContent ?? "", /Too early to say/);
  assert.ok(screen.getByTestId("break-even-missing"));
  assert.equal(screen.queryByTestId("break-even"), null);
});

test("no mode run shows its own state and a way to start one", async () => {
  serve("none");
  open();
  assert.equal((await screen.findByTestId("evidence-pill")).textContent, "No mode run yet");
  assert.ok(screen.getByTestId("evidence-start-mode"));
  assert.match(screen.getByTestId("before-after-none").textContent ?? "", /No usage mode ran in this period/);
});

test("the chart shades the mode period, draws each model, and keeps every number in a table", async () => {
  open();
  const chart = await screen.findByTestId("evidence-chart");
  assert.equal(within(chart).getAllByTestId("evidence-mode-band").length, 1);
  assert.equal(within(chart).getAllByTestId("evidence-series").length, 2);
  const table = within(chart).getByTestId("evidence-chart-table");
  assert.equal(within(table).getAllByRole("columnheader").map((h) => h.textContent).join(","), "Day,terra,luna");
});

test("more than three models become small charts on one scale instead of crowding one", async () => {
  measures.usageOutcomes.mockImplementation(async ({ by }: { by: string }) => {
    const response = makeUsageOutcomes("observational", by);
    response.series = ["a", "b", "c", "d"].map((model) => ({ ...response.series[0]!, model }));
    return response;
  });
  open();
  const multiples = await screen.findByTestId("evidence-multiples");
  assert.equal(within(multiples).getAllByTestId("evidence-series").length, 4);
});

test("quality compares work with the mode on and off, in plain columns", async () => {
  open();
  const rows = await screen.findAllByTestId("quality-row");
  assert.match(rows[0]?.textContent ?? "", /No mode on.*operator sessions.*14% \(9% to 20%\).*4 of 40/);
  assert.match(rows[1]?.textContent ?? "", /Save usage.*18% \(10% to 27%\).*6 of 36/);
});

test("operator signals say not available yet until the measure is defined", async () => {
  open();
  await waitFor(() => assert.match(screen.getByTestId("operator-signals").textContent ?? "", /not available yet/));
});

test("a failing definitions read still shows the page, with operator signals not available", async () => {
  measures.allMeasureDefinitions.mockRejectedValue(new Error("down"));
  open();
  await screen.findByTestId("evidence-conclusion");
  assert.match(screen.getByTestId("operator-signals").textContent ?? "", /not available yet/);
});

test("coverage by session type reads headless, interactive and imported from the parity numbers", async () => {
  open();
  const rows = await screen.findAllByTestId("session-coverage-row");
  assert.deepEqual(rows.map((row) => row.textContent), [
    "Headless runs100 runs, 100 counted92% have usage data80% have an outcome",
    "Interactive sessions50 runs, 50 counted50% have usage data40% have an outcome",
    "Imported sessions20 runs, 20 counted35% have usage data10% have an outcome",
  ]);
});

test("before and after compares the span before a mode with the mode and prints the usage compare command", async () => {
  const user = userEvent.setup();
  open();
  await user.click(await screen.findByTestId("before-after-open"));
  const result = await screen.findByTestId("before-after-result");
  assert.match(result.textContent ?? "", /Cost per accepted outcome fell.*-50% per accepted outcome/);
  const call = measures.usageCompare.mock.calls[0]?.[0];
  assert.equal(call.left.window.window.case, "custom");
  assert.match(within(result).getByTestId("before-after-command").textContent ?? "", /^agent-manager usage compare --left-filter-json '\{"window":\{"custom":\{"from":"2026-10-07T00:00:00Z","to":"2026-10-11T00:00:00Z"\}\}\}' --right-filter-json '\{"window":\{"custom":\{"from":"2026-10-11T00:00:00Z","to":"2026-10-15T00:00:00Z"\}\}\}'$/);
});

test("an unreachable ledger says it could not load evidence, not that there is none", async () => {
  measures.usageOutcomes.mockRejectedValue(new Error("the run ledger is unavailable"));
  open();
  assert.ok(await screen.findByRole("alert"));
  assert.equal(screen.queryByTestId("evidence-conclusion"), null);
});

test("the window chart shades a mode inside the window and clips it to the window, leaving the run-out band", async () => {
  const { render } = await import("@testing-library/react");
  const { UsageWindowChart } = await import("./UsageWindowChart.js");
  const { makeUsageForecast } = await import("../../test-utils/fixtures/usage.js");
  const forecast = makeUsageForecast().windows.find((w) => w.series.length >= 2);
  assert.ok(forecast, "fixture has a window with readings");
  const start = new Date(forecast.windowStart).toISOString();
  const { queryAllByTestId } = render(createElement(UsageWindowChart, { forecast, modes: [
    { id: "in", label: "Save usage", startedAt: start, endsAt: "" },
    { id: "before", label: "Old", startedAt: "2020-01-01T00:00:00Z", endsAt: "2020-01-02T00:00:00Z" },
  ] }));
  assert.equal(queryAllByTestId("window-mode-band").length, 1);
});
