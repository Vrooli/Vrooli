import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConnectError, Code } from "@connectrpc/connect";
import { createElement } from "react";
import { useLocation } from "react-router-dom";
import { afterEach, beforeEach, test, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { UsageWindowForecastSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { UsagePage } from "./UsagePage.js";
import { renderWithProviders } from "../../test-utils/index.js";
import { BURNER_RUN_ID, CLAUDE_SCOPE, CODEX_SCOPE, FRICTION_FINGERPRINT, makeAdmittedInterventions, makeDetailsMeasures, makePooledInterventions, makeUsageBurners, makeUsageCoverage, makeUsageForecast, makeUsageFriction, makeUsageModes, makeUsageRecommendations, USAGE_NOW } from "../../test-utils/fixtures/usage.js";

const measures = vi.hoisted(() => ({
  usageForecast: vi.fn(), usageBurners: vi.fn(), usageRecommendations: vi.fn(), usageCoverage: vi.fn(), usageFriction: vi.fn(), listUsageModes: vi.fn(), startUsageMode: vi.fn(), endUsageMode: vi.fn(),
  runSuccessRate: vi.fn(), runCost: vi.fn(), runCycleTime: vi.fn(), runStatusDistribution: vi.fn(), modelBreakdown: vi.fn(), profileBreakdown: vi.fn(),
}));
vi.mock("./measuresClient.js", () => ({ measuresClient: measures }));
vi.mock("./ManualImport.js", () => ({ ManualImport: () => createElement("div", { "data-testid": "manual-import" }, "Manual import") }));

function serveFixtures() {
  measures.usageForecast.mockResolvedValue(makeUsageForecast());
  measures.usageBurners.mockImplementation(async ({ groupBy }: { groupBy: string }) => makeUsageBurners(groupBy));
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());
  measures.usageCoverage.mockResolvedValue(makeUsageCoverage());
  measures.usageFriction.mockResolvedValue(makeUsageFriction());
  measures.listUsageModes.mockResolvedValue(makeUsageModes(false));
  const details = makeDetailsMeasures();
  for (const [name, answer] of Object.entries(details)) measures[name as keyof typeof details].mockResolvedValue(answer);
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

const section = (id: string) => screen.findByTestId(`usage-section-${id}`);

/** Rendered copy only: RCL components inline their stylesheet, whose CSS is not page text. */
function visibleText(element: HTMLElement): string {
  const copy = element.cloneNode(true) as HTMLElement;
  copy.querySelectorAll("style").forEach((style) => style.remove());
  return copy.textContent ?? "";
}

test("the overview shows one compact tile per window and selects the most urgent one", async () => {
  renderUsage();
  const [codex, claude] = await screen.findAllByTestId("usage-window");
  assert.ok(codex && claude);
  assert.ok(within(codex).getByText("Codex · weekly"), "scopes are named like the CLI, not by raw provider/pool ids");
  assert.ok(within(claude).getByText("Claude · 5-hour"));
  assert.equal(codex.getAttribute("data-selected"), "true", "the will-run-out window is selected by default");
  assert.ok(within(codex).getByText("Will run out"));
  assert.ok(within(codex).getByText("33%"));
  assert.ok(within(claude).getByText("No reading"));
  assert.equal(/\d+%/.test(visibleText(claude)), false, "a window without readings shows no percentage");
});

test("?window= selects that window and asks the API for its recommendations only", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ idleClaude: true }));
  renderUsage(`/usage?window=${encodeURIComponent("anthropic/five_hour/300m")}`);
  const [, claude] = await screen.findAllByTestId("usage-window");
  assert.equal(claude?.getAttribute("data-selected"), "true");
  assert.match(claude?.textContent ?? "", /Idle, lasts at this pace/);
  await waitFor(() => assert.ok(measures.usageRecommendations.mock.calls.some(([request]) => request.window === CLAUDE_SCOPE.window && request.pool === CLAUDE_SCOPE.pool && request.provider === CLAUDE_SCOPE.provider)));
  assert.equal(screen.queryByTestId("usage-focus")?.textContent?.includes("runs out"), false, "an idle window claims no run-out");
});

