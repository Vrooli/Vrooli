import { afterEach, describe, expect, it, vi } from "vitest";

import {
  acquireSession,
  auditRecords,
  connectDevice,
  killSession,
  listDevices,
  listSessions,
  listStrategies,
  releaseSession,
  runFlow,
  validateFlow,
} from "./deviceControl";
import {
  getAuthProfile,
  listAuthProfiles,
  testAuthProfile,
  unlockDevice,
  updateAuthProfile,
} from "./authentication";

// The request modules are intentionally thin, but their shared error contract
// is important: operator surfaces must receive server messages/codes rather
// than a generic fetch failure.
describe("device-control API request adapters", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("covers all read and mutation adapters on successful responses", async () => {
    const fetchMock = vi.fn(() => new Response(JSON.stringify({ ok: true }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await listAuthProfiles();
    await getAuthProfile("profile/1");
    await updateAuthProfile("profile/1", { status: "active" });
    await testAuthProfile("profile/1");
    await unlockDevice("profile/1", "device/1", "operator", "lease");
    await listDevices();
    await listStrategies();
    await listSessions();
    await killSession("session/1");
    await releaseSession("session/1");
    await acquireSession("device/1", "operator");
    await connectDevice("android");
    await validateFlow("strategy", { steps: [] });
    await runFlow("device/1", "operator", "lease", { steps: [] });
    await auditRecords();

    expect(fetchMock).toHaveBeenCalledTimes(15);
    const calls = fetchMock.mock.calls as unknown as Array<[string, RequestInit]>;
    expect(String(calls[1]?.[0])).toContain("profile%2F1");
  });

  it("prefers structured server messages, then codes, then status", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ message: "denied" }), { status: 403 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ code: "not_ready" }), { status: 409 }))
      .mockResolvedValueOnce(new Response("not-json", { status: 500 }))
      .mockResolvedValueOnce(new Response(JSON.stringify("denied"), { status: 403 }))
      .mockResolvedValueOnce(new Response(JSON.stringify("denied"), { status: 403 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(listAuthProfiles()).rejects.toThrow("denied");
    await expect(listDevices()).rejects.toThrow("not_ready");
    await expect(listStrategies()).rejects.toThrow("Request failed (500)");
    await expect(listAuthProfiles()).rejects.toThrow("Request failed (403)");
    await expect(listDevices()).rejects.toThrow("Request failed (403)");
  });
});
