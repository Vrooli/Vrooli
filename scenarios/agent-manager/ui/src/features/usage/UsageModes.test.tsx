import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { createElement } from "react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { UsagePage } from "./UsagePage.js";
import { starterText } from "./modes.js";
import { renderWithProviders } from "../../test-utils/index.js";
import { makeDetailsMeasures, makeUsageBurners, makeUsageCoverage, makeUsageForecast, makeUsageFriction, makeUsageModes, makeUsageRecommendations } from "../../test-utils/fixtures/usage.js";

const measures = vi.hoisted(() => ({
  usageForecast: vi.fn(), usageBurners: vi.fn(), usageRecommendations: vi.fn(), usageCoverage: vi.fn(), usageFriction: vi.fn(),
  listUsageModes: vi.fn(), startUsageMode: vi.fn(), endUsageMode: vi.fn(), extendUsageMode: vi.fn(), rebaseUsageModeRuns: vi.fn(),
  runSuccessRate: vi.fn(), runCost: vi.fn(), runCycleTime: vi.fn(), runStatusDistribution: vi.fn(), modelBreakdown: vi.fn(), profileBreakdown: vi.fn(),
}));
vi.mock("./measuresClient.js", () => ({ measuresClient: measures }));
vi.mock("./ManualImport.js", () => ({ ManualImport: () => createElement("div") }));

function serve(active: boolean) {
  measures.usageForecast.mockResolvedValue(makeUsageForecast());
  measures.usageBurners.mockImplementation(async ({ groupBy }: { groupBy: string }) => makeUsageBurners(groupBy));
  measures.usageRecommendations.mockResolvedValue(makeUsageRecommendations());
  measures.usageCoverage.mockResolvedValue(makeUsageCoverage());
  measures.usageFriction.mockResolvedValue(makeUsageFriction());
  measures.listUsageModes.mockResolvedValue(makeUsageModes(active));
  measures.startUsageMode.mockResolvedValue({});
  measures.endUsageMode.mockResolvedValue({ endedIds: ["um-test-1"] });
  measures.extendUsageMode.mockResolvedValue({});
  measures.rebaseUsageModeRuns.mockResolvedValue({ results: [] });
  for (const [name, answer] of Object.entries(makeDetailsMeasures())) measures[name as keyof typeof measures].mockResolvedValue(answer);
}

beforeEach(() => serve(false));
afterEach(() => Object.values(measures).forEach((fn) => fn.mockReset()));

test("a quota-lever card starts a mode: unmeasured range, real reach, future-only, and the call carries the card's runner", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  const [lever] = await screen.findAllByTestId("usage-action-start-mode");
  assert.equal((await screen.findAllByTestId("usage-recommendation")).length >= 2, true);
  // Only quota levers offer it: the tool-friction card has no start action.
  assert.equal(screen.getAllByTestId("usage-recommendation").filter((card) => within(card).queryByTestId("usage-action-start-mode")).length, 2);
  await user.click(lever as HTMLElement);
  const sheet = await screen.findByTestId("usage-mode-sheet");
  assert.match(within(sheet).getByTestId("mode-effect").textContent ?? "", /not measured yet[\s\S]*between 0 and about 12(\.0)? pts/i);
  assert.match(within(sheet).getByTestId("mode-reach-shares").textContent ?? "", /50% from operator sessions[\s\S]*20% the mode may not reach/);
  const running = within(sheet).getByLabelText("Also switch running sessions") as HTMLInputElement;
  assert.equal(running.disabled, false, "the switch control is live");
  assert.equal(running.checked, false, "nothing switches unless it is asked for");
  assert.doesNotMatch(sheet.textContent ?? "", /Coming later/);
  assert.equal((within(sheet).getByTestId("mode-duration-five_hour").closest("input") ?? within(sheet).getByTestId("mode-duration-five_hour") as HTMLInputElement).hasAttribute("disabled"), true, "Codex has no 5-hour reading");
  await user.click(within(sheet).getByTestId("mode-start"));
  await waitFor(() => assert.equal(measures.startUsageMode.mock.calls.length, 1));
  assert.deepEqual(measures.startUsageMode.mock.calls[0]?.[0], { preset: "one-tier-down", runner: "codex", role: "", profile: "", until: "weekly-reset" });
  await waitFor(() => assert.equal(screen.queryByTestId("usage-mode-sheet"), null));
});

