import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { recipientsClient, registerBrowserPushSubscription } from "./notifications";

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
});
