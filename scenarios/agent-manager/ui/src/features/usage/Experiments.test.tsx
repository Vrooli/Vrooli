import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { createElement } from "react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { Route, Routes } from "react-router-dom";
import { LearnSetupPage } from "./LearnSetupPage.js";
import { UsagePage } from "./UsagePage.js";
import { renderWithProviders } from "../../test-utils/index.js";
import { makeUsageExperiment, makeUsageOutcomes, makeUsageParity } from "../../test-utils/fixtures/usage.js";
import { create } from "@bufbuild/protobuf";
import { UsageOutcomesResponseSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";

const measures = vi.hoisted(() => ({
  usageOutcomes: vi.fn(), usageParity: vi.fn(), allMeasureDefinitions: vi.fn(), usageCompare: vi.fn(),
  usageExperiment: vi.fn(), planUsageExperiment: vi.fn(), startUsageExperiment: vi.fn(), proposeRoleDefault: vi.fn(),
  usageForecast: vi.fn(), usageBurners: vi.fn(), usageRecommendations: vi.fn(), usageCoverage: vi.fn(), listUsageModes: vi.fn(),
}));
vi.mock("./measuresClient.js", () => ({ measuresClient: measures }));

const roleRows = (by: string) => by === "role"
  ? create(UsageOutcomesResponseSchema, { by, rows: [
    { key: "code.default", runner: "codex", units: 68n }, { key: "code.economy.delivery", runner: "codex", units: 9n },
    { key: "code.review", runner: "codex", units: 12n }, { key: "operator_session", runner: "codex", units: 31n },
  ] })
  : makeUsageOutcomes("observational", by);

const plan = (days: number, extra: Record<string, unknown> = {}) => ({
  plan: { unitsPerDay: 0.3, costCv: 1.2, pilotUnits: 20n, effect: 0.5, power: 0.8, minAcceptedPerArm: 80n, minEffect: 0.3, reworkMargin: 0.2, detectableIn90Days: 0, acceptedShare: 0.4, completedShare: 0.9, powerAtLook: 0.82, sameAtLook: 0.1, detect: { reached: true, days, lowDays: days * 0.5, highDays: days * 2, perArm: 200n }, same: { reached: false }, completedOnly: { reached: true, days: days / 2, lowDays: days * 0.25, highDays: days, perArm: 89n }, ...extra },
});

beforeEach(() => {
  measures.usageOutcomes.mockImplementation(async ({ by }: { by: string }) => roleRows(by));
  measures.usageParity.mockResolvedValue(makeUsageParity());
  measures.allMeasureDefinitions.mockResolvedValue({ definitions: [] });
  measures.usageCoverage.mockResolvedValue({ roots: [] });
  measures.usageExperiment.mockResolvedValue(makeUsageExperiment());
  measures.planUsageExperiment.mockImplementation(async ({ role }: { role: string }) => plan(role === "code.default" ? 28 : 200));
  measures.startUsageExperiment.mockResolvedValue({ mode: { id: "um-learn-1" } });
  measures.proposeRoleDefault.mockResolvedValue({ recommendation: { decisionId: "decision-1", decisionState: "pending" } });
});
afterEach(() => Object.values(measures).forEach((fn) => fn.mockReset()));

function openSetup() {
  return renderWithProviders(createElement(Routes, null,
    createElement(Route, { path: "/usage/learn", element: createElement(LearnSetupPage) }),
    createElement(Route, { path: "/usage", element: createElement("div", { "data-testid": "evidence-landed" }) })), { initialEntries: ["/usage/learn"] });
}

test("setup lists managed roles with their pace, leaves exempt roles out, and shows no estimate until a role is chosen", async () => {
  openSetup();
  const roles = await screen.findAllByTestId("learn-role");
  assert.equal(roles.length, 3, "sessions are not roles");
  assert.match(roles[0]?.textContent ?? "", /code\.default[\s\S]*about 16 a week/);
  const review = screen.getByTestId("learn-role-code.review") as HTMLInputElement;
  assert.equal(review.disabled, true);
  assert.match(roles[2]?.textContent ?? "", /Left out/);
  assert.match(screen.getByTestId("learn-headline").textContent ?? "", /Choose a role/);
  assert.equal((screen.getByTestId("learn-start") as HTMLButtonElement).disabled, true);
  assert.equal(measures.planUsageExperiment.mock.calls.length, 0, "no plan is read for a role nobody chose");
});

test("choosing roles shows the expected time before anything starts, decided by the slowest role, then starts one comparison per role", async () => {
  const user = userEvent.setup();
  openSetup();
  await user.click(await screen.findByTestId("learn-role-code.default"));
  await waitFor(() => assert.match(screen.getByTestId("learn-headline").textContent ?? "", /Probably 2 weeks to 8 weeks/));
  await user.click(screen.getByTestId("learn-role-code.economy.delivery"));
  await waitFor(() => assert.match(screen.getByTestId("learn-headline").textContent ?? "", /Probably 3 months to 14 months|Probably 100 days|months/));
  assert.equal(screen.getAllByTestId("learn-role-estimate").length, 2);
  assert.match(screen.getAllByTestId("learn-role-estimate")[0]?.textContent ?? "", /read once, when each side holds 80 accepted outcomes.*40% of its finished launches record one.*read cheaper about 82%.*only finished would take.*never a verdict/);
  assert.equal(measures.startUsageExperiment.mock.calls.length, 0, "reading the estimate starts nothing");
  assert.match(screen.getByTestId("learn-cost-warning").textContent ?? "", /about half of these launches use the cheaper model, whatever its quality/);
  await user.click(screen.getByTestId("learn-start"));
  await waitFor(() => assert.equal(measures.startUsageExperiment.mock.calls.length, 2));
  assert.deepEqual(measures.startUsageExperiment.mock.calls.map((call) => call[0]), [
    { runner: "codex", role: "code.default", splitPercent: 50, sessions: false, until: "off" },
    { runner: "codex", role: "code.economy.delivery", splitPercent: 50, sessions: false, until: "off" },
  ]);
  await screen.findByTestId("evidence-landed");
});

test("the split and the session opt-in reach the request, and the warning follows the split", async () => {
  const user = userEvent.setup();
  openSetup();
  await user.click(await screen.findByTestId("learn-role-code.default"));
  await user.click(screen.getByTestId("learn-split-20"));
  await user.click(screen.getByTestId("learn-sessions"));
  assert.match(screen.getByTestId("learn-cost-warning").textContent ?? "", /about a fifth/);
  await user.click(screen.getByTestId("learn-start"));
  await waitFor(() => assert.equal(measures.startUsageExperiment.mock.calls.length, 1));
  assert.deepEqual(measures.startUsageExperiment.mock.calls[0]?.[0], { runner: "codex", role: "code.default", splitPercent: 20, sessions: true, until: "off" });
  assert.deepEqual(measures.planUsageExperiment.mock.calls.at(-1)?.[0], { runner: "codex", role: "code.default", splitPercent: 20 });
});

test("a refused start says only a verified operator can start it and stays on the page", async () => {
  const user = userEvent.setup();
  measures.startUsageExperiment.mockRejectedValue(new ConnectError("usage modes can be started or ended only by a verified operator, not an agent", Code.PermissionDenied));
  openSetup();
  await user.click(await screen.findByTestId("learn-role-code.default"));
  await user.click(screen.getByTestId("learn-start"));
  assert.match((await screen.findByTestId("learn-start-error")).textContent ?? "", /Only a verified operator can start Learn mode/);
  assert.equal(screen.queryByTestId("evidence-landed"), null);
});

test("an unreadable role list says nothing can be chosen, and no quiet roles read as an empty state", async () => {
  measures.usageOutcomes.mockRejectedValue(new Error("the run ledger is unavailable"));
  openSetup();
  assert.ok(await screen.findByTestId("learn-roles-error"));
});

test("no managed role in the last 30 days is an empty state, not an error", async () => {
  measures.usageOutcomes.mockResolvedValue(create(UsageOutcomesResponseSchema, { by: "role", rows: [] }));
  openSetup();
  assert.match((await screen.findByTestId("learn-no-roles")).textContent ?? "", /nothing to compare yet/);
});

test("the evidence tab shows the design from the assignment basis, the verdict, the arms and the exact policy change that nothing applied", async () => {
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  const card = await screen.findByTestId("experiment-card");
  assert.equal(within(card).getByTestId("experiment-design").textContent, "Randomized");
  assert.equal(within(card).getByTestId("experiment-verdict").textContent, "Cheaper looks better");
  assert.match(within(card).getByTestId("experiment-sentence").textContent ?? "", /For code\.default, the cheaper model looks better/);
  assert.match(within(card).getByTestId("experiment-stats").textContent ?? "", /0\.50x \(90% range 0\.42x to 0\.60x\), permutation test p=0\.001/);
  assert.match(within(card).getByTestId("experiment-arm-challenger").textContent ?? "", /2 exceptions/);
  const recommendation = within(card).getByTestId("experiment-recommendation");
  assert.match(recommendation.textContent ?? "", /Nothing is applied until a verified person approves, and even then Agent Manager does not edit resource files/);
  assert.match(within(recommendation).getByTestId("experiment-policy-file").textContent ?? "", /resources\/codex\/model-policy\.json/);
  assert.match(within(recommendation).getByTestId("experiment-policy-ops").textContent ?? "", /replace \/roles\/code\.default\/model/);
  assert.equal(measures.proposeRoleDefault.mock.calls.length, 0, "showing a recommendation asks for nothing");
});

test("asking for approval raises one proposal for that comparison, and a refusal is explained", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  await user.click(await screen.findByTestId("experiment-propose"));
  await waitFor(() => assert.deepEqual(measures.proposeRoleDefault.mock.calls[0]?.[0], { modeId: "um-learn-1" }));
  assert.match((await screen.findByTestId("experiment-propose")).textContent ?? "", /Approval requested/);

});

