import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useProfiles, useTasks } from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
}

test("profile and task controls serialize edits and refresh their canonical lists", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.endsWith("/profiles")) return json({ profiles: [] });
    if (url.endsWith("/tasks")) return json({ tasks: [] });
    return json({});
  });
  vi.stubGlobal("fetch", fetch);
  const profiles = renderHook(() => useProfiles());
  const tasks = renderHook(() => useTasks());
  await waitFor(() => assert.equal(fetch.mock.calls.length, 2));

  await act(async () => {
    await profiles.result.current.createProfile({ name: "Investigator", profileKey: "investigator", roleRef: "agent-manager/investigate", maxTurns: 12, timeoutMinutes: 15, sandboxMode: "protected", networkAccess: "localhost", allowedTools: ["bash"] });
    await profiles.result.current.updateProfile("profile-a", { name: "Investigator", roleRef: "agent-manager/investigate", deniedTools: ["rm"] });
    await profiles.result.current.deleteProfile("profile-a");
    await tasks.result.current.createTask({ title: "Inspect run", description: "Find friction", scopePath: "scenarios/agent-manager", projectRoot: "/workspace" });
    await tasks.result.current.updateTask("task-a", { title: "Inspect run", scopePath: "scenarios/agent-manager" });
    await tasks.result.current.getTask("task-a");
    await tasks.result.current.cancelTask("task-a");
    await tasks.result.current.deleteTask("task-a");
  });

  const calls = fetch.mock.calls.map(([url, options]) => [String(url), options] as const);
  const request = (path: string) => calls.find(([url]) => url.includes(path));
  assert.equal(request("/profiles/profile-a")?.[1]?.method, "PUT");
  assert.match(String(request("/profiles/profile-a")?.[1]?.body), /agent-manager\/investigate/);
  assert.equal(request("/tasks/task-a/cancel")?.[1]?.method, "POST");
  assert.equal(request("/tasks/task-a")?.[1]?.method, "PUT");
  assert.match(String(request("/tasks/task-a")?.[1]?.body), /scenarios\/agent-manager/);
  assert.ok(calls.some(([url, options]) => url.endsWith("/profiles/profile-a") && options?.method === "DELETE"));
  assert.ok(calls.some(([url, options]) => url.endsWith("/tasks/task-a") && options?.method === "DELETE"));
});
