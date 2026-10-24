import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { runInNewContext } from "node:vm";
import { afterEach, describe, expect, it, vi } from "vitest";
import { initPendingNavigation } from "./features/pendingNav";

type Listener = (event: Record<string, unknown>) => void;

interface FakeWindow {
  navigate?: (url: string) => Promise<FakeWindow | null>;
  focus: () => Promise<FakeWindow>;
  postMessage?: (message: unknown) => void;
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
  // Minimal Cache API so the notification-tap stash is testable as shipped.
  const stash = new Map<string, string>();
  const cache = {
    put: vi.fn((key: string, res: { _body: string }) => {
      stash.set(key, res._body);
      return Promise.resolve();
    }),
    match: vi.fn((key: string) => {
      const body = stash.get(key);
      return Promise.resolve(body === undefined ? undefined : {
        text: () => Promise.resolve(body),
        json: () => Promise.resolve(JSON.parse(body)),
      });
    }),
    delete: vi.fn((key: string) => Promise.resolve(stash.delete(key))),
  };
  const caches = {
    open: vi.fn(() => Promise.resolve(cache)),
    keys: vi.fn(() => Promise.resolve([])),
    delete: vi.fn(() => Promise.resolve(true)),
    match: vi.fn(() => Promise.resolve(undefined)),
  };
  const FakeResponse = class {
    _body: string;
    constructor(body: string) {
      this._body = body;
    }
    json() {
      return Promise.resolve(JSON.parse(this._body));
    }
  };
  const fetch = vi.fn(() => Promise.resolve({ ok: true }));
  const source = readFileSync(resolve(__dirname, "../public/sw.js"), "utf8");
  runInNewContext(source, {
    self: fakeSelf,
    clients: fakeClients,
    caches,
    Response: FakeResponse,
    URL,
    fetch,
  });
  return {
    stash,
    cache,
    caches,
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
        preventDefault: vi.fn(),
        waitUntil: (promise: Promise<unknown>) => {
          settled = promise;
        },
      });
      await settled;
    },
  };
}

