import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import {
  useInvestigationSettings,
  usePermissionPolicy,
  useHealth,
  useRunners,
  useRuns,
  useWorkflowExecutions,
} from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

test("workflow controls encode versioned operations, refresh projections, and react to lifecycle events", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.includes("/trace")) return json({ execution: { id: "workflow-1", version: "7" }, attempts: [], journal: [] });
    if (url.includes("/signals")) return json({ execution: { id: "workflow-1", version: "8" } });
    if (url.includes("/cancel")) return json({ execution: { id: "workflow-1", version: "8" } });
    if (init?.method === "POST") return json({});
    return json({ executions: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const workflows = renderHook(() => useWorkflowExecutions());
  await waitFor(() => assert.equal(workflows.result.current.loading, false));

  await act(async () => {
    const trace = await workflows.result.current.getTrace("workflow / 1");
    assert.equal(trace.execution?.id, "workflow-1");
    await workflows.result.current.control({ id: "workflow-1", version: 7n } as never, "cancel");
    await workflows.result.current.signal({ id: "workflow-1", version: 7n } as never, "approval", { approved: true, paths: ["ui"] });
  });
  await act(async () => {
    window.dispatchEvent(new Event("agent-manager:workflow-lifecycle"));
  });
  await waitFor(() => assert.ok(fetch.mock.calls.filter(([url]) => String(url).endsWith("/workflow-executions?limit=100")).length >= 4));

  const traceCall = fetch.mock.calls.find(([url]) => String(url).includes("/trace"));
  const cancelCall = fetch.mock.calls.find(([url]) => String(url).includes("/cancel"));
  const signalCall = fetch.mock.calls.find(([url]) => String(url).includes("/signals"));
  assert.match(String(traceCall?.[0]), /workflow%20%2F%201/);
  assert.match(String(cancelCall?.[1]?.body), /"expected_version":"7"/);
  assert.match(String(signalCall?.[1]?.body), /"approved"/);
  assert.match(String(signalCall?.[1]?.body), /"paths"/);
  workflows.unmount();
});

test("workflow signals preserve every supported JSON payload shape for agent handoffs", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("/signals")) return json({ execution: { id: "workflow-1", version: "8" } });
    return json({ executions: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const workflows = renderHook(() => useWorkflowExecutions());
  await waitFor(() => assert.equal(workflows.result.current.loading, false));
  const execution = { id: "workflow-1", version: 7n } as never;

  await act(async () => {
    await workflows.result.current.signal(execution, "null", null);
    await workflows.result.current.signal(execution, "number", 1.25);
    await workflows.result.current.signal(execution, "integer", 2);
    await workflows.result.current.signal(execution, "string", "ready");
    await workflows.result.current.signal(execution, "wrapped", {
      objectValue: { fields: { nested: { list_value: { values: [true, null] } } } },
    });
  });
  const bodies = fetch.mock.calls
    .filter(([url]) => String(url).includes("/signals"))
    .map(([, init]) => String((init as RequestInit).body));
  assert.ok(bodies.some((body) => body.includes("NULL_VALUE")));
  assert.ok(bodies.some((body) => body.includes("1.25")));
  assert.ok(bodies.some((body) => body.includes('"signal":"integer"')));
  assert.ok(bodies.some((body) => body.includes("ready")));
  assert.ok(bodies.some((body) => body.includes("nested")));
  workflows.unmount();
});

test("workflow signals normalize sparse wrapped values without leaking invalid JSON-value shapes", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("/signals")) return json({ execution: { id: "workflow-1", version: "8" } });
    return json({ executions: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const workflows = renderHook(() => useWorkflowExecutions());
  await waitFor(() => assert.equal(workflows.result.current.loading, false));
  const execution = { id: "workflow-1", version: 7n } as never;

  await act(async () => {
    // Agents can hand off partially wrapped values from older runners.  The
    // UI must turn those into legal protobuf values instead of passing the
    // malformed wrapper through to the service.
    await workflows.result.current.signal(execution, "plain-object", { objectValue: { phase: "review" } });
    await workflows.result.current.signal(execution, "empty-object", { object_value: null } as never);
    await workflows.result.current.signal(execution, "empty-list", { listValue: {} } as never);
    await workflows.result.current.signal(execution, "array-list", { list_value: ["one", false] } as never);
  });

  const bodies = fetch.mock.calls
    .filter(([url]) => String(url).includes("/signals"))
    .map(([, init]) => String((init as RequestInit).body));
  assert.ok(bodies.some((body) => body.includes('"signal":"plain-object"') && body.includes("phase")));
  assert.ok(bodies.some((body) => body.includes('"signal":"empty-object"') && body.includes("object_value")));
  assert.ok(bodies.some((body) => body.includes('"signal":"empty-list"') && body.includes("list_value")));
  assert.ok(bodies.some((body) => body.includes('"signal":"array-list"') && body.includes("one")));
  workflows.unmount();
});

