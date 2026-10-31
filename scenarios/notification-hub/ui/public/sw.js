// SW_VERSION is the one knob to bump on every service-worker change. The
// Settings "Connection status" panel reads the RUNNING worker's version (via a
// GET_VERSION message) and the AVAILABLE version (by fetching this file fresh)
// and flags the worker as stale when they differ — so a worker iOS refused to
// swap is visible instead of silent. Keep this a simple literal the panel can
// also parse out of the served source with a regex.
const SW_VERSION = "2026-10-25.1";
const CACHE_NAME = "notification-hub-app-shell-v2";
const APP_SHELL_URLS = ["./", "./public/site.webmanifest"];

// Where a notification tap stashes its intended destination. iOS does NOT honor
// a URL passed to navigate()/openWindow() from a backgrounded home-screen app —
// it just foregrounds the app at whatever page it was on. So the worker can't
// drive the navigation itself; it records the target here and the app reads it
// when it comes to the foreground (see src/features/pendingNav.ts).
const PENDING_NAV_CACHE = "notification-hub-pending-nav";
const PENDING_NAV_KEY = "/__pending_nav";

async function stashPendingNav(url) {
  if (!url) return;
  try {
    const cache = await caches.open(PENDING_NAV_CACHE);
    await cache.put(
      PENDING_NAV_KEY,
      new Response(JSON.stringify({ url, ts: Date.now() }), { headers: { "Content-Type": "application/json" } }),
    );
  } catch {
    // A failed stash just means the foreground app has nothing to act on; the
    // notification still opened the app. Never throw out of a click handler.
  }
}

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL_URLS))
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(
          names
            .filter((name) => name !== CACHE_NAME && name !== PENDING_NAV_CACHE)
            .map((name) => caches.delete(name)),
        )
      )
      .then(() => self.clients.claim())
  );
});

// The page talks to the worker to read its version and to force a stuck update
// through. GET_VERSION replies on the MessageChannel port the page supplied;
// SKIP_WAITING lets the Settings "Update now" button promote a waiting worker
// immediately rather than waiting for iOS's own (slow, opaque) swap timing.
self.addEventListener("message", (event) => {
  const data = event.data || {};
  if (data.type === "GET_VERSION") {
    const port = event.ports && event.ports[0];
    if (port) port.postMessage({ type: "VERSION", version: SW_VERSION });
    return;
  }
  if (data.type === "SKIP_WAITING") {
    self.skipWaiting();
  }
});

self.addEventListener("push", (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch {
    payload = { body: event.data ? event.data.text() : "Notification available" };
  }
  event.waitUntil(self.registration.showNotification(payload.title || "Chimesail", {
    body: payload.body || "Notification available",
    tag: payload.id || undefined,
    icon: payload.icon || "/public/icon-192.png",
    data: { id: payload.id || "", url: payload.url || "" },
  }));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const data = event.notification.data || {};
  const url = typeof data.url === "string" ? data.url : "";
  event.waitUntil((async () => {
    // Record the destination first — this is what the foregrounded app acts on,
    // and the only mechanism that works on iOS (navigate/openWindow with a path
    // are ignored there). Then try the direct paths too, which DO work on
    // desktop/Android, and post the target to any live client so an
    // already-open app can act immediately without waiting on the cache read.
    await stashPendingNav(url);
    const windows = await clients.matchAll({ type: "window", includeUncontrolled: true });
    for (const client of windows) {
      client.postMessage({ type: "PENDING_NAV", url });
    }
    if (windows.length > 0) {
      // On iOS this focus is all that happens — the client's PENDING_NAV /
      // foreground handler performs the actual navigation. On desktop focus
      // also surfaces the window the app will route in.
      try {
        await windows[0].focus();
      } catch {
        // ignore
      }
      return;
    }
    // App isn't open: launch it at the root; it reads the stash on boot.
    await clients.openWindow("./");
  })());
});

self.addEventListener("pushsubscriptionchange", (event) => {
  event.waitUntil((async () => {
    // Reuse the old subscription's options so the VAPID applicationServerKey
    // carries over; subscribing without it fails.
    const options = event.oldSubscription && event.oldSubscription.options ? event.oldSubscription.options : { userVisibleOnly: true };
    const replacement = await self.registration.pushManager.subscribe(options);
    await registerWithHub(replacement.toJSON());
  })());
});

// Tell the hub about a replaced subscription. The owner session cookie rides
// along (same origin), so this works with no window open; without it the hub
// would keep the dead endpoint and the operator would silently stop hearing.
function registerWithHub(subscription) {
  const keys = subscription.keys || {};
  const url = new URL("vrooli.notification_hub.v1.recipients.RecipientsService/RegisterPushSubscription", self.registration.scope);
  return fetch(url.href, {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", "Connect-Protocol-Version": "1", "X-Vrooli-Browser-Session": "1" },
    body: JSON.stringify({ endpoint: subscription.endpoint, p256dh: keys.p256dh, auth: keys.auth, origin: new URL(self.registration.scope).origin }),
  });
}

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request).catch(() => caches.match("./").then((response) => response || Response.error()))
    );
    return;
  }

  event.respondWith(
    caches.match(request).then((cached) => cached || fetch(request))
  );
});
