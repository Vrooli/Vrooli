import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useInvestigationSettings, useRecurringFindings, useWorkflowExecutions } from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

test("recurring findings load in priority order and retain an API failure", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({ findings: [
      { id: "older", occurrences: 2, createdAt: "2026-07-01T00:00:00Z" },
      { id: "frequent", occurrences: 4, createdAt: "2026-06-01T00:00:00Z" },
      { id: "newer", occurrences: 2, createdAt: "2026-07-02T00:00:00Z" },
    ] }))
    .mockResolvedValueOnce(json({ error: "findings down" }, 503));
  vi.stubGlobal("fetch", fetch);
  const findings = renderHook(() => useRecurringFindings());
  await waitFor(() => assert.deepEqual(findings.result.current.data?.map((finding) => finding.id), ["frequent", "newer", "older"]));
  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/findings");
  await act(async () => { await findings.result.current.refetch(); });
  assert.equal(findings.result.current.error, "Request failed: 503");
});

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
  assert.equal(hook.result.current.error, "Request failed: 403");
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
  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/workflow-executions?limit=100");
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
  assert.equal(fetch.mock.calls[1]?.[0], "http://localhost:3000/api/v1/workflow-executions/execution%2Fa/trace?limit=500");
  assert.equal(fetch.mock.calls[2]?.[0], "http://localhost:3000/api/v1/workflow-executions/execution%2Fa/retry");
  assert.equal(fetch.mock.calls[2]?.[1]?.method, "POST");
  assert.match(String(fetch.mock.calls[2]?.[1]?.body), /ui-retry-execution\/a-7/);
  assert.equal(fetch.mock.calls[4]?.[0], "http://localhost:3000/api/v1/workflow-executions/execution%2Fa/signals");
  assert.match(String(fetch.mock.calls[4]?.[1]?.body), /"signal":"approve"/);
  assert.match(String(fetch.mock.calls[4]?.[1]?.body), /"bool_value":true/);
});