test("permission-policy controls keep reads opt-in, refresh only state-changing commands, and surface read failures", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("/permission-policy/status")) return json({});
    if (url.includes("/permission-policy/catalog")) return json({});
    return json({});
  });
  vi.stubGlobal("fetch", fetch);
  const policy = renderHook(() => usePermissionPolicy({ enabled: false }));
  await act(async () => {});
  assert.equal(fetch.mock.calls.length, 0);

  await act(async () => {
    await policy.result.current.validate();
    await policy.result.current.reload();
    await policy.result.current.plan();
    await policy.result.current.doctor();
    await policy.result.current.reconcile();
  });
  const calls = fetch.mock.calls.map(([url, init]) => [String(url), init] as const);
  const posts = calls.filter(([, init]) => init?.method === "POST");
  assert.deepEqual(posts.map(([url]) => url.replace("http://localhost:3000/api/v1", "")), [
    "/permission-policy/validate", "/permission-policy/reload", "/permission-policy/plan",
    "/permission-policy/doctor", "/permission-policy/reconcile",
  ]);
  assert.match(String(posts.at(-1)?.[1]?.body), /"explicitly_authorized":true/);
  // Validate, reload, and reconcile refresh the two read projections; plan and doctor are read-only diagnostics.
  assert.equal(calls.filter(([url]) => /permission-policy\/(status|catalog)$/.test(url)).length, 6);

  fetch.mockImplementation(async () => json({ message: "policy file is unavailable" }, 503));
  await act(async () => { await policy.result.current.refetch(); });
  assert.equal(policy.result.current.error, "policy file is unavailable");
});

test("investigation settings exposes update and reset failures without discarding the last successful settings", async () => {
  const settings = { promptTemplate: "baseline", applyPromptTemplate: "apply", defaultDepth: "standard" };
  const fetch = vi.fn()
    .mockResolvedValueOnce(json(settings))
    .mockResolvedValueOnce(json({ message: "settings are locked" }, 409))
    .mockResolvedValueOnce(json({ message: "reset is unavailable" }, 503));
  vi.stubGlobal("fetch", fetch);
  const investigation = renderHook(() => useInvestigationSettings());
  await waitFor(() => assert.equal(investigation.result.current.data?.promptTemplate, "baseline"));

  await act(async () => {
    await assert.rejects(investigation.result.current.updateSettings({ promptTemplate: "changed" }), /settings are locked/);
  });
  assert.equal(investigation.result.current.error, "settings are locked");
  assert.equal(investigation.result.current.data?.promptTemplate, "baseline");

  await act(async () => {
    await assert.rejects(investigation.result.current.resetSettings(), /reset is unavailable/);
  });
  assert.equal(investigation.result.current.error, "reset is unavailable");
  assert.equal(investigation.result.current.data?.promptTemplate, "baseline");
  assert.equal((fetch.mock.calls[1]?.[1] as RequestInit).method, "PUT");
  assert.equal((fetch.mock.calls[2]?.[1] as RequestInit).method, "POST");
});

test("core runtime projections tolerate sparse payloads while preserving bounded event cursors", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.endsWith("/health")) return json("not-an-object");
    if (url.endsWith("/runners")) return json({ runners: [{ runnerType: "RUNNER_TYPE_CODEX", available: true }] });
    if (url.includes("/events")) return json({});
    return json({});
  });
  vi.stubGlobal("fetch", fetch);

  const health = renderHook(() => useHealth());
  const runners = renderHook(() => useRunners());
  const runs = renderHook(() => useRuns({ enabled: false, limit: 0 }));

  await waitFor(() => {
    const available = Object.values(runners.result.current.data ?? {}).some((runner) => runner.available);
    assert.equal(available, true);
  });
  await waitFor(() => assert.ok(health.result.current.error));
  await act(async () => {
    await runs.result.current.refetch();
    // A background refresh must not reintroduce the initial-load spinner.
    await runs.result.current.refetch();
    assert.deepEqual(await runs.result.current.getRunEvents("run / one"), []);
    assert.deepEqual(await runs.result.current.getRunEvents("run / one", { afterSequence: 9n }), []);
  });

  assert.equal(fetch.mock.calls.filter(([url]) => url === "http://localhost:3000/api/v1/runs?limit=0").length, 2);
  assert.ok(fetch.mock.calls.some(([url]) => url === "http://localhost:3000/api/v1/runs/run / one/events"));
  assert.ok(fetch.mock.calls.some(([url]) => url === "http://localhost:3000/api/v1/runs/run / one/events?after_sequence=9"));
  assert.ok(fetch.mock.calls.some(([url]) => url === "http://localhost:3000/api/v1/runners"));
  health.unmount();
  runners.unmount();
  runs.unmount();
});