test("choosing a window tile puts it in the URL", async () => {
  const user = userEvent.setup();
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ idleClaude: true }));
  renderUsage();
  const [, claude] = await screen.findAllByTestId("usage-window");
  await user.click(claude as HTMLElement);
  assert.equal(screen.getByTestId("location").textContent, `/usage?window=${encodeURIComponent("anthropic/five_hour/300m")}`);
});

test("the focus band states the burn, what can be afforded, and the API's cut, combined saving and consequence", async () => {
  renderUsage();
  const focus = await screen.findByTestId("usage-focus");
  const text = visibleText(focus);
  assert.match(text, /Codex · weekly runs out around/);
  assert.match(text, /Used33%/);
  assert.match(text, /Burning now \(24h\)1\.5%\/h/);
  assert.match(text, /Can afford until reset≤ 0\.46%\/h/);
  assert.match(text, /Needs a cut of69%/);
  assert.match(text, /The quota action below can still save≈ 9.0 pts/);
  const runOut = new Intl.DateTimeFormat("en-US", { weekday: "short", month: "short", day: "numeric", hour: "numeric" }).format(Date.parse(makeUsageRecommendations().outcome?.runOutAt ?? ""));
  assert.ok(text.includes(`move the run-out to around ${runOut}, still 4 days short of the reset`), "the consequence is the API's projected outcome, not a page computation");
});

test("one ranked list: owner, action, score and native effect, evidence on demand, coverage and the materiality count", async () => {
  const user = userEvent.setup();
  renderUsage();
  const cards = await within(await section("recommendations")).findAllByTestId("usage-recommendation");
  assert.equal(cards.length, 3);
  const first = within(cards[0] as HTMLElement);
  assert.ok(first.getByText("Compact codex sessions after ~60 requests of growth"));
  assert.match(cards[0]?.textContent ?? "", /harness: codex session context/);
  assert.match(cards[0]?.textContent ?? "", /26\.7%.*≈ 12 pts · projected.*26\.7% of the openai 10080m window's 45\.0 observed points/);
  assert.match(cards[1]?.textContent ?? "", /command: test-genie runs wait.*7\.5%.*161 calls · observed.*of the window's 2149 failed calls/);
  assert.match(cards[2]?.textContent ?? "", /unproven.*0 to 6\.0 pts · unproven/);
  assert.equal(first.queryByRole("button", { name: /Dismiss/ }), null, "the API has no recommendation dismissal, so none is offered");
  assert.match((await section("recommendations")).textContent ?? "", /3 scoring under 1%/, "server-folded items are counted");
  assert.equal(first.queryByTestId("usage-evidence"), null);
  await user.click(first.getByRole("button", { name: "Show evidence" }));
  assert.ok(first.getByRole("link", { name: "sub-agent · 00b65371" }));
  assert.match(cards[0]?.textContent ?? "", /Token usage analysis · would have saved ≈4\.0 pts \(finished\); compacted 6×/);
  assert.match(cards[0]?.textContent ?? "", /Verify: after the change/);
  assert.match(within(await section("recommendations")).getByTestId("usage-recommendation-notes").textContent ?? "", /cannot tell, not none.*kinds of harm weigh equally/);
});

test("where it went lists points and share per row, folds near-zero rows, and drills into runs", async () => {
  const user = userEvent.setup();
  renderUsage();
  const burners = within(await section("burners"));
  const rows = await burners.findAllByTestId("usage-burner");
  assert.equal(rows.length, 2);
  assert.match(rows[0]?.textContent ?? "", /Research orchestrator.*28\.0 · 84%/);
  assert.equal(burners.getByRole("link", { name: "Vega E1 Overview" }).getAttribute("href"), `/runs/${BURNER_RUN_ID}?tab=usage`);
  assert.match(burners.getByTestId("usage-burner-other").textContent ?? "", /77 smaller items \(under 0\.25 pts each, or past the rows shown\).*0\.3/, "the fold rule and its sum are the API's");
  assert.match(burners.getByTestId("usage-burner-unattributed").textContent ?? "", /Unattributed \/ off-disk.*0\.4/);
  assert.match(burners.getByTestId("usage-metered-lines").textContent ?? "", /Web search · 3 calls · charge unknown/);
  await user.click(burners.getByText("How this is estimated"));
  assert.match((await section("burners")).textContent ?? "", /fitted on 48 slices, R² 0\.91/);
  assert.equal(within(await section("burners")).getByTestId("usage-rules-not-fired").textContent, "R-idle did not fire: no session made wait-only requests this window");
  await user.click(burners.getByRole("button", { name: "Profile" }));
  await waitFor(() => assert.ok(measures.usageBurners.mock.calls.some(([request]) => request.groupBy === "profile")));
  assert.equal(burners.getByRole("button", { name: "Profile" }).getAttribute("aria-pressed"), "true");
});

test("the selected window's projection and chart sit below the actions, with the stale note when the reading is old", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ observedAt: new Date(USAGE_NOW - 9 * 3600_000).toISOString() }));
  renderUsage();
  const detail = await screen.findByTestId("usage-window-detail");
  assert.match(detail.textContent ?? "", /Projected to run out .+ – .+; the reset is/);
  assert.match(detail.textContent ?? "", /Last 1h—no reading in 1h/, "an unavailable horizon is unknown, not zero");
  assert.match(detail.textContent ?? "", /newest reading says 27%; it lags the highest reading this window \(33%\)/);
  assert.ok(within(detail).getByTestId("usage-window-chart"));
  assert.match(within(detail).getByRole("note").textContent ?? "", /newest reading is 9h ago/);
});

