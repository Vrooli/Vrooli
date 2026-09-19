import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { RunClassBreakdown } from "../../src/features/stats/components/breakdown/RunClassBreakdown.js";
import { renderWithProviders } from "@vrooli/api-base/testing";

afterEach(() => vi.unstubAllGlobals());

test("RunClassBreakdown fetches and presents executed, imported, and interactive run classes", async () => {
  const fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    json: async () => ({
      classes: [
        { class: "executed", run_count: 4, success_count: 3, failed_count: 1 },
        { class: "imported", run_count: 2, success_count: 2, failed_count: 0 },
        { class: "interactive", run_count: 1, success_count: 1, failed_count: 0 },
      ],
      executed_denominator: 4,
      missing_model_runs: 1,
      missing_model_rate: 0.25,
      excluded_classes: ["imported", "interactive"],
    }),
  });
  vi.stubGlobal("fetch", fetchMock);

  renderWithProviders(createElement(RunClassBreakdown));

  await waitFor(() => assert.ok(screen.getByTestId("run-class-breakdown")));
  assert.ok(screen.getByText("Executed measures exclude imported and interactive runs."));
  assert.ok(screen.getByText("denominator 4"));
  assert.ok(screen.getByText("executed"));
  assert.ok(screen.getByText("2 runs · 2 success · 0 failed"));
  assert.ok(screen.getByText("Residual missing-model rate: 25.00% (1 runs)"));
  assert.equal(fetchMock.mock.calls[0]?.[0], "/api/v1/stats/run-classes?preset=7d");
});

test("RunClassBreakdown exposes loading and API error states", async () => {
  let resolve: ((value: unknown) => void) | undefined;
  vi.stubGlobal("fetch", vi.fn(() => new Promise((done) => { resolve = done; })));

  renderWithProviders(createElement(RunClassBreakdown));
  assert.ok(screen.getByText("Loading run classes…"));
  resolve?.({ ok: false, status: 503 });
  await waitFor(() => assert.ok(screen.getByText("Run classes: Run-class API error 503")));
});
