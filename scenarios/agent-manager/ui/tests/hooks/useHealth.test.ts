import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useHealth } from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

test("useHealth normalizes ordinary JSON dependency and metric values into the common proto contract", async () => {
  const fetch = vi.fn(async () => new Response(JSON.stringify({
    status: "HEALTH_STATUS_HEALTHY", service: "agent-manager", timestamp: "2026-07-29T00:00:00Z", readiness: true,
    dependencies: { postgres: { status: "healthy", latency_ms: 5 } }, metrics: { goroutines: 42, heap_mb: 1.5 },
  }), { status: 200, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetch);
  const health = renderHook(() => useHealth());
  await waitFor(() => assert.equal(health.result.current.data?.service, "agent-manager"));
  assert.equal(health.result.current.data?.readiness, true);
  assert.ok(health.result.current.data?.dependencies.postgres);
  assert.ok(health.result.current.data?.metrics.goroutines);
  await act(async () => { await health.result.current.refetch(); });
  assert.equal(fetch.mock.calls.length, 2);
  const firstSignal = fetch.mock.calls[0]?.[1]?.signal as AbortSignal;
  assert.equal(firstSignal.aborted, true);
});

test("useHealth ignores an aborted previous request but exposes a real health error", async () => {
  const abortError = new DOMException("aborted", "AbortError");
  const fetch = vi.fn()
    .mockRejectedValueOnce(abortError)
    .mockResolvedValueOnce(new Response(JSON.stringify({ error: "unavailable" }), { status: 503, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetch);
  const health = renderHook(() => useHealth());
  await waitFor(() => assert.equal(health.result.current.error, null));
  await act(async () => { await health.result.current.refetch(); });
  await waitFor(() => assert.equal(health.result.current.error, "Request failed: 503"));
});
