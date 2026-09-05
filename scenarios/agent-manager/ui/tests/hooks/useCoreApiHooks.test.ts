import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useHealth, useMaintenance, useProfiles, useRunStatusCounts, useRunners, useRuns, useTasks } from "../../src/hooks/useApi.js";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

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

test("health, status, and runner projections normalize their independent endpoints", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.endsWith("/health")) return json({ status: "HEALTH_STATUS_READY", dependencies: { api: "ready" }, metrics: { queued: 2 } });
    if (url.endsWith("/stats/status-distribution")) return json({ statusCounts: { pending: 1, running: 2, complete: 3, failed: 4, cancelled: 5, needsReview: 6, total: 21 } });
    return json({ runners: [{ runnerType: "RUNNER_TYPE_CODEX", available: true }] });
  });
  vi.stubGlobal("fetch", fetch);
  const health = renderHook(() => useHealth());
  const counts = renderHook(() => useRunStatusCounts());
  const runners = renderHook(() => useRunners());
  await waitFor(() => assert.equal(counts.result.current.data?.total, 21));
  await waitFor(() => assert.equal(Object.keys(runners.result.current.data ?? {}).length, 1));
  await waitFor(() => assert.ok(health.result.current.data));
  assert.equal(fetch.mock.calls.some(([url]) => url === "http://localhost:3000/api/v1/health"), true);
  health.unmount(); counts.unmount(); runners.unmount();
});

test("status and runner projections retain errors, support disabled mode, and tolerate empty API payloads", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({}))
    .mockResolvedValueOnce(json({ runners: [] }))
    .mockResolvedValueOnce(json({ error: "stats unavailable" }, 503))
    .mockResolvedValueOnce(json({ error: "runners unavailable" }, 502));
  vi.stubGlobal("fetch", fetch);

  renderHook(() => useRunStatusCounts({ enabled: false }));
  renderHook(() => useRunners({ enabled: false }));
  await act(async () => {});
  assert.equal(fetch.mock.calls.length, 0);

  const counts = renderHook(() => useRunStatusCounts());
  const runners = renderHook(() => useRunners());
  await waitFor(() => {
    assert.equal(counts.result.current.data, null);
    assert.deepEqual(runners.result.current.data, {});
  });
  await act(async () => {
    await counts.result.current.refetch();
    await runners.result.current.refetch();
  });
  assert.equal(counts.result.current.error, "Request failed: 503");
  assert.equal(runners.result.current.error, "Request failed: 502");
});

test("list projections accept incomplete successful envelopes and an unbounded run refresh", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.endsWith("/profiles")) return json({});
    if (url.endsWith("/tasks")) return json({});
    if (url.endsWith("/runners")) return json({});
    return json({});
  });
  vi.stubGlobal("fetch", fetch);

  const profiles = renderHook(() => useProfiles());
  const tasks = renderHook(() => useTasks());
  const runners = renderHook(() => useRunners());
  const runs = renderHook(() => useRuns());
  await waitFor(() => {
    assert.deepEqual(profiles.result.current.data, []);
    assert.deepEqual(tasks.result.current.data, []);
    assert.deepEqual(runners.result.current.data, {});
    assert.deepEqual(runs.result.current.data, []);
  });

  await act(async () => { await runs.result.current.refetch(); });
  const runCalls = fetch.mock.calls.filter(([url]) => String(url).endsWith("/runs"));
  assert.equal(runCalls.length, 2);
  assert.equal(runs.result.current.loading, false);
  assert.equal(runs.result.current.error, null);
});

test("maintenance distinguishes dry-run preview from explicitly destructive purge", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({ matched: { profiles: 2, runs: 4 } }))
    .mockResolvedValueOnce(json({ deleted: { profiles: 2, runs: 4 } }));
  vi.stubGlobal("fetch", fetch);
  const maintenance = renderHook(() => useMaintenance());
  await act(async () => {
    const preview = await maintenance.result.current.previewPurge("^test", [PurgeTarget.PROFILES, PurgeTarget.RUNS]);
    const executed = await maintenance.result.current.executePurge("^test", [PurgeTarget.PROFILES, PurgeTarget.RUNS]);
    assert.equal(preview.profiles, 2); assert.equal(preview.runs, 4);
    assert.equal(executed.profiles, 2); assert.equal(executed.runs, 4);
  });
  assert.match(String(fetch.mock.calls[0]?.[1]?.body), /"dry_run":true/);
  // Proto JSON elides the false default; the server interprets its absence as execute mode.
  assert.doesNotMatch(String(fetch.mock.calls[1]?.[1]?.body), /"dry_run"/);
});

test("maintenance treats successful purge responses with omitted count maps as empty results", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(json({}))
    .mockResolvedValueOnce(json({}));
  vi.stubGlobal("fetch", fetch);
  const maintenance = renderHook(() => useMaintenance());

  await act(async () => {
    assert.deepEqual(await maintenance.result.current.previewPurge("^ephemeral", []), {});
    assert.deepEqual(await maintenance.result.current.executePurge("^ephemeral", []), {});
  });
  assert.equal(fetch.mock.calls.length, 2);
});

