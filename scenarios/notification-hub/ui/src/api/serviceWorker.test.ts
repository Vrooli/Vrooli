import { afterEach, describe, expect, it, vi } from "vitest";
import { availableServiceWorkerVersion, readConnectionStatus } from "./serviceWorker";

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("availableServiceWorkerVersion", () => {
  it("parses the SW_VERSION literal out of the served worker source", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: true, text: () => Promise.resolve('const SW_VERSION = "2026-10-14.4";\n') })));
    expect(await availableServiceWorkerVersion()).toBe("2026-10-14.4");
  });

  it("returns null when the worker source is unreadable", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: false })));
    expect(await availableServiceWorkerVersion()).toBeNull();
  });

  it("returns null (not a throw) when the fetch itself fails", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.reject(new Error("offline"))));
    expect(await availableServiceWorkerVersion()).toBeNull();
  });
});

describe("readConnectionStatus staleness", () => {
  it("flags stale only when both versions are known and differ", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: true, text: () => Promise.resolve('const SW_VERSION = "2026-10-14.5";') })));
    const active = { postMessage: (_m: unknown, ports: MessagePort[]) => ports[0]!.postMessage({ type: "VERSION", version: "2026-10-14.4" }) };
    const registration = { active, pushManager: { getSubscription: () => Promise.resolve({ endpoint: "https://web.push.apple.com/abc" }) } };
    vi.stubGlobal("navigator", { serviceWorker: { controller: {}, getRegistration: () => Promise.resolve(registration) } });
    vi.stubGlobal("PushManager", function () {});
    vi.stubGlobal("Notification", { permission: "granted" });

    const status = await readConnectionStatus();
    expect(status.runningVersion).toBe("2026-10-14.4");
    expect(status.availableVersion).toBe("2026-10-14.5");
    expect(status.stale).toBe(true);
    expect(status.subscribed).toBe(true);
    expect(status.endpointHost).toBe("web.push.apple.com");
    expect(status.permission).toBe("granted");
  });

  it("flags stale when a controlling worker is too old to report its version", async () => {
    // Old worker: no GET_VERSION handler, so runningServiceWorkerVersion times
    // out to null. It is still in control, and a newer version is deployed.
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: true, text: () => Promise.resolve('const SW_VERSION = "2026-10-14.4";') })));
    const active = { postMessage: () => {} }; // never replies
    const registration = { active, pushManager: { getSubscription: () => Promise.resolve(null) } };
    vi.stubGlobal("navigator", { serviceWorker: { controller: {}, getRegistration: () => Promise.resolve(registration) } });
    vi.stubGlobal("PushManager", function () {});
    vi.stubGlobal("Notification", { permission: "granted" });

    const status = await readConnectionStatus();
    expect(status.runningVersion).toBeNull();
    expect(status.availableVersion).toBe("2026-10-14.4");
    expect(status.stale).toBe(true);
  });

  it("is not stale when the running and available versions match", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: true, text: () => Promise.resolve('const SW_VERSION = "2026-10-14.4";') })));
    const active = { postMessage: (_m: unknown, ports: MessagePort[]) => ports[0]!.postMessage({ type: "VERSION", version: "2026-10-14.4" }) };
    const registration = { active, pushManager: { getSubscription: () => Promise.resolve(null) } };
    vi.stubGlobal("navigator", { serviceWorker: { controller: {}, getRegistration: () => Promise.resolve(registration) } });
    vi.stubGlobal("PushManager", function () {});
    vi.stubGlobal("Notification", { permission: "default" });

    const status = await readConnectionStatus();
    expect(status.stale).toBe(false);
    expect(status.subscribed).toBe(false);
    expect(status.controlled).toBe(true);
  });
});
