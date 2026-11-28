import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppNotificationSettings, initAppNotifications } from "@vrooli/react-component-library/AppNotificationSettings/1.0.1";

const IPHONE =
  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.5 Mobile/15E148 Safari/604.1";

interface Call {
  url: string;
  method: string;
  body: Record<string, unknown>;
}

interface Browser {
  calls: Call[];
  register: ReturnType<typeof vi.fn>;
  subscribe: ReturnType<typeof vi.fn>;
  unsubscribe: ReturnType<typeof vi.fn>;
  requestPermission: ReturnType<typeof vi.fn>;
  listeners: Map<string, (event: MessageEvent) => void>;
}

const offState = {
  scenario: "web-console",
  displayName: "Web Console",
  vapidPublicKey:
    "BHEkLmPt8VDsitW0Gjb10rglXC0MuFEFVxhBVHUQToicx7Z_kTQstNzS6gw38y6l4Wa-hPRr0TOLRMjk-4Vbrpw",
  eligible: true,
  publicUrl: "https://web-console.example.test",
  devices: [
    { group: "mac-group", label: "Mac · Chrome", lastUsed: "2026-10-25T10:00:00Z" },
    {
      group: "iphone-group",
      label: "iPhone",
      lastUsed: "2026-10-28T10:00:00Z",
      bestMatch: true,
      apps: ["Agent Manager"],
    },
  ],
};

const pairedState = {
  ...offState,
  subscription: {
    id: "s1",
    deviceGroup: "iphone-group",
    deviceLabel: "iPhone",
    permission: "granted",
    confirmedAt: "2026-10-28T10:00:00Z",
    healthy: true,
  },
};

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

/**
 * An installed iPhone app: service worker, push and Notification APIs, and a
 * scenario web server answering /vrooli-notify with `routes`.
 */
function installBrowser(options: {
  routes: Record<string, (call: Call) => Response>;
  subscribed?: boolean;
  standalone?: boolean;
}): Browser {
  const calls: Call[] = [];
  const unsubscribe = vi.fn().mockResolvedValue(true);
  const subscription = {
    endpoint: "https://web.push.apple.com/this-app",
    toJSON: () => ({
      endpoint: "https://web.push.apple.com/this-app",
      keys: { p256dh: "p256", auth: "auth" },
    }),
    unsubscribe,
  };
  const subscribe = vi.fn().mockResolvedValue(subscription);
  const pushManager = {
    getSubscription: vi.fn().mockResolvedValue(options.subscribed ? subscription : null),
    subscribe,
  };
  const registration = { active: {}, pushManager };
  const register = vi.fn().mockResolvedValue(registration);
  const listeners = new Map<string, (event: MessageEvent) => void>();
  Object.defineProperty(navigator, "serviceWorker", {
    configurable: true,
    value: {
      register,
      getRegistration: vi.fn().mockResolvedValue(options.subscribed ? registration : undefined),
      addEventListener: (type: string, fn: (event: MessageEvent) => void) =>
        listeners.set(type, fn),
      removeEventListener: (type: string) => listeners.delete(type),
    },
  });
  Object.defineProperty(navigator, "userAgent", { configurable: true, get: () => IPHONE });
  Object.defineProperty(navigator, "standalone", {
    configurable: true,
    value: options.standalone ?? true,
  });
  (window as unknown as { PushManager: unknown }).PushManager = function PushManager() {};
  const requestPermission = vi.fn().mockResolvedValue("granted");
  (globalThis as unknown as { Notification: unknown }).Notification = {
    permission: options.subscribed ? "granted" : "default",
    requestPermission,
  };
  vi.spyOn(globalThis, "fetch").mockImplementation((input, init) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const body =
      typeof init?.body === "string" ? (JSON.parse(init.body) as Record<string, unknown>) : {};
    const call: Call = { url, method: init?.method ?? "GET", body };
    calls.push(call);
    const route = options.routes[url];
    return Promise.resolve(
      route ? route(call) : json({ code: "not_found", message: `no route for ${url}` }, 404),
    );
  });
  return { calls, register, subscribe, unsubscribe, requestPermission, listeners };
}

