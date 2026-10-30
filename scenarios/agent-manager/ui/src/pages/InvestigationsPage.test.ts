import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { InvestigationsPage } from "./InvestigationsPage.js";
import { renderWithProviders } from "../test-utils/index.js";

const episodeCohort = vi.hoisted(() => vi.fn());
vi.mock("../features/stats/api/measuresClient.js", () => ({ measuresClient: { episodeCohort } }));
const cohort = (state: string, signals: Array<Record<string, unknown>> = []) => episodeCohort.mockResolvedValue({ availabilityState: state, availabilityReason: "", signals });

afterEach(() => { vi.restoreAllMocks(); episodeCohort.mockReset(); });

test("InvestigationsPage renders ranked signals, detail evidence, rollups, and availability", async () => {
  cohort("available", [{ fingerprint: "fp", occurrences: 2n, distinctRuns: 2n, summedCostMs: 50n, confidence: "high", representativeRunIds: ["run-1"] }]);
  const fetchMock = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [] }))).mockResolvedValueOnce(new Response(JSON.stringify({ episodes: [{ episodeId: "episode-1", pattern: "stall", causeScope: "run-execution", severity: "recurring", turns: 2, tokens: 3, wallClockMs: 50, suspectedOwnerScenario: "target", suspectedOwnerCommand: "target run", ownerConfidence: "receipt-verified", evidenceEventIds: ["event-1"] }] }))).mockResolvedValueOnce(new Response(JSON.stringify({ ledgerAvailability: { state: "unobserved", reason: "none" }, projectionAvailability: { state: "policy_absent" }, ledgerTargetRollups: [{ targetScenario: "target", calls: 1, failures: 0, medianDurationMs: 4 }] })));
  vi.stubGlobal("fetch", fetchMock);
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("fp")); fireEvent.click(screen.getByText("fp"));
  assert.ok(await screen.findByText("Episode detail")); assert.ok(screen.getByTestId("availability-unobserved")); assert.ok(screen.getByTestId("availability-policy_absent"));
  fireEvent.click(screen.getByRole("button", { name: "Close" }));
  assert.equal(screen.queryByText("Episode detail"), null);
  await waitFor(() => assert.equal(fetchMock.mock.calls.length, 3));
});

test("InvestigationsPage labels unavailable cohort evidence", async () => {
  episodeCohort.mockRejectedValue(new Error("unavailable"));
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("no", { status: 503 })));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok((await screen.findAllByRole("alert")).length > 0);
});

test("InvestigationsPage leaves a signal without a representative run closed", async () => {
  cohort("degraded", [{ fingerprint: "empty", occurrences: 1n, distinctRuns: 0n, summedCostMs: 0n, confidence: "medium", representativeRunIds: [] }]);
  const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ investigations: [] })));
  vi.stubGlobal("fetch", fetchMock);
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("empty"));
  assert.ok(screen.getByTestId("availability-degraded"));
  fireEvent.click(screen.getByText("empty"));
  assert.equal(screen.queryByText("Episode detail"), null);
  assert.equal(fetchMock.mock.calls.length, 1);
});

test("InvestigationsPage handles a non-Error request rejection", async () => {
  episodeCohort.mockRejectedValue("offline");
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue("offline"));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok((await screen.findAllByRole("alert")).length > 0);
});

test("InvestigationsPage empty lifecycle state has no accessibility violations", async () => {
  cohort("available");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ investigations: [] }))));
  const { container } = renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("No investigations yet"));
  await expectNoA11yViolations(container);
});

test("InvestigationsPage exposes superseded applicability for a stale diagnosis", async () => {
  cohort("available");
  vi.stubGlobal("fetch", vi.fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [{
      investigationId: "investigation-stale",
      operationStatus: "completed",
      request: { subject: { kind: "run-set", ref: "run-stale", runIds: ["run-stale"] } },
      result: {
        diagnosis: { condition: "complete", disposition: "no_intervention" },
        coverage: [{ plane: "run_state", state: "complete" }],
        applicability: { state: "superseded" },
      },
    }] }))));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("Superseded"));
  assert.ok(screen.getByText("Complete · No intervention"));
});

