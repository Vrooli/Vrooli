import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { runInNewContext } from "node:vm";
import { describe, expect, it, vi } from "vitest";

type Listener = (event: Record<string, unknown>) => void;

interface FakeWindow {
  navigate?: (url: string) => Promise<FakeWindow | null>;
  focus: () => Promise<FakeWindow>;
}

/**
 * Loads public/sw.js into a fake worker global so the notification click
 * behaviour is tested as shipped, not as a copy.
 */
function loadWorker(windows: FakeWindow[]) {
  const listeners = new Map<string, Listener>();
  const openWindow = vi.fn(() => Promise.resolve(null));
  const fakeSelf = {
    registration: { scope: "https://hub.example.test/" },
    addEventListener: (type: string, listener: Listener) =>
      listeners.set(type, listener),
    skipWaiting: vi.fn(),
  };
  const fakeClients = {
    matchAll: vi.fn(() => Promise.resolve(windows)),
    openWindow,
    claim: vi.fn(),
  };
  const fetch = vi.fn(() => Promise.resolve({ ok: true }));
  const source = readFileSync(resolve(__dirname, "../public/sw.js"), "utf8");
  runInNewContext(source, {
    self: fakeSelf,
    clients: fakeClients,
    caches: {},
    URL,
    fetch,
  });
  return {
    openWindow,
    fetch,
    async changeSubscription(replacement: unknown, oldOptions: unknown) {
      const subscribe = vi.fn(() =>
        Promise.resolve({ toJSON: () => replacement }),
      );
      Object.assign(fakeSelf.registration, { pushManager: { subscribe } });
      let settled: Promise<unknown> = Promise.resolve();
      listeners.get("pushsubscriptionchange")?.({
        oldSubscription: { options: oldOptions },
        waitUntil: (promise: Promise<unknown>) => {
          settled = promise;
        },
      });
      await settled;
      return subscribe;
    },
    async click(data: Record<string, string>) {
      let settled: Promise<unknown> = Promise.resolve();
      listeners.get("notificationclick")?.({
        notification: { close: vi.fn(), data },
        waitUntil: (promise: Promise<unknown>) => {
          settled = promise;
        },
      });
      await settled;
    },
  };
}

describe("service worker notification click", () => {
  it("opens the ask page the push named when no window is open", async () => {
    const worker = loadWorker([]);
    await worker.click({ id: "notification-1", url: "asks/ask-1" });
    expect(worker.openWindow).toHaveBeenCalledWith(
      "https://hub.example.test/asks/ask-1",
    );
  });

  it("navigates an open window to the ask page instead of only focusing it", async () => {
    const open: FakeWindow = {
      focus: vi.fn(() => Promise.resolve(open)),
      navigate: vi.fn(() => Promise.resolve(open)),
    };
    const worker = loadWorker([open]);
    await worker.click({ id: "notification-1", url: "asks/ask-1" });
    expect(open.navigate).toHaveBeenCalledWith(
      "https://hub.example.test/asks/ask-1",
    );
    expect(open.focus).toHaveBeenCalled();
    expect(worker.openWindow).not.toHaveBeenCalled();
  });

  it("never follows a push link to another origin", async () => {
    const worker = loadWorker([]);
    await worker.click({
      id: "notification-1",
      url: "https://evil.example/phish",
    });
    expect(worker.openWindow).toHaveBeenCalledWith("https://hub.example.test/");
  });

  it("opens the app root for a notification without a link", async () => {
    const worker = loadWorker([]);
    await worker.click({ id: "notification-1" });
    expect(worker.openWindow).toHaveBeenCalledWith("https://hub.example.test/");
  });

  it("re-registers a replaced push subscription with the hub, keeping the push key", async () => {
    const worker = loadWorker([]);
    const oldOptions = {
      userVisibleOnly: true,
      applicationServerKey: "vapid-key",
    };
    const subscribe = await worker.changeSubscription(
      {
        endpoint: "https://push.example/new",
        keys: { p256dh: "client-key", auth: "client-auth" },
      },
      oldOptions,
    );
    expect(subscribe).toHaveBeenCalledWith(oldOptions);
    expect(worker.fetch).toHaveBeenCalledWith(
      "https://hub.example.test/vrooli.notification_hub.v1.recipients.RecipientsService/RegisterPushSubscription",
      expect.objectContaining({ method: "POST", credentials: "same-origin" }),
    );
    const [, init] = worker.fetch.mock.calls[0] as unknown as [
      string,
      { body: string },
    ];
    expect(JSON.parse(init.body)).toEqual({
      endpoint: "https://push.example/new",
      p256dh: "client-key",
      auth: "client-auth",
      origin: "https://hub.example.test",
    });
  });
});
