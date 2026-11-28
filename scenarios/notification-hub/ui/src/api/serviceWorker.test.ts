import { afterEach, describe, expect, it, vi } from "vitest";
import { availableServiceWorkerVersion, forceServiceWorkerUpdate, readConnectionStatus, runningServiceWorkerVersion } from "./serviceWorker";

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

describe("runningServiceWorkerVersion", () => {
  it("is null where service workers do not exist", async () => {
    vi.stubGlobal("navigator", {});
    expect(await runningServiceWorkerVersion()).toBeNull();
  });

  it("is null when nothing is registered or active yet", async () => {
    vi.stubGlobal("navigator", { serviceWorker: { getRegistration: () => Promise.resolve(undefined) } });
    expect(await runningServiceWorkerVersion()).toBeNull();
    vi.stubGlobal("navigator", { serviceWorker: { getRegistration: () => Promise.resolve({ active: null }) } });
    expect(await runningServiceWorkerVersion()).toBeNull();
  });

  it("gives up after two seconds on a worker that never answers", async () => {
    vi.useFakeTimers();
    try {
      vi.stubGlobal("navigator", { serviceWorker: { getRegistration: () => Promise.resolve({ active: { postMessage: () => undefined } }) } });
      const pending = runningServiceWorkerVersion();
      await vi.advanceTimersByTimeAsync(2000);
      expect(await pending).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it("ignores a reply that carries no version", async () => {
    const active = { postMessage: (_m: unknown, ports: MessagePort[]) => ports[0]!.postMessage({ type: "VERSION" }) };
    vi.stubGlobal("navigator", { serviceWorker: { getRegistration: () => Promise.resolve({ active }) } });
    expect(await runningServiceWorkerVersion()).toBeNull();
  });
});

describe("readConnectionStatus where push is not possible", () => {
  it("reports unsupported, uncontrolled and unsubscribed without throwing", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: false })));
    vi.stubGlobal("navigator", {});
    vi.stubGlobal("Notification", undefined);
    const status = await readConnectionStatus();
    expect(status).toMatchObject({ supported: false, permission: "unsupported", controlled: false, stale: false, subscribed: false, endpointHost: "" });
  });

  it("shows no host for a push endpoint that is not a URL", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: false })));
    const registration = { active: null, pushManager: { getSubscription: () => Promise.resolve({ endpoint: "not a url" }) } };
    vi.stubGlobal("navigator", { serviceWorker: { controller: null, getRegistration: () => Promise.resolve(registration) } });
    vi.stubGlobal("Notification", { permission: "default" });
    const status = await readConnectionStatus();
    expect(status.endpointHost).toBe("");
    expect(status.subscribed).toBe(false);
  });

  it("does not call a worker stale when the deployed version cannot be read", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.reject(new Error("offline"))));
    vi.stubGlobal("navigator", { serviceWorker: { controller: {}, getRegistration: () => Promise.resolve({ active: null, pushManager: { getSubscription: () => Promise.resolve(null) } }) } });
    vi.stubGlobal("PushManager", function () {});
    vi.stubGlobal("Notification", { permission: "granted" });
    expect((await readConnectionStatus()).stale).toBe(false);
  });
});

describe("forceServiceWorkerUpdate", () => {
  const withRegistration = (registration: unknown) => {
    const addEventListener = vi.fn();
    vi.stubGlobal("navigator", { serviceWorker: { getRegistration: () => Promise.resolve(registration), addEventListener } });
    return addEventListener;
  };

  it("does nothing where service workers do not exist or none is registered", async () => {
    vi.stubGlobal("navigator", {});
    expect(await forceServiceWorkerUpdate()).toBe(false);
    withRegistration(undefined);
    expect(await forceServiceWorkerUpdate()).toBe(false);
  });

  it("promotes a waiting worker and reloads once it takes control", async () => {
    const postMessage = vi.fn();
    const update = vi.fn(() => Promise.resolve());
    const addEventListener = withRegistration({ update, waiting: { postMessage }, installing: null });
    expect(await forceServiceWorkerUpdate()).toBe(true);
    expect(update).toHaveBeenCalled();
    expect(postMessage).toHaveBeenCalledWith({ type: "SKIP_WAITING" });
    expect(addEventListener).toHaveBeenCalledWith("controllerchange", expect.any(Function), { once: true });
  });

  it("reports false when the re-check finds nothing newer", async () => {
    withRegistration({ update: () => Promise.resolve(), waiting: null, installing: null });
    expect(await forceServiceWorkerUpdate()).toBe(false);
  });
});