test("a refused approval request is explained", async () => {
  const user = userEvent.setup();
  measures.proposeRoleDefault.mockRejectedValue(new ConnectError("not an agent", Code.PermissionDenied));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  await user.click(await screen.findByTestId("experiment-propose"));
  assert.match((await screen.findByTestId("experiment-propose-error")).textContent ?? "", /Only a verified operator can ask for this approval/);
});

test("a comparison without enough data says what is still needed and how long it should take, and offers no recommendation", async () => {
  measures.usageExperiment.mockResolvedValue(makeUsageExperiment("not_enough_data"));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  const card = await screen.findByTestId("experiment-card");
  assert.equal(within(card).getByTestId("experiment-verdict").textContent, "Not enough data");
  const needed = within(card).getByTestId("experiment-needed").textContent ?? "";
  assert.match(needed, /Still needed: 66 more accepted outcomes/);
  assert.match(needed, /over 3 years, at least 15 months|over 3 years/);
  assert.match(needed, /variance assumed/);
  assert.equal(within(card).queryByTestId("experiment-recommendation"), null);
});

test("with no comparison the panel says so and links to setup; an unreadable one says it is unavailable", async () => {
  measures.usageExperiment.mockResolvedValue({ experiments: [] });
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  assert.ok(await screen.findByTestId("experiments-none"));
  assert.equal(screen.getByTestId("evidence-learn-setup").getAttribute("href"), "/usage/learn");
});

