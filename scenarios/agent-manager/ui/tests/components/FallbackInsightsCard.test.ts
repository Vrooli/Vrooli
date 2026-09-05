import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { FallbackInsightsCard } from "../../src/features/stats/components/operational/FallbackInsightsCard.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const state = vi.hoisted(() => ({ hook: vi.fn() }));
vi.mock("../../src/features/stats/hooks/useOperationalStats.js", () => ({ useFallbackInsights: state.hook }));
afterEach(() => vi.resetAllMocks());

test("FallbackInsightsCard distinguishes loading, error, absent, and insufficient evidence honestly", () => {
  state.hook.mockReturnValue({ data: null, isLoading: true, error: null });
  const loading = renderWithProviders(createElement(FallbackInsightsCard)); assert.ok(screen.getByText("Loading…")); loading.unmount();
  state.hook.mockReturnValue({ data: null, isLoading: false, error: new Error("metrics unavailable") });
  const failed = renderWithProviders(createElement(FallbackInsightsCard)); assert.ok(screen.getByText(/metrics unavailable/)); failed.unmount();
  state.hook.mockReturnValue({ data: null, isLoading: false, error: null });
  const absent = renderWithProviders(createElement(FallbackInsightsCard)); assert.equal(absent.container.innerHTML, ""); absent.unmount();
  state.hook.mockReturnValue({ data: { runner_attempts: 2, model_attempts: 1 }, isLoading: false, error: null });
  renderWithProviders(createElement(FallbackInsightsCard)); assert.ok(screen.getByTestId("fallback-insufficient"));
});

test("FallbackInsightsCard presents bounded runner/model transitions, reasons, presets, and history", () => {
  state.hook.mockReturnValue({ data: {
    runner_attempts: 4, runner_exhausted: 1, model_attempts: 3, model_exhausted: 2,
    runner_chain_depth: { "2": 3, "1": 1, bogus: 99 }, model_chain_depth: { "3": 3 },
    runner_by_pair: [{ from: "codex", to: "claude", reason: "timeout", count: 3 }], model_by_pair: [{ from: "primary", to: "fallback", count: 2 }],
    runner_by_reason: { timeout: 3, unavailable: 1 }, model_by_reason: { rate_limit: 2 }, model_by_preset: { quality: 2, "": 1 }, event_count: 7, history: { history_days: 1.2 },
  }, isLoading: false, error: null });
  renderWithProviders(createElement(FallbackInsightsCard));
  assert.ok(screen.getByTestId("fallback-insights")); assert.ok(screen.getByText("Runner fallbacks (4 attempted, 1 exhausted)"));
  assert.ok(screen.getByText("codex → claude (timeout)")); assert.ok(screen.getByText("primary → fallback"));
  assert.ok(screen.getByText("rate_limit")); assert.ok(screen.getByText("(none)"));
  assert.match(screen.getByTestId("fallback-summary-footer").textContent ?? "", /7 fallback events over 1 day/);
});
