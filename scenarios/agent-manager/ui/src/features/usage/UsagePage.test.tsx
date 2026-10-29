import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConnectError, Code } from "@connectrpc/connect";
import { createElement } from "react";
import { useLocation } from "react-router-dom";
import { afterEach, beforeEach, test, vi } from "vitest";
import { UsagePage } from "./UsagePage.js";
import { renderWithProviders } from "../../test-utils/index.js";
import { BURNER_RUN_ID, makeUsageBurners, makeUsageCoverage, makeUsageForecast, makeUsageRecommendations, USAGE_NOW } from "../../test-utils/fixtures/usage.js";

const measures = vi.hoisted(() => ({ usageForecast: vi.fn(), usageBurners: vi.fn(), usageRecommendations: vi.fn(), usageCoverage: vi.fn() }));
vi.mock("../stats/api/measuresClient.js", () => ({ measuresClient: measures }));
vi.mock("./ManualImport.js", () => ({ ManualImport: () => createElement("div", { "data-testid": "manual-import" }, "Manual import") }));
vi.mock("../stats/index.js", () => ({ StatsPage: () => createElement("div", { "data-testid": "stats-panels" }, "Run analytics panels") }));

function serveFixtures() {
  measures.usageForecast.mockResolvedValue(makeUsageForecast());
  measures.usageBurners.mockImplementation(async ({ groupBy }: { groupBy: string }) => makeUsageBurners(groupBy));
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());
  measures.usageCoverage.mockResolvedValue(makeUsageCoverage());
}

beforeEach(serveFixtures);
afterEach(() => Object.values(measures).forEach((fn) => fn.mockReset()));

function LocationProbe() {
  const location = useLocation();
  return createElement("output", { "data-testid": "location" }, `${location.pathname}${location.search}`);
}

function renderUsage(entry = "/usage") {
  return renderWithProviders(createElement("div", null, createElement(UsagePage), createElement(LocationProbe)), { initialEntries: [entry] });
}

const section = (id: string) => screen.getByTestId(`usage-section-${id}`);

/** Rendered copy only: RCL components inline their stylesheet, whose CSS is not page text. */
function visibleText(element: HTMLElement): string {
  const copy = element.cloneNode(true) as HTMLElement;
  copy.querySelectorAll("style").forEach((style) => style.remove());
  return copy.textContent ?? "";
}

test("a provider window shows used percent, run-out band, the rate that lasts, and unknown horizons as —", async () => {
  renderUsage();
  const [codex, claude] = await screen.findAllByTestId("usage-window");
  assert.ok(codex && claude);
  const card = within(codex);
  assert.ok(card.getByText("Codex · weekly"), "scopes are named like the CLI, not by raw provider/pool ids");
  assert.ok(within(claude).getByText("Claude · 5-hour"));
  assert.ok(card.getByText("Will run out before reset"));
  assert.ok(card.getByText("33%"));
  assert.match(codex.textContent ?? "", /Projected to run out .+ – .+; the reset is/);
  assert.match(codex.textContent ?? "", /burn at most 0\.46%\/h/);
  assert.match(codex.textContent ?? "", /Last 1h—no reading in 1h/, "an unavailable horizon is unknown, not zero");
  assert.match(codex.textContent ?? "", /newest reading says 27%; it lags the highest reading this window \(33%\)/);
  assert.match(codex.textContent ?? "", /calibrated on 4 past windows \(×1\.12\)/);
  assert.match(codex.textContent ?? "", /Last 6h3\.4%\/h/);
  assert.ok(card.getByTestId("usage-window-chart"));
  assert.equal(card.queryByRole("note"), null, "a fresh reading carries no stale note");
  assert.ok(within(claude).getByText("No reading yet"));
  assert.ok(within(claude).getByText("No status line reading yet."));
  assert.equal(/\d+%/.test(visibleText(claude)), false, "a window without readings shows no percentage");
});

test("a window with no reading since its reset collapses to one muted line, with times formatted", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ dormant: true }));
  renderUsage();
  const windows = section("windows");
  const dormant = await within(windows).findByTestId("usage-window-dormant");
  assert.match(dormant.textContent ?? "", /^Codex · 5-hour: no reading since [A-Z][a-z]{2} \d{1,2}, \d{1,2}:\d{2}/);
  assert.equal(within(windows).getAllByTestId("usage-window").length, 2, "the dead window is not a forecast card");
  assert.equal(within(windows).queryByText(/Forecast unknown/), null);
  assert.equal(/\d{4}-\d{2}-\d{2}T/.test(visibleText(windows)), false, "no raw RFC 3339 time reaches the page");
});