describe("service worker notification click", () => {
  it("stashes the target and opens the app when no window is open", async () => {
    // The worker can't drive navigation on iOS, so it records the target for
    // the app to act on and just launches the app at its root.
    const worker = loadWorker([]);
    await worker.click({
      id: "notification-1",
      url: "https://web-console.example.test/?session=s1",
    });
    expect(worker.openWindow).toHaveBeenCalledWith("./");
    const stashed = JSON.parse(worker.stash.get("/__pending_nav") ?? "{}");
    expect(stashed.url).toBe("https://web-console.example.test/?session=s1");
    expect(typeof stashed.ts).toBe("number");
  });

  it("posts the target to an open window and focuses it without opening a new one", async () => {
    const open: FakeWindow = {
      focus: vi.fn(() => Promise.resolve(open)),
      postMessage: vi.fn(),
    };
    const worker = loadWorker([open]);
    await worker.click({ id: "notification-1", url: "asks/ask-1" });
    expect(open.postMessage).toHaveBeenCalledWith({ type: "PENDING_NAV", url: "asks/ask-1" });
    expect(open.focus).toHaveBeenCalled();
    expect(worker.openWindow).not.toHaveBeenCalled();
    expect(JSON.parse(worker.stash.get("/__pending_nav") ?? "{}").url).toBe("asks/ask-1");
  });

  it("opens the app root and stashes nothing for a notification without a link", async () => {
    const worker = loadWorker([]);
    await worker.click({ id: "notification-1" });
    expect(worker.openWindow).toHaveBeenCalledWith("./");
    expect(worker.stash.has("/__pending_nav")).toBe(false);
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


function setPageLocation(href: string) {
  const current = new URL(href);
  const assign = vi.fn((next: string) => {
    current.href = next;
  });
  const location = {
    get href() { return current.href; },
    get hostname() { return current.hostname; },
    get origin() { return current.origin; },
    assign,
  };
  Object.defineProperty(window, "location", { configurable: true, value: location });
  return assign;
}

const originalLocation = Object.getOwnPropertyDescriptor(window, "location");

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  if (originalLocation) {
    Object.defineProperty(window, "location", originalLocation);
  }
});

function setDocumentVisibility(state: "hidden" | "visible") {
  let current = state;
  Object.defineProperty(document, "visibilityState", { configurable: true, get: () => current });
  return (next: "hidden" | "visible") => {
    current = next;
    document.dispatchEvent(new Event("visibilitychange"));
  };
}

describe("notification click to foreground navigation bridge", () => {
  const approvalURL = "https://agent-manager.itsagitime.com/api/v1/authorizations/ffe2823f-ea1f-4e77-88ab-630270ecfbef/approve";

  it("routes a warm hidden app when the worker stashes, posts, and focuses it", async () => {
    const assign = setPageLocation("https://notification-hub.itsagitime.com/");
    const setVisibility = setDocumentVisibility("hidden");
    const pageMessages: Array<(event: MessageEvent) => void> = [];
    vi.stubGlobal("navigator", {
      serviceWorker: {
        addEventListener: (_type: string, listener: (event: MessageEvent) => void) => pageMessages.push(listener),
        removeEventListener: () => {},
      },
    });
    let hasPendingTarget = () => false;
    const page: FakeWindow = {
      postMessage: (data) => pageMessages.forEach((listener) => listener({ data } as MessageEvent)),
      focus: async () => {
        // Model the app returning to the foreground after the click handler
        // has persisted its target.
        expect(hasPendingTarget()).toBe(true);
        setVisibility("visible");
        return page;
      },
    };
    const worker = loadWorker([page]);
    hasPendingTarget = () => worker.stash.has("/__pending_nav");
    vi.stubGlobal("caches", worker.caches);
    const dispose = initPendingNavigation();

    await worker.click({ id: "approval-reminder", url: approvalURL });
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(assign).toHaveBeenCalledWith(approvalURL);
    expect(worker.stash.has("/__pending_nav")).toBe(false);
    dispose();
  });

  it("routes an already-visible app from the worker message after the click stash", async () => {
    const assign = setPageLocation("https://notification-hub.itsagitime.com/");
    setDocumentVisibility("visible");
    const pageMessages: Array<(event: MessageEvent) => void> = [];
    vi.stubGlobal("navigator", {
      serviceWorker: {
        addEventListener: (_type: string, listener: (event: MessageEvent) => void) => pageMessages.push(listener),
        removeEventListener: () => {},
      },
    });
    let hasPendingTarget = () => false;
    const page: FakeWindow = {
      postMessage: (data) => {
        expect(hasPendingTarget()).toBe(true);
        pageMessages.forEach((listener) => listener({ data } as MessageEvent));
      },
      focus: async () => page,
    };
    const worker = loadWorker([page]);
    hasPendingTarget = () => worker.stash.has("/__pending_nav");
    vi.stubGlobal("caches", worker.caches);
    const dispose = initPendingNavigation();

    await worker.click({ id: "approval-reminder", url: approvalURL });
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(assign).toHaveBeenCalledWith(approvalURL);
    expect(worker.stash.has("/__pending_nav")).toBe(false);
    dispose();
  });

  it("routes a cold launch from the cache after the worker opens the app root", async () => {
    const assign = setPageLocation("https://notification-hub.itsagitime.com/");
    const worker = loadWorker([]);
    await worker.click({ id: "approval-reminder", url: approvalURL });
    expect(worker.openWindow).toHaveBeenCalledWith("./");

    vi.stubGlobal("caches", worker.caches);
    vi.stubGlobal("navigator", { serviceWorker: { addEventListener: () => {}, removeEventListener: () => {} } });
    const dispose = initPendingNavigation();
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(assign).toHaveBeenCalledWith(approvalURL);
    expect(worker.stash.has("/__pending_nav")).toBe(false);
    dispose();
  });

  it("writes the destination before any resumed-window message or focus", async () => {
    const order: string[] = [];
    const worker = loadWorker([{
      postMessage: () => { order.push(worker.stash.has("/__pending_nav") ? "message-after-stash" : "message-before-stash"); },
      focus: async () => { order.push(worker.stash.has("/__pending_nav") ? "focus-after-stash" : "focus-before-stash"); return {} as FakeWindow; },
    }]);
    await worker.click({ id: "approval-reminder", url: approvalURL });
    expect(order).toEqual(["message-after-stash", "focus-after-stash"]);
  });
});