describe("AppNotificationSettings", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("asks an iPhone user to open the app from the Home Screen first", () => {
    installBrowser({ routes: {}, standalone: false });
    render(<AppNotificationSettings showCategories={false} />);
    expect(
      screen.getByText(/add it to your Home Screen and open it from there/),
    ).toBeInTheDocument();
  });

  it("turns on: permission from the tap, the shared worker, then pairing with the preselected device", async () => {
    const browser = installBrowser({
      routes: {
        "/vrooli-notify/rpc/GetAppState": () => json(offState),
        "/vrooli-notify/rpc/EnableApp": () => json(pairedState),
      },
    });
    render(<AppNotificationSettings showCategories={false} />);

    fireEvent.click(await screen.findByRole("button", { name: "Get notifications in this app" }));
    expect(await screen.findByText("Which device is this?")).toBeInTheDocument();
    expect(browser.requestPermission).toHaveBeenCalledTimes(1);
    expect(browser.register).toHaveBeenCalledWith("/vrooli-notify/sw.js", {
      scope: "/vrooli-notify/",
    });
    expect(browser.subscribe).toHaveBeenCalledWith(
      expect.objectContaining({ userVisibleOnly: true }),
    );
    expect(screen.getByRole("radio", { name: /iPhone/ })).toBeChecked();
    expect(screen.getByText("Also here: Agent Manager")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Use this app here" }));
    expect(
      await screen.findByText(
        "Web Console notifications appear in this app on iPhone, instead of in Chimesail.",
      ),
    ).toBeInTheDocument();
    const enable = browser.calls.find((c) => c.url.endsWith("/EnableApp"));
    expect(enable?.body).toMatchObject({
      endpoint: "https://web.push.apple.com/this-app",
      p256dh: "p256",
      auth: "auth",
      origin: window.location.origin,
      pairWithGroup: "iphone-group",
      standalone: false,
    });
    expect(enable?.body).not.toHaveProperty("scenario");
  });

  it("explains why app notifications are unavailable for a scenario without a permanent address", async () => {
    installBrowser({
      routes: {
        "/vrooli-notify/rpc/GetAppState": () =>
          json({
            ...offState,
            eligible: false,
            ineligibleReason: "web-console only has a temporary address.",
          }),
      },
    });
    render(<AppNotificationSettings showCategories={false} />);
    expect(
      await screen.findByText(
        /web-console only has a temporary address\. Notifications still arrive in Chimesail\./,
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Get notifications in this app" }),
    ).not.toBeInTheDocument();
  });

  it("shows why the hub is using Chimesail and switches back on request", async () => {
    const browser = installBrowser({
      subscribed: true,
      routes: {
        "/vrooli-notify/rpc/GetAppState": () =>
          json({
            ...pairedState,
            subscription: {
              ...pairedState.subscription,
              healthy: false,
              unhealthyReason:
                "Notifications are turned off for this app in the device settings; using Chimesail",
            },
          }),
        "/vrooli-notify/rpc/DisableApp": () => json({}),
      },
    });
    render(<AppNotificationSettings showCategories={false} />);
    expect(
      await screen.findByText(/turned off for this app in the device settings/),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Use Chimesail instead" }));
    await waitFor(() => {
      expect(browser.unsubscribe).toHaveBeenCalled();
    });
    expect(browser.calls.find((c) => c.url.endsWith("/DisableApp"))?.body).toEqual({
      endpoint: "https://web.push.apple.com/this-app",
    });
  });

  it("renders nothing while the scenario has not turned on app notifications", async () => {
    const browser = installBrowser({ routes: {} });
    vi.mocked(globalThis.fetch).mockImplementation((input) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      browser.calls.push({ url, method: "GET", body: {} });
      return Promise.resolve(
        new Response("Proxy route not handled", {
          status: 404,
          headers: { "Content-Type": "text/plain" },
        }),
      );
    });
    const { container } = render(<AppNotificationSettings />);
    await waitFor(() => {
      expect(browser.calls.some((c) => c.url.endsWith("/GetAppState"))).toBe(true);
    });
    await waitFor(() => {
      expect(container).toBeEmptyDOMElement();
    });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("reports a non-JSON server failure without a raw parse error", async () => {
    installBrowser({
      routes: {
        "/vrooli-notify/rpc/GetAppState": () =>
          new Response("<html>Bad gateway</html>", {
            status: 502,
            headers: { "Content-Type": "text/html" },
          }),
      },
    });
    render(<AppNotificationSettings showCategories={false} />);
    expect(await screen.findByText("Notifications answered 502.")).toBeInTheDocument();
    expect(screen.queryByText(/JSON|Unexpected token/)).not.toBeInTheDocument();
  });

  it("renders from an injected client without touching the network", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    render(
      <AppNotificationSettings
        showCategories={false}
        client={{
          support: () => "supported",
          getState: () =>
            Promise.resolve({ ...offState, ineligibleReason: "", subscription: null, devices: [] }),
          prepareSubscription: () => Promise.reject(new Error("unused")),
          enable: () => Promise.reject(new Error("unused")),
          disable: () => Promise.resolve(),
          confirm: () => Promise.resolve(true),
          fetchCategories: () => Promise.resolve([]),
          saveCategory: () => Promise.resolve(),
        }}
      />,
    );
    expect(
      await screen.findByRole("button", { name: "Get notifications in this app" }),
    ).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("saves a notification type change for this scenario", async () => {
    const browser = installBrowser({
      routes: {
        "/vrooli-notify/rpc/GetAppState": () => json(offState),
        "/vrooli-notify/catalog": () =>
          json({
            sources: [
              {
                source_scenario: "web-console",
                categories: [
                  {
                    category: "session.turn_completed",
                    display_name: "Session finished its turn",
                    description: "Idle and waiting.",
                    default_enabled: true,
                    default_min_urgency: "normal",
                  },
                ],
              },
            ],
            preferences: [],
          }),
        "/vrooli-notify/preferences": () => json({}),
      },
    });
    render(<AppNotificationSettings />);
    fireEvent.click(await screen.findByRole("switch", { name: /Session finished its turn/ }));
    await waitFor(() => {
      expect(browser.calls.some((c) => c.method === "PUT")).toBe(true);
    });
    expect(browser.calls.find((c) => c.method === "PUT")?.body).toMatchObject({
      category: "session.turn_completed",
      enabled: false,
      min_urgency: "normal",
    });
  });
});

describe("initAppNotifications", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("re-confirms the app and opens the screen a notification tap asked for", async () => {
    const browser = installBrowser({
      subscribed: true,
      routes: { "/vrooli-notify/rpc/ConfirmApp": () => json({ registered: true, healthy: true }) },
    });
    const opened: string[] = [];
    const cleanup = initAppNotifications({
      onNavigate: (url) => {
        opened.push(url);
        return true;
      },
    });

    await waitFor(() => {
      expect(browser.calls.some((c) => c.url.endsWith("/ConfirmApp"))).toBe(true);
    });
    expect(browser.calls.find((c) => c.url.endsWith("/ConfirmApp"))?.body).toMatchObject({
      endpoint: "https://web.push.apple.com/this-app",
      permission: "granted",
    });

    browser.listeners.get("message")?.({
      data: { type: "vrooli-notify:navigate", url: "/sessions/abc" },
    } as MessageEvent);
    browser.listeners.get("message")?.({
      data: { type: "vrooli-notify:navigate", url: "https://elsewhere.example.test/x" },
    } as MessageEvent);
    expect(opened).toEqual([`${window.location.origin}/sessions/abc`]);
    cleanup();
  });

  it("re-pairs by itself when the hub no longer knows this app", async () => {
    localStorage.setItem("vrooli-notify:pairing", JSON.stringify({ group: "iphone-group" }));
    const browser = installBrowser({
      subscribed: true,
      routes: {
        "/vrooli-notify/rpc/ConfirmApp": () => json({}),
        "/vrooli-notify/rpc/EnableApp": () => json(pairedState),
      },
    });
    const cleanup = initAppNotifications();
    await waitFor(() => {
      expect(browser.calls.find((c) => c.url.endsWith("/EnableApp"))?.body).toMatchObject({
        pairWithGroup: "iphone-group",
      });
    });
    cleanup();
  });

  it("does nothing until notifications were turned on in this app", async () => {
    const browser = installBrowser({ routes: {} });
    const cleanup = initAppNotifications();
    await new Promise((resolve) => setTimeout(resolve, 10));
    expect(browser.calls).toHaveLength(0);
    cleanup();
  });
});
