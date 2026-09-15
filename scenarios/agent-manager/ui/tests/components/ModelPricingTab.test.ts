import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { ModelPricingTab } from "../../src/components/dialogs/SettingsDialog/ModelPricingTab.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const hooks = vi.hoisted(() => ({
  model: vi.fn(), settings: vi.fn(), update: vi.fn(), cache: vi.fn(), refresh: vi.fn(), recalculate: vi.fn(), overrides: vi.fn(), setOverride: vi.fn(), deleteOverride: vi.fn(),
}));
vi.mock("../../src/hooks/usePricing.js", () => ({
  useModelPricing: hooks.model, usePricingSettings: hooks.settings, useUpdatePricingSettings: hooks.update,
  usePricingCacheStatus: hooks.cache, useRefreshAllPricing: hooks.refresh, useRecalculateModelPricing: hooks.recalculate,
  useModelOverrides: hooks.overrides, useSetOverride: hooks.setOverride, useDeleteOverride: hooks.deleteOverride,
}));

const model = { model: "gpt-5", canonicalName: "gpt-5", inputPricePer1M: 2, outputPricePer1M: 8, cacheReadPricePer1M: 0.5, inputSource: "provider_api", outputSource: "manual_override", cacheReadSource: "historical_average", fetchedAt: "2026-07-29T00:00:00Z" };
function setup(overrides: Record<string, unknown> = {}) {
  const refetch = vi.fn(); const refetchSettings = vi.fn(); const refetchCache = vi.fn();
  hooks.model.mockReturnValue({ data: [model], loading: false, error: null, refetch });
  hooks.settings.mockReturnValue({ data: { historicalAverageDays: 7, providerCacheTtlSeconds: 3600 }, loading: false, refetch: refetchSettings });
  hooks.update.mockReturnValue({ updateSettings: vi.fn(async () => undefined) });
  hooks.cache.mockReturnValue({ data: { totalModels: 1, expiredCount: 1, providers: [{ provider: "openai", modelCount: 1, lastFetchedAt: "2026-07-29T00:00:00Z", isStale: true }] }, loading: false, refetch: refetchCache });
  hooks.refresh.mockReturnValue({ refreshAll: vi.fn(async () => undefined) });
  hooks.recalculate.mockReturnValue({ recalculate: vi.fn(async () => undefined) });
  hooks.overrides.mockReturnValue({ data: [], loading: false, refetch: vi.fn() });
  hooks.setOverride.mockReturnValue({ setOverride: vi.fn(async () => undefined), loading: false });
  hooks.deleteOverride.mockReturnValue({ deleteOverride: vi.fn(async () => undefined), loading: false });
  Object.assign(hooks, overrides);
  return { refetch, refetchSettings, refetchCache };
}
afterEach(() => vi.resetAllMocks());

test("ModelPricingTab filters, sorts, refreshes, and persists pricing settings", async () => {
  const user = userEvent.setup();
  const calls = setup();
  renderWithProviders(createElement(ModelPricingTab));
  assert.ok(screen.getByText("gpt-5"));
  assert.ok(screen.getByText("Stale"));
  await user.type(screen.getByPlaceholderText("Search models..."), "missing");
  assert.ok(screen.getByText("No models match your filters."));
  await user.clear(screen.getByPlaceholderText("Search models..."));
  await user.click(screen.getAllByText("Provider")[0]!);
  assert.ok(screen.getByText("gpt-5"));
  await user.click(screen.getByRole("button", { name: "Clear" }));
  await user.click(screen.getByRole("button", { name: "Input" }));
  await user.click(screen.getByTitle("Refresh pricing"));
  await user.click(screen.getByRole("button", { name: "Refresh All Pricing" }));
  await user.clear(screen.getByLabelText("Historical Average Period (days)"));
  await user.type(screen.getByLabelText("Historical Average Period (days)"), "14");
  await user.click(screen.getByRole("button", { name: "Save Settings" }));
  assert.ok(calls.refetch.mock.calls.length >= 2);
  assert.equal(calls.refetchCache.mock.calls.length, 1);
});