test("a refused start (not a verified operator) says so and keeps the sheet open", async () => {
  const user = userEvent.setup();
  measures.startUsageMode.mockRejectedValue(new ConnectError("usage modes can be started or ended only by a verified operator", Code.PermissionDenied));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  await user.click(await screen.findByTestId("mode-open-sheet"));
  await user.click(await screen.findByTestId("mode-start"));
  assert.match((await screen.findByTestId("mode-start-error")).textContent ?? "", /Only a verified operator/);
  assert.ok(screen.getByTestId("usage-mode-sheet"));
});

test("the banner says what, until when and the burn since start, ends the mode, and extends the same mode without replacing it", async () => {
  const user = userEvent.setup();
  serve(true);
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  const banner = await screen.findByTestId("usage-mode-banner");
  assert.match(banner.textContent ?? "", /Lower tier for codex/);
  assert.match(banner.textContent ?? "", /by operator@example.com/);
  assert.match(within(banner).getByTestId("mode-until").textContent ?? "", /^until /);
  assert.match(within(banner).getByTestId("mode-burn").textContent ?? "", /2\.4.*Codex/);
  assert.equal(screen.queryByTestId("mode-open-sheet"), null, "with a mode on, the banner is the control");
  await user.click(within(banner).getByTestId("mode-extend"));
  await user.click(await within(banner).findByTestId("mode-extend-off"));
  await waitFor(() => assert.equal(measures.extendUsageMode.mock.calls.length, 1));
  assert.deepEqual(measures.extendUsageMode.mock.calls[0]?.[0], { id: "um-test-1", to: "off" });
  assert.equal(measures.startUsageMode.mock.calls.length, 0, "extending must not start a replacement mode (that resets burn since start)");
  await user.click(within(banner).getByTestId("mode-end"));
  await waitFor(() => assert.deepEqual(measures.endUsageMode.mock.calls[0]?.[0], { ids: ["um-test-1"] }));
});

test("two more hours extends by a duration the service understands", async () => {
  const user = userEvent.setup();
  serve(true);
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  const banner = await screen.findByTestId("usage-mode-banner");
  await user.click(within(banner).getByTestId("mode-extend"));
  await user.click(await within(banner).findByTestId("mode-extend-two_hours"));
  await waitFor(() => assert.deepEqual(measures.extendUsageMode.mock.calls[0]?.[0], { id: "um-test-1", to: "2h" }));
});

const switchResults = [
  { runId: "11111111-aaaa", status: "switched_at_next_resume", reason: "applies at the next resume", fromModel: "gpt-6.1-terra", toModel: "gpt-6-luna", estimatedRereadTokens: 90000n },
  { runId: "22222222-bbbb", status: "applies_on_next_launch", reason: "an interactive session keeps its model", fromModel: "gpt-6.1-terra", toModel: "", estimatedRereadTokens: 0n },
  { runId: "33333333-cccc", status: "unable", reason: "the run is mid-turn", fromModel: "gpt-6.1-terra", toModel: "", estimatedRereadTokens: 0n },
];

