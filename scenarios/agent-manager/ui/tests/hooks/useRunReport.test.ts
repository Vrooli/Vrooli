import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useRunReport } from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

test("useRunReport fetches the bounded report and can refresh it", async () => {
  const fetch = vi.fn(async () => new Response(JSON.stringify({ run_id: "run/a", status: "failed", tools: [] }), { status: 200, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetch);
  const { result } = renderHook(() => useRunReport("run/a"));
  await waitFor(() => assert.equal(result.current.data?.run_id, "run/a"));
  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/runs/run%2Fa/report");
  await act(async () => { await result.current.refetch(); });
  assert.equal(fetch.mock.calls.length, 2);
  assert.equal(result.current.loading, false);
  assert.equal(result.current.error, null);
});

test("useRunReport preserves an API error and does not fetch an empty id", async () => {
  const fetch = vi.fn(async () => new Response(JSON.stringify({ error: "report unavailable" }), { status: 503, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetch);
  const failing = renderHook(() => useRunReport("run-2"));
  await waitFor(() => assert.equal(failing.result.current.error, "Request failed: 503"));
  assert.equal(failing.result.current.loading, false);

  const empty = renderHook(() => useRunReport(""));
  await act(async () => { await empty.result.current.refetch(); });
  assert.equal(fetch.mock.calls.length, 1);
});