test("ModelPricingTab presents loading, failure, and empty pricing states", () => {
  setup(); hooks.model.mockReturnValue({ data: [], loading: true, error: null, refetch: vi.fn() });
  const loading = renderWithProviders(createElement(ModelPricingTab));
  assert.ok(screen.getByText("Loading pricing data..."));
  loading.unmount();
  const retry = vi.fn(); hooks.model.mockReturnValue({ data: [], loading: false, error: "provider unavailable", refetch: retry });
  renderWithProviders(createElement(ModelPricingTab));
  assert.ok(screen.getByText(/Error loading pricing: provider unavailable/));
  screen.getByRole("button", { name: "Retry" }).click();
  assert.equal(retry.mock.calls.length, 1);
});

test("ModelPricingTab orders incomplete model data and handles dialog keyboard editing", async () => {
  const user = userEvent.setup();
  const alpha = { ...model, model: "alpha", canonicalName: "alpha", inputPricePer1M: undefined, outputPricePer1M: undefined, cacheReadPricePer1M: undefined, cacheReadSource: undefined, fetchedAt: "" };
  const zulu = { ...model, model: "zulu", canonicalName: "zulu", inputPricePer1M: 1, outputPricePer1M: 3, cacheReadPricePer1M: 0, fetchedAt: "2026-07-30T00:00:00Z" };
  setup();
  hooks.model.mockReturnValue({ data: [zulu, alpha], loading: false, error: null, refetch: vi.fn() });
  const setOverride = vi.fn(async () => undefined);
  hooks.setOverride.mockReturnValue({ setOverride, loading: false });
  renderWithProviders(createElement(ModelPricingTab));

  await user.click(screen.getByRole("button", { name: "Input" }));
  assert.equal(screen.getAllByRole("button", { name: "Edit pricing overrides" })[0]?.closest("tr")?.textContent?.includes("alpha"), true);
  await user.click(screen.getAllByRole("button", { name: "Edit pricing overrides" })[0]!);
  assert.ok(screen.getByText("Cache Creation"));
  assert.ok(screen.getByText("Web Search"));
  await user.click(screen.getAllByTitle("Set override")[1]!);
  const price = screen.getByPlaceholderText("0.00");
  await user.type(price, "2.25");
  await user.keyboard("{Escape}");
  assert.equal(screen.queryByPlaceholderText("0.00"), null);
  assert.equal(screen.queryByRole("heading", { name: "Edit Pricing: alpha" }), null);
  await user.click(screen.getAllByRole("button", { name: "Edit pricing overrides" })[0]!);
  await user.click(screen.getAllByTitle("Set override")[1]!);
  await user.type(screen.getByPlaceholderText("0.00"), "2.25");
  await user.keyboard("{Enter}");
  await vi.waitFor(() => assert.deepEqual(setOverride.mock.calls, [["alpha", { component: "output_tokens", priceUsd: 0.00000225 }]]));
});

test("ModelPricingTab edits, saves, and clears a component-level manual override", async () => {
  const user = userEvent.setup();
  setup();
  const setOverride = vi.fn(async () => undefined); const deleteOverride = vi.fn(async () => undefined); const refetchOverrides = vi.fn();
  hooks.overrides.mockReturnValue({ data: [{ component: "input_tokens", priceUsd: 0.000003 }], loading: false, refetch: refetchOverrides });
  hooks.setOverride.mockReturnValue({ setOverride, loading: false });
  hooks.deleteOverride.mockReturnValue({ deleteOverride, loading: false });
  renderWithProviders(createElement(ModelPricingTab));
  await user.click(screen.getByTitle("Edit pricing overrides"));
  assert.ok(screen.getByRole("heading", { name: "Edit Pricing: gpt-5" }));
  await user.click(screen.getByTitle("Edit override"));
  const inputs = screen.getAllByPlaceholderText("0.00");
  await user.clear(inputs[0]!); await user.type(inputs[0]!, "4.5");
  await user.click(screen.getByTitle("Save override"));
  await vi.waitFor(() => assert.deepEqual(setOverride.mock.calls, [["gpt-5", { component: "input_tokens", priceUsd: 0.0000045 }]]));
  await user.click(screen.getByTitle("Clear override"));
  await vi.waitFor(() => assert.deepEqual(deleteOverride.mock.calls, [["gpt-5", "input_tokens"]]));
  assert.ok(refetchOverrides.mock.calls.length >= 2);
});