test("switching running work previews every run and its re-read cost, changes nothing until confirmed, then reports each run", async () => {
  const user = userEvent.setup();
  serve(true);
  measures.rebaseUsageModeRuns.mockResolvedValue({ results: switchResults });
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  const banner = await screen.findByTestId("usage-mode-banner");
  await user.click(within(banner).getByTestId("mode-switch-runs"));
  const panel = await within(banner).findByTestId("mode-switch-panel");
  await within(panel).findAllByTestId("mode-switch-result");
  assert.deepEqual(measures.rebaseUsageModeRuns.mock.calls.map((call) => call[0]), [{ modeId: "um-test-1", runIds: [], dryRun: true }], "opening the panel only previews");
  assert.match(within(panel).getByTestId("mode-switch-cost").textContent ?? "", /1 managed run can switch[\s\S]*re-reads about 90k tokens[\s\S]*Interactive sessions are not changed/);
  const rows = within(panel).getAllByTestId("mode-switch-result");
  assert.deepEqual(rows.map((row) => row.getAttribute("data-status")), ["switched_at_next_resume", "applies_on_next_launch", "unable"]);
  assert.match(rows[2]?.textContent ?? "", /mid-turn/);
  await user.click(within(panel).getByTestId("mode-switch-confirm"));
  await waitFor(() => assert.equal(measures.rebaseUsageModeRuns.mock.calls.length, 2));
  assert.deepEqual(measures.rebaseUsageModeRuns.mock.calls[1]?.[0], { modeId: "um-test-1", runIds: ["11111111-aaaa"], dryRun: false }, "only the runs that can switch are sent");
  await waitFor(() => assert.equal(within(panel).queryByTestId("mode-switch-confirm"), null));
});

test("a refused switch says why and leaves the panel usable", async () => {
  const user = userEvent.setup();
  serve(true);
  measures.rebaseUsageModeRuns.mockRejectedValue(new ConnectError("usage modes can be started or ended only by a verified operator", Code.PermissionDenied));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  const banner = await screen.findByTestId("usage-mode-banner");
  await user.click(within(banner).getByTestId("mode-switch-runs"));
  assert.match((await within(banner).findByTestId("mode-switch-error")).textContent ?? "", /Only a verified operator/);
});

test("the sheet's switch option leads to the preview after the mode starts, not to a silent switch", async () => {
  const user = userEvent.setup();
  measures.startUsageMode.mockResolvedValue({ mode: { id: "um-new-1" } });
  measures.rebaseUsageModeRuns.mockResolvedValue({ results: switchResults });
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  await user.click(await screen.findByTestId("mode-open-sheet"));
  const sheet = await screen.findByTestId("usage-mode-sheet");
  await user.click(within(sheet).getByLabelText("Also switch running sessions"));
  await user.click(within(sheet).getByTestId("mode-start"));
  await within(sheet).findByTestId("mode-started");
  await within(sheet).findAllByTestId("mode-switch-result");
  assert.deepEqual(measures.rebaseUsageModeRuns.mock.calls.map((call) => call[0]), [{ modeId: "um-new-1", runIds: [], dryRun: true }]);
});

test("a bare subject id is shortened, an email is shown as is", () => {
  assert.equal(starterText("operator@example.com"), "operator@example.com");
  assert.equal(starterText("d47163a3-1ba3-5e7b-96dd-e6f8fc450b3f"), "operator d471…");
  assert.equal(starterText(""), "an operator");
});

test("the window chart shades a mode that has already ended, not only the active ones", async () => {
  const { makeUsageModes: modes } = await import("../../test-utils/fixtures/usage.js");
  const at = (hours: number) => new Date(Date.now() + hours * 3_600_000).toISOString();
  measures.listUsageModes.mockResolvedValue(modes(false, [{ id: "um-ended", label: "Stretch the week", runner: "codex", startedAt: at(-20), endedAt: at(-10) }]));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  const detail = await screen.findByTestId("usage-window-detail");
  await waitFor(() => assert.equal(within(detail).queryAllByTestId("window-mode-band").length, 1));
  assert.match(within(detail).getByTestId("usage-window-chart").textContent ?? "", /Stretch the week was on/);
});

test("a period of another provider's runner is not shaded on this window", async () => {
  const { makeUsageModes: modes } = await import("../../test-utils/fixtures/usage.js");
  const at = (hours: number) => new Date(Date.now() + hours * 3_600_000).toISOString();
  measures.listUsageModes.mockResolvedValue(modes(false, [{ id: "um-claude", label: "Claude only", runner: "claude", startedAt: at(-20), endedAt: at(-10) }]));
  renderWithProviders(createElement(UsagePage), { initialEntries: ["/usage"] });
  const detail = await screen.findByTestId("usage-window-detail");
  assert.equal(within(detail).queryAllByTestId("window-mode-band").length, 0);
});