test("server sentences show their timestamps like every other time", async () => {
  const forecast = makeUsageForecast();
  const claude = forecast.windows[1]!;
  claude.sampleCount = 3n;
  claude.observedAt = new Date(USAGE_NOW - 600_000).toISOString();
  claude.statusReason = "the window reset at 2026-09-09T20:28:00Z";
  measures.usageForecast.mockResolvedValue(forecast);
  renderUsage();
  const [, card] = await screen.findAllByTestId("usage-window");
  assert.doesNotMatch(card?.textContent ?? "", /2026-09-09T20:28:00Z/);
  assert.match(card?.textContent ?? "", /the window reset at Sep 9, \d{1,2}:28 [AP]M/);
});

test("a reading older than six hours is flagged as stale", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ observedAt: new Date(USAGE_NOW - 9 * 3600_000).toISOString() }));
  renderUsage();
  const [codex] = await screen.findAllByTestId("usage-window");
  assert.match(within(codex as HTMLElement).getByRole("note").textContent ?? "", /newest reading is 9h ago/);
});

test("what's using it lists estimated points, the remainder rows, metered work and the calibration, and drills into runs", async () => {
  const user = userEvent.setup();
  renderUsage();
  const burners = within(section("burners"));
  const rows = await burners.findAllByTestId("usage-burner");
  assert.equal(rows.length, 2);
  assert.match(rows[0]?.textContent ?? "", /Research orchestrator.*≈ 28 pts/);
  assert.match(rows[0]?.textContent ?? "", /12 images in/);
  assert.equal(burners.getByRole("link", { name: "Vega E1 Overview" }).getAttribute("href"), `/runs/${BURNER_RUN_ID}?tab=usage`);
  assert.match(burners.getByTestId("usage-burner-other").textContent ?? "", /77 more.*≈ 0\.3 pts/);
  assert.match(burners.getByTestId("usage-burner-unattributed").textContent ?? "", /Unattributed \/ off-disk.*≈ 0\.4 pts/);
  assert.match(burners.getByTestId("usage-metered-lines").textContent ?? "", /Web Search: 3 · charge unknown/);
  assert.ok(burners.getByText(/120 K tokens were used after the newest reading/));
  await user.click(burners.getByText("How this is estimated"));
  assert.match(section("burners").textContent ?? "", /fitted on 48 slices, R² 0\.91/);

  await user.click(burners.getByRole("button", { name: "Profile" }));
  await waitFor(() => assert.ok(measures.usageBurners.mock.calls.some(([request]) => request.groupBy === "profile")));
  assert.equal(burners.getByRole("button", { name: "Profile" }).getAttribute("aria-pressed"), "true");
  await user.click(burners.getByRole("button", { name: "Group by" }));
  await user.click(screen.getByRole("option", { name: "Tool kind" }));
  await waitFor(() => assert.ok(measures.usageBurners.mock.calls.some(([request]) => request.groupBy === "tool_kind")));
});

test("recommendations are ranked with saving, confidence, evidence, and say why a rule did not fire", async () => {
  renderUsage();
  const recommendations = within(section("recommendations"));
  const rows = await recommendations.findAllByTestId("usage-recommendation");
  assert.equal(rows.length, 2);
  assert.match(rows[0]?.textContent ?? "", /1.*Start a fresh session.*Context.*−12 pts\/wk.*high confidence/);
  assert.equal(rows[0]?.hasAttribute("open"), true, "the top recommendation opens on its evidence");
  assert.match(rows[0]?.textContent ?? "", /Re-read per turn185 K/);
  assert.match(rows[0]?.textContent ?? "", /Sub agent points11\.8 pts/);
  const evidence = within(rows[0] as HTMLElement);
  assert.equal(evidence.getByRole("link", { name: "Vega E1 Overview · 2d230f9d" }).getAttribute("href"), `/runs/${BURNER_RUN_ID}?tab=usage`);
  // Same-label sub-agents read differently, each by its short id.
  assert.equal(evidence.getByRole("link", { name: "sub-agent · 00b65371" }).getAttribute("href"), "/runs/00b65371-7359-42d6-b8f4-83218914f65f?tab=usage");
  assert.ok(evidence.getByRole("link", { name: "sub-agent · 1d804127" }));
  assert.ok(evidence.getByText("web-console 1e5904ed"), "a label that already names its id is not repeated");
  assert.equal(rows[1]?.hasAttribute("open"), false);
  assert.equal(recommendations.getByTestId("usage-rule-not-fired").textContent, "R-idle did not fire: no parked session consumed quota this window");
});

test("coverage: the header chip and overview summarise, the Coverage tab names every unreadable file", async () => {
  const user = userEvent.setup();
  renderUsage();
  const chip = await screen.findByTestId("usage-coverage-chip");
  assert.match(chip.textContent ?? "", /Coverage 97% · 7 days/);
  const overview = within(section("coverage-summary"));
  assert.equal((await overview.findAllByTestId("usage-coverage-root")).length, 2);
  await user.click(overview.getByRole("link", { name: /2 files could not be read/ }));
  assert.equal(screen.getByTestId("location").textContent, "/usage?tab=coverage");
  const issues = await screen.findByTestId("usage-coverage-issues");
  assert.match(issues.textContent ?? "", /a\.jsonl — still being written/);
  assert.ok(screen.getByText(/Compare with: find/));
  assert.ok(screen.getByTestId("manual-import"));
});

