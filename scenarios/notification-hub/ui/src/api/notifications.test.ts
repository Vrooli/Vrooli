import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { identityClient, recipientsClient, registerBrowserPushSubscription } from "./notifications";

describe("browser push registration", () => {
  const register = vi.spyOn(recipientsClient, "registerPushSubscription");

  beforeEach(() => {
    register.mockResolvedValue({} as never);
    Object.defineProperty(navigator, "serviceWorker", {
      configurable: true,
      value: {
        ready: Promise.resolve({
          pushManager: {
            subscribe: vi.fn().mockResolvedValue({
              toJSON: () => ({
                endpoint: "https://push.example/subscription",
                keys: { p256dh: "client-key", auth: "client-auth" },
              }),
            }),
          },
        }),
      },
    });
  });

  afterEach(() => {
    register.mockReset();
  });

  it("subscribes, persists the complete browser address, and returns its JSON form", async () => {
    const result = await registerBrowserPushSubscription(new Uint8Array([1, 2, 3]));

    expect(result.endpoint).toBe("https://push.example/subscription");
    expect(register).toHaveBeenCalledWith({
      endpoint: "https://push.example/subscription",
      p256dh: "client-key",
      auth: "client-auth",
      origin: window.location.origin,
    });
  });

  it("rejects a provider response that omits encryption keys", async () => {
    Object.defineProperty(navigator, "serviceWorker", {
      configurable: true,
      value: {
        ready: Promise.resolve({
          pushManager: {
            subscribe: vi.fn().mockResolvedValue({ toJSON: () => ({ endpoint: "https://push.example/incomplete" }) }),
          },
        }),
      },
    });

    await expect(registerBrowserPushSubscription(new Uint8Array([1]))).rejects.toThrow(
      "browser returned an incomplete push subscription",
    );
    expect(register).not.toHaveBeenCalled();
  });

  it("announces an expired owner session on a protected API 401", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("", { status: 401 }));
    const expired = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    window.addEventListener("notification-hub:session-expired", expired);
    try {
      await expect(identityClient.getSession({})).rejects.toBeDefined();
      expect(expired).toHaveBeenCalledTimes(1);
      expect(fetchMock).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ credentials: "same-origin" }));
    } finally {
      window.removeEventListener("notification-hub:session-expired", expired);
      vi.unstubAllGlobals();
    }
  });

  it("forwards routed test mode and selected fixtures only when explicitly requested", async () => {
    window.history.replaceState({}, "", "/settings/reach?vrooli_test_mode=1&e21_fixture=webpush-content-access&e22_fixture=settings-owner&e22_case=e21-device-read-failure-dark-phone&e22_fault=device-read-failure");
    const fetchMock = vi.fn().mockResolvedValue(new Response("", { status: 401 }));
    vi.stubGlobal("fetch", fetchMock);
    try {
      await expect(identityClient.getSession({})).rejects.toBeDefined();
      const [, init] = fetchMock.mock.calls[0] as [RequestInfo | URL, RequestInit];
      expect(new Headers(init.headers).get("X-Vrooli-Test-Mode")).toBe("1");
      expect(new Headers(init.headers).get("X-Notification-Hub-E21-Fixture")).toBe("webpush-content-access");
      expect(new Headers(init.headers).get("X-Notification-Hub-E22-Fixture")).toBe("settings-owner");
      expect(new Headers(init.headers).get("X-Notification-Hub-E22-Case")).toBe("e21-device-read-failure-dark-phone");
      expect(new Headers(init.headers).get("X-Notification-Hub-E22-Fault")).toBe("device-read-failure");

      window.history.replaceState({}, "", "/settings?view=all&vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=e21-device-read-failure-dark-phone&e22_fault=device-read-failure");
      await expect(recipientsClient.listDevices({})).rejects.toBeDefined();
      const settingsInit = vi.mocked(fetch).mock.calls.at(-1)?.[1] as RequestInit;
      expect(new Headers(settingsInit.headers).has("X-Notification-Hub-E22-Fault")).toBe(false);

      window.history.replaceState({}, "", "/");
      fetchMock.mockClear();
      await expect(identityClient.getSession({})).rejects.toBeDefined();
      const [, ordinaryInit] = fetchMock.mock.calls[0] as [RequestInfo | URL, RequestInit];
      expect(new Headers(ordinaryInit.headers).has("X-Vrooli-Test-Mode")).toBe(false);
      expect(new Headers(ordinaryInit.headers).has("X-Notification-Hub-E21-Fixture")).toBe(false);
      expect(new Headers(ordinaryInit.headers).has("X-Notification-Hub-E22-Fixture")).toBe(false);
      expect(new Headers(ordinaryInit.headers).has("X-Notification-Hub-E22-Case")).toBe(false);
      expect(new Headers(ordinaryInit.headers).has("X-Notification-Hub-E22-Fault")).toBe(false);
    } finally {
      window.history.replaceState({}, "", "/");
      vi.unstubAllGlobals();
    }
  });
});
