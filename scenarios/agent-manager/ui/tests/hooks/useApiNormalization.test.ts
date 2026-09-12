import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { ensureProfile, useProfiles, useRuns } from "../../src/hooks/useApi.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

test("profile writes normalize generated and supplied keys with each supported sandbox and network mode", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/profiles") && init?.method === "POST") return json({ profile: { id: "profile-1" } });
    return json({ profiles: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const profiles = renderHook(() => useProfiles({ enabled: false }));

  await act(async () => {
    await profiles.result.current.createProfile({
      name: "  Investigation / Triage  ",
      profileKey: "   ",
      roleRef: "investigator",
      sandboxMode: "off",
      networkAccess: "none",
    });
    await profiles.result.current.createProfile({
      name: "Implementation",
      profileKey: "  implementation-review  ",
      roleRef: "implementer",
      sandboxMode: "tracking",
      networkAccess: "full",
    });
  });

  const writes = fetch.mock.calls.filter(([url, init]) => String(url).endsWith("/profiles") && (init as RequestInit)?.method === "POST");
  assert.equal(writes.length, 2);
  const first = String(writes[0]?.[1]?.body);
  const second = String(writes[1]?.[1]?.body);
  assert.match(first, /"profile_key":"investigation-triage"/);
  assert.match(first, /"mode":"SANDBOX_MODE_OFF"/);
  assert.match(first, /"network_access":"NETWORK_ACCESS_NONE"/);
  assert.match(second, /"profile_key":"implementation-review"/);
  assert.match(second, /"mode":"SANDBOX_MODE_TRACKING"/);
  assert.match(second, /"network_access":"NETWORK_ACCESS_FULL"/);
});

test("run creation omits unset inline overrides while minting a conversation id", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/runs") && init?.method === "POST") return json({ run: { id: "run-1" } });
    return json({ runs: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const runs = renderHook(() => useRuns({ enabled: false }));

  await act(async () => {
    await runs.result.current.createRun({ taskId: "task-1", agentProfileId: "profile-1" });
  });

  const body = String(fetch.mock.calls.find(([url, init]) => String(url).endsWith("/runs") && (init as RequestInit)?.method === "POST")?.[1]?.body);
  assert.match(body, /"conversation_id":"[^"]+"/);
  assert.doesNotMatch(body, /"inline_config"/);
});

test("standard API envelopes prioritize a non-empty operator message over generic HTTP status", async () => {
  const fetch = vi.fn(async () => json({
    code: "RUN_BLOCKED",
    message: "generic backend detail",
    details: { fields: { user_message: { string_value: "Choose an approved profile first" } } },
  }, 409));
  vi.stubGlobal("fetch", fetch);
  const profiles = renderHook(() => useProfiles());

  await waitFor(() => assert.equal(profiles.result.current.error, "Choose an approved profile first"));
  assert.deepEqual(profiles.result.current.data, []);
});

test("profile creation generates a safe fallback key and preserves runner-specific flags", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/profiles") && init?.method === "POST") return json({ profile: { id: "profile-fallback" } });
    return json({ profiles: [] });
  });
  vi.stubGlobal("fetch", fetch);
  vi.spyOn(Math, "random").mockReturnValue(0.123456789);
  const profiles = renderHook(() => useProfiles({ enabled: false }));

  await act(async () => {
    await profiles.result.current.createProfile({
      name: " !!! ",
      roleRef: "investigator",
      extraFlags: { codex: ["--json", "--sandbox=protected"] },
    });
  });

  const body = String(fetch.mock.calls.find(([url, init]) => String(url).endsWith("/profiles") && (init as RequestInit)?.method === "POST")?.[1]?.body);
  assert.match(body, /"profile_key":"profile-4fzzzx"/);
  assert.match(body, /"codex":\{"flags":\["--json","--sandbox=protected"\]\}/);
});

test("profile and run forms preserve explicit false feature flags and fall back safely for malformed select values", async () => {
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/profiles") && init?.method === "POST") return json({ profile: { id: "profile-1" } });
    if (url.endsWith("/runs") && init?.method === "POST") return json({ run: { id: "run-1" } });
    return json({ runs: [], profiles: [] });
  });
  vi.stubGlobal("fetch", fetch);
  const profiles = renderHook(() => useProfiles({ enabled: false }));
  const runs = renderHook(() => useRuns({ enabled: false }));

  await act(async () => {
    await profiles.result.current.createProfile({
      name: "Fallback forms",
      roleRef: "investigator",
      features: { enableBrowser: false },
      sandboxMode: "unexpected" as never,
      networkAccess: "unexpected" as never,
    });
    await runs.result.current.createRun({
      taskId: "task-1",
      agentProfileId: "profile-1",
      allowedTools: ["Read"],
      allowedPaths: ["src"],
      features: { enableBrowser: false },
      sandboxMode: "unexpected" as never,
      networkAccess: "unexpected" as never,
    });
  });

  const profileBody = String(fetch.mock.calls.find(([url, init]) => String(url).endsWith("/profiles") && (init as RequestInit)?.method === "POST")?.[1]?.body);
  const runBody = String(fetch.mock.calls.find(([url, init]) => String(url).endsWith("/runs") && (init as RequestInit)?.method === "POST")?.[1]?.body);
  assert.doesNotMatch(profileBody, /enable_browser/);
  assert.match(profileBody, /"network_access":"NETWORK_ACCESS_LOCALHOST"/);
  assert.match(runBody, /"allowed_tools":\["Read"\]/);
  assert.doesNotMatch(runBody, /clear_allowed_tools|clear_allowed_paths/);
  // Proto JSON elides explicit default values, while retaining the feature and
  // sandbox message itself so the API can apply its contract defaults.
  assert.match(runBody, /"features":\{\}/);
  assert.match(runBody, /"sandbox_config":\{\}/);
  assert.match(runBody, /"network_access":"NETWORK_ACCESS_LOCALHOST"/);
});

test("ensureProfile returns the canonical profile projection through the shared API client", async () => {
  const fetch = vi.fn(async () => json({ profile: { id: "existing-profile", profileKey: "agent-manager-investigation" } }));
  vi.stubGlobal("fetch", fetch);

  const profile = await ensureProfile("agent-manager-investigation");

  assert.equal(profile.id, "existing-profile");
  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/profiles/ensure");
  assert.equal((fetch.mock.calls[0]?.[1] as RequestInit).method, "POST");
  assert.match(String((fetch.mock.calls[0]?.[1] as RequestInit).body), /agent-manager-investigation/);
});
