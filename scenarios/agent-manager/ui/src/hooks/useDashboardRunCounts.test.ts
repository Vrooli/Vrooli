import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useDashboardRunCounts } from "./useApi.js";
import { RunScope } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";

const measures = vi.hoisted(() => ({ runStatusDistribution: vi.fn() }));
vi.mock("../features/usage/measuresClient.js", () => ({ measuresClient: measures }));

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
  measures.runStatusDistribution.mockReset();
});

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function statusRows(counts: Record<string, number>) {
  return { rows: Object.entries(counts).map(([status, count]) => ({ status, count: BigInt(count) })), executedQuery: "", definitionId: "throughput.run_status_distribution" };
}

/** ListRuns with a status filter: `total` is the full matching count, the page holds at most `limit` runs. */
function listRunsFetch(totals: Record<string, number>) {
  return vi.fn(async (url: string) => {
    const query = new URL(url).searchParams;
    const status = query.get("status") ?? "";
    const runs = (totals[status] ?? 0) > 0 ? [{ id: `${status}-1` }] : [];
    return json({ runs, total: totals[status] ?? 0, hasMore: (totals[status] ?? 0) > runs.length });
  });
}

test("dashboard counters use the API's matching totals, not the page length, and Measures for the 7d window", async () => {
  measures.runStatusDistribution.mockImplementation(async (request: { filter: { scope: RunScope } }) =>
    request.filter.scope === RunScope.OPERATOR_SESSIONS ? statusRows({ complete: 400, failed: 12 }) : statusRows({ complete: 600, failed: 40, cancelled: 2 }));
  const fetch = listRunsFetch({ running: 19, starting: 1, needs_review: 83 });
  vi.stubGlobal("fetch", fetch);

  const counts = renderHook(() => useDashboardRunCounts());
  await waitFor(() => assert.ok(counts.result.current.data));

  assert.deepEqual(counts.result.current.data, { active: 20, review: 83, windowTotal: 642, operatorSessions: 412 });
  const urls = fetch.mock.calls.map(([url]) => String(url));
  for (const status of ["running", "starting", "needs_review"]) {
    assert.ok(urls.includes(`http://localhost:3000/api/v1/runs?status=${status}&limit=1`), `missing count read for ${status}`);
  }
  assert.equal(urls.some((url) => url.includes("/stats/")), false, "dashboard counters must not read the REST stats layer");
  const custom = measures.runStatusDistribution.mock.calls[0][0].window.window.value;
  assert.deepEqual(measures.runStatusDistribution.mock.calls.map(([request]) => request.filter.scope).sort(), [RunScope.AGENT_MANAGER, RunScope.OPERATOR_SESSIONS]);
  assert.equal(Number(custom.to.seconds - custom.from.seconds), 7 * 24 * 3600);
  counts.unmount();
});

test("dashboard counters stay idle when disabled and surface a failed read", async () => {
  measures.runStatusDistribution.mockRejectedValue(new Error("measures unavailable"));
  const fetch = listRunsFetch({});
  vi.stubGlobal("fetch", fetch);

  renderHook(() => useDashboardRunCounts({ enabled: false }));
  await act(async () => {});
  assert.equal(fetch.mock.calls.length, 0);
  assert.equal(measures.runStatusDistribution.mock.calls.length, 0);

  const counts = renderHook(() => useDashboardRunCounts());
  await waitFor(() => assert.equal(counts.result.current.error, "measures unavailable"));
  assert.equal(counts.result.current.data, null);
  counts.unmount();
});

test("a burst of refetches after websocket status events causes one read", async () => {
  vi.useFakeTimers({ shouldAdvanceTime: true });
  measures.runStatusDistribution.mockResolvedValue(statusRows({}));
  const fetch = listRunsFetch({ running: 1 });
  vi.stubGlobal("fetch", fetch);

  const counts = renderHook(() => useDashboardRunCounts());
  await waitFor(() => assert.equal(measures.runStatusDistribution.mock.calls.length, 2));
  act(() => {
    for (let i = 0; i < 5; i += 1) counts.result.current.refetch();
  });
  await act(async () => { await vi.advanceTimersByTimeAsync(1_000); });
  assert.equal(measures.runStatusDistribution.mock.calls.length, 4);
  counts.unmount();
});
