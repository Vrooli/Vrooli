import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, test, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/index.js";
import { MoodSection, moodVerdict } from "./UsageMood.js";
import { valueDomain } from "./MoodChart.js";

const PROVEN = { gate_state: "proven", gate_model: "winnow:e4b", gate_role: "decide.local.accurate", split: "holdout", precision: "0.9", recall: "0.8", ece: "0.04", gold: "128", gold_positives: "15", temperature: "1.4", calibration: "calibrated", target_precision: "0.85", target_recall: "0.7", min_gold: "40" };
const UNMEASURED = { gate_state: "unmeasured", target_precision: "0.85", target_recall: "0.7", min_gold: "40" };

function gate(humanMessages: string | undefined, staleRuns?: string) {
  const coverage: Record<string, string> = { kind: "coverage", pack: "default", pack_version: "7", eligible_runs: "10", observed_runs: "4", coverage: "0.4" };
  if (humanMessages !== undefined) Object.assign(coverage, { human_text_runs: "3", human_messages: humanMessages });
  if (staleRuns !== undefined) coverage.stale_runs = staleRuns;
  return [
    coverage,
    { kind: "label", label: "operator.dissatisfied", group: "operator", scope: "message", state: "partial", friction: "false", observed: "4", eligible: "30", proven_observed: "3", ...PROVEN },
    { kind: "label", label: "operator.pleased", group: "operator", scope: "message", state: "unproven", friction: "false", observed: "4", eligible: "30", proven_observed: "0", ...UNMEASURED },
    { kind: "label", label: "operator.impatient", group: "operator", scope: "message", state: "not_observed", friction: "false", observed: "0", eligible: "30", proven_observed: "0", ...UNMEASURED },
    { kind: "label", label: "work.blocked_by_tool", group: "agent", scope: "run", state: "proven", friction: "true", observed: "4", eligible: "4", proven_observed: "4", ...PROVEN },
  ];
}

const SERIES: Record<string, unknown> = {
  "operator.dissatisfied": {
    label: "operator.dissatisfied", bucket: "day", window: { since: "2026-10-21T00:00:00Z", until: "2026-11-20T00:00:00Z" }, messages: 4, proven_messages: 3,
    points: [
      { start: "2026-11-10T00:00:00Z", messages: 2, proven_messages: 2, mean: 0.4, proven_mean: 0.5 },
      { start: "2026-11-11T00:00:00Z", messages: 2, proven_messages: 1, mean: 0.2, proven_mean: 0.1 },
    ],
  },
  "operator.pleased": {
    label: "operator.pleased", bucket: "day", window: { since: "2026-10-21T00:00:00Z", until: "2026-11-20T00:00:00Z" }, messages: 4, proven_messages: 0,
    points: [{ start: "2026-11-10T00:00:00Z", messages: 4, proven_messages: 0, mean: 0.7 }],
  },
  "operator.impatient": { label: "operator.impatient", bucket: "day", messages: 0, proven_messages: 0, points: [], note: "no message of human operator text has been observed" },
};

function serve(rows: unknown[]) {
  const urls: string[] = [];
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    urls.push(`${init?.method ?? "GET"} ${url}`);
    if (url.includes("/annotations/gate")) return new Response(JSON.stringify({ fields: rows }), { status: 200 });
    const label = new URL(url, "http://localhost").searchParams.get("label") ?? "";
    return new Response(JSON.stringify(SERIES[label]), { status: 200 });
  }));
  return urls;
}

afterEach(() => vi.unstubAllGlobals());

test("the mood card says it reads human-written messages only and how many there are", async () => {
  serve(gate("30"));
  renderWithProviders(<MoodSection />);
  const population = await screen.findByTestId("usage-mood-population");
  assert.match(population.textContent ?? "", /human-written messages only; workflow and delegation briefs are excluded/);
  assert.match(population.textContent ?? "", /30 messages in 3 runs/);
});

test("runs not yet classified are named so the population never silently undercounts", async () => {
  serve(gate("30", "5"));
  renderWithProviders(<MoodSection />);
  const population = await screen.findByTestId("usage-mood-population");
  assert.match(population.textContent ?? "", /30 messages in 3 runs\. 5 runs not yet classified/);
});

test("no note appears when every run is classified", async () => {
  serve(gate("30"));
  renderWithProviders(<MoodSection />);
  const population = await screen.findByTestId("usage-mood-population");
  assert.doesNotMatch(population.textContent ?? "", /not yet classified/);
});