test("InvestigationsPage reads statuses, diagnosis and coverage in words and links back to Health", async () => {
  cohort("available");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [
    { investigationId: "inv-failed", operationStatus: "failed", request: { subject: { kind: "workflow", ref: "smoke" } }, updatedAt: "2026-01-01T00:00:00Z" },
    { investigationId: "inv-done", operationStatus: "completed", request: { subject: { kind: "run-set", ref: "set-1", runIds: ["a", "b"] } },
      result: { diagnosis: { condition: "stalled_run", disposition: "needs_follow_up" }, coverage: [{ plane: "run_state", state: "partial_coverage" }, { plane: "events", state: "complete" }], applicability: { state: "current" } } },
    { investigationId: "inv-bare", operationStatus: "cancelled", request: {} },
  ] }))));
  renderWithProviders(createElement(InvestigationsPage));
  const done = await screen.findByTestId("typed-investigation-inv-done");
  assert.match(done.textContent ?? "", /Completed/);
  assert.match(done.textContent ?? "", /Run set: set-1/);
  assert.match(done.textContent ?? "", /2 runs/);
  assert.match(done.textContent ?? "", /Stalled run · Needs follow up/);
  assert.match(done.textContent ?? "", /Run state: Partial coverage · Events: Complete/);
  assert.match(done.textContent ?? "", /Current/);
  const failed = screen.getByTestId("typed-investigation-inv-failed");
  assert.match(failed.textContent ?? "", /Failed/);
  assert.match(failed.textContent ?? "", /Pending/);
  assert.match(failed.textContent ?? "", /Not yet available/);
  assert.match(screen.getByTestId("typed-investigation-inv-bare").textContent ?? "", /Subject: inv-bare/);
  assert.equal(screen.getByRole("link", { name: /Health/ }).getAttribute("href"), "/observability");
  assert.equal(screen.queryByText(/stalled_run|partial_coverage/), null, "raw codes never reach the page");
});

test("InvestigationsPage keeps the page usable when the episode detail cannot be read", async () => {
  cohort("available", [{ fingerprint: "fp-broken", occurrences: 1n, distinctRuns: 1n, summedCostMs: 3000n, confidence: "low", representativeRunIds: ["run-9"] }]);
  vi.stubGlobal("fetch", vi.fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [] })))
    .mockResolvedValue(new Response(JSON.stringify({ message: "episodes unavailable" }), { status: 503 })));
  renderWithProviders(createElement(InvestigationsPage));
  fireEvent.click(await screen.findByText("fp-broken"));
  assert.ok(await screen.findByText(/Episode detail/));
  assert.equal(screen.queryByText("Episode detail", { selector: "h2" }), null);
  assert.match(document.body.textContent ?? "", /3\.0 s/);
  assert.match(document.body.textContent ?? "", /Low/);
});

