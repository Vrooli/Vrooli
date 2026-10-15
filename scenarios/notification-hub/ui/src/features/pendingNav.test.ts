// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { initPendingNavigation, sameSiteTarget, takePendingNav } from "./pendingNav";

const HUB = "https://notification-hub.itsagitime.com/";

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
});

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
});
