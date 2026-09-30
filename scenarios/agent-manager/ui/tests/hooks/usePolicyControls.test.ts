import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { probeRunner, usePermissionPolicy, useRolePolicyCatalog, useRunStatusCounts, useRunners } from "../../src/hooks/useApi.js";
import { RunnerType } from "../../src/types.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
}

test("policy and runner controls use canonical read and whole-document action routes", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("status-distribution")) return json({ statusCounts: { complete: 2, failed: 1, total: 3 } });
    if (url.endsWith("/runners")) return json({ runners: [] });
    return json({});
  });
  vi.stubGlobal("fetch", fetch);
  const counts = renderHook(() => useRunStatusCounts());
  const runners = renderHook(() => useRunners());
  const roleCatalog = renderHook(() => useRolePolicyCatalog());
  const policy = renderHook(() => usePermissionPolicy());
  await waitFor(() => assert.ok(fetch.mock.calls.length >= 5));
  await waitFor(() => assert.ok(roleCatalog.result.current.data));
  assert.equal(counts.result.current.data?.total, 3);
  assert.deepEqual(runners.result.current.data, {});
  assert.ok(roleCatalog.result.current.data);
  await act(async () => {
    await policy.result.current.validate();
    await policy.result.current.reload();
    await policy.result.current.plan();
    await policy.result.current.doctor();
    await policy.result.current.reconcile();
    await probeRunner(RunnerType.CODEX);
  });
  const calls = fetch.mock.calls.map(([url, options]) => [String(url), options] as const);
  for (const path of ["/permission-policy/validate", "/permission-policy/reload", "/permission-policy/plan", "/permission-policy/doctor", "/permission-policy/reconcile"]) {
    assert.equal(calls.find(([url]) => url.includes(path))?.[1]?.method, "POST");
  }
  assert.match(String(calls.find(([url]) => url.includes("/permission-policy/reconcile"))?.[1]?.body), /explicitly_authorized/);
  assert.ok(calls.some(([url, options]) => url.endsWith("/runners/codex/probe") && options?.method === "POST"));
});