test("an RPC the running API does not serve yet reads as 'not available yet', never as a raw error", async () => {
  const unimplemented = new ConnectError("agent_manager.v1.measures.MeasuresService.UsageBurners is not implemented", Code.Unimplemented);
  measures.usageBurners.mockRejectedValue(unimplemented);
  measures.usageRecommendations.mockRejectedValue(unimplemented);
  renderUsage();
  const burners = within(section("burners"));
  assert.match((await burners.findByRole("alert")).textContent ?? "", /Attribution: Not available yet/);
  assert.equal(section("burners").textContent?.includes("[unimplemented]"), false);
  assert.equal(burners.queryByText("Nothing to attribute yet"), null, "a failed load never reads as empty");
});

test("slow attribution and recommendations show layout skeletons, then their content", async () => {
  let answerBurners: (value: unknown) => void = () => undefined;
  measures.usageBurners.mockImplementation(() => new Promise((resolve) => { answerBurners = resolve; }));
  measures.usageRecommendations.mockImplementation(() => new Promise(() => undefined));
  renderUsage();
  for (const id of ["burners", "recommendations"]) {
    const skeleton = within(section(id)).getByTestId("data-state-skeleton");
    assert.equal(skeleton.getAttribute("aria-busy"), "true");
  }
  answerBurners(makeUsageBurners("session"));
  assert.equal((await within(section("burners")).findAllByTestId("usage-burner")).length, 2);
  assert.equal(within(section("burners")).queryByTestId("data-state-skeleton"), null);
});

test("a slow RPC ends in a timeout state, not an endless spinner", async () => {
  measures.usageRecommendations.mockRejectedValue(new ConnectError("the operation timed out", Code.DeadlineExceeded));
  renderUsage();
  const alert = await within(section("recommendations")).findByRole("alert");
  assert.match(alert.textContent ?? "", /Recommendations: Request timed out/);
  assert.ok(measures.usageRecommendations.mock.calls[0]?.[1]?.timeoutMs > 0, "every usage call carries a deadline");
});

test("switching group-by keeps the previous grouping visible until the new one answers", async () => {
  const user = userEvent.setup();
  renderUsage();
  const burners = within(section("burners"));
  await burners.findAllByTestId("usage-burner");
  measures.usageBurners.mockImplementation(() => new Promise(() => undefined));
  await user.click(burners.getByRole("button", { name: "Model" }));
  assert.equal(burners.getAllByTestId("usage-burner").length, 2);
  assert.equal(burners.queryByTestId("data-state-skeleton"), null);
});

test("empty answers keep honest empty states without fabricated numbers", async () => {
  measures.usageForecast.mockResolvedValue({ windows: [] });
  measures.usageBurners.mockResolvedValue({ windows: [] });
  measures.usageRecommendations.mockResolvedValue({ recommendations: [], rules: [] });
  measures.usageCoverage.mockResolvedValue({ roots: [] });
  renderUsage();
  for (const [id, empty] of [["windows", "No provider readings yet"], ["burners", "Nothing to attribute yet"], ["recommendations", "No recommendations yet"], ["coverage-summary", "Coverage not measured yet"]] as const) {
    assert.ok(await within(section(id)).findByText(empty));
    assert.equal(/\d+\s*%/.test(visibleText(section(id))), false, `${id} must not show a percentage without data`);
  }
  assert.equal(screen.queryByTestId("usage-coverage-chip"), null);
});

test("Coverage hosts manual import and Details hosts the run analytics, both addressable by URL", async () => {
  const user = userEvent.setup();
  renderUsage();
  await user.click(screen.getByTestId("usage-tab-coverage"));
  assert.ok(screen.getByTestId("manual-import"));
  assert.equal(screen.getByTestId("location").textContent, "/usage?tab=coverage");

  await user.click(screen.getByTestId("usage-tab-details"));
  assert.ok(await screen.findByTestId("stats-panels"));
  assert.equal(screen.getByTestId("location").textContent, "/usage?tab=details");

  await user.click(screen.getByTestId("usage-tab-overview"));
  assert.equal(screen.getByTestId("location").textContent, "/usage");
});

test("a deep link opens its tab and an unknown tab falls back to the overview", async () => {
  const first = renderUsage("/usage?tab=details");
  assert.ok(await screen.findByTestId("stats-panels"));
  assert.equal(screen.getByTestId("usage-tab-details").getAttribute("aria-selected"), "true");
  first.unmount();

  renderUsage("/usage?tab=bogus");
  assert.equal(screen.getByTestId("usage-tab-overview").getAttribute("aria-selected"), "true");
});