test("a window with no reading since its reset collapses to one muted line", async () => {
  measures.usageForecast.mockResolvedValue(makeUsageForecast({ dormant: true }));
  renderUsage();
  const dormant = await screen.findByTestId("usage-window-dormant");
  assert.match(dormant.textContent ?? "", /^Codex · 5-hour: no reading since [A-Z][a-z]{2} \d{1,2}, \d{1,2}:\d{2}/);
  assert.equal(screen.getAllByTestId("usage-window").length, 2, "the dead window is not a tile");
});

test("coverage: the header chip links to the Coverage tab, which names every unreadable file and how it was measured", async () => {
  const user = userEvent.setup();
  renderUsage();
  const chip = await screen.findByTestId("usage-coverage-chip");
  assert.match(chip.textContent ?? "", /Coverage 97% · 7 days/);
  await user.click(chip);
  assert.equal(screen.getByTestId("location").textContent, "/usage?tab=coverage");
  assert.equal((await screen.findAllByTestId("usage-coverage-root")).length, 2);
  assert.match((await screen.findByTestId("usage-coverage-issues")).textContent ?? "", /a\.jsonl — still being written/);
  await user.click(screen.getByText("How this is measured"));
  assert.ok(screen.getByText(/Compare with: find/));
  assert.ok(screen.getByTestId("manual-import"));
});

test("coverage with no observed bytes is unknown rather than 100%", async () => {
  const coverage = makeUsageCoverage();
  coverage.totals!.bytes = 0n;
  coverage.totals!.capturedShare = 1;
  coverage.roots[0].bytes = 0n;
  coverage.roots[0].capturedShare = 1;
  measures.usageCoverage.mockResolvedValue(coverage);
  renderUsage("/usage?tab=coverage");
  assert.match((await screen.findByTestId("usage-coverage-chip")).textContent ?? "", /Coverage n\/a/);
  assert.match((await screen.findAllByTestId("usage-coverage-root"))[0].textContent ?? "", /n\/a/);
  assert.match(document.body.textContent ?? "", /Captured n\/a of transcript bytes/);
});

