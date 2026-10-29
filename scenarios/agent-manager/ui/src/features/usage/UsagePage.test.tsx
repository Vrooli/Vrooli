import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConnectError, Code } from "@connectrpc/connect";
import { createElement } from "react";
import { useLocation } from "react-router-dom";
import { afterEach, beforeEach, test, vi } from "vitest";
import { UsagePage } from "./UsagePage.js";
import { renderWithProviders } from "../../test-utils/index.js";
import { create } from "@bufbuild/protobuf";
import { UsageRecommendationGroupSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { BURNER_RUN_ID, CLAUDE_SCOPE, CODEX_SCOPE, FRICTION_FINGERPRINT, makeDetailsMeasures, makeUsageBurners, makeUsageCoverage, makeUsageForecast, makeUsageFriction, makeUsageRecommendations, USAGE_NOW } from "../../test-utils/fixtures/usage.js";

const measures = vi.hoisted(() => ({
  usageForecast: vi.fn(), usageBurners: vi.fn(), usageRecommendations: vi.fn(), usageCoverage: vi.fn(), usageFriction: vi.fn(),
  runSuccessRate: vi.fn(), runCost: vi.fn(), runCycleTime: vi.fn(), runStatusDistribution: vi.fn(), modelBreakdown: vi.fn(), profileBreakdown: vi.fn(),
}));
vi.mock("../stats/api/measuresClient.js", () => ({ measuresClient: measures }));
vi.mock("./ManualImport.js", () => ({ ManualImport: () => createElement("div", { "data-testid": "manual-import" }, "Manual import") }));

function serveFixtures() {
  measures.usageForecast.mockResolvedValue(makeUsageForecast());
  measures.usageBurners.mockImplementation(async ({ groupBy }: { groupBy: string }) => makeUsageBurners(groupBy));
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());
  measures.usageCoverage.mockResolvedValue(makeUsageCoverage());
  measures.usageFriction.mockResolvedValue(makeUsageFriction());
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

test("the focus band states the burn, what can be afforded, the cut and an honest consequence", async () => {
  renderUsage();
  const focus = await screen.findByTestId("usage-focus");
  const text = visibleText(focus);
  assert.match(text, /Codex · weekly runs out around/);
  assert.match(text, /Used33%/);
  assert.match(text, /Burning now \(24h\)1\.5%\/h/);
  assert.match(text, /Can afford until reset≤ 0\.46%\/h/);
  assert.match(text, /Needs a cut of69%/);
  assert.match(text, /The session action below saves≈ 9.0 pts/);
  assert.match(text, /still \d+ (days|hours) short of the reset|last until it resets/, "the consequence says whether the reset is reached");
});

test("actions are grouped per target: trimmed title, used and projected points, levers, and what each button does", async () => {
  const user = userEvent.setup();
  renderUsage();
  const cards = await within(await section("recommendations")).findAllByTestId("usage-recommendation");
  assert.equal(cards.length, 2);
  const first = within(cards[0] as HTMLElement);
  assert.ok(first.getByText("Research orchestrator"));
  assert.match(cards[0]?.textContent ?? "", /session 1e5904ed · still running/);
  assert.match(cards[0]?.textContent ?? "", /Used so far 28 pts.*At this rate ≈ 60 pts.*80% of the window's burn/);
  assert.match(cards[0]?.textContent ?? "", /Context.*−9.0 pts/);
  assert.equal(first.getByRole("link", { name: "Open session" }).getAttribute("href"), `/runs/${BURNER_RUN_ID}?tab=usage`);
  assert.equal(first.queryByRole("button", { name: /Dismiss/ }), null, "the API has no recommendation dismissal, so none is offered");
  assert.match((await section("recommendations")).textContent ?? "", /\+ 3 smaller actions for this window/, "server-folded items are counted");
  assert.equal(first.queryByTestId("usage-evidence"), null);
  await user.click(first.getByRole("button", { name: /Show evidence \(3 runs\)/ }));
  assert.ok(first.getByRole("link", { name: "sub-agent · 00b65371" }));
  assert.ok(first.getByText("web-console 1e5904ed"), "a label that already names its id is not repeated");
  assert.equal(within(await section("recommendations")).queryByTestId("usage-rules-not-fired"), null, "rule diagnostics are not in the main view");
});

test("where it went lists points and share per row, folds near-zero rows, and drills into runs", async () => {
  const user = userEvent.setup();
  renderUsage();
  const burners = within(await section("burners"));
  const rows = await burners.findAllByTestId("usage-burner");
  assert.equal(rows.length, 2);
  assert.match(rows[0]?.textContent ?? "", /Research orchestrator.*28\.0 · 84%/);
  assert.equal(burners.getByRole("link", { name: "Vega E1 Overview" }).getAttribute("href"), `/runs/${BURNER_RUN_ID}?tab=usage`);
  assert.match(burners.getByTestId("usage-burner-other").textContent ?? "", /77 smaller items, under 0\.1 pts each.*0\.3/);
  assert.match(burners.getByTestId("usage-burner-unattributed").textContent ?? "", /Unattributed \/ off-disk.*0\.4/);
  assert.match(burners.getByTestId("usage-metered-lines").textContent ?? "", /Web search · 3 calls · charge unknown/);
  await user.click(burners.getByText("How this is estimated"));
  assert.match((await section("burners")).textContent ?? "", /fitted on 48 slices, R² 0\.91/);
  assert.equal(within(await section("burners")).getByTestId("usage-rules-not-fired").textContent, "R-idle did not fire: no parked session consumed quota this window");
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

const episodes = vi.hoisted(() => ({ reportFrictionProblem: vi.fn(), dismissFrictionProblem: vi.fn() }));
vi.mock("@connectrpc/connect", async (original) => {
  const actual = await original<typeof import("@connectrpc/connect")>();
  return { ...actual, createClient: (service: { typeName: string }, ...rest: unknown[]) => (service.typeName === "agent_manager.v1.EpisodesService" ? episodes : (actual.createClient as (...args: unknown[]) => unknown)(service, ...rest)) };
});

test("friction lists open problems in plain words with owner, count and cost, and filters by status", async () => {
  const user = userEvent.setup();
  renderUsage("/usage?tab=friction");
  const rows = await screen.findAllByTestId("usage-friction-row");
  assert.equal(rows.length, 3, "only open problems by default");
  assert.equal(within(rows[0] as HTMLElement).queryByRole("link", { name: "Suggest a fix" }), null, "a command owner gets Report bug");
  const behaviour = within(rows[1] as HTMLElement);
  assert.equal(behaviour.queryByRole("button", { name: "Report bug" }), null, "agent behaviour files no bug");
  assert.equal(behaviour.getByRole("link", { name: "Suggest a fix" }).getAttribute("href"), `/usage?window=${encodeURIComponent("openai/primary/10080m")}#friction-${"a2".repeat(16)}`);
  assert.ok(behaviour.getByRole("button", { name: "Dismiss" }));
  assert.match(rows[0]?.textContent ?? "", /fails, then the agent falls back.*agent-manager.*18× · 3 runs.*3 m 5 s/);
  assert.match(rows[2]?.textContent ?? "", /120 calls · 4\.5 M tok/, "work after declaring done is not time lost");
  assert.doesNotMatch(rows[2]?.textContent ?? "", /2 h 30 m/);
  assert.doesNotMatch(visibleText(screen.getByTestId("usage-friction")), /Wall-clock lost.*2 h 30 m/, "handoff rows are excluded from the lost-time KPI");
  assert.doesNotMatch(document.body.textContent ?? "", new RegExp(FRICTION_FINGERPRINT), "the fingerprint is detail, not the headline");
  assert.match(visibleText(screen.getByTestId("usage-friction")), /Reported to owners1/);
  await user.click(screen.getByRole("button", { name: "Reported" }));
  assert.match(screen.getByTestId("usage-friction-row").textContent ?? "", /Reported · knw-1793200000000000001/);
  await user.click(screen.getByRole("button", { name: "Dismissed" }));
  assert.match(screen.getByTestId("usage-friction-row").textContent ?? "", /Dismissed/);
});

test("Report bug confirms inline, calls the API once, and shows the bug id when the list refreshes", async () => {
  const user = userEvent.setup();
  let answer: (value: unknown) => void = () => undefined;
  episodes.reportFrictionProblem.mockImplementation(() => new Promise((resolve) => { answer = resolve; }));
  renderUsage("/usage?tab=friction");
  const [row] = await screen.findAllByTestId("usage-friction-row");
  const first = within(row as HTMLElement);
  await user.click(first.getByRole("button", { name: "Report bug" }));
  assert.equal(episodes.reportFrictionProblem.mock.calls.length, 0, "the first click only asks");
  assert.ok(first.getByText(/File a bug with agent-manager\?/));
  await user.click(first.getByRole("button", { name: "Confirm" }));
  await waitFor(() => assert.equal(first.getByRole("button", { name: "Working…" }).hasAttribute("disabled"), true));
  assert.equal(episodes.reportFrictionProblem.mock.calls[0]?.[0].fingerprint, FRICTION_FINGERPRINT);
  const reported = makeUsageFriction();
  reported.problems[0]!.status = "reported";
  reported.problems[0]!.bugId = "knw-42";
  measures.usageFriction.mockResolvedValue(reported);
  answer({ fingerprint: FRICTION_FINGERPRINT, status: "reported", bugId: "knw-42" });
  await user.click(await screen.findByRole("button", { name: "Reported" }));
  assert.ok(await screen.findByText("Reported · knw-42"));
});

test("Dismiss confirms inline and a failed call says so on the row", async () => {
  const user = userEvent.setup();
  episodes.dismissFrictionProblem.mockRejectedValue(new Error("api down"));
  renderUsage("/usage?tab=friction");
  const [row] = await screen.findAllByTestId("usage-friction-row");
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

test("friction plans cut across sessions, so the band adds only session plans and the card says it is not added", async () => {
  const recommendations = makeUsageRecommendations();
  recommendations.groups.push(create(UsageRecommendationGroupSchema, { rank: 3, targetKind: "friction", crossCut: true, targetId: "ff".repeat(16), targetLabel: "An agent reads a file again", ...CODEX_SCOPE, combinedSavedPoints: 50, levers: [{ rule: "R-friction", lever: "friction", savedPoints: 50, marginalPoints: 50, confidence: "medium", action: "Fix the recurring problem" }] }));
  measures.usageRecommendations.mockResolvedValue(recommendations);
  renderUsage();
  const cards = await screen.findAllByTestId("usage-recommendation");
  assert.equal(cards.length, 3);
  assert.match(cards[2]?.textContent ?? "", /also in the sessions above, not added to them/);
  assert.match(visibleText(await screen.findByTestId("usage-focus")), /The session action below saves≈ 9.0 pts/, "the 50 pts of friction are not in the sum");
});

test("friction rows label their columns inline for phones, show command owners as code, and keep one short line for handoffs", async () => {
  renderUsage("/usage?tab=friction");
  const rows = await screen.findAllByTestId("usage-friction-row");
  assert.equal(within(rows[0] as HTMLElement).getByText("agent-manager").tagName, "CODE", "a command owner reads as code");
  assert.ok(within(rows[1] as HTMLElement).getByText("Agent behaviour"));
  assert.match(rows[0]?.textContent ?? "", /Owner.*Seen.*Cost/);
  const handoff = within(rows[2] as HTMLElement).getByText("120 calls · 4.5 M tok");
  assert.match(handoff.getAttribute("title") ?? "", /tool calls and 4\.5 M tokens of work after declaring done/);
  assert.match(rows[0]?.textContent ?? "", /typically 10 s each time/, "a typical time of a second or more is shown once");
});