test("only message-scope labels are drawn, each with its own gate and series", async () => {
  const urls = serve(gate("30"));
  renderWithProviders(<MoodSection />);
  const labels = await screen.findAllByTestId("mood-label");
  assert.deepEqual(labels.map((label) => label.getAttribute("aria-label")), ["operator.dissatisfied", "operator.pleased", "operator.impatient"]);
  await waitFor(() => assert.equal(screen.getAllByTestId("mood-verdict").length, 3));
  const [dissatisfied] = labels;
  assert.match(dissatisfied?.textContent ?? "", /partial/);
  assert.match(dissatisfied?.textContent ?? "", /Proven for winnow:e4b · decide\.local\.accurate · calibrated \(T 1\.4\)/);
  assert.match(dissatisfied?.textContent ?? "", /4 human-written messages, 3 from a proven model/);
  assert.equal(within(dissatisfied as HTMLElement).getAllByTestId("mood-dot-proven").length, 2);
  assert.ok(urls.some((url) => url.includes("/annotations/series?label=operator.dissatisfied&window=30d&bucket=day")), urls.join("\n"));
});

test("a label with no proven model draws only the unproven line and says there is no proven model yet", async () => {
  serve(gate("30"));
  renderWithProviders(<MoodSection />);
  await waitFor(() => assert.equal(screen.getAllByTestId("mood-verdict").length, 3));
  const pleased = screen.getAllByTestId("mood-label")[1] as HTMLElement;
  assert.match(within(pleased).getByTestId("mood-verdict").textContent ?? "", /No proven model yet: the dashed line is unproven evidence of mood, not a measurement/);
  assert.equal(within(pleased).queryByTestId("mood-line-proven"), null);
  assert.equal(within(pleased).queryByTestId("mood-dot-proven"), null);
  assert.match(pleased.textContent ?? "", /unproven/);
});

test("a label with no messages is unknown, not neutral, and draws nothing", async () => {
  serve(gate("30"));
  renderWithProviders(<MoodSection />);
  await waitFor(() => assert.equal(screen.getAllByTestId("mood-verdict").length, 3));
  const impatient = screen.getAllByTestId("mood-label")[2] as HTMLElement;
  assert.match(within(impatient).getByTestId("mood-verdict").textContent ?? "", /unknown, not neutral/);
  assert.equal(within(impatient).queryByTestId("mood-chart"), null);
});

test("the window toggle reads the shorter span from the API", async () => {
  const user = userEvent.setup();
  const urls = serve(gate("30"));
  renderWithProviders(<MoodSection />);
  await screen.findAllByTestId("mood-label");
  await user.click(screen.getByRole("button", { name: "7 days" }));
  await waitFor(() => assert.ok(urls.some((url) => url.includes("window=7d"))));
  assert.ok(urls.some((url) => url.includes("/annotations/gate?window=7d")), urls.join("\n"));
});

test("an annotation-free window says the number of messages is unknown", async () => {
  serve(gate(undefined));
  renderWithProviders(<MoodSection />);
  const population = await screen.findByTestId("usage-mood-population");
  assert.match(population.textContent ?? "", /the number of messages is unknown/);
});

test("the verdict never calls a window with unproven answers a measurement", () => {
  const points = [{ start: "2026-11-10T00:00:00Z", messages: 4, proven_messages: 3, mean: 0.4, proven_mean: 0.5 }];
  assert.match(moodVerdict({ messages: 4, proven_messages: 3, points }), /Solid line: messages read by a proven model/);
  assert.match(moodVerdict({ messages: 4, proven_messages: 4, points }), /read by a proven model/);
  const derived = [{ start: "2026-11-10T00:00:00Z", messages: 4, proven_messages: 4 }];
  assert.match(moodVerdict({ messages: 4, proven_messages: 4, points: derived }), /nothing to chart/);
});

test("the value axis is 0 to 1 and widens only when a label's own scale reaches outside it", () => {
  assert.deepEqual(valueDomain([{ start: "2026-11-10T00:00:00Z", messages: 1, proven_messages: 1, mean: 0.3, proven_mean: 0.4 }]), { low: 0, high: 1 });
  assert.deepEqual(valueDomain([{ start: "2026-11-10T00:00:00Z", messages: 1, proven_messages: 1, mean: -0.6, proven_mean: 0.4 }]), { low: -0.6, high: 1 });
});