test("an unknown 'since' date is not replaced by today's", async () => {
  measures.usageCoverage.mockResolvedValue({ ...makeUsageCoverage(), since: "" });
  renderUsage("/usage?tab=coverage");
  assert.equal((await screen.findAllByTestId("usage-coverage-root")).length, 2);
  assert.doesNotMatch(document.body.textContent ?? "", /bytes since/);
});

test("an RPC the running API does not serve yet reads as 'not available yet', never as a raw error", async () => {
  measures.usageBurners.mockRejectedValue(new ConnectError("UsageBurners is not implemented", Code.Unimplemented));
  renderUsage();
  const burners = within(await section("burners"));
  assert.match((await burners.findByRole("alert")).textContent ?? "", /Attribution: Not available yet/);
  assert.equal(burners.queryByText("Nothing to attribute yet"), null, "a failed load never reads as empty");
});

test("a slow RPC ends in a timeout state carrying a deadline, not an endless spinner", async () => {
  measures.usageRecommendations.mockRejectedValue(new ConnectError("the operation timed out", Code.DeadlineExceeded));
  renderUsage();
  const alert = await within(await section("recommendations")).findByRole("alert");
  assert.match(alert.textContent ?? "", /Recommendations: Request timed out/);
  assert.ok(measures.usageRecommendations.mock.calls[0]?.[1]?.timeoutMs > 0, "every usage call carries a deadline");
});

test("slow attribution and recommendations show layout skeletons, then their content", async () => {
  let answerBurners: (value: unknown) => void = () => undefined;
  measures.usageBurners.mockImplementation(() => new Promise((resolve) => { answerBurners = resolve; }));
  measures.usageRecommendations.mockImplementation(() => new Promise(() => undefined));
  renderUsage();
  for (const id of ["burners", "recommendations"]) {
    assert.equal((await within(await section(id)).findByTestId("data-state-skeleton")).getAttribute("aria-busy"), "true");
  }
  answerBurners(makeUsageBurners("session"));
  assert.equal((await within(await section("burners")).findAllByTestId("usage-burner")).length, 2);
});

test("switching group-by keeps the previous grouping visible until the new one answers", async () => {
  const user = userEvent.setup();
  renderUsage();
  const burners = within(await section("burners"));
  await burners.findAllByTestId("usage-burner");
  measures.usageBurners.mockImplementation(() => new Promise(() => undefined));
  await user.click(burners.getByRole("button", { name: "Model" }));
  assert.equal(burners.getAllByTestId("usage-burner").length, 2);
  assert.equal(burners.queryByTestId("data-state-skeleton"), null);
});

test("empty answers keep honest empty states without fabricated numbers", async () => {
  measures.usageForecast.mockResolvedValue({ windows: [] });
  measures.usageCoverage.mockResolvedValue({ roots: [] });
  renderUsage();
  assert.ok(await screen.findByText("No provider readings yet"));
  assert.equal(screen.queryByTestId("usage-focus"), null);
  assert.equal(screen.queryByTestId("usage-coverage-chip"), null);
});

test("tabs are addressable by URL; switching resets the previous tab's parameters; an unknown tab opens the overview", async () => {
  const user = userEvent.setup();
  const first = renderUsage("/usage?tab=details&range=24h");
  assert.equal(screen.getByTestId("usage-tab-details").getAttribute("aria-selected"), "true");
  await user.click(screen.getByTestId("usage-tab-friction"));
  assert.equal(screen.getByTestId("location").textContent, "/usage?tab=friction");
  await user.click(screen.getByTestId("usage-tab-overview"));
  assert.equal(screen.getByTestId("location").textContent, "/usage");
  first.unmount();
  renderUsage("/usage?tab=bogus");
  assert.equal(screen.getByTestId("usage-tab-overview").getAttribute("aria-selected"), "true");
});

const frictionWrites = vi.hoisted(() => ({ reportFriction: vi.fn(), dismissFriction: vi.fn() }));
vi.mock("@connectrpc/connect", async (original) => {
  const actual = await original<typeof import("@connectrpc/connect")>();
  return { ...actual, createClient: (service: { typeName: string }, ...rest: unknown[]) => (service.typeName === "agent_manager.v1.FrictionService" ? frictionWrites : (actual.createClient as (...args: unknown[]) => unknown)(service, ...rest)) };
});

