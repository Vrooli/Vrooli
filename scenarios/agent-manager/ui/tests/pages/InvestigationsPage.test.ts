import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { InvestigationsPage } from "../../src/pages/InvestigationsPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

afterEach(() => vi.restoreAllMocks());

test("InvestigationsPage renders ranked signals, detail evidence, rollups, and availability", async () => {
  const fetchMock = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ availability: { state: "available" }, signals: [{ fingerprint: "fp", occurrences: 2, distinctRuns: 2, summedCostMs: 50, confidence: "high", representativeRunIds: ["run-1"] }] }))).mockResolvedValueOnce(new Response(JSON.stringify({ investigations: [] }))).mockResolvedValueOnce(new Response(JSON.stringify({ episodes: [{ episodeId: "episode-1", pattern: "stall", causeScope: "run-execution", severity: "recurring", turns: 2, tokens: 3, wallClockMs: 50, suspectedOwnerScenario: "target", suspectedOwnerCommand: "target run", ownerConfidence: "receipt-verified", evidenceEventIds: ["event-1"] }] }))).mockResolvedValueOnce(new Response(JSON.stringify({ ledgerAvailability: { state: "unobserved", reason: "none" }, projectionAvailability: { state: "policy_absent" }, ledgerTargetRollups: [{ targetScenario: "target", calls: 1, failures: 0, medianDurationMs: 4 }] })));
  vi.stubGlobal("fetch", fetchMock);
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("fp")); assert.ok(screen.getByTestId("availability-available")); fireEvent.click(screen.getByText("fp"));
  assert.ok(await screen.findByText("Episode detail")); assert.ok(screen.getByTestId("availability-unobserved")); assert.ok(screen.getByTestId("availability-policy_absent"));
  fireEvent.click(screen.getByRole("button", { name: "Close" }));
  assert.equal(screen.queryByText("Episode detail"), null);
  await waitFor(() => assert.equal(fetchMock.mock.calls.length, 4));
});

test("InvestigationsPage labels unavailable cohort evidence", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("no", { status: 503 })));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByTestId("availability-unavailable"));
});

test("InvestigationsPage leaves a signal without a representative run closed", async () => {
  const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ availability: { state: "degraded" }, signals: [{ fingerprint: "empty", occurrences: 1, distinctRuns: 0, summedCostMs: 0, confidence: "medium", representativeRunIds: [] }] })));
  vi.stubGlobal("fetch", fetchMock);
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("empty"));
  assert.ok(screen.getByTestId("availability-degraded"));
  fireEvent.click(screen.getByText("empty"));
  assert.equal(screen.queryByText("Episode detail"), null);
  assert.equal(fetchMock.mock.calls.length, 2);
});

test("InvestigationsPage handles a non-Error request rejection", async () => {
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue("offline"));
  renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByTestId("availability-unavailable"));
});

test("InvestigationsPage empty lifecycle state has no accessibility violations", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ availability: { state: "available" }, signals: [] }))));
  const { container } = renderWithProviders(createElement(InvestigationsPage));
  assert.ok(await screen.findByText("No caller-neutral investigations have been admitted."));
  await expectNoA11yViolations(container);
});

test("InvestigationsPage exposes superseded applicability for a stale diagnosis", async () => {
  vi.stubGlobal("fetch", vi.fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ availability: { state: "available" }, signals: [] })))
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
  assert.ok(await screen.findByText("superseded"));
  assert.ok(screen.getByText("complete · no_intervention"));
});
