import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useRuns } from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
}

test("run controls encode inspection, investigation, review, continuation, and lifecycle requests", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("/runs?") || url.endsWith("/runs")) return json({ runs: [] });
    if (url.includes("/continue")) return json({ success: true, run: {} });
    if (url.endsWith("/events?after_sequence=3")) return json({ events: [] });
    if (url.endsWith("/events")) return json({ events: [] });
    if (url.endsWith("/diff")) return json({ diff: {} });
    return json({});
  });
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns({ limit: 10 }));
  await waitFor(() => assert.equal(fetch.mock.calls.length, 1));
  const run = { id: "run-a", taskId: "task-a", agentProfileId: "profile-a" } as never;

  await act(async () => {
    await runs.result.current.createRun({ taskId: "task-a", agentProfileId: "profile-a", prompt: "inspect", conversationId: "conversation-a" });
    await runs.result.current.retryRun(run);
    await runs.result.current.investigateRuns(["run-a"], "focus failures", "deep", "/workspace", ["src"], ["attachment-a"], { roleRef: "agent-manager/investigate" });
    await runs.result.current.applyInvestigation("investigation-a", ["finding-a"], "apply safely", ["attachment-a"], { roleRef: "agent-manager/investigate" });
    await runs.result.current.resumeFromFailedRun("run-a", "continue", ["attachment-a"]);
    await runs.result.current.getRun("run-a");
    await runs.result.current.stopRun("run-a");
    await runs.result.current.deleteRun("run-a");
    await runs.result.current.getRunEvents("run-a", { afterSequence: 3n });
    await runs.result.current.getRunDiff("run-a");
    await runs.result.current.approveRun("run-a", { actor: " operator ", commitMsg: "approve", force: true });
    await runs.result.current.rejectRun("run-a", { actor: "operator", reason: "needs work" });
    await runs.result.current.partialApproveRun("run-a", ["file-a"], "operator", "partial");
    await runs.result.current.continueRun("run-a", "follow up", ["attachment-a"]);
    await runs.result.current.deleteRunMessage("run-a", "event-a");
  });

  const calls = fetch.mock.calls.map(([url, options]) => [String(url), options] as const);
  const request = (path: string) => calls.find(([url]) => url.includes(path));
  assert.equal(request("/runs/investigate")?.[1]?.method, "POST");
  assert.match(String(request("/runs/investigate")?.[1]?.body), /"roleRef":"agent-manager\/investigate"/);
  assert.equal(request("/runs/investigation-apply")?.[1]?.method, "POST");
  assert.match(String(request("/runs/resume-from-failed")?.[1]?.body), /"attachment-a"/);
  assert.equal(request("/runs/run-a/stop")?.[1]?.method, "POST");
  assert.equal(request("/runs/run-a/messages/event-a/delete")?.[1]?.method, "POST");
  assert.ok(calls.some(([url]) => url.endsWith("/runs/run-a/events?after_sequence=3")));
  assert.ok(calls.some(([url]) => url.endsWith("/runs/run-a/diff")));
  assert.match(String(request("/runs/run-a/continue")?.[1]?.body), /"attachment_ids"/);
});

test("run continuation preserves a server rejection instead of treating it as success", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({ runs: [] }))
    .mockResolvedValueOnce(json({ success: false, error: "continuation rejected" }));
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns());
  await waitFor(() => assert.equal(fetch.mock.calls.length, 1));
  await assert.rejects(runs.result.current.continueRun("run-a", "follow up"), /continuation rejected/);
});
