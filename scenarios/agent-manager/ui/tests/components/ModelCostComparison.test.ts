import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ModelCostComparison } from "../../src/components/ModelCostComparison.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const comparisonHook = vi.fn();
vi.mock("../../src/features/stats/hooks/useModelCostComparison.js", () => ({
  useModelCostComparison: (input: unknown) => comparisonHook(input),
}));
vi.mock("../../src/components/dialogs/EditPricingDialog.js", () => ({
  EditPricingDialog: ({ model, onClose, onPricingUpdated }: { model: string | null; onClose: () => void; onPricingUpdated: () => void }) => model ? createElement("button", { onClick: () => { onPricingUpdated(); onClose(); } }, `edit ${model}`) : null,
}));

const totals = { inputTokens: 10, outputTokens: 20, cacheCreationTokens: 1, cacheReadTokens: 2, totalCostUsd: 0.1, webSearchRequests: 1, serverToolUseRequests: 2, models: [], serviceTiers: [], events: 2 };
comparisonHook.mockReturnValue({ data: null, isLoading: false, error: null, refetch: vi.fn() });

test("ModelCostComparison hides zero-event runs and renders loading, error, empty, and comparison states", async () => {
  const { rerender } = renderWithProviders(createElement(ModelCostComparison, { costTotals: { ...totals, events: 0 }, actualModel: "gpt-5" }));
  assert.equal(screen.queryByText("Compare Models"), null);

  comparisonHook.mockReturnValue({ data: null, isLoading: true, error: null, refetch: vi.fn() });
  rerender(createElement(ModelCostComparison, { costTotals: totals, actualModel: "gpt-5" }));
  assert.ok(screen.getByText("Compare Models"));

  comparisonHook.mockReturnValue({ data: null, isLoading: false, error: new Error("offline"), refetch: vi.fn() });
  rerender(createElement(ModelCostComparison, { costTotals: totals, actualModel: "gpt-5" }));
  assert.ok(screen.getByText("Failed to load comparison: offline"));

  comparisonHook.mockReturnValue({ data: { comparisons: [] }, isLoading: false, error: null, refetch: vi.fn() });
  rerender(createElement(ModelCostComparison, { costTotals: totals, actualModel: "gpt-5" }));
  assert.ok(screen.getByText("No model pricing data available for comparison"));
});

test("ModelCostComparison switches lists, formats differences, edits canonical models, and refreshes pricing", async () => {
  const user = userEvent.setup();
  const refetch = vi.fn();
  comparisonHook.mockReturnValue({ data: { comparisons: [
    { model: "very-long-model-name-that-is-truncated", canonicalModel: "canonical", estimatedCostUsd: 0.2, differenceUsd: -0.1, differencePercent: 10, isActualModel: true },
    { model: "other", estimatedCostUsd: 0.3, differenceUsd: 0.2, differencePercent: -3, isActualModel: false },
  ] }, isLoading: false, error: null, refetch });
  renderWithProviders(createElement(ModelCostComparison, { costTotals: totals, actualModel: "gpt-5" }));
  assert.ok(screen.getByText("(actual)"));
  assert.ok(screen.getByText("-$0.1000"));
  assert.ok(screen.getByText("-3.0%"));
  await user.click(screen.getByRole("button", { name: "Recent" }));
  assert.equal(comparisonHook.mock.calls.at(-1)?.[0].request.modelList, "recent");
  await user.click(screen.getAllByTitle("Edit pricing")[0]!);
  await user.click(screen.getByRole("button", { name: "edit canonical" }));
  assert.equal(refetch.mock.calls.length, 1);
});
