import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useInvestigationSettings, useRunReport, useWorkflowExecutions } from "./useApi.js";
import { rpcBody, rpcCalls, rpcMethod } from "../test-utils/rpc.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

test("investigation settings load, update, reset, and surface write errors", async () => {
  const settings = { promptTemplate: "initial", defaultDepth: "standard" };
  const fetch = vi.fn()
    .mockResolvedValueOnce(json(settings))
    .mockResolvedValueOnce(json({ ...settings, promptTemplate: "updated" }))
    .mockResolvedValueOnce(json({ ...settings, promptTemplate: "default" }))
    .mockResolvedValueOnce(json({ error: "write denied" }, 403));
  vi.stubGlobal("fetch", fetch);
  const hook = renderHook(() => useInvestigationSettings());
  await waitFor(() => assert.equal(hook.result.current.data?.promptTemplate, "initial"));
  await act(async () => {
    const updated = await hook.result.current.updateSettings({ promptTemplate: "updated" });
    assert.equal(updated.promptTemplate, "updated");
    const reset = await hook.result.current.resetSettings();
    assert.equal(reset.promptTemplate, "default");
    await assert.rejects(hook.result.current.updateSettings({ promptTemplate: "blocked" }), /Request failed: 403/);
  });
  await waitFor(() => assert.equal(hook.result.current.error, "Request failed: 403"));
  assert.equal(fetch.mock.calls[1]?.[0], "http://localhost:3000/api/v1/investigation-settings");
  assert.equal(fetch.mock.calls[1]?.[1]?.method, "PUT");
  assert.equal(fetch.mock.calls[2]?.[0], "http://localhost:3000/api/v1/investigation-settings/reset");
  assert.equal(fetch.mock.calls[2]?.[1]?.method, "POST");
});

test("workflow execution list uses the shared versioned API base exactly once", async () => {
  const fetch = vi.fn(async () => json({ executions: [] }));
  vi.stubGlobal("fetch", fetch);
  const workflows = renderHook(() => useWorkflowExecutions());
  await waitFor(() => assert.equal(fetch.mock.calls.length, 1));
  assert.deepEqual(workflows.result.current.data, []);
  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/agent_manager.v1.AgentManagerService/ListWorkflowExecutions");
  assert.equal(rpcBody(fetch.mock.calls[0] as unknown as unknown[]).limit, 100);
});

test("workflow lifecycle events recover a failed projection and stop refreshing after unmount", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({ code: "unavailable", message: "workflow projection unavailable" }, 503))
    .mockResolvedValueOnce(json({ executions: [] }));
  vi.stubGlobal("fetch", fetch);
  const workflows = renderHook(() => useWorkflowExecutions());
  await waitFor(() => assert.equal(workflows.result.current.error, "[unavailable] workflow projection unavailable"));

  await act(async () => {
    window.dispatchEvent(new Event("agent-manager:workflow-lifecycle"));
  });
  await waitFor(() => {
    assert.equal(fetch.mock.calls.length, 2);
    assert.deepEqual(workflows.result.current.data, []);
    assert.equal(workflows.result.current.error, null);
  });

  workflows.unmount();
  window.dispatchEvent(new Event("agent-manager:workflow-lifecycle"));
  await act(async () => {});
  assert.equal(fetch.mock.calls.length, 2);
});

test("workflow execution inspection, control, and signal paths encode ids and refresh durable state", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({ executions: [] }))
    .mockResolvedValueOnce(json({ attempts: [], journal: [] }))
    .mockResolvedValueOnce(json({}))
    .mockResolvedValueOnce(json({ executions: [] }))
    .mockResolvedValueOnce(json({}))
    .mockResolvedValueOnce(json({ executions: [] }));
  vi.stubGlobal("fetch", fetch);
  const workflows = renderHook(() => useWorkflowExecutions());
  await waitFor(() => assert.equal(fetch.mock.calls.length, 1));
  const execution = { id: "execution/a", version: 7n } as never;
  await act(async () => {
    const trace = await workflows.result.current.getTrace("execution/a");
    assert.deepEqual(trace.attempts, []);
    await workflows.result.current.control(execution, "retry");
    await workflows.result.current.signal(execution, "approve", { approved: true });
  });
  const calls = fetch.mock.calls as unknown as unknown[][];
  assert.deepEqual(calls.map(rpcMethod), ["ListWorkflowExecutions", "GetWorkflowExecutionTrace", "RetryWorkflowExecution", "ListWorkflowExecutions", "SignalWorkflowExecution", "ListWorkflowExecutions"]);
  assert.deepEqual(rpcBody(calls[1]!), { executionId: "execution/a", limit: 500 });
  assert.equal(rpcBody(calls[2]!).idempotencyKey, "ui-retry-execution/a-7");
  const signal = JSON.stringify(rpcBody(calls[4]!));
  assert.match(signal, /"signal":"approve"/);
  assert.match(signal, /"bool_value":true/);
});

test("workflow signals preserve nested JSON values and canonicalize pre-encoded protobuf values", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.endsWith("/ListWorkflowExecutions")) return json({ executions: [] });
    return json({});
  });
  vi.stubGlobal("fetch", fetch);
  const workflows = renderHook(() => useWorkflowExecutions());
  await waitFor(() => assert.equal(fetch.mock.calls.length, 1));

  await act(async () => {
    await workflows.result.current.signal(
      { id: "execution-1", version: 2n } as never,
      "evidence",
      {
        accepted: true,
        count: 4,
        ratio: 1.5,
        note: "ready",
        none: null,
        paths: ["api", false],
        nested: { fields: { source: "receipt" } },
        preEncoded: { objectValue: { fields: { answer: { intValue: 42 } } } },
        preEncodedList: { listValue: { values: [{ stringValue: "already-encoded" }] } },
        preEncodedNull: { nullValue: "NULL_VALUE" },
      },
    );
  });

  const body = JSON.stringify(rpcBody(rpcCalls(fetch.mock.calls as unknown as unknown[][], "SignalWorkflowExecution")[0]!));
  for (const expected of [
    '"bool_value":true', '"int_value":4', '"double_value":1.5', '"string_value":"ready"',
    '"null_value":"NULL_VALUE"', '"list_value"', '"object_value"', '"source":{"string_value":"receipt"}',
    '"answer":{"int_value":42}', '"expectedVersion":"2"',
    '"preEncodedList":{"list_value":{"values":[{"string_value":"already-encoded"}]}}',
    '"preEncodedNull":{"null_value":"NULL_VALUE"}',
  ]) {
    assert.ok(body.includes(expected), `expected serialized signal to contain ${expected}`);
  }
});

test("run report hook fetches the unified report by encoded run id and preserves a useful error", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({ run_id: "run/a", status: "failed", turns: 2, tokens: 10, cost_usd: 0, result: {}, event_counts: {}, tools: [] }))
    .mockResolvedValueOnce(json({ error: "report unavailable" }, 503));
  vi.stubGlobal("fetch", fetch);
  const report = renderHook(() => useRunReport("run/a"));
  await waitFor(() => assert.equal(report.result.current.data?.run_id, "run/a"));
  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/runs/run%2Fa/report");
  await act(async () => { await report.result.current.refetch(); });
  await waitFor(() => assert.equal(report.result.current.error, "Request failed: 503"));
  const empty = renderHook(() => useRunReport(""));
  await act(async () => {});
  assert.equal(empty.result.current.loading, false);
});