test("friction lists open tool problems in plain words with owner, count, cost and coverage, and filters by status", async () => {
  const user = userEvent.setup();
  renderUsage("/usage?tab=friction");
  const tools = await screen.findAllByTestId("usage-tool-finding");
  assert.equal(tools.length, 2, "only open tool problems by default");
  assert.match(tools[0]?.textContent ?? "", /identity_refresh_unavailable.*failure.*search-hub query recall notes.*vrooli.*128 calls · 20 runs.*failure 101 calls = 12\.4% of the window's 812 Vrooli CLI failed calls.*degraded 63 calls = 0\.3% of the window's 24041 Vrooli CLI calls/i, "a roll-up states each kind's share over its own base");
  assert.match(tools[1]?.textContent ?? "", /output-weight 19\.3M tokens = 3\.1% of the window's context tax/, "cost is in the unit of the harm");
  assert.doesNotMatch(screen.getByTestId("usage-friction").textContent ?? "", /of the window's command calls/, "no share is labelled with a base it was not divided by");
  assert.ok(within(tools[0] as HTMLElement).getByRole("button", { name: "Report bug" }), "a tool problem is filed with its owner");
  assert.match(screen.getByTestId("usage-friction-coverage").textContent ?? "", /Seen since 2026-10-22: 175706 commands, outcome known for 40%/);
  assert.doesNotMatch(document.body.textContent ?? "", new RegExp(FRICTION_FINGERPRINT), "the fingerprint is detail, not the headline");
  assert.match(visibleText(screen.getByTestId("usage-friction")), /Reported to owners1/);
  assert.match(visibleText(screen.getByTestId("usage-friction")), /Open tool problems3/, "the open count is the API's, not the rows on screen");
  assert.match(screen.getByTestId("usage-friction-below").textContent ?? "", /371 more findings score under the materiality floor \(1%\): counted, not listed/);
  await user.click(screen.getByRole("button", { name: "Reported" }));
  assert.match(screen.getByTestId("usage-tool-finding").textContent ?? "", /Reported · knw-1793200000000000001/);
  await user.click(screen.getByRole("button", { name: "Dismissed" }));
  assert.match(screen.getByTestId("usage-tool-finding").textContent ?? "", /Dismissed/);
});

test("an empty friction list says what was seen, never a bare no-problems", async () => {
  const empty = makeUsageFriction();
  empty.toolFindings = [];
  measures.usageFriction.mockResolvedValue(empty);
  renderUsage("/usage?tab=friction");
  assert.match((await screen.findByText(/None met its threshold in what was seen/)).textContent ?? "", /175706 commands, outcome known for 40%/);
});

test("Report bug confirms inline, calls the API once with the listed window, and shows the bug id when the list refreshes", async () => {
  const user = userEvent.setup();
  let answer: (value: unknown) => void = () => undefined;
  frictionWrites.reportFriction.mockImplementation(() => new Promise((resolve) => { answer = resolve; }));
  renderUsage("/usage?tab=friction");
  const [row] = await screen.findAllByTestId("usage-tool-finding");
  const first = within(row as HTMLElement);
  await user.click(first.getByRole("button", { name: "Report bug" }));
  assert.equal(frictionWrites.reportFriction.mock.calls.length, 0, "the first click only asks");
  assert.ok(first.getByText(/File a bug with vrooli\?/));
  await user.click(first.getByRole("button", { name: "Confirm" }));
  await waitFor(() => assert.equal(first.getByRole("button", { name: "Working…" }).hasAttribute("disabled"), true));
  assert.equal(frictionWrites.reportFriction.mock.calls[0]?.[0].fingerprint, FRICTION_FINGERPRINT);
  assert.equal(frictionWrites.reportFriction.mock.calls[0]?.[0].since, "2026-10-22T06:00:00Z", "the finding is looked up in the window the page listed it in");
  const reported = makeUsageFriction();
  reported.toolFindings[0]!.status = "reported";
  reported.toolFindings[0]!.bugId = "knw-42";
  measures.usageFriction.mockResolvedValue(reported);
  answer({ fingerprint: FRICTION_FINGERPRINT, status: "reported", bugId: "knw-42" });
  await user.click(await screen.findByRole("button", { name: "Reported" }));
  assert.ok(await screen.findByText("Reported · knw-42"));
});

test("Dismiss confirms inline and a failed call says so on the row", async () => {
  const user = userEvent.setup();
  frictionWrites.dismissFriction.mockRejectedValue(new Error("api down"));
  renderUsage("/usage?tab=friction");
  const [row] = await screen.findAllByTestId("usage-tool-finding");
  const first = within(row as HTMLElement);
  await user.click(first.getByRole("button", { name: "Dismiss" }));
  await user.click(first.getByRole("button", { name: "Confirm" }));
  assert.match((await first.findByRole("alert")).textContent ?? "", /Could not complete: api down/);
});

test("Details defaults to Agent Manager runs over 7 days and reads every tile from its own measure", async () => {
  renderUsage("/usage?tab=details");
  const tiles = await screen.findByTestId("usage-details-tiles");
  await waitFor(() => assert.match(visibleText(tiles), /509\.6 M/));
  const text = visibleText(tiles);
  assert.match(text, /Runs finished.*47.*95\.7% succeeded/);
  assert.match(text, /97% re-read from cache · 13\.6 M new/);
  assert.doesNotMatch(text, /\$/, "no dollars without per-token billed runs");
  assert.match(text, /Median run time.*14 s.*p90 1 m 50 s · 47 runs/);
  assert.match(text, /Queue.*3.*4 active/);
  assert.match(text, /Quota used \(current windows\)/);
  assert.match(text, /Codex · weekly 33 pts/);
  assert.doesNotMatch(text, /\d+ pts[^C]*\d+ pts.*Median/, "windows are listed, never summed");
  const request = measures.runSuccessRate.mock.calls[0]?.[0];
  assert.equal(request.filter.scope, 1, "RUN_SCOPE_AGENT_MANAGER");
  assert.equal(Number(request.window.window.value.to.seconds) - Number(request.window.window.value.from.seconds), 7 * 86400);
});

test("the quota tile lists each provider pool's primary window as the API marks it, a provider's second pool included", async () => {
  const forecast = makeUsageForecast({ dormant: true });
  // Codex's second pool reports its own weekly window: the API marks it primary for that pool.
  forecast.windows.push(create(UsageWindowForecastSchema, { ...CODEX_SCOPE, pool: "secondary", windowMinutes: 10080n, status: "on_track", hasPercent: true, usedPercent: 12, windowStart: new Date(USAGE_NOW - 24 * 3600_000).toISOString(), resetsAt: new Date(USAGE_NOW + 6 * 24 * 3600_000).toISOString(), observedAt: new Date(USAGE_NOW - 60_000).toISOString(), sampleCount: 9n, primary: true }));
  measures.usageForecast.mockResolvedValue(forecast);
  renderUsage("/usage?tab=details");
  const tiles = await screen.findByTestId("usage-details-tiles");
  await waitFor(() => assert.match(visibleText(tiles), /Codex · weekly · secondary 12 pts/));
  assert.match(visibleText(tiles), /Codex · weekly · primary 33 pts/);
  assert.doesNotMatch(visibleText(tiles), /5-hour 0 pts/, "a dormant, non-primary window is not a current reading");
});

test("Details scope and range live in the URL; the profile table says 'No profile (interactive)', never 'unknown'", async () => {
  const user = userEvent.setup();
  renderUsage("/usage?tab=details");
  assert.ok(await screen.findByText("No profile (interactive)"));
  assert.equal(screen.queryByText("unknown"), null);
  assert.ok(screen.getByText("gpt-6.1-terra"));
  await user.click(screen.getByRole("button", { name: "Operator sessions" }));
  await user.click(screen.getByRole("button", { name: "24h" }));
  assert.equal(screen.getByTestId("location").textContent, "/usage?tab=details&scope=operator_sessions&range=24h");
  await waitFor(() => assert.equal(measures.runCycleTime.mock.calls.at(-1)?.[0].filter.scope, 2));
});

test("each Details section fails and retries on its own", async () => {
  const user = userEvent.setup();
  measures.modelBreakdown.mockRejectedValueOnce(new ConnectError("boom", Code.Internal));
  renderUsage("/usage?tab=details");
  const alert = await screen.findByRole("alert");
  assert.match(alert.textContent ?? "", /Models/);
  assert.ok(await screen.findByText("No profile (interactive)"), "the profile table is unaffected");
  await user.click(within(alert).getByRole("button", { name: /retry/i }));
  assert.ok(await screen.findByText("gpt-6.1-terra"));
});

test("with no category through the gate, the interventions section gives coverage and why, never no friction", async () => {
  renderUsage("/usage?tab=friction");
  assert.match((await screen.findByTestId("usage-interventions-coverage")).textContent ?? "", /120 operator messages, 90% classified, over 40\.0 agent-hours/);
  assert.match(screen.getByTestId("usage-interventions-empty").textContent ?? "", /None shown: no category has passed the accuracy gate yet.*not "no friction"/);
  assert.match(visibleText(screen.getByTestId("usage-friction")), /Operator interventionsnot shown/);
  assert.equal(screen.queryAllByTestId("usage-intervention-category").length, 0);
});

test("an admitted category shows its count, owner, rate, preceding work and an example linking to its run", async () => {
  const friction = makeUsageFriction();
  friction.interventions = makeAdmittedInterventions();
  measures.usageFriction.mockResolvedValue(friction);
  renderUsage("/usage?tab=friction");
  const row = await screen.findByTestId("usage-intervention-category");
  assert.match(row.textContent ?? "", /Quality miss.*3 repeat reports.*scientific-debugging.*20 · 0\.50 per agent-hour.*80 requests · 55 tool calls · 1\.2 M tokens/);
  const example = screen.getByTestId("usage-intervention-example");
  assert.match(example.textContent ?? "", /Still broken on mobile.*repeats an earlier report/);
  assert.equal(within(example).getByRole("link", { name: "open run" }).getAttribute("href"), "/runs/run-7");
  assert.equal(screen.queryByTestId("usage-interventions-empty"), null);
});

test("the pooled bucket reads as category unproven and labels every hint and example unproven", async () => {
  const friction = makeUsageFriction();
  friction.interventions = makePooledInterventions();
  measures.usageFriction.mockResolvedValue(friction);
  renderUsage("/usage?tab=friction");
  const row = await screen.findByTestId("usage-intervention-category");
  assert.match(visibleText(screen.getByTestId("usage-friction")), /Operator interventions18/, "the shown count is the API's");
  assert.match(row.textContent ?? "", /Operator friction \(category unproven\).*11 look like Quality miss \(unproven\): scientific-debugging.*4 look like Early stop \(unproven\): harness-goal-authoring.*18 · 0\.45 per agent-hour/);
  assert.match(screen.getByTestId("usage-intervention-example").textContent ?? "", /Quality miss \(unproven\): “It is not fixed”/);
});

test("friction rows label their columns inline for phones and show owners as code", async () => {
  renderUsage("/usage?tab=friction");
  const tools = await screen.findAllByTestId("usage-tool-finding");
  assert.equal(within(tools[0] as HTMLElement).getByText("vrooli").tagName, "CODE", "an owner reads as code");
  assert.match(tools[0]?.textContent ?? "", /Owner.*Seen.*Cost/);
});

test("the page owns its scroll container, since the app shell's main area does not scroll", async () => {
  renderUsage();
  const scroller = (await screen.findByTestId("usage-page")).parentElement;
  expect(scroller?.className).toMatch(/\bh-full\b/);
  expect(scroller?.className).toMatch(/\boverflow-y-auto\b/);
});