test("InvestigationsPage counts runs in the singular, shortens raw ids and labels evidence links in words", async () => {
  const uuid = "6d9754e9-804a-48a9-8477-7cb972b960f7";
  cohort("available", [{ fingerprint: "fp", occurrences: 1n, distinctRuns: 1n, summedCostMs: 10n, confidence: "high", representativeRunIds: ["run-1"] }]);
  vi.stubGlobal("fetch", vi.fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [{ investigationId: "inv", operationStatus: "completed", request: { subject: { kind: "run-set", ref: `set ${uuid}`, runIds: [uuid] } } }] })))
    .mockResolvedValueOnce(new Response(JSON.stringify({ episodes: [{ episodeId: "e", pattern: "stall", causeScope: "run", ownerConfidence: "high", turns: 1, tokens: 1, wallClockMs: 1, evidenceEventIds: ["0f3c9a1e-aaaa-bbbb-cccc-1234567890ab", "second"], suspectedOwnerScenario: `owner ${uuid}` }] })))
    .mockResolvedValueOnce(new Response(JSON.stringify({ ledgerAvailability: { state: "available" }, projectionAvailability: { state: "available" }, ledgerTargetRollups: [] }))));
  renderWithProviders(createElement(InvestigationsPage));
  const row = await screen.findByTestId("typed-investigation-inv");
  assert.match(row.textContent ?? "", /1 run(?!s)/);
  assert.doesNotMatch(row.textContent ?? "", /1 runs/);
  assert.match(row.textContent ?? "", /set 6d9754e9/);
  assert.equal((row.textContent ?? "").includes(uuid), false);
  fireEvent.click(await screen.findByText("fp"));
  const first = await screen.findByRole("link", { name: "Evidence 1" });
  assert.match(first.getAttribute("href") ?? "", /^\/runs\/run-1\?event=0f3c9a1e/);
  assert.ok(first.className.includes("min-h-11"));
  assert.ok(screen.getByRole("link", { name: "Evidence 2" }));
  assert.equal(document.body.textContent?.includes(uuid), false);
});

test("InvestigationsPage owns its scroll container, since the app shell's main area does not scroll", async () => {
  episodeCohort.mockRejectedValue(new Error("unavailable"));
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("no", { status: 503 })));
  renderWithProviders(createElement(InvestigationsPage));
  const page = await screen.findByTestId("investigations-page");
  assert.match(page.className, /\bh-full\b/);
  assert.match(page.className, /\boverflow-y-auto\b/);
});

test("InvestigationsPage shows recurring friction a page at a time", async () => {
  cohort("available", Array.from({ length: 25 }, (_, i) => ({ fingerprint: `fp-${i}`, occurrences: 2n, distinctRuns: 2n, summedCostMs: 50n, confidence: "high", representativeRunIds: ["run-1"] })));
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ investigations: [] }))));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("fp-19"));
  assert.equal(screen.queryByText("fp-20"), null);
  fireEvent.click(screen.getByTestId("friction-signals-show-more"));
  assert.ok(await screen.findByText("fp-24"));
  assert.equal(screen.queryByTestId("friction-signals-show-more"), null);
});

test("InvestigationsPage says how many quarantined-pattern episodes the cohort hides", async () => {
  episodeCohort.mockResolvedValue({ availabilityState: "available", availabilityReason: "", hiddenQuarantinedEpisodes: 7n, signals: [{ fingerprint: "fp", occurrences: 2n, distinctRuns: 2n, summedCostMs: 50n, confidence: "high", representativeRunIds: ["run-1"] }] });
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ investigations: [] }))));
  renderWithProviders(createElement(InvestigationsPage));
  assert.match((await screen.findByTestId("hidden-quarantined-episodes")).textContent ?? "", /7 episodes of quarantined patterns hidden/);
});

test("InvestigationsPage calls an episode's count redundant calls and hides absent tokens and time", async () => {
  cohort("available", [{ fingerprint: "fp", occurrences: 1n, distinctRuns: 1n, summedCostMs: 10n, confidence: "high", representativeRunIds: ["run-1"] }]);
  vi.stubGlobal("fetch", vi.fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [] })))
    .mockResolvedValueOnce(new Response(JSON.stringify({ episodes: [{ episodeId: "e", pattern: "stall", causeScope: "run", ownerConfidence: "high", turns: 3, tokens: 0, wallClockMs: 0, evidenceEventIds: ["x"] }] })))
    .mockResolvedValueOnce(new Response(JSON.stringify({ ledgerAvailability: { state: "available" }, projectionAvailability: { state: "available" }, ledgerTargetRollups: [] }))));
  renderWithProviders(createElement(InvestigationsPage));
  fireEvent.click(await screen.findByText("fp"));
  const article = (await screen.findByText(/3 redundant calls/)).textContent ?? "";
  assert.equal(article, "3 redundant calls");
  assert.doesNotMatch(document.body.textContent ?? "", /requests,|0 tokens/);
});