test("an unreadable comparison list does not hide the rest of the evidence", async () => {
  measures.usageExperiment.mockRejectedValue(new Error("down"));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  assert.ok(await screen.findByTestId("experiments-unavailable"));
  assert.ok(await screen.findByTestId("evidence-conclusion"));
});

test("a paused comparison that includes sessions shows its state, unknown ranges as unknown, and a decision already made", async () => {
  measures.usageExperiment.mockResolvedValue(makeUsageExperiment("about_the_same", {
    paused: true, pausedReason: "forecast: openai quota is forecast to run out", sessionsOptIn: true, active: false,
    costRatio: { available: false }, plan: undefined,
    incumbent: { arm: "incumbent", units: 1n, accepted: 0n },
    challenger: { arm: "challenger", units: 2n, accepted: 1n, exceptions: 1n },
    recommendation: { available: true, role: "code.default", toModel: "gpt-6-luna", fromModel: "gpt-6.1-terra", policyOps: [], policyNote: "role has no resource entry", decisionId: "decision-1", decisionState: "approved", decisionNote: "approved by operator@example.com. Not applied." },
  }));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  const card = await screen.findByTestId("experiment-card");
  assert.match(card.textContent ?? "", /Paused: forecast: openai quota is forecast to run out/);
  assert.match(card.textContent ?? "", /your own sessions included in day blocks/);
  assert.match(within(card).getByTestId("experiment-arm-incumbent").textContent ?? "", /unknown per accepted outcome.*rework unknown/);
  assert.match(within(card).getByTestId("experiment-arm-challenger").textContent ?? "", /1 exception$/);
  assert.equal(within(card).queryByTestId("experiment-stats"), null);
  assert.equal(within(card).queryByTestId("experiment-policy-ops"), null);
  assert.match(within(card).getByTestId("experiment-recommendation").textContent ?? "", /role has no resource entry/);
  assert.match(within(card).getByTestId("experiment-decision").textContent ?? "", /Decision approved\. approved by operator@example\.com\. Not applied\./);
  assert.equal(within(card).queryByTestId("experiment-propose"), null);
});