test("run controls preserve investigation context, review attribution, continuation attachments, and refresh state", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/runs") && (!init?.method || init.method === "GET")) return json({ runs: [] });
    if (url.includes("/continue")) return json({ success: true, run: { id: "continued" } });
    if (url.includes("/events")) return json({ events: [] });
    if (url.endsWith("/diff")) return json({ diff: {} });
    if (/\/runs\/run-1$/.test(url) && (!init?.method || init.method === "GET")) return json({ run: { id: "run-1" } });
    if (url.includes("/approve")) return json({ result: { remaining: 0 } });
    if (url.includes("/partial-approve")) return json({ result: { remaining: 1 } });
    if (url.includes("/investigate") || url.includes("investigation-apply") || url.includes("resume-from-failed") || (url.endsWith("/runs") && init?.method === "POST")) return json({ run: { id: "created" } });
    return json({});
  });
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns({ enabled: false }));
  await act(async () => {
    await runs.result.current.createRun({ taskId: "task-1", roleRef: "investigator", conversationId: "conversation-1", parentRunId: "parent-1", networkAccess: "full" });
    await runs.result.current.investigateRuns(["source-1"], "focus receipts", "deep", "/repo", ["api"], ["attachment-1"], { roleRef: "investigator" });
    await runs.result.current.applyInvestigation("investigation-1", ["recommendation-1"], "apply safely", ["attachment-2"], { roleRef: "implementer" });
    await runs.result.current.resumeFromFailedRun("failed-1", "finish safely", ["attachment-3"]);
    await runs.result.current.getRun("run-1"); await runs.result.current.getRunEvents("run-1", { afterSequence: 9n }); await runs.result.current.getRunDiff("run-1");
    await runs.result.current.approveRun("run-1", { actor: "  operator ", commitMsg: "reviewed", force: true });
    await runs.result.current.rejectRun("run-1", { actor: "  operator ", reason: "insufficient evidence" });
    await runs.result.current.partialApproveRun("run-1", ["file-1"], "  operator ", "partial");
    await runs.result.current.continueRun("run-1", "continue after report", ["attachment-4"]);
    await runs.result.current.stopRun("run-1"); await runs.result.current.deleteRunMessage("run-1", "event-1"); await runs.result.current.deleteRun("run-1");
  });
  const bodyFor = (needle: string) => String(fetch.mock.calls.find(([url]) => String(url).includes(needle))?.[1]?.body);
  assert.match(bodyFor("/investigate"), /"depth":"deep"/); assert.match(bodyFor("/investigate"), /"roleRef":"investigator"/);
  assert.match(bodyFor("investigation-apply"), /"roleRef":"implementer"/); assert.match(bodyFor("/continue"), /"attachment_ids"/);
  assert.match(bodyFor("/approve"), /"actor":"operator"/); assert.match(bodyFor("/reject"), /"reason":"insufficient evidence"/);
  assert.equal(fetch.mock.calls.some(([url]) => String(url).includes("after_sequence=9")), true);
});

test("run retry and continuation failure paths preserve the server error and omit empty attachments", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/runs") && init?.method === "POST") return json({ run: { id: "retried" } });
    if (url.includes("/continue")) return json({ success: false, error: "run is no longer resumable" });
    return json({ runs: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns({ enabled: false }));

  await act(async () => {
    await runs.result.current.retryRun({ taskId: "task-1", agentProfileId: "profile-1" } as never);
    await assert.rejects(
      runs.result.current.continueRun("run-1", "resume", []),
      /run is no longer resumable/,
    );
  });

  const retryBody = String(fetch.mock.calls.find(([url, init]) => String(url).endsWith("/runs") && init?.method === "POST")?.[1]?.body);
  const continueBody = String(fetch.mock.calls.find(([url]) => String(url).includes("/continue"))?.[1]?.body);
  assert.match(retryBody, /"conversation_id":"[^"]+"/);
  assert.doesNotMatch(continueBody, /attachment_ids/);
});

test("run read and review controls keep optional request fields absent when operators leave them blank", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/runs") && (!init?.method || init.method === "GET")) return json({ runs: [] });
    if (url.includes("/events")) return json({});
    if (url.includes("/approve")) return json({ result: {} });
    if (url.includes("/reject")) return new Response(null, { status: 204 });
    if (url.includes("/partial-approve")) return json({ result: {} });
    return json({});
  });
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns({ enabled: false }));

  await act(async () => {
    assert.deepEqual(await runs.result.current.getRunEvents("run-1"), []);
    await runs.result.current.approveRun("run-1", { actor: "   " });
    await runs.result.current.rejectRun("run-1", { actor: "   ", reason: "not ready" });
    await runs.result.current.partialApproveRun("run-1", [], "   ", "");
  });

  const bodies = fetch.mock.calls.map(([, init]) => String((init as RequestInit | undefined)?.body));
  const reviews = bodies.filter((body) => body !== "undefined").join("\n");
  assert.doesNotMatch(reviews, /"actor"/);
  assert.doesNotMatch(reviews, /"force"/);
  assert.doesNotMatch(reviews, /"commit_msg"/);
  assert.equal(fetch.mock.calls.some(([url]) => String(url).endsWith("/events")), true);
});

