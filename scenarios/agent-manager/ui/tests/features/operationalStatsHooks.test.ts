import assert from "node:assert/strict";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { afterEach, test, vi } from "vitest";
import { createTestQueryClient } from "../../src/test-utils/renderWithProviders.js";
import { useFallbackInsights, useHealthSummary } from "../../src/features/stats/hooks/useOperationalStats.js";

const client = vi.hoisted(() => ({
  fallback: vi.fn(async () => ({ event_count: 7 })),
  health: vi.fn(async () => ({ failing_last_hour: [] })),
}));

vi.mock("../../src/features/stats/api/operationalClient.js", () => ({
  fetchFallbackInsights: client.fallback,
  fetchHealthSummary: client.health,
  operationalQueryKeys: {
    fallback: () => ["operational-stats", "fallback"],
    health: () => ["operational-stats", "health"],
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  return createElement(QueryClientProvider, { client: createTestQueryClient() }, children);
}

afterEach(() => vi.clearAllMocks());

test("operational stats hooks fetch their typed resources by default", async () => {
  const fallback = renderHook(() => useFallbackInsights(), { wrapper });
  const health = renderHook(() => useHealthSummary(), { wrapper });

  await waitFor(() => {
    assert.equal(fallback.result.current.isSuccess, true);
    assert.equal(health.result.current.isSuccess, true);
  });

  assert.equal(client.fallback.mock.calls.length, 1);
  assert.equal(client.health.mock.calls.length, 1);
  assert.deepEqual(fallback.result.current.data, { event_count: 7 });
  assert.deepEqual(health.result.current.data, { failing_last_hour: [] });
});

test("operational stats hooks stay idle when an enclosing view disables them", () => {
  const fallback = renderHook(() => useFallbackInsights({ enabled: false }), { wrapper });
  const health = renderHook(() => useHealthSummary({ enabled: false }), { wrapper });

  assert.equal(fallback.result.current.fetchStatus, "idle");
  assert.equal(health.result.current.fetchStatus, "idle");
  assert.equal(client.fallback.mock.calls.length, 0);
  assert.equal(client.health.mock.calls.length, 0);
});