test("an ended comparison with a higher-looks-better verdict gives no recommendation", async () => {
  measures.usageExperiment.mockResolvedValue(makeUsageExperiment("cheaper_looks_better", { active: false, verdict: "higher_looks_better", recommendation: { available: false } }));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  const card = await screen.findByTestId("experiment-card");
  assert.match(card.textContent ?? "", /Ended/);
  assert.equal(within(card).getByTestId("experiment-verdict").textContent, "Higher looks better");
  assert.equal(within(card).queryByTestId("experiment-recommendation"), null);
});

test("a role that only ever finishes is told no verdict is reachable, not given a long wait", async () => {
  const user = userEvent.setup();
  measures.planUsageExperiment.mockImplementation(async () => plan(100, { acceptedShare: 0, completedShare: 1, pilotUnits: 68n, detect: { reached: false }, powerAtLook: 0 }));
  openSetup();
  await user.click(await screen.findByTestId("learn-role-code.default"));
  await waitFor(() => assert.match(screen.getByTestId("learn-headline").textContent ?? "", /No verdict is reachable yet/));
  const text = screen.getByTestId("learn-role-estimate").textContent ?? "";
  assert.match(text, /no verdict is reachable from this role's work/);
  assert.match(text, /only finished would take.*never a verdict/);
});

test("per-role estimates say when a role is refused, has no plan, or can detect only large drops, and an unmeasured spread is flagged", async () => {
  const user = userEvent.setup();
  measures.planUsageExperiment.mockImplementation(async ({ role }: { role: string }) => {
    if (role === "code.default") return { refusal: "role code.default is exempt" };
    return plan(100, { unitsPerDay: 0, varianceAssumed: true, pilotUnits: 3n, detectableIn90Days: 0.4 });
  });
  openSetup();
  await user.click(await screen.findByTestId("learn-role-code.default"));
  await user.click(screen.getByTestId("learn-role-code.economy.delivery"));
  await waitFor(() => assert.equal(screen.getAllByTestId("learn-role-estimate").length, 2));
  const text = screen.getAllByTestId("learn-role-estimate").map((el) => el.textContent ?? "").join("\n");
  assert.match(text, /code\.default: role code\.default is exempt/);
  assert.match(text, /With no finished launches/);
  assert.match(text, /A 40% or larger drop could be seen within 90 days/);
  assert.match(text, /spread was assumed because only 3 launches/);
});

test("a design that is not randomized is not labelled so, and the heading makes no claim of fairness", async () => {
  measures.usageExperiment.mockResolvedValue(makeUsageExperiment("not_enough_data", { design: "observational" }));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  const card = await screen.findByTestId("experiment-card");
  assert.equal(within(card).getByTestId("experiment-design").textContent, "Observational");
  const panel = screen.getByTestId("evidence-experiments");
  assert.doesNotMatch(panel.textContent ?? "", /fair comparison/);
  assert.equal(within(panel).queryByText("Randomized comparisons"), null);
});

test("inconclusive and withheld verdicts are named, not shown as not enough data", async () => {
  for (const [verdict, label] of [["inconclusive", "Inconclusive"], ["withheld", "Withheld"]] as const) {
    measures.usageExperiment.mockResolvedValue(makeUsageExperiment(verdict));
    const { unmount } = renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
    const card = await screen.findByTestId("experiment-card");
    assert.equal(within(card).getByTestId("experiment-verdict").textContent, label);
    assert.doesNotMatch(within(card).getByTestId("experiment-sentence").textContent ?? "", /not enough data/);
    assert.equal(within(card).queryByTestId("experiment-needed"), null);
    unmount();
  }
});

test("a comparison whose acceptance rests on the agent's own verdict says acceptance is partly assumed", async () => {
  const base = makeUsageExperiment().experiments[0];
  measures.usageExperiment.mockResolvedValue(makeUsageExperiment("cheaper_looks_better", { challenger: { ...base?.challenger, assumedAccepted: 40n }, incumbent: { ...base?.incumbent, assumedAccepted: 10n } }));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage?tab=evidence"] });
  assert.match((await screen.findByTestId("experiment-assumed")).textContent ?? "", /partly assumed: 50 of 199 accepted outcomes rest only on the agent's own "complete" verdict/);
});
