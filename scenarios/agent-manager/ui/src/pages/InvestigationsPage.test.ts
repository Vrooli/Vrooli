import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { InvestigationsPage } from "./InvestigationsPage.js";
import { renderWithProviders } from "../test-utils/index.js";

afterEach(() => { vi.restoreAllMocks(); });

test("InvestigationsPage labels an unavailable lifecycle read", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("no", { status: 503 })));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok((await screen.findAllByRole("alert")).length > 0);
});

test("InvestigationsPage handles a non-Error request rejection", async () => {
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue("offline"));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok((await screen.findAllByRole("alert")).length > 0);
});

test("InvestigationsPage empty lifecycle state has no accessibility violations", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ investigations: [] }))));
  const { container } = renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("No investigations yet"));
  await expectNoA11yViolations(container);
});

test("InvestigationsPage exposes superseded applicability for a stale diagnosis", async () => {
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

test("InvestigationsPage counts runs in the singular and shortens raw ids", async () => {
  const uuid = "6d9754e9-804a-48a9-8477-7cb972b960f7";
  vi.stubGlobal("fetch", vi.fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [{ investigationId: "inv", operationStatus: "completed", request: { subject: { kind: "run-set", ref: `set ${uuid}`, runIds: [uuid] } } }] }))));
  renderWithProviders(createElement(InvestigationsPage));
  const row = await screen.findByTestId("typed-investigation-inv");
  assert.match(row.textContent ?? "", /1 run(?!s)/);
  assert.doesNotMatch(row.textContent ?? "", /1 runs/);
  assert.match(row.textContent ?? "", /set 6d9754e9/);
  assert.equal((row.textContent ?? "").includes(uuid), false);
});

test("InvestigationsPage owns its scroll container, since the app shell's main area does not scroll", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("no", { status: 503 })));
  renderWithProviders(createElement(InvestigationsPage));
  const page = await screen.findByTestId("investigations-page");
  assert.match(page.className, /\bh-full\b/);
  assert.match(page.className, /\boverflow-y-auto\b/);
});
