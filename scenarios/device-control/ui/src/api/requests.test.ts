import { afterEach, describe, expect, it, vi } from "vitest";

import {
  DeviceControlApiError,
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
    await connectDevice("google-tv");
    await validateFlow("strategy", { steps: [] });
    await runFlow("device/1", "operator", "lease", { steps: [] });
    await auditRecords();

    expect(fetchMock).toHaveBeenCalledTimes(15);
    const calls = fetchMock.mock.calls as unknown as Array<[string, RequestInit]>;
    expect(String(calls[1]?.[0])).toContain("profile%2F1");
  });

  it("preserves Device Control error codes and HTTP status", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ message: "denied" }), { status: 403 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ code: "not_ready" }), { status: 409 }))
      .mockResolvedValueOnce(new Response("not-json", { status: 500 }))
      .mockResolvedValueOnce(new Response(JSON.stringify("denied"), { status: 403 }))
      .mockResolvedValueOnce(new Response(JSON.stringify("denied"), { status: 403 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(listAuthProfiles()).rejects.toThrow("denied");
    await expect(listDevices()).rejects.toMatchObject({ code: "not_ready", status: 409, message: "not_ready: Device Control request failed (HTTP 409)" });
    await expect(listStrategies()).rejects.toMatchObject({ code: "request_failed", status: 500, message: "request_failed: Device Control request failed (HTTP 500)" });
    await expect(listAuthProfiles()).rejects.toThrow("Request failed (403)");
    await expect(listDevices()).rejects.toMatchObject({ code: "request_failed", status: 403, message: "request_failed: Device Control request failed (HTTP 403)" });
  });

  it("shows a sanitized UI-proxy error for failed flow capture requests", async () => {
    const fetchMock = vi.fn(() => new Response(JSON.stringify({
      error: "API server unavailable",
      details: "connect ECONNREFUSED 127.0.0.1:16465",
      target: "http://127.0.0.1:16465/api/v1/flows/run?token=private",
    }), {
      status: 502,
      headers: { "X-Vrooli-Error-Hop": "ui-proxy" },
    }));
    vi.stubGlobal("fetch", fetchMock);

    const error = await runFlow("device/1", "browser-operator", "lease", { steps: [{ kind: "observe" }] })
      .catch((cause: unknown) => cause);

    expect(error).toBeInstanceOf(DeviceControlApiError);
    expect(error).toMatchObject({
      code: "api_unreachable_or_timeout",
      status: 502,
      message: "api_unreachable_or_timeout: Could not reach the API server or the request timed out (HTTP 502)",
    });
    expect((error as Error).message).not.toContain("127.0.0.1");
    expect((error as Error).message).not.toContain("private");
  });

  it("keeps structured flow error code, message, and status without exposing metadata", async () => {
    vi.stubGlobal("fetch", vi.fn(() => new Response(JSON.stringify({
      code: "run_failed",
      message: "Observe step failed",
      details: "private internal diagnostic",
      target: "http://private-host/",
    }), { status: 409 })));

    const error = await runFlow("device/1", "browser-operator", "lease", { steps: [{ kind: "observe" }] })
      .catch((cause: unknown) => cause);

    expect(error).toBeInstanceOf(DeviceControlApiError);
    expect(error).toMatchObject({
      code: "run_failed",
      status: 409,
      message: "run_failed: Observe step failed (HTTP 409)",
    });
    expect((error as Error).message).not.toContain("private internal diagnostic");
    expect((error as Error).message).not.toContain("private-host");
  });
});