test("createRun serializes complete inline governance overrides and clears intentionally empty lists", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/runs") && init?.method === "POST") return json({ run: { id: "created" } });
    return json({ runs: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns({ enabled: false }));

  await act(async () => {
    await runs.result.current.createRun({
      taskId: "task-1",
      agentProfileId: "profile-1",
      roleRef: "  investigator ",
      maxTurns: 12,
      timeoutMinutes: 3,
      effort: "high",
      allowedTools: [],
      deniedTools: ["Shell"],
      skipPermissionPrompt: true,
      sandboxMode: "protected",
      networkAccess: "none",
      allowedPaths: [],
      deniedPaths: [".env"],
      features: { enableBrowser: true },
      extraFlags: {},
      conversationId: "conversation-1",
    });
  });

  const body = String(fetch.mock.calls.find(([url, init]) => String(url).endsWith("/runs") && init?.method === "POST")?.[1]?.body);
  for (const expected of [
    '"role_ref":"investigator"', '"max_turns":12', '"timeout":"180s"', '"effort":"high"',
    '"clear_allowed_tools":true', '"denied_tools":["Shell"]', '"skip_permission_prompt":true',
    '"mode":"SANDBOX_MODE_PROTECTED"', '"network_access":"NETWORK_ACCESS_NONE"',
    '"clear_allowed_paths":true', '"denied_paths":[".env"]', '"enable_browser":true',
    '"clear_extra_flags":true', '"conversation_id":"conversation-1"',
  ]) {
    assert.ok(body.includes(expected), `expected serialized run configuration to contain ${expected}`);
  }
});

test("createRun explicitly clears every intentionally empty governance collection", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/runs") && init?.method === "POST") return json({ run: { id: "created" } });
    return json({ runs: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns({ enabled: false }));

  await act(async () => {
    await runs.result.current.createRun({
      taskId: "task-1",
      agentProfileId: "profile-1",
      deniedTools: [],
      deniedPaths: [],
      extraFlags: { codex: [] },
    });
  });

  const body = String(fetch.mock.calls.find(([url, init]) => String(url).endsWith("/runs") && init?.method === "POST")?.[1]?.body);
  for (const expected of [
    '"clear_denied_tools":true',
    '"clear_denied_paths":true',
  ]) {
    assert.ok(body.includes(expected), `expected explicit empty collection contract to contain ${expected}`);
  }
});

test("profile and task controls preserve editable governance fields across create, update, read, cancel, and delete", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (init?.method === "DELETE" || url.endsWith("/cancel")) return new Response(null, { status: 204 });
    if (url.includes("/profiles/") && init?.method === "PUT") return json({ profile: { id: "profile-1", name: "Updated" } });
    if (url.endsWith("/profiles") && init?.method === "POST") return json({ profile: { id: "profile-1", name: "Created" } });
    if (url.includes("/tasks/") && init?.method === "PUT") return json({ task: { id: "task-1", title: "Updated" } });
    if (url.endsWith("/tasks") && init?.method === "POST") return json({ task: { id: "task-1", title: "Created" } });
    if (url.endsWith("/tasks/task-1")) return json({ task: { id: "task-1", title: "Read" } });
    if (url.endsWith("/profiles")) return json({ profiles: [] });
    return json({ tasks: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const profiles = renderHook(() => useProfiles({ enabled: false }));
  const tasks = renderHook(() => useTasks({ enabled: false }));
  await act(async () => {
    await profiles.result.current.createProfile({ name: "Investigator", roleRef: "investigator", allowedTools: ["Read"], sandboxMode: "protected", networkAccess: "localhost" });
    await profiles.result.current.updateProfile("profile-1", { name: "Updated", roleRef: "investigator", effort: "high" });
    await profiles.result.current.deleteProfile("profile-1");
    await tasks.result.current.createTask({ title: "Inspect", description: "capture evidence", scopePath: "api:cli", projectRoot: "/repo", contextAttachments: [{ type: "note", content: "focus tools" }] });
    await tasks.result.current.updateTask("task-1", { title: "Updated", scopePath: "api" });
    await tasks.result.current.getTask("task-1"); await tasks.result.current.cancelTask("task-1"); await tasks.result.current.deleteTask("task-1");
  });
  const bodies = fetch.mock.calls.map(([, init]) => String((init as RequestInit | undefined)?.body)).join("\n");
  assert.match(bodies, /"role_ref":"investigator"/); assert.match(bodies, /"context_attachments"/); assert.match(bodies, /"scope_path":"api:cli"/);
});
