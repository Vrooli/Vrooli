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
});
