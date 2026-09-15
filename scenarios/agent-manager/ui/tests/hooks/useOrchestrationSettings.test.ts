import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useOrchestrationSettings, type OrchestrationSettings } from "../../src/hooks/useOrchestrationSettings.js";

const settings: OrchestrationSettings = {
  runExecution: { runTimeoutMinutes: 20, maxConcurrentRuns: 2, maxTurns: 8 },
  safetyIsolation: { requireSandbox: true, requireApproval: true, networkAccess: "localhost" },
  healthDetection: { heartbeatIntervalSeconds: 10, staleThresholdSeconds: 60, maxRecoveryAgeSeconds: 120, reconcilerIntervalSeconds: 30 },
  processTermination: { gracePeriodSeconds: 5, killProcessGroup: true, killOrphans: true, orphanGracePeriodSeconds: 10, terminationMaxRetries: 2 },
};

function ok(payload: unknown) {
  return new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } });
}

afterEach(() => vi.unstubAllGlobals());

test("orchestration settings hook loads, updates, resets, and keeps the returned settings", async () => {
  const reset = { ...settings, runExecution: { ...settings.runExecution, maxTurns: 10 } };
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/reset")) return ok(reset);
    if (init?.method === "PUT") return ok(JSON.parse(String(init.body)));
    return ok(settings);
  });
  vi.stubGlobal("fetch", fetch);
  const hook = renderHook(() => useOrchestrationSettings());

  await waitFor(() => assert.equal(hook.result.current.data?.runExecution.maxTurns, 8));
  const changed = { ...settings, runExecution: { ...settings.runExecution, maxTurns: 12 } };
  await act(async () => { await hook.result.current.updateSettings(changed); });
  assert.equal(hook.result.current.data?.runExecution.maxTurns, 12);
  await act(async () => { await hook.result.current.resetSettings(); });
  assert.equal(hook.result.current.data?.runExecution.maxTurns, 10);

  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/orchestration-settings");
  assert.equal(fetch.mock.calls[1]?.[1]?.method, "PUT");
  assert.deepEqual(JSON.parse(String(fetch.mock.calls[1]?.[1]?.body)), changed);
  assert.equal(fetch.mock.calls[2]?.[0], "http://localhost:3000/api/v1/orchestration-settings/reset");
  assert.equal(fetch.mock.calls[2]?.[1]?.method, "POST");
});

test("orchestration settings hook exposes failures without replacing the last safe settings", async () => {
  let calls = 0;
  vi.stubGlobal("fetch", vi.fn(async () => {
    calls += 1;
    return calls === 1 ? ok(settings) : new Response(JSON.stringify({ message: "settings unavailable" }), { status: 503 });
  }));
  const hook = renderHook(() => useOrchestrationSettings());
  await waitFor(() => assert.equal(hook.result.current.data?.runExecution.maxTurns, 8));
  await act(async () => {
    await assert.rejects(hook.result.current.resetSettings(), /settings unavailable/);
  });
  await waitFor(() => assert.equal(hook.result.current.error, "settings unavailable"));
  assert.equal(hook.result.current.data?.runExecution.maxTurns, 8);
});
