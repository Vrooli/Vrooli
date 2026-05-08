import { screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { beforeEach, describe, it, expect, vi } from "vitest";
import { FallbackInsightsCard } from "../../src/features/stats/components/operational/FallbackInsightsCard.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch;
});

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  }) as Response;
}

function mockFallback(body: unknown) {
  fetchMock.mockImplementation(async () => jsonResponse(body));
}

describe("FallbackInsightsCard", () => {
  it("shows InsufficientDataCard when total attempts < 5", async () => {
    mockFallback({
      generated_at: new Date().toISOString(),
      history: { earliest_event_at: "", history_days: 0, has_history: false, min_sample_meaningful: 5 },
      event_count: 2,
      runner_attempts: 1,
      runner_exhausted: 0,
      runner_by_reason: {},
      runner_by_pair: [],
      runner_chain_depth: {},
      model_attempts: 1,
      model_exhausted: 0,
      model_by_reason: {},
      model_by_pair: [],
      model_chain_depth: {},
      model_by_preset: {},
    });

    renderWithProviders(createElement(FallbackInsightsCard));

    await waitFor(() => {
      expect(screen.getByTestId("fallback-insufficient")).toBeTruthy();
    });
    expect(screen.queryByTestId("fallback-insights")).toBeNull();
  });

  it("renders chain depth, top pairs, and top reasons when sample size meets threshold", async () => {
    mockFallback({
      generated_at: new Date().toISOString(),
      history: { earliest_event_at: new Date().toISOString(), history_days: 7, has_history: true, min_sample_meaningful: 5 },
      event_count: 12,
      runner_attempts: 4,
      runner_exhausted: 1,
      runner_by_reason: { binary_missing: 3, network_transient: 1 },
      runner_by_pair: [
        { from: "codex", to: "claude-code", reason: "binary_missing", count: 3 },
      ],
      runner_chain_depth: { "1": 3, "2": 1 },
      model_attempts: 6,
      model_exhausted: 2,
      model_by_reason: { rate_limit: 4, quota_exhausted: 2 },
      model_by_pair: [
        { from: "gpt-5.2-codex", to: "claude-opus-4-7", reason: "rate_limit", count: 4 },
      ],
      model_chain_depth: { "1": 4, "2": 2 },
      model_by_preset: { CHEAP: 3 },
    });

    renderWithProviders(createElement(FallbackInsightsCard));

    await waitFor(() => {
      expect(screen.getByTestId("fallback-insights")).toBeTruthy();
    });
    expect(screen.getByTestId("runner-pair-0")).toHaveTextContent("codex → claude-code");
    expect(screen.getByTestId("model-reason-0")).toHaveTextContent("rate_limit");
    expect(screen.getByTestId("fallback-summary-footer")).toHaveTextContent("12 fallback events");
  });
});
