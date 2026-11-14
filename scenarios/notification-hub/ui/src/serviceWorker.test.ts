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
function loadWorker(windows: FakeWindow[], scope = "https://hub.example.test/") {
  const listeners = new Map<string, Listener>();
  const openWindow = vi.fn(() => Promise.resolve(null));
  const fakeSelf = {
    registration: { scope },
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

describe("service worker notification tap destination", () => {
  const stashedURL = async (url: unknown, scope?: string) => {
    const worker = loadWorker([], scope);
    await worker.click({ id: "notification-1", url } as Record<string, string>);
    const raw = worker.stash.get("/__pending_nav");
    return { stashed: raw === undefined ? null : (JSON.parse(raw) as { url: string }).url, worker };
  };

  describe("with the Hub served at an origin root", () => {
    const scope = "https://hub.example.test/";

    it.each([
      ["a record path", "asks/ask-1", "https://hub.example.test/asks/ask-1"],
      ["a delivery record path", "delivery/n-1", "https://hub.example.test/delivery/n-1"],
      ["a Hub-relative path with a leading slash", "/asks/ask-1", "https://hub.example.test/asks/ask-1"],
      ["a path with a query", "/delivery/n-1?tab=attempts", "https://hub.example.test/delivery/n-1?tab=attempts"],
      ["an absolute URL on the Hub's own origin", "https://hub.example.test/asks/ask-1", "https://hub.example.test/asks/ask-1"],
    ])("opens %s inside the Hub", async (_name, url, expected) => {
      expect((await stashedURL(url, scope)).stashed).toBe(expected);
    });

    it.each([
      ["another site", "https://evil.example/phish"],
      ["a sibling subdomain of the Hub's site", "https://web-console.example.test/?session=s1"],
      ["a lookalike of the Hub's host", "https://hub.example.test.evil.example/"],
      ["an absolute URL whose userinfo imitates the Hub's host", "https://hub.example.test@evil.example/"],
      ["an absolute URL with credentials", "https://user:pw@hub.example.test/asks/1"],
      ["a protocol-relative URL", "//evil.example/x"],
      ["a backslash path browsers read as a slash", "/\\evil.example/x"],
      ["a javascript: URL", "javascript:alert(1)"],
      ["a data: URL", "data:text/html,<script>alert(1)</script>"],
      ["a file: URL", "file:///etc/passwd"],
      ["a bare host", "evil.example:8080/x"],
      ["a non-string", 42],
    ])("refuses %s and opens the app without a target", async (_name, url) => {
      const { stashed, worker } = await stashedURL(url, scope);
      expect(stashed).toBeNull();
      expect(worker.openWindow).toHaveBeenCalledWith("./");
      expect(worker.openWindow).not.toHaveBeenCalledWith(expect.stringContaining("evil"));
    });

    it("never hands a refused target to an open window either", async () => {
      const open: FakeWindow = { focus: vi.fn(() => Promise.resolve(open)), postMessage: vi.fn() };
      const worker = loadWorker([open], scope);
      await worker.click({ id: "n-1", url: "https://evil.example/phish" });
      expect(open.postMessage).not.toHaveBeenCalled();
      expect(open.focus).toHaveBeenCalled();
      expect(worker.stash.has("/__pending_nav")).toBe(false);
    });
  });

  describe("with the Hub served under a tunnel base path", () => {
    const scope = "https://tunnel.example.test/apps/hub/";

    it.each([
      ["a record path", "asks/ask-1", "https://tunnel.example.test/apps/hub/asks/ask-1"],
      ["a Hub-relative path with a leading slash", "/asks/ask-1", "https://tunnel.example.test/apps/hub/asks/ask-1"],
      ["an absolute URL inside the base", "https://tunnel.example.test/apps/hub/delivery/n-1", "https://tunnel.example.test/apps/hub/delivery/n-1"],
    ])("keeps the base for %s", async (_name, url, expected) => {
      expect((await stashedURL(url, scope)).stashed).toBe(expected);
    });

    it.each([
      ["a path that climbs out of the base", "../../other/asks/1"],
      ["an absolute URL on the same origin outside the base", "https://tunnel.example.test/apps/other/asks/1"],
      ["the tunnel's root", "https://tunnel.example.test/"],
    ])("refuses %s", async (_name, url) => {
      expect((await stashedURL(url, scope)).stashed).toBeNull();
    });
  });
});

describe("service worker notification click", () => {
  it("stashes the target and opens the app when no window is open", async () => {
    // The worker can't drive navigation on iOS, so it records the target for
    // the app to act on and just launches the app at its root.
    const worker = loadWorker([]);
    await worker.click({ id: "notification-1", url: "asks/ask-1" });
    expect(worker.openWindow).toHaveBeenCalledWith("./");
    const stashed = JSON.parse(worker.stash.get("/__pending_nav") ?? "{}");
    expect(stashed.url).toBe("https://hub.example.test/asks/ask-1");
    expect(typeof stashed.ts).toBe("number");
  });

  it("posts the target to an open window and focuses it without opening a new one", async () => {
    const open: FakeWindow = {
      focus: vi.fn(() => Promise.resolve(open)),
      postMessage: vi.fn(),
    };
    const worker = loadWorker([open]);
    await worker.click({ id: "notification-1", url: "asks/ask-1" });
    expect(open.postMessage).toHaveBeenCalledWith({ type: "PENDING_NAV", url: "https://hub.example.test/asks/ask-1" });
    expect(open.focus).toHaveBeenCalled();
    expect(worker.openWindow).not.toHaveBeenCalled();
    expect(JSON.parse(worker.stash.get("/__pending_nav") ?? "{}").url).toBe("https://hub.example.test/asks/ask-1");
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
  const hubScope = "https://notification-hub.itsagitime.com/";
  const approvalURL = `${hubScope}delivery/approval-reminder`;

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
      focus: () => {
        // Model the app returning to the foreground after the click handler
        // has persisted its target.
        expect(hasPendingTarget()).toBe(true);
        setVisibility("visible");
        return Promise.resolve(page);
      },
    };
    const worker = loadWorker([page], hubScope);
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
      focus: () => Promise.resolve(page),
    };
    const worker = loadWorker([page], hubScope);
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
    const worker = loadWorker([], hubScope);
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
      focus: () => { order.push(worker.stash.has("/__pending_nav") ? "focus-after-stash" : "focus-before-stash"); return Promise.resolve({} as FakeWindow); },
    }], hubScope);
    await worker.click({ id: "approval-reminder", url: approvalURL });
    expect(order).toEqual(["message-after-stash", "focus-after-stash"]);
  });
});
