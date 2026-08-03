import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test } from "vitest";
import { MeasureFrame } from "../../src/features/stats/components/measure/MeasureFrame.js";
import { renderWithProviders } from "../../src/test-utils/renderWithProviders.js";

const available = {
  executedQuery: "SELECT 1",
  definitionId: "throughput.run_volume",
  validity: { state: "available" as const, reason: "enough evidence", sampleSize: 12, largestFingerprintShare: 0.2 },
  provenance: {
    sourceTable: "agent_runs",
    windowStart: "2026-07-01T00:00:00Z",
    windowEnd: "2026-07-02T00:00:00Z",
    rowCount: 12,
    appliedFilters: [{ field: "runner_type", value: "codex" }],
  },
};

const definition = {
  id: "throughput.run_volume",
  counts: "terminal runs",
  numerator: "terminal runs",
  denominator: "none",
  sourceTable: "agent_runs",
  limitation: "historical read model coverage",
};

test("measure frame exposes loading, error, and data-quality states", () => {
  const { rerender } = renderWithProviders(createElement(MeasureFrame, { label: "Volume", loading: true, children: "value", testId: "measure" }));
  assert.equal(screen.getByTestId("measure").className.includes("animate-pulse"), true);

  rerender(createElement(MeasureFrame, { label: "Volume", error: "backend unavailable", children: "value", testId: "measure" }));
  assert.equal(screen.getByRole("alert").textContent, "Volume: backend unavailable");

  rerender(createElement(MeasureFrame, { label: "Volume", children: "value", testId: "measure" }));
  assert.match(screen.getByText("Not enough data yet").textContent ?? "", /Not enough data yet/);

  rerender(createElement(MeasureFrame, { label: "Volume", result: { ...available, validity: { ...available.validity, state: "unavailable" as const, reason: "outside read-model history" } }, children: "value", testId: "measure" }));
  assert.equal(screen.getByTestId("measure").textContent?.includes("outside read-model history"), true);

  rerender(createElement(MeasureFrame, { label: "Volume", result: { ...available, validity: { ...available.validity, state: "unreliable" as const, reason: "small sample" } }, children: "value", testId: "measure" }));
  assert.equal(screen.getByTestId("measure").textContent?.includes("small sample"), true);
});

test("measure frame shows children, definition, and provenance evidence", () => {
  renderWithProviders(createElement(MeasureFrame, { label: "Volume", result: available, definition, children: "measured value", testId: "measure" }));
  assert.equal(screen.getByTestId("measure").textContent?.includes("measured value"), true);
  fireEvent.click(screen.getByLabelText("Definition for Volume"));
  fireEvent.click(screen.getByLabelText("Provenance for Volume"));
  assert.equal(screen.getByText("historical read model coverage").textContent, "Limitation: historical read model coverage");
  assert.equal(screen.getByText("Filters:").parentElement?.textContent, "Filters: runner_type=codex");

  const noEvidence = { ...available, provenance: { ...available.provenance, appliedFilters: [] } };
  const { rerender } = renderWithProviders(createElement(MeasureFrame, { label: "Volume", result: noEvidence, children: "measured value", testId: "measure-no-filters" }));
  rerender(createElement(MeasureFrame, { label: "Volume", result: { ...available, definitionId: "" }, definition: { ...definition, limitation: "" }, children: "measured value", testId: "measure-no-limitation" }));
  assert.equal(screen.getByTestId("measure-no-limitation").textContent?.includes("Limitation:"), false);
});
