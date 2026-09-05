import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { TokenAttributionBreakdown } from "../../src/features/stats/components/breakdown/TokenAttributionBreakdown.js";
import { useTokenAttribution } from "../../src/features/stats/hooks/useTokenAttribution.js";
import type { DurableTokenAttribution } from "../../src/features/stats/api/statsClient.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

vi.mock("../../src/features/stats/hooks/useTokenAttribution.js", () => ({ useTokenAttribution: vi.fn() }));

function result(data: DurableTokenAttribution): ReturnType<typeof useTokenAttribution> {
  return { data, isLoading: false, error: null } as ReturnType<typeof useTokenAttribution>;
}

function data(view: DurableTokenAttribution["view"], groupBy: DurableTokenAttribution["groupBy"]): DurableTokenAttribution {
  return {
    groupBy,
    view,
    estimatedTokenShare: 0.25,
    rows: [{ groupBy, value: "filesystem", callCount: 4, totalTokens: 1500, estimatedTokens: 375, estimatedTokenShare: 0.25, p50FootprintTokens: 100, p95FootprintTokens: 600, maxFootprintTokens: 800 }, { groupBy, value: "unattributed", callCount: 1, totalTokens: 200, estimatedTokens: 0, estimatedTokenShare: 0, p50FootprintTokens: 0, p95FootprintTokens: 0, maxFootprintTokens: 0 }],
    executedQuery: "SELECT durable token attribution",
    validity: { state: "available", reason: "", sampleSize: 5, largestFingerprintShare: 0 },
    definitionId: "friction.token_attribution",
  };
}

test("TokenAttributionBreakdown renders all views, dimensions, shares, residual, and footprint percentiles", async () => {
  const user = userEvent.setup();
  vi.mocked(useTokenAttribution).mockImplementation(({ groupBy, view }) => result(data(view, groupBy)));
  renderWithProviders(createElement(TokenAttributionBreakdown));

  assert.ok(screen.getByText("Footprint"));
  assert.ok(screen.getByText("Intrinsic payload added by each invocation; use it to find commands worth shrinking."));
  assert.ok(screen.getByText("Estimated share"));
  assert.ok(screen.getByText("P95 footprint"));
  assert.ok(screen.getByText("unattributed"));
  assert.ok(screen.getByText("25.0%"));

  await user.selectOptions(screen.getByLabelText("Token attribution view"), "residency");
  assert.ok(screen.getByText("Payload multiplied by the turns it remains in context; compaction attenuation is an approximation."));
  assert.equal(screen.queryByText("P95 footprint"), null);

  await user.selectOptions(screen.getByLabelText("Token attribution group by"), "target_scenario_operation");
  await user.selectOptions(screen.getByLabelText("Token attribution view"), "incurred");
  assert.ok(screen.getByText("Provider-reported usage associated with the invocation; use it as the accounting view."));
  assert.ok(vi.mocked(useTokenAttribution).mock.calls.some(([options]) => options.groupBy === "target_scenario_operation" && options.view === "incurred"));
});

test("TokenAttributionBreakdown renders an explicit empty state", () => {
  const empty = data("footprint", "capability");
  empty.rows = [];
  vi.mocked(useTokenAttribution).mockReturnValue(result(empty));
  renderWithProviders(createElement(TokenAttributionBreakdown));
  assert.ok(screen.getByText("No token attribution data available for the selected window."));
});
