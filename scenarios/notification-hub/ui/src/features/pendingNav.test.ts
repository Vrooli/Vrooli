// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { initPendingNavigation, sameSiteTarget, takePendingNav } from "./pendingNav";

const HUB = "https://notification-hub.itsagitime.com/";
let originalVisibility: PropertyDescriptor | undefined;

beforeEach(() => {
  originalVisibility = Object.getOwnPropertyDescriptor(document, "visibilityState");
});

function setLocation(href: string) {
  const u = new URL(href);
  const assign = vi.fn();
  Object.defineProperty(window, "location", {
    configurable: true,
    value: { href: u.href, hostname: u.hostname, origin: u.origin, assign },
  });
  return assign;
}

function mockCache(entry?: { url: string; ts: number }) {
  const store = new Map<string, string>();
  if (entry) store.set("/__pending_nav", JSON.stringify(entry));
  const cache = {
    // Capture the body at match time, like the real Cache API — the module
    // deletes the entry before reading json(), so a lazy read would see nothing.
    match: (key: string) => {
      const raw = store.get(key);
      return Promise.resolve(raw === undefined ? undefined : { json: () => Promise.resolve(JSON.parse(raw)) });
    },
    delete: (key: string) => Promise.resolve(store.delete(key)),
    put: () => Promise.resolve(),
  };
  vi.stubGlobal("caches", { open: () => Promise.resolve(cache) });
  return store;
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  if (originalVisibility) {
    Object.defineProperty(document, "visibilityState", originalVisibility);
  } else {
    Reflect.deleteProperty(document, "visibilityState");
  }
});

function setVisibility(state: "hidden" | "visible") {
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    get: () => state,
  });
}

describe("sameSiteTarget", () => {
  it("accepts the exact same host", () => {
    setLocation(HUB);
    expect(sameSiteTarget("https://notification-hub.itsagitime.com/asks/1")).toBe(
      "https://notification-hub.itsagitime.com/asks/1",
    );
  });

  it("accepts a sibling scenario on the same registrable domain", () => {
    setLocation(HUB);
    expect(sameSiteTarget("https://web-console.itsagitime.com/?session=s1")).toBe(
      "https://web-console.itsagitime.com/?session=s1",
    );
  });

  it("rejects a foreign domain", () => {
    setLocation(HUB);
    expect(sameSiteTarget("https://evil.example/phish")).toBeNull();
  });

  it("rejects a lookalike suffix domain", () => {
    setLocation(HUB);
    expect(sameSiteTarget("https://web.notitsagitime.com/")).toBeNull();
  });

  it("rejects a non-http(s) scheme", () => {
    setLocation(HUB);
    expect(sameSiteTarget("javascript:alert(1)")).toBeNull();
  });
});

describe("takePendingNav", () => {
  it("returns a fresh same-site target and clears it", async () => {
    setLocation(HUB);
    const store = mockCache({ url: "https://web-console.itsagitime.com/?session=s1", ts: 1_000 });
    const got = await takePendingNav(1_000 + 5_000);
    expect(got).toBe("https://web-console.itsagitime.com/?session=s1");
    expect(store.has("/__pending_nav")).toBe(false); // consumed
  });

  it("drops a stale stash", async () => {
    setLocation(HUB);
    mockCache({ url: "https://web-console.itsagitime.com/", ts: 1_000 });
    expect(await takePendingNav(1_000 + 200_000)).toBeNull();
  });

  it("drops an off-site stash", async () => {
    setLocation(HUB);
    mockCache({ url: "https://evil.example/", ts: 1_000 });
    expect(await takePendingNav(1_000 + 1_000)).toBeNull();
  });

  it("returns null when nothing is stashed", async () => {
    setLocation(HUB);
    mockCache();
    expect(await takePendingNav(5_000)).toBeNull();
  });
});

