import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useProfiles, useRuns, useTasks } from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

test("profile, task, and run lists honor disabled mode and use canonical bounded endpoints", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("/profiles")) return json({ profiles: [] });
    if (url.includes("/tasks")) return json({ tasks: [] });
    return json({ runs: [] });
  });
  vi.stubGlobal("fetch", fetch);

  renderHook(() => useProfiles({ enabled: false }));
  renderHook(() => useTasks({ enabled: false }));
  renderHook(() => useRuns({ enabled: false }));
  await act(async () => {});
  assert.equal(fetch.mock.calls.length, 0);

  const profiles = renderHook(() => useProfiles());
  const tasks = renderHook(() => useTasks());
  const runs = renderHook(() => useRuns({ limit: 25 }));
  await waitFor(() => assert.equal(fetch.mock.calls.length, 3));
  assert.deepEqual(profiles.result.current.data, []);
  assert.deepEqual(tasks.result.current.data, []);
  assert.deepEqual(runs.result.current.data, []);
  assert.ok(fetch.mock.calls.some(([url]) => url === "http://localhost:3000/api/v1/profiles"));
  assert.ok(fetch.mock.calls.some(([url]) => url === "http://localhost:3000/api/v1/tasks"));
  assert.ok(fetch.mock.calls.some(([url]) => url === "http://localhost:3000/api/v1/runs?limit=25"));
});

test("core list hooks retain typed API failures instead of replacing visible data", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({ error: "profiles unavailable" }, 503))
    .mockResolvedValueOnce(json({ error: "tasks unavailable" }, 502))
    .mockResolvedValueOnce(json({ error: "runs unavailable" }, 500));
  vi.stubGlobal("fetch", fetch);
  const profiles = renderHook(() => useProfiles());
  const tasks = renderHook(() => useTasks());
  const runs = renderHook(() => useRuns());
  await waitFor(() => {
    assert.equal(profiles.result.current.error, "Request failed: 503");
    assert.equal(tasks.result.current.error, "Request failed: 502");
    assert.equal(runs.result.current.error, "Request failed: 500");
  });
  assert.deepEqual(profiles.result.current.data, []);
  assert.deepEqual(tasks.result.current.data, []);
  assert.deepEqual(runs.result.current.data, []);
});