test("ModelPricingTab supports canonical-name search, source toggling, empty data, and every sortable price column", async () => {
  const user = userEvent.setup();
  const alpha = { ...model, model: "alpha-display", canonicalName: "canonical-alpha", inputPricePer1M: 4, outputPricePer1M: 1, cacheReadPricePer1M: 3, cacheReadSource: "provider_api", fetchedAt: "2026-07-28T00:00:00Z" };
  const zulu = { ...model, model: "zulu-display", canonicalName: "canonical-zulu", inputPricePer1M: 1, outputPricePer1M: 4, cacheReadPricePer1M: 0.5, fetchedAt: "2026-07-30T00:00:00Z", inputSource: "historical_average", outputSource: "historical_average", cacheReadSource: "historical_average" };
  setup();
  hooks.model.mockReturnValue({ data: [alpha, zulu], loading: false, error: null, refetch: vi.fn() });
  renderWithProviders(createElement(ModelPricingTab));

  const search = screen.getByPlaceholderText("Search models...");
  await user.type(search, "canonical-alpha");
  assert.ok(screen.getByText("alpha-display"));
  assert.equal(screen.queryByText("zulu-display"), null);
  await user.clear(search);
  await user.click(screen.getAllByText("Historical")[0]!);
  assert.ok(screen.getByText("zulu-display"));
  assert.equal(screen.queryByText("alpha-display"), null);
  await user.click(screen.getAllByText("Historical")[0]!);
  assert.ok(screen.getByText("alpha-display"));

  for (const column of ["Output", "Cache Read", "Fetched", "Model"]) {
    await user.click(screen.getByRole("button", { name: column }));
    await user.click(screen.getByRole("button", { name: column }));
  }

  await user.type(search, "not-present");
  assert.ok(screen.getByText("No models match your filters."));
});

test("ModelPricingTab shows the honest empty-data and unavailable side-panel states", () => {
  setup();
  hooks.model.mockReturnValue({ data: [], loading: false, error: null, refetch: vi.fn() });
  hooks.settings.mockReturnValue({ data: null, loading: true, refetch: vi.fn() });
  hooks.cache.mockReturnValue({ data: null, loading: true, refetch: vi.fn() });
  renderWithProviders(createElement(ModelPricingTab));
  assert.ok(screen.getByText("No pricing data available."));
  assert.equal(screen.queryByText("Total Models:"), null);
  assert.equal((screen.getByRole("button", { name: "Save Settings" }) as HTMLButtonElement).disabled, true);
  assert.equal((screen.getByRole("button", { name: "Refresh All Pricing" }) as HTMLButtonElement).disabled, true);
});

test("ModelPricingTab exposes fresh cache metadata, recalculates an individual model, and keeps override details loading-safe", async () => {
  const user = userEvent.setup();
  const recalculate = vi.fn(async () => undefined);
  const calls = setup();
  hooks.cache.mockReturnValue({
    data: { totalModels: 2, expiredCount: 0, providers: [{ provider: "anthropic", modelCount: 2, lastFetchedAt: "", isStale: false }] },
    loading: false,
    refetch: calls.refetchCache,
  });
  hooks.recalculate.mockReturnValue({ recalculate });
  hooks.overrides.mockReturnValue({ data: undefined, loading: true, refetch: vi.fn() });
  renderWithProviders(createElement(ModelPricingTab));

  assert.ok(screen.getByText("anthropic"));
  assert.equal(screen.queryByText("Stale"), null);
  await user.click(screen.getByTitle("Refresh pricing"));
  await vi.waitFor(() => assert.deepEqual(recalculate.mock.calls, [["gpt-5"]]));
  assert.equal(calls.refetch.mock.calls.length, 1);

  await user.click(screen.getByTitle("Edit pricing overrides"));
  assert.ok(screen.getByText("Loading..."));
});