describe("initPendingNavigation", () => {
  it("navigates to a fresh stashed target on boot", async () => {
    const assign = setLocation(HUB);
    mockCache({ url: "https://web-console.itsagitime.com/?session=s1", ts: Date.now() });
    const dispose = initPendingNavigation();
    await new Promise((r) => setTimeout(r, 0));
    expect(assign).toHaveBeenCalledWith("https://web-console.itsagitime.com/?session=s1");
    dispose();
  });

  it("acts on a PENDING_NAV message from the worker", async () => {
    const assign = setLocation(HUB);
    mockCache();
    const handlers: Array<(e: MessageEvent) => void> = [];
    vi.stubGlobal("navigator", {
      serviceWorker: {
        addEventListener: (_t: string, h: (e: MessageEvent) => void) => handlers.push(h),
        removeEventListener: () => {},
      },
    });
    const dispose = initPendingNavigation();
    handlers.forEach((h) => h({ data: { type: "PENDING_NAV", url: "https://web-console.itsagitime.com/?session=s2" } } as MessageEvent));
    await new Promise((r) => setTimeout(r, 0));
    expect(assign).toHaveBeenCalledWith("https://web-console.itsagitime.com/?session=s2");
    dispose();
  });

  it("keeps a hidden-start cache entry until a visible focus without another message", async () => {
    const assign = setLocation(HUB);
    setVisibility("hidden");
    const store = mockCache({ url: "https://agent-manager.itsagitime.com/api/v1/authorizations/a1/approve", ts: Date.now() });
    const dispose = initPendingNavigation();
    await new Promise((r) => setTimeout(r, 0));

    expect(assign).not.toHaveBeenCalled();
    expect(store.has("/__pending_nav")).toBe(true);

    // Model the app becoming visible without a visibilitychange or worker
    // message. The focus signal must retry the same durable cache consumer.
    setVisibility("visible");
    window.dispatchEvent(new Event("focus"));
    await new Promise((r) => setTimeout(r, 0));

    expect(assign).toHaveBeenCalledWith("https://agent-manager.itsagitime.com/api/v1/authorizations/a1/approve");
    expect(store.has("/__pending_nav")).toBe(false);
    dispose();
  });

  it("consumes only the stash matching a worker message and preserves a newer target", async () => {
    const assign = setLocation(HUB);
    setVisibility("visible");
    const store = mockCache({ url: "https://agent-manager.itsagitime.com/api/v1/authorizations/a1/approve", ts: Date.now() });
    const handlers: Array<(e: MessageEvent) => void> = [];
    vi.stubGlobal("navigator", {
      serviceWorker: {
        addEventListener: (_t: string, h: (e: MessageEvent) => void) => handlers.push(h),
        removeEventListener: () => {},
      },
    });
    const dispose = initPendingNavigation();
    await new Promise((r) => setTimeout(r, 0));
    expect(assign).toHaveBeenCalledWith("https://agent-manager.itsagitime.com/api/v1/authorizations/a1/approve");
    expect(store.has("/__pending_nav")).toBe(false);

    const oldMessageURL = "https://agent-manager.itsagitime.com/api/v1/authorizations/a2/approve";
    store.set("/__pending_nav", JSON.stringify({
      url: "https://agent-manager.itsagitime.com/api/v1/authorizations/a3/approve",
      ts: Date.now(),
    }));
    handlers.forEach((h) => h({ data: { type: "PENDING_NAV", url: oldMessageURL } } as MessageEvent));
    await new Promise((r) => setTimeout(r, 0));

    expect(assign).toHaveBeenLastCalledWith(oldMessageURL);
    expect(JSON.parse(store.get("/__pending_nav") ?? "{}").url).toBe("https://agent-manager.itsagitime.com/api/v1/authorizations/a3/approve");
    dispose();
  });

  it("recovers a fresh hidden-start cache entry on pageshow without visibilitychange or message", async () => {
    const assign = setLocation(HUB);
    setVisibility("hidden");
    const store = mockCache();
    const dispose = initPendingNavigation();
    await new Promise((r) => setTimeout(r, 0));

    expect(assign).not.toHaveBeenCalled();
    expect(store.has("/__pending_nav")).toBe(false);

    // Model the live worker writing the durable stash after app startup while
    // its PENDING_NAV message is lost during suspension.
    store.set("/__pending_nav", JSON.stringify({
      url: "https://agent-manager.itsagitime.com/api/v1/authorizations/a2/approve",
      ts: Date.now(),
    }));

    setVisibility("visible");
    window.dispatchEvent(new Event("pageshow"));
    await new Promise((r) => setTimeout(r, 0));

    expect(assign).toHaveBeenCalledWith("https://agent-manager.itsagitime.com/api/v1/authorizations/a2/approve");
    expect(store.has("/__pending_nav")).toBe(false);
    dispose();
  });
});
